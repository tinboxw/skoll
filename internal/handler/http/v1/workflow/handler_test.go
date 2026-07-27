package workflow_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
	skollhttp "github.com/tinboxw/skoll/internal/handler/http"
	workflowhttp "github.com/tinboxw/skoll/internal/handler/http/v1/workflow"
	"github.com/tinboxw/skoll/internal/handler/middleware"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	permissionsvc "github.com/tinboxw/skoll/internal/service/permission"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"github.com/tinboxw/skoll/internal/store"
)

func TestWorkflowAPIRunsThreeScenariosWithPermissionAndAuditRecords(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}
	workflowService := workflowsvc.NewService(workflowsvc.NewMemoryRepository())
	permissionService := permissionsvc.NewService(bundle.Permissions)
	auditEventService := auditsvc.NewEventService(bundle.AuditEvents)
	router := skollhttp.NewRouter(
		skollhttp.Dependencies{
			WorkflowService:   workflowService,
			PermissionService: permissionService,
			AuditEventService: auditEventService,
		},
		middleware.RequestAudit(auditEventService, middleware.WithRequestAuditID(sequentialAuditIDs()), middleware.WithRequestAuditClock(fixedAuditClock())),
	)

	created := workflowPost(t, router, "/skoll/v1/workflows/definitions", approvalDefinitionBody("wf-api"))
	if got := itemString(created, "status"); got != "draft" {
		t.Fatalf("expected draft definition, got %q", got)
	}
	published := workflowPost(t, router, "/skoll/v1/workflows/definitions/wf-api/publish", nil)
	if got := itemString(published, "status"); got != "published" {
		t.Fatalf("expected published definition, got %q", got)
	}

	approved := workflowPost(t, router, "/skoll/v1/workflows/instances", startBody("inst-approved", "oa.leave", "leave-1", "Leave request"))
	approvedTask := firstTaskID(t, approved)
	approved = workflowPost(t, router, "/skoll/v1/workflows/instances/inst-approved/tasks/"+approvedTask+"/approve", actionBody("approver-1", "approved"))
	if got := itemString(approved, "status"); got != "approved" {
		t.Fatalf("expected approved scenario to finish approved, got %q", got)
	}

	rejected := workflowPost(t, router, "/skoll/v1/workflows/instances", startBody("inst-rejected", "oa.purchase", "purchase-1", "Purchase request"))
	rejectedTask := firstTaskID(t, rejected)
	rejected = workflowPost(t, router, "/skoll/v1/workflows/instances/inst-rejected/tasks/"+rejectedTask+"/reject", actionBody("approver-1", "rejected"))
	if got := itemString(rejected, "status"); got != "rejected" {
		t.Fatalf("expected rejected scenario to finish rejected, got %q", got)
	}

	transferred := workflowPost(t, router, "/skoll/v1/workflows/instances", startBody("inst-transferred", "oa.inbound", "inbound-1", "Inbound approval"))
	originalTask := firstTaskID(t, transferred)
	_ = workflowPost(t, router, "/skoll/v1/workflows/instances/inst-transferred/tasks/"+originalTask+"/copy", targetBody("approver-1", "observer-1", "copied"))
	transferred = workflowPost(t, router, "/skoll/v1/workflows/instances/inst-transferred/tasks/"+originalTask+"/transfer", targetBody("approver-1", "approver-2", "transferred"))
	transferTask := pendingTaskFor(t, transferred, "approver-2")
	transferred = workflowPost(t, router, "/skoll/v1/workflows/instances/inst-transferred/tasks/"+transferTask+"/approve", actionBody("approver-2", "approved by transfer target"))
	if got := itemString(transferred, "status"); got != "approved" {
		t.Fatalf("expected transfer scenario to finish approved, got %q", got)
	}
	if !timelineContains(transferred, "copy") || !timelineContains(transferred, "transfer") {
		t.Fatalf("expected copy and transfer actions in timeline: %+v", transferred["item"])
	}

	for _, key := range []string{
		workflowhttp.PermissionWorkflowDefinitionManage,
		workflowhttp.PermissionWorkflowInstanceStart,
		workflowhttp.PermissionWorkflowInstanceRead,
		workflowhttp.PermissionWorkflowTaskAct,
	} {
		item, err := permissionService.GetResource(context.Background(), key)
		if err != nil {
			t.Fatalf("permission %s lookup error: %v", key, err)
		}
		if item == nil || item.Key() != key || item.Module() != "workflow" || item.Source() != "system" {
			t.Fatalf("permission %s was not registered correctly: %+v", key, item)
		}
	}

	events, err := auditEventService.ListEvents(context.Background(), auditsvc.EventFilter{ResourceType: "http_request", Limit: 50})
	if err != nil {
		t.Fatalf("list audit events error: %v", err)
	}
	if len(events) < 10 {
		t.Fatalf("expected audit event for each workflow API call, got %d", len(events))
	}
	if !auditContains(events, http.MethodPost+" /skoll/v1/workflows/instances/inst-approved/tasks/"+approvedTask+"/approve") {
		t.Fatalf("expected approve request audit event, events=%+v", events)
	}
	if !auditContains(events, http.MethodPost+" /skoll/v1/workflows/instances/inst-rejected/tasks/"+rejectedTask+"/reject") {
		t.Fatalf("expected reject request audit event, events=%+v", events)
	}
	if !auditContains(events, http.MethodPost+" /skoll/v1/workflows/instances/inst-transferred/tasks/"+originalTask+"/transfer") {
		t.Fatalf("expected transfer request audit event, events=%+v", events)
	}
}

func TestWorkflowAPIRejectsInvalidPayload(t *testing.T) {
	router := skollhttp.NewRouter(skollhttp.Dependencies{WorkflowService: workflowsvc.NewService(workflowsvc.NewMemoryRepository())})
	req := httptest.NewRequest(http.MethodPost, "/skoll/v1/workflows/definitions", strings.NewReader("{"))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("invalid definition status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func workflowPost(t *testing.T, router http.Handler, path string, body any) map[string]any {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader([]byte(`{}`))
	} else {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(http.MethodPost, path, reader)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code < 200 || resp.Code >= 300 {
		t.Fatalf("%s status=%d body=%s", path, resp.Code, resp.Body.String())
	}
	var decoded map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode response: %v body=%s", err, resp.Body.String())
	}
	data, ok := decoded["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected response data object, got %+v", decoded)
	}
	return data
}

func approvalDefinitionBody(id string) map[string]any {
	return map[string]any{
		"id":      id,
		"key":     "pharma.approval",
		"name":    "Pharma Approval",
		"version": 1,
		"nodes": []map[string]any{
			{"id": "start", "key": "start", "name": "Start", "type": "start"},
			{"id": "review", "key": "review", "name": "Review", "type": "approval", "assignees": []string{"approver-1"}, "decision": map[string]any{"strategy": "any", "quorum": 1}},
			{"id": "end", "key": "end", "name": "End", "type": "end"},
		},
		"transitions": []map[string]string{
			{"from": "start", "to": "review"},
			{"from": "review", "to": "end"},
		},
	}
}

func startBody(id, businessType, businessID, title string) map[string]any {
	return map[string]any{
		"id":           id,
		"definitionId": "wf-api",
		"businessType": businessType,
		"businessId":   businessID,
		"title":        title,
		"starter":      map[string]string{"id": "starter-1", "name": "Starter"},
		"variables":    map[string]any{},
	}
}

func actionBody(actorID, comment string) map[string]any {
	return map[string]any{"actor": map[string]string{"id": actorID, "name": actorID}, "comment": comment}
}

func targetBody(actorID, targetID, comment string) map[string]any {
	return map[string]any{
		"actor":   map[string]string{"id": actorID, "name": actorID},
		"target":  map[string]string{"id": targetID, "name": targetID},
		"comment": comment,
	}
}

func itemString(data map[string]any, key string) string {
	item, _ := data["item"].(map[string]any)
	value, _ := item[key].(string)
	return value
}

func firstTaskID(t *testing.T, data map[string]any) string {
	t.Helper()
	item, _ := data["item"].(map[string]any)
	tasks, _ := item["tasks"].([]any)
	if len(tasks) == 0 {
		t.Fatalf("expected at least one task: %+v", item)
	}
	task, _ := tasks[0].(map[string]any)
	id, _ := task["id"].(string)
	if id == "" {
		t.Fatalf("expected task id: %+v", task)
	}
	return id
}

func pendingTaskFor(t *testing.T, data map[string]any, assigneeID string) string {
	t.Helper()
	item, _ := data["item"].(map[string]any)
	tasks, _ := item["tasks"].([]any)
	for _, raw := range tasks {
		task, _ := raw.(map[string]any)
		assignee, _ := task["assignee"].(map[string]any)
		if task["status"] == "pending" && assignee["id"] == assigneeID {
			return task["id"].(string)
		}
	}
	t.Fatalf("pending task for %s not found in %+v", assigneeID, tasks)
	return ""
}

func timelineContains(data map[string]any, actionType string) bool {
	item, _ := data["item"].(map[string]any)
	timeline, _ := item["timeline"].([]any)
	for _, raw := range timeline {
		action, _ := raw.(map[string]any)
		if action["type"] == actionType {
			return true
		}
	}
	return false
}

func auditContains(events []*domainaudit.Event, resourceID string) bool {
	for _, event := range events {
		if event != nil && event.Resource.ID == resourceID && event.Result == domainaudit.EventResultSuccess {
			return true
		}
	}
	return false
}

func sequentialAuditIDs() middleware.AuditIDFunc {
	counter := 0
	return func() shared.ID {
		counter++
		return shared.ID("workflow-api-audit-" + time.Unix(0, int64(counter)).Format("150405.000000000"))
	}
}

func fixedAuditClock() middleware.AuditClock {
	now := time.Date(2026, time.July, 4, 10, 0, 0, 0, time.UTC)
	return func() time.Time { return now }
}

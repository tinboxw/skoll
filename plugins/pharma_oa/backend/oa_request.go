package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const oaRequestTable = "oa_requests"

var (
	oaRequestFields = []string{
		"id", "request_type", "title", "description", "form_data", "status", "approver_id", "approver_name",
		"workflow_definition_id", "workflow_instance_id", "submitted_at", "last_operation_key",
		"tenant_id", "organization_id", "owner_id", "created_at", "updated_at",
	}
	oaRequestTypes = map[string]struct{}{"leave": {}, "expense": {}, "procurement": {}, "contract": {}, "custom": {}}
)

type oaRequest struct {
	ID                   string         `json:"id"`
	RequestType          string         `json:"requestType"`
	Title                string         `json:"title"`
	Description          string         `json:"description"`
	FormData             map[string]any `json:"formData"`
	Status               string         `json:"status"`
	ApproverID           string         `json:"approverId"`
	ApproverName         string         `json:"approverName"`
	WorkflowDefinitionID string         `json:"workflowDefinitionId,omitempty"`
	WorkflowInstanceID   string         `json:"workflowInstanceId,omitempty"`
	SubmittedAt          string         `json:"submittedAt,omitempty"`
	Version              int64          `json:"version"`
	CreatedAt            string         `json:"createdAt"`
	UpdatedAt            string         `json:"updatedAt"`
	LastOperationKey     string         `json:"-"`
	scope                employeeScope
}

type oaRequestWriteInput struct {
	TenantID       string         `json:"tenantId"`
	OrganizationID string         `json:"organizationId"`
	RequestType    string         `json:"requestType"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	FormData       map[string]any `json:"formData"`
	ApproverID     string         `json:"approverId"`
	ApproverName   string         `json:"approverName"`
	Version        int64          `json:"version,omitempty"`
}

type oaRequestSubmitInput struct {
	Version int64 `json:"version"`
}

func (s *server) listOARequests(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	offset, limit, err := employeePagination(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	items, err := s.queryOARequests(ctx, strings.TrimSpace(r.URL.Query().Get("keyword")), strings.TrimSpace(r.URL.Query().Get("requestType")), strings.TrimSpace(r.URL.Query().Get("status")))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	total := len(items)
	if offset > total {
		offset = total
	}
	end := min(total, offset+limit)
	writeOK(w, map[string]any{"items": items[offset:end], "total": total, "offset": offset, "limit": limit})
}

func (s *server) getOARequestHandler(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	item, err := s.getOARequest(ctx, oaRequestPermission("read"), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	data := map[string]any{"item": item}
	if item.WorkflowInstanceID != "" {
		workflow, workflowErr := s.host.Workflows.GetInstance(ctx, item.WorkflowInstanceID)
		if workflowErr != nil {
			writeServiceError(w, workflowErr)
			return
		}
		data["workflow"] = oaWorkflowView(workflow)
	}
	writeOK(w, data)
}

func (s *server) createOARequest(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	requestKey, err := mutationKey(r, "create")
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var input oaRequestWriteInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := validateOARequestWrite(&input, false); err != nil {
		writeServiceError(w, err)
		return
	}
	scope, err := s.exactWriteScope(ctx, oaRequestPermission("create"), input.TenantID, input.OrganizationID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	item := oaRequest{
		ID: newEntityID("request"), RequestType: input.RequestType, Title: input.Title, Description: input.Description,
		FormData: input.FormData, Status: "draft", ApproverID: input.ApproverID, ApproverName: input.ApproverName,
		LastOperationKey: requestKey, scope: scope,
	}
	var created oaRequest
	err = s.transaction(ctx, func(tx context.Context) error {
		result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{
			Table: oaRequestTable, Operation: pluginsdk.DataMutationInsert, Scope: oaRequestIntent("create", scope),
			Key: map[string]pluginsdk.DataValue{"id": stringValue(item.ID)}, Values: oaRequestValues(item),
			Returning: oaRequestFields, IdempotencyKey: requestKey,
		})
		if mutationErr != nil {
			return mutationErr
		}
		created, mutationErr = oaRequestFromMutation(result)
		if mutationErr != nil {
			return mutationErr
		}
		return s.audit(tx, "pharma_oa.oa_request.create", created.ID, pluginsdk.AuditRiskMedium, map[string]any{"requestType": created.RequestType, "approverId": created.ApproverID})
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeCreated(w, map[string]any{"item": created})
}

func (s *server) updateOARequest(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	requestKey, err := mutationKey(r, "update")
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var input oaRequestWriteInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := validateOARequestWrite(&input, true); err != nil {
		writeServiceError(w, err)
		return
	}
	item, err := s.getOARequest(ctx, oaRequestPermission("update"), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if item.Status != "draft" {
		writeServiceError(w, newHTTPError(http.StatusConflict, "request_not_editable", "only draft requests can be edited"))
		return
	}
	item.RequestType, item.Title, item.Description = input.RequestType, input.Title, input.Description
	item.FormData, item.ApproverID, item.ApproverName, item.LastOperationKey = input.FormData, input.ApproverID, input.ApproverName, requestKey
	updated, err := s.mutateOARequest(ctx, "update", "update", requestKey, item, input.Version, pluginsdk.AuditRiskMedium, map[string]any{"requestType": item.RequestType, "approverId": item.ApproverID})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, map[string]any{"item": updated})
}

func (s *server) submitOARequest(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	requestKey, err := mutationKey(r, "submit")
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var input oaRequestSubmitInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.Version < 1 {
		writeServiceError(w, newHTTPError(http.StatusBadRequest, "invalid_request", "current version is required"))
		return
	}
	item, err := s.getOARequest(ctx, oaRequestPermission("submit"), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if item.Status == "pending" && item.LastOperationKey == requestKey {
		workflow, workflowErr := s.host.Workflows.GetInstance(ctx, item.WorkflowInstanceID)
		if workflowErr != nil {
			writeServiceError(w, workflowErr)
			return
		}
		writeOK(w, map[string]any{"item": item, "workflow": oaWorkflowView(workflow), "duplicate": true})
		return
	}
	if item.Status != "draft" {
		writeServiceError(w, newHTTPError(http.StatusConflict, "request_not_submittable", "only draft requests can be submitted"))
		return
	}
	if input.Version != item.Version {
		writeServiceError(w, newHTTPError(http.StatusConflict, "stale_request", "request version is stale"))
		return
	}
	definitionID := item.ID + "-definition"
	instanceID := item.ID + "-workflow"
	var workflow pluginsdk.WorkflowInstance
	var updated oaRequest
	err = s.transaction(ctx, func(tx context.Context) error {
		if _, createErr := s.host.Workflows.CreateDefinition(tx, pluginsdk.WorkflowDefinitionInput{
			ID: definitionID, Key: "oa-request-" + item.ID, Name: item.Title + "审批", Version: 1,
			Nodes: []pluginsdk.WorkflowNode{
				{ID: "start", Key: "start", Name: "发起", Type: pluginsdk.WorkflowNodeStart},
				{ID: "approval", Key: "approval", Name: "审批", Type: pluginsdk.WorkflowNodeApproval, AssigneeIDs: []string{item.ApproverID}},
				{ID: "end", Key: "end", Name: "完成", Type: pluginsdk.WorkflowNodeEnd},
			},
			Transitions: []pluginsdk.WorkflowTransition{{From: "start", To: "approval"}, {From: "approval", To: "end"}},
		}); createErr != nil {
			return createErr
		}
		if _, publishErr := s.host.Workflows.PublishDefinition(tx, definitionID); publishErr != nil {
			return publishErr
		}
		var startErr error
		workflow, startErr = s.host.Workflows.Start(tx, pluginsdk.WorkflowStartInput{
			ID: instanceID, DefinitionID: definitionID, BusinessType: "oa_request", BusinessID: item.ID, Title: item.Title,
		})
		if startErr != nil {
			return startErr
		}
		item.Status, item.WorkflowDefinitionID, item.WorkflowInstanceID = "pending", definitionID, instanceID
		item.SubmittedAt, item.LastOperationKey = s.now().UTC().Format(time.RFC3339Nano), requestKey
		result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{
			Table: oaRequestTable, Operation: pluginsdk.DataMutationUpdate, Scope: oaRequestIntent("submit", item.scope),
			Key: map[string]pluginsdk.DataValue{"id": stringValue(item.ID)}, Values: oaRequestValues(item), Returning: oaRequestFields,
			IdempotencyKey: requestKey, ExpectedVersion: &input.Version,
		})
		if mutationErr != nil {
			return mutationErr
		}
		updated, mutationErr = oaRequestFromMutation(result)
		if mutationErr != nil {
			return mutationErr
		}
		return s.audit(tx, "pharma_oa.oa_request.submit", item.ID, pluginsdk.AuditRiskHigh, map[string]any{"requestType": item.RequestType, "workflowInstanceId": instanceID})
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, map[string]any{"item": updated, "workflow": oaWorkflowView(workflow), "duplicate": false})
}

func (s *server) mutateOARequest(ctx context.Context, permissionAction, auditAction, requestKey string, item oaRequest, expectedVersion int64, risk pluginsdk.AuditRisk, detail map[string]any) (oaRequest, error) {
	if expectedVersion < 1 || expectedVersion != item.Version {
		return oaRequest{}, newHTTPError(http.StatusConflict, "stale_request", "request version is stale")
	}
	var updated oaRequest
	err := s.transaction(ctx, func(tx context.Context) error {
		result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{
			Table: oaRequestTable, Operation: pluginsdk.DataMutationUpdate, Scope: oaRequestIntent(permissionAction, item.scope),
			Key: map[string]pluginsdk.DataValue{"id": stringValue(item.ID)}, Values: oaRequestValues(item), Returning: oaRequestFields,
			IdempotencyKey: requestKey, ExpectedVersion: &expectedVersion,
		})
		if mutationErr != nil {
			return mutationErr
		}
		updated, mutationErr = oaRequestFromMutation(result)
		if mutationErr != nil {
			return mutationErr
		}
		return s.audit(tx, "pharma_oa.oa_request."+auditAction, item.ID, risk, detail)
	})
	return updated, err
}

func (s *server) getOARequest(ctx context.Context, permission pluginsdk.Permission, id string) (oaRequest, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return oaRequest{}, newHTTPError(http.StatusBadRequest, "invalid_request", "request id is required")
	}
	value := stringValue(id)
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
		Table: oaRequestTable, Fields: oaRequestFields, Scope: pluginsdk.DataScopeIntent{Permission: permission},
		Filter: &pluginsdk.DataFilter{Field: "id", Operator: pluginsdk.DataOperatorEqual, Value: &value},
		Sort:   []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 1},
	})
	if err != nil {
		return oaRequest{}, err
	}
	if len(page.Records) != 1 {
		return oaRequest{}, newHTTPError(http.StatusNotFound, "request_not_found", "request was not found in the trusted scope")
	}
	return oaRequestFromRecord(page.Records[0])
}

func (s *server) queryOARequests(ctx context.Context, keyword, requestType, status string) ([]oaRequest, error) {
	filter, err := oaRequestFilter(keyword, requestType, status)
	if err != nil {
		return nil, err
	}
	items := make([]oaRequest, 0)
	cursor := ""
	for pageNumber := 0; pageNumber < 25; pageNumber++ {
		page, queryErr := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
			Table: oaRequestTable, Fields: oaRequestFields, Scope: pluginsdk.DataScopeIntent{Permission: oaRequestPermission("read")}, Filter: filter,
			Sort: []pluginsdk.DataSort{{Field: "updated_at", Direction: pluginsdk.DataSortDescending}}, Page: pluginsdk.DataPageRequest{Cursor: cursor, Limit: 200},
		})
		if queryErr != nil {
			return nil, queryErr
		}
		for _, record := range page.Records {
			item, parseErr := oaRequestFromRecord(record)
			if parseErr != nil {
				return nil, parseErr
			}
			items = append(items, item)
		}
		if !page.HasMore {
			sort.SliceStable(items, func(i, j int) bool { return items[i].UpdatedAt > items[j].UpdatedAt })
			return items, nil
		}
		cursor = page.NextCursor
	}
	return nil, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorLimitExceeded, "oa_requests", "request query exceeds 5000 records", false)
}

func oaRequestFilter(keyword, requestType, status string) (*pluginsdk.DataFilter, error) {
	filters := make([]pluginsdk.DataFilter, 0, 3)
	keyword = strings.TrimSpace(keyword)
	if len(keyword) > 100 {
		return nil, newHTTPError(http.StatusBadRequest, "invalid_request", "keyword is too long")
	}
	if keyword != "" {
		value := stringValue(keyword)
		filters = append(filters, pluginsdk.DataFilter{Any: []pluginsdk.DataFilter{
			{Field: "title", Operator: pluginsdk.DataOperatorContains, Value: &value},
			{Field: "description", Operator: pluginsdk.DataOperatorContains, Value: &value},
		}})
	}
	requestType = strings.TrimSpace(requestType)
	if requestType != "" {
		if _, ok := oaRequestTypes[requestType]; !ok {
			return nil, newHTTPError(http.StatusBadRequest, "invalid_request_type", "request type is invalid")
		}
		value := stringValue(requestType)
		filters = append(filters, pluginsdk.DataFilter{Field: "request_type", Operator: pluginsdk.DataOperatorEqual, Value: &value})
	}
	status = strings.TrimSpace(status)
	if status != "" {
		if !containsString([]string{"draft", "pending", "approved", "rejected", "withdrawn", "canceled"}, status) {
			return nil, newHTTPError(http.StatusBadRequest, "invalid_status", "request status is invalid")
		}
		value := stringValue(status)
		filters = append(filters, pluginsdk.DataFilter{Field: "status", Operator: pluginsdk.DataOperatorEqual, Value: &value})
	}
	if len(filters) == 0 {
		return nil, nil
	}
	if len(filters) == 1 {
		return &filters[0], nil
	}
	return &pluginsdk.DataFilter{All: filters}, nil
}

func validateOARequestWrite(input *oaRequestWriteInput, update bool) error {
	input.RequestType, input.Title = strings.TrimSpace(input.RequestType), strings.TrimSpace(input.Title)
	input.Description, input.ApproverID, input.ApproverName = strings.TrimSpace(input.Description), strings.TrimSpace(input.ApproverID), strings.TrimSpace(input.ApproverName)
	if _, ok := oaRequestTypes[input.RequestType]; !ok {
		return newHTTPError(http.StatusBadRequest, "invalid_request_type", "requestType must be leave, expense, procurement, contract, or custom")
	}
	if input.Title == "" || len(input.Title) > 200 || len(input.Description) > 5000 || input.ApproverID == "" || len(input.ApproverID) > 128 || input.ApproverName == "" || len(input.ApproverName) > 200 {
		return newHTTPError(http.StatusBadRequest, "invalid_oa_request", "title and approver are required and must be bounded")
	}
	if input.FormData == nil {
		return newHTTPError(http.StatusBadRequest, "invalid_oa_request", "formData is required")
	}
	raw, err := json.Marshal(input.FormData)
	if err != nil || len(raw) > 64<<10 {
		return newHTTPError(http.StatusBadRequest, "invalid_oa_request", "formData must be valid and no larger than 64 KiB")
	}
	if update && input.Version < 1 {
		return newHTTPError(http.StatusBadRequest, "invalid_oa_request", "current version is required")
	}
	return validateOARequestForm(input.RequestType, input.FormData)
}

func validateOARequestForm(requestType string, data map[string]any) error {
	requiredString := func(key string) string {
		value, ok := data[key].(string)
		if !ok {
			return ""
		}
		return strings.TrimSpace(value)
	}
	positiveAmount := func() bool {
		value, ok := data["amount"].(float64)
		return ok && value > 0
	}
	switch requestType {
	case "leave":
		start, startErr := parseDate(requiredString("startDate"))
		end, endErr := parseDate(requiredString("endDate"))
		if startErr != nil || endErr != nil || end.Before(start) || requiredString("leaveType") == "" {
			return newHTTPError(http.StatusBadRequest, "invalid_leave_request", "leaveType and a valid startDate/endDate range are required")
		}
	case "expense":
		if !positiveAmount() || requiredString("category") == "" {
			return newHTTPError(http.StatusBadRequest, "invalid_expense_request", "positive amount and category are required")
		}
	case "procurement":
		if !positiveAmount() || requiredString("purpose") == "" {
			return newHTTPError(http.StatusBadRequest, "invalid_procurement_request", "positive amount and purpose are required")
		}
	case "contract":
		if !positiveAmount() || requiredString("counterparty") == "" || requiredString("effectiveDate") == "" {
			return newHTTPError(http.StatusBadRequest, "invalid_contract_request", "positive amount, counterparty, and effectiveDate are required")
		}
		if _, err := parseDate(requiredString("effectiveDate")); err != nil {
			return newHTTPError(http.StatusBadRequest, "invalid_contract_request", "effectiveDate must be YYYY-MM-DD or RFC3339")
		}
	case "custom":
		if requiredString("formKey") == "" {
			return newHTTPError(http.StatusBadRequest, "invalid_custom_request", "formKey is required")
		}
	}
	return nil
}

func oaRequestIntent(action string, scope employeeScope) pluginsdk.DataScopeIntent {
	return pluginsdk.DataScopeIntent{Permission: oaRequestPermission(action), Filter: pluginsdk.ScopeFilter{
		TenantIDs: []string{scope.TenantID}, OrganizationIDs: []string{scope.OrganizationID}, OwnerIDs: []string{scope.OwnerID},
	}}
}

func oaRequestPermission(action string) pluginsdk.Permission {
	return pluginsdk.Permission{Resource: "pharma_oa.oa_request", Action: action}
}

func oaRequestValues(item oaRequest) map[string]pluginsdk.DataValue {
	return map[string]pluginsdk.DataValue{
		"request_type": stringValue(item.RequestType), "title": stringValue(item.Title), "description": nullableStringValue(item.Description),
		"form_data": jsonValue(item.FormData), "status": stringValue(item.Status), "approver_id": stringValue(item.ApproverID), "approver_name": stringValue(item.ApproverName),
		"workflow_definition_id": nullableStringValue(item.WorkflowDefinitionID), "workflow_instance_id": nullableStringValue(item.WorkflowInstanceID),
		"submitted_at": nullableTimestampValue(item.SubmittedAt), "last_operation_key": stringValue(item.LastOperationKey),
	}
}

func oaRequestFromMutation(result pluginsdk.DataMutationResult) (oaRequest, error) {
	if result.Record == nil || result.RowsAffected != 1 {
		return oaRequest{}, fmt.Errorf("OA request mutation returned no record")
	}
	return oaRequestFromRecord(*result.Record)
}

func oaRequestFromRecord(record pluginsdk.DataRecord) (oaRequest, error) {
	item := oaRequest{
		ID: dataString(record, "id"), RequestType: dataString(record, "request_type"), Title: dataString(record, "title"), Description: dataString(record, "description"),
		Status: dataString(record, "status"), ApproverID: dataString(record, "approver_id"), ApproverName: dataString(record, "approver_name"),
		WorkflowDefinitionID: dataString(record, "workflow_definition_id"), WorkflowInstanceID: dataString(record, "workflow_instance_id"),
		SubmittedAt: dataString(record, "submitted_at"), LastOperationKey: dataString(record, "last_operation_key"),
		Version: record.Version, CreatedAt: dataString(record, "created_at"), UpdatedAt: dataString(record, "updated_at"),
		scope:    employeeScope{TenantID: dataString(record, "tenant_id"), OrganizationID: dataString(record, "organization_id"), OwnerID: dataString(record, "owner_id")},
		FormData: map[string]any{},
	}
	if raw := dataString(record, "form_data"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &item.FormData); err != nil {
			return oaRequest{}, fmt.Errorf("decode OA request form data: %w", err)
		}
	}
	return item, nil
}

func oaWorkflowView(instance pluginsdk.WorkflowInstance) map[string]any {
	tasks := make([]map[string]any, 0, len(instance.Tasks))
	for _, task := range instance.Tasks {
		tasks = append(tasks, map[string]any{
			"id": task.ID, "instanceId": task.InstanceID, "nodeId": task.NodeID,
			"assignee": map[string]any{"id": task.Assignee.ID, "name": task.Assignee.Name},
			"status":   task.Status, "createdAt": task.CreatedAt, "completedAt": task.CompletedAt,
		})
	}
	timeline := make([]map[string]any, 0, len(instance.Timeline))
	for _, action := range instance.Timeline {
		timeline = append(timeline, map[string]any{
			"id": action.ID, "type": action.Type, "instanceId": action.InstanceID, "taskId": action.TaskID, "nodeId": action.NodeID,
			"actor": map[string]any{"id": action.Actor.ID, "name": action.Actor.Name}, "target": map[string]any{"id": action.Target.ID, "name": action.Target.Name},
			"comment": action.Comment, "createdAt": action.CreatedAt,
		})
	}
	return map[string]any{
		"id": instance.ID, "definitionId": instance.DefinitionID, "definitionKey": instance.DefinitionKey,
		"businessType": instance.BusinessType, "businessId": instance.BusinessID, "title": instance.Title,
		"status": instance.Status, "starter": map[string]any{"id": instance.Starter.ID, "name": instance.Starter.Name},
		"currentNode": instance.CurrentNode, "tasks": tasks, "timeline": timeline, "createdAt": instance.CreatedAt, "updatedAt": instance.UpdatedAt,
	}
}

package hostservice

import (
	"context"
	"testing"

	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"github.com/tinboxw/skoll/pkg/security"
)

func TestWorkflowServiceBindsIdentityAndNamespace(t *testing.T) {
	backend := workflowsvc.NewService(workflowsvc.NewMemoryRepository())
	audit, err := NewAuditService("plugin_a", auditsvc.NewService(clickhouse.NewAuditStore()))
	if err != nil {
		t.Fatalf("NewAuditService error: %v", err)
	}
	service, err := NewWorkflowService("plugin_a", backend, audit)
	if err != nil {
		t.Fatalf("NewWorkflowService error: %v", err)
	}
	ctx := security.WithJWTClaimsContext(context.Background(), &security.JWTClaims{Subject: "approver-1"})
	definition, err := service.CreateDefinition(ctx, pluginsdk.WorkflowDefinitionInput{
		ID: "approval", Key: "approval", Name: "Approval", Version: 1,
		Nodes: []pluginsdk.WorkflowNode{
			{ID: "start", Key: "start", Name: "Start", Type: pluginsdk.WorkflowNodeStart},
			{ID: "review", Key: "review", Name: "Review", Type: pluginsdk.WorkflowNodeApproval, AssigneeIDs: []string{"approver-1"}, Decision: &pluginsdk.WorkflowDecisionRule{Strategy: pluginsdk.WorkflowDecisionAny, Quorum: 1}},
			{ID: "end", Key: "end", Name: "End", Type: pluginsdk.WorkflowNodeEnd},
		},
		Transitions: []pluginsdk.WorkflowTransition{{From: "start", To: "review"}, {From: "review", To: "end"}},
	})
	if err != nil || definition.ID != "approval" || definition.Key != "approval" {
		t.Fatalf("CreateDefinition item=%+v err=%v", definition, err)
	}
	if _, err = service.PublishDefinition(ctx, definition.ID); err != nil {
		t.Fatalf("PublishDefinition error: %v", err)
	}
	instance, err := service.Start(ctx, pluginsdk.WorkflowStartInput{ID: "request", DefinitionID: definition.ID, BusinessType: "request", BusinessID: "1", Title: "Request"})
	if err != nil || instance.Starter.ID != "approver-1" || len(instance.Tasks) != 1 {
		t.Fatalf("Start item=%+v err=%v", instance, err)
	}
	approved, err := service.Approve(ctx, pluginsdk.WorkflowTaskActionInput{InstanceID: instance.ID, TaskID: instance.Tasks[0].ID})
	if err != nil || approved.Status != pluginsdk.WorkflowInstanceApproved {
		t.Fatalf("Approve item=%+v err=%v", approved, err)
	}
	otherAudit, _ := NewAuditService("plugin_b", auditsvc.NewService(clickhouse.NewAuditStore()))
	other, err := NewWorkflowService("plugin_b", backend, otherAudit)
	if err != nil {
		t.Fatalf("NewWorkflowService other error: %v", err)
	}
	if _, err := other.GetDefinition(ctx, "approval"); err == nil {
		t.Fatal("another plugin read a foreign workflow definition")
	}
}

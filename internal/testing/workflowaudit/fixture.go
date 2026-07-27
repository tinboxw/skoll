package workflowaudit

import (
	"fmt"
	"sort"
	"strings"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
)

type Fixture struct {
	Definition domainworkflow.Definition
	Instance   domainworkflow.Instance
	Events     []*domainaudit.Event
}

type Query struct {
	ActorID      string
	Action       domainaudit.AuditAction
	ResourceType string
	ResourceID   string
	Result       domainaudit.EventResult
}

func BuildApprovalChainFixture() (Fixture, error) {
	base := time.Date(2026, time.July, 4, 9, 0, 0, 0, time.UTC)
	definition, err := domainworkflow.NewDefinition(
		"wf-def-purchase",
		"pharma.oa.purchase.approval",
		"Purchase Approval",
		1,
		[]domainworkflow.Node{
			{ID: "start", Key: "start", Name: "Start", Type: domainworkflow.NodeStart},
			{ID: "approval", Key: "approval", Name: "Approval", Type: domainworkflow.NodeApproval, Assignees: []shared.ID{"approver-1"}, Decision: domainworkflow.DecisionRule{Strategy: domainworkflow.DecisionAny, Quorum: 1}},
			{ID: "end", Key: "end", Name: "End", Type: domainworkflow.NodeEnd},
		},
		[]domainworkflow.Transition{
			{From: "start", To: "approval"},
			{From: "approval", To: "end"},
		},
		base,
	)
	if err != nil {
		return Fixture{}, err
	}
	if err := definition.Publish(base.Add(time.Minute)); err != nil {
		return Fixture{}, err
	}

	instance, err := domainworkflow.Start(domainworkflow.StartInput{
		ID:           "wf-purchase-001",
		Definition:   *definition,
		BusinessType: "oa.purchase",
		BusinessID:   "purchase-001",
		Title:        "Purchase Request",
		Starter:      domainworkflow.Actor{ID: "starter-1", Name: "Starter One"},
		Now:          base.Add(2 * time.Minute),
	})
	if err != nil {
		return Fixture{}, err
	}
	firstTask := pendingTask(*instance, "approver-1")
	if firstTask.ID.IsZero() {
		return Fixture{}, fmt.Errorf("fixture missing first pending task")
	}
	if err := instance.Copy(firstTask.ID, domainworkflow.Actor{ID: "approver-1", Name: "Approver One"}, domainworkflow.Actor{ID: "observer-1", Name: "Observer One"}, "copy for awareness", base.Add(3*time.Minute)); err != nil {
		return Fixture{}, err
	}
	if err := instance.Transfer(firstTask.ID, domainworkflow.Actor{ID: "approver-1", Name: "Approver One"}, domainworkflow.Actor{ID: "approver-2", Name: "Approver Two"}, "transfer to backup approver", base.Add(4*time.Minute)); err != nil {
		return Fixture{}, err
	}
	secondTask := pendingTask(*instance, "approver-2")
	if secondTask.ID.IsZero() {
		return Fixture{}, fmt.Errorf("fixture missing transferred pending task")
	}
	if err := instance.Approve(*definition, secondTask.ID, domainworkflow.Actor{ID: "approver-2", Name: "Approver Two"}, "approved with audit fixture", base.Add(5*time.Minute)); err != nil {
		return Fixture{}, err
	}

	events, err := EventsFromInstance(*instance)
	if err != nil {
		return Fixture{}, err
	}
	return Fixture{Definition: *definition, Instance: *instance, Events: events}, nil
}

func EventsFromInstance(instance domainworkflow.Instance) ([]*domainaudit.Event, error) {
	actions := append([]domainworkflow.Action(nil), instance.Timeline...)
	sort.Slice(actions, func(i, j int) bool {
		return actions[i].CreatedAt.Before(actions[j].CreatedAt)
	})
	events := make([]*domainaudit.Event, 0, len(actions))
	for _, action := range actions {
		event, err := eventFromAction(instance, action)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

func FindEvents(events []*domainaudit.Event, query Query) []*domainaudit.Event {
	out := make([]*domainaudit.Event, 0, len(events))
	for _, event := range events {
		if event == nil {
			continue
		}
		if query.ActorID != "" && event.Actor.ID.String() != query.ActorID {
			continue
		}
		if query.Action != "" && event.Action != query.Action {
			continue
		}
		if query.ResourceType != "" && event.Resource.Type != query.ResourceType {
			continue
		}
		if query.ResourceID != "" && event.Resource.ID != query.ResourceID {
			continue
		}
		if query.Result != "" && event.Result != query.Result {
			continue
		}
		out = append(out, event)
	}
	return out
}

func eventFromAction(instance domainworkflow.Instance, action domainworkflow.Action) (*domainaudit.Event, error) {
	auditAction := auditActionForWorkflow(action.Type)
	metadata := map[string]any{
		"workflowAction": string(action.Type),
		"definitionKey":  instance.DefinitionKey,
		"businessType":   instance.BusinessType,
		"businessId":     instance.BusinessID,
		"status":         string(instance.Status),
	}
	if strings.TrimSpace(action.Comment) != "" {
		metadata["comment"] = action.Comment
	}
	if !action.Target.ID.IsZero() {
		metadata["targetActorId"] = action.Target.ID.String()
		metadata["targetActorName"] = action.Target.Name
	}
	return domainaudit.NewEvent(domainaudit.EventInput{
		ID:     shared.ID(fmt.Sprintf("audit-%s-%s", instance.ID, action.Type)),
		Type:   domainaudit.EventTypeOperation,
		Action: auditAction,
		Actor: domainaudit.ActorRef{
			Type: "user",
			ID:   action.Actor.ID,
			Name: action.Actor.Name,
		},
		Resource: domainaudit.ResourceRef{
			Type: "workflow_instance",
			ID:   instance.ID.String(),
			Name: instance.Title,
		},
		Result: domainaudit.EventResultSuccess,
		Trace: domainaudit.TraceContext{
			TraceID:   fmt.Sprintf("trace-%s", instance.ID),
			RequestID: action.ID.String(),
			Method:    "POST",
			Path:      fmt.Sprintf("/skoll/v1/workflows/instances/%s/%s", instance.ID, action.Type),
		},
		Risk:       riskForWorkflow(action.Type),
		Metadata:   metadata,
		SourceData: sourceDataForWorkflow(instance, action),
		OccurredAt: action.CreatedAt,
	})
}

func auditActionForWorkflow(action domainworkflow.ActionType) domainaudit.AuditAction {
	switch action {
	case domainworkflow.ActionStart:
		return domainaudit.AuditAction("workflow.instance.start")
	case domainworkflow.ActionApprove:
		return domainaudit.AuditAction("workflow.task.approve")
	case domainworkflow.ActionReject:
		return domainaudit.AuditAction("workflow.task.reject")
	case domainworkflow.ActionWithdraw:
		return domainaudit.AuditAction("workflow.instance.withdraw")
	case domainworkflow.ActionTransfer:
		return domainaudit.AuditAction("workflow.task.transfer")
	case domainworkflow.ActionCopy:
		return domainaudit.AuditAction("workflow.task.copy")
	default:
		return domainaudit.AuditAction("workflow.task.update")
	}
}

func riskForWorkflow(action domainworkflow.ActionType) domainaudit.EventRisk {
	switch action {
	case domainworkflow.ActionReject, domainworkflow.ActionWithdraw, domainworkflow.ActionTransfer:
		return domainaudit.EventRiskMedium
	default:
		return domainaudit.EventRiskLow
	}
}

func sourceDataForWorkflow(instance domainworkflow.Instance, action domainworkflow.Action) map[string]any {
	return map[string]any{
		"kind":          "workflow_audit_replay",
		"instanceId":    instance.ID.String(),
		"definitionId":  instance.DefinitionID.String(),
		"definitionKey": instance.DefinitionKey,
		"businessType":  instance.BusinessType,
		"businessId":    instance.BusinessID,
		"title":         instance.Title,
		"actionId":      action.ID.String(),
		"action":        string(action.Type),
		"taskId":        action.TaskID.String(),
		"nodeId":        action.NodeID.String(),
		"actorId":       action.Actor.ID.String(),
		"actorName":     action.Actor.Name,
		"targetId":      action.Target.ID.String(),
		"targetName":    action.Target.Name,
		"comment":       action.Comment,
		"occurredAt":    action.CreatedAt.Format(time.RFC3339Nano),
	}
}

func pendingTask(instance domainworkflow.Instance, actorID string) domainworkflow.Task {
	for _, task := range instance.Tasks {
		if task.Assignee.ID.String() == actorID && task.Status == domainworkflow.TaskPending {
			return task
		}
	}
	return domainworkflow.Task{}
}

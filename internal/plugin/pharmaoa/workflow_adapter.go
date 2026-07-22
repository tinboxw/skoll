package pharmaoa

import (
	"context"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type hostWorkflowAdapter struct {
	workflow pluginsdk.WorkflowService
}

func newHostWorkflowAdapter(workflow pluginsdk.WorkflowService) *hostWorkflowAdapter {
	return &hostWorkflowAdapter{workflow: workflow}
}

func (a *hostWorkflowAdapter) CreateDefinition(ctx context.Context, in workflowsvc.CreateDefinitionInput) (*domainworkflow.Definition, error) {
	nodes := make([]pluginsdk.WorkflowNode, 0, len(in.Nodes))
	for _, node := range in.Nodes {
		assignees := make([]string, len(node.Assignees))
		for index := range node.Assignees {
			assignees[index] = node.Assignees[index].String()
		}
		nodes = append(nodes, pluginsdk.WorkflowNode{
			ID: node.ID.String(), Key: node.Key, Name: node.Name, Type: pluginsdk.WorkflowNodeType(node.Type), AssigneeIDs: assignees,
		})
	}
	transitions := make([]pluginsdk.WorkflowTransition, 0, len(in.Transitions))
	for _, transition := range in.Transitions {
		transitions = append(transitions, pluginsdk.WorkflowTransition{From: transition.From.String(), To: transition.To.String()})
	}
	item, err := a.workflow.CreateDefinition(ctx, pluginsdk.WorkflowDefinitionInput{
		ID: in.ID.String(), Key: in.Key, Name: in.Name, Version: in.Version, Nodes: nodes, Transitions: transitions,
	})
	if err != nil {
		return nil, err
	}
	return domainWorkflowDefinition(item), nil
}

func (a *hostWorkflowAdapter) GetDefinition(ctx context.Context, id shared.ID) (*domainworkflow.Definition, error) {
	item, err := a.workflow.GetDefinition(ctx, id.String())
	if err != nil {
		return nil, err
	}
	return domainWorkflowDefinition(item), nil
}

func (a *hostWorkflowAdapter) PublishDefinition(ctx context.Context, id shared.ID, _ time.Time) (*domainworkflow.Definition, error) {
	item, err := a.workflow.PublishDefinition(ctx, id.String())
	if err != nil {
		return nil, err
	}
	return domainWorkflowDefinition(item), nil
}

func (a *hostWorkflowAdapter) Start(ctx context.Context, in workflowsvc.StartInput) (*domainworkflow.Instance, error) {
	item, err := a.workflow.Start(ctx, pluginsdk.WorkflowStartInput{
		ID: in.ID.String(), DefinitionID: in.DefinitionID.String(), BusinessType: pluginAuditName(in.BusinessType, "business"),
		BusinessID: in.BusinessID, Title: in.Title,
	})
	if err != nil {
		return nil, err
	}
	return domainWorkflowInstance(item), nil
}

func (a *hostWorkflowAdapter) GetInstance(ctx context.Context, id shared.ID) (*domainworkflow.Instance, error) {
	item, err := a.workflow.GetInstance(ctx, id.String())
	if err != nil {
		return nil, err
	}
	return domainWorkflowInstance(item), nil
}

func (a *hostWorkflowAdapter) Approve(ctx context.Context, in workflowsvc.TaskActionInput) (*domainworkflow.Instance, error) {
	item, err := a.workflow.Approve(ctx, pluginsdk.WorkflowTaskActionInput{InstanceID: in.InstanceID.String(), TaskID: in.TaskID.String(), Comment: in.Comment})
	if err != nil {
		return nil, err
	}
	return domainWorkflowInstance(item), nil
}

func (a *hostWorkflowAdapter) Reject(ctx context.Context, in workflowsvc.TaskActionInput) (*domainworkflow.Instance, error) {
	item, err := a.workflow.Reject(ctx, pluginsdk.WorkflowTaskActionInput{InstanceID: in.InstanceID.String(), TaskID: in.TaskID.String(), Comment: in.Comment})
	if err != nil {
		return nil, err
	}
	return domainWorkflowInstance(item), nil
}

func (a *hostWorkflowAdapter) Withdraw(ctx context.Context, in workflowsvc.InstanceActionInput) (*domainworkflow.Instance, error) {
	item, err := a.workflow.Withdraw(ctx, pluginsdk.WorkflowInstanceActionInput{InstanceID: in.InstanceID.String(), Comment: in.Comment})
	if err != nil {
		return nil, err
	}
	return domainWorkflowInstance(item), nil
}

func (a *hostWorkflowAdapter) Cancel(ctx context.Context, in workflowsvc.InstanceActionInput) (*domainworkflow.Instance, error) {
	item, err := a.workflow.Cancel(ctx, pluginsdk.WorkflowInstanceActionInput{InstanceID: in.InstanceID.String(), Comment: in.Comment})
	if err != nil {
		return nil, err
	}
	return domainWorkflowInstance(item), nil
}

func (a *hostWorkflowAdapter) Transfer(ctx context.Context, in workflowsvc.TaskTargetActionInput) (*domainworkflow.Instance, error) {
	item, err := a.workflow.Transfer(ctx, pluginsdk.WorkflowTargetActionInput{
		InstanceID: in.InstanceID.String(), TaskID: in.TaskID.String(),
		Target: pluginsdk.WorkflowActor{ID: in.Target.ID.String(), Name: in.Target.Name}, Comment: in.Comment,
	})
	if err != nil {
		return nil, err
	}
	return domainWorkflowInstance(item), nil
}

func (a *hostWorkflowAdapter) Copy(ctx context.Context, in workflowsvc.TaskTargetActionInput) (*domainworkflow.Instance, error) {
	item, err := a.workflow.Copy(ctx, pluginsdk.WorkflowTargetActionInput{
		InstanceID: in.InstanceID.String(), TaskID: in.TaskID.String(),
		Target: pluginsdk.WorkflowActor{ID: in.Target.ID.String(), Name: in.Target.Name}, Comment: in.Comment,
	})
	if err != nil {
		return nil, err
	}
	return domainWorkflowInstance(item), nil
}

func domainWorkflowDefinition(item pluginsdk.WorkflowDefinition) *domainworkflow.Definition {
	nodes := make([]domainworkflow.Node, 0, len(item.Nodes))
	for _, node := range item.Nodes {
		assignees := make([]shared.ID, len(node.AssigneeIDs))
		for index := range node.AssigneeIDs {
			assignees[index] = shared.ID(node.AssigneeIDs[index])
		}
		nodes = append(nodes, domainworkflow.Node{
			ID: shared.ID(node.ID), Key: node.Key, Name: node.Name, Type: domainworkflow.NodeType(node.Type), Assignees: assignees,
		})
	}
	transitions := make([]domainworkflow.Transition, 0, len(item.Transitions))
	for _, transition := range item.Transitions {
		transitions = append(transitions, domainworkflow.Transition{From: shared.ID(transition.From), To: shared.ID(transition.To)})
	}
	return &domainworkflow.Definition{
		ID: shared.ID(item.ID), Key: item.Key, Name: item.Name, Version: item.Version,
		Status: domainworkflow.DefinitionStatus(item.Status), Nodes: nodes, Transitions: transitions,
		Meta: shared.AuditMeta{CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt},
	}
}

func domainWorkflowInstance(item pluginsdk.WorkflowInstance) *domainworkflow.Instance {
	tasks := make([]domainworkflow.Task, 0, len(item.Tasks))
	for _, task := range item.Tasks {
		tasks = append(tasks, domainworkflow.Task{
			ID: shared.ID(task.ID), InstanceID: shared.ID(task.InstanceID), NodeID: shared.ID(task.NodeID),
			Assignee: domainWorkflowActor(task.Assignee), Status: domainworkflow.TaskStatus(task.Status),
			CreatedAt: task.CreatedAt, CompletedAt: cloneWorkflowTime(task.CompletedAt),
		})
	}
	timeline := make([]domainworkflow.Action, 0, len(item.Timeline))
	for _, action := range item.Timeline {
		timeline = append(timeline, domainworkflow.Action{
			ID: shared.ID(action.ID), Type: domainworkflow.ActionType(action.Type), InstanceID: shared.ID(action.InstanceID),
			TaskID: shared.ID(action.TaskID), NodeID: shared.ID(action.NodeID), Actor: domainWorkflowActor(action.Actor),
			Target: domainWorkflowActor(action.Target), Comment: action.Comment, CreatedAt: action.CreatedAt,
		})
	}
	return &domainworkflow.Instance{
		ID: shared.ID(item.ID), DefinitionID: shared.ID(item.DefinitionID), DefinitionKey: item.DefinitionKey,
		BusinessType: PluginID + "." + item.BusinessType, BusinessID: item.BusinessID, Title: item.Title,
		Status: domainworkflow.InstanceStatus(item.Status), Starter: domainWorkflowActor(item.Starter),
		CurrentNode: shared.ID(item.CurrentNode), Tasks: tasks, Timeline: timeline,
		Meta: shared.AuditMeta{CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt},
	}
}

func domainWorkflowActor(actor pluginsdk.WorkflowActor) domainworkflow.Actor {
	return domainworkflow.Actor{ID: shared.ID(actor.ID), Name: actor.Name}
}

func cloneWorkflowTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

var _ workflowsvc.Service = (*hostWorkflowAdapter)(nil)

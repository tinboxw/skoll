package hostservice

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type workflowService struct {
	pluginID string
	workflow workflowsvc.Service
	audit    pluginsdk.AuditService
	now      func() time.Time
}

var pluginLocalIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.:@-]{0,511}$`)

func NewWorkflowService(pluginID string, workflow workflowsvc.Service, audit pluginsdk.AuditService) (pluginsdk.WorkflowService, error) {
	pluginID = strings.ToLower(strings.TrimSpace(pluginID))
	if pluginID == "" {
		return nil, fmt.Errorf("plugin host identity is required")
	}
	if workflow == nil || audit == nil {
		return nil, fmt.Errorf("plugin host workflow dependencies are required")
	}
	return &workflowService{pluginID: pluginID, workflow: workflow, audit: audit, now: func() time.Time { return time.Now().UTC() }}, nil
}

func (s *workflowService) CreateDefinition(ctx context.Context, input pluginsdk.WorkflowDefinitionInput) (pluginsdk.WorkflowDefinition, error) {
	id, err := s.boundID(input.ID, "workflow definition")
	if err != nil {
		return pluginsdk.WorkflowDefinition{}, err
	}
	key, err := s.boundName(input.Key, "workflow definition key")
	if err != nil {
		return pluginsdk.WorkflowDefinition{}, err
	}
	nodes := make([]domainworkflow.Node, 0, len(input.Nodes))
	for _, item := range input.Nodes {
		nodeID, bindErr := validatePluginLocalID(item.ID, "workflow node")
		if bindErr != nil {
			return pluginsdk.WorkflowDefinition{}, bindErr
		}
		nodeKey, bindErr := validatePluginName(item.Key, "workflow node key")
		if bindErr != nil {
			return pluginsdk.WorkflowDefinition{}, bindErr
		}
		assignees := make([]shared.ID, 0, len(item.AssigneeIDs))
		for _, assignee := range item.AssigneeIDs {
			assignee = strings.TrimSpace(assignee)
			if assignee == "" {
				return pluginsdk.WorkflowDefinition{}, fmt.Errorf("workflow assignee is required")
			}
			assignees = append(assignees, shared.ID(assignee))
		}
		nodes = append(nodes, domainworkflow.Node{
			ID: shared.ID(nodeID), Key: nodeKey, Name: strings.TrimSpace(item.Name),
			Type: domainworkflow.NodeType(item.Type), Assignees: assignees,
		})
	}
	transitions := make([]domainworkflow.Transition, 0, len(input.Transitions))
	for _, item := range input.Transitions {
		from, bindErr := validatePluginLocalID(item.From, "workflow transition source")
		if bindErr != nil {
			return pluginsdk.WorkflowDefinition{}, bindErr
		}
		to, bindErr := validatePluginLocalID(item.To, "workflow transition target")
		if bindErr != nil {
			return pluginsdk.WorkflowDefinition{}, bindErr
		}
		transitions = append(transitions, domainworkflow.Transition{From: shared.ID(from), To: shared.ID(to)})
	}
	item, err := s.workflow.CreateDefinition(ctx, workflowsvc.CreateDefinitionInput{
		ID: shared.ID(id), Key: key, Name: strings.TrimSpace(input.Name), Version: input.Version,
		Nodes: nodes, Transitions: transitions, Now: s.now(),
	})
	if err != nil {
		return pluginsdk.WorkflowDefinition{}, err
	}
	if err := s.record(ctx, "workflow.definition.create", input.ID, map[string]any{"key": input.Key, "version": input.Version}); err != nil {
		return pluginsdk.WorkflowDefinition{}, err
	}
	return s.definition(*item)
}

func (s *workflowService) GetDefinition(ctx context.Context, id string) (pluginsdk.WorkflowDefinition, error) {
	bound, err := s.boundID(id, "workflow definition")
	if err != nil {
		return pluginsdk.WorkflowDefinition{}, err
	}
	item, err := s.workflow.GetDefinition(ctx, shared.ID(bound))
	if err != nil {
		return pluginsdk.WorkflowDefinition{}, err
	}
	return s.definition(*item)
}

func (s *workflowService) PublishDefinition(ctx context.Context, id string) (pluginsdk.WorkflowDefinition, error) {
	bound, err := s.boundID(id, "workflow definition")
	if err != nil {
		return pluginsdk.WorkflowDefinition{}, err
	}
	item, err := s.workflow.PublishDefinition(ctx, shared.ID(bound), s.now())
	if err != nil {
		return pluginsdk.WorkflowDefinition{}, err
	}
	if err := s.record(ctx, "workflow.definition.publish", id, nil); err != nil {
		return pluginsdk.WorkflowDefinition{}, err
	}
	return s.definition(*item)
}

func (s *workflowService) Start(ctx context.Context, input pluginsdk.WorkflowStartInput) (pluginsdk.WorkflowInstance, error) {
	id, err := s.boundID(input.ID, "workflow instance")
	if err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	definitionID, err := s.boundID(input.DefinitionID, "workflow definition")
	if err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	businessType, err := s.boundName(input.BusinessType, "workflow business type")
	if err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	actor := trustedHostActor(ctx, s.pluginID)
	item, err := s.workflow.Start(ctx, workflowsvc.StartInput{
		ID: shared.ID(id), DefinitionID: shared.ID(definitionID), BusinessType: businessType,
		BusinessID: strings.TrimSpace(input.BusinessID), Title: strings.TrimSpace(input.Title),
		Starter: domainworkflow.Actor{ID: shared.ID(actor.id), Name: actor.name}, Now: s.now(),
	})
	if err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	if err := s.record(ctx, "workflow.instance.start", input.ID, map[string]any{"businessType": input.BusinessType, "businessId": input.BusinessID}); err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	return s.instance(*item)
}

func (s *workflowService) GetInstance(ctx context.Context, id string) (pluginsdk.WorkflowInstance, error) {
	bound, err := s.boundID(id, "workflow instance")
	if err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	item, err := s.workflow.GetInstance(ctx, shared.ID(bound))
	if err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	return s.instance(*item)
}

func (s *workflowService) Approve(ctx context.Context, input pluginsdk.WorkflowTaskActionInput) (pluginsdk.WorkflowInstance, error) {
	return s.taskAction(ctx, "approve", input)
}

func (s *workflowService) Reject(ctx context.Context, input pluginsdk.WorkflowTaskActionInput) (pluginsdk.WorkflowInstance, error) {
	return s.taskAction(ctx, "reject", input)
}

func (s *workflowService) Withdraw(ctx context.Context, input pluginsdk.WorkflowInstanceActionInput) (pluginsdk.WorkflowInstance, error) {
	return s.instanceAction(ctx, "withdraw", input)
}

func (s *workflowService) Cancel(ctx context.Context, input pluginsdk.WorkflowInstanceActionInput) (pluginsdk.WorkflowInstance, error) {
	return s.instanceAction(ctx, "cancel", input)
}

func (s *workflowService) instanceAction(ctx context.Context, action string, input pluginsdk.WorkflowInstanceActionInput) (pluginsdk.WorkflowInstance, error) {
	instanceID, err := s.boundID(input.InstanceID, "workflow instance")
	if err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	actor := trustedHostActor(ctx, s.pluginID)
	serviceInput := workflowsvc.InstanceActionInput{
		InstanceID: shared.ID(instanceID), Actor: domainworkflow.Actor{ID: shared.ID(actor.id), Name: actor.name},
		Comment: strings.TrimSpace(input.Comment), Now: s.now(),
	}
	var item *domainworkflow.Instance
	if action == "withdraw" {
		item, err = s.workflow.Withdraw(ctx, serviceInput)
	} else {
		item, err = s.workflow.Cancel(ctx, serviceInput)
	}
	if err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	if err := s.record(ctx, "workflow.instance."+action, input.InstanceID, nil); err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	return s.instance(*item)
}

func (s *workflowService) Transfer(ctx context.Context, input pluginsdk.WorkflowTargetActionInput) (pluginsdk.WorkflowInstance, error) {
	return s.targetAction(ctx, "transfer", input)
}

func (s *workflowService) Copy(ctx context.Context, input pluginsdk.WorkflowTargetActionInput) (pluginsdk.WorkflowInstance, error) {
	return s.targetAction(ctx, "copy", input)
}

func (s *workflowService) taskAction(ctx context.Context, action string, input pluginsdk.WorkflowTaskActionInput) (pluginsdk.WorkflowInstance, error) {
	instanceID, err := s.boundID(input.InstanceID, "workflow instance")
	if err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	taskID, err := s.boundID(input.TaskID, "workflow task")
	if err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	actor := trustedHostActor(ctx, s.pluginID)
	serviceInput := workflowsvc.TaskActionInput{
		InstanceID: shared.ID(instanceID), TaskID: shared.ID(taskID),
		Actor: domainworkflow.Actor{ID: shared.ID(actor.id), Name: actor.name}, Comment: strings.TrimSpace(input.Comment), Now: s.now(),
	}
	var item *domainworkflow.Instance
	if action == "approve" {
		item, err = s.workflow.Approve(ctx, serviceInput)
	} else {
		item, err = s.workflow.Reject(ctx, serviceInput)
	}
	if err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	if err := s.record(ctx, "workflow.task."+action, input.InstanceID, map[string]any{"taskId": input.TaskID}); err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	return s.instance(*item)
}

func (s *workflowService) targetAction(ctx context.Context, action string, input pluginsdk.WorkflowTargetActionInput) (pluginsdk.WorkflowInstance, error) {
	instanceID, err := s.boundID(input.InstanceID, "workflow instance")
	if err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	taskID, err := s.boundID(input.TaskID, "workflow task")
	if err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	if strings.TrimSpace(input.Target.ID) == "" {
		return pluginsdk.WorkflowInstance{}, fmt.Errorf("workflow target is required")
	}
	actor := trustedHostActor(ctx, s.pluginID)
	serviceInput := workflowsvc.TaskTargetActionInput{
		InstanceID: shared.ID(instanceID), TaskID: shared.ID(taskID),
		Actor:   domainworkflow.Actor{ID: shared.ID(actor.id), Name: actor.name},
		Target:  domainworkflow.Actor{ID: shared.ID(strings.TrimSpace(input.Target.ID)), Name: strings.TrimSpace(input.Target.Name)},
		Comment: strings.TrimSpace(input.Comment), Now: s.now(),
	}
	var item *domainworkflow.Instance
	if action == "transfer" {
		item, err = s.workflow.Transfer(ctx, serviceInput)
	} else {
		item, err = s.workflow.Copy(ctx, serviceInput)
	}
	if err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	if err := s.record(ctx, "workflow.task."+action, input.InstanceID, map[string]any{"taskId": input.TaskID, "targetId": input.Target.ID}); err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	return s.instance(*item)
}

func (s *workflowService) boundID(value, label string) (string, error) {
	value, err := validatePluginLocalID(value, label)
	if err != nil {
		return "", err
	}
	return "plugin:" + s.pluginID + ":" + value, nil
}

func (s *workflowService) localID(value, label string) (string, error) {
	prefix := "plugin:" + s.pluginID + ":"
	if !strings.HasPrefix(value, prefix) {
		return "", fmt.Errorf("%s is outside plugin namespace", label)
	}
	return validatePluginLocalID(strings.TrimPrefix(value, prefix), label)
}

func (s *workflowService) boundName(value, label string) (string, error) {
	value, err := validatePluginName(value, label)
	if err != nil {
		return "", err
	}
	return s.pluginID + "." + value, nil
}

func (s *workflowService) localName(value, label string) (string, error) {
	prefix := s.pluginID + "."
	if !strings.HasPrefix(value, prefix) {
		return "", fmt.Errorf("%s is outside plugin namespace", label)
	}
	return strings.TrimPrefix(value, prefix), nil
}

func (s *workflowService) definition(item domainworkflow.Definition) (pluginsdk.WorkflowDefinition, error) {
	id, err := s.localID(item.ID.String(), "workflow definition")
	if err != nil {
		return pluginsdk.WorkflowDefinition{}, err
	}
	key, err := s.localName(item.Key, "workflow definition key")
	if err != nil {
		return pluginsdk.WorkflowDefinition{}, err
	}
	nodes := make([]pluginsdk.WorkflowNode, 0, len(item.Nodes))
	for _, node := range item.Nodes {
		nodeID, convertErr := validatePluginLocalID(node.ID.String(), "workflow node")
		if convertErr != nil {
			return pluginsdk.WorkflowDefinition{}, convertErr
		}
		nodeKey, convertErr := validatePluginName(node.Key, "workflow node key")
		if convertErr != nil {
			return pluginsdk.WorkflowDefinition{}, convertErr
		}
		assignees := make([]string, len(node.Assignees))
		for index := range node.Assignees {
			assignees[index] = node.Assignees[index].String()
		}
		nodes = append(nodes, pluginsdk.WorkflowNode{ID: nodeID, Key: nodeKey, Name: node.Name, Type: pluginsdk.WorkflowNodeType(node.Type), AssigneeIDs: assignees})
	}
	transitions := make([]pluginsdk.WorkflowTransition, 0, len(item.Transitions))
	for _, transition := range item.Transitions {
		from, convertErr := validatePluginLocalID(transition.From.String(), "workflow transition source")
		if convertErr != nil {
			return pluginsdk.WorkflowDefinition{}, convertErr
		}
		to, convertErr := validatePluginLocalID(transition.To.String(), "workflow transition target")
		if convertErr != nil {
			return pluginsdk.WorkflowDefinition{}, convertErr
		}
		transitions = append(transitions, pluginsdk.WorkflowTransition{From: from, To: to})
	}
	return pluginsdk.WorkflowDefinition{
		ID: id, Key: key, Name: item.Name, Version: item.Version, Status: pluginsdk.WorkflowDefinitionStatus(item.Status),
		Nodes: nodes, Transitions: transitions, CreatedAt: item.Meta.CreatedAt, UpdatedAt: item.Meta.UpdatedAt,
	}, nil
}

func (s *workflowService) instance(item domainworkflow.Instance) (pluginsdk.WorkflowInstance, error) {
	id, err := s.localID(item.ID.String(), "workflow instance")
	if err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	definitionID, err := s.localID(item.DefinitionID.String(), "workflow definition")
	if err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	definitionKey, err := s.localName(item.DefinitionKey, "workflow definition key")
	if err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	businessType, err := s.localName(item.BusinessType, "workflow business type")
	if err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	currentNode, err := validatePluginLocalID(item.CurrentNode.String(), "workflow node")
	if err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	tasks := make([]pluginsdk.WorkflowTask, 0, len(item.Tasks))
	for _, task := range item.Tasks {
		taskID, convertErr := s.localID(task.ID.String(), "workflow task")
		if convertErr != nil {
			return pluginsdk.WorkflowInstance{}, convertErr
		}
		nodeID, convertErr := validatePluginLocalID(task.NodeID.String(), "workflow node")
		if convertErr != nil {
			return pluginsdk.WorkflowInstance{}, convertErr
		}
		tasks = append(tasks, pluginsdk.WorkflowTask{
			ID: taskID, InstanceID: id, NodeID: nodeID, Assignee: workflowActor(task.Assignee),
			Status: pluginsdk.WorkflowTaskStatus(task.Status), CreatedAt: task.CreatedAt, CompletedAt: cloneTime(task.CompletedAt),
		})
	}
	timeline := make([]pluginsdk.WorkflowAction, 0, len(item.Timeline))
	for _, action := range item.Timeline {
		actionID, convertErr := s.localID(action.ID.String(), "workflow action")
		if convertErr != nil {
			return pluginsdk.WorkflowInstance{}, convertErr
		}
		taskID, convertErr := s.optionalLocalID(action.TaskID.String(), "workflow task")
		if convertErr != nil {
			return pluginsdk.WorkflowInstance{}, convertErr
		}
		nodeID, convertErr := optionalPluginLocalID(action.NodeID.String(), "workflow node")
		if convertErr != nil {
			return pluginsdk.WorkflowInstance{}, convertErr
		}
		timeline = append(timeline, pluginsdk.WorkflowAction{
			ID: actionID, Type: pluginsdk.WorkflowActionType(action.Type), InstanceID: id, TaskID: taskID, NodeID: nodeID,
			Actor: workflowActor(action.Actor), Target: workflowActor(action.Target), Comment: action.Comment, CreatedAt: action.CreatedAt,
		})
	}
	return pluginsdk.WorkflowInstance{
		ID: id, DefinitionID: definitionID, DefinitionKey: definitionKey, BusinessType: businessType,
		BusinessID: item.BusinessID, Title: item.Title, Status: pluginsdk.WorkflowInstanceStatus(item.Status),
		Starter: workflowActor(item.Starter), CurrentNode: currentNode, Tasks: tasks, Timeline: timeline,
		CreatedAt: item.Meta.CreatedAt, UpdatedAt: item.Meta.UpdatedAt,
	}, nil
}

func (s *workflowService) optionalLocalID(value, label string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	return s.localID(value, label)
}

func validatePluginLocalID(value, label string) (string, error) {
	value = strings.TrimSpace(value)
	if !pluginLocalIDPattern.MatchString(value) {
		return "", fmt.Errorf("%s id is invalid", label)
	}
	return value, nil
}

func validatePluginName(value, label string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if !auditNamePattern.MatchString(value) {
		return "", fmt.Errorf("%s is invalid", label)
	}
	return value, nil
}

func optionalPluginLocalID(value, label string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	return validatePluginLocalID(value, label)
}

func (s *workflowService) record(ctx context.Context, action, id string, detail map[string]any) error {
	_, err := s.audit.Record(ctx, pluginsdk.AuditEntry{
		Action: action, Resource: "workflow", ResourceID: strings.TrimSpace(id), Risk: pluginsdk.AuditRiskMedium, Detail: detail,
	})
	return err
}

func workflowActor(actor domainworkflow.Actor) pluginsdk.WorkflowActor {
	return pluginsdk.WorkflowActor{ID: actor.ID.String(), Name: actor.Name}
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

var _ pluginsdk.WorkflowService = (*workflowService)(nil)

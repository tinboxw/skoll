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
		decision := domainworkflow.DecisionRule{}
		if item.Decision != nil {
			decision = domainworkflow.DecisionRule{Strategy: domainworkflow.DecisionStrategy(item.Decision.Strategy), Quorum: item.Decision.Quorum}
		}
		var escalation *domainworkflow.EscalationRule
		if item.Escalation != nil {
			if strings.TrimSpace(item.Escalation.Target.ID) == "" {
				return pluginsdk.WorkflowDefinition{}, fmt.Errorf("workflow escalation target is required")
			}
			escalation = &domainworkflow.EscalationRule{
				After:  time.Duration(item.Escalation.AfterSeconds) * time.Second,
				Target: domainworkflow.Actor{ID: shared.ID(strings.TrimSpace(item.Escalation.Target.ID)), Name: strings.TrimSpace(item.Escalation.Target.Name)},
			}
		}
		var signature *domainworkflow.SignaturePolicy
		if item.Signature != nil {
			signature = &domainworkflow.SignaturePolicy{
				Meaning: strings.TrimSpace(item.Signature.Meaning), RequireEvidence: item.Signature.RequireEvidence,
			}
		}
		nodes = append(nodes, domainworkflow.Node{
			ID: shared.ID(nodeID), Key: nodeKey, Name: strings.TrimSpace(item.Name),
			Type: domainworkflow.NodeType(item.Type), Assignees: assignees,
			Decision: decision, Escalation: escalation, Signature: signature,
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
		condition, bindErr := workflowConditionToDomain(item.Condition)
		if bindErr != nil {
			return pluginsdk.WorkflowDefinition{}, bindErr
		}
		transitions = append(transitions, domainworkflow.Transition{From: shared.ID(from), To: shared.ID(to), Condition: condition})
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
		Variables: workflowVariablesToDomain(input.Variables),
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

func (s *workflowService) Delegate(ctx context.Context, input pluginsdk.WorkflowTargetActionInput) (pluginsdk.WorkflowInstance, error) {
	return s.targetAction(ctx, "delegate", input)
}

func (s *workflowService) Copy(ctx context.Context, input pluginsdk.WorkflowTargetActionInput) (pluginsdk.WorkflowInstance, error) {
	return s.targetAction(ctx, "copy", input)
}

func (s *workflowService) CreateSubstitution(ctx context.Context, input pluginsdk.WorkflowSubstitutionInput) (pluginsdk.WorkflowSubstitution, error) {
	id, err := s.boundID(input.ID, "workflow substitution")
	if err != nil {
		return pluginsdk.WorkflowSubstitution{}, err
	}
	actor := trustedHostActor(ctx, s.pluginID)
	if strings.TrimSpace(input.Substitute.ID) == "" {
		return pluginsdk.WorkflowSubstitution{}, fmt.Errorf("workflow substitute is required")
	}
	item, err := s.workflow.CreateSubstitution(ctx, workflowsvc.CreateSubstitutionInput{
		ID:         shared.ID(id),
		Principal:  domainworkflow.Actor{ID: shared.ID(actor.id), Name: actor.name},
		Substitute: domainworkflow.Actor{ID: shared.ID(strings.TrimSpace(input.Substitute.ID)), Name: strings.TrimSpace(input.Substitute.Name)},
		StartsAt:   input.StartsAt, EndsAt: input.EndsAt,
		CreatedBy: domainworkflow.Actor{ID: shared.ID(actor.id), Name: actor.name},
		Reason:    strings.TrimSpace(input.Reason), Now: s.now(),
	})
	if err != nil {
		return pluginsdk.WorkflowSubstitution{}, err
	}
	if err := s.record(ctx, "workflow.substitution.create", input.ID, map[string]any{"substituteId": input.Substitute.ID}); err != nil {
		return pluginsdk.WorkflowSubstitution{}, err
	}
	return s.substitution(*item)
}

func (s *workflowService) RevokeSubstitution(ctx context.Context, id string) (pluginsdk.WorkflowSubstitution, error) {
	bound, err := s.boundID(id, "workflow substitution")
	if err != nil {
		return pluginsdk.WorkflowSubstitution{}, err
	}
	actor := trustedHostActor(ctx, s.pluginID)
	item, err := s.workflow.RevokeSubstitution(ctx, workflowsvc.RevokeSubstitutionInput{
		ID: shared.ID(bound), Principal: domainworkflow.Actor{ID: shared.ID(actor.id), Name: actor.name}, Now: s.now(),
	})
	if err != nil {
		return pluginsdk.WorkflowSubstitution{}, err
	}
	if err := s.record(ctx, "workflow.substitution.revoke", id, nil); err != nil {
		return pluginsdk.WorkflowSubstitution{}, err
	}
	return s.substitution(*item)
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
	if input.Signature != nil {
		evidenceIDs := make([]shared.ID, 0, len(input.Signature.EvidenceIDs))
		for _, id := range input.Signature.EvidenceIDs {
			evidenceIDs = append(evidenceIDs, shared.ID(strings.TrimSpace(id)))
		}
		serviceInput.Signature = &workflowsvc.DecisionSignatureInput{
			Proof: input.Signature.Proof, Meaning: input.Signature.Meaning,
			Audience: "plugin:" + s.pluginID, EvidenceIDs: evidenceIDs,
		}
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
	result, err := s.instance(*item)
	if err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	detail := map[string]any{"taskId": input.TaskID}
	for index := len(result.Receipts) - 1; index >= 0; index-- {
		receipt := result.Receipts[index]
		if receipt.TaskID != input.TaskID || string(receipt.Action) != action {
			continue
		}
		detail["receiptId"] = receipt.ID
		detail["evidenceDigest"] = receipt.EvidenceDigest
		detail["auditCorrelationId"] = receipt.AuditCorrelationID
		break
	}
	if err := s.record(ctx, "workflow.task."+action, input.InstanceID, detail); err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	return result, nil
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
	if action == "delegate" {
		item, err = s.workflow.Delegate(ctx, serviceInput)
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
		var decision *pluginsdk.WorkflowDecisionRule
		var escalation *pluginsdk.WorkflowEscalationRule
		var signature *pluginsdk.WorkflowSignaturePolicy
		if node.Type == domainworkflow.NodeApproval {
			decision = &pluginsdk.WorkflowDecisionRule{Strategy: pluginsdk.WorkflowDecisionStrategy(node.Decision.Strategy), Quorum: node.Decision.Quorum}
			if node.Escalation != nil {
				escalation = &pluginsdk.WorkflowEscalationRule{
					AfterSeconds: int64(node.Escalation.After / time.Second), Target: workflowActor(node.Escalation.Target),
				}
			}
			if node.Signature != nil {
				signature = &pluginsdk.WorkflowSignaturePolicy{
					Meaning: node.Signature.Meaning, RequireEvidence: node.Signature.RequireEvidence,
				}
			}
		}
		nodes = append(nodes, pluginsdk.WorkflowNode{
			ID: nodeID, Key: nodeKey, Name: node.Name, Type: pluginsdk.WorkflowNodeType(node.Type), AssigneeIDs: assignees,
			Decision: decision, Escalation: escalation, Signature: signature,
		})
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
		transitions = append(transitions, pluginsdk.WorkflowTransition{From: from, To: to, Condition: workflowConditionFromDomain(transition.Condition)})
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
			OriginalAssignee: workflowActor(task.OriginalAssignee), Assignment: pluginsdk.WorkflowAssignmentKind(task.Assignment),
			AuthorizedBy: workflowActor(task.AuthorizedBy), AuthorizationID: task.AuthorizationID.String(),
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
		receiptID, convertErr := s.optionalLocalID(action.ReceiptID.String(), "workflow signature receipt")
		if convertErr != nil {
			return pluginsdk.WorkflowInstance{}, convertErr
		}
		timeline = append(timeline, pluginsdk.WorkflowAction{
			ID: actionID, Type: pluginsdk.WorkflowActionType(action.Type), InstanceID: id, TaskID: taskID, NodeID: nodeID,
			Actor: workflowActor(action.Actor), Target: workflowActor(action.Target), Comment: action.Comment,
			ReceiptID: receiptID, CreatedAt: action.CreatedAt,
		})
	}
	receipts := make([]pluginsdk.WorkflowSignatureReceipt, 0, len(item.Receipts))
	for _, receipt := range item.Receipts {
		receiptID, convertErr := s.localID(receipt.ID.String(), "workflow signature receipt")
		if convertErr != nil {
			return pluginsdk.WorkflowInstance{}, convertErr
		}
		receiptActionID, convertErr := s.localID(receipt.ActionID.String(), "workflow action")
		if convertErr != nil {
			return pluginsdk.WorkflowInstance{}, convertErr
		}
		receiptTaskID, convertErr := s.localID(receipt.TaskID.String(), "workflow task")
		if convertErr != nil {
			return pluginsdk.WorkflowInstance{}, convertErr
		}
		evidence := make([]pluginsdk.WorkflowEvidenceReference, 0, len(receipt.Evidence))
		for _, reference := range receipt.Evidence {
			evidence = append(evidence, pluginsdk.WorkflowEvidenceReference{
				FileID: reference.FileID.String(), Name: reference.Name, Hash: reference.Hash, Size: reference.Size, MIME: reference.MIME,
			})
		}
		receipts = append(receipts, pluginsdk.WorkflowSignatureReceipt{
			ID: receiptID, ActionID: receiptActionID, InstanceID: id,
			DefinitionID: definitionID, DefinitionKey: definitionKey, BusinessType: businessType, BusinessID: receipt.BusinessID,
			TaskID: receiptTaskID, NodeID: receipt.NodeID.String(),
			Action: pluginsdk.WorkflowActionType(receipt.Action), Actor: workflowActor(receipt.Actor), Meaning: receipt.Meaning,
			VerificationID: receipt.VerificationID.String(), VerificationMethod: receipt.VerificationMethod,
			VerificationAt: receipt.VerificationAt, Audience: receipt.Audience, Evidence: evidence,
			CommentDigest: receipt.CommentDigest, EvidenceDigest: receipt.EvidenceDigest,
			AuditCorrelationID: receiptID, SignedAt: receipt.SignedAt,
		})
	}
	return pluginsdk.WorkflowInstance{
		ID: id, DefinitionID: definitionID, DefinitionKey: definitionKey, BusinessType: businessType,
		BusinessID: item.BusinessID, Title: item.Title, Status: pluginsdk.WorkflowInstanceStatus(item.Status),
		Starter: workflowActor(item.Starter), CurrentNode: currentNode, ActiveNodes: workflowIDs(item.ActiveNodes),
		Variables: workflowVariablesFromDomain(item.Variables), Tasks: tasks, Timeline: timeline, Receipts: receipts,
		CreatedAt: item.Meta.CreatedAt, UpdatedAt: item.Meta.UpdatedAt,
	}, nil
}

func (s *workflowService) substitution(item domainworkflow.SubstitutionWindow) (pluginsdk.WorkflowSubstitution, error) {
	id, err := s.localID(item.ID.String(), "workflow substitution")
	if err != nil {
		return pluginsdk.WorkflowSubstitution{}, err
	}
	return pluginsdk.WorkflowSubstitution{
		ID: id, Principal: workflowActor(item.Principal), Substitute: workflowActor(item.Substitute),
		StartsAt: item.StartsAt, EndsAt: item.EndsAt, CreatedBy: workflowActor(item.CreatedBy),
		Reason: item.Reason, CreatedAt: item.CreatedAt, RevokedAt: cloneTime(item.RevokedAt),
	}, nil
}

func workflowConditionToDomain(condition *pluginsdk.WorkflowCondition) (*domainworkflow.Condition, error) {
	if condition == nil {
		return nil, nil
	}
	out := &domainworkflow.Condition{Match: domainworkflow.ConditionMatch(condition.Match), Predicates: make([]domainworkflow.Predicate, 0, len(condition.Predicates))}
	for _, predicate := range condition.Predicates {
		var value *domainworkflow.Value
		if predicate.Value != nil {
			value = &domainworkflow.Value{Type: domainworkflow.ValueType(predicate.Value.Type), Value: predicate.Value.Value}
		}
		out.Predicates = append(out.Predicates, domainworkflow.Predicate{
			Field: strings.TrimSpace(predicate.Field), Operator: domainworkflow.PredicateOperator(predicate.Operator), Value: value,
		})
	}
	if err := out.Validate(); err != nil {
		return nil, err
	}
	return out, nil
}

func workflowConditionFromDomain(condition *domainworkflow.Condition) *pluginsdk.WorkflowCondition {
	if condition == nil {
		return nil
	}
	out := &pluginsdk.WorkflowCondition{Match: pluginsdk.WorkflowConditionMatch(condition.Match), Predicates: make([]pluginsdk.WorkflowPredicate, 0, len(condition.Predicates))}
	for _, predicate := range condition.Predicates {
		var value *pluginsdk.WorkflowValue
		if predicate.Value != nil {
			value = &pluginsdk.WorkflowValue{Type: pluginsdk.WorkflowValueType(predicate.Value.Type), Value: predicate.Value.Value}
		}
		out.Predicates = append(out.Predicates, pluginsdk.WorkflowPredicate{
			Field: predicate.Field, Operator: pluginsdk.WorkflowPredicateOperator(predicate.Operator), Value: value,
		})
	}
	return out
}

func workflowVariablesToDomain(variables map[string]pluginsdk.WorkflowValue) map[string]domainworkflow.Value {
	out := make(map[string]domainworkflow.Value, len(variables))
	for key, value := range variables {
		out[key] = domainworkflow.Value{Type: domainworkflow.ValueType(value.Type), Value: value.Value}
	}
	return out
}

func workflowVariablesFromDomain(variables map[string]domainworkflow.Value) map[string]pluginsdk.WorkflowValue {
	out := make(map[string]pluginsdk.WorkflowValue, len(variables))
	for key, value := range variables {
		out[key] = pluginsdk.WorkflowValue{Type: pluginsdk.WorkflowValueType(value.Type), Value: value.Value}
	}
	return out
}

func workflowIDs(ids []shared.ID) []string {
	out := make([]string, len(ids))
	for index, id := range ids {
		out[index] = id.String()
	}
	return out
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

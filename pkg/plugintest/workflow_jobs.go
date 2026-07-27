package plugintest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const (
	OperationWorkflowCreateDefinition   = "workflows.create_definition"
	OperationWorkflowGetDefinition      = "workflows.get_definition"
	OperationWorkflowPublishDefinition  = "workflows.publish_definition"
	OperationWorkflowStart              = "workflows.start"
	OperationWorkflowGetInstance        = "workflows.get_instance"
	OperationWorkflowApprove            = "workflows.approve"
	OperationWorkflowReject             = "workflows.reject"
	OperationWorkflowWithdraw           = "workflows.withdraw"
	OperationWorkflowCancel             = "workflows.cancel"
	OperationWorkflowDelegate           = "workflows.delegate"
	OperationWorkflowCopy               = "workflows.copy"
	OperationWorkflowCreateSubstitution = "workflows.create_substitution"
	OperationWorkflowRevokeSubstitution = "workflows.revoke_substitution"
	OperationJobSchedule                = "jobs.schedule"
	OperationJobLeaseDue                = "jobs.lease_due"
	OperationJobComplete                = "jobs.complete"
	OperationJobFail                    = "jobs.fail"
	OperationJobGet                     = "jobs.get"
	OperationJobList                    = "jobs.list"
)

type WorkflowService struct {
	mu            sync.Mutex
	Clock         *Clock
	Identity      Identity
	Failures      *FailurePlan
	definitions   map[string]pluginsdk.WorkflowDefinition
	instances     map[string]pluginsdk.WorkflowInstance
	substitutions map[string]pluginsdk.WorkflowSubstitution
	actionCount   int
}

func newWorkflowService(clock *Clock, identity Identity, failures *FailurePlan) *WorkflowService {
	return &WorkflowService{
		Clock: clock, Identity: identity, Failures: failures,
		definitions:   make(map[string]pluginsdk.WorkflowDefinition),
		instances:     make(map[string]pluginsdk.WorkflowInstance),
		substitutions: make(map[string]pluginsdk.WorkflowSubstitution),
	}
}

func (s *WorkflowService) CreateDefinition(_ context.Context, input pluginsdk.WorkflowDefinitionInput) (pluginsdk.WorkflowDefinition, error) {
	if err := s.Failures.take(OperationWorkflowCreateDefinition); err != nil {
		return pluginsdk.WorkflowDefinition{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.Clock.Now()
	item := pluginsdk.WorkflowDefinition{
		ID: input.ID, Key: input.Key, Name: input.Name, Version: input.Version,
		Status: pluginsdk.WorkflowDefinitionDraft, Nodes: append([]pluginsdk.WorkflowNode(nil), input.Nodes...),
		Transitions: append([]pluginsdk.WorkflowTransition(nil), input.Transitions...), CreatedAt: now, UpdatedAt: now,
	}
	s.definitions[item.ID] = item
	return item, nil
}

func (s *WorkflowService) GetDefinition(_ context.Context, id string) (pluginsdk.WorkflowDefinition, error) {
	if err := s.Failures.take(OperationWorkflowGetDefinition); err != nil {
		return pluginsdk.WorkflowDefinition{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.definitions[id]
	if !ok {
		return pluginsdk.WorkflowDefinition{}, fmt.Errorf("fixture workflow definition %q not found", id)
	}
	return item, nil
}

func (s *WorkflowService) PublishDefinition(_ context.Context, id string) (pluginsdk.WorkflowDefinition, error) {
	if err := s.Failures.take(OperationWorkflowPublishDefinition); err != nil {
		return pluginsdk.WorkflowDefinition{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.definitions[id]
	if !ok {
		return pluginsdk.WorkflowDefinition{}, fmt.Errorf("fixture workflow definition %q not found", id)
	}
	item.Status = pluginsdk.WorkflowDefinitionPublished
	item.UpdatedAt = s.Clock.Now()
	s.definitions[id] = item
	return item, nil
}

func (s *WorkflowService) Start(_ context.Context, input pluginsdk.WorkflowStartInput) (pluginsdk.WorkflowInstance, error) {
	if err := s.Failures.take(OperationWorkflowStart); err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	definition, ok := s.definitions[input.DefinitionID]
	if !ok || definition.Status != pluginsdk.WorkflowDefinitionPublished {
		return pluginsdk.WorkflowInstance{}, errors.New("fixture workflow requires a published definition")
	}
	now := s.Clock.Now()
	startNode, currentNode := workflowEntryNodes(definition.Nodes)
	instance := pluginsdk.WorkflowInstance{
		ID: input.ID, DefinitionID: definition.ID, DefinitionKey: definition.Key,
		BusinessType: input.BusinessType, BusinessID: input.BusinessID, Title: input.Title,
		Status: pluginsdk.WorkflowInstanceRunning, Starter: s.actor(), CurrentNode: currentNode,
		ActiveNodes: []string{currentNode}, Variables: cloneWorkflowValues(input.Variables),
		CreatedAt: now, UpdatedAt: now,
	}
	instance.Timeline = append(instance.Timeline, s.action(pluginsdk.WorkflowActionStart, instance.ID, "", startNode, pluginsdk.WorkflowActor{}, ""))
	if node, ok := workflowNodeByID(definition.Nodes, currentNode); ok && node.Type == pluginsdk.WorkflowNodeApproval {
		assignee := pluginsdk.WorkflowActor{ID: "fixture-approver", Name: "Fixture Approver"}
		if len(node.AssigneeIDs) > 0 {
			assignee.ID = node.AssigneeIDs[0]
			assignee.Name = node.AssigneeIDs[0]
		}
		instance.Tasks = []pluginsdk.WorkflowTask{{
			ID: input.ID + "-task-1", InstanceID: input.ID, NodeID: node.ID,
			Assignee: assignee, OriginalAssignee: assignee, Assignment: pluginsdk.WorkflowAssignmentDirect,
			Status: pluginsdk.WorkflowTaskPending, CreatedAt: now,
		}}
	}
	s.instances[instance.ID] = instance
	return instance, nil
}

func (s *WorkflowService) GetInstance(_ context.Context, id string) (pluginsdk.WorkflowInstance, error) {
	if err := s.Failures.take(OperationWorkflowGetInstance); err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.instances[id]
	if !ok {
		return pluginsdk.WorkflowInstance{}, fmt.Errorf("fixture workflow instance %q not found", id)
	}
	return item, nil
}

func (s *WorkflowService) Approve(_ context.Context, input pluginsdk.WorkflowTaskActionInput) (pluginsdk.WorkflowInstance, error) {
	return s.completeTask(input, pluginsdk.WorkflowActionApprove, pluginsdk.WorkflowTaskApproved, pluginsdk.WorkflowInstanceApproved, OperationWorkflowApprove)
}

func (s *WorkflowService) Reject(_ context.Context, input pluginsdk.WorkflowTaskActionInput) (pluginsdk.WorkflowInstance, error) {
	return s.completeTask(input, pluginsdk.WorkflowActionReject, pluginsdk.WorkflowTaskRejected, pluginsdk.WorkflowInstanceRejected, OperationWorkflowReject)
}

func (s *WorkflowService) Withdraw(_ context.Context, input pluginsdk.WorkflowInstanceActionInput) (pluginsdk.WorkflowInstance, error) {
	return s.completeInstance(input, pluginsdk.WorkflowActionWithdraw, pluginsdk.WorkflowInstanceWithdrawn, OperationWorkflowWithdraw)
}

func (s *WorkflowService) Cancel(_ context.Context, input pluginsdk.WorkflowInstanceActionInput) (pluginsdk.WorkflowInstance, error) {
	return s.completeInstance(input, pluginsdk.WorkflowActionCancel, pluginsdk.WorkflowInstanceCanceled, OperationWorkflowCancel)
}

func (s *WorkflowService) Delegate(_ context.Context, input pluginsdk.WorkflowTargetActionInput) (pluginsdk.WorkflowInstance, error) {
	if err := s.Failures.take(OperationWorkflowDelegate); err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	instance, ok := s.instances[input.InstanceID]
	if !ok {
		return pluginsdk.WorkflowInstance{}, fmt.Errorf("fixture workflow instance %q not found", input.InstanceID)
	}
	for index := range instance.Tasks {
		if instance.Tasks[index].ID != input.TaskID {
			continue
		}
		instance.Tasks[index].Status = pluginsdk.WorkflowTaskDelegated
		instance.Tasks[index].Assignee = input.Target
		instance.Tasks[index].Assignment = pluginsdk.WorkflowAssignmentDelegated
		instance.Tasks[index].AuthorizedBy = s.actor()
	}
	instance.Timeline = append(instance.Timeline, s.action(pluginsdk.WorkflowActionDelegate, instance.ID, input.TaskID, instance.CurrentNode, input.Target, input.Comment))
	instance.UpdatedAt = s.Clock.Now()
	s.instances[instance.ID] = instance
	return instance, nil
}

func (s *WorkflowService) Copy(_ context.Context, input pluginsdk.WorkflowTargetActionInput) (pluginsdk.WorkflowInstance, error) {
	if err := s.Failures.take(OperationWorkflowCopy); err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	instance, ok := s.instances[input.InstanceID]
	if !ok {
		return pluginsdk.WorkflowInstance{}, fmt.Errorf("fixture workflow instance %q not found", input.InstanceID)
	}
	now := s.Clock.Now()
	instance.Tasks = append(instance.Tasks, pluginsdk.WorkflowTask{
		ID: input.InstanceID + "-copy-" + strconv.Itoa(len(instance.Tasks)+1), InstanceID: instance.ID,
		NodeID: instance.CurrentNode, Assignee: input.Target, OriginalAssignee: input.Target,
		Assignment: pluginsdk.WorkflowAssignmentDirect, Status: pluginsdk.WorkflowTaskCopied, CreatedAt: now,
	})
	instance.Timeline = append(instance.Timeline, s.action(pluginsdk.WorkflowActionCopy, instance.ID, input.TaskID, instance.CurrentNode, input.Target, input.Comment))
	instance.UpdatedAt = now
	s.instances[instance.ID] = instance
	return instance, nil
}

func (s *WorkflowService) CreateSubstitution(_ context.Context, input pluginsdk.WorkflowSubstitutionInput) (pluginsdk.WorkflowSubstitution, error) {
	if err := s.Failures.take(OperationWorkflowCreateSubstitution); err != nil {
		return pluginsdk.WorkflowSubstitution{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item := pluginsdk.WorkflowSubstitution{
		ID: input.ID, Principal: s.actor(), Substitute: input.Substitute, StartsAt: input.StartsAt,
		EndsAt: input.EndsAt, CreatedBy: s.actor(), Reason: input.Reason, CreatedAt: s.Clock.Now(),
	}
	s.substitutions[item.ID] = item
	return item, nil
}

func (s *WorkflowService) RevokeSubstitution(_ context.Context, id string) (pluginsdk.WorkflowSubstitution, error) {
	if err := s.Failures.take(OperationWorkflowRevokeSubstitution); err != nil {
		return pluginsdk.WorkflowSubstitution{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.substitutions[id]
	if !ok {
		return pluginsdk.WorkflowSubstitution{}, fmt.Errorf("fixture workflow substitution %q not found", id)
	}
	now := s.Clock.Now()
	item.RevokedAt = &now
	s.substitutions[id] = item
	return item, nil
}

func (s *WorkflowService) completeTask(input pluginsdk.WorkflowTaskActionInput, action pluginsdk.WorkflowActionType, taskStatus pluginsdk.WorkflowTaskStatus, instanceStatus pluginsdk.WorkflowInstanceStatus, operation string) (pluginsdk.WorkflowInstance, error) {
	if err := s.Failures.take(operation); err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	instance, ok := s.instances[input.InstanceID]
	if !ok {
		return pluginsdk.WorkflowInstance{}, fmt.Errorf("fixture workflow instance %q not found", input.InstanceID)
	}
	now := s.Clock.Now()
	for index := range instance.Tasks {
		if instance.Tasks[index].ID == input.TaskID {
			instance.Tasks[index].Status = taskStatus
			instance.Tasks[index].CompletedAt = &now
		}
	}
	instance.Status = instanceStatus
	instance.CurrentNode = "end"
	instance.ActiveNodes = []string{"end"}
	instance.Timeline = append(instance.Timeline, s.action(action, instance.ID, input.TaskID, instance.CurrentNode, pluginsdk.WorkflowActor{}, input.Comment))
	instance.UpdatedAt = now
	s.instances[instance.ID] = instance
	return instance, nil
}

func (s *WorkflowService) completeInstance(input pluginsdk.WorkflowInstanceActionInput, action pluginsdk.WorkflowActionType, status pluginsdk.WorkflowInstanceStatus, operation string) (pluginsdk.WorkflowInstance, error) {
	if err := s.Failures.take(operation); err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	instance, ok := s.instances[input.InstanceID]
	if !ok {
		return pluginsdk.WorkflowInstance{}, fmt.Errorf("fixture workflow instance %q not found", input.InstanceID)
	}
	now := s.Clock.Now()
	instance.Status = status
	instance.UpdatedAt = now
	instance.Timeline = append(instance.Timeline, s.action(action, instance.ID, "", instance.CurrentNode, pluginsdk.WorkflowActor{}, input.Comment))
	s.instances[instance.ID] = instance
	return instance, nil
}

func (s *WorkflowService) actor() pluginsdk.WorkflowActor {
	return pluginsdk.WorkflowActor{ID: s.Identity.Subject, Name: s.Identity.Subject}
}

func (s *WorkflowService) action(action pluginsdk.WorkflowActionType, instanceID, taskID, nodeID string, target pluginsdk.WorkflowActor, comment string) pluginsdk.WorkflowAction {
	s.actionCount++
	return pluginsdk.WorkflowAction{
		ID: "fixture-action-" + strconv.Itoa(s.actionCount), Type: action, InstanceID: instanceID,
		TaskID: taskID, NodeID: nodeID, Actor: s.actor(), Target: target, Comment: comment, CreatedAt: s.Clock.Now(),
	}
}

func workflowEntryNodes(nodes []pluginsdk.WorkflowNode) (string, string) {
	start, current := "start", "end"
	for _, node := range nodes {
		if node.Type == pluginsdk.WorkflowNodeStart {
			start = node.ID
		}
		if current == "end" && node.Type == pluginsdk.WorkflowNodeApproval {
			current = node.ID
		}
	}
	return start, current
}

func workflowNodeByID(nodes []pluginsdk.WorkflowNode, id string) (pluginsdk.WorkflowNode, bool) {
	for _, node := range nodes {
		if node.ID == id {
			return node, true
		}
	}
	return pluginsdk.WorkflowNode{}, false
}

func cloneWorkflowValues(values map[string]pluginsdk.WorkflowValue) map[string]pluginsdk.WorkflowValue {
	result := make(map[string]pluginsdk.WorkflowValue, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

type JobService struct {
	mu       sync.Mutex
	Clock    *Clock
	Failures *FailurePlan
	jobs     map[string]pluginsdk.Job
}

func newJobService(clock *Clock, failures *FailurePlan) *JobService {
	return &JobService{Clock: clock, Failures: failures, jobs: make(map[string]pluginsdk.Job)}
}

func (s *JobService) Schedule(_ context.Context, input pluginsdk.JobScheduleInput) (pluginsdk.Job, error) {
	if err := s.Failures.take(OperationJobSchedule); err != nil {
		return pluginsdk.Job{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.findByIdempotency(input.IdempotencyKey); ok {
		return existing, nil
	}
	now := s.Clock.Now()
	runAt := input.RunAt.UTC()
	if runAt.IsZero() {
		runAt = now
	}
	maxAttempts := input.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	item := pluginsdk.Job{
		ID: input.ID, Kind: input.Kind, IdempotencyKey: input.IdempotencyKey,
		Payload: cloneJSON(input.Payload), Status: pluginsdk.JobStatusScheduled, RunAt: runAt,
		MaxAttempts: maxAttempts, CreatedAt: now, UpdatedAt: now,
	}
	s.jobs[item.ID] = item
	return item, nil
}

func (s *JobService) LeaseDue(_ context.Context, input pluginsdk.JobLeaseInput) ([]pluginsdk.Job, error) {
	if err := s.Failures.take(OperationJobLeaseDue); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.Clock.Now()
	ids := sortedJobIDs(s.jobs)
	items := make([]pluginsdk.Job, 0)
	for _, id := range ids {
		item := s.jobs[id]
		if item.Status != pluginsdk.JobStatusScheduled && item.Status != pluginsdk.JobStatusRetryWait {
			continue
		}
		if item.RunAt.After(now) {
			continue
		}
		if input.Limit > 0 && len(items) >= input.Limit {
			break
		}
		expires := now.Add(input.LeaseDuration)
		item.Status = pluginsdk.JobStatusRunning
		item.AttemptCount++
		item.LeaseOwner = input.WorkerID
		item.LeaseToken = "fixture-lease-" + item.ID + "-" + strconv.Itoa(item.AttemptCount)
		item.LeaseExpiresAt = &expires
		item.UpdatedAt = now
		s.jobs[id] = item
		items = append(items, item)
	}
	return items, nil
}

func (s *JobService) Complete(_ context.Context, input pluginsdk.JobCompleteInput) (pluginsdk.Job, error) {
	if err := s.Failures.take(OperationJobComplete); err != nil {
		return pluginsdk.Job{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, err := s.leased(input.JobID, input.LeaseToken)
	if err != nil {
		return pluginsdk.Job{}, err
	}
	now := s.Clock.Now()
	item.Status = pluginsdk.JobStatusSucceeded
	item.Result = cloneJSON(input.Result)
	item.UpdatedAt = now
	item.CompletedAt = &now
	item.LeaseExpiresAt = nil
	s.jobs[item.ID] = item
	return item, nil
}

func (s *JobService) Fail(_ context.Context, input pluginsdk.JobFailInput) (pluginsdk.Job, error) {
	if err := s.Failures.take(OperationJobFail); err != nil {
		return pluginsdk.Job{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, err := s.leased(input.JobID, input.LeaseToken)
	if err != nil {
		return pluginsdk.Job{}, err
	}
	now := s.Clock.Now()
	item.LastError = input.Error
	item.UpdatedAt = now
	item.LeaseExpiresAt = nil
	if item.AttemptCount >= item.MaxAttempts {
		item.Status = pluginsdk.JobStatusDeadLetter
		item.DeadLetteredAt = &now
	} else {
		item.Status = pluginsdk.JobStatusRetryWait
		item.RunAt = now.Add(input.RetryAfter)
	}
	s.jobs[item.ID] = item
	return item, nil
}

func (s *JobService) Get(_ context.Context, id string) (pluginsdk.Job, error) {
	if err := s.Failures.take(OperationJobGet); err != nil {
		return pluginsdk.Job{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.jobs[id]
	if !ok {
		return pluginsdk.Job{}, fmt.Errorf("fixture job %q not found", id)
	}
	return item, nil
}

func (s *JobService) List(_ context.Context, query pluginsdk.JobQuery) ([]pluginsdk.Job, error) {
	if err := s.Failures.take(OperationJobList); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]pluginsdk.Job, 0)
	for _, id := range sortedJobIDs(s.jobs) {
		item := s.jobs[id]
		if query.Kind != "" && item.Kind != query.Kind {
			continue
		}
		if query.Status != "" && item.Status != query.Status {
			continue
		}
		if query.Limit > 0 && len(items) >= query.Limit {
			break
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *JobService) leased(id, token string) (pluginsdk.Job, error) {
	item, ok := s.jobs[id]
	if !ok {
		return pluginsdk.Job{}, fmt.Errorf("fixture job %q not found", id)
	}
	if item.Status != pluginsdk.JobStatusRunning || item.LeaseToken != token {
		return pluginsdk.Job{}, errors.New("fixture job lease is invalid")
	}
	return item, nil
}

func (s *JobService) findByIdempotency(key string) (pluginsdk.Job, bool) {
	if strings.TrimSpace(key) == "" {
		return pluginsdk.Job{}, false
	}
	for _, item := range s.jobs {
		if item.IdempotencyKey == key {
			return item, true
		}
	}
	return pluginsdk.Job{}, false
}

func sortedJobIDs(items map[string]pluginsdk.Job) []string {
	ids := make([]string, 0, len(items))
	for id := range items {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func cloneJSON(value json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), value...)
}

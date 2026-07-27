package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
	jobsvc "github.com/tinboxw/skoll/internal/service/job"
)

const (
	workflowTimerNamespace = "system.workflow"
	workflowTimerKind      = "approval-escalation"
)

type escalationTimerPayload struct {
	InstanceID string `json:"instanceId"`
	TaskID     string `json:"taskId"`
	TargetID   string `json:"targetId"`
	TargetName string `json:"targetName,omitempty"`
}

func (s *serviceImpl) scheduleTimers(ctx context.Context, definition domainworkflow.Definition, instance domainworkflow.Instance) error {
	if instance.Status != domainworkflow.InstanceRunning {
		return nil
	}
	nodes := make(map[shared.ID]domainworkflow.Node, len(definition.Nodes))
	for _, node := range definition.Nodes {
		nodes[node.ID] = node
	}
	for _, task := range instance.Tasks {
		if task.Status != domainworkflow.TaskPending || task.Assignment == domainworkflow.AssignmentEscalated {
			continue
		}
		node, exists := nodes[task.NodeID]
		if !exists || node.Escalation == nil {
			continue
		}
		payload, err := json.Marshal(escalationTimerPayload{
			InstanceID: instance.ID.String(),
			TaskID:     task.ID.String(),
			TargetID:   node.Escalation.Target.ID.String(),
			TargetName: node.Escalation.Target.Name,
		})
		if err != nil {
			return fmt.Errorf("encode workflow escalation timer: %w", err)
		}
		identity := timerIdentity(instance.ID, task.ID)
		if _, err := s.opts.Jobs.Schedule(ctx, jobsvc.ScheduleInput{
			ID: identity, Namespace: workflowTimerNamespace, Kind: workflowTimerKind,
			IdempotencyKey: identity, Payload: payload, RunAt: task.CreatedAt.Add(node.Escalation.After), MaxAttempts: 8,
		}); err != nil {
			return fmt.Errorf("schedule workflow escalation timer: %w", err)
		}
	}
	return nil
}

func (s *serviceImpl) ProcessDueTimers(ctx context.Context, workerID string, limit int, leaseDuration time.Duration) (int, error) {
	workerID = strings.TrimSpace(workerID)
	if workerID == "" {
		return 0, fmt.Errorf("workflow timer worker id is required")
	}
	jobs, err := s.opts.Jobs.LeaseDue(ctx, jobsvc.LeaseInput{
		Namespace: workflowTimerNamespace, WorkerID: workerID, Limit: limit, LeaseDuration: leaseDuration,
	})
	if err != nil {
		return 0, err
	}
	completed := 0
	for _, timer := range jobs {
		if err := s.executeTimer(ctx, timer); err != nil {
			if _, failErr := s.opts.Jobs.Fail(ctx, jobsvc.FailInput{
				JobID: timer.ID, LeaseToken: timer.LeaseToken, Error: err.Error(), RetryAfter: 5 * time.Second,
			}); failErr != nil {
				return completed, fmt.Errorf("execute workflow timer: %v; persist retry: %w", err, failErr)
			}
			continue
		}
		completed++
	}
	return completed, nil
}

func (s *serviceImpl) executeTimer(ctx context.Context, timer jobsvc.Job) error {
	if timer.Namespace != workflowTimerNamespace || timer.Kind != workflowTimerKind {
		return fmt.Errorf("workflow timer job contract is invalid")
	}
	var payload escalationTimerPayload
	if err := json.Unmarshal(timer.Payload, &payload); err != nil {
		return fmt.Errorf("decode workflow escalation timer: %w", err)
	}
	if strings.TrimSpace(payload.InstanceID) == "" || strings.TrimSpace(payload.TaskID) == "" || strings.TrimSpace(payload.TargetID) == "" {
		return fmt.Errorf("workflow escalation timer payload is incomplete")
	}
	return s.inTransaction(ctx, func(txCtx context.Context) error {
		_, err := s.updateInstance(
			txCtx,
			shared.ID(payload.InstanceID),
			instanceActionIdentity{
				actionType: domainworkflow.ActionEscalate,
				taskID:     shared.ID(payload.TaskID),
				actorID:    shared.ID("system:workflow-timer"),
				targetID:   shared.ID(payload.TargetID),
			},
			func(instance *domainworkflow.Instance) error {
				return instance.Escalate(
					shared.ID(payload.TaskID),
					domainworkflow.Actor{ID: shared.ID(payload.TargetID), Name: payload.TargetName},
					s.opts.Now().UTC(),
				)
			},
		)
		if err != nil {
			return err
		}
		result, err := json.Marshal(map[string]string{"outcome": "completed"})
		if err != nil {
			return err
		}
		_, err = s.opts.Jobs.Complete(txCtx, jobsvc.CompleteInput{
			JobID: timer.ID, LeaseToken: timer.LeaseToken, Result: result,
		})
		return err
	})
}

func (s *serviceImpl) RunTimerWorker(ctx context.Context, workerID string, pollInterval time.Duration, batchSize int) error {
	if pollInterval <= 0 {
		return fmt.Errorf("workflow timer poll interval must be positive")
	}
	if batchSize <= 0 || batchSize > 100 {
		return fmt.Errorf("workflow timer batch size must be between 1 and 100")
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		if _, err := s.ProcessDueTimers(ctx, workerID, batchSize, 2*pollInterval); err != nil && ctx.Err() == nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func timerIdentity(instanceID, taskID shared.ID) string {
	digest := sha256.Sum256([]byte(instanceID.String() + "\x00" + taskID.String()))
	return "workflow:escalation:" + hex.EncodeToString(digest[:16])
}

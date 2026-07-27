package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
	"github.com/tinboxw/skoll/internal/repository"
)

type serviceImpl struct {
	repo Repository
	opts Options
}

func NewService(repo Repository, opts Options) Service {
	if repo == nil || opts.Jobs == nil || opts.UnitOfWork == nil || opts.Now == nil {
		panic("workflow repository, jobs, unit of work, and clock are required")
	}
	return &serviceImpl{repo: repo, opts: opts}
}

func (s *serviceImpl) CreateDefinition(ctx context.Context, in CreateDefinitionInput) (*domainworkflow.Definition, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("workflow service repository is required")
	}
	definition, err := domainworkflow.NewDefinition(in.ID, in.Key, in.Name, in.Version, in.Nodes, in.Transitions, in.Now)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SaveDefinition(ctx, *definition); err != nil {
		return nil, err
	}
	return definition, nil
}

func (s *serviceImpl) GetDefinition(ctx context.Context, id shared.ID) (*domainworkflow.Definition, error) {
	return s.definition(ctx, id)
}

func (s *serviceImpl) PublishDefinition(ctx context.Context, id shared.ID, now time.Time) (*domainworkflow.Definition, error) {
	definition, err := s.definition(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := definition.Publish(now); err != nil {
		return nil, err
	}
	if err := s.repo.SaveDefinition(ctx, *definition); err != nil {
		return nil, err
	}
	return definition, nil
}

func (s *serviceImpl) Start(ctx context.Context, in StartInput) (*domainworkflow.Instance, error) {
	definition, err := s.definition(ctx, in.DefinitionID)
	if err != nil {
		return nil, err
	}
	var result *domainworkflow.Instance
	operation := func(txCtx context.Context) error {
		instance, err := domainworkflow.Start(domainworkflow.StartInput{
			ID:           in.ID,
			Definition:   *definition,
			BusinessType: in.BusinessType,
			BusinessID:   in.BusinessID,
			Title:        in.Title,
			Starter:      in.Starter,
			Variables:    in.Variables,
			Now:          in.Now,
		})
		if err != nil {
			return err
		}
		if err := s.applySubstitutions(txCtx, instance, *definition, in.Now); err != nil {
			return err
		}
		if err := s.repo.SaveInstance(txCtx, *instance); err != nil {
			return err
		}
		if err := s.scheduleTimers(txCtx, *definition, *instance); err != nil {
			return err
		}
		result = instance
		return nil
	}
	if definitionHasEscalation(*definition) {
		err = s.inTransaction(ctx, operation)
	} else {
		err = operation(ctx)
	}
	return result, err
}

func (s *serviceImpl) GetInstance(ctx context.Context, id shared.ID) (*domainworkflow.Instance, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("workflow service repository is required")
	}
	if id.IsZero() {
		return nil, fmt.Errorf("workflow instance id is required")
	}
	return s.repo.GetInstance(ctx, id)
}

func (s *serviceImpl) Approve(ctx context.Context, in TaskActionInput) (*domainworkflow.Instance, error) {
	return s.updateInstanceWithTimers(ctx, in.InstanceID, instanceActionIdentity{actionType: domainworkflow.ActionApprove, taskID: in.TaskID, actorID: in.Actor.ID}, in.Now, func(instance *domainworkflow.Instance, definition domainworkflow.Definition) error {
		return instance.Approve(definition, in.TaskID, in.Actor, in.Comment, in.Now)
	})
}

func (s *serviceImpl) Reject(ctx context.Context, in TaskActionInput) (*domainworkflow.Instance, error) {
	return s.updateInstanceWithTimers(ctx, in.InstanceID, instanceActionIdentity{actionType: domainworkflow.ActionReject, taskID: in.TaskID, actorID: in.Actor.ID}, in.Now, func(instance *domainworkflow.Instance, _ domainworkflow.Definition) error {
		return instance.Reject(in.TaskID, in.Actor, in.Comment, in.Now)
	})
}

func (s *serviceImpl) Withdraw(ctx context.Context, in InstanceActionInput) (*domainworkflow.Instance, error) {
	return s.updateInstanceWithTimers(ctx, in.InstanceID, instanceActionIdentity{actionType: domainworkflow.ActionWithdraw, actorID: in.Actor.ID}, in.Now, func(instance *domainworkflow.Instance, _ domainworkflow.Definition) error {
		return instance.Withdraw(in.Actor, in.Comment, in.Now)
	})
}

func (s *serviceImpl) Cancel(ctx context.Context, in InstanceActionInput) (*domainworkflow.Instance, error) {
	return s.updateInstanceWithTimers(ctx, in.InstanceID, instanceActionIdentity{actionType: domainworkflow.ActionCancel, actorID: in.Actor.ID}, in.Now, func(instance *domainworkflow.Instance, _ domainworkflow.Definition) error {
		return instance.Cancel(in.Actor, in.Comment, in.Now)
	})
}

func (s *serviceImpl) Delegate(ctx context.Context, in TaskTargetActionInput) (*domainworkflow.Instance, error) {
	return s.updateInstanceWithTimers(ctx, in.InstanceID, instanceActionIdentity{actionType: domainworkflow.ActionDelegate, taskID: in.TaskID, actorID: in.Actor.ID, targetID: in.Target.ID}, in.Now, func(instance *domainworkflow.Instance, _ domainworkflow.Definition) error {
		return instance.Delegate(in.TaskID, in.Actor, in.Target, in.Comment, in.Now)
	})
}

func (s *serviceImpl) Copy(ctx context.Context, in TaskTargetActionInput) (*domainworkflow.Instance, error) {
	return s.updateInstanceWithTimers(ctx, in.InstanceID, instanceActionIdentity{actionType: domainworkflow.ActionCopy, taskID: in.TaskID, actorID: in.Actor.ID, targetID: in.Target.ID}, in.Now, func(instance *domainworkflow.Instance, _ domainworkflow.Definition) error {
		return instance.Copy(in.TaskID, in.Actor, in.Target, in.Comment, in.Now)
	})
}

func (s *serviceImpl) CreateSubstitution(ctx context.Context, in CreateSubstitutionInput) (*domainworkflow.SubstitutionWindow, error) {
	window, err := domainworkflow.NewSubstitutionWindow(
		in.ID, in.Principal, in.Substitute, in.CreatedBy, in.StartsAt, in.EndsAt, in.Now, in.Reason,
	)
	if err != nil {
		return nil, err
	}
	stored, created, err := s.repo.CreateSubstitution(ctx, *window)
	if err != nil {
		return nil, err
	}
	if !created && !sameSubstitution(*stored, *window) {
		return nil, fmt.Errorf("workflow substitution identity conflicts with persisted window")
	}
	return stored, nil
}

func (s *serviceImpl) RevokeSubstitution(ctx context.Context, in RevokeSubstitutionInput) (*domainworkflow.SubstitutionWindow, error) {
	if in.ID.IsZero() || in.Principal.ID.IsZero() {
		return nil, fmt.Errorf("workflow substitution id and principal are required")
	}
	return s.repo.RevokeSubstitution(ctx, in.ID, in.Principal.ID, in.Now)
}

func (s *serviceImpl) definition(ctx context.Context, id shared.ID) (*domainworkflow.Definition, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("workflow service repository is required")
	}
	if id.IsZero() {
		return nil, fmt.Errorf("workflow definition id is required")
	}
	return s.repo.GetDefinition(ctx, id)
}

func (s *serviceImpl) definitionForInstance(ctx context.Context, id shared.ID) (*domainworkflow.Definition, error) {
	instance, err := s.GetInstance(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.definition(ctx, instance.DefinitionID)
}

func (s *serviceImpl) updateInstance(ctx context.Context, id shared.ID, identity instanceActionIdentity, mutate func(*domainworkflow.Instance) error) (*domainworkflow.Instance, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("workflow service repository is required")
	}
	if id.IsZero() {
		return nil, fmt.Errorf("workflow instance id is required")
	}
	return s.repo.UpdateInstance(ctx, id, func(instance *domainworkflow.Instance) (bool, error) {
		if identity.applied(instance) {
			return false, nil
		}
		if err := mutate(instance); err != nil {
			return false, err
		}
		return true, nil
	})
}

func (s *serviceImpl) updateInstanceWithTimers(
	ctx context.Context,
	id shared.ID,
	identity instanceActionIdentity,
	now time.Time,
	mutate func(*domainworkflow.Instance, domainworkflow.Definition) error,
) (*domainworkflow.Instance, error) {
	definition, err := s.definitionForInstance(ctx, id)
	if err != nil {
		return nil, err
	}
	windows, err := s.activeSubstitutions(ctx, *definition, now)
	if err != nil {
		return nil, err
	}
	var result *domainworkflow.Instance
	operation := func(txCtx context.Context) error {
		result, err = s.updateInstance(txCtx, id, identity, func(instance *domainworkflow.Instance) error {
			if err := mutate(instance, *definition); err != nil {
				return err
			}
			if instance.Status != domainworkflow.InstanceRunning {
				return nil
			}
			return instance.ApplySubstitutions(windows, now)
		})
		if err != nil {
			return err
		}
		return s.scheduleTimers(txCtx, *definition, *result)
	}
	if definitionHasEscalation(*definition) {
		err = s.inTransaction(ctx, operation)
	} else {
		err = operation(ctx)
	}
	return result, err
}

func (s *serviceImpl) applySubstitutions(ctx context.Context, instance *domainworkflow.Instance, definition domainworkflow.Definition, now time.Time) error {
	if instance.Status != domainworkflow.InstanceRunning {
		return nil
	}
	windows, err := s.activeSubstitutions(ctx, definition, now)
	if err != nil {
		return err
	}
	return instance.ApplySubstitutions(windows, now)
}

func (s *serviceImpl) activeSubstitutions(ctx context.Context, definition domainworkflow.Definition, now time.Time) ([]domainworkflow.SubstitutionWindow, error) {
	principals := make([]shared.ID, 0)
	for _, node := range definition.Nodes {
		principals = append(principals, node.Assignees...)
	}
	return s.repo.ListActiveSubstitutions(ctx, principals, s.resolveNow(now))
}

func (s *serviceImpl) inTransaction(ctx context.Context, operation func(context.Context) error) error {
	if operation == nil {
		return fmt.Errorf("workflow transaction operation is required")
	}
	return s.opts.UnitOfWork.Do(ctx, func(tx repository.Tx) error {
		return operation(tx.Context())
	})
}

func sameSubstitution(left, right domainworkflow.SubstitutionWindow) bool {
	return left.ID == right.ID && left.Principal.ID == right.Principal.ID && left.Substitute.ID == right.Substitute.ID &&
		left.StartsAt.Equal(right.StartsAt) && left.EndsAt.Equal(right.EndsAt) && left.CreatedBy.ID == right.CreatedBy.ID &&
		left.Reason == right.Reason
}

func (s *serviceImpl) resolveNow(now time.Time) time.Time {
	if now.IsZero() {
		return s.opts.Now().UTC()
	}
	return now.UTC()
}

func definitionHasEscalation(definition domainworkflow.Definition) bool {
	for _, node := range definition.Nodes {
		if node.Escalation != nil {
			return true
		}
	}
	return false
}

type instanceActionIdentity struct {
	actionType domainworkflow.ActionType
	taskID     shared.ID
	actorID    shared.ID
	targetID   shared.ID
}

func (identity instanceActionIdentity) applied(instance *domainworkflow.Instance) bool {
	if instance == nil {
		return false
	}
	for _, action := range instance.Timeline {
		if action.Type == identity.actionType && action.TaskID == identity.taskID && action.Actor.ID == identity.actorID && action.Target.ID == identity.targetID {
			return true
		}
	}
	return false
}

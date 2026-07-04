package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
)

type serviceImpl struct {
	repo Repository
}

func NewService(repo Repository) Service {
	if repo == nil {
		repo = NewMemoryRepository()
	}
	return &serviceImpl{repo: repo}
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
	instance, err := domainworkflow.Start(domainworkflow.StartInput{
		ID:           in.ID,
		Definition:   *definition,
		BusinessType: in.BusinessType,
		BusinessID:   in.BusinessID,
		Title:        in.Title,
		Starter:      in.Starter,
		Now:          in.Now,
	})
	if err != nil {
		return nil, err
	}
	if err := s.repo.SaveInstance(ctx, *instance); err != nil {
		return nil, err
	}
	return instance, nil
}

func (s *serviceImpl) Approve(ctx context.Context, in TaskActionInput) (*domainworkflow.Instance, error) {
	return s.updateInstance(ctx, in.InstanceID, func(instance *domainworkflow.Instance) error {
		return instance.Approve(in.TaskID, in.Actor, in.Comment, in.Now)
	})
}

func (s *serviceImpl) Reject(ctx context.Context, in TaskActionInput) (*domainworkflow.Instance, error) {
	return s.updateInstance(ctx, in.InstanceID, func(instance *domainworkflow.Instance) error {
		return instance.Reject(in.TaskID, in.Actor, in.Comment, in.Now)
	})
}

func (s *serviceImpl) Withdraw(ctx context.Context, in InstanceActionInput) (*domainworkflow.Instance, error) {
	return s.updateInstance(ctx, in.InstanceID, func(instance *domainworkflow.Instance) error {
		return instance.Withdraw(in.Actor, in.Comment, in.Now)
	})
}

func (s *serviceImpl) Transfer(ctx context.Context, in TaskTargetActionInput) (*domainworkflow.Instance, error) {
	return s.updateInstance(ctx, in.InstanceID, func(instance *domainworkflow.Instance) error {
		return instance.Transfer(in.TaskID, in.Actor, in.Target, in.Comment, in.Now)
	})
}

func (s *serviceImpl) Copy(ctx context.Context, in TaskTargetActionInput) (*domainworkflow.Instance, error) {
	return s.updateInstance(ctx, in.InstanceID, func(instance *domainworkflow.Instance) error {
		return instance.Copy(in.TaskID, in.Actor, in.Target, in.Comment, in.Now)
	})
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

func (s *serviceImpl) updateInstance(ctx context.Context, id shared.ID, mutate func(*domainworkflow.Instance) error) (*domainworkflow.Instance, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("workflow service repository is required")
	}
	if id.IsZero() {
		return nil, fmt.Errorf("workflow instance id is required")
	}
	instance, err := s.repo.GetInstance(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := mutate(instance); err != nil {
		return nil, err
	}
	if err := s.repo.SaveInstance(ctx, *instance); err != nil {
		return nil, err
	}
	return instance, nil
}

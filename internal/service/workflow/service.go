package workflow

import (
	"context"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
	"github.com/tinboxw/skoll/internal/repository"
	jobsvc "github.com/tinboxw/skoll/internal/service/job"
)

type DefinitionRepository interface {
	SaveDefinition(ctx context.Context, definition domainworkflow.Definition) error
	GetDefinition(ctx context.Context, id shared.ID) (*domainworkflow.Definition, error)
}

type InstanceRepository interface {
	SaveInstance(ctx context.Context, instance domainworkflow.Instance) error
	GetInstance(ctx context.Context, id shared.ID) (*domainworkflow.Instance, error)
	UpdateInstance(ctx context.Context, id shared.ID, mutate InstanceMutation) (*domainworkflow.Instance, error)
}

type SubstitutionRepository interface {
	CreateSubstitution(ctx context.Context, window domainworkflow.SubstitutionWindow) (*domainworkflow.SubstitutionWindow, bool, error)
	RevokeSubstitution(ctx context.Context, id, principalID shared.ID, now time.Time) (*domainworkflow.SubstitutionWindow, error)
	ListActiveSubstitutions(ctx context.Context, principalIDs []shared.ID, at time.Time) ([]domainworkflow.SubstitutionWindow, error)
}

// InstanceMutation runs inside the repository's atomic update boundary.
// Implementations may retry it, so the callback must not perform external side effects.
type InstanceMutation func(instance *domainworkflow.Instance) (changed bool, err error)

type Repository interface {
	DefinitionRepository
	InstanceRepository
	SubstitutionRepository
}

type Service interface {
	CreateDefinition(ctx context.Context, in CreateDefinitionInput) (*domainworkflow.Definition, error)
	GetDefinition(ctx context.Context, id shared.ID) (*domainworkflow.Definition, error)
	PublishDefinition(ctx context.Context, id shared.ID, now time.Time) (*domainworkflow.Definition, error)
	Start(ctx context.Context, in StartInput) (*domainworkflow.Instance, error)
	GetInstance(ctx context.Context, id shared.ID) (*domainworkflow.Instance, error)
	Approve(ctx context.Context, in TaskActionInput) (*domainworkflow.Instance, error)
	Reject(ctx context.Context, in TaskActionInput) (*domainworkflow.Instance, error)
	Withdraw(ctx context.Context, in InstanceActionInput) (*domainworkflow.Instance, error)
	Cancel(ctx context.Context, in InstanceActionInput) (*domainworkflow.Instance, error)
	Delegate(ctx context.Context, in TaskTargetActionInput) (*domainworkflow.Instance, error)
	Copy(ctx context.Context, in TaskTargetActionInput) (*domainworkflow.Instance, error)
	CreateSubstitution(ctx context.Context, in CreateSubstitutionInput) (*domainworkflow.SubstitutionWindow, error)
	RevokeSubstitution(ctx context.Context, in RevokeSubstitutionInput) (*domainworkflow.SubstitutionWindow, error)
	ProcessDueTimers(ctx context.Context, workerID string, limit int, leaseDuration time.Duration) (int, error)
	RunTimerWorker(ctx context.Context, workerID string, pollInterval time.Duration, batchSize int) error
}

type Options struct {
	Jobs       *jobsvc.Service
	UnitOfWork repository.UnitOfWork
	Now        func() time.Time
}

type CreateDefinitionInput struct {
	ID          shared.ID
	Key         string
	Name        string
	Version     int
	Nodes       []domainworkflow.Node
	Transitions []domainworkflow.Transition
	Now         time.Time
}

type StartInput struct {
	ID           shared.ID
	DefinitionID shared.ID
	BusinessType string
	BusinessID   string
	Title        string
	Starter      domainworkflow.Actor
	Variables    map[string]domainworkflow.Value
	Now          time.Time
}

type TaskActionInput struct {
	InstanceID shared.ID
	TaskID     shared.ID
	Actor      domainworkflow.Actor
	Comment    string
	Now        time.Time
}

type InstanceActionInput struct {
	InstanceID shared.ID
	Actor      domainworkflow.Actor
	Comment    string
	Now        time.Time
}

type TaskTargetActionInput struct {
	InstanceID shared.ID
	TaskID     shared.ID
	Actor      domainworkflow.Actor
	Target     domainworkflow.Actor
	Comment    string
	Now        time.Time
}

type CreateSubstitutionInput struct {
	ID         shared.ID
	Principal  domainworkflow.Actor
	Substitute domainworkflow.Actor
	StartsAt   time.Time
	EndsAt     time.Time
	CreatedBy  domainworkflow.Actor
	Reason     string
	Now        time.Time
}

type RevokeSubstitutionInput struct {
	ID        shared.ID
	Principal domainworkflow.Actor
	Now       time.Time
}

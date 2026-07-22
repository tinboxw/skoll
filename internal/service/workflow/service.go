package workflow

import (
	"context"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
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

// InstanceMutation runs inside the repository's atomic update boundary.
// Implementations may retry it, so the callback must not perform external side effects.
type InstanceMutation func(instance *domainworkflow.Instance) (changed bool, err error)

type Repository interface {
	DefinitionRepository
	InstanceRepository
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
	Transfer(ctx context.Context, in TaskTargetActionInput) (*domainworkflow.Instance, error)
	Copy(ctx context.Context, in TaskTargetActionInput) (*domainworkflow.Instance, error)
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

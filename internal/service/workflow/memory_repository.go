package workflow

import (
	"context"
	"fmt"
	"sync"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
)

type MemoryRepository struct {
	mu          sync.RWMutex
	definitions map[shared.ID]domainworkflow.Definition
	instances   map[shared.ID]domainworkflow.Instance
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		definitions: map[shared.ID]domainworkflow.Definition{},
		instances:   map[shared.ID]domainworkflow.Instance{},
	}
}

func (r *MemoryRepository) SaveDefinition(_ context.Context, definition domainworkflow.Definition) error {
	if r == nil {
		return fmt.Errorf("workflow repository is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.definitions[definition.ID] = cloneDefinition(definition)
	return nil
}

func (r *MemoryRepository) GetDefinition(_ context.Context, id shared.ID) (*domainworkflow.Definition, error) {
	if r == nil {
		return nil, fmt.Errorf("workflow repository is required")
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	definition, ok := r.definitions[id]
	if !ok {
		return nil, fmt.Errorf("workflow definition not found")
	}
	definition = cloneDefinition(definition)
	return &definition, nil
}

func (r *MemoryRepository) SaveInstance(_ context.Context, instance domainworkflow.Instance) error {
	if r == nil {
		return fmt.Errorf("workflow repository is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.instances[instance.ID] = cloneInstance(instance)
	return nil
}

func (r *MemoryRepository) GetInstance(_ context.Context, id shared.ID) (*domainworkflow.Instance, error) {
	if r == nil {
		return nil, fmt.Errorf("workflow repository is required")
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	instance, ok := r.instances[id]
	if !ok {
		return nil, fmt.Errorf("workflow instance not found")
	}
	instance = cloneInstance(instance)
	return &instance, nil
}

func (r *MemoryRepository) UpdateInstance(ctx context.Context, id shared.ID, mutate InstanceMutation) (*domainworkflow.Instance, error) {
	if r == nil {
		return nil, fmt.Errorf("workflow repository is required")
	}
	if mutate == nil {
		return nil, fmt.Errorf("workflow instance mutation is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	stored, ok := r.instances[id]
	if !ok {
		return nil, fmt.Errorf("workflow instance not found")
	}
	instance := cloneInstance(stored)
	changed, err := mutate(&instance)
	if err != nil {
		return nil, err
	}
	if changed {
		r.instances[id] = cloneInstance(instance)
	}
	result := cloneInstance(instance)
	return &result, nil
}

func cloneDefinition(definition domainworkflow.Definition) domainworkflow.Definition {
	definition.Nodes = append([]domainworkflow.Node(nil), definition.Nodes...)
	for idx := range definition.Nodes {
		definition.Nodes[idx].Assignees = append([]shared.ID(nil), definition.Nodes[idx].Assignees...)
	}
	definition.Transitions = append([]domainworkflow.Transition(nil), definition.Transitions...)
	return definition
}

func cloneInstance(instance domainworkflow.Instance) domainworkflow.Instance {
	instance.Tasks = append([]domainworkflow.Task(nil), instance.Tasks...)
	for idx := range instance.Tasks {
		if instance.Tasks[idx].CompletedAt != nil {
			completedAt := *instance.Tasks[idx].CompletedAt
			instance.Tasks[idx].CompletedAt = &completedAt
		}
	}
	instance.Timeline = append([]domainworkflow.Action(nil), instance.Timeline...)
	return instance
}

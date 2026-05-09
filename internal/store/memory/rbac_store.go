package memory

import (
	"context"
	"sync"

	"github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type RBACStore struct {
	mu       sync.RWMutex
	bindings map[shared.ID]*rbac.Binding
	rules    map[shared.ID][]rbac.PolicyRule
}

func NewRBACStore() *RBACStore {
	return &RBACStore{
		bindings: make(map[shared.ID]*rbac.Binding),
		rules:    make(map[shared.ID][]rbac.PolicyRule),
	}
}

func (s *RBACStore) CreateBinding(_ context.Context, binding *rbac.Binding) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bindings[binding.ID] = binding
	return nil
}

func (s *RBACStore) DeleteBinding(_ context.Context, id shared.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.bindings, id)
	return nil
}

func (s *RBACStore) ListBindingsBySubject(_ context.Context, subjectType rbac.SubjectType, subjectID shared.ID) ([]*rbac.Binding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*rbac.Binding, 0)
	for _, b := range s.bindings {
		if b.SubjectType == subjectType && b.SubjectID == subjectID {
			out = append(out, b)
		}
	}
	return out, nil
}

func (s *RBACStore) ListPolicyRulesByRoleID(_ context.Context, roleID shared.ID) ([]rbac.PolicyRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rules := s.rules[roleID]
	cloned := make([]rbac.PolicyRule, len(rules))
	copy(cloned, rules)
	return cloned, nil
}

func (s *RBACStore) ReplacePolicyRules(_ context.Context, roleID shared.ID, rules []rbac.PolicyRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cloned := make([]rbac.PolicyRule, len(rules))
	copy(cloned, rules)
	s.rules[roleID] = cloned
	return nil
}

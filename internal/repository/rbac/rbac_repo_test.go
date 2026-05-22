package rbac

import (
	"context"
	"errors"
	"testing"

	"github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

// MockRBACRepository implements RBACRepository for testing
type MockRBACRepository struct {
	bindings     map[shared.ID]*rbac.Binding
	subjectIndex map[string][]*rbac.Binding // "user:123" -> [binding1, binding2]
	policyRules  map[shared.ID][]rbac.PolicyRule
	err          error
}

func NewMockRBACRepository() *MockRBACRepository {
	return &MockRBACRepository{
		bindings:     make(map[shared.ID]*rbac.Binding),
		subjectIndex: make(map[string][]*rbac.Binding),
		policyRules:  make(map[shared.ID][]rbac.PolicyRule),
	}
}

func (m *MockRBACRepository) WithError(err error) *MockRBACRepository {
	m.err = err
	return m
}

func (m *MockRBACRepository) WithBindings(bindings ...*rbac.Binding) *MockRBACRepository {
	for _, b := range bindings {
		m.bindings[b.ID] = b
		key := string(b.SubjectType) + ":" + string(b.SubjectID)
		m.subjectIndex[key] = append(m.subjectIndex[key], b)
	}
	return m
}

func (m *MockRBACRepository) CreateBinding(ctx context.Context, binding *rbac.Binding) error {
	if m.err != nil {
		return m.err
	}
	m.bindings[binding.ID] = binding
	key := string(binding.SubjectType) + ":" + string(binding.SubjectID)
	m.subjectIndex[key] = append(m.subjectIndex[key], binding)
	return nil
}

func (m *MockRBACRepository) DeleteBinding(ctx context.Context, id shared.ID) error {
	if m.err != nil {
		return m.err
	}
	b, ok := m.bindings[id]
	if !ok {
		return errors.New("binding not found")
	}
	delete(m.bindings, id)
	key := string(b.SubjectType) + ":" + string(b.SubjectID)
	bindings := m.subjectIndex[key]
	for i, binding := range bindings {
		if binding.ID == id {
			m.subjectIndex[key] = append(bindings[:i], bindings[i+1:]...)
			break
		}
	}
	return nil
}

func (m *MockRBACRepository) ListBindingsBySubject(ctx context.Context, subjectType rbac.SubjectType, subjectID shared.ID) ([]*rbac.Binding, error) {
	if m.err != nil {
		return nil, m.err
	}
	key := string(subjectType) + ":" + string(subjectID)
	return m.subjectIndex[key], nil
}

func (m *MockRBACRepository) ListPolicyRulesByRoleID(ctx context.Context, roleID shared.ID) ([]rbac.PolicyRule, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.policyRules[roleID], nil
}

func (m *MockRBACRepository) ReplacePolicyRules(ctx context.Context, roleID shared.ID, rules []rbac.PolicyRule) error {
	if m.err != nil {
		return m.err
	}
	m.policyRules[roleID] = rules
	return nil
}

// Tests
func TestMockRBACRepository_CreateBinding(t *testing.T) {
	repo := NewMockRBACRepository()

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		binding := &rbac.Binding{
			ID:          shared.ID("binding-1"),
			SubjectType: rbac.SubjectUser,
			SubjectID:   shared.ID("user-1"),
			RoleID:      shared.ID("role-1"),
		}
		err := repo.CreateBinding(ctx, binding)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify we can retrieve it
		bindings, err := repo.ListBindingsBySubject(ctx, rbac.SubjectUser, shared.ID("user-1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(bindings) != 1 {
			t.Errorf("expected 1 binding, got %d", len(bindings))
		}
	})

	t.Run("with error", func(t *testing.T) {
		errRepo := repo.WithError(errors.New("database error"))
		ctx := context.Background()
		err := errRepo.CreateBinding(ctx, &rbac.Binding{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestMockRBACRepository_DeleteBinding(t *testing.T) {
	repo := NewMockRBACRepository()
	testBinding := &rbac.Binding{
		ID:          shared.ID("delete-binding"),
		SubjectType: rbac.SubjectUser,
		SubjectID:   shared.ID("user-2"),
	}
	repo.WithBindings(testBinding)

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		err := repo.DeleteBinding(ctx, testBinding.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Verify it's deleted
		bindings, _ := repo.ListBindingsBySubject(ctx, rbac.SubjectUser, shared.ID("user-2"))
		if len(bindings) != 0 {
			t.Errorf("expected 0 bindings after delete, got %d", len(bindings))
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctx := context.Background()
		err := repo.DeleteBinding(ctx, shared.ID("nonexistent"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestMockRBACRepository_ListPolicyRulesByRoleID(t *testing.T) {
	repo := NewMockRBACRepository()
	roleID := shared.ID("role-admin")
	rules := []rbac.PolicyRule{
		{Resource: "users", Action: "read"},
		{Resource: "users", Action: "write"},
	}
	repo.policyRules[roleID] = rules

	t.Run("found", func(t *testing.T) {
		ctx := context.Background()
		result, err := repo.ListPolicyRulesByRoleID(ctx, roleID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 rules, got %d", len(result))
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctx := context.Background()
		result, err := repo.ListPolicyRulesByRoleID(ctx, shared.ID("nonexistent-role"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil && len(result) != 0 {
			t.Errorf("expected nil or empty, got %d rules", len(result))
		}
	})
}

func TestMockRBACRepository_ReplacePolicyRules(t *testing.T) {
	repo := NewMockRBACRepository()
	roleID := shared.ID("role-editor")

	t.Run("replace existing", func(t *testing.T) {
		ctx := context.Background()
		initialRules := []rbac.PolicyRule{{Resource: "posts", Action: "read"}}
		repo.policyRules[roleID] = initialRules

		newRules := []rbac.PolicyRule{
			{Resource: "posts", Action: "read"},
			{Resource: "posts", Action: "write"},
			{Resource: "comments", Action: "moderate"},
		}
		err := repo.ReplacePolicyRules(ctx, roleID, newRules)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := repo.ListPolicyRulesByRoleID(ctx, roleID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("expected 3 rules after replace, got %d", len(result))
		}
	})
}

package role

import (
	"context"
	"errors"
	"testing"

	"github.com/tinboxw/skoll/internal/domain/role"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

// MockRoleRepository implements RoleRepository for testing
type MockRoleRepository struct {
	roles map[shared.ID]*role.Role
	byKey map[string]*role.Role
	err   error
}

func NewMockRoleRepository() *MockRoleRepository {
	return &MockRoleRepository{
		roles: make(map[shared.ID]*role.Role),
		byKey: make(map[string]*role.Role),
	}
}

func (m *MockRoleRepository) WithError(err error) *MockRoleRepository {
	m.err = err
	return m
}

func (m *MockRoleRepository) WithRoles(roles ...*role.Role) *MockRoleRepository {
	for _, r := range roles {
		m.roles[r.ID] = r
		m.byKey[r.Key] = r
	}
	return m
}

func (m *MockRoleRepository) GetByID(ctx context.Context, id shared.ID) (*role.Role, error) {
	if m.err != nil {
		return nil, m.err
	}
	r, ok := m.roles[id]
	if !ok {
		return nil, errors.New("role not found")
	}
	return r, nil
}

func (m *MockRoleRepository) GetByKey(ctx context.Context, key string) (*role.Role, error) {
	if m.err != nil {
		return nil, m.err
	}
	r, ok := m.byKey[key]
	if !ok {
		return nil, errors.New("role not found")
	}
	return r, nil
}

func (m *MockRoleRepository) List(ctx context.Context, offset, limit int) ([]*role.Role, error) {
	if m.err != nil {
		return nil, m.err
	}
	var roles []*role.Role
	count := 0
	skipped := 0
	for _, r := range m.roles {
		if skipped < offset {
			skipped++
			continue
		}
		if count >= limit {
			break
		}
		roles = append(roles, r)
		count++
	}
	return roles, nil
}

func (m *MockRoleRepository) Save(ctx context.Context, entity *role.Role) error {
	if m.err != nil {
		return m.err
	}
	m.roles[entity.ID] = entity
	m.byKey[entity.Key] = entity
	return nil
}

func (m *MockRoleRepository) Delete(ctx context.Context, id shared.ID) error {
	if m.err != nil {
		return m.err
	}
	r, ok := m.roles[id]
	if !ok {
		return errors.New("role not found")
	}
	delete(m.roles, id)
	delete(m.byKey, r.Key)
	return nil
}

// Tests
func TestMockRoleRepository_GetByID(t *testing.T) {
	repo := NewMockRoleRepository()
	testID := shared.ID("role-id-1")
	testRole := &role.Role{
		ID:   testID,
		Key:  "admin",
		Name: "Administrator",
	}
	repo.WithRoles(testRole)

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		result, err := repo.GetByID(ctx, testID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != testID {
			t.Errorf("expected ID %s, got %s", testID, result.ID)
		}
		if result.Key != "admin" {
			t.Errorf("expected Key admin, got %s", result.Key)
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctx := context.Background()
		_, err := repo.GetByID(ctx, shared.ID("nonexistent"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestMockRoleRepository_GetByKey(t *testing.T) {
	repo := NewMockRoleRepository()
	testRole := &role.Role{
		ID:   shared.ID("role-id-2"),
		Key:  "editor",
		Name: "Editor",
	}
	repo.WithRoles(testRole)

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		result, err := repo.GetByKey(ctx, "editor")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Key != "editor" {
			t.Errorf("expected Key editor, got %s", result.Key)
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctx := context.Background()
		_, err := repo.GetByKey(ctx, "nonexistent")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestMockRoleRepository_List(t *testing.T) {
	repo := NewMockRoleRepository()
	roles := []*role.Role{
		{ID: shared.ID("r1"), Key: "role1"},
		{ID: shared.ID("r2"), Key: "role2"},
		{ID: shared.ID("r3"), Key: "role3"},
	}
	repo.WithRoles(roles...)

	t.Run("list all", func(t *testing.T) {
		ctx := context.Background()
		result, err := repo.List(ctx, 0, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("expected 3 roles, got %d", len(result))
		}
	})

	t.Run("with pagination", func(t *testing.T) {
		ctx := context.Background()
		result, err := repo.List(ctx, 1, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 roles, got %d", len(result))
		}
	})
}

func TestMockRoleRepository_SaveAndDelete(t *testing.T) {
	repo := NewMockRoleRepository()

	t.Run("save and retrieve", func(t *testing.T) {
		newRole := &role.Role{
			ID:   shared.ID("new-role"),
			Key:  "new-key",
			Name: "New Role",
		}
		ctx := context.Background()
		err := repo.Save(ctx, newRole)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		saved, err := repo.GetByKey(ctx, "new-key")
		if err != nil {
			t.Fatalf("failed to retrieve saved role: %v", err)
		}
		if saved.Name != "New Role" {
			t.Errorf("expected Name New Role, got %s", saved.Name)
		}
	})

	t.Run("delete", func(t *testing.T) {
		testRole := &role.Role{
			ID:  shared.ID("delete-role"),
			Key: "delete-key",
		}
		repo.WithRoles(testRole)
		ctx := context.Background()
		err := repo.Delete(ctx, testRole.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_, err = repo.GetByID(ctx, testRole.ID)
		if err == nil {
			t.Fatal("expected error after delete, got nil")
		}
	})
}

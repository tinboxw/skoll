package user

import (
	"context"
	"errors"
	"testing"

	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/domain/user"
)

// MockUserRepository implements UserRepository for testing
type MockUserRepository struct {
	users     map[shared.ID]*user.User
	byAccount map[string]*user.User
	byEmail   map[string]user.Email
	err       error
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users:     make(map[shared.ID]*user.User),
		byAccount: make(map[string]*user.User),
		byEmail:   make(map[string]user.Email),
	}
}

func (m *MockUserRepository) WithError(err error) *MockUserRepository {
	m.err = err
	return m
}

func (m *MockUserRepository) WithUsers(users ...*user.User) *MockUserRepository {
	for _, u := range users {
		m.users[u.ID] = u
		m.byAccount[u.Account] = u
		if string(u.Email) != "" {
			m.byEmail[string(u.Email)] = u.Email
		}
	}
	return m
}

func (m *MockUserRepository) GetByID(ctx context.Context, id shared.ID) (*user.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	u, ok := m.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}

func (m *MockUserRepository) GetByAccount(ctx context.Context, account string) (*user.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	u, ok := m.byAccount[account]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email user.Email) (*user.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	emailStr := string(email)
	for _, u := range m.users {
		if string(u.Email) == emailStr {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *MockUserRepository) List(ctx context.Context, offset, limit int) ([]*user.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	var users []*user.User
	count := 0
	skipped := 0
	for _, u := range m.users {
		if skipped < offset {
			skipped++
			continue
		}
		if count >= limit {
			break
		}
		users = append(users, u)
		count++
	}
	return users, nil
}

func (m *MockUserRepository) Save(ctx context.Context, entity *user.User) error {
	if m.err != nil {
		return m.err
	}
	m.users[entity.ID] = entity
	m.byAccount[entity.Account] = entity
	return nil
}

func (m *MockUserRepository) Delete(ctx context.Context, id shared.ID) error {
	if m.err != nil {
		return m.err
	}
	u, ok := m.users[id]
	if !ok {
		return errors.New("user not found")
	}
	delete(m.users, id)
	delete(m.byAccount, u.Account)
	return nil
}

// Tests
func TestMockUserRepository_GetByID(t *testing.T) {
	repo := NewMockUserRepository()
	testID := shared.ID("test-id-1")
	testUser := &user.User{
		ID:      testID,
		Account: "testuser",
		Name:    "Test User",
	}
	repo.WithUsers(testUser)

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		result, err := repo.GetByID(ctx, testID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != testID {
			t.Errorf("expected ID %s, got %s", testID, result.ID)
		}
		if result.Account != "testuser" {
			t.Errorf("expected Account testuser, got %s", result.Account)
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctx := context.Background()
		_, err := repo.GetByID(ctx, shared.ID("nonexistent"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("with error", func(t *testing.T) {
		errRepo := repo.WithError(errors.New("database error"))
		ctx := context.Background()
		_, err := errRepo.GetByID(ctx, testID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestMockUserRepository_GetByAccount(t *testing.T) {
	repo := NewMockUserRepository()
	testUser := &user.User{
		ID:      shared.ID("test-id-2"),
		Account: "admin",
		Name:    "Admin User",
	}
	repo.WithUsers(testUser)

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		result, err := repo.GetByAccount(ctx, "admin")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Account != "admin" {
			t.Errorf("expected Account admin, got %s", result.Account)
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctx := context.Background()
		_, err := repo.GetByAccount(ctx, "nonexistent")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestMockUserRepository_List(t *testing.T) {
	repo := NewMockUserRepository()
	users := []*user.User{
		{ID: shared.ID("id-1"), Account: "user1"},
		{ID: shared.ID("id-2"), Account: "user2"},
		{ID: shared.ID("id-3"), Account: "user3"},
	}
	repo.WithUsers(users...)

	t.Run("list all", func(t *testing.T) {
		ctx := context.Background()
		result, err := repo.List(ctx, 0, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("expected 3 users, got %d", len(result))
		}
	})

	t.Run("with pagination", func(t *testing.T) {
		ctx := context.Background()
		result, err := repo.List(ctx, 1, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 users, got %d", len(result))
		}
	})
}

func TestMockUserRepository_Save(t *testing.T) {
	repo := NewMockUserRepository()
	newUser := &user.User{
		ID:      shared.ID("new-id"),
		Account: "newuser",
		Name:    "New User",
	}

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		err := repo.Save(ctx, newUser)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Verify it was saved
		saved, err := repo.GetByID(ctx, newUser.ID)
		if err != nil {
			t.Fatalf("failed to retrieve saved user: %v", err)
		}
		if saved.Account != "newuser" {
			t.Errorf("expected Account newuser, got %s", saved.Account)
		}
	})
}

func TestMockUserRepository_Delete(t *testing.T) {
	repo := NewMockUserRepository()
	testUser := &user.User{
		ID:      shared.ID("delete-id"),
		Account: "deletable",
	}
	repo.WithUsers(testUser)

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		err := repo.Delete(ctx, testUser.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Verify it was deleted
		_, err = repo.GetByID(ctx, testUser.ID)
		if err == nil {
			t.Fatal("expected error after delete, got nil")
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctx := context.Background()
		err := repo.Delete(ctx, shared.ID("nonexistent"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

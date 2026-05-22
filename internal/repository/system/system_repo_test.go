package system

import (
	"context"
	"errors"
	"testing"

	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/domain/system"
)

// MockSystemRepository implements SystemRepository for testing
type MockSystemRepository struct {
	settings map[string]*system.Setting
	err      error
}

func NewMockSystemRepository() *MockSystemRepository {
	return &MockSystemRepository{
		settings: make(map[string]*system.Setting),
	}
}

func (m *MockSystemRepository) WithError(err error) *MockSystemRepository {
	m.err = err
	return m
}

func (m *MockSystemRepository) WithSettings(settings ...*system.Setting) *MockSystemRepository {
	for _, s := range settings {
		m.settings[s.Key] = s
	}
	return m
}

// Note: You'll need to add these methods based on the actual SystemRepository interface
// For now, I'm implementing common CRUD operations that are likely present

func (m *MockSystemRepository) GetByID(ctx context.Context, id shared.ID) (*system.Setting, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, s := range m.settings {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, errors.New("setting not found")
}

func (m *MockSystemRepository) GetByKey(ctx context.Context, key string) (*system.Setting, error) {
	if m.err != nil {
		return nil, m.err
	}
	s, ok := m.settings[key]
	if !ok {
		return nil, errors.New("setting not found")
	}
	return s, nil
}

func (m *MockSystemRepository) List(ctx context.Context, offset, limit int) ([]*system.Setting, error) {
	if m.err != nil {
		return nil, m.err
	}
	var settings []*system.Setting
	count := 0
	skipped := 0
	for _, s := range m.settings {
		if skipped < offset {
			skipped++
			continue
		}
		if count >= limit {
			break
		}
		settings = append(settings, s)
		count++
	}
	return settings, nil
}

func (m *MockSystemRepository) Save(ctx context.Context, entity *system.Setting) error {
	if m.err != nil {
		return m.err
	}
	m.settings[entity.Key] = entity
	return nil
}

func (m *MockSystemRepository) Delete(ctx context.Context, id shared.ID) error {
	if m.err != nil {
		return m.err
	}
	var keyToDelete string
	for key, s := range m.settings {
		if s.ID == id {
			keyToDelete = key
			break
		}
	}
	if keyToDelete == "" {
		return errors.New("setting not found")
	}
	delete(m.settings, keyToDelete)
	return nil
}

// Tests
func TestMockSystemRepository_GetByKey(t *testing.T) {
	repo := NewMockSystemRepository()
	testSetting := &system.Setting{
		ID:    shared.ID("setting-1"),
		Key:   "app.name",
		Value: "Skoll Platform",
	}
	repo.WithSettings(testSetting)

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		result, err := repo.GetByKey(ctx, "app.name")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Key != "app.name" {
			t.Errorf("expected Key app.name, got %s", result.Key)
		}
		if result.Value != "Skoll Platform" {
			t.Errorf("expected Value Skoll Platform, got %s", result.Value)
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctx := context.Background()
		_, err := repo.GetByKey(ctx, "nonexistent.key")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("with error", func(t *testing.T) {
		errRepo := repo.WithError(errors.New("database error"))
		ctx := context.Background()
		_, err := errRepo.GetByKey(ctx, "app.name")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestMockSystemRepository_List(t *testing.T) {
	repo := NewMockSystemRepository()
	settings := []*system.Setting{
		{ID: shared.ID("s1"), Key: "setting.1", Value: "value1"},
		{ID: shared.ID("s2"), Key: "setting.2", Value: "value2"},
		{ID: shared.ID("s3"), Key: "setting.3", Value: "value3"},
	}
	repo.WithSettings(settings...)

	t.Run("list all", func(t *testing.T) {
		ctx := context.Background()
		result, err := repo.List(ctx, 0, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("expected 3 settings, got %d", len(result))
		}
	})

	t.Run("with pagination", func(t *testing.T) {
		ctx := context.Background()
		result, err := repo.List(ctx, 1, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 settings, got %d", len(result))
		}
	})
}

func TestMockSystemRepository_SaveAndDelete(t *testing.T) {
	repo := NewMockSystemRepository()

	t.Run("save and retrieve", func(t *testing.T) {
		newSetting := &system.Setting{
			ID:    shared.ID("new-setting"),
			Key:   "new.setting.key",
			Value: "new value",
		}
		ctx := context.Background()
		err := repo.Save(ctx, newSetting)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		saved, err := repo.GetByKey(ctx, "new.setting.key")
		if err != nil {
			t.Fatalf("failed to retrieve saved setting: %v", err)
		}
		if saved.Value != "new value" {
			t.Errorf("expected Value 'new value', got '%s'", saved.Value)
		}
	})

	t.Run("delete existing", func(t *testing.T) {
		deleteSetting := &system.Setting{
			ID:    shared.ID("delete-setting"),
			Key:   "delete.key",
			Value: "delete me",
		}
		repo.WithSettings(deleteSetting)
		ctx := context.Background()
		err := repo.Delete(ctx, deleteSetting.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_, err = repo.GetByKey(ctx, "delete.key")
		if err == nil {
			t.Fatal("expected error after delete, got nil")
		}
	})

	t.Run("delete nonexistent", func(t *testing.T) {
		ctx := context.Background()
		err := repo.Delete(ctx, shared.ID("nonexistent"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

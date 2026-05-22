package plugin

import (
	"context"
	"errors"
	"testing"

	"github.com/tinboxw/skoll/internal/plugin"
)

// MockPluginRepository implements PluginRepository for testing
type MockPluginRepository struct {
	plugins map[string]*plugin.Info
	err     error
}

func NewMockPluginRepository() *MockPluginRepository {
	return &MockPluginRepository{
		plugins: make(map[string]*plugin.Info),
	}
}

func (m *MockPluginRepository) WithError(err error) *MockPluginRepository {
	m.err = err
	return m
}

func (m *MockPluginRepository) WithPlugins(plugins ...plugin.Info) *MockPluginRepository {
	for _, p := range plugins {
		m.plugins[p.ID] = &p
	}
	return m
}

func (m *MockPluginRepository) Get(ctx context.Context, pluginID string) (*plugin.Info, error) {
	if m.err != nil {
		return nil, m.err
	}
	p, ok := m.plugins[pluginID]
	if !ok {
		return nil, errors.New("plugin not found")
	}
	return p, nil
}

func (m *MockPluginRepository) List(ctx context.Context) ([]plugin.Info, error) {
	if m.err != nil {
		return nil, m.err
	}
	var plugins []plugin.Info
	for _, p := range m.plugins {
		plugins = append(plugins, *p)
	}
	return plugins, nil
}

func (m *MockPluginRepository) Save(ctx context.Context, info plugin.Info) error {
	if m.err != nil {
		return m.err
	}
	m.plugins[info.ID] = &info
	return nil
}

func (m *MockPluginRepository) Delete(ctx context.Context, pluginID string) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.plugins[pluginID]; !ok {
		return errors.New("plugin not found")
	}
	delete(m.plugins, pluginID)
	return nil
}

// Tests
func TestMockPluginRepository_Get(t *testing.T) {
	repo := NewMockPluginRepository()
	testPlugin := plugin.Info{
		ID:          "test-plugin-1",
		Name:        "Test Plugin",
		Version:     "1.0.0",
		Description: "A test plugin",
		State:       plugin.StateEnabled,
	}
	repo.WithPlugins(testPlugin)

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		result, err := repo.Get(ctx, "test-plugin-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != "test-plugin-1" {
			t.Errorf("expected ID test-plugin-1, got %s", result.ID)
		}
		if result.Name != "Test Plugin" {
			t.Errorf("expected Name Test Plugin, got %s", result.Name)
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctx := context.Background()
		_, err := repo.Get(ctx, "nonexistent")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("with error", func(t *testing.T) {
		errRepo := repo.WithError(errors.New("database error"))
		ctx := context.Background()
		_, err := errRepo.Get(ctx, "test-plugin-1")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestMockPluginRepository_List(t *testing.T) {
	repo := NewMockPluginRepository()
	plugins := []plugin.Info{
		{ID: "plugin-a", Name: "Plugin A", Version: "1.0.0"},
		{ID: "plugin-b", Name: "Plugin B", Version: "2.0.0"},
		{ID: "plugin-c", Name: "Plugin C", Version: "3.0.0"},
	}
	repo.WithPlugins(plugins...)

	t.Run("list all", func(t *testing.T) {
		ctx := context.Background()
		result, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("expected 3 plugins, got %d", len(result))
		}
	})

	t.Run("empty list", func(t *testing.T) {
		emptyRepo := NewMockPluginRepository()
		ctx := context.Background()
		result, err := emptyRepo.List(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected 0 plugins, got %d", len(result))
		}
	})
}

func TestMockPluginRepository_SaveAndDelete(t *testing.T) {
	repo := NewMockPluginRepository()

	t.Run("save and retrieve", func(t *testing.T) {
		newPlugin := plugin.Info{
			ID:          "new-plugin",
			Name:        "New Plugin",
			Version:     "1.0.0",
			Description: "A newly created plugin",
		}
		ctx := context.Background()
		err := repo.Save(ctx, newPlugin)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		saved, err := repo.Get(ctx, "new-plugin")
		if err != nil {
			t.Fatalf("failed to retrieve saved plugin: %v", err)
		}
		if saved.Name != "New Plugin" {
			t.Errorf("expected Name New Plugin, got %s", saved.Name)
		}
	})

	t.Run("delete existing", func(t *testing.T) {
		deletePlugin := plugin.Info{
			ID:   "delete-plugin",
			Name: "Delete Me",
		}
		repo.WithPlugins(deletePlugin)
		ctx := context.Background()
		err := repo.Delete(ctx, "delete-plugin")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_, err = repo.Get(ctx, "delete-plugin")
		if err == nil {
			t.Fatal("expected error after delete, got nil")
		}
	})

	t.Run("delete nonexistent", func(t *testing.T) {
		ctx := context.Background()
		err := repo.Delete(ctx, "nonexistent")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

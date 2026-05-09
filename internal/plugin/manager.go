package plugin

import (
	"sort"
	"sync"
	"time"
)

type Manager interface {
	Install(path string) (Info, error)
	Enable(pluginID string) error
	Disable(pluginID string) error
	Uninstall(pluginID string) error
	List() []Info
	Get(pluginID string) (Info, error)
}

type RuntimeManager struct {
	mu       sync.RWMutex
	loader   MetadataLoader
	resolver DependencyResolver
	plugins  map[string]Info
}

func NewRuntimeManager(loader MetadataLoader, resolver DependencyResolver) *RuntimeManager {
	if loader == nil {
		loader = NewFileLoader()
	}
	if resolver == nil {
		resolver = NewTopologicalResolver()
	}

	return &RuntimeManager{
		loader:   loader,
		resolver: resolver,
		plugins:  make(map[string]Info),
	}
}

func (m *RuntimeManager) Install(path string) (Info, error) {
	info, err := m.loader.Load(path)
	if err != nil {
		return Info{}, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.plugins[info.ID]; ok && existing.State != StateUninstalled {
		return Info{}, ErrPluginAlreadyExists
	}

	now := time.Now().UTC()
	info.State = StateInstalled
	info.InstalledAt = now
	info.EnabledAt = nil
	info.Source = path
	m.plugins[info.ID] = info

	return info, nil
}

func (m *RuntimeManager) Enable(pluginID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, ok := m.plugins[pluginID]
	if !ok {
		return ErrPluginNotFound
	}
	if info.State == StateUninstalled {
		return ErrPluginInvalidState
	}

	order, err := m.resolver.ResolveEnableOrder(pluginID, m.plugins)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	for _, id := range order {
		p := m.plugins[id]
		if p.State == StateEnabled {
			continue
		}
		if p.State == StateUninstalled {
			return ErrPluginInvalidState
		}
		p.State = StateEnabled
		p.EnabledAt = &now
		m.plugins[id] = p
	}

	return nil
}

func (m *RuntimeManager) Disable(pluginID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, ok := m.plugins[pluginID]
	if !ok {
		return ErrPluginNotFound
	}
	if info.State == StateUninstalled {
		return ErrPluginInvalidState
	}

	if err := m.resolver.CanDisable(pluginID, m.plugins); err != nil {
		return err
	}

	if info.State == StateEnabled {
		info.State = StateDisabled
		info.EnabledAt = nil
		m.plugins[pluginID] = info
	}

	return nil
}

func (m *RuntimeManager) Uninstall(pluginID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, ok := m.plugins[pluginID]
	if !ok {
		return ErrPluginNotFound
	}
	if info.State == StateUninstalled {
		return nil
	}

	if err := m.resolver.CanDisable(pluginID, m.plugins); err != nil {
		return err
	}

	info.State = StateUninstalled
	info.EnabledAt = nil
	m.plugins[pluginID] = info
	return nil
}

func (m *RuntimeManager) List() []Info {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugins := make([]Info, 0, len(m.plugins))
	for _, info := range m.plugins {
		plugins = append(plugins, info)
	}

	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].ID < plugins[j].ID
	})

	return plugins
}

func (m *RuntimeManager) Get(pluginID string) (Info, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, ok := m.plugins[pluginID]
	if !ok {
		return Info{}, ErrPluginNotFound
	}

	return info, nil
}

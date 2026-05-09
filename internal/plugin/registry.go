package plugin

import (
	"sync"
)

type RouteExtension struct {
	Method string
	Path   string
}

type MenuExtension struct {
	Name string
	Path string
}

type WidgetExtension struct {
	Name string
}

type SettingExtension struct {
	Name string
	Path string
}

type RegistrySnapshot struct {
	Routes      []RouteExtension
	Middlewares []string
	Events      []string
	Menus       []MenuExtension
	Widgets     []WidgetExtension
	Settings    []SettingExtension
}

type ExtensionRegistry interface {
	RegisterRoute(route RouteExtension)
	RegisterMiddleware(name string)
	RegisterEventHandler(name string)
	RegisterMenuItem(menu MenuExtension)
	RegisterWidget(widget WidgetExtension)
	RegisterSettingPage(setting SettingExtension)
	Snapshot() RegistrySnapshot
}

type MemoryRegistry struct {
	mu          sync.RWMutex
	routes      []RouteExtension
	middlewares []string
	events      []string
	menus       []MenuExtension
	widgets     []WidgetExtension
	settings    []SettingExtension
}

func NewMemoryRegistry() *MemoryRegistry {
	return &MemoryRegistry{}
}

func (r *MemoryRegistry) RegisterRoute(route RouteExtension) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes = append(r.routes, route)
}

func (r *MemoryRegistry) RegisterMiddleware(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middlewares = append(r.middlewares, name)
}

func (r *MemoryRegistry) RegisterEventHandler(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, name)
}

func (r *MemoryRegistry) RegisterMenuItem(menu MenuExtension) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.menus = append(r.menus, menu)
}

func (r *MemoryRegistry) RegisterWidget(widget WidgetExtension) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.widgets = append(r.widgets, widget)
}

func (r *MemoryRegistry) RegisterSettingPage(setting SettingExtension) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.settings = append(r.settings, setting)
}

func (r *MemoryRegistry) Snapshot() RegistrySnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return RegistrySnapshot{
		Routes:      append([]RouteExtension(nil), r.routes...),
		Middlewares: append([]string(nil), r.middlewares...),
		Events:      append([]string(nil), r.events...),
		Menus:       append([]MenuExtension(nil), r.menus...),
		Widgets:     append([]WidgetExtension(nil), r.widgets...),
		Settings:    append([]SettingExtension(nil), r.settings...),
	}
}

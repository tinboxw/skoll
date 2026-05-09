package dashboard

import "github.com/tinboxw/skoll/internal/plugin"

// Plugin wires builtin dashboard extension points.
type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) ID() string {
	return "builtin-dashboard"
}

func (p *Plugin) Name() string {
	return "Builtin Dashboard"
}

func (p *Plugin) Version() string {
	return "1.0.0"
}

func (p *Plugin) Register(registry plugin.ExtensionRegistry) error {
	registry.RegisterRoute(plugin.RouteExtension{Method: "GET", Path: "/v1/dashboard/widgets"})
	registry.RegisterWidget(plugin.WidgetExtension{Name: "system-health-widget"})
	registry.RegisterWidget(plugin.WidgetExtension{Name: "active-plugin-widget"})
	registry.RegisterMenuItem(plugin.MenuExtension{Name: "仪表盘", Path: "/dashboard"})
	return nil
}

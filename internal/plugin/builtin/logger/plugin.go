package logger

import "github.com/tinboxw/skoll/internal/plugin"

// Plugin wires builtin logger related extension points.
type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) ID() string {
	return "builtin-logger"
}

func (p *Plugin) Name() string {
	return "Builtin Logger"
}

func (p *Plugin) Version() string {
	return "1.0.0"
}

func (p *Plugin) Register(registry plugin.ExtensionRegistry) error {
	registry.RegisterRoute(plugin.RouteExtension{Method: "GET", Path: "/v1/logs"})
	registry.RegisterEventHandler("audit.log.created")
	registry.RegisterMenuItem(plugin.MenuExtension{Name: "日志中心", Path: "/logs"})
	registry.RegisterSettingPage(plugin.SettingExtension{Name: "日志配置", Path: "/settings/logs"})
	return nil
}

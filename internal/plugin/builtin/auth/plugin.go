package auth

import "github.com/tinboxw/skoll/internal/plugin"

// Plugin is a builtin auth plugin example used to verify extension-point wiring.
type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) ID() string {
	return "builtin-auth"
}

func (p *Plugin) Name() string {
	return "Builtin Auth"
}

func (p *Plugin) Version() string {
	return "1.0.0"
}

func (p *Plugin) Register(registry plugin.ExtensionRegistry) error {
	registry.RegisterRoute(plugin.RouteExtension{Method: "POST", Path: "/v1/auth/login"})
	registry.RegisterMiddleware("auth.jwt")
	registry.RegisterEventHandler("user.login")
	registry.RegisterMenuItem(plugin.MenuExtension{Name: "认证管理", Path: "/auth"})
	registry.RegisterWidget(plugin.WidgetExtension{Name: "auth-login-metrics"})
	registry.RegisterSettingPage(plugin.SettingExtension{Name: "认证配置", Path: "/settings/auth"})
	return nil
}

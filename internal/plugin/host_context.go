package plugin

import (
	"strings"
	"time"
)

type HostCapability string

const (
	HostCapabilityAuth         HostCapability = "auth"
	HostCapabilityUser         HostCapability = "user"
	HostCapabilityOrganization HostCapability = "organization"
	HostCapabilityDictionary   HostCapability = "dictionary"
	HostCapabilityFile         HostCapability = "file"
	HostCapabilityAudit        HostCapability = "audit"
	HostCapabilityConfig       HostCapability = "config"
	HostCapabilityPermission   HostCapability = "permission"
)

var defaultHostCapabilities = []HostCapability{
	HostCapabilityAuth,
	HostCapabilityUser,
	HostCapabilityOrganization,
	HostCapabilityDictionary,
	HostCapabilityFile,
	HostCapabilityAudit,
	HostCapabilityConfig,
	HostCapabilityPermission,
}

type HostEndpoint struct {
	Capability HostCapability
	Method     string
	Path       string
}

type PluginContext struct {
	PluginID     string
	PluginName   string
	Version      string
	APIPrefix    string
	Locale       string
	Locales      []string
	Capabilities []HostCapability
	Endpoints    []HostEndpoint
	IssuedAt     time.Time
}

func NewPluginContext(info Info, apiPrefix string, locale string, now time.Time) PluginContext {
	prefix := normalizeHostAPIPrefix(apiPrefix)
	activeLocale := strings.TrimSpace(locale)
	if activeLocale == "" {
		activeLocale = "zh-CN"
	}
	locales := normalizeHostLocales(info.I18nLocales)
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return PluginContext{
		PluginID:     strings.TrimSpace(info.ID),
		PluginName:   strings.TrimSpace(info.Name),
		Version:      strings.TrimSpace(info.Version),
		APIPrefix:    prefix,
		Locale:       activeLocale,
		Locales:      locales,
		Capabilities: append([]HostCapability(nil), defaultHostCapabilities...),
		Endpoints:    defaultHostEndpoints(prefix, strings.TrimSpace(info.ID)),
		IssuedAt:     now,
	}
}

func (c PluginContext) HasCapability(capability HostCapability) bool {
	for _, item := range c.Capabilities {
		if item == capability {
			return true
		}
	}
	return false
}

func normalizeHostAPIPrefix(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "/skoll"
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	return strings.TrimRight(value, "/")
}

func normalizeHostLocales(raw []string) []string {
	if len(raw) == 0 {
		return []string{"zh-CN", "en-US"}
	}
	out := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, item := range raw {
		locale := strings.TrimSpace(item)
		if locale == "" {
			continue
		}
		if _, ok := seen[locale]; ok {
			continue
		}
		seen[locale] = struct{}{}
		out = append(out, locale)
	}
	if len(out) == 0 {
		return []string{"zh-CN", "en-US"}
	}
	return out
}

func defaultHostEndpoints(apiPrefix, pluginID string) []HostEndpoint {
	configPath := "/v1/plugins/{pluginId}/config"
	if pluginID != "" {
		configPath = "/v1/plugins/" + pluginID + "/config"
	}
	endpoints := []HostEndpoint{
		{Capability: HostCapabilityAuth, Method: "GET", Path: "/v1/auth/me"},
		{Capability: HostCapabilityUser, Method: "GET", Path: "/v1/auth/me"},
		{Capability: HostCapabilityOrganization, Method: "GET", Path: "/v1/system/settings/skoll.organization.departments"},
		{Capability: HostCapabilityOrganization, Method: "GET", Path: "/v1/system/settings/skoll.organization.positions"},
		{Capability: HostCapabilityDictionary, Method: "GET", Path: "/v1/system/dictionaries"},
		{Capability: HostCapabilityFile, Method: "GET", Path: "/v1/files"},
		{Capability: HostCapabilityAudit, Method: "GET", Path: "/v1/audit"},
		{Capability: HostCapabilityConfig, Method: "GET", Path: configPath},
		{Capability: HostCapabilityConfig, Method: "PUT", Path: configPath},
		{Capability: HostCapabilityPermission, Method: "GET", Path: "/v1/permissions"},
	}
	for i := range endpoints {
		endpoints[i].Path = apiPrefix + endpoints[i].Path
	}
	return endpoints
}

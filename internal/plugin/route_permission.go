package plugin

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
)

type RoutePermissionErrorCode string

const (
	RoutePermissionMethodInvalid      RoutePermissionErrorCode = "plugin.route_permission.method_invalid"
	RoutePermissionPathInvalid        RoutePermissionErrorCode = "plugin.route_permission.path_invalid"
	RoutePermissionKeyInvalid         RoutePermissionErrorCode = "plugin.route_permission.permission_invalid"
	RoutePermissionAuditInvalid       RoutePermissionErrorCode = "plugin.route_permission.audit_action_invalid"
	RoutePermissionSourceInvalid      RoutePermissionErrorCode = "plugin.route_permission.source_invalid"
	RoutePermissionRouteConflict      RoutePermissionErrorCode = "plugin.route_permission.route_conflict"
	defaultRoutePermissionErrorLocale                          = "zh-CN"
)

// RoutePermissionDescriptor is the normalized authorization contract for one plugin API route.
type RoutePermissionDescriptor struct {
	Method      string
	Path        string
	Permission  string
	AuditAction string
	Source      string
}

type RoutePermissionResolver interface {
	ResolveRoutePermission(method, path string) (RoutePermissionDescriptor, bool)
}

type RoutePermissionError struct {
	Code           RoutePermissionErrorCode
	Method         string
	Path           string
	Source         string
	ConflictSource string
}

func (e *RoutePermissionError) Error() string {
	return e.Message(defaultRoutePermissionErrorLocale)
}

func (e *RoutePermissionError) Message(locale string) string {
	if e == nil {
		return ""
	}
	if strings.EqualFold(strings.TrimSpace(locale), "en-US") {
		return e.messageEN()
	}
	return e.messageZH()
}

func (e *RoutePermissionError) messageZH() string {
	switch e.Code {
	case RoutePermissionMethodInvalid:
		return fmt.Sprintf("插件路由方法无效：%s", e.Method)
	case RoutePermissionPathInvalid:
		return fmt.Sprintf("插件路由路径无效：%s", e.Path)
	case RoutePermissionKeyInvalid:
		return fmt.Sprintf("插件路由权限标识无效：%s %s", e.Method, e.Path)
	case RoutePermissionAuditInvalid:
		return fmt.Sprintf("插件路由审计动作无效：%s %s", e.Method, e.Path)
	case RoutePermissionSourceInvalid:
		return fmt.Sprintf("插件路由来源无效：%s", e.Source)
	case RoutePermissionRouteConflict:
		return fmt.Sprintf("插件路由冲突：%s %s（%s 与 %s）", e.Method, e.Path, e.ConflictSource, e.Source)
	default:
		return "插件路由权限声明无效"
	}
}

func (e *RoutePermissionError) messageEN() string {
	switch e.Code {
	case RoutePermissionMethodInvalid:
		return fmt.Sprintf("invalid plugin route method: %s", e.Method)
	case RoutePermissionPathInvalid:
		return fmt.Sprintf("invalid plugin route path: %s", e.Path)
	case RoutePermissionKeyInvalid:
		return fmt.Sprintf("invalid plugin route permission: %s %s", e.Method, e.Path)
	case RoutePermissionAuditInvalid:
		return fmt.Sprintf("invalid plugin route audit action: %s %s", e.Method, e.Path)
	case RoutePermissionSourceInvalid:
		return fmt.Sprintf("invalid plugin route source: %s", e.Source)
	case RoutePermissionRouteConflict:
		return fmt.Sprintf("conflicting plugin route: %s %s (%s and %s)", e.Method, e.Path, e.ConflictSource, e.Source)
	default:
		return "invalid plugin route permission declaration"
	}
}

type RoutePermissionRegistry struct {
	descriptors []RoutePermissionDescriptor
	byRoute     map[string]RoutePermissionDescriptor
}

func NewRoutePermissionRegistry(routes []RouteExtension) (*RoutePermissionRegistry, error) {
	registry := &RoutePermissionRegistry{
		descriptors: make([]RoutePermissionDescriptor, 0, len(routes)),
		byRoute:     make(map[string]RoutePermissionDescriptor, len(routes)),
	}
	for _, route := range routes {
		descriptor, err := normalizeRoutePermissionDescriptor(route)
		if err != nil {
			return nil, err
		}
		key := routePermissionKey(descriptor.Method, descriptor.Path)
		if current, ok := registry.byRoute[key]; ok {
			return nil, &RoutePermissionError{
				Code:           RoutePermissionRouteConflict,
				Method:         descriptor.Method,
				Path:           descriptor.Path,
				Source:         descriptor.Source,
				ConflictSource: current.Source,
			}
		}
		registry.byRoute[key] = descriptor
		registry.descriptors = append(registry.descriptors, descriptor)
	}
	sort.Slice(registry.descriptors, func(i, j int) bool {
		left := registry.descriptors[i]
		right := registry.descriptors[j]
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		if left.Method != right.Method {
			return left.Method < right.Method
		}
		return left.Source < right.Source
	})
	return registry, nil
}

func (r *RoutePermissionRegistry) ResolveRoutePermission(method, path string) (RoutePermissionDescriptor, bool) {
	if r == nil {
		return RoutePermissionDescriptor{}, false
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	path = normalizeRoutePermissionPath(path)
	descriptor, ok := r.byRoute[routePermissionKey(method, path)]
	return descriptor, ok
}

func (r *RoutePermissionRegistry) Descriptors() []RoutePermissionDescriptor {
	if r == nil {
		return []RoutePermissionDescriptor{}
	}
	return append([]RoutePermissionDescriptor(nil), r.descriptors...)
}

func normalizeRoutePermissionDescriptor(route RouteExtension) (RoutePermissionDescriptor, error) {
	descriptor := RoutePermissionDescriptor{
		Method:      strings.ToUpper(strings.TrimSpace(route.Method)),
		Path:        normalizeRoutePermissionPath(route.Path),
		Permission:  strings.ToLower(strings.TrimSpace(route.Permission)),
		AuditAction: strings.ToLower(strings.TrimSpace(route.AuditAction)),
		Source:      strings.ToLower(strings.TrimSpace(route.Source)),
	}
	if !isSupportedPluginRouteMethod(descriptor.Method) {
		return RoutePermissionDescriptor{}, &RoutePermissionError{Code: RoutePermissionMethodInvalid, Method: descriptor.Method, Path: descriptor.Path, Source: descriptor.Source}
	}
	if !isValidRoutePermissionPath(descriptor.Path) {
		return RoutePermissionDescriptor{}, &RoutePermissionError{Code: RoutePermissionPathInvalid, Method: descriptor.Method, Path: descriptor.Path, Source: descriptor.Source}
	}
	if err := domainpermission.ValidateKey(descriptor.Permission); err != nil {
		return RoutePermissionDescriptor{}, &RoutePermissionError{Code: RoutePermissionKeyInvalid, Method: descriptor.Method, Path: descriptor.Path, Source: descriptor.Source}
	}
	if descriptor.AuditAction != "" && !auditActionPattern.MatchString(descriptor.AuditAction) {
		return RoutePermissionDescriptor{}, &RoutePermissionError{Code: RoutePermissionAuditInvalid, Method: descriptor.Method, Path: descriptor.Path, Source: descriptor.Source}
	}
	if descriptor.Source != "" {
		if err := domainpermission.ValidateSource(descriptor.Source); err != nil {
			return RoutePermissionDescriptor{}, &RoutePermissionError{Code: RoutePermissionSourceInvalid, Method: descriptor.Method, Path: descriptor.Path, Source: descriptor.Source}
		}
	}
	return descriptor, nil
}

func normalizeRoutePermissionPath(path string) string {
	return NormalizeEntryPath(strings.TrimSpace(path))
}

func isValidRoutePermissionPath(path string) bool {
	if path == "" || !strings.HasPrefix(path, "/") || strings.Contains(path, "..") || strings.ContainsAny(path, "?#") {
		return false
	}
	return !strings.ContainsFunc(path, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	})
}

func isSupportedPluginRouteMethod(method string) bool {
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

func routePermissionKey(method, path string) string {
	return method + " " + path
}

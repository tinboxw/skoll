package http

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/tinboxw/skoll/internal/plugin"
)

type pluginOpenAPIOperation struct {
	PluginID    string
	Method      string
	Path        string
	Summary     string
	Permission  string
	AuditAction string
	Source      string
}

func aggregatePluginOpenAPI(document string, manager plugin.Manager) string {
	paths := renderPluginOpenAPIPaths(manager)
	if paths == "" {
		return document
	}
	for _, marker := range []string{"paths:\r\n", "paths:\n"} {
		if strings.Contains(document, marker) {
			return strings.Replace(document, marker, marker+paths, 1)
		}
	}
	return document
}

func renderPluginOpenAPIPaths(manager plugin.Manager) string {
	provider, ok := manager.(pluginExtensionSnapshotProvider)
	if manager == nil || !ok {
		return ""
	}

	items := append([]plugin.Info(nil), manager.List()...)
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].ID) < strings.ToLower(items[j].ID)
	})
	operations := make(map[string]pluginOpenAPIOperation)
	for _, item := range items {
		if item.State != plugin.StateEnabled {
			continue
		}
		snapshot, exists := provider.GetExtensionSnapshot(item.ID)
		if !exists {
			continue
		}
		for _, route := range snapshot.Routes {
			registry, err := plugin.NewRoutePermissionRegistry([]plugin.RouteExtension{route})
			if err != nil {
				continue
			}
			descriptors := registry.Descriptors()
			if len(descriptors) != 1 || !isAllowedPluginRoute(descriptors[0].Method, descriptors[0].Path) {
				continue
			}
			descriptor := descriptors[0]
			key := descriptor.Path + "\x00" + descriptor.Method
			if _, duplicate := operations[key]; duplicate {
				continue
			}
			operations[key] = pluginOpenAPIOperation{
				PluginID:    strings.TrimSpace(item.ID),
				Method:      descriptor.Method,
				Path:        descriptor.Path,
				Summary:     strings.TrimSpace(route.Summary),
				Permission:  descriptor.Permission,
				AuditAction: descriptor.AuditAction,
				Source:      descriptor.Source,
			}
		}
	}

	ordered := make([]pluginOpenAPIOperation, 0, len(operations))
	for _, operation := range operations {
		ordered = append(ordered, operation)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Path != ordered[j].Path {
			return ordered[i].Path < ordered[j].Path
		}
		return ordered[i].Method < ordered[j].Method
	})

	var out strings.Builder
	currentPath := ""
	for _, operation := range ordered {
		if operation.Path != currentPath {
			currentPath = operation.Path
			fmt.Fprintf(&out, "  %s:\n", yamlString(operation.Path))
		}
		summary := operation.Summary
		if summary == "" {
			summary = operation.Method + " " + operation.Path
		}
		fmt.Fprintf(&out, "    %s:\n", strings.ToLower(operation.Method))
		fmt.Fprintf(&out, "      summary: %s\n", yamlString(summary))
		out.WriteString("      tags:\n        - Plugin APIs\n")
		out.WriteString("      security:\n        - bearerAuth: []\n")
		fmt.Fprintf(&out, "      x-skoll-plugin-id: %s\n", yamlString(operation.PluginID))
		fmt.Fprintf(&out, "      x-skoll-plugin-source: %s\n", yamlString(operation.Source))
		fmt.Fprintf(&out, "      x-skoll-permission: %s\n", yamlString(operation.Permission))
		if operation.AuditAction != "" {
			fmt.Fprintf(&out, "      x-skoll-audit-action: %s\n", yamlString(operation.AuditAction))
		}
		out.WriteString("      responses:\n")
		out.WriteString("        '200':\n          description: OK\n")
		out.WriteString("        '401':\n          description: Unauthorized\n")
		out.WriteString("        '403':\n          description: Forbidden\n")
	}
	return out.String()
}

func yamlString(value string) string {
	return strconv.Quote(value)
}

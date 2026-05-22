package plugin

import (
	"context"
	"net/http"
	"sort"
	"strings"

	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	rolesvc "github.com/tinboxw/skoll/internal/service/role"
)

type devPermissionCatalogPluginGroup struct {
	PluginID    string   `json:"pluginId"`
	PluginName  string   `json:"pluginName,omitempty"`
	Permissions []string `json:"permissions"`
}

type devPermissionCatalogResponse struct {
	Operation   string                            `json:"operation"`
	Status      string                            `json:"status"`
	PluginsRoot string                            `json:"pluginsRoot"`
	PluginID    string                            `json:"pluginId"`
	Self        []string                          `json:"self"`
	Framework   []string                          `json:"framework"`
	Plugins     []devPermissionCatalogPluginGroup `json:"plugins"`
	Suggested   []string                          `json:"suggested"`
}

var frameworkPermissionSeeds = []string{
	"plugin.manage",
	"plugin.install",
	"plugin.debug",
	"plugin.config",
	"plugin.marketplace",
	"plugin.marketplace.install",
	"plugin.marketplace.publish",
	"admin.access",
	"permission.manage",
	"role.manage",
	"user.manage",
	"audit.read",
	"metrics.read",
	"profile.manage",
	"plugin.read",
}

func (h *PluginHandler) frameworkPermissionsFromRoles(ctx context.Context) []string {
	if h == nil || h.roleCatalog == nil {
		return sortedUniquePermissions(frameworkPermissionSeeds)
	}

	roles, err := h.roleCatalog.List(ctx, rolesvc.ListInput{Offset: 0, Limit: 1000})
	if err != nil {
		return sortedUniquePermissions(frameworkPermissionSeeds)
	}

	permissions := make([]string, 0, 64)
	for _, item := range roles {
		if item == nil || !item.BuiltIn {
			continue
		}
		permissions = append(permissions, item.Permissions...)
	}

	if len(permissions) == 0 {
		return sortedUniquePermissions(frameworkPermissionSeeds)
	}
	return sortedUniquePermissions(permissions)
}

func (h *PluginHandler) devPermissionCatalog(w http.ResponseWriter, r *http.Request) {
	if !h.devPortalEnabled {
		apiv1.WriteMessage(w, http.StatusNotFound, "not_found", "developer portal is disabled")
		return
	}

	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "super_admin role required")
		return
	}

	pluginsRoot, err := h.resolveDevPluginsRoot(r.URL.Query().Get("pluginsRoot"))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	pluginID := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("pluginId")))
	selfPermissions := []string{}
	if pluginID != "" {
		pluginDir, _, err := h.resolveManifestPath(pluginsRoot, pluginID)
		if err != nil {
			apiv1.WriteError(w, http.StatusBadRequest, err)
			return
		}

		selfInfo, err := h.loader.Load(pluginDir)
		if err != nil {
			apiv1.WriteError(w, http.StatusBadRequest, err)
			return
		}

		selfPermissions = sortedUniquePermissions(selfInfo.Permissions)
		if len(selfPermissions) == 0 {
			selfPermissions = []string{pluginID + ".read"}
		}
	}

	pluginDirs, err := discoverPluginManifestDirs(pluginsRoot)
	if err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	otherGroups := make([]devPermissionCatalogPluginGroup, 0, len(pluginDirs))
	for _, dir := range pluginDirs {
		info, loadErr := h.loader.Load(dir)
		if loadErr != nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(info.ID), pluginID) {
			continue
		}

		perms := sortedUniquePermissions(info.Permissions)
		if len(perms) == 0 {
			continue
		}

		otherGroups = append(otherGroups, devPermissionCatalogPluginGroup{
			PluginID:    strings.TrimSpace(info.ID),
			PluginName:  strings.TrimSpace(info.Name),
			Permissions: perms,
		})
	}

	sort.Slice(otherGroups, func(i, j int) bool {
		left := strings.TrimSpace(otherGroups[i].PluginID)
		right := strings.TrimSpace(otherGroups[j].PluginID)
		if left == right {
			return strings.TrimSpace(otherGroups[i].PluginName) < strings.TrimSpace(otherGroups[j].PluginName)
		}
		return left < right
	})

	frameworkPermissions := h.frameworkPermissionsFromRoles(r.Context())
	suggested := mergePermissionSets(selfPermissions, frameworkPermissions)
	for _, group := range otherGroups {
		suggested = mergePermissionSets(suggested, group.Permissions)
	}

	h.appendAudit(r, "dev_permission_catalog", "plugin", pluginID, map[string]any{
		"pluginsRoot":  pluginsRoot,
		"self":         len(selfPermissions),
		"framework":    len(frameworkPermissions),
		"otherPlugins": len(otherGroups),
	})

	apiv1.WriteJSON(w, http.StatusOK, devPermissionCatalogResponse{
		Operation:   "permission_catalog",
		Status:      "ok",
		PluginsRoot: pluginsRoot,
		PluginID:    pluginID,
		Self:        selfPermissions,
		Framework:   frameworkPermissions,
		Plugins:     otherGroups,
		Suggested:   suggested,
	})
}

func sortedUniquePermissions(input []string) []string {
	if len(input) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(input))
	result := make([]string, 0, len(input))
	for _, item := range input {
		permission := strings.TrimSpace(item)
		if permission == "" {
			continue
		}
		if _, ok := seen[permission]; ok {
			continue
		}
		seen[permission] = struct{}{}
		result = append(result, permission)
	}
	sort.Strings(result)
	return result
}

func mergePermissionSets(base []string, addition []string) []string {
	if len(addition) == 0 {
		return sortedUniquePermissions(base)
	}
	merged := make([]string, 0, len(base)+len(addition))
	merged = append(merged, base...)
	merged = append(merged, addition...)
	return sortedUniquePermissions(merged)
}

package bootstrap

import (
	"context"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	permissionsvc "github.com/tinboxw/skoll/internal/service/permission"
	"github.com/tinboxw/skoll/pkg/logging"
)

var systemPermissionCatalogSeeds = []permissionsvc.RegisterResourceInput{
	{
		Key:    "permission.manage",
		Type:   domainpermission.ResourceTypeAPI,
		Module: "permission",
		Source: "system",
		Name:   "Manage permission catalog",
		Risk:   domainpermission.RiskLevelHigh,
		Metadata: map[string]string{
			"routes": "GET /v1/permissions;GET /v1/permissions/{key};POST /v1/permissions/{key}/enable;POST /v1/permissions/{key}/disable;POST /v1/permissions/diff",
		},
	},
	{
		Key:    "menu.read",
		Type:   domainpermission.ResourceTypeAPI,
		Module: "menu",
		Source: "system",
		Name:   "Read menu registry",
		Risk:   domainpermission.RiskLevelLow,
		Metadata: map[string]string{
			"routes": "GET /v1/menus/tree",
		},
	},
	{
		Key:    "menu.manage",
		Type:   domainpermission.ResourceTypeAPI,
		Module: "menu",
		Source: "system",
		Name:   "Manage menu registry",
		Risk:   domainpermission.RiskLevelMedium,
		Metadata: map[string]string{
			"routes": "PUT /v1/menus;POST /v1/menus/reorder;PATCH /v1/menus/visibility",
		},
	},
}

func ensureSystemPermissionCatalog(ctx context.Context, logger logging.Logger, service permissionsvc.Service) {
	if service == nil {
		return
	}
	for _, seed := range systemPermissionCatalogSeeds {
		if _, err := service.RegisterResource(ctx, seed); err != nil {
			if logger != nil {
				logger.Warn("seed permission catalog failed", "key", seed.Key, "error", err)
			}
		}
	}
}

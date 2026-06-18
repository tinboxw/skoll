package permission

import (
	"context"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
)

type ListFilter struct {
	Type    domainpermission.ResourceType
	Module  string
	Source  string
	Enabled *bool
}

type PermissionRepository interface {
	Register(ctx context.Context, resource domainpermission.PermissionResource) error
	Get(ctx context.Context, key string) (*domainpermission.PermissionResource, error)
	List(ctx context.Context, filter ListFilter, offset, limit int) ([]domainpermission.PermissionResource, error)
	SetEnabled(ctx context.Context, key string, enabled bool) error
}

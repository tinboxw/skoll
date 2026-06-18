package permission

import (
	"context"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
)

type Service interface {
	RegisterResource(ctx context.Context, in RegisterResourceInput) (*domainpermission.PermissionResource, error)
	ListResources(ctx context.Context, in ListResourcesInput) ([]domainpermission.PermissionResource, error)
	GetResource(ctx context.Context, key string) (*domainpermission.PermissionResource, error)
	EnableResource(ctx context.Context, key string) error
	DisableResource(ctx context.Context, key string) error
	DiffResources(ctx context.Context, in DiffResourcesInput) (*DiffResult, error)
}

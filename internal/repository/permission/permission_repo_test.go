package permission

import (
	"context"
	"testing"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
)

func TestPermissionRepositoryInterfaceShape(t *testing.T) {
	var _ PermissionRepository = (*stubPermissionRepository)(nil)
}

type stubPermissionRepository struct{}

func (s *stubPermissionRepository) Register(context.Context, domainpermission.PermissionResource) error {
	return nil
}

func (s *stubPermissionRepository) Get(context.Context, string) (*domainpermission.PermissionResource, error) {
	return nil, nil
}

func (s *stubPermissionRepository) List(context.Context, ListFilter, int, int) ([]domainpermission.PermissionResource, error) {
	return nil, nil
}

func (s *stubPermissionRepository) SetEnabled(context.Context, string, bool) error {
	return nil
}

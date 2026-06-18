package permission

import (
	"context"
	"testing"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
)

func TestServiceInterfaceShape(t *testing.T) {
	var _ Service = (*stubService)(nil)
}

type stubService struct{}

func (s *stubService) RegisterResource(context.Context, RegisterResourceInput) (*domainpermission.PermissionResource, error) {
	return nil, nil
}

func (s *stubService) ListResources(context.Context, ListResourcesInput) ([]domainpermission.PermissionResource, error) {
	return nil, nil
}

func (s *stubService) GetResource(context.Context, string) (*domainpermission.PermissionResource, error) {
	return nil, nil
}

func (s *stubService) EnableResource(context.Context, string) error {
	return nil
}

func (s *stubService) DisableResource(context.Context, string) error {
	return nil
}

func (s *stubService) DiffResources(context.Context, DiffResourcesInput) (*DiffResult, error) {
	return nil, nil
}

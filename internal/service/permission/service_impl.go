package permission

import (
	"context"
	"fmt"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	permissionrepo "github.com/tinboxw/skoll/internal/repository/permission"
)

type serviceImpl struct {
	repo permissionrepo.PermissionRepository
}

func NewService(repo permissionrepo.PermissionRepository) Service {
	return &serviceImpl{repo: repo}
}

func (s *serviceImpl) RegisterResource(ctx context.Context, in RegisterResourceInput) (*domainpermission.PermissionResource, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("permission repository is not configured")
	}
	risk := in.Risk
	if risk == "" {
		risk = domainpermission.RiskLevelLow
	}
	resource, err := domainpermission.NewResourceWithMetadata(domainpermission.ResourceIdentity{
		Key:    in.Key,
		Type:   in.Type,
		Module: in.Module,
		Source: in.Source,
	}, in.Name, risk, in.Metadata)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Register(ctx, resource); err != nil {
		return nil, err
	}
	return &resource, nil
}

func (s *serviceImpl) ListResources(context.Context, ListResourcesInput) ([]domainpermission.PermissionResource, error) {
	return nil, fmt.Errorf("ListResources is not implemented")
}

func (s *serviceImpl) GetResource(context.Context, string) (*domainpermission.PermissionResource, error) {
	return nil, fmt.Errorf("GetResource is not implemented")
}

func (s *serviceImpl) EnableResource(context.Context, string) error {
	return fmt.Errorf("EnableResource is not implemented")
}

func (s *serviceImpl) DisableResource(context.Context, string) error {
	return fmt.Errorf("DisableResource is not implemented")
}

func (s *serviceImpl) DiffResources(context.Context, DiffResourcesInput) (*DiffResult, error) {
	return nil, fmt.Errorf("DiffResources is not implemented")
}

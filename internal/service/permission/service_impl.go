package permission

import (
	"context"
	"fmt"
	"strings"

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

func (s *serviceImpl) ListResources(ctx context.Context, in ListResourcesInput) ([]domainpermission.PermissionResource, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("permission repository is not configured")
	}
	if in.Offset < 0 || in.Limit < 0 {
		return nil, fmt.Errorf("invalid pagination")
	}
	return s.repo.List(ctx, permissionrepo.ListFilter{
		Type:    in.Type,
		Module:  in.Module,
		Source:  in.Source,
		Enabled: in.Enabled,
	}, in.Offset, in.Limit)
}

func (s *serviceImpl) GetResource(ctx context.Context, key string) (*domainpermission.PermissionResource, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("permission repository is not configured")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("permission key is required")
	}
	return s.repo.Get(ctx, key)
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

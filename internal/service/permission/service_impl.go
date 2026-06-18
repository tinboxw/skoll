package permission

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	permissionrepo "github.com/tinboxw/skoll/internal/repository/permission"
)

type serviceImpl struct {
	repo    permissionrepo.PermissionRepository
	auditFn func(ctx context.Context, key string, enabled bool) error
}

func NewService(repo permissionrepo.PermissionRepository) Service {
	return &serviceImpl{repo: repo}
}

func (s *serviceImpl) RegisterResource(ctx context.Context, in RegisterResourceInput) (*domainpermission.PermissionResource, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("permission repository is not configured")
	}
	resource, err := buildPermissionResource(in)
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

func (s *serviceImpl) EnableResource(ctx context.Context, key string) error {
	return s.setResourceEnabled(ctx, key, true)
}

func (s *serviceImpl) DisableResource(ctx context.Context, key string) error {
	return s.setResourceEnabled(ctx, key, false)
}

func (s *serviceImpl) DiffResources(ctx context.Context, in DiffResourcesInput) (*DiffResult, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("permission repository is not configured")
	}
	source := strings.TrimSpace(strings.ToLower(in.Source))
	if source == "" {
		return nil, fmt.Errorf("permission source is required")
	}

	current, err := s.repo.List(ctx, permissionrepo.ListFilter{Source: source}, 0, 0)
	if err != nil {
		return nil, err
	}
	currentByKey := make(map[string]domainpermission.PermissionResource, len(current))
	for _, item := range current {
		currentByKey[item.Key()] = item
	}

	result := &DiffResult{
		Added:   make([]domainpermission.PermissionResource, 0),
		Updated: make([]domainpermission.PermissionResource, 0),
		Removed: make([]domainpermission.PermissionResource, 0),
	}
	desiredKeys := make(map[string]struct{}, len(in.Desired))
	for _, desiredInput := range in.Desired {
		if strings.TrimSpace(desiredInput.Source) == "" {
			desiredInput.Source = source
		}
		resource, err := buildPermissionResource(desiredInput)
		if err != nil {
			return nil, err
		}
		if resource.Source() != source {
			return nil, fmt.Errorf("permission source mismatch")
		}
		desiredKeys[resource.Key()] = struct{}{}
		existing, ok := currentByKey[resource.Key()]
		if !ok {
			result.Added = append(result.Added, resource)
			continue
		}
		if permissionResourceChanged(existing, resource) {
			result.Updated = append(result.Updated, resource)
		}
	}

	removedKeys := make([]string, 0)
	for key := range currentByKey {
		if _, ok := desiredKeys[key]; !ok {
			removedKeys = append(removedKeys, key)
		}
	}
	sort.Strings(removedKeys)
	for _, key := range removedKeys {
		result.Removed = append(result.Removed, currentByKey[key])
	}
	return result, nil
}

func (s *serviceImpl) setResourceEnabled(ctx context.Context, key string, enabled bool) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("permission repository is not configured")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("permission key is required")
	}
	if err := s.repo.SetEnabled(ctx, key, enabled); err != nil {
		return err
	}
	if s.auditFn != nil {
		return s.auditFn(ctx, key, enabled)
	}
	return nil
}

func buildPermissionResource(in RegisterResourceInput) (domainpermission.PermissionResource, error) {
	risk := in.Risk
	if risk == "" {
		risk = domainpermission.RiskLevelLow
	}
	return domainpermission.NewResourceWithMetadata(domainpermission.ResourceIdentity{
		Key:    in.Key,
		Type:   in.Type,
		Module: in.Module,
		Source: in.Source,
	}, in.Name, risk, in.Metadata)
}

func permissionResourceChanged(a domainpermission.PermissionResource, b domainpermission.PermissionResource) bool {
	return a.Type() != b.Type() ||
		a.Module() != b.Module() ||
		a.Source() != b.Source() ||
		a.Name != b.Name ||
		a.Risk != b.Risk ||
		!reflect.DeepEqual(domainpermission.NormalizeMetadata(a.Metadata), domainpermission.NormalizeMetadata(b.Metadata))
}

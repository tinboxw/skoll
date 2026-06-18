package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	permissionrepo "github.com/tinboxw/skoll/internal/repository/permission"
)

type PermissionStore struct {
	mu    sync.RWMutex
	items map[string]domainpermission.PermissionResource
}

func NewPermissionStore() *PermissionStore {
	return &PermissionStore{items: make(map[string]domainpermission.PermissionResource)}
}

func (s *PermissionStore) Register(_ context.Context, resource domainpermission.PermissionResource) error {
	key := normalizePermissionKey(resource.Key())
	if key == "" {
		return nil
	}
	resource = clonePermissionResource(resource)
	resource.Identity.Key = key

	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = resource
	return nil
}

func (s *PermissionStore) Get(_ context.Context, key string) (*domainpermission.PermissionResource, error) {
	key = normalizePermissionKey(key)
	if key == "" {
		return nil, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[key]
	if !ok {
		return nil, nil
	}
	item = clonePermissionResource(item)
	return &item, nil
}

func (s *PermissionStore) List(_ context.Context, filter permissionrepo.ListFilter, offset, limit int) ([]domainpermission.PermissionResource, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.items))
	for key, item := range s.items {
		if matchesPermissionFilter(item, filter) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = len(keys)
	}
	if offset > len(keys) {
		return []domainpermission.PermissionResource{}, nil
	}
	end := offset + limit
	if end > len(keys) {
		end = len(keys)
	}

	out := make([]domainpermission.PermissionResource, 0, end-offset)
	for _, key := range keys[offset:end] {
		out = append(out, clonePermissionResource(s.items[key]))
	}
	return out, nil
}

func (s *PermissionStore) SetEnabled(_ context.Context, key string, enabled bool) error {
	key = normalizePermissionKey(key)
	if key == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[key]
	if !ok {
		return nil
	}
	item.Enabled = enabled
	s.items[key] = item
	return nil
}

func matchesPermissionFilter(item domainpermission.PermissionResource, filter permissionrepo.ListFilter) bool {
	if filter.Type != "" && item.Type() != filter.Type {
		return false
	}
	if strings.TrimSpace(filter.Module) != "" && item.Module() != strings.TrimSpace(strings.ToLower(filter.Module)) {
		return false
	}
	if strings.TrimSpace(filter.Source) != "" && item.Source() != strings.TrimSpace(strings.ToLower(filter.Source)) {
		return false
	}
	if filter.Enabled != nil && item.Enabled != *filter.Enabled {
		return false
	}
	return true
}

func clonePermissionResource(resource domainpermission.PermissionResource) domainpermission.PermissionResource {
	resource.Metadata = domainpermission.NormalizeMetadata(resource.Metadata)
	return resource
}

func normalizePermissionKey(key string) string {
	return strings.TrimSpace(strings.ToLower(key))
}

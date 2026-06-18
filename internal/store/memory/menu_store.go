package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	domainmenu "github.com/tinboxw/skoll/internal/domain/menu"
	menurepo "github.com/tinboxw/skoll/internal/repository/menu"
)

type MenuStore struct {
	mu    sync.RWMutex
	items map[string]domainmenu.MenuNode
}

func NewMenuStore() *MenuStore {
	return &MenuStore{items: make(map[string]domainmenu.MenuNode)}
}

func (s *MenuStore) Tree(_ context.Context, filter menurepo.ListFilter) ([]domainmenu.MenuNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	candidates := make([]domainmenu.MenuNode, 0, len(s.items))
	for _, item := range s.items {
		if matchesMenuFilter(item, filter, false) {
			candidates = append(candidates, cloneMenuNode(item))
		}
	}
	return flattenMenuTree(candidates, normalizeMenuKey(filter.ParentKey)), nil
}

func (s *MenuStore) List(_ context.Context, filter menurepo.ListFilter, offset, limit int) ([]domainmenu.MenuNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]domainmenu.MenuNode, 0, len(s.items))
	for _, item := range s.items {
		if matchesMenuFilter(item, filter, true) {
			out = append(out, cloneMenuNode(item))
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ParentKey() == out[j].ParentKey() && out[i].Sort != out[j].Sort {
			return out[i].Sort < out[j].Sort
		}
		return out[i].Key() < out[j].Key()
	})

	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = len(out)
	}
	if offset > len(out) {
		return []domainmenu.MenuNode{}, nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return append([]domainmenu.MenuNode(nil), out[offset:end]...), nil
}

func (s *MenuStore) Upsert(_ context.Context, node domainmenu.MenuNode) error {
	key := normalizeMenuKey(node.Key())
	if key == "" {
		return nil
	}
	node = cloneMenuNode(node)
	node.Identity.Key = key
	node.Identity.ParentKey = normalizeMenuKey(node.ParentKey())
	node.Identity.Source = strings.TrimSpace(strings.ToLower(node.Source()))

	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = node
	return nil
}

func (s *MenuStore) Reorder(_ context.Context, parentKey string, orderedKeys []string) error {
	parentKey = normalizeMenuKey(parentKey)

	s.mu.Lock()
	defer s.mu.Unlock()
	for i, key := range orderedKeys {
		key = normalizeMenuKey(key)
		item, ok := s.items[key]
		if !ok || item.ParentKey() != parentKey {
			continue
		}
		item.Sort = i * 10
		s.items[key] = item
	}
	return nil
}

func matchesMenuFilter(item domainmenu.MenuNode, filter menurepo.ListFilter, matchParent bool) bool {
	if matchParent && strings.TrimSpace(filter.ParentKey) != "" && item.ParentKey() != normalizeMenuKey(filter.ParentKey) {
		return false
	}
	if strings.TrimSpace(filter.Source) != "" && item.Source() != strings.TrimSpace(strings.ToLower(filter.Source)) {
		return false
	}
	if filter.Visible != nil && item.Visible != *filter.Visible {
		return false
	}
	return true
}

func flattenMenuTree(nodes []domainmenu.MenuNode, rootParent string) []domainmenu.MenuNode {
	byParent := make(map[string][]domainmenu.MenuNode)
	for _, node := range nodes {
		byParent[node.ParentKey()] = append(byParent[node.ParentKey()], node)
	}

	out := make([]domainmenu.MenuNode, 0, len(nodes))
	var walk func(parent string)
	walk = func(parent string) {
		for _, node := range domainmenu.SortSiblings(byParent[parent]) {
			out = append(out, cloneMenuNode(node))
			walk(node.Key())
		}
	}
	walk(rootParent)
	return out
}

func cloneMenuNode(node domainmenu.MenuNode) domainmenu.MenuNode {
	node.RequiredRoles = append([]string(nil), node.RequiredRoles...)
	node.RequiredPermissions = append([]string(nil), node.RequiredPermissions...)
	return node
}

func normalizeMenuKey(key string) string {
	return strings.TrimSpace(strings.ToLower(key))
}

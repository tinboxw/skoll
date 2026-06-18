package menu

import (
	"context"
	"fmt"
	"strings"

	domainmenu "github.com/tinboxw/skoll/internal/domain/menu"
	menurepo "github.com/tinboxw/skoll/internal/repository/menu"
)

type serviceImpl struct {
	repo menurepo.MenuRepository
}

func NewService(repo menurepo.MenuRepository) Service {
	return &serviceImpl{repo: repo}
}

func (s *serviceImpl) MergeNodes(ctx context.Context, in MergeNodesInput) ([]domainmenu.MenuNode, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("menu repository is not configured")
	}
	merged := stableMergeNodes(in.Nodes)
	for _, node := range merged {
		if err := s.repo.Upsert(ctx, node); err != nil {
			return nil, err
		}
	}
	return flattenMenuNodes(merged, ""), nil
}

func (s *serviceImpl) Tree(ctx context.Context, in TreeInput) ([]domainmenu.MenuNode, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("menu repository is not configured")
	}
	return s.repo.Tree(ctx, menurepo.ListFilter{
		ParentKey: in.ParentKey,
		Source:    in.Source,
		Visible:   in.Visible,
	})
}

func (s *serviceImpl) Filter(context.Context, FilterInput) ([]domainmenu.MenuNode, error) {
	return nil, fmt.Errorf("Filter is not implemented")
}

func (s *serviceImpl) Reorder(ctx context.Context, in ReorderInput) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("menu repository is not configured")
	}
	return s.repo.Reorder(ctx, in.ParentKey, in.OrderedKeys)
}

func stableMergeNodes(nodes []domainmenu.MenuNode) []domainmenu.MenuNode {
	positions := make([]string, 0, len(nodes))
	byKey := make(map[string]domainmenu.MenuNode, len(nodes))
	for _, node := range nodes {
		key := strings.TrimSpace(strings.ToLower(node.Key()))
		if key == "" {
			continue
		}
		node.Identity.Key = key
		node.Identity.ParentKey = strings.TrimSpace(strings.ToLower(node.ParentKey()))
		if _, ok := byKey[key]; !ok {
			positions = append(positions, key)
		}
		byKey[key] = node
	}
	out := make([]domainmenu.MenuNode, 0, len(positions))
	for _, key := range positions {
		out = append(out, byKey[key])
	}
	return out
}

func flattenMenuNodes(nodes []domainmenu.MenuNode, rootParent string) []domainmenu.MenuNode {
	byParent := make(map[string][]domainmenu.MenuNode)
	for _, node := range nodes {
		byParent[node.ParentKey()] = append(byParent[node.ParentKey()], node)
	}
	out := make([]domainmenu.MenuNode, 0, len(nodes))
	var walk func(parent string)
	walk = func(parent string) {
		for _, node := range domainmenu.SortSiblings(byParent[parent]) {
			out = append(out, node)
			walk(node.Key())
		}
	}
	walk(strings.TrimSpace(strings.ToLower(rootParent)))
	return out
}

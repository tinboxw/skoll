package menu

import (
	"context"
	"testing"

	domainmenu "github.com/tinboxw/skoll/internal/domain/menu"
)

func TestServiceInterfaceShape(t *testing.T) {
	var _ Service = (*stubService)(nil)
}

type stubService struct{}

func (s *stubService) MergeNodes(context.Context, MergeNodesInput) ([]domainmenu.MenuNode, error) {
	return nil, nil
}

func (s *stubService) Tree(context.Context, TreeInput) ([]domainmenu.MenuNode, error) {
	return nil, nil
}

func (s *stubService) Filter(context.Context, FilterInput) ([]domainmenu.MenuNode, error) {
	return nil, nil
}

func (s *stubService) Reorder(context.Context, ReorderInput) error {
	return nil
}

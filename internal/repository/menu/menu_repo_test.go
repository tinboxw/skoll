package menu

import (
	"context"
	"testing"

	domainmenu "github.com/tinboxw/skoll/internal/domain/menu"
)

func TestMenuRepositoryInterfaceShape(t *testing.T) {
	var _ MenuRepository = (*stubMenuRepository)(nil)
}

type stubMenuRepository struct{}

func (s *stubMenuRepository) Tree(context.Context, ListFilter) ([]domainmenu.MenuNode, error) {
	return nil, nil
}

func (s *stubMenuRepository) List(context.Context, ListFilter, int, int) ([]domainmenu.MenuNode, error) {
	return nil, nil
}

func (s *stubMenuRepository) Upsert(context.Context, domainmenu.MenuNode) error {
	return nil
}

func (s *stubMenuRepository) Reorder(context.Context, string, []string) error {
	return nil
}

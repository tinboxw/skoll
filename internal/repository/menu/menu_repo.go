package menu

import (
	"context"

	domainmenu "github.com/tinboxw/skoll/internal/domain/menu"
)

type ListFilter struct {
	ParentKey string
	Source    string
	Visible   *bool
}

type MenuRepository interface {
	Tree(ctx context.Context, filter ListFilter) ([]domainmenu.MenuNode, error)
	List(ctx context.Context, filter ListFilter, offset, limit int) ([]domainmenu.MenuNode, error)
	Upsert(ctx context.Context, node domainmenu.MenuNode) error
	Reorder(ctx context.Context, parentKey string, orderedKeys []string) error
}

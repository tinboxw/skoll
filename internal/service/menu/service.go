package menu

import (
	"context"

	domainmenu "github.com/tinboxw/skoll/internal/domain/menu"
)

type Service interface {
	MergeNodes(ctx context.Context, in MergeNodesInput) ([]domainmenu.MenuNode, error)
	Tree(ctx context.Context, in TreeInput) ([]domainmenu.MenuNode, error)
	Filter(ctx context.Context, in FilterInput) ([]domainmenu.MenuNode, error)
	Reorder(ctx context.Context, in ReorderInput) error
}

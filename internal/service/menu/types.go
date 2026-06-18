package menu

import domainmenu "github.com/tinboxw/skoll/internal/domain/menu"

type MergeNodesInput struct {
	Nodes []domainmenu.MenuNode
}

type TreeInput struct {
	ParentKey string
	Source    string
	Visible   *bool
}

type FilterInput struct {
	Nodes       []domainmenu.MenuNode
	Roles       []string
	Permissions []string
	VisibleOnly bool
}

type ReorderInput struct {
	ParentKey   string
	OrderedKeys []string
}

package menu

type NodeIdentity struct {
	Key       string
	ParentKey string
	Source    string
}

type NodeView struct {
	Name      string
	Path      string
	Component string
	Icon      string
}

type MenuNode struct {
	Identity            NodeIdentity
	View                NodeView
	Sort                int
	Visible             bool
	RequiredRoles       []string
	RequiredPermissions []string
}

func NewNode(identity NodeIdentity, view NodeView, sort int) MenuNode {
	return MenuNode{
		Identity: identity,
		View:     view,
		Sort:     sort,
		Visible:  true,
	}
}

func (n MenuNode) Key() string {
	return n.Identity.Key
}

func (n MenuNode) ParentKey() string {
	return n.Identity.ParentKey
}

func (n MenuNode) Source() string {
	return n.Identity.Source
}

func (n MenuNode) Name() string {
	return n.View.Name
}

func (n MenuNode) Path() string {
	return n.View.Path
}

func (n MenuNode) Component() string {
	return n.View.Component
}

func (n MenuNode) Icon() string {
	return n.View.Icon
}

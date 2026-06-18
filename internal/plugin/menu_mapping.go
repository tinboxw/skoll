package plugin

import (
	"fmt"
	"strings"

	domainmenu "github.com/tinboxw/skoll/internal/domain/menu"
)

func (i Info) MenuNodes() ([]domainmenu.MenuNode, error) {
	if i.UIMenu == nil {
		return []domainmenu.MenuNode{}, nil
	}

	sourceID := strings.TrimSpace(strings.ToLower(i.ID))
	key := strings.TrimSpace(strings.ToLower(i.UIMenu.Key))
	if key == "" {
		key = "plugin." + sourceID
	}
	name := strings.TrimSpace(i.UIMenu.Label)
	if name == "" {
		name = strings.TrimSpace(i.UIMenu.LabelZhCN)
	}
	if name == "" {
		name = strings.TrimSpace(i.UIMenu.LabelEnUS)
	}
	if name == "" {
		name = strings.TrimSpace(i.Name)
	}
	path := NormalizeEntryPath(i.UIMenu.Path)
	if path == "" {
		path = ResolveFrontendEntry(i)
	}
	if sourceID == "" || path == "" {
		return nil, ErrPluginManifestBroken
	}

	node, err := domainmenu.NewNode(domainmenu.NodeIdentity{
		Key:       key,
		ParentKey: strings.TrimSpace(strings.ToLower(i.UIMenu.ParentKey)),
		Source:    fmt.Sprintf("plugin.%s", sourceID),
	}, domainmenu.NodeView{
		Name:      name,
		Path:      path,
		Component: strings.TrimSpace(i.UIMenu.Component),
		Icon:      strings.TrimSpace(i.UIMenu.Icon),
	}, i.UIMenu.Order)
	if err != nil {
		return nil, err
	}
	if i.UIMenu.Visible != nil {
		node.Visible = *i.UIMenu.Visible
	}
	node.RequiredRoles = normalizeStringList(i.UIMenu.RequiredRoles)
	node.RequiredPermissions = normalizeStringList(i.UIMenu.RequiredPermissions)
	return []domainmenu.MenuNode{node}, nil
}

func normalizeStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

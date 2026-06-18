package gormrepo

import (
	"context"
	"strings"

	domainmenu "github.com/tinboxw/skoll/internal/domain/menu"
	menurepo "github.com/tinboxw/skoll/internal/repository/menu"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MenuStore struct {
	db *gorm.DB
}

func NewMenuStore(db *gorm.DB) *MenuStore {
	return &MenuStore{db: db}
}

func (s *MenuStore) Tree(ctx context.Context, filter menurepo.ListFilter) ([]domainmenu.MenuNode, error) {
	nodes, err := s.list(ctx, filter, false, 0, 0)
	if err != nil {
		return nil, err
	}
	return flattenSQLMenuTree(nodes, normalizeSQLMenuKey(filter.ParentKey)), nil
}

func (s *MenuStore) List(ctx context.Context, filter menurepo.ListFilter, offset, limit int) ([]domainmenu.MenuNode, error) {
	return s.list(ctx, filter, true, offset, limit)
}

func (s *MenuStore) Upsert(ctx context.Context, node domainmenu.MenuNode) error {
	row := MenuNodeModelFromDomain(node)
	if strings.TrimSpace(row.MenuKey) == "" {
		return nil
	}
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "menu_key"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"parent_key",
				"source",
				"name",
				"path",
				"component",
				"icon",
				"sort",
				"visible",
				"required_roles_json",
				"required_permissions_json",
				"updated_at",
			}),
		}).Create(&row).Error
	})
}

func (s *MenuStore) Reorder(ctx context.Context, parentKey string, orderedKeys []string) error {
	parentKey = normalizeSQLMenuKey(parentKey)
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			for i, key := range orderedKeys {
				key = normalizeSQLMenuKey(key)
				if key == "" {
					continue
				}
				if err := tx.Model(&MenuNodeModel{}).
					Where("menu_key = ? AND parent_key = ?", key, parentKey).
					Update("sort", i*10).Error; err != nil {
					return err
				}
			}
			return nil
		})
	})
}

func (s *MenuStore) list(ctx context.Context, filter menurepo.ListFilter, matchParent bool, offset, limit int) ([]domainmenu.MenuNode, error) {
	q := s.db.WithContext(ctx).Model(&MenuNodeModel{}).Order("parent_key asc, sort asc, menu_key asc")
	if matchParent && strings.TrimSpace(filter.ParentKey) != "" {
		q = q.Where("parent_key = ?", normalizeSQLMenuKey(filter.ParentKey))
	}
	if strings.TrimSpace(filter.Source) != "" {
		q = q.Where("source = ?", strings.TrimSpace(strings.ToLower(filter.Source)))
	}
	if filter.Visible != nil {
		q = q.Where("visible = ?", *filter.Visible)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}

	var rows []MenuNodeModel
	if err := withDBRetry(func() error { return q.Find(&rows).Error }); err != nil {
		return nil, err
	}
	out := make([]domainmenu.MenuNode, 0, len(rows))
	for _, row := range rows {
		node, err := row.ToDomain()
		if err != nil {
			return nil, err
		}
		out = append(out, node)
	}
	return out, nil
}

func flattenSQLMenuTree(nodes []domainmenu.MenuNode, rootParent string) []domainmenu.MenuNode {
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
	walk(rootParent)
	return out
}

func normalizeSQLMenuKey(key string) string {
	return strings.TrimSpace(strings.ToLower(key))
}

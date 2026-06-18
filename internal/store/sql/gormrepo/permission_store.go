package gormrepo

import (
	"context"
	"strings"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	permissionrepo "github.com/tinboxw/skoll/internal/repository/permission"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PermissionStore struct {
	db *gorm.DB
}

func NewPermissionStore(db *gorm.DB) *PermissionStore {
	return &PermissionStore{db: db}
}

func (s *PermissionStore) Register(ctx context.Context, resource domainpermission.PermissionResource) error {
	row := PermissionResourceModelFromDomain(resource)
	if strings.TrimSpace(row.PermissionKey) == "" {
		return nil
	}
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "permission_key"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"resource_type",
				"module",
				"source",
				"name",
				"risk",
				"metadata_json",
				"enabled",
				"updated_at",
			}),
		}).Create(&row).Error
	})
}

func (s *PermissionStore) Get(ctx context.Context, key string) (*domainpermission.PermissionResource, error) {
	key = strings.TrimSpace(strings.ToLower(key))
	if key == "" {
		return nil, nil
	}

	var row PermissionResourceModel
	err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Where("permission_key = ?", key).First(&row).Error
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	resource, err := row.ToDomain()
	if err != nil {
		return nil, err
	}
	return &resource, nil
}

func (s *PermissionStore) List(ctx context.Context, filter permissionrepo.ListFilter, offset, limit int) ([]domainpermission.PermissionResource, error) {
	q := s.db.WithContext(ctx).Model(&PermissionResourceModel{}).Order("permission_key asc")
	if filter.Type != "" {
		q = q.Where("resource_type = ?", string(filter.Type))
	}
	if strings.TrimSpace(filter.Module) != "" {
		q = q.Where("module = ?", strings.TrimSpace(strings.ToLower(filter.Module)))
	}
	if strings.TrimSpace(filter.Source) != "" {
		q = q.Where("source = ?", strings.TrimSpace(strings.ToLower(filter.Source)))
	}
	if filter.Enabled != nil {
		q = q.Where("enabled = ?", *filter.Enabled)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}

	var rows []PermissionResourceModel
	if err := withDBRetry(func() error { return q.Find(&rows).Error }); err != nil {
		return nil, err
	}
	out := make([]domainpermission.PermissionResource, 0, len(rows))
	for _, row := range rows {
		resource, err := row.ToDomain()
		if err != nil {
			return nil, err
		}
		out = append(out, resource)
	}
	return out, nil
}

func (s *PermissionStore) SetEnabled(ctx context.Context, key string, enabled bool) error {
	key = strings.TrimSpace(strings.ToLower(key))
	if key == "" {
		return nil
	}
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).
			Model(&PermissionResourceModel{}).
			Where("permission_key = ?", key).
			Update("enabled", enabled).Error
	})
}

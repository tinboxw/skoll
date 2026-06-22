package gormrepo

import (
	"context"
	"fmt"
	"strings"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainsystem "github.com/tinboxw/skoll/internal/domain/system"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *SystemStore) GetDictionaryTypeByID(ctx context.Context, id shared.ID) (*domainsystem.DictionaryType, error) {
	if id.IsZero() {
		return nil, nil
	}
	var row DictionaryTypeModel
	err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Where("id = ?", id.String()).First(&row).Error
	})
	return dictionaryTypeFromRow(row, err)
}

func (s *SystemStore) GetDictionaryTypeByCode(ctx context.Context, code string) (*domainsystem.DictionaryType, error) {
	code = normalizeDictionaryCode(code)
	if code == "" {
		return nil, nil
	}
	var row DictionaryTypeModel
	err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Where("code = ?", code).First(&row).Error
	})
	return dictionaryTypeFromRow(row, err)
}

func (s *SystemStore) ListDictionaryTypes(ctx context.Context, offset, limit int) ([]domainsystem.DictionaryType, error) {
	q := s.db.WithContext(ctx).Model(&DictionaryTypeModel{}).Order("sort asc").Order("code asc")
	if offset > 0 {
		q = q.Offset(offset)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	var rows []DictionaryTypeModel
	if err := withDBRetry(func() error { return q.Find(&rows).Error }); err != nil {
		return nil, err
	}
	out := make([]domainsystem.DictionaryType, 0, len(rows))
	for _, row := range rows {
		item, err := row.ToDomain()
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, nil
}

func (s *SystemStore) SaveDictionaryType(ctx context.Context, dictType *domainsystem.DictionaryType) error {
	if dictType == nil {
		return fmt.Errorf("dictionary type is required")
	}
	row := DictionaryTypeModelFromDomain(*dictType)
	if strings.TrimSpace(row.ID) == "" {
		return fmt.Errorf("dictionary type id is required")
	}
	if _, err := row.ToDomain(); err != nil {
		return err
	}
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"code",
				"name",
				"description",
				"status",
				"sort",
				"builtin",
				"updated_at",
			}),
		}).Create(&row).Error
	})
}

func (s *SystemStore) DeleteDictionaryType(ctx context.Context, id shared.ID) error {
	if id.IsZero() {
		return nil
	}
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var row DictionaryTypeModel
			if err := tx.Where("id = ?", id.String()).First(&row).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					return nil
				}
				return err
			}
			if err := tx.Where("type_code = ?", row.Code).Delete(&DictionaryItemModel{}).Error; err != nil {
				return err
			}
			return tx.Delete(&DictionaryTypeModel{}, "id = ?", id.String()).Error
		})
	})
}

func (s *SystemStore) GetDictionaryItemByID(ctx context.Context, id shared.ID) (*domainsystem.DictionaryItem, error) {
	if id.IsZero() {
		return nil, nil
	}
	var row DictionaryItemModel
	err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Where("id = ?", id.String()).First(&row).Error
	})
	return dictionaryItemFromRow(row, err)
}

func (s *SystemStore) GetDictionaryItemByTypeAndValue(ctx context.Context, typeCode, value string) (*domainsystem.DictionaryItem, error) {
	typeCode = normalizeDictionaryCode(typeCode)
	value = strings.TrimSpace(value)
	if typeCode == "" || value == "" {
		return nil, nil
	}
	var row DictionaryItemModel
	err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Where("type_code = ? AND value = ?", typeCode, value).First(&row).Error
	})
	return dictionaryItemFromRow(row, err)
}

func (s *SystemStore) ListDictionaryItems(ctx context.Context, typeCode string, offset, limit int) ([]domainsystem.DictionaryItem, error) {
	typeCode = normalizeDictionaryCode(typeCode)
	q := s.db.WithContext(ctx).Model(&DictionaryItemModel{}).Order("sort asc").Order("value asc")
	if typeCode != "" {
		q = q.Where("type_code = ?", typeCode)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	var rows []DictionaryItemModel
	if err := withDBRetry(func() error { return q.Find(&rows).Error }); err != nil {
		return nil, err
	}
	out := make([]domainsystem.DictionaryItem, 0, len(rows))
	for _, row := range rows {
		item, err := row.ToDomain()
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, nil
}

func (s *SystemStore) SaveDictionaryItem(ctx context.Context, item *domainsystem.DictionaryItem) error {
	if item == nil {
		return fmt.Errorf("dictionary item is required")
	}
	row := DictionaryItemModelFromDomain(*item)
	if strings.TrimSpace(row.ID) == "" {
		return fmt.Errorf("dictionary item id is required")
	}
	if _, err := row.ToDomain(); err != nil {
		return err
	}
	if row.TypeCode == "" {
		return fmt.Errorf("dictionary item type code is required")
	}
	var exists int64
	if err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Model(&DictionaryTypeModel{}).Where("code = ?", row.TypeCode).Count(&exists).Error
	}); err != nil {
		return err
	}
	if exists == 0 {
		return fmt.Errorf("dictionary type does not exist")
	}
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"type_code",
				"label",
				"value",
				"status",
				"sort",
				"builtin",
				"updated_at",
			}),
		}).Create(&row).Error
	})
}

func (s *SystemStore) DeleteDictionaryItem(ctx context.Context, id shared.ID) error {
	if id.IsZero() {
		return nil
	}
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).Delete(&DictionaryItemModel{}, "id = ?", id.String()).Error
	})
}

func dictionaryTypeFromRow(row DictionaryTypeModel, err error) (*domainsystem.DictionaryType, error) {
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return row.ToDomain()
}

func dictionaryItemFromRow(row DictionaryItemModel, err error) (*domainsystem.DictionaryItem, error) {
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return row.ToDomain()
}

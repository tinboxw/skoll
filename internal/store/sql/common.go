package sql

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type Dialect string

const (
	DialectMySQL    Dialect = "mysql"
	DialectPostgres Dialect = "postgres"
)

func ParseDialect(raw string) (Dialect, error) {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case string(DialectMySQL):
		return DialectMySQL, nil
	case string(DialectPostgres):
		return DialectPostgres, nil
	default:
		return "", fmt.Errorf("unsupported sql dialect: %q", raw)
	}
}

func NormalizeDSN(dsn string) string {
	return strings.TrimSpace(dsn)
}

// FirstWhere loads one record by condition and returns nil when no rows match.
func FirstWhere[M any](ctx context.Context, db *gorm.DB, where any, args ...any) (*M, error) {
	var model M
	err := db.WithContext(ctx).Where(where, args...).First(&model).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &model, nil
}

// ListOrdered loads records with optional order/pagination.
func ListOrdered[M any](ctx context.Context, db *gorm.DB, order string, offset, limit int) ([]M, error) {
	q := db.WithContext(ctx)
	if strings.TrimSpace(order) != "" {
		q = q.Order(order)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	var models []M
	if err := q.Find(&models).Error; err != nil {
		return nil, err
	}
	return models, nil
}

// SaveModel upserts a model by primary key semantics.
func SaveModel[M any](ctx context.Context, db *gorm.DB, model *M) error {
	return db.WithContext(ctx).Save(model).Error
}

// DeleteByID removes one record by id column value.
func DeleteByID[M any](ctx context.Context, db *gorm.DB, id string) error {
	return db.WithContext(ctx).Delete(new(M), "id = ?", id).Error
}

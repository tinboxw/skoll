package sql

import (
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
)

func BuildSelectByLower(dialect, table, column, value string, limit uint) (string, []any, error) {
	if strings.TrimSpace(table) == "" {
		return "", nil, fmt.Errorf("table is required")
	}
	if strings.TrimSpace(column) == "" {
		return "", nil, fmt.Errorf("column is required")
	}
	ds := goqu.Dialect(strings.ToLower(strings.TrimSpace(dialect))).
		From(goqu.T(strings.TrimSpace(table))).
		Select(goqu.Star()).
		Prepared(true).
		Where(goqu.Func("LOWER", goqu.C(strings.TrimSpace(column))).Eq(strings.ToLower(strings.TrimSpace(value))))
	if limit > 0 {
		ds = ds.Limit(limit)
	}
	q, args, err := ds.ToSQL()
	if err != nil {
		return "", nil, err
	}
	return q, args, nil
}

func BuildOrderedSelect(dialect, table, orderColumn string, offset, limit int) (string, []any, error) {
	if strings.TrimSpace(table) == "" {
		return "", nil, fmt.Errorf("table is required")
	}
	if strings.TrimSpace(orderColumn) == "" {
		return "", nil, fmt.Errorf("order column is required")
	}
	ds := goqu.Dialect(strings.ToLower(strings.TrimSpace(dialect))).
		From(goqu.T(strings.TrimSpace(table))).
		Select(goqu.Star()).
		Prepared(true).
		Order(goqu.C(strings.TrimSpace(orderColumn)).Asc())
	if offset > 0 {
		ds = ds.Offset(uint(offset))
	}
	if limit > 0 {
		ds = ds.Limit(uint(limit))
	}
	q, args, err := ds.ToSQL()
	if err != nil {
		return "", nil, err
	}
	return q, args, nil
}

package sql

import (
	"fmt"
	"strings"
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

package db

import (
	"fmt"

	"gorm.io/gorm"
)

// Migrator is implemented by any package that owns a set of gorm models. The
// Adapter bootstrap calls Models() on each registered migrator and feeds the
// concatenated set into AutoMigrate.
type Migrator interface {
	// Models returns the gorm model values whose schema should be ensured.
	Models() []any
}

// Migrate runs gorm AutoMigrate for the supplied model values. It is the
// minimum-viable schema bootstrap and is intended to be replaced (or
// supplemented) by a versioned migration tool once schema churn slows down.
func Migrate(db *gorm.DB, models ...any) error {
	if db == nil {
		return fmt.Errorf("db.Migrate: nil db")
	}
	if len(models) == 0 {
		return nil
	}
	if err := db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("db.Migrate: %w", err)
	}
	return nil
}

// MigrateAll collects models from each Migrator and runs AutoMigrate once.
func MigrateAll(db *gorm.DB, migrators ...Migrator) error {
	var all []any
	for _, m := range migrators {
		if m == nil {
			continue
		}
		all = append(all, m.Models()...)
	}
	return Migrate(db, all...)
}

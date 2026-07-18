package gormrepo

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	pharmaoarepo "github.com/tinboxw/skoll/internal/repository/pharmaoa"
	mysqldriver "gorm.io/driver/mysql"
	postgresdriver "gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPharmaSchemaBaselineHasModelsAndMigrations(t *testing.T) {
	db := TestDB(t)
	migrations := pharmaMigrationText(t)
	baseline := pharmaoarepo.SchemaBaseline()
	if len(baseline) != 29 {
		t.Fatalf("unexpected Pharma OA schema table count: got %d want 29", len(baseline))
	}
	for _, table := range baseline {
		if !db.Migrator().HasTable(table.Name) {
			t.Fatalf("GORM model missing schema table %s", table.Name)
		}
		for _, column := range table.Columns {
			if !db.Migrator().HasColumn(table.Name, column) {
				t.Fatalf("GORM model %s missing schema column %s", table.Name, column)
			}
		}
		for _, index := range table.Indexes {
			if !db.Migrator().HasIndex(table.Name, index.Name) {
				t.Fatalf("GORM model %s missing schema index %s", table.Name, index.Name)
			}
		}
		for dialect, text := range migrations {
			if !strings.Contains(text, table.Name) {
				t.Fatalf("%s migrations missing schema table %s", dialect, table.Name)
			}
		}
	}
}

func TestPharmaSchemaUpgradePreservesExistingMasterData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "upgrade.db")
	db := openSchemaAcceptanceSQLite(t, path)
	master := []any{&PharmaEmployeeModel{}, &PharmaProductModel{}, &PharmaSupplierModel{}, &PharmaCustomerModel{}, &PharmaWarehouseModel{}}
	if err := db.AutoMigrate(master...); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)
	employee := &PharmaEmployeeModel{ID: "upgrade-employee", Code: "UPGRADE-EMP-001", Name: "升级保留员工", Status: "active", CertificatesJSON: "[]", CreatedAt: now, UpdatedAt: now, CreatedBy: "acceptance", UpdatedBy: "acceptance"}
	if err := db.Create(employee).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(AllModels()...); err != nil {
		t.Fatal(err)
	}
	var restored PharmaEmployeeModel
	if err := db.Where("id = ?", employee.ID).Take(&restored).Error; err != nil {
		t.Fatal(err)
	}
	if restored.Code != employee.Code || restored.Name != employee.Name {
		t.Fatalf("upgrade changed existing data: %+v", restored)
	}
	for _, table := range pharmaoarepo.SchemaBaseline() {
		if !db.Migrator().HasTable(table.Name) {
			t.Fatalf("upgrade missing table %s", table.Name)
		}
	}
}

func TestPharmaMigrationSQLMySQL(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("SKOLL_TEST_MYSQL_DSN"))
	if dsn == "" {
		t.Skip("SKOLL_TEST_MYSQL_DSN is not set")
	}
	runPharmaMigrationSQLContract(t, "mysql", mysqldriver.Open(dsn))
}

func TestPharmaMigrationSQLPostgreSQL(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("SKOLL_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("SKOLL_TEST_POSTGRES_DSN is not set")
	}
	runPharmaMigrationSQLContract(t, "postgres", postgresdriver.Open(dsn))
}

func runPharmaMigrationSQLContract(t *testing.T, dialect string, dialector gorm.Dialector) {
	t.Helper()
	db, err := gorm.Open(dialector, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open %s migration database: %v", dialect, err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	databaseName := externalDatabaseName(t, db, dialect)
	if databaseName != "skoll_acceptance" && !strings.HasSuffix(databaseName, "_acceptance") {
		t.Fatalf("refuse to reset non-acceptance %s database %q", dialect, databaseName)
	}
	resetPharmaMigrationTables(t, db, dialect)

	paths := pharmaMigrationPaths(t, dialect)
	for pass := 1; pass <= 2; pass++ {
		for _, path := range paths {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			for _, statement := range strings.Split(string(raw), ";") {
				statement = strings.TrimSpace(statement)
				if statement == "" {
					continue
				}
				if err := db.Exec(statement).Error; err != nil {
					t.Fatalf("execute %s migration %s pass %d: %v", dialect, filepath.Base(path), pass, err)
				}
			}
		}
	}

	for _, table := range pharmaoarepo.SchemaBaseline() {
		if !db.Migrator().HasTable(table.Name) {
			t.Fatalf("%s migration missing table %s", dialect, table.Name)
		}
		for _, column := range table.Columns {
			if !db.Migrator().HasColumn(table.Name, column) {
				t.Fatalf("%s migration table %s missing column %s", dialect, table.Name, column)
			}
		}
		for _, index := range table.Indexes {
			if !db.Migrator().HasIndex(table.Name, index.Name) {
				t.Fatalf("%s migration table %s missing index %s", dialect, table.Name, index.Name)
			}
		}
	}
}

func externalDatabaseName(t *testing.T, db *gorm.DB, dialect string) string {
	t.Helper()
	query := "SELECT current_database()"
	if dialect == "mysql" {
		query = "SELECT DATABASE()"
	}
	var name string
	if err := db.Raw(query).Scan(&name).Error; err != nil {
		t.Fatalf("read %s database name: %v", dialect, err)
	}
	return strings.TrimSpace(name)
}

func resetPharmaMigrationTables(t *testing.T, db *gorm.DB, dialect string) {
	t.Helper()
	tables := pharmaoarepo.SchemaBaseline()
	for i := len(tables) - 1; i >= 0; i-- {
		name := tables[i].Name
		statement := fmt.Sprintf("DROP TABLE IF EXISTS \"%s\"", name)
		if dialect == "mysql" {
			statement = fmt.Sprintf("DROP TABLE IF EXISTS `%s`", name)
		}
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("reset %s migration table %s: %v", dialect, name, err)
		}
	}
}

func pharmaMigrationText(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, dialect := range []string{"mysql", "postgres"} {
		var combined strings.Builder
		for _, path := range pharmaMigrationPaths(t, dialect) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			text := strings.ToLower(string(raw))
			if strings.Contains(text, "drop table") || strings.Contains(text, "truncate table") {
				t.Fatalf("destructive Pharma OA migration: %s", path)
			}
			combined.WriteString(text)
		}
		out[dialect] = combined.String()
	}
	return out
}

func pharmaMigrationPaths(t *testing.T, dialect string) []string {
	t.Helper()
	root := filepath.Join("..", "..", "..", "..", "migrations")
	matches, err := filepath.Glob(filepath.Join(root, dialect, "20260718_*.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(matches)
	if len(matches) != 4 {
		t.Fatalf("%s migration sequence is incomplete: %v", dialect, matches)
	}
	return matches
}

func openSchemaAcceptanceSQLite(t *testing.T, path string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	})
	return db
}

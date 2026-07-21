package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const fixturePrefix = "h5-bench-"

type options struct {
	dsn          string
	rows         int
	iterations   int
	output       string
	allowWrite   bool
	keepFixtures bool
}

type querySpec struct {
	Name            string
	Category        string
	SQL             string
	Args            []any
	P95BudgetMillis float64
	RequiredIndexes []string
}

type queryPlan struct {
	AccessType string `json:"accessType"`
	Key        string `json:"key"`
	Rows       int64  `json:"rows"`
	Extra      string `json:"extra"`
}

type queryResult struct {
	Name            string    `json:"name"`
	Category        string    `json:"category"`
	P50Millis       float64   `json:"p50Millis"`
	P95Millis       float64   `json:"p95Millis"`
	P99Millis       float64   `json:"p99Millis"`
	BudgetMillis    float64   `json:"p95BudgetMillis"`
	Iterations      int       `json:"iterations"`
	Plan            queryPlan `json:"plan"`
	RequiredIndexes []string  `json:"requiredIndexes"`
	Passed          bool      `json:"passed"`
	Failure         string    `json:"failure,omitempty"`
}

type report struct {
	WorkItem    string         `json:"workItem"`
	GeneratedAt string         `json:"generatedAt"`
	Database    string         `json:"database"`
	Version     string         `json:"version"`
	FixtureRows map[string]int `json:"fixtureRows"`
	Iterations  int            `json:"iterations"`
	Results     []queryResult  `json:"results"`
	Passed      bool           `json:"passed"`
}

func main() {
	opts := parseOptions()
	if !opts.allowWrite {
		log.Fatal("refusing to write benchmark fixtures without --allow-write-fixtures")
	}
	db, err := sql.Open("mysql", opts.dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(8)
	db.SetConnMaxLifetime(2 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("connect benchmark database: %v", err)
	}

	result, runErr := run(ctx, db, opts)
	if err := writeReport(opts.output, result); err != nil {
		log.Fatalf("write benchmark report: %v", err)
	}
	if runErr != nil {
		log.Fatal(runErr)
	}
	fmt.Printf("H5 MySQL benchmark passed: %d queries, %d iterations, %d fixture rows.\n", len(result.Results), result.Iterations, totalFixtures(result.FixtureRows))
}

func parseOptions() options {
	var opts options
	defaultDSN := strings.TrimSpace(os.Getenv("SKOLL_BENCHMARK_MYSQL_DSN"))
	flag.StringVar(&opts.dsn, "dsn", defaultDSN, "MySQL DSN; prefer SKOLL_BENCHMARK_MYSQL_DSN")
	flag.IntVar(&opts.rows, "rows", 10000, "base fixture rows per table")
	flag.IntVar(&opts.iterations, "iterations", 100, "timed query iterations")
	flag.StringVar(&opts.output, "output", "", "optional JSON report path")
	flag.BoolVar(&opts.allowWrite, "allow-write-fixtures", false, "allow prefixed fixture writes and cleanup")
	flag.BoolVar(&opts.keepFixtures, "keep-fixtures", false, "retain fixtures for manual query-plan inspection")
	flag.Parse()
	if opts.dsn == "" {
		log.Fatal("MySQL DSN is required through --dsn or SKOLL_BENCHMARK_MYSQL_DSN")
	}
	if opts.rows < 5000 || opts.rows > 100000 {
		log.Fatal("--rows must be between 5000 and 100000 so query plans represent large-list behavior")
	}
	if opts.iterations < 20 || opts.iterations > 1000 {
		log.Fatal("--iterations must be between 20 and 1000")
	}
	return opts
}

func run(ctx context.Context, db *sql.DB, opts options) (report, error) {
	var databaseName, version string
	if err := db.QueryRowContext(ctx, "SELECT DATABASE(), VERSION()").Scan(&databaseName, &version); err != nil {
		return report{}, err
	}
	if databaseName == "" {
		return report{}, errors.New("benchmark DSN must select a database")
	}
	if err := cleanupFixtures(ctx, db); err != nil {
		return report{}, fmt.Errorf("initial fixture cleanup: %w", err)
	}
	if !opts.keepFixtures {
		defer func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			if err := cleanupFixtures(cleanupCtx, db); err != nil {
				log.Printf("final fixture cleanup failed: %v", err)
			}
		}()
	}

	fixtureRows, err := seedFixtures(ctx, db, opts.rows)
	if err != nil {
		return report{}, fmt.Errorf("seed benchmark fixtures: %w", err)
	}
	if err := analyzeFixtures(ctx, db); err != nil {
		return report{}, fmt.Errorf("analyze benchmark fixtures: %w", err)
	}

	result := report{
		WorkItem:    "H5-01",
		GeneratedAt: time.Now().Format(time.RFC3339),
		Database:    databaseName,
		Version:     version,
		FixtureRows: fixtureRows,
		Iterations:  opts.iterations,
		Passed:      true,
	}
	for _, spec := range benchmarkQueries() {
		item, err := measureQuery(ctx, db, spec, opts.iterations)
		if err != nil {
			item = queryResult{Name: spec.Name, Category: spec.Category, BudgetMillis: spec.P95BudgetMillis, RequiredIndexes: spec.RequiredIndexes, Iterations: opts.iterations, Failure: err.Error()}
		}
		if !item.Passed {
			result.Passed = false
		}
		result.Results = append(result.Results, item)
	}
	if !result.Passed {
		return result, errors.New("one or more database query budgets or index plans failed")
	}
	return result, nil
}

func benchmarkQueries() []querySpec {
	return []querySpec{
		{
			Name: "customer_common_list", Category: "common_list",
			SQL:  "SELECT id, code, name FROM pharma_oa_customers WHERE organization_id = ? AND owner_id = ? AND status = ? ORDER BY code LIMIT 50",
			Args: []any{"h5-org-01", "h5-owner-001", "active"}, P95BudgetMillis: 25,
			RequiredIndexes: []string{"idx_pharma_customers_scope_status"},
		},
		{
			Name: "inventory_alert_dashboard", Category: "dashboard",
			SQL:  "SELECT type, COUNT(*) FROM pharma_oa_inventory_alerts WHERE status = ? AND last_seen_at >= ? GROUP BY type",
			Args: []any{"open", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}, P95BudgetMillis: 40,
			RequiredIndexes: []string{"idx_pharma_inventory_alerts_status_type_seen"},
		},
		{
			Name: "stock_balance_position_list", Category: "inventory",
			SQL:  "SELECT id, quantity, locked_quantity FROM pharma_oa_stock_balances WHERE warehouse_id = ? AND product_id = ? ORDER BY id LIMIT 100",
			Args: []any{"h5-warehouse-01", "h5-product-01"}, P95BudgetMillis: 25,
			RequiredIndexes: []string{"uk_pharma_balance_position"},
		},
		{
			Name: "customer_organization_scope", Category: "scope",
			SQL:  "SELECT id, code, name FROM pharma_oa_customers WHERE organization_id = ? AND status = ? ORDER BY code LIMIT 50",
			Args: []any{"h5-org-01", "active"}, P95BudgetMillis: 25,
			RequiredIndexes: []string{"idx_pharma_customers_org_status_code"},
		},
		{
			Name: "customer_owner_scope", Category: "scope",
			SQL:  "SELECT id, code, name FROM pharma_oa_customers WHERE owner_id = ? AND status = ? ORDER BY code LIMIT 50",
			Args: []any{"h5-owner-003", "active"}, P95BudgetMillis: 25,
			RequiredIndexes: []string{"idx_pharma_customers_owner_status_code"},
		},
		{
			Name: "report_export_owner_queue", Category: "report",
			SQL:  "SELECT id, status, created_at FROM pharma_oa_report_export_jobs WHERE owner_id = ? AND status = ? ORDER BY created_at DESC LIMIT 50",
			Args: []any{"h5-owner-001", "queued"}, P95BudgetMillis: 25,
			RequiredIndexes: []string{"idx_pharma_export_jobs_owner_status_created"},
		},
	}
}

func measureQuery(ctx context.Context, db *sql.DB, spec querySpec, iterations int) (queryResult, error) {
	plan, err := explainQuery(ctx, db, spec)
	if err != nil {
		return queryResult{}, fmt.Errorf("explain %s: %w", spec.Name, err)
	}
	for i := 0; i < 10; i++ {
		if err := consumeQuery(ctx, db, spec.SQL, spec.Args...); err != nil {
			return queryResult{}, err
		}
	}
	durations := make([]time.Duration, 0, iterations)
	for i := 0; i < iterations; i++ {
		started := time.Now()
		if err := consumeQuery(ctx, db, spec.SQL, spec.Args...); err != nil {
			return queryResult{}, err
		}
		durations = append(durations, time.Since(started))
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	item := queryResult{
		Name: spec.Name, Category: spec.Category, Iterations: iterations, Plan: plan,
		P50Millis: millis(percentile(durations, 0.50)), P95Millis: millis(percentile(durations, 0.95)), P99Millis: millis(percentile(durations, 0.99)),
		BudgetMillis: spec.P95BudgetMillis, RequiredIndexes: spec.RequiredIndexes,
	}
	missing := make([]string, 0)
	for _, index := range spec.RequiredIndexes {
		if !strings.Contains(plan.Key, index) {
			missing = append(missing, index)
		}
	}
	switch {
	case strings.EqualFold(plan.AccessType, "ALL") || plan.Key == "":
		item.Failure = fmt.Sprintf("query plan uses %s without an index", plan.AccessType)
	case len(missing) > 0:
		item.Failure = "query plan is missing required indexes: " + strings.Join(missing, ", ")
	case item.P95Millis > item.BudgetMillis:
		item.Failure = fmt.Sprintf("P95 %.3fms exceeds %.3fms budget", item.P95Millis, item.BudgetMillis)
	default:
		item.Passed = true
	}
	return item, nil
}

func explainQuery(ctx context.Context, db *sql.DB, spec querySpec) (queryPlan, error) {
	rows, err := db.QueryContext(ctx, "EXPLAIN "+spec.SQL, spec.Args...)
	if err != nil {
		return queryPlan{}, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return queryPlan{}, err
	}
	if !rows.Next() {
		return queryPlan{}, errors.New("empty EXPLAIN result")
	}
	values := make([]sql.RawBytes, len(columns))
	targets := make([]any, len(columns))
	for i := range values {
		targets[i] = &values[i]
	}
	if err := rows.Scan(targets...); err != nil {
		return queryPlan{}, err
	}
	fields := make(map[string]string, len(columns))
	for i, column := range columns {
		fields[strings.ToLower(column)] = string(values[i])
	}
	var estimated int64
	fmt.Sscan(fields["rows"], &estimated)
	return queryPlan{AccessType: fields["type"], Key: fields["key"], Rows: estimated, Extra: fields["extra"]}, nil
}

func consumeQuery(ctx context.Context, db *sql.DB, statement string, args ...any) error {
	rows, err := db.QueryContext(ctx, statement, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	values := make([]sql.RawBytes, len(columns))
	targets := make([]any, len(columns))
	for i := range values {
		targets[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(targets...); err != nil {
			return err
		}
	}
	return rows.Err()
}

func percentile(values []time.Duration, p float64) time.Duration {
	if len(values) == 0 {
		return 0
	}
	index := int(float64(len(values)-1) * p)
	return values[index]
}

func millis(value time.Duration) float64 { return float64(value) / float64(time.Millisecond) }

func seedFixtures(ctx context.Context, db *sql.DB, baseRows int) (map[string]int, error) {
	counts := map[string]int{
		"pharma_oa_customers":          baseRows,
		"pharma_oa_stock_balances":     baseRows * 2,
		"pharma_oa_inventory_alerts":   baseRows,
		"pharma_oa_report_export_jobs": baseRows,
	}
	now := time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC)
	if err := insertBatches(ctx, db, "pharma_oa_customers", []string{"id", "code", "name", "region", "organization_id", "owner_id", "rating", "status", "disable_reason", "contacts_json", "qualifications_json", "created_at", "updated_at", "created_by", "updated_by"}, counts["pharma_oa_customers"], func(i int) []any {
		return []any{fixtureID("customer", i), fixtureCode("CUS", i), fmt.Sprintf("H5 Customer %06d", i), []string{"East", "South", "West", "North"}[i%4], fmt.Sprintf("h5-org-%02d", i%10), fmt.Sprintf("h5-owner-%03d", i%100), i%5 + 1, status(i), "", "[]", "[]", now.Add(time.Duration(i) * time.Second), now.Add(time.Duration(i) * time.Second), "h5-benchmark", "h5-benchmark"}
	}); err != nil {
		return nil, err
	}
	if err := insertBatches(ctx, db, "pharma_oa_stock_balances", []string{"id", "product_id", "warehouse_id", "area_id", "location_id", "batch_id", "quantity", "locked_quantity", "version", "created_at", "updated_at", "created_by", "updated_by"}, counts["pharma_oa_stock_balances"], func(i int) []any {
		return []any{fixtureID("balance", i), fmt.Sprintf("h5-product-%02d", i%50), fmt.Sprintf("h5-warehouse-%02d", i%20), fmt.Sprintf("h5-area-%02d", i%10), fmt.Sprintf("h5-location-%03d", i%200), fmt.Sprintf("h5-batch-%08d", i), i%1000 + 1, i % 20, 1, now, now, "h5-benchmark", "h5-benchmark"}
	}); err != nil {
		return nil, err
	}
	if err := insertBatches(ctx, db, "pharma_oa_inventory_alerts", []string{"id", "type", "status", "balance_id", "product_id", "warehouse_id", "batch_id", "recipient_id", "notification_id", "last_seen_at", "resolved_at", "payload_json", "created_at", "updated_at", "created_by", "updated_by"}, counts["pharma_oa_inventory_alerts"], func(i int) []any {
		var resolved any
		if i%2 != 0 {
			resolved = now.Add(time.Duration(i) * time.Second)
		}
		return []any{fixtureID("alert", i), []string{"low_stock", "near_expiry", "overstock"}[i%3], []string{"open", "resolved"}[i%2], fixtureID("balance", i%(baseRows*2)), fmt.Sprintf("h5-product-%02d", i%50), fmt.Sprintf("h5-warehouse-%02d", i%20), fmt.Sprintf("h5-batch-%05d", i%5000), fmt.Sprintf("h5-owner-%03d", i%100), fixtureID("notification", i), now.Add(time.Duration(i) * time.Second), resolved, "{}", now, now, "h5-benchmark", "h5-benchmark"}
	}); err != nil {
		return nil, err
	}
	if err := insertBatches(ctx, db, "pharma_oa_report_export_jobs", []string{"id", "report_type", "status", "owner_id", "idempotency_key", "retry_count", "file_id", "error", "started_at", "completed_at", "payload_json", "created_at", "updated_at", "created_by", "updated_by"}, counts["pharma_oa_report_export_jobs"], func(i int) []any {
		return []any{fixtureID("report", i), []string{"inventory", "sales", "compliance"}[i%3], []string{"queued", "running", "completed"}[i%3], fmt.Sprintf("h5-owner-%03d", i%100), fixtureID("report-key", i), 0, "", "", nil, nil, "{}", now.Add(time.Duration(i) * time.Second), now.Add(time.Duration(i) * time.Second), "h5-benchmark", "h5-benchmark"}
	}); err != nil {
		return nil, err
	}
	return counts, nil
}

func insertBatches(ctx context.Context, db *sql.DB, table string, columns []string, count int, row func(int) []any) error {
	const batchSize = 250
	for start := 0; start < count; start += batchSize {
		end := start + batchSize
		if end > count {
			end = count
		}
		placeholders := make([]string, 0, end-start)
		args := make([]any, 0, (end-start)*len(columns))
		rowPlaceholder := "(" + strings.TrimSuffix(strings.Repeat("?,", len(columns)), ",") + ")"
		for i := start; i < end; i++ {
			values := row(i)
			if len(values) != len(columns) {
				return fmt.Errorf("fixture %s row %d has %d values for %d columns", table, i, len(values), len(columns))
			}
			placeholders = append(placeholders, rowPlaceholder)
			args = append(args, values...)
		}
		statement := "INSERT INTO " + table + " (" + strings.Join(columns, ",") + ") VALUES " + strings.Join(placeholders, ",")
		if _, err := db.ExecContext(ctx, statement, args...); err != nil {
			return fmt.Errorf("insert %s rows %d-%d: %w", table, start, end, err)
		}
	}
	return nil
}

func analyzeFixtures(ctx context.Context, db *sql.DB) error {
	for _, table := range []string{"pharma_oa_customers", "pharma_oa_stock_balances", "pharma_oa_inventory_alerts", "pharma_oa_report_export_jobs"} {
		if _, err := db.ExecContext(ctx, "ANALYZE TABLE "+table); err != nil {
			return err
		}
	}
	return nil
}

func cleanupFixtures(ctx context.Context, db *sql.DB) error {
	statements := []string{
		"DELETE FROM pharma_oa_inventory_alerts WHERE id LIKE 'h5-bench-%'",
		"DELETE FROM pharma_oa_report_export_jobs WHERE id LIKE 'h5-bench-%'",
		"DELETE FROM pharma_oa_stock_balances WHERE id LIKE 'h5-bench-%'",
		"DELETE FROM pharma_oa_customers WHERE id LIKE 'h5-bench-%'",
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func fixtureID(kind string, i int) string   { return fmt.Sprintf("%s%s-%08d", fixturePrefix, kind, i) }
func fixtureCode(kind string, i int) string { return fmt.Sprintf("H5-%s-%08d", kind, i) }
func status(i int) string {
	if i%5 == 0 {
		return "disabled"
	}
	return "active"
}

func writeReport(path string, value report) error {
	if strings.TrimSpace(path) == "" {
		raw, _ := json.MarshalIndent(value, "", "  ")
		fmt.Println(string(raw))
		return nil
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func totalFixtures(counts map[string]int) int {
	total := 0
	for _, count := range counts {
		total += count
	}
	return total
}

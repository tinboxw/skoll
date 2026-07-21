package plugin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var migrationFilePattern = regexp.MustCompile(`^([0-9]{3,})_[a-zA-Z0-9._-]+\.(up|down)\.sql$`)

type MigrationStep struct {
	Version  int
	Name     string
	Checksum string
	UpPath   string
	DownPath string
}

type MigrationRecord struct {
	PluginID  string
	Version   int
	Name      string
	Checksum  string
	AppliedAt time.Time
}

type MigrationPlan struct {
	Applied []MigrationStep
	Pending []MigrationStep
}

type MigrationTransaction interface {
	ExecSQL(sql string) error
	MarkApplied(record MigrationRecord) error
	RemoveApplied(pluginID string, version int) error
}

type MigrationStore interface {
	ListApplied(ctx context.Context, pluginID string) ([]MigrationRecord, error)
	WithTransaction(ctx context.Context, fn func(tx MigrationTransaction) error) error
	RequiresApplyCompensation() bool
}

type MigrationPlanner struct {
	pluginDir          string
	migrationDirectory string
}

type Migrator struct {
	pluginID string
	planner  *MigrationPlanner
	store    MigrationStore
	now      func() time.Time
}

func NewMigrationPlanner(pluginDir, migrationDirectory string) *MigrationPlanner {
	directory := strings.Trim(strings.TrimSpace(migrationDirectory), `/\`)
	if directory == "" {
		directory = "migrations"
	}
	return &MigrationPlanner{
		pluginDir:          strings.TrimSpace(pluginDir),
		migrationDirectory: directory,
	}
}

func NewMigrator(pluginID, pluginDir, migrationDirectory string, store MigrationStore) *Migrator {
	return &Migrator{
		pluginID: strings.TrimSpace(pluginID),
		planner:  NewMigrationPlanner(pluginDir, migrationDirectory),
		store:    store,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (p *MigrationPlanner) Plan(applied []MigrationRecord) (MigrationPlan, error) {
	steps, err := p.loadSteps()
	if err != nil {
		return MigrationPlan{}, err
	}

	appliedByVersion := make(map[int]MigrationRecord, len(applied))
	for _, record := range applied {
		if record.Version <= 0 {
			return MigrationPlan{}, fmt.Errorf("invalid applied migration version %d", record.Version)
		}
		if _, exists := appliedByVersion[record.Version]; exists {
			return MigrationPlan{}, fmt.Errorf("duplicate applied migration version %03d", record.Version)
		}
		appliedByVersion[record.Version] = record
	}

	plan := MigrationPlan{
		Applied: make([]MigrationStep, 0, len(applied)),
		Pending: make([]MigrationStep, 0, len(steps)),
	}
	for _, step := range steps {
		record, ok := appliedByVersion[step.Version]
		if !ok {
			plan.Pending = append(plan.Pending, step)
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(record.Checksum), step.Checksum) {
			return MigrationPlan{}, fmt.Errorf("migration %03d checksum does not match the applied ledger", step.Version)
		}
		plan.Applied = append(plan.Applied, step)
		delete(appliedByVersion, step.Version)
	}
	if len(appliedByVersion) > 0 {
		versions := make([]int, 0, len(appliedByVersion))
		for version := range appliedByVersion {
			versions = append(versions, version)
		}
		sort.Ints(versions)
		return MigrationPlan{}, fmt.Errorf("applied migration %03d is missing from the plugin package", versions[0])
	}
	return plan, nil
}

func (m *Migrator) Plan(ctx context.Context) (MigrationPlan, error) {
	if err := m.validate(); err != nil {
		return MigrationPlan{}, err
	}
	records, err := m.store.ListApplied(normalizeMigrationContext(ctx), m.pluginID)
	if err != nil {
		return MigrationPlan{}, fmt.Errorf("list applied plugin migrations: %w", err)
	}
	return m.planner.Plan(records)
}

func (m *Migrator) Apply(ctx context.Context, limit int) ([]MigrationStep, error) {
	ctx = normalizeMigrationContext(ctx)
	plan, err := m.Plan(ctx)
	if err != nil {
		return nil, err
	}
	if len(plan.Pending) == 0 {
		return []MigrationStep{}, nil
	}
	if limit <= 0 || limit > len(plan.Pending) {
		limit = len(plan.Pending)
	}
	toApply := append([]MigrationStep(nil), plan.Pending[:limit]...)
	scripts, err := readMigrationScripts(toApply, true)
	if err != nil {
		return nil, err
	}

	downScripts := []string(nil)
	if m.store.RequiresApplyCompensation() {
		downScripts, err = readMigrationScripts(toApply, false)
		if err != nil {
			return nil, err
		}
	}
	attempted := 0
	err = m.store.WithTransaction(ctx, func(tx MigrationTransaction) error {
		for i, step := range toApply {
			attempted = i + 1
			if err := tx.ExecSQL(scripts[i]); err != nil {
				return fmt.Errorf("apply migration %03d: %w", step.Version, err)
			}
			appliedAt := time.Now().UTC()
			if m.now != nil {
				appliedAt = m.now().UTC()
			}
			if err := tx.MarkApplied(MigrationRecord{
				PluginID:  m.pluginID,
				Version:   step.Version,
				Name:      step.Name,
				Checksum:  step.Checksum,
				AppliedAt: appliedAt,
			}); err != nil {
				return fmt.Errorf("record migration %03d: %w", step.Version, err)
			}
		}
		return nil
	})
	if err != nil {
		if attempted > 0 && m.store.RequiresApplyCompensation() {
			if compensationErr := m.compensateApply(ctx, toApply[:attempted], downScripts[:attempted]); compensationErr != nil {
				return nil, errors.Join(err, fmt.Errorf("migration compensation failed: %w", compensationErr))
			}
		}
		return nil, err
	}
	return toApply, nil
}

func (m *Migrator) compensateApply(ctx context.Context, steps []MigrationStep, scripts []string) error {
	compensationErrors := make([]error, 0)
	err := m.store.WithTransaction(ctx, func(tx MigrationTransaction) error {
		for i := len(steps) - 1; i >= 0; i-- {
			if execErr := tx.ExecSQL(scripts[i]); execErr != nil {
				compensationErrors = append(compensationErrors, fmt.Errorf("rollback attempted migration %03d: %w", steps[i].Version, execErr))
			}
			if ledgerErr := tx.RemoveApplied(m.pluginID, steps[i].Version); ledgerErr != nil {
				compensationErrors = append(compensationErrors, fmt.Errorf("remove attempted migration %03d ledger entry: %w", steps[i].Version, ledgerErr))
			}
		}
		return nil
	})
	if err != nil {
		compensationErrors = append(compensationErrors, err)
	}
	return errors.Join(compensationErrors...)
}

func (m *Migrator) Rollback(ctx context.Context, limit int) ([]MigrationStep, error) {
	ctx = normalizeMigrationContext(ctx)
	plan, err := m.Plan(ctx)
	if err != nil {
		return nil, err
	}
	if len(plan.Applied) == 0 {
		return []MigrationStep{}, nil
	}
	if limit <= 0 || limit > len(plan.Applied) {
		limit = len(plan.Applied)
	}

	toRollback := make([]MigrationStep, 0, limit)
	for i := len(plan.Applied) - 1; i >= 0 && len(toRollback) < limit; i-- {
		toRollback = append(toRollback, plan.Applied[i])
	}
	scripts, err := readMigrationScripts(toRollback, false)
	if err != nil {
		return nil, err
	}

	err = m.store.WithTransaction(ctx, func(tx MigrationTransaction) error {
		for i, step := range toRollback {
			if err := tx.ExecSQL(scripts[i]); err != nil {
				return fmt.Errorf("rollback migration %03d: %w", step.Version, err)
			}
			if err := tx.RemoveApplied(m.pluginID, step.Version); err != nil {
				return fmt.Errorf("remove migration %03d ledger entry: %w", step.Version, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return toRollback, nil
}

func (m *Migrator) validate() error {
	if m == nil || strings.TrimSpace(m.pluginID) == "" {
		return fmt.Errorf("plugin id is required")
	}
	if m.planner == nil {
		return fmt.Errorf("migration planner is required")
	}
	if m.store == nil {
		return fmt.Errorf("transactional plugin migration store is required")
	}
	return nil
}

func (p *MigrationPlanner) loadSteps() ([]MigrationStep, error) {
	if p == nil || p.pluginDir == "" {
		return nil, fmt.Errorf("plugin dir is required")
	}
	migrationsDir := filepath.Join(p.pluginDir, p.migrationDirectory)
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	type pair struct {
		version int
		name    string
		up      string
		down    string
	}
	pairs := map[int]*pair{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		match := migrationFilePattern.FindStringSubmatch(name)
		if len(match) != 3 {
			continue
		}
		version, convErr := strconv.Atoi(match[1])
		if convErr != nil {
			return nil, fmt.Errorf("invalid migration version in %s", name)
		}
		typ := match[2]
		p, ok := pairs[version]
		if !ok {
			p = &pair{version: version}
			pairs[version] = p
		}
		fullPath := filepath.Join(migrationsDir, name)
		if typ == "up" {
			p.up = fullPath
			p.name = strings.TrimSuffix(name, ".up.sql")
		} else {
			p.down = fullPath
		}
	}

	if len(pairs) == 0 {
		return nil, fmt.Errorf("no migration files found under %s", migrationsDir)
	}
	versions := make([]int, 0, len(pairs))
	for version := range pairs {
		versions = append(versions, version)
	}
	sort.Ints(versions)

	steps := make([]MigrationStep, 0, len(versions))
	for _, version := range versions {
		p := pairs[version]
		if strings.TrimSpace(p.up) == "" || strings.TrimSpace(p.down) == "" {
			return nil, fmt.Errorf("migration %03d requires both up and down sql files", version)
		}
		upSQL, err := readMigrationSQL(p.up)
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256([]byte(upSQL))
		steps = append(steps, MigrationStep{
			Version:  version,
			Name:     p.name,
			Checksum: hex.EncodeToString(digest[:]),
			UpPath:   p.up,
			DownPath: p.down,
		})
	}
	return steps, nil
}

func readMigrationScripts(steps []MigrationStep, up bool) ([]string, error) {
	scripts := make([]string, 0, len(steps))
	for _, step := range steps {
		path := step.DownPath
		if up {
			path = step.UpPath
		}
		sql, err := readMigrationSQL(path)
		if err != nil {
			return nil, err
		}
		scripts = append(scripts, sql)
	}
	return scripts, nil
}

func readMigrationSQL(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sql := strings.TrimSpace(string(raw))
	if sql == "" {
		return "", fmt.Errorf("migration sql is empty: %s", path)
	}
	return sql, nil
}

func normalizeMigrationContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var migrationFilePattern = regexp.MustCompile(`^([0-9]{3,})_[a-zA-Z0-9._-]+\.(up|down)\.sql$`)

type MigrationStep struct {
	Version  int
	Name     string
	UpPath   string
	DownPath string
}

type MigrationState struct {
	AppliedVersions []int `json:"appliedVersions"`
}

type MigrationPlan struct {
	Applied []MigrationStep
	Pending []MigrationStep
}

type Migrator struct {
	pluginDir string
}

func NewMigrator(pluginDir string) *Migrator {
	return &Migrator{pluginDir: strings.TrimSpace(pluginDir)}
}

func (m *Migrator) Plan() (MigrationPlan, error) {
	steps, err := m.loadSteps()
	if err != nil {
		return MigrationPlan{}, err
	}
	state, err := m.loadState()
	if err != nil {
		return MigrationPlan{}, err
	}

	appliedMap := make(map[int]struct{}, len(state.AppliedVersions))
	for _, v := range state.AppliedVersions {
		appliedMap[v] = struct{}{}
	}

	plan := MigrationPlan{
		Applied: make([]MigrationStep, 0),
		Pending: make([]MigrationStep, 0),
	}
	for _, step := range steps {
		if _, ok := appliedMap[step.Version]; ok {
			plan.Applied = append(plan.Applied, step)
			continue
		}
		plan.Pending = append(plan.Pending, step)
	}
	return plan, nil
}

func (m *Migrator) Apply(limit int) ([]MigrationStep, error) {
	plan, err := m.Plan()
	if err != nil {
		return nil, err
	}
	if len(plan.Pending) == 0 {
		return nil, nil
	}

	if limit <= 0 || limit > len(plan.Pending) {
		limit = len(plan.Pending)
	}
	toApply := plan.Pending[:limit]

	state, err := m.loadState()
	if err != nil {
		return nil, err
	}
	for _, step := range toApply {
		if err := ensureMigrationSQL(step.UpPath); err != nil {
			return nil, err
		}
		state.AppliedVersions = append(state.AppliedVersions, step.Version)
	}
	if err := m.saveState(state); err != nil {
		return nil, err
	}
	return toApply, nil
}

func (m *Migrator) Rollback(limit int) ([]MigrationStep, error) {
	plan, err := m.Plan()
	if err != nil {
		return nil, err
	}
	if len(plan.Applied) == 0 {
		return nil, nil
	}

	if limit <= 0 || limit > len(plan.Applied) {
		limit = len(plan.Applied)
	}

	toRollback := make([]MigrationStep, 0, limit)
	for i := len(plan.Applied) - 1; i >= 0 && len(toRollback) < limit; i-- {
		toRollback = append(toRollback, plan.Applied[i])
	}

	state, err := m.loadState()
	if err != nil {
		return nil, err
	}
	for _, step := range toRollback {
		if err := ensureMigrationSQL(step.DownPath); err != nil {
			return nil, err
		}
		state.AppliedVersions = removeMigrationVersion(state.AppliedVersions, step.Version)
	}
	if err := m.saveState(state); err != nil {
		return nil, err
	}
	return toRollback, nil
}

func (m *Migrator) loadSteps() ([]MigrationStep, error) {
	if m == nil || m.pluginDir == "" {
		return nil, fmt.Errorf("plugin dir is required")
	}
	migrationsDir := filepath.Join(m.pluginDir, "migrations")
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
			p = &pair{version: version, name: name}
			pairs[version] = p
		}

		fullPath := filepath.Join(migrationsDir, name)
		if typ == "up" {
			p.up = fullPath
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
		steps = append(steps, MigrationStep{
			Version:  version,
			Name:     p.name,
			UpPath:   p.up,
			DownPath: p.down,
		})
	}
	return steps, nil
}

func (m *Migrator) loadState() (MigrationState, error) {
	statePath := m.stateFilePath()
	raw, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return MigrationState{AppliedVersions: []int{}}, nil
		}
		return MigrationState{}, err
	}

	var state MigrationState
	if err := json.Unmarshal(raw, &state); err != nil {
		return MigrationState{}, err
	}
	if state.AppliedVersions == nil {
		state.AppliedVersions = []int{}
	}
	sort.Ints(state.AppliedVersions)
	return state, nil
}

func (m *Migrator) saveState(state MigrationState) error {
	sort.Ints(state.AppliedVersions)
	state.AppliedVersions = uniqueSortedVersions(state.AppliedVersions)
	statePath := m.stateFilePath()
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath, raw, 0o644)
}

func (m *Migrator) stateFilePath() string {
	return filepath.Join(m.pluginDir, ".skoll", "migration-state.json")
}

func ensureMigrationSQL(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(raw)) == "" {
		return fmt.Errorf("migration sql is empty: %s", path)
	}
	return nil
}

func removeMigrationVersion(versions []int, target int) []int {
	if len(versions) == 0 {
		return versions
	}
	out := make([]int, 0, len(versions))
	removed := false
	for i := len(versions) - 1; i >= 0; i-- {
		v := versions[i]
		if !removed && v == target {
			removed = true
			continue
		}
		out = append(out, v)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func uniqueSortedVersions(in []int) []int {
	if len(in) == 0 {
		return in
	}
	out := make([]int, 0, len(in))
	last := in[0] - 1
	for _, v := range in {
		if len(out) == 0 || v != last {
			out = append(out, v)
			last = v
		}
	}
	return out
}

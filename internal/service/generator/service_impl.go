package generator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	domaingenerator "github.com/tinboxw/skoll/internal/domain/generator"
)

type HistoryStore interface {
	Save(ctx context.Context, history GenerationHistory) error
	Get(ctx context.Context, batchID string) (*GenerationHistory, error)
}

type serviceImpl struct {
	history HistoryStore
}

func NewService() Service {
	return &serviceImpl{history: NewMemoryHistoryStore()}
}

func NewServiceWithHistory(history HistoryStore) Service {
	if history == nil {
		history = NewMemoryHistoryStore()
	}
	return &serviceImpl{history: history}
}

func (s *serviceImpl) DryRun(_ context.Context, in DryRunInput) (*DryRunResult, error) {
	if in.Spec == nil {
		return nil, fmt.Errorf("generator spec is required")
	}
	batchID := strings.TrimSpace(in.BatchID)
	if batchID == "" {
		batchID = "dry-run"
	}
	actorID := strings.TrimSpace(in.ActorID)
	migrationTimestamp := strings.TrimSpace(in.MigrationTimestamp)
	if migrationTimestamp == "" {
		migrationTimestamp = in.Spec.Meta.CreatedAt.UTC().Format("20060102_150405")
	}

	snapshots := make(map[string]FileSnapshot, len(in.ExistingFiles))
	for _, snapshot := range in.ExistingFiles {
		path := normalizePath(snapshot.Path)
		if path == "" {
			return nil, fmt.Errorf("file snapshot path is required")
		}
		snapshot.Path = path
		snapshots[path] = snapshot
	}

	candidates := buildCandidates(*in.Spec, migrationTimestamp)
	files := make([]FilePlan, 0, len(candidates))
	summary := DryRunSummary{}
	for _, candidate := range candidates {
		plan := classifyCandidate(candidate, snapshots[candidate.Path])
		files = append(files, plan)
		summary.add(plan.Status)
	}
	sort.SliceStable(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	return &DryRunResult{
		BatchID:  batchID,
		SpecID:   in.Spec.ID.String(),
		ActorID:  actorID,
		SpecHash: specHash(*in.Spec),
		Files:    files,
		Summary:  summary,
	}, nil
}

func (s *serviceImpl) RecordHistory(ctx context.Context, in RecordHistoryInput) (*GenerationHistory, error) {
	if in.DryRun == nil {
		return nil, fmt.Errorf("generator dry-run result is required")
	}
	if in.Spec == nil {
		return nil, fmt.Errorf("generator spec is required")
	}
	if s == nil || s.history == nil {
		return nil, fmt.Errorf("generator history store is not configured")
	}
	actorID := strings.TrimSpace(in.ActorID)
	if actorID == "" {
		actorID = strings.TrimSpace(in.DryRun.ActorID)
	}
	if actorID == "" {
		return nil, fmt.Errorf("generator actor id is required")
	}
	createdAt := in.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	history := GenerationHistory{
		BatchID:      strings.TrimSpace(in.DryRun.BatchID),
		ActorID:      actorID,
		SpecID:       in.Spec.ID.String(),
		SpecHash:     specHash(*in.Spec),
		SpecSnapshot: specSnapshot(*in.Spec),
		Files:        make([]GeneratedFileRecord, 0, len(in.DryRun.Files)),
		CreatedAt:    createdAt,
	}
	if history.BatchID == "" {
		return nil, fmt.Errorf("generator batch id is required")
	}
	for _, file := range in.DryRun.Files {
		history.Files = append(history.Files, GeneratedFileRecord{
			Path:             file.Path,
			TemplateID:       file.TemplateID,
			Status:           file.Status,
			Hash:             file.ContentHash,
			GeneratedContent: file.GeneratedContent,
			PreviousHash:     file.CurrentHash,
			PreviousContent:  file.CurrentContent,
		})
	}
	if err := s.history.Save(ctx, history); err != nil {
		return nil, err
	}
	return &history, nil
}

func (s *serviceImpl) GetHistory(ctx context.Context, batchID string) (*GenerationHistory, error) {
	if s == nil || s.history == nil {
		return nil, fmt.Errorf("generator history store is not configured")
	}
	batchID = strings.TrimSpace(batchID)
	if batchID == "" {
		return nil, fmt.Errorf("generator batch id is required")
	}
	return s.history.Get(ctx, batchID)
}

func (s *serviceImpl) PlanRollback(ctx context.Context, in RollbackInput) (*RollbackPlan, error) {
	history, err := s.GetHistory(ctx, in.BatchID)
	if err != nil {
		return nil, err
	}
	current := make(map[string]FileSnapshot, len(in.CurrentFiles))
	for _, file := range in.CurrentFiles {
		path := normalizePath(file.Path)
		if path == "" {
			return nil, fmt.Errorf("rollback current file path is required")
		}
		file.Path = path
		current[path] = file
	}
	plan := &RollbackPlan{
		BatchID:   history.BatchID,
		Files:     make([]RollbackFilePlan, 0, len(history.Files)),
		Conflicts: make([]RollbackFilePlan, 0),
	}
	for _, file := range history.Files {
		item := rollbackFile(file, current[file.Path])
		plan.Files = append(plan.Files, item)
		if item.Action == RollbackActionConflict || item.Action == RollbackActionManual {
			plan.Conflicts = append(plan.Conflicts, item)
		}
	}
	return plan, nil
}

type fileCandidate struct {
	Path       string
	TemplateID string
	Payload    string
}

func buildCandidates(spec domaingenerator.GeneratorSpec, migrationTimestamp string) []fileCandidate {
	module := spec.Module.Package
	domainName := spec.Table.DomainName
	table := spec.Table.Name
	candidates := []fileCandidate{
		candidate("backend.domain.doc", fmt.Sprintf("internal/domain/%s/doc.go", module), spec),
		candidate("backend.domain.entity", fmt.Sprintf("internal/domain/%s/entity.go", module), spec),
		candidate("backend.repository", fmt.Sprintf("internal/repository/%s/%s_repo.go", module, module), spec),
		candidate("backend.store.memory", fmt.Sprintf("internal/store/memory/%s_store.go", module), spec),
		candidate("backend.store.gorm.model", fmt.Sprintf("internal/store/sql/gormrepo/%s_model.go", module), spec),
		candidate("backend.store.gorm.repo", fmt.Sprintf("internal/store/sql/gormrepo/%s_store.go", module), spec),
		candidate("backend.store.gorm.registry", "internal/store/sql/gormrepo/all_models.go", spec),
		candidate("backend.store.factory", "internal/store/factory.go", spec),
		candidate("backend.migration.mysql", fmt.Sprintf("migrations/mysql/%s_create_%s.sql", migrationTimestamp, table), spec),
		candidate("backend.migration.postgres", fmt.Sprintf("migrations/postgres/%s_create_%s.sql", migrationTimestamp, table), spec),
		candidate("backend.service", fmt.Sprintf("internal/service/%s/service.go", module), spec),
		candidate("backend.service.impl", fmt.Sprintf("internal/service/%s/service_impl.go", module), spec),
		candidate("backend.handler", fmt.Sprintf("internal/handler/http/v1/%s/handler.go", module), spec),
		candidate("backend.router", "internal/handler/http/v1/router.go", spec),
		candidate("backend.openapi.docs", "docs/api/openapi.yaml", spec),
		candidate("backend.openapi.runtime", "internal/handler/http/openapi.yaml", spec),
		candidate("backend.permission.seed", "internal/bootstrap/permission_menu_seed.go", spec),
		candidate("frontend.api", fmt.Sprintf("web/src/api/%s.ts", module), spec),
		candidate("frontend.store", fmt.Sprintf("web/src/stores/%s.ts", module), spec),
		candidate("frontend.view", fmt.Sprintf("web/src/views/%s/index.vue", domainName), spec),
	}
	if spec.Plugin.Enabled {
		pluginRoot := fmt.Sprintf("examples/plugins/%s", spec.Plugin.ID)
		candidates = append(candidates,
			candidate("plugin.manifest", pluginRoot+"/plugin.yaml", spec),
			candidate("plugin.migration.up", fmt.Sprintf("%s/%s/001_create_%s.up.sql", pluginRoot, spec.Plugin.MigrationDirectory, table), spec),
			candidate("plugin.migration.down", fmt.Sprintf("%s/%s/001_create_%s.down.sql", pluginRoot, spec.Plugin.MigrationDirectory, table), spec),
			candidate("plugin.frontend.api", fmt.Sprintf("%s/web/src/api/%s.ts", pluginRoot, module), spec),
			candidate("plugin.frontend.store", fmt.Sprintf("%s/web/src/stores/%s.ts", pluginRoot, module), spec),
			candidate("plugin.frontend.view", fmt.Sprintf("%s/web/src/views/%s/index.vue", pluginRoot, domainName), spec),
			candidate("plugin.acceptance.test", pluginRoot+"/plugin_acceptance_test.go", spec),
			candidate("plugin.readme", pluginRoot+"/README.md", spec),
		)
	}
	return candidates
}

func candidate(templateID, path string, spec domaingenerator.GeneratorSpec) fileCandidate {
	return fileCandidate{
		Path:       normalizePath(path),
		TemplateID: templateID,
		Payload:    renderCandidateContent(templateID, spec),
	}
}

func classifyCandidate(candidate fileCandidate, snapshot FileSnapshot) FilePlan {
	hash := contentHash(candidate.Payload)
	plan := FilePlan{
		Path:             candidate.Path,
		TemplateID:       candidate.TemplateID,
		ContentHash:      hash,
		GeneratedContent: candidate.Payload,
	}
	if !allowedOutputPath(candidate.Path) {
		plan.Status = FileStatusBlocked
		plan.Reason = "path is outside generator output matrix"
		plan.Summary = "blocked: " + candidate.Path
		return plan
	}
	if snapshot.Path == "" {
		plan.Status = FileStatusCreate
		plan.Diff = buildCreateDiff(candidate.Path, candidate.Payload)
		plan.Summary = "create: " + candidate.Path
		return plan
	}
	plan.CurrentHash = strings.TrimSpace(snapshot.CurrentHash)
	plan.CurrentContent = snapshot.CurrentContent
	plan.PreviousGeneratedHash = strings.TrimSpace(snapshot.PreviousGeneratedHash)
	if plan.CurrentHash == hash {
		plan.Status = FileStatusUnchanged
		plan.Summary = "unchanged: " + candidate.Path
		return plan
	}
	if plan.PreviousGeneratedHash != "" && plan.CurrentHash == plan.PreviousGeneratedHash {
		plan.Status = FileStatusUpdateClean
		plan.Diff = buildChangeDiff(candidate.Path, snapshot.CurrentContent, candidate.Payload)
		plan.Summary = "update-clean: " + candidate.Path
		return plan
	}
	plan.Status = FileStatusConflict
	if plan.PreviousGeneratedHash == "" {
		plan.Reason = "existing file has no generated ownership record"
	} else {
		plan.Reason = "existing file differs from previous generated hash"
	}
	plan.Diff = buildChangeDiff(candidate.Path, snapshot.CurrentContent, candidate.Payload)
	plan.Summary = "conflict: " + candidate.Path
	return plan
}

func (s *DryRunSummary) add(status FileStatus) {
	s.Total++
	switch status {
	case FileStatusCreate:
		s.Create++
	case FileStatusUnchanged:
		s.Unchanged++
	case FileStatusUpdateClean:
		s.UpdateClean++
	case FileStatusConflict:
		s.Conflict++
	case FileStatusBlocked:
		s.Blocked++
	}
}

func contentHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func specHash(spec domaingenerator.GeneratorSpec) string {
	return contentHash(specSnapshot(spec))
}

func specSnapshot(spec domaingenerator.GeneratorSpec) string {
	data, err := json.Marshal(spec)
	if err != nil {
		return spec.ID.String()
	}
	return string(data)
}

type memoryHistoryStore struct {
	mu      sync.RWMutex
	records map[string]GenerationHistory
}

func NewMemoryHistoryStore() HistoryStore {
	return &memoryHistoryStore{records: make(map[string]GenerationHistory)}
}

func (s *memoryHistoryStore) Save(_ context.Context, history GenerationHistory) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	copyHistory := cloneHistory(history)
	s.records[history.BatchID] = copyHistory
	return nil
}

func (s *memoryHistoryStore) Get(_ context.Context, batchID string) (*GenerationHistory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	history, ok := s.records[batchID]
	if !ok {
		return nil, fmt.Errorf("generator history not found")
	}
	copyHistory := cloneHistory(history)
	return &copyHistory, nil
}

func cloneHistory(history GenerationHistory) GenerationHistory {
	history.Files = append([]GeneratedFileRecord(nil), history.Files...)
	return history
}

func rollbackFile(record GeneratedFileRecord, current FileSnapshot) RollbackFilePlan {
	item := RollbackFilePlan{
		Path:         record.Path,
		ExpectedHash: record.Hash,
		CurrentHash:  strings.TrimSpace(current.CurrentHash),
	}
	if record.Status == FileStatusUnchanged {
		item.Action = RollbackActionNoop
		item.Reason = "file was unchanged by generation"
		return item
	}
	if record.Status == FileStatusConflict || record.Status == FileStatusBlocked {
		item.Action = RollbackActionManual
		item.Reason = "file was not written by generation"
		return item
	}
	if current.Path == "" {
		item.Action = RollbackActionNoop
		item.Reason = "generated file is already absent"
		return item
	}
	if item.CurrentHash != record.Hash {
		item.Action = RollbackActionConflict
		item.Reason = "current file differs from generated history hash"
		return item
	}
	switch record.Status {
	case FileStatusCreate:
		item.Action = RollbackActionDelete
		item.Reason = "generated file can be removed"
	case FileStatusUpdateClean:
		item.Action = RollbackActionRestore
		item.Reason = "generated file can be restored to previous content"
		item.RestoredHash = record.PreviousHash
		item.RestoredContent = record.PreviousContent
	default:
		item.Action = RollbackActionManual
		item.Reason = "rollback action is not defined for file status"
	}
	return item
}

func buildCreateDiff(path, generated string) string {
	var b strings.Builder
	b.WriteString("--- /dev/null\n")
	b.WriteString("+++ generated/")
	b.WriteString(path)
	b.WriteString("\n@@\n")
	for _, line := range splitDiffLines(generated) {
		b.WriteString("+")
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func buildChangeDiff(path, current, generated string) string {
	if current == "" {
		current = "(current content unavailable)"
	}
	var b strings.Builder
	b.WriteString("--- current/")
	b.WriteString(path)
	b.WriteString("\n+++ generated/")
	b.WriteString(path)
	b.WriteString("\n@@\n")
	for _, line := range splitDiffLines(current) {
		b.WriteString("-")
		b.WriteString(line)
		b.WriteString("\n")
	}
	for _, line := range splitDiffLines(generated) {
		b.WriteString("+")
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func splitDiffLines(value string) []string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.TrimSuffix(value, "\n")
	if value == "" {
		return []string{""}
	}
	return strings.Split(value, "\n")
}

func normalizePath(path string) string {
	path = strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	return strings.TrimPrefix(path, "./")
}

func allowedOutputPath(path string) bool {
	if path == "" || strings.HasPrefix(path, "/") || strings.Contains(path, ":") || strings.Contains(path, "../") || strings.HasPrefix(path, "..") {
		return false
	}
	allowedPrefixes := []string{
		"internal/domain/",
		"internal/repository/",
		"internal/store/memory/",
		"internal/store/sql/gormrepo/",
		"internal/store/factory.go",
		"migrations/mysql/",
		"migrations/postgres/",
		"internal/service/",
		"internal/handler/http/v1/",
		"internal/handler/http/openapi.yaml",
		"docs/api/openapi.yaml",
		"internal/bootstrap/permission_menu_seed.go",
		"web/src/api/",
		"web/src/stores/",
		"web/src/views/",
		"examples/plugins/",
	}
	for _, prefix := range allowedPrefixes {
		if path == prefix || strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

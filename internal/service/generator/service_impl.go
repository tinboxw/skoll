package generator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	domaingenerator "github.com/tinboxw/skoll/internal/domain/generator"
)

type serviceImpl struct{}

func NewService() Service {
	return &serviceImpl{}
}

func (s *serviceImpl) DryRun(_ context.Context, in DryRunInput) (*DryRunResult, error) {
	if in.Spec == nil {
		return nil, fmt.Errorf("generator spec is required")
	}
	batchID := strings.TrimSpace(in.BatchID)
	if batchID == "" {
		batchID = "dry-run"
	}
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
		BatchID: batchID,
		SpecID:  in.Spec.ID.String(),
		Files:   files,
		Summary: summary,
	}, nil
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
	return []fileCandidate{
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
}

func candidate(templateID, path string, spec domaingenerator.GeneratorSpec) fileCandidate {
	return fileCandidate{
		Path:       normalizePath(path),
		TemplateID: templateID,
		Payload: fmt.Sprintf("%s|%s|%s|%s|%s",
			templateID,
			spec.ID.String(),
			spec.Module.Package,
			spec.Table.Name,
			strings.Join(spec.FieldNames(), ",")),
	}
}

func classifyCandidate(candidate fileCandidate, snapshot FileSnapshot) FilePlan {
	hash := contentHash(candidate.Payload)
	plan := FilePlan{
		Path:        candidate.Path,
		TemplateID:  candidate.TemplateID,
		ContentHash: hash,
	}
	if !allowedOutputPath(candidate.Path) {
		plan.Status = FileStatusBlocked
		plan.Reason = "path is outside generator output matrix"
		plan.Summary = "blocked: " + candidate.Path
		return plan
	}
	if snapshot.Path == "" {
		plan.Status = FileStatusCreate
		plan.Summary = "create: " + candidate.Path
		return plan
	}
	plan.CurrentHash = strings.TrimSpace(snapshot.CurrentHash)
	plan.PreviousGeneratedHash = strings.TrimSpace(snapshot.PreviousGeneratedHash)
	if plan.CurrentHash == hash {
		plan.Status = FileStatusUnchanged
		plan.Summary = "unchanged: " + candidate.Path
		return plan
	}
	if plan.PreviousGeneratedHash != "" && plan.CurrentHash == plan.PreviousGeneratedHash {
		plan.Status = FileStatusUpdateClean
		plan.Summary = "update-clean: " + candidate.Path
		return plan
	}
	plan.Status = FileStatusConflict
	if plan.PreviousGeneratedHash == "" {
		plan.Reason = "existing file has no generated ownership record"
	} else {
		plan.Reason = "existing file differs from previous generated hash"
	}
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
	}
	for _, prefix := range allowedPrefixes {
		if path == prefix || strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

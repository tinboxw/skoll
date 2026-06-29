package generator

import domaingenerator "github.com/tinboxw/skoll/internal/domain/generator"

type FileStatus string

const (
	FileStatusCreate      FileStatus = "create"
	FileStatusUnchanged   FileStatus = "unchanged"
	FileStatusUpdateClean FileStatus = "update-clean"
	FileStatusConflict    FileStatus = "conflict"
	FileStatusBlocked     FileStatus = "blocked"
)

type DryRunInput struct {
	Spec               *domaingenerator.GeneratorSpec
	BatchID            string
	MigrationTimestamp string
	ExistingFiles      []FileSnapshot
}

type FileSnapshot struct {
	Path                  string
	CurrentHash           string
	PreviousGeneratedHash string
	CurrentContent        string
}

type FilePlan struct {
	Path                  string
	TemplateID            string
	Status                FileStatus
	ContentHash           string
	GeneratedContent      string
	CurrentHash           string
	PreviousGeneratedHash string
	Diff                  string
	Summary               string
	Reason                string
}

type DryRunSummary struct {
	Total       int
	Create      int
	Unchanged   int
	UpdateClean int
	Conflict    int
	Blocked     int
}

type DryRunResult struct {
	BatchID string
	SpecID  string
	Files   []FilePlan
	Summary DryRunSummary
}

package generator

import domaingenerator "github.com/tinboxw/skoll/internal/domain/generator"
import "time"

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
	ActorID            string
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
	CurrentContent        string
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
	BatchID  string
	SpecID   string
	ActorID  string
	SpecHash string
	Files    []FilePlan
	Summary  DryRunSummary
}

type RecordHistoryInput struct {
	DryRun    *DryRunResult
	Spec      *domaingenerator.GeneratorSpec
	ActorID   string
	CreatedAt time.Time
}

type GeneratedFileRecord struct {
	Path             string
	TemplateID       string
	Status           FileStatus
	Hash             string
	GeneratedContent string
	PreviousHash     string
	PreviousContent  string
}

type GenerationHistory struct {
	BatchID      string
	ActorID      string
	SpecID       string
	SpecHash     string
	SpecSnapshot string
	Files        []GeneratedFileRecord
	CreatedAt    time.Time
}

type RollbackAction string

const (
	RollbackActionDelete   RollbackAction = "delete"
	RollbackActionRestore  RollbackAction = "restore"
	RollbackActionNoop     RollbackAction = "noop"
	RollbackActionConflict RollbackAction = "conflict"
	RollbackActionManual   RollbackAction = "manual"
)

type RollbackInput struct {
	BatchID      string
	CurrentFiles []FileSnapshot
}

type RollbackFilePlan struct {
	Path            string
	Action          RollbackAction
	Reason          string
	ExpectedHash    string
	CurrentHash     string
	RestoredHash    string
	RestoredContent string
}

type RollbackPlan struct {
	BatchID   string
	Files     []RollbackFilePlan
	Conflicts []RollbackFilePlan
}

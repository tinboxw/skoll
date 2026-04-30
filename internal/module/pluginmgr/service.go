package pluginmgr

import (
	"errors"
	"sync"
	"time"
)

var ErrPluginNotFound = errors.New("plugin not found")
var ErrInvalidPluginVersion = errors.New("invalid plugin version")
var ErrPluginSignatureInvalid = errors.New("plugin signature verification failed")
var ErrPluginDependencyUnsatisfied = errors.New("plugin dependency precheck failed")
var ErrHookNotFound = errors.New("hook not found")
var ErrHookNamespaceRequired = errors.New("hook namespace is required")
var ErrHookNameRequired = errors.New("hook name is required")
var ErrMarketplaceTrustRootNotFound = errors.New("marketplace trust root not found")
var ErrMarketplaceIndexExpired = errors.New("marketplace index expired")
var ErrMarketplaceIndexSignatureInvalid = errors.New("marketplace index signature invalid")

type VersionCheckResult struct {
	Name            string `json:"name"`
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	UpdateAvailable bool   `json:"update_available"`
}

type Manifest struct {
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	Hooks        []string     `json:"hooks"`
	Enabled      bool         `json:"enabled"`
	PackageURL   string       `json:"package_url,omitempty"`
	PackageHash  string       `json:"package_hash,omitempty"`
	Signature    string       `json:"signature,omitempty"`
	Dependencies []Dependency `json:"dependencies,omitempty"`
	InstalledAt  time.Time    `json:"installed_at"`
}

type Dependency struct {
	Name       string `json:"name"`
	MinVersion string `json:"min_version"`
}

type UpgradeResult struct {
	Name            string `json:"name"`
	PreviousVersion string `json:"previous_version"`
	TargetVersion   string `json:"target_version"`
	Succeeded       bool   `json:"succeeded"`
	RolledBack      bool   `json:"rolled_back"`
	Reason          string `json:"reason,omitempty"`
}

type LifecycleResult struct {
	Name       string `json:"name"`
	Action     string `json:"action"`
	Succeeded  bool   `json:"succeeded"`
	Idempotent bool   `json:"idempotent"`
	Message    string `json:"message,omitempty"`
}

type CompatibilityResult struct {
	Name       string    `json:"name"`
	Version    string    `json:"version"`
	Compatible bool      `json:"compatible"`
	Blockers   []string  `json:"blockers,omitempty"`
	CheckedAt  time.Time `json:"checked_at"`
}

type HookRegistration struct {
	Name          string    `json:"name"`
	Namespace     string    `json:"namespace"`
	Version       string    `json:"version"`
	Enabled       bool      `json:"enabled"`
	Order         int       `json:"order"`
	TimeoutMillis int       `json:"timeout_millis"`
	RetryLimit    int       `json:"retry_limit"`
	DeadLetter    bool      `json:"dead_letter"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type HookExecutionResult struct {
	Name         string   `json:"name"`
	Namespace    string   `json:"namespace"`
	Success      bool     `json:"success"`
	Attempts     int      `json:"attempts"`
	MaxAttempts  int      `json:"max_attempts"`
	DeadLettered bool     `json:"dead_lettered"`
	Diagnostics  []string `json:"diagnostics"`
}

type HookDeadLetterRecord struct {
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	Attempts  int       `json:"attempts"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

type MarketplaceIndexPackage struct {
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	PackageURL   string       `json:"package_url"`
	PackageHash  string       `json:"package_hash"`
	Dependencies []Dependency `json:"dependencies,omitempty"`
}

type MarketplaceIndexSource struct {
	Source         string    `json:"source"`
	SignedBy       string    `json:"signed_by"`
	ExpiresAt      time.Time `json:"expires_at"`
	PackageCount   int       `json:"package_count"`
	LastIngestedAt time.Time `json:"last_ingested_at"`
}

type MarketplaceIndexIngestResult struct {
	Accepted     bool                      `json:"accepted"`
	Source       string                    `json:"source"`
	SignedBy     string                    `json:"signed_by"`
	PackageCount int                       `json:"package_count"`
	Reason       string                    `json:"reason,omitempty"`
	IndexedAt    time.Time                 `json:"indexed_at"`
	Packages     []MarketplaceIndexPackage `json:"packages,omitempty"`
}

type DependencySolveItem struct {
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	Dependencies []Dependency `json:"dependencies,omitempty"`
}

type DependencySolveResult struct {
	Deterministic bool     `json:"deterministic"`
	Resolved      []string `json:"resolved"`
	Conflicts     []string `json:"conflicts,omitempty"`
}

type UpgradeCheckpoint struct {
	Step   string    `json:"step"`
	Status string    `json:"status"`
	Detail string    `json:"detail,omitempty"`
	At     time.Time `json:"at"`
}

type UpgradeTransactionResult struct {
	Name            string              `json:"name"`
	TransactionID   string              `json:"transaction_id"`
	PreviousVersion string              `json:"previous_version"`
	TargetVersion   string              `json:"target_version"`
	Succeeded       bool                `json:"succeeded"`
	RolledBack      bool                `json:"rolled_back"`
	Checkpoints     []UpgradeCheckpoint `json:"checkpoints"`
	Reason          string              `json:"reason,omitempty"`
}

type UpgradeProvenanceRecord struct {
	TransactionID   string    `json:"transaction_id"`
	Name            string    `json:"name"`
	PreviousVersion string    `json:"previous_version"`
	TargetVersion   string    `json:"target_version"`
	Succeeded       bool      `json:"succeeded"`
	RolledBack      bool      `json:"rolled_back"`
	Reason          string    `json:"reason,omitempty"`
	RecordedAt      time.Time `json:"recorded_at"`
}

type Service struct {
	mu          sync.RWMutex
	items       map[string]Manifest
	hooks       map[string]HookRegistration
	deadLetters []HookDeadLetterRecord
	trustRoots  map[string]struct{}
	marketIndex map[string]MarketplaceIndexSource
	provenance  []UpgradeProvenanceRecord
}

func NewService() *Service {
	return &Service{
		items:       make(map[string]Manifest),
		hooks:       make(map[string]HookRegistration),
		trustRoots:  make(map[string]struct{}),
		marketIndex: make(map[string]MarketplaceIndexSource),
	}
}

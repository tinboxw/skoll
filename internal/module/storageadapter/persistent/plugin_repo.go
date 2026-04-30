package persistent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tinboxw/skoll/internal/module/pluginmgr"
	"github.com/tinboxw/skoll/internal/module/storageadapter/contracts"
)

const pluginStateRowID = "plugin"

// PluginStateModel persists the entire plugin-manager state as a single
// JSON document. Plugin counts are typically small enough that snapshotting
// the full aggregate on every mutation is acceptable; this avoids an
// explosion of micro-tables for hooks/trust-roots/marketplace/provenance.
type PluginStateModel struct {
	ID      string `gorm:"primaryKey;size:32"` // always "plugin"
	Payload string `gorm:"type:text;not null"`
}

func (PluginStateModel) TableName() string { return "skoll_plugin_state" }

// PluginRepository implements contracts.PluginRepository using a
// write-through pattern over pluginmgr.NewService(). The service's full
// snapshot is persisted after each mutation.
type PluginRepository struct {
	db    *gorm.DB
	mu    sync.Mutex
	inner *pluginmgr.Service
}

// NewPluginRepository constructs a SQL-backed plugin repository, hydrating
// state from the database. Tables must already be migrated.
func NewPluginRepository(database *gorm.DB) (*PluginRepository, error) {
	if database == nil {
		return nil, fmt.Errorf("persistent: nil db")
	}
	r := &PluginRepository{db: database, inner: pluginmgr.NewService()}
	if err := r.hydrate(); err != nil {
		return nil, err
	}
	return r, nil
}

// Models returns the gorm models managed by this repository.
func (r *PluginRepository) Models() []any {
	return []any{&PluginStateModel{}}
}

func (r *PluginRepository) ctx() context.Context { return context.Background() }

func (r *PluginRepository) hydrate() error {
	var row PluginStateModel
	err := r.db.WithContext(r.ctx()).First(&row, "id = ?", pluginStateRowID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return fmt.Errorf("hydrate plugin state: %w", err)
	}
	var snap pluginmgr.PluginStateSnapshot
	if err := json.Unmarshal([]byte(row.Payload), &snap); err != nil {
		return fmt.Errorf("decode plugin state: %w", err)
	}
	r.inner.ImportSnapshot(snap)
	return nil
}

func (r *PluginRepository) saveAll() {
	snap := r.inner.ExportSnapshot()
	payload, err := json.Marshal(snap)
	if err != nil {
		panic(fmt.Errorf("encode plugin state: %w", err))
	}
	row := PluginStateModel{ID: pluginStateRowID, Payload: string(payload)}
	if err := r.db.WithContext(r.ctx()).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"payload"}),
	}).Create(&row).Error; err != nil {
		panic(fmt.Errorf("persist plugin state: %w", err))
	}
}

// --- contracts.PluginRepository (mutating ops are wrapped) ------------------

func (r *PluginRepository) Install(name, version string, hooks []string) pluginmgr.Manifest {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.inner.Install(name, version, hooks)
	r.saveAll()
	return out
}

func (r *PluginRepository) InstallPackage(name, version, packageURL, packageHash string, hooks []string) (pluginmgr.Manifest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out, err := r.inner.InstallPackage(name, version, packageURL, packageHash, hooks)
	if err != nil {
		return out, err
	}
	r.saveAll()
	return out, nil
}

func (r *PluginRepository) InstallPackageVerified(name, version, packageURL, packageHash, signature string, dependencies []pluginmgr.Dependency, hooks []string) (pluginmgr.Manifest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out, err := r.inner.InstallPackageVerified(name, version, packageURL, packageHash, signature, dependencies, hooks)
	if err != nil {
		return out, err
	}
	r.saveAll()
	return out, nil
}

func (r *PluginRepository) Get(name string) (pluginmgr.Manifest, error) { return r.inner.Get(name) }
func (r *PluginRepository) List() []pluginmgr.Manifest                  { return r.inner.List() }

func (r *PluginRepository) Enable(name string) (pluginmgr.Manifest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out, err := r.inner.Enable(name)
	if err != nil {
		return out, err
	}
	r.saveAll()
	return out, nil
}

func (r *PluginRepository) Disable(name string) (pluginmgr.Manifest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out, err := r.inner.Disable(name)
	if err != nil {
		return out, err
	}
	r.saveAll()
	return out, nil
}

func (r *PluginRepository) CheckVersion(name, latestVersion string) (pluginmgr.VersionCheckResult, error) {
	return r.inner.CheckVersion(name, latestVersion)
}

func (r *PluginRepository) UpgradePackage(name, targetVersion, packageURL, packageHash, signature string, dependencies []pluginmgr.Dependency, hooks []string) (pluginmgr.UpgradeResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out, err := r.inner.UpgradePackage(name, targetVersion, packageURL, packageHash, signature, dependencies, hooks)
	r.saveAll()
	return out, err
}

func (r *PluginRepository) Remove(name string) pluginmgr.LifecycleResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.inner.Remove(name)
	r.saveAll()
	return out
}

func (r *PluginRepository) CheckCompatibility(name, version string, dependencies []pluginmgr.Dependency) pluginmgr.CompatibilityResult {
	return r.inner.CheckCompatibility(name, version, dependencies)
}

func (r *PluginRepository) RegisterHook(name, namespace, version string, order, timeoutMillis, retryLimit int, deadLetter bool) (pluginmgr.HookRegistration, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out, err := r.inner.RegisterHook(name, namespace, version, order, timeoutMillis, retryLimit, deadLetter)
	if err != nil {
		return out, err
	}
	r.saveAll()
	return out, nil
}

func (r *PluginRepository) ListHooks() []pluginmgr.HookRegistration { return r.inner.ListHooks() }

func (r *PluginRepository) SetHookEnabled(name, namespace string, enabled bool) (pluginmgr.HookRegistration, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out, err := r.inner.SetHookEnabled(name, namespace, enabled)
	if err != nil {
		return out, err
	}
	r.saveAll()
	return out, nil
}

func (r *PluginRepository) SetHookOrder(name, namespace string, order int) (pluginmgr.HookRegistration, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out, err := r.inner.SetHookOrder(name, namespace, order)
	if err != nil {
		return out, err
	}
	r.saveAll()
	return out, nil
}

func (r *PluginRepository) SetHookRuntimePolicy(name, namespace string, timeoutMillis, retryLimit int, deadLetter bool) (pluginmgr.HookRegistration, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out, err := r.inner.SetHookRuntimePolicy(name, namespace, timeoutMillis, retryLimit, deadLetter)
	if err != nil {
		return out, err
	}
	r.saveAll()
	return out, nil
}

func (r *PluginRepository) ExecuteHookDiagnostic(name, namespace string, failTimes int) (pluginmgr.HookExecutionResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out, err := r.inner.ExecuteHookDiagnostic(name, namespace, failTimes)
	r.saveAll()
	return out, err
}

func (r *PluginRepository) ListHookDeadLetters() []pluginmgr.HookDeadLetterRecord {
	return r.inner.ListHookDeadLetters()
}

func (r *PluginRepository) SetMarketplaceTrustRoots(roots []string) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.inner.SetMarketplaceTrustRoots(roots)
	r.saveAll()
	return out
}

func (r *PluginRepository) ListMarketplaceTrustRoots() []string {
	return r.inner.ListMarketplaceTrustRoots()
}

func (r *PluginRepository) IngestMarketplaceIndex(source, signedBy, signature string, expiresAt time.Time, packages []pluginmgr.MarketplaceIndexPackage, now time.Time) (pluginmgr.MarketplaceIndexIngestResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out, err := r.inner.IngestMarketplaceIndex(source, signedBy, signature, expiresAt, packages, now)
	if err != nil {
		return out, err
	}
	r.saveAll()
	return out, nil
}

func (r *PluginRepository) ListMarketplaceIndexSources() []pluginmgr.MarketplaceIndexSource {
	return r.inner.ListMarketplaceIndexSources()
}

func (r *PluginRepository) SolveDependencies(items []pluginmgr.DependencySolveItem) pluginmgr.DependencySolveResult {
	return r.inner.SolveDependencies(items)
}

func (r *PluginRepository) UpgradePackageTransactional(transactionID, name, targetVersion, packageURL, packageHash, signature string, dependencies []pluginmgr.Dependency, hooks []string, now time.Time) (pluginmgr.UpgradeTransactionResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out, err := r.inner.UpgradePackageTransactional(transactionID, name, targetVersion, packageURL, packageHash, signature, dependencies, hooks, now)
	r.saveAll()
	return out, err
}

func (r *PluginRepository) ListUpgradeProvenance(limit int) []pluginmgr.UpgradeProvenanceRecord {
	return r.inner.ListUpgradeProvenance(limit)
}

var _ contracts.PluginRepository = (*PluginRepository)(nil)

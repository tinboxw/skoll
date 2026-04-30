// Package persistent contains SQL-backed adapter components. The RBAC repository
// uses a write-through pattern: it owns an in-memory rbac.Service that is
// hydrated from SQL on construction and updated synchronously after each
// mutation. This preserves all existing normalization, snapshot, and
// consistency-check semantics while persisting the resulting state.
package persistent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tinboxw/skoll/internal/module/rbac"
	"github.com/tinboxw/skoll/internal/module/storageadapter/contracts"
)

const (
	rbacKindMenus       = "menus"
	rbacKindAPIs        = "apis"
	rbacKindPolicies    = "policies"
	rbacKindDataScope   = "data_scope"
	rbacKindRoute       = "route"
	rbacKindSnapshotSeq = "snapshot_seq"
)

// RBACStateModel stores per-role aggregate state as JSON keyed by (role_id, kind).
// Keeping aggregates in opaque JSON columns sidesteps cross-dialect type
// concerns (mysql/pg/sqlite) and keeps schema small at the cost of
// in-DB queryability — RBAC reads are served from the in-memory service.
type RBACStateModel struct {
	RoleID  int64  `gorm:"primaryKey;column:role_id"`
	Kind    string `gorm:"primaryKey;column:kind;size:32"`
	Payload string `gorm:"type:text"`
}

func (RBACStateModel) TableName() string { return "skoll_rbac_state" }

// RBACSnapshotModel stores append-only role policy snapshots.
type RBACSnapshotModel struct {
	ID         int64  `gorm:"primaryKey;autoIncrement"`
	RoleID     int64  `gorm:"index:idx_rbac_snapshot_role_version,priority:1,unique"`
	Version    string `gorm:"size:64;index:idx_rbac_snapshot_role_version,priority:2,unique"`
	RulesJSON  string `gorm:"type:text"`
	SequenceID int64  `gorm:"column:seq_id"`
}

func (RBACSnapshotModel) TableName() string { return "skoll_rbac_role_snapshots" }

// RBACRepository is a write-through SQL-backed implementation of contracts.RBACRepository.
type RBACRepository struct {
	db    *gorm.DB
	mu    sync.Mutex
	inner *rbac.Service
}

// NewRBACRepository constructs the repository, ensures schema, and hydrates
// the in-memory service from any previously-persisted state.
func NewRBACRepository(db *gorm.DB) (*RBACRepository, error) {
	r := &RBACRepository{db: db, inner: rbac.NewService()}
	if err := r.hydrate(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *RBACRepository) ctx() context.Context { return context.Background() }

// Models returns gorm models owned by this repo for migration registration.
func (r *RBACRepository) Models() []any {
	return []any{&RBACStateModel{}, &RBACSnapshotModel{}}
}

func (r *RBACRepository) hydrate() error {
	var stateRows []RBACStateModel
	if err := r.db.WithContext(r.ctx()).Find(&stateRows).Error; err != nil {
		return fmt.Errorf("rbac hydrate state: %w", err)
	}

	type aggregate struct {
		menus       []int64
		apis        []string
		policies    []rbac.PolicyRule
		dataScope   rbac.DataScope
		route       rbac.RoutePermissionContract
		hasMenus    bool
		hasAPIs     bool
		hasPolicies bool
		hasScope    bool
		hasRoute    bool
		seq         int64
	}
	roles := map[int64]*aggregate{}
	getAgg := func(id int64) *aggregate {
		if a, ok := roles[id]; ok {
			return a
		}
		a := &aggregate{}
		roles[id] = a
		return a
	}

	for _, row := range stateRows {
		agg := getAgg(row.RoleID)
		switch row.Kind {
		case rbacKindMenus:
			if err := json.Unmarshal([]byte(row.Payload), &agg.menus); err != nil {
				return fmt.Errorf("rbac hydrate menus role=%d: %w", row.RoleID, err)
			}
			agg.hasMenus = true
		case rbacKindAPIs:
			if err := json.Unmarshal([]byte(row.Payload), &agg.apis); err != nil {
				return fmt.Errorf("rbac hydrate apis role=%d: %w", row.RoleID, err)
			}
			agg.hasAPIs = true
		case rbacKindPolicies:
			if err := json.Unmarshal([]byte(row.Payload), &agg.policies); err != nil {
				return fmt.Errorf("rbac hydrate policies role=%d: %w", row.RoleID, err)
			}
			agg.hasPolicies = true
		case rbacKindDataScope:
			if err := json.Unmarshal([]byte(row.Payload), &agg.dataScope); err != nil {
				return fmt.Errorf("rbac hydrate data scope role=%d: %w", row.RoleID, err)
			}
			agg.hasScope = true
		case rbacKindRoute:
			if err := json.Unmarshal([]byte(row.Payload), &agg.route); err != nil {
				return fmt.Errorf("rbac hydrate route role=%d: %w", row.RoleID, err)
			}
			agg.hasRoute = true
		case rbacKindSnapshotSeq:
			if err := json.Unmarshal([]byte(row.Payload), &agg.seq); err != nil {
				return fmt.Errorf("rbac hydrate seq role=%d: %w", row.RoleID, err)
			}
		}
	}

	for roleID, agg := range roles {
		r.inner.ImportRoleAggregate(roleID, agg.menus, agg.apis, agg.policies, agg.dataScope, agg.route)
	}

	var snapshots []RBACSnapshotModel
	if err := r.db.WithContext(r.ctx()).Order("role_id ASC, seq_id ASC").Find(&snapshots).Error; err != nil {
		return fmt.Errorf("rbac hydrate snapshots: %w", err)
	}
	bucket := map[int64][]rbac.PolicySnapshot{}
	for _, sn := range snapshots {
		var rules []rbac.PolicyRule
		if err := json.Unmarshal([]byte(sn.RulesJSON), &rules); err != nil {
			return fmt.Errorf("rbac hydrate snapshot rules id=%d: %w", sn.ID, err)
		}
		bucket[sn.RoleID] = append(bucket[sn.RoleID], rbac.PolicySnapshot{RoleID: sn.RoleID, Version: sn.Version, Rules: rules})
	}
	for roleID, list := range bucket {
		seq := int64(0)
		if agg, ok := roles[roleID]; ok {
			seq = agg.seq
		}
		r.inner.ImportRolePolicySnapshots(roleID, list, seq)
	}

	return nil
}

func (r *RBACRepository) saveState(roleID int64, kind string, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("rbac persist %s role=%d: marshal: %w", kind, roleID, err)
	}
	row := RBACStateModel{RoleID: roleID, Kind: kind, Payload: string(payload)}
	return r.db.WithContext(r.ctx()).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "role_id"}, {Name: "kind"}},
		DoUpdates: clause.AssignmentColumns([]string{"payload"}),
	}).Create(&row).Error
}

// SetRoleMenus delegates to the in-memory service then persists the
// normalized result. Returns the normalized list.
func (r *RBACRepository) SetRoleMenus(roleID int64, menuIDs []int64) []int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.inner.SetRoleMenus(roleID, menuIDs)
	if err := r.saveState(roleID, rbacKindMenus, out); err != nil {
		panic(err)
	}
	return out
}

// GetRoleMenus reads from the in-memory service.
func (r *RBACRepository) GetRoleMenus(roleID int64) []int64 {
	return r.inner.GetRoleMenus(roleID)
}

// SetRoleAPIs delegates and persists.
func (r *RBACRepository) SetRoleAPIs(roleID int64, apis []string) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.inner.SetRoleAPIs(roleID, apis)
	if err := r.saveState(roleID, rbacKindAPIs, out); err != nil {
		panic(err)
	}
	return out
}

// GetRoleAPIs reads from the in-memory service.
func (r *RBACRepository) GetRoleAPIs(roleID int64) []string {
	return r.inner.GetRoleAPIs(roleID)
}

// SetRolePolicies delegates and persists.
func (r *RBACRepository) SetRolePolicies(roleID int64, rules []rbac.PolicyRule) []rbac.PolicyRule {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.inner.SetRolePolicies(roleID, rules)
	if err := r.saveState(roleID, rbacKindPolicies, out); err != nil {
		panic(err)
	}
	return out
}

// GetRolePolicies reads from the in-memory service.
func (r *RBACRepository) GetRolePolicies(roleID int64) []rbac.PolicyRule {
	return r.inner.GetRolePolicies(roleID)
}

// CreateRolePolicySnapshot creates and persists a new snapshot.
func (r *RBACRepository) CreateRolePolicySnapshot(roleID int64) rbac.PolicySnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	snap := r.inner.CreateRolePolicySnapshot(roleID)
	rulesJSON, err := json.Marshal(snap.Rules)
	if err != nil {
		panic(err)
	}
	row := RBACSnapshotModel{
		RoleID:    snap.RoleID,
		Version:   snap.Version,
		RulesJSON: string(rulesJSON),
	}
	if err := r.db.WithContext(r.ctx()).Create(&row).Error; err != nil {
		panic(err)
	}
	if err := r.db.WithContext(r.ctx()).Model(&RBACSnapshotModel{}).Where("id = ?", row.ID).
		Update("seq_id", row.ID).Error; err != nil {
		panic(err)
	}
	// Persist the latest sequence (last-used) so hydration restores it.
	all := r.inner.ListRolePolicySnapshots(roleID)
	if err := r.saveState(roleID, rbacKindSnapshotSeq, int64(len(all))); err != nil {
		panic(err)
	}
	return snap
}

// ListRolePolicySnapshots reads from the in-memory service.
func (r *RBACRepository) ListRolePolicySnapshots(roleID int64) []rbac.PolicySnapshot {
	return r.inner.ListRolePolicySnapshots(roleID)
}

// RollbackRolePolicies delegates and persists the resulting policies.
func (r *RBACRepository) RollbackRolePolicies(roleID int64, version string) ([]rbac.PolicyRule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rules, err := r.inner.RollbackRolePolicies(roleID, version)
	if err != nil {
		if errors.Is(err, rbac.ErrPolicySnapshotNotFound) {
			return nil, err
		}
		return nil, err
	}
	if err := r.saveState(roleID, rbacKindPolicies, rules); err != nil {
		panic(err)
	}
	return rules, nil
}

// PermissionBundle reads from the in-memory service.
func (r *RBACRepository) PermissionBundle(roleID int64) rbac.PermissionBundle {
	return r.inner.PermissionBundle(roleID)
}

// SetRoleDataScope delegates and persists.
func (r *RBACRepository) SetRoleDataScope(roleID int64, scope rbac.DataScope) rbac.DataScope {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.inner.SetRoleDataScope(roleID, scope)
	if err := r.saveState(roleID, rbacKindDataScope, out); err != nil {
		panic(err)
	}
	return out
}

// GetRoleDataScope reads from the in-memory service.
func (r *RBACRepository) GetRoleDataScope(roleID int64) rbac.DataScope {
	return r.inner.GetRoleDataScope(roleID)
}

// SetRoleRoutePermissions delegates and persists.
func (r *RBACRepository) SetRoleRoutePermissions(roleID int64, version string, items []rbac.RoutePermissionItem) rbac.RoutePermissionContract {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.inner.SetRoleRoutePermissions(roleID, version, items)
	if err := r.saveState(roleID, rbacKindRoute, out); err != nil {
		panic(err)
	}
	return out
}

// GetRoleRoutePermissions reads from the in-memory service.
func (r *RBACRepository) GetRoleRoutePermissions(roleID int64) rbac.RoutePermissionContract {
	return r.inner.GetRoleRoutePermissions(roleID)
}

// CheckRoleRoutePermissionConsistency reads from the in-memory service.
func (r *RBACRepository) CheckRoleRoutePermissionConsistency(roleID int64) rbac.RoutePermissionConsistency {
	return r.inner.CheckRoleRoutePermissionConsistency(roleID)
}

// Compile-time interface check.
var _ contracts.RBACRepository = (*RBACRepository)(nil)

package rbac

// ImportRoleAggregate replaces the in-memory aggregate state for the role with
// the supplied (already-persisted) values. Inputs are re-normalized for
// safety so callers do not need to repeat normalization. Used by SQL-backed
// adapters during hydration on boot.
func (s *Service) ImportRoleAggregate(roleID int64, menus []int64, apis []string, policies []PolicyRule, scope DataScope, route RoutePermissionContract) {
	if roleID <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if m := normalizeInt64List(menus); m != nil {
		s.roleMenu[roleID] = m
	} else {
		delete(s.roleMenu, roleID)
	}
	if a := normalizeStringList(apis); a != nil {
		s.roleAPI[roleID] = a
	} else {
		delete(s.roleAPI, roleID)
	}
	if p := normalizePolicyRules(policies); p != nil {
		s.policies[roleID] = p
	} else {
		delete(s.policies, roleID)
	}
	if scope.RequireOwnerMatch || len(scope.TenantIDs) > 0 || len(scope.CrossTenantAdminAllow) > 0 {
		s.data[roleID] = normalizeDataScope(scope)
	} else {
		delete(s.data, roleID)
	}
	if route.Version != "" || len(route.Items) > 0 {
		s.route[roleID] = normalizeRouteContract(route.Version, route.Items)
	} else {
		delete(s.route, roleID)
	}
}

// ImportRolePolicySnapshots restores previously-persisted snapshot history
// and the next-version sequence counter. Snapshots are stored as-provided
// (callers are expected to have persisted normalized rules).
func (s *Service) ImportRolePolicySnapshots(roleID int64, snapshots []PolicySnapshot, nextSeq int64) {
	if roleID <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(snapshots) == 0 {
		delete(s.snapshots, roleID)
	} else {
		clones := make([]PolicySnapshot, len(snapshots))
		for i, sn := range snapshots {
			clones[i] = clonePolicySnapshot(sn)
		}
		s.snapshots[roleID] = clones
	}
	if nextSeq > 0 {
		s.snapshotSeq[roleID] = nextSeq
	} else {
		delete(s.snapshotSeq, roleID)
	}
}

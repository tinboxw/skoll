package pluginmgr

// PluginStateSnapshot bundles the persistent state of pluginmgr.Service.
type PluginStateSnapshot struct {
	Items       map[string]Manifest
	Hooks       map[string]HookRegistration
	DeadLetters []HookDeadLetterRecord
	TrustRoots  []string
	MarketIndex map[string]MarketplaceIndexSource
	Provenance  []UpgradeProvenanceRecord
}

// ImportSnapshot replaces in-memory state with the supplied snapshot.
func (s *Service) ImportSnapshot(snap PluginStateSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items = make(map[string]Manifest, len(snap.Items))
	for k, v := range snap.Items {
		s.items[k] = v
	}

	s.hooks = make(map[string]HookRegistration, len(snap.Hooks))
	for k, v := range snap.Hooks {
		s.hooks[k] = v
	}

	s.deadLetters = append([]HookDeadLetterRecord(nil), snap.DeadLetters...)

	s.trustRoots = make(map[string]struct{}, len(snap.TrustRoots))
	for _, root := range snap.TrustRoots {
		s.trustRoots[root] = struct{}{}
	}

	s.marketIndex = make(map[string]MarketplaceIndexSource, len(snap.MarketIndex))
	for k, v := range snap.MarketIndex {
		s.marketIndex[k] = v
	}

	s.provenance = append([]UpgradeProvenanceRecord(nil), snap.Provenance...)
}

// ExportSnapshot returns a copy of the persistent state for SQL adapters.
func (s *Service) ExportSnapshot() PluginStateSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := PluginStateSnapshot{
		Items:       make(map[string]Manifest, len(s.items)),
		Hooks:       make(map[string]HookRegistration, len(s.hooks)),
		DeadLetters: append([]HookDeadLetterRecord(nil), s.deadLetters...),
		MarketIndex: make(map[string]MarketplaceIndexSource, len(s.marketIndex)),
		Provenance:  append([]UpgradeProvenanceRecord(nil), s.provenance...),
	}
	for k, v := range s.items {
		out.Items[k] = v
	}
	for k, v := range s.hooks {
		out.Hooks[k] = v
	}
	out.TrustRoots = make([]string, 0, len(s.trustRoots))
	for r := range s.trustRoots {
		out.TrustRoots = append(out.TrustRoots, r)
	}
	for k, v := range s.marketIndex {
		out.MarketIndex[k] = v
	}
	return out
}

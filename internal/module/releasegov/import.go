package releasegov

// ImportEvidence rebuilds the evidence map from a previously persisted set
// of Evidence rows. The order within each milestone is preserved as supplied.
// Existing in-memory state is replaced.
func (s *Service) ImportEvidence(rows []Evidence) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.evidence = make(map[string][]Evidence, len(rows))
	for _, ev := range rows {
		key := ev.Milestone
		s.evidence[key] = append(s.evidence[key], ev)
	}
}

// ExportEvidenceFlat returns all evidence rows across all milestones in a
// single slice. Order within each milestone is preserved.
func (s *Service) ExportEvidenceFlat() []Evidence {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Evidence, 0)
	for _, list := range s.evidence {
		out = append(out, list...)
	}
	return out
}

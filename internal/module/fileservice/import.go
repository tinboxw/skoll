package fileservice

// ImportFiles rebuilds the in-memory file index from a previously persisted
// catalog. Existing state is replaced; nextID is advanced past every imported
// row.
func (s *Service) ImportFiles(rows []File) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = make(map[int64]File, len(rows))
	for _, f := range rows {
		s.items[f.ID] = f
		if f.ID >= s.nextID {
			s.nextID = f.ID + 1
		}
	}
}

// ExportFiles returns a copy of every file metadata record currently tracked
// by the service. Used by SQL-backed adapters to capture state.
func (s *Service) ExportFiles() []File {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]File, 0, len(s.items))
	for _, item := range s.items {
		out = append(out, item)
	}
	return out
}

// ExportNextID returns the current next-id counter for persistence.
func (s *Service) ExportNextID() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nextID
}

package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	domainfile "github.com/tinboxw/skoll/internal/domain/file"
	"github.com/tinboxw/skoll/internal/domain/shared"
	filerepo "github.com/tinboxw/skoll/internal/repository/file"
)

type FileStore struct {
	mu     sync.RWMutex
	items  map[shared.ID]domainfile.FileObject
	keyIDs map[string]shared.ID
}

func NewFileStore() *FileStore {
	return &FileStore{
		items:  make(map[shared.ID]domainfile.FileObject),
		keyIDs: make(map[string]shared.ID),
	}
}

func (s *FileStore) Upsert(_ context.Context, object *domainfile.FileObject) error {
	if object == nil {
		return fmt.Errorf("file object is required")
	}
	item := cloneFileObject(*object)
	key := domainfile.NormalizeKey(item.Key)
	if err := domainfile.ValidateFileObjectInput(domainfile.FileObjectInput{
		ID:            item.ID,
		Key:           item.Key,
		Name:          item.Name,
		Size:          item.Size,
		MIME:          item.MIME,
		Hash:          item.Hash,
		Owner:         item.Owner,
		Visibility:    item.Visibility,
		StorageDriver: item.StorageDriver,
		Status:        item.Status,
		Source:        item.Source,
		Metadata:      item.Metadata,
		CreatedAt:     item.Meta.CreatedAt,
		UpdatedAt:     item.Meta.UpdatedAt,
	}); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if existingID, ok := s.keyIDs[key]; ok && existingID != item.ID {
		return fmt.Errorf("file object key already exists")
	}
	if old, ok := s.items[item.ID]; ok && old.Key != key {
		delete(s.keyIDs, old.Key)
	}
	item.Key = key
	s.items[item.ID] = item
	s.keyIDs[key] = item.ID
	return nil
}

func (s *FileStore) Get(_ context.Context, id shared.ID) (*domainfile.FileObject, error) {
	if id.IsZero() {
		return nil, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	item = cloneFileObject(item)
	return &item, nil
}

func (s *FileStore) GetByKey(_ context.Context, key string) (*domainfile.FileObject, error) {
	key = domainfile.NormalizeKey(key)
	if key == "" {
		return nil, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.keyIDs[key]
	if !ok {
		return nil, nil
	}
	item := cloneFileObject(s.items[id])
	return &item, nil
}

func (s *FileStore) List(_ context.Context, filter filerepo.ListFilter, offset, limit int) ([]domainfile.FileObject, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]domainfile.FileObject, 0, len(s.items))
	for _, item := range s.items {
		if matchesFileFilter(item, filter) {
			out = append(out, cloneFileObject(item))
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Key < out[j].Key
	})
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = len(out)
	}
	if offset > len(out) {
		return []domainfile.FileObject{}, nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return append([]domainfile.FileObject(nil), out[offset:end]...), nil
}

func (s *FileStore) SetStatus(_ context.Context, id shared.ID, status domainfile.Status, now time.Time) error {
	if id.IsZero() {
		return nil
	}
	if err := domainfile.ValidateStatus(status); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if !ok {
		return nil
	}
	item.Status = status
	item.Meta.Touch(now)
	s.items[id] = item
	return nil
}

func (s *FileStore) Delete(_ context.Context, id shared.ID) error {
	if id.IsZero() {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if !ok {
		return nil
	}
	delete(s.items, id)
	delete(s.keyIDs, item.Key)
	return nil
}

func matchesFileFilter(item domainfile.FileObject, filter filerepo.ListFilter) bool {
	if strings.TrimSpace(filter.OwnerType) != "" && item.Owner.Type != strings.TrimSpace(strings.ToLower(filter.OwnerType)) {
		return false
	}
	if !filter.OwnerID.IsZero() && item.Owner.ID != filter.OwnerID {
		return false
	}
	if filter.Visibility != "" && item.Visibility != filter.Visibility {
		return false
	}
	if strings.TrimSpace(filter.StorageDriver) != "" && item.StorageDriver != strings.TrimSpace(strings.ToLower(filter.StorageDriver)) {
		return false
	}
	if filter.Status != "" && item.Status != filter.Status {
		return false
	}
	if strings.TrimSpace(filter.SourceModule) != "" && item.Source.Module != strings.TrimSpace(strings.ToLower(filter.SourceModule)) {
		return false
	}
	if strings.TrimSpace(filter.SourcePluginID) != "" && item.Source.PluginID != strings.TrimSpace(strings.ToLower(filter.SourcePluginID)) {
		return false
	}
	return true
}

func cloneFileObject(item domainfile.FileObject) domainfile.FileObject {
	item.Metadata = domainfile.NormalizeObjectMetadata(item.Metadata)
	return item
}

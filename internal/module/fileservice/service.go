package fileservice

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var ErrFileNotFound = errors.New("file not found")

type File struct {
	ID         int64
	Name       string
	Size       int64
	StorageKey string
	CreatedAt  time.Time
}

type Backend interface {
	Save(name string, content []byte) (string, error)
	Open(storageKey string) ([]byte, error)
}

type LocalBackend struct {
	root string
}

func NewLocalBackend(root string) (*LocalBackend, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("root directory is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &LocalBackend{root: root}, nil
}

func (b *LocalBackend) Save(name string, content []byte) (string, error) {
	ext := filepath.Ext(strings.TrimSpace(name))
	hash := sha1.Sum(append([]byte(time.Now().UTC().Format(time.RFC3339Nano)), content...))
	storageKey := hex.EncodeToString(hash[:]) + ext
	abs := filepath.Join(b.root, storageKey)
	if err := os.WriteFile(abs, content, 0o644); err != nil {
		return "", err
	}
	return storageKey, nil
}

func (b *LocalBackend) Open(storageKey string) ([]byte, error) {
	storageKey = filepath.Clean(strings.TrimSpace(storageKey))
	if storageKey == "." || storageKey == "" {
		return nil, fmt.Errorf("invalid storage key")
	}
	return os.ReadFile(filepath.Join(b.root, storageKey))
}

type Service struct {
	mu      sync.RWMutex
	nextID  int64
	backend Backend
	items   map[int64]File
}

func NewService(backend Backend) *Service {
	return &Service{nextID: 1, backend: backend, items: make(map[int64]File)}
}

func (s *Service) Upload(name string, content []byte) (File, error) {
	if strings.TrimSpace(name) == "" {
		return File{}, fmt.Errorf("file name is required")
	}
	if len(content) == 0 {
		return File{}, fmt.Errorf("file content is required")
	}
	storageKey, err := s.backend.Save(name, content)
	if err != nil {
		return File{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	item := File{
		ID:         s.nextID,
		Name:       name,
		Size:       int64(len(content)),
		StorageKey: storageKey,
		CreatedAt:  time.Now().UTC(),
	}
	s.nextID++
	s.items[item.ID] = item
	return item, nil
}

func (s *Service) Get(id int64) (File, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[id]
	if !ok {
		return File{}, ErrFileNotFound
	}
	return item, nil
}

func (s *Service) List() []File {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]File, 0, len(s.items))
	for _, item := range s.items {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Service) Download(id int64) (File, []byte, error) {
	item, err := s.Get(id)
	if err != nil {
		return File{}, nil, err
	}
	content, err := s.backend.Open(item.StorageKey)
	if err != nil {
		return File{}, nil, err
	}
	return item, content, nil
}

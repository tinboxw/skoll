package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Store struct {
	mu    sync.RWMutex
	path  string
	state State
}

func OpenStore(dataDir string) (*Store, error) {
	dataDir = strings.TrimSpace(dataDir)
	if dataDir == "" {
		return nil, errors.New("SKOLL_PLUGIN_DATA_DIR is required")
	}
	abs, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, fmt.Errorf("resolve plugin data directory: %w", err)
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return nil, fmt.Errorf("create plugin data directory: %w", err)
	}
	store := &Store{path: filepath.Join(abs, "equipment-maintenance.json"), state: newState()}
	raw, err := os.ReadFile(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read plugin data: %w", err)
	}
	if err := json.Unmarshal(raw, &store.state); err != nil {
		return nil, fmt.Errorf("decode plugin data: %w", err)
	}
	store.state.normalize()
	return store, nil
}

func (s *Store) View(fn func(State) error) error {
	if s == nil || fn == nil {
		return errors.New("plugin store view is required")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	copy, err := cloneState(s.state)
	if err != nil {
		return err
	}
	return fn(copy)
}

func (s *Store) Update(fn func(*State) error) error {
	if s == nil || fn == nil {
		return errors.New("plugin store update is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	next, err := cloneState(s.state)
	if err != nil {
		return err
	}
	if err := fn(&next); err != nil {
		return err
	}
	next.normalize()
	if err := s.persist(next); err != nil {
		return err
	}
	s.state = next
	return nil
}

func (s *Store) persist(state State) error {
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode plugin data: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(s.path), ".equipment-maintenance-*.json")
	if err != nil {
		return fmt.Errorf("create plugin data transaction: %w", err)
	}
	tempPath := temp.Name()
	committed := false
	defer func() {
		_ = temp.Close()
		if !committed {
			_ = os.Remove(tempPath)
		}
	}()
	if err := temp.Chmod(0o600); err != nil {
		return err
	}
	if _, err := temp.Write(raw); err != nil {
		return fmt.Errorf("write plugin data transaction: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("sync plugin data transaction: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close plugin data transaction: %w", err)
	}
	if err := os.Rename(tempPath, s.path); err != nil {
		backup, backupErr := os.CreateTemp(filepath.Dir(s.path), ".equipment-maintenance-previous-*.json")
		if backupErr != nil {
			return fmt.Errorf("prepare plugin data replacement: %w", errors.Join(err, backupErr))
		}
		backupPath := backup.Name()
		if closeErr := backup.Close(); closeErr != nil {
			return fmt.Errorf("prepare plugin data replacement: %w", closeErr)
		}
		if removeErr := os.Remove(backupPath); removeErr != nil {
			return fmt.Errorf("prepare plugin data replacement: %w", removeErr)
		}
		if backupErr := os.Rename(s.path, backupPath); backupErr != nil && !errors.Is(backupErr, os.ErrNotExist) {
			return fmt.Errorf("backup plugin data: %w", errors.Join(err, backupErr))
		}
		if retryErr := os.Rename(tempPath, s.path); retryErr != nil {
			if restoreErr := os.Rename(backupPath, s.path); restoreErr != nil && !errors.Is(restoreErr, os.ErrNotExist) {
				return fmt.Errorf("replace and restore plugin data: %w", errors.Join(retryErr, restoreErr))
			}
			return fmt.Errorf("replace plugin data: %w", retryErr)
		}
		if removeErr := os.Remove(backupPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return fmt.Errorf("remove replaced plugin data backup: %w", removeErr)
		}
	}
	committed = true
	return nil
}

func cloneState(state State) (State, error) {
	raw, err := json.Marshal(state)
	if err != nil {
		return State{}, err
	}
	var copy State
	if err := json.Unmarshal(raw, &copy); err != nil {
		return State{}, err
	}
	copy.normalize()
	return copy, nil
}

func nextID(state *State, prefix string) string {
	state.Sequence++
	return fmt.Sprintf("%s-%06d", prefix, state.Sequence)
}

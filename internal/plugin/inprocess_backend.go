package plugin

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// InProcessBackendFactory creates a backend owned by an enabled plugin.
// The registry calls factories lazily and discards instances on lifecycle changes.
type InProcessBackendFactory func() (http.Handler, error)

type inProcessBackendEntry struct {
	mu      sync.Mutex
	factory InProcessBackendFactory
	handler http.Handler
}

type InProcessBackendRegistry struct {
	mu      sync.RWMutex
	entries map[string]*inProcessBackendEntry
}

func NewInProcessBackendRegistry() *InProcessBackendRegistry {
	return &InProcessBackendRegistry{entries: make(map[string]*inProcessBackendEntry)}
}

func (r *InProcessBackendRegistry) Register(pluginID string, factory InProcessBackendFactory) error {
	if r == nil {
		return fmt.Errorf("in-process plugin backend registry is nil")
	}
	pluginID = strings.TrimSpace(pluginID)
	if pluginID == "" {
		return fmt.Errorf("plugin id is required")
	}
	if factory == nil {
		return fmt.Errorf("plugin backend factory is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.entries[pluginID]; exists {
		return fmt.Errorf("in-process plugin backend already registered: %s", pluginID)
	}
	r.entries[pluginID] = &inProcessBackendEntry{factory: factory}
	return nil
}

func (r *InProcessBackendRegistry) ServeHTTP(pluginID string, w http.ResponseWriter, req *http.Request) (bool, error) {
	if r == nil {
		return false, nil
	}
	r.mu.RLock()
	entry := r.entries[strings.TrimSpace(pluginID)]
	r.mu.RUnlock()
	if entry == nil {
		return false, nil
	}

	entry.mu.Lock()
	if entry.handler == nil {
		handler, err := entry.factory()
		if err != nil {
			entry.mu.Unlock()
			return true, err
		}
		if handler == nil {
			entry.mu.Unlock()
			return true, fmt.Errorf("in-process plugin backend factory returned nil handler")
		}
		entry.handler = handler
	}
	handler := entry.handler
	entry.mu.Unlock()

	handler.ServeHTTP(w, req)
	return true, nil
}

func (r *InProcessBackendRegistry) Reset(pluginID string) error {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	entry := r.entries[strings.TrimSpace(pluginID)]
	r.mu.RUnlock()
	if entry == nil {
		return nil
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.handler == nil {
		return nil
	}
	var err error
	if closer, ok := entry.handler.(io.Closer); ok {
		err = closer.Close()
	}
	entry.handler = nil
	return err
}

func (r *InProcessBackendRegistry) Close() error {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	ids := make([]string, 0, len(r.entries))
	for pluginID := range r.entries {
		ids = append(ids, pluginID)
	}
	r.mu.RUnlock()

	var firstErr error
	for _, pluginID := range ids {
		if err := r.Reset(pluginID); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

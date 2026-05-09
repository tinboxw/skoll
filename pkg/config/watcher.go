package config

import "sync"

// Watcher provides in-process config change callbacks.
type Watcher struct {
	mu        sync.Mutex
	handlers  []func(AppConfig)
	lastValue AppConfig
}

func NewWatcher(initial AppConfig) *Watcher {
	return &Watcher{lastValue: initial}
}

func (w *Watcher) Register(handler func(AppConfig)) {
	if handler == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.handlers = append(w.handlers, handler)
}

// Notify broadcasts the latest config value to all callbacks.
func (w *Watcher) Notify(next AppConfig) {
	w.mu.Lock()
	handlers := append([]func(AppConfig){}, w.handlers...)
	w.lastValue = next
	w.mu.Unlock()

	for _, h := range handlers {
		h(next)
	}
}

func (w *Watcher) LastValue() AppConfig {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.lastValue
}

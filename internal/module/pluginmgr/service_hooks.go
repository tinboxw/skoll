package pluginmgr

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func (s *Service) RegisterHook(name, namespace, version string, order, timeoutMillis, retryLimit int, deadLetter bool) (HookRegistration, error) {
	name = strings.TrimSpace(name)
	namespace = strings.TrimSpace(namespace)
	version = strings.TrimSpace(version)
	if name == "" {
		return HookRegistration{}, ErrHookNameRequired
	}
	if namespace == "" {
		return HookRegistration{}, ErrHookNamespaceRequired
	}
	if version == "" {
		version = "1.0.0"
	}
	if !isValidVersion(version) {
		return HookRegistration{}, ErrInvalidPluginVersion
	}
	if order < 0 {
		order = 0
	}
	if timeoutMillis <= 0 {
		timeoutMillis = 500
	}
	if timeoutMillis > 60000 {
		timeoutMillis = 60000
	}
	if retryLimit < 0 {
		retryLimit = 0
	}
	if retryLimit > 10 {
		retryLimit = 10
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	item := HookRegistration{
		Name:          name,
		Namespace:     namespace,
		Version:       version,
		Enabled:       true,
		Order:         order,
		TimeoutMillis: timeoutMillis,
		RetryLimit:    retryLimit,
		DeadLetter:    deadLetter,
		UpdatedAt:     time.Now().UTC(),
	}
	s.hooks[hookKey(namespace, name)] = item
	return item, nil
}

func (s *Service) ListHooks() []HookRegistration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]HookRegistration, 0, len(s.hooks))
	for _, item := range s.hooks {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Namespace == out[j].Namespace {
			if out[i].Order == out[j].Order {
				return out[i].Name < out[j].Name
			}
			return out[i].Order < out[j].Order
		}
		return out[i].Namespace < out[j].Namespace
	})
	return out
}

func (s *Service) SetHookEnabled(name, namespace string, enabled bool) (HookRegistration, error) {
	return s.updateHook(name, namespace, func(item HookRegistration) HookRegistration {
		item.Enabled = enabled
		return item
	})
}

func (s *Service) SetHookOrder(name, namespace string, order int) (HookRegistration, error) {
	if order < 0 {
		order = 0
	}
	return s.updateHook(name, namespace, func(item HookRegistration) HookRegistration {
		item.Order = order
		return item
	})
}

func (s *Service) SetHookRuntimePolicy(name, namespace string, timeoutMillis, retryLimit int, deadLetter bool) (HookRegistration, error) {
	if timeoutMillis <= 0 {
		timeoutMillis = 500
	}
	if timeoutMillis > 60000 {
		timeoutMillis = 60000
	}
	if retryLimit < 0 {
		retryLimit = 0
	}
	if retryLimit > 10 {
		retryLimit = 10
	}
	return s.updateHook(name, namespace, func(item HookRegistration) HookRegistration {
		item.TimeoutMillis = timeoutMillis
		item.RetryLimit = retryLimit
		item.DeadLetter = deadLetter
		return item
	})
}

func (s *Service) ExecuteHookDiagnostic(name, namespace string, failTimes int) (HookExecutionResult, error) {
	name = strings.TrimSpace(name)
	namespace = strings.TrimSpace(namespace)
	if name == "" {
		return HookExecutionResult{}, ErrHookNameRequired
	}
	if namespace == "" {
		return HookExecutionResult{}, ErrHookNamespaceRequired
	}
	if failTimes < 0 {
		failTimes = 0
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.hooks[hookKey(namespace, name)]
	if !ok {
		return HookExecutionResult{}, ErrHookNotFound
	}

	maxAttempts := item.RetryLimit + 1
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	diag := make([]string, 0, maxAttempts+1)
	result := HookExecutionResult{Name: name, Namespace: namespace, MaxAttempts: maxAttempts}
	if !item.Enabled {
		result.Diagnostics = []string{"hook disabled"}
		return result, nil
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result.Attempts = attempt
		if attempt <= failTimes {
			diag = append(diag, fmt.Sprintf("attempt %d failed", attempt))
			continue
		}
		result.Success = true
		diag = append(diag, fmt.Sprintf("attempt %d succeeded", attempt))
		break
	}

	if !result.Success {
		diag = append(diag, "max attempts exhausted")
		if item.DeadLetter {
			result.DeadLettered = true
			s.deadLetters = append(s.deadLetters, HookDeadLetterRecord{
				Name:      name,
				Namespace: namespace,
				Attempts:  result.Attempts,
				Reason:    "max attempts exhausted",
				CreatedAt: time.Now().UTC(),
			})
		}
	}
	result.Diagnostics = diag
	return result, nil
}

func (s *Service) ListHookDeadLetters() []HookDeadLetterRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.deadLetters) == 0 {
		return nil
	}
	out := make([]HookDeadLetterRecord, len(s.deadLetters))
	copy(out, s.deadLetters)
	return out
}

func (s *Service) updateHook(name, namespace string, fn func(HookRegistration) HookRegistration) (HookRegistration, error) {
	name = strings.TrimSpace(name)
	namespace = strings.TrimSpace(namespace)
	if name == "" {
		return HookRegistration{}, ErrHookNameRequired
	}
	if namespace == "" {
		return HookRegistration{}, ErrHookNamespaceRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	key := hookKey(namespace, name)
	item, ok := s.hooks[key]
	if !ok {
		return HookRegistration{}, ErrHookNotFound
	}
	item = fn(item)
	item.UpdatedAt = time.Now().UTC()
	s.hooks[key] = item
	return item, nil
}

func hookKey(namespace, name string) string {
	return strings.TrimSpace(namespace) + "::" + strings.TrimSpace(name)
}

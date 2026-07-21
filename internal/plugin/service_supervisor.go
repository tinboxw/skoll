package plugin

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type ServiceState string

const (
	ServiceStateNotApplicable ServiceState = "not_applicable"
	ServiceStateStarting      ServiceState = "starting"
	ServiceStateReady         ServiceState = "ready"
	ServiceStateStopping      ServiceState = "stopping"
	ServiceStateStopped       ServiceState = "stopped"
	ServiceStateFailed        ServiceState = "failed"
)

var ErrPluginServiceAlreadyRunning = errors.New("plugin service is already running")

type ServiceSnapshot struct {
	PluginID  string       `json:"pluginId"`
	State     ServiceState `json:"state"`
	Code      string       `json:"code"`
	UpdatedAt time.Time    `json:"updatedAt"`
}

type ServiceLifecycleEvent struct {
	PluginID  string
	State     ServiceState
	Code      string
	Timestamp time.Time
}

type ServiceAuditSink interface {
	RecordPluginServiceEvent(event ServiceLifecycleEvent) error
}

// ServiceHandle owns one launched service. ForceStop must release resources after graceful shutdown fails.
type ServiceHandle interface {
	Done() <-chan error
	Stop(ctx context.Context) error
	ForceStop() error
}

// ServiceLauncher returns only after readiness and must clean up any partial launch before returning an error.
type ServiceLauncher interface {
	Start(ctx context.Context, info Info) (ServiceHandle, error)
}

type supervisedService struct {
	snapshot ServiceSnapshot
	handle   ServiceHandle
}

type ServiceSupervisor struct {
	lifecycleMu  sync.Mutex
	mu           sync.RWMutex
	launcher     ServiceLauncher
	audit        ServiceAuditSink
	startTimeout time.Duration
	stopTimeout  time.Duration
	nowFn        func() time.Time
	services     map[string]*supervisedService
}

func NewServiceSupervisor(launcher ServiceLauncher, audit ServiceAuditSink, startTimeout, stopTimeout time.Duration) *ServiceSupervisor {
	if startTimeout <= 0 {
		startTimeout = 5 * time.Second
	}
	if stopTimeout <= 0 {
		stopTimeout = 5 * time.Second
	}
	return &ServiceSupervisor{
		launcher:     launcher,
		audit:        audit,
		startTimeout: startTimeout,
		stopTimeout:  stopTimeout,
		nowFn:        func() time.Time { return time.Now().UTC() },
		services:     make(map[string]*supervisedService),
	}
}

func (s *ServiceSupervisor) Start(ctx context.Context, info Info) error {
	if s == nil {
		return errors.New("plugin service supervisor is not configured")
	}
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	pluginID := strings.TrimSpace(info.ID)
	if pluginID == "" {
		return ErrPluginNotFound
	}
	if strings.TrimSpace(info.ServiceBaseURL) == "" {
		s.transition(pluginID, ServiceStateNotApplicable, "service_not_applicable", nil)
		return nil
	}
	if s.launcher == nil {
		s.transition(pluginID, ServiceStateFailed, "service_launcher_unavailable", nil)
		return errors.New("plugin service launcher is not configured")
	}

	s.mu.Lock()
	if current := s.services[pluginID]; current != nil {
		if current.snapshot.State == ServiceStateReady {
			s.mu.Unlock()
			return nil
		}
		if current.snapshot.State == ServiceStateStarting || current.snapshot.State == ServiceStateStopping {
			s.mu.Unlock()
			return ErrPluginServiceAlreadyRunning
		}
	}
	s.mu.Unlock()
	s.transition(pluginID, ServiceStateStarting, "service_starting", nil)

	startCtx, cancel := context.WithTimeout(contextOrBackground(ctx), s.startTimeout)
	defer cancel()
	handle, err := s.launcher.Start(startCtx, info)
	if err != nil {
		code := "service_start_failed"
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(startCtx.Err(), context.DeadlineExceeded) {
			code = "service_start_timeout"
		}
		s.transition(pluginID, ServiceStateFailed, code, nil)
		return fmt.Errorf("%s: %w", code, err)
	}
	if handle == nil || handle.Done() == nil {
		s.transition(pluginID, ServiceStateFailed, "service_handle_invalid", nil)
		return errors.New("plugin service launcher returned an invalid handle")
	}

	s.transition(pluginID, ServiceStateReady, "service_ready", handle)
	go s.watch(pluginID, handle)
	return nil
}

func (s *ServiceSupervisor) Stop(ctx context.Context, pluginID string) error {
	if s == nil {
		return nil
	}
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	pluginID = strings.TrimSpace(pluginID)
	if pluginID == "" {
		return ErrPluginNotFound
	}

	s.mu.RLock()
	current := s.services[pluginID]
	s.mu.RUnlock()
	if current == nil || current.handle == nil {
		s.transition(pluginID, ServiceStateStopped, "service_stopped", nil)
		return nil
	}
	s.transition(pluginID, ServiceStateStopping, "service_stopping", current.handle)

	stopCtx, cancel := context.WithTimeout(contextOrBackground(ctx), s.stopTimeout)
	defer cancel()
	if err := current.handle.Stop(stopCtx); err != nil {
		code := "service_stop_failed"
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(stopCtx.Err(), context.DeadlineExceeded) {
			code = "service_stop_timeout"
		}
		s.transition(pluginID, ServiceStateFailed, code, current.handle)
		if forceErr := current.handle.ForceStop(); forceErr != nil {
			s.transition(pluginID, ServiceStateFailed, "service_force_stop_failed", current.handle)
			return fmt.Errorf("%s: %w; force stop: %v", code, err, forceErr)
		}
		s.transition(pluginID, ServiceStateStopped, "service_force_stopped", nil)
		return fmt.Errorf("%s: %w", code, err)
	}
	s.transition(pluginID, ServiceStateStopped, "service_stopped", nil)
	return nil
}

func (s *ServiceSupervisor) Shutdown(ctx context.Context) error {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	ids := make([]string, 0, len(s.services))
	for id, item := range s.services {
		if item != nil && item.handle != nil {
			ids = append(ids, id)
		}
	}
	s.mu.RUnlock()
	sort.Strings(ids)

	var failures []string
	for _, id := range ids {
		if err := s.Stop(ctx, id); err != nil {
			failures = append(failures, id)
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("stop plugin services: %s", strings.Join(failures, ","))
	}
	return nil
}

func (s *ServiceSupervisor) Snapshot(pluginID string) (ServiceSnapshot, bool) {
	if s == nil {
		return ServiceSnapshot{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item := s.services[strings.TrimSpace(pluginID)]
	if item == nil {
		return ServiceSnapshot{}, false
	}
	return item.snapshot, true
}

func (s *ServiceSupervisor) watch(pluginID string, handle ServiceHandle) {
	err, ok := <-handle.Done()
	s.mu.RLock()
	current := s.services[pluginID]
	state := ServiceState("")
	if current != nil {
		state = current.snapshot.State
	}
	s.mu.RUnlock()
	if current == nil || current.handle != handle || state == ServiceStateStopping || state == ServiceStateStopped {
		return
	}
	code := "service_exited"
	if ok && err != nil {
		code = "service_crashed"
	}
	s.transition(pluginID, ServiceStateFailed, code, nil)
}

func (s *ServiceSupervisor) transition(pluginID string, state ServiceState, code string, handle ServiceHandle) {
	now := s.nowFn().UTC()
	snapshot := ServiceSnapshot{PluginID: pluginID, State: state, Code: code, UpdatedAt: now}
	s.mu.Lock()
	s.services[pluginID] = &supervisedService{snapshot: snapshot, handle: handle}
	s.mu.Unlock()
	if s.audit != nil {
		_ = s.audit.RecordPluginServiceEvent(ServiceLifecycleEvent{PluginID: pluginID, State: state, Code: code, Timestamp: now})
	}
}

func contextOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

type ExternalServiceLauncher struct {
	checker      HealthChecker
	pollInterval time.Duration
}

func NewExternalServiceLauncher(checker HealthChecker, pollInterval time.Duration) *ExternalServiceLauncher {
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}
	return &ExternalServiceLauncher{checker: checker, pollInterval: pollInterval}
}

func (l *ExternalServiceLauncher) Start(ctx context.Context, info Info) (ServiceHandle, error) {
	if l == nil || l.checker == nil {
		return nil, errors.New("plugin health checker is not configured")
	}
	report := l.checker.Check(contextOrBackground(ctx), info)
	if !report.Ready() {
		return nil, fmt.Errorf("plugin service is not ready: %s", report.Code)
	}
	handle := newExternalServiceHandle(info, l.checker, l.pollInterval)
	return handle, nil
}

type externalServiceHandle struct {
	info     Info
	checker  HealthChecker
	interval time.Duration
	cancel   context.CancelFunc
	done     chan error
	stopOnce sync.Once
}

func newExternalServiceHandle(info Info, checker HealthChecker, interval time.Duration) *externalServiceHandle {
	ctx, cancel := context.WithCancel(context.Background())
	h := &externalServiceHandle{info: info, checker: checker, interval: interval, cancel: cancel, done: make(chan error, 1)}
	go h.monitor(ctx)
	return h
}

func (h *externalServiceHandle) Done() <-chan error { return h.done }

func (h *externalServiceHandle) Stop(ctx context.Context) error {
	if h == nil {
		return nil
	}
	h.stopOnce.Do(h.cancel)
	select {
	case <-h.done:
		return nil
	case <-contextOrBackground(ctx).Done():
		return contextOrBackground(ctx).Err()
	}
}

func (h *externalServiceHandle) ForceStop() error {
	if h == nil {
		return nil
	}
	h.stopOnce.Do(h.cancel)
	<-h.done
	return nil
}

func (h *externalServiceHandle) monitor(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()
	defer close(h.done)
	for {
		select {
		case <-ctx.Done():
			h.done <- nil
			return
		case <-ticker.C:
			report := h.checker.Check(ctx, h.info)
			if !report.Ready() {
				h.done <- fmt.Errorf("plugin service unhealthy: %s", report.Code)
				return
			}
		}
	}
}

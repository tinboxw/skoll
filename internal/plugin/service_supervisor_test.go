package plugin

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestServiceSupervisorLifecycleCrashAndAudit(t *testing.T) {
	handle := newFakeServiceHandle()
	launcher := &fakeServiceLauncher{handle: handle}
	audit := &recordingServiceAudit{}
	supervisor := NewServiceSupervisor(launcher, audit, time.Second, time.Second)
	info := Info{ID: "reports", ServiceBaseURL: "http://service", ServiceHealthURL: "http://service/health"}

	if err := supervisor.Start(context.Background(), info); err != nil {
		t.Fatalf("start service: %v", err)
	}
	assertServiceState(t, supervisor, "reports", ServiceStateReady, "service_ready")

	handle.finish(errors.New("crashed"))
	waitForServiceState(t, supervisor, "reports", ServiceStateFailed)
	if got := audit.codes(); !containsServiceCode(got, "service_starting") || !containsServiceCode(got, "service_ready") || !containsServiceCode(got, "service_crashed") {
		t.Fatalf("unexpected audit codes: %v", got)
	}
}

func TestServiceSupervisorStopShutdownAndTimeout(t *testing.T) {
	launcher := &fakeServiceLauncher{factory: newFakeServiceHandle}
	supervisor := NewServiceSupervisor(launcher, nil, time.Second, 30*time.Millisecond)
	for _, id := range []string{"alpha", "beta"} {
		if err := supervisor.Start(context.Background(), Info{ID: id, ServiceBaseURL: "http://service", ServiceHealthURL: "http://service/health"}); err != nil {
			t.Fatalf("start %s: %v", id, err)
		}
	}
	if err := supervisor.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	assertServiceState(t, supervisor, "alpha", ServiceStateStopped, "service_stopped")
	assertServiceState(t, supervisor, "beta", ServiceStateStopped, "service_stopped")

	timeoutHandle := newFakeServiceHandle()
	timeoutHandle.blockStop = true
	timeoutAudit := &recordingServiceAudit{}
	timeoutSupervisor := NewServiceSupervisor(&fakeServiceLauncher{handle: timeoutHandle}, timeoutAudit, time.Second, 20*time.Millisecond)
	if err := timeoutSupervisor.Start(context.Background(), Info{ID: "slow", ServiceBaseURL: "http://service", ServiceHealthURL: "http://service/health"}); err != nil {
		t.Fatalf("start slow service: %v", err)
	}
	if err := timeoutSupervisor.Stop(context.Background(), "slow"); err == nil {
		t.Fatal("expected stop timeout")
	}
	assertServiceState(t, timeoutSupervisor, "slow", ServiceStateStopped, "service_force_stopped")
	if got := timeoutAudit.codes(); !containsServiceCode(got, "service_stop_timeout") || !containsServiceCode(got, "service_force_stopped") {
		t.Fatalf("unexpected timeout audit codes: %v", got)
	}

	startTimeoutSupervisor := NewServiceSupervisor(&fakeServiceLauncher{blockStart: true}, nil, 20*time.Millisecond, time.Second)
	if err := startTimeoutSupervisor.Start(context.Background(), Info{ID: "slow-start", ServiceBaseURL: "http://service", ServiceHealthURL: "http://service/health"}); err == nil {
		t.Fatal("expected start timeout")
	}
	assertServiceState(t, startTimeoutSupervisor, "slow-start", ServiceStateFailed, "service_start_timeout")
}

type fakeServiceLauncher struct {
	handle     *fakeServiceHandle
	factory    func() *fakeServiceHandle
	blockStart bool
}

func (l *fakeServiceLauncher) Start(ctx context.Context, _ Info) (ServiceHandle, error) {
	if l.blockStart {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if l.factory != nil {
		return l.factory(), nil
	}
	return l.handle, nil
}

type fakeServiceHandle struct {
	done      chan error
	stopOnce  sync.Once
	blockStop bool
}

func newFakeServiceHandle() *fakeServiceHandle {
	return &fakeServiceHandle{done: make(chan error, 1)}
}

func (h *fakeServiceHandle) Done() <-chan error { return h.done }

func (h *fakeServiceHandle) Stop(ctx context.Context) error {
	if h.blockStop {
		<-ctx.Done()
		return ctx.Err()
	}
	h.finish(nil)
	return nil
}

func (h *fakeServiceHandle) ForceStop() error {
	h.finish(nil)
	return nil
}

func (h *fakeServiceHandle) finish(err error) {
	h.stopOnce.Do(func() {
		h.done <- err
		close(h.done)
	})
}

type recordingServiceAudit struct {
	mu     sync.Mutex
	events []ServiceLifecycleEvent
}

func (a *recordingServiceAudit) RecordPluginServiceEvent(event ServiceLifecycleEvent) error {
	a.mu.Lock()
	a.events = append(a.events, event)
	a.mu.Unlock()
	return nil
}

func (a *recordingServiceAudit) codes() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]string, 0, len(a.events))
	for _, event := range a.events {
		out = append(out, event.Code)
	}
	return out
}

func assertServiceState(t *testing.T, supervisor *ServiceSupervisor, pluginID string, state ServiceState, code string) {
	t.Helper()
	snapshot, ok := supervisor.Snapshot(pluginID)
	if !ok || snapshot.State != state || snapshot.Code != code {
		t.Fatalf("snapshot=%+v exists=%v want state=%s code=%s", snapshot, ok, state, code)
	}
}

func waitForServiceState(t *testing.T, supervisor *ServiceSupervisor, pluginID string, state ServiceState) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if snapshot, ok := supervisor.Snapshot(pluginID); ok && snapshot.State == state {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("service %s did not reach %s", pluginID, state)
}

func containsServiceCode(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

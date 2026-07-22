package plugin

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestManagedProcessLauncherLifecycleEnvironmentAndHealth(t *testing.T) {
	const pluginID = "managed_test"
	address := reserveManagedProcessAddress(t)
	pluginDir := buildManagedTestBackend(t, pluginID)
	info := managedTestInfo(pluginID, pluginDir, address)
	t.Setenv("SKOLL_TEST_SECRET", "must-not-leak")

	checker := NewHTTPHealthChecker(200 * time.Millisecond)
	supervisor := NewServiceSupervisor(NewManagedProcessLauncher(checker, 20*time.Millisecond), nil, 5*time.Second, time.Second)
	t.Cleanup(func() { _ = supervisor.Shutdown(context.Background()) })
	if err := supervisor.Start(context.Background(), info); err != nil {
		t.Fatalf("start managed backend: %v", err)
	}
	assertServiceState(t, supervisor, pluginID, ServiceStateReady, "service_ready")

	response, err := http.Get(info.ServiceBaseURL + "/runtime")
	if err != nil {
		t.Fatalf("read managed runtime: %v", err)
	}
	defer response.Body.Close()
	var runtimeInfo map[string]string
	if err := json.NewDecoder(response.Body).Decode(&runtimeInfo); err != nil {
		t.Fatal(err)
	}
	if runtimeInfo["pluginId"] != pluginID || runtimeInfo["address"] != address || filepath.Clean(runtimeInfo["pluginDir"]) != filepath.Clean(pluginDir) {
		t.Fatalf("managed runtime context = %+v", runtimeInfo)
	}
	if runtimeInfo["secret"] != "" {
		t.Fatalf("parent secret leaked into plugin process: %+v", runtimeInfo)
	}

	if _, err := http.Post(info.ServiceBaseURL+"/unhealthy", "application/json", nil); err != nil {
		t.Fatalf("mark backend unhealthy: %v", err)
	}
	waitForServiceState(t, supervisor, pluginID, ServiceStateFailed)

	if err := supervisor.Start(context.Background(), info); err != nil {
		t.Fatalf("restart managed backend: %v", err)
	}
	if err := supervisor.Stop(context.Background(), pluginID); err != nil {
		t.Fatalf("stop managed backend: %v", err)
	}
	assertServiceState(t, supervisor, pluginID, ServiceStateStopped, "service_stopped")
	waitManagedBackendUnavailable(t, info.ServiceHealthURL)
}

func TestManagedProcessLauncherReportsCrash(t *testing.T) {
	const pluginID = "managed_crash"
	address := reserveManagedProcessAddress(t)
	info := managedTestInfo(pluginID, buildManagedTestBackend(t, pluginID), address)
	supervisor := NewServiceSupervisor(NewManagedProcessLauncher(NewHTTPHealthChecker(200*time.Millisecond), 20*time.Millisecond), nil, 5*time.Second, time.Second)
	t.Cleanup(func() { _ = supervisor.Shutdown(context.Background()) })
	if err := supervisor.Start(context.Background(), info); err != nil {
		t.Fatal(err)
	}
	_, _ = http.Post(info.ServiceBaseURL+"/crash", "application/json", nil)
	waitForServiceState(t, supervisor, pluginID, ServiceStateFailed)
	snapshot, _ := supervisor.Snapshot(pluginID)
	if snapshot.Code != "service_crashed" {
		t.Fatalf("crash state = %+v", snapshot)
	}
}

func TestManagedProcessLauncherRejectsMissingEntryAndRemoteService(t *testing.T) {
	launcher := NewManagedProcessLauncher(NewHTTPHealthChecker(time.Second), time.Millisecond)
	missing := Info{
		ID: "missing", Source: t.TempDir(),
		ServiceBaseURL: "http://127.0.0.1:19090", ServiceHealthURL: "http://127.0.0.1:19090/health",
	}
	if _, err := launcher.Start(context.Background(), missing); err == nil || !strings.Contains(err.Error(), "backend entry") {
		t.Fatalf("missing backend error = %v", err)
	}

	pluginDir := buildManagedTestBackend(t, "remote")
	remote := Info{
		ID: "remote", Source: pluginDir,
		ServiceBaseURL: "https://plugins.example.com", ServiceHealthURL: "https://plugins.example.com/health",
	}
	if _, err := launcher.Start(context.Background(), remote); err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("remote backend error = %v", err)
	}
}

func buildManagedTestBackend(t *testing.T, pluginID string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), pluginID)
	entry := filepath.Join(dir, filepath.FromSlash(managedBackendRelativePath(pluginID)))
	if err := os.MkdirAll(filepath.Dir(entry), 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(dir, "managed_test_server.go")
	program := `package main
import (
  "encoding/json"
  "net/http"
  "os"
  "os/signal"
  "sync/atomic"
  "syscall"
)
func main() {
  var healthy atomic.Bool
  healthy.Store(true)
  mux := http.NewServeMux()
  mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { if !healthy.Load() { http.Error(w, "unhealthy", http.StatusServiceUnavailable); return }; w.WriteHeader(http.StatusNoContent) })
  mux.HandleFunc("GET /runtime", func(w http.ResponseWriter, _ *http.Request) { _ = json.NewEncoder(w).Encode(map[string]string{"pluginId": os.Getenv("SKOLL_PLUGIN_ID"), "address": os.Getenv("SKOLL_PLUGIN_ADDRESS"), "pluginDir": os.Getenv("SKOLL_PLUGIN_DIR"), "secret": os.Getenv("SKOLL_TEST_SECRET")}) })
  mux.HandleFunc("POST /unhealthy", func(w http.ResponseWriter, _ *http.Request) { healthy.Store(false); w.WriteHeader(http.StatusNoContent) })
  mux.HandleFunc("POST /crash", func(http.ResponseWriter, *http.Request) { os.Exit(23) })
  server := &http.Server{Addr: os.Getenv("SKOLL_PLUGIN_ADDRESS"), Handler: mux}
  go func() { _ = server.ListenAndServe() }()
  signals := make(chan os.Signal, 1)
  signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
  <-signals
  _ = server.Close()
}
`
	if err := os.WriteFile(source, []byte(program), 0o600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "build", "-o", entry, source)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build managed test backend: %v\n%s", err, output)
	}
	return dir
}

func managedTestInfo(pluginID string, pluginDir string, address string) Info {
	base := "http://" + address
	return Info{ID: pluginID, Source: pluginDir, ServiceBaseURL: base, ServiceHealthURL: base + "/health"}
}

func reserveManagedProcessAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	return address
}

func waitManagedBackendUnavailable(t *testing.T, healthURL string) {
	t.Helper()
	client := &http.Client{Timeout: 50 * time.Millisecond}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		response, err := client.Get(healthURL)
		if response != nil {
			_ = response.Body.Close()
		}
		if err != nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("managed backend remained available at %s", healthURL)
}

func TestManagedBackendRelativePathMatchesPlatform(t *testing.T) {
	want := filepath.ToSlash(filepath.Join("backend", "bin", "reports-server"))
	if runtime.GOOS == "windows" {
		want += ".exe"
	}
	if got := managedBackendRelativePath("reports"); got != want {
		t.Fatalf("managed backend path = %q, want %q", got, want)
	}
}

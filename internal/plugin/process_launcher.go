package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/tinboxw/skoll/internal/plugin/quota"
)

const managedBackendDirectory = "backend/bin"

// ManagedProcessLauncher starts the one packaged backend executable owned by a plugin.
type ManagedProcessLauncher struct {
	checker      HealthChecker
	pollInterval time.Duration
	credentials  ProcessCredentialIssuer
	data         ProcessDataDirectory
	quotas       *quota.Controller
}

func NewManagedProcessLauncher(checker HealthChecker, pollInterval time.Duration, credentials ProcessCredentialIssuer, data ProcessDataDirectory, quotas *quota.Controller) *ManagedProcessLauncher {
	if pollInterval <= 0 {
		pollInterval = 100 * time.Millisecond
	}
	return &ManagedProcessLauncher{checker: checker, pollInterval: pollInterval, credentials: credentials, data: data, quotas: quotas}
}

func (l *ManagedProcessLauncher) Start(ctx context.Context, info Info) (ServiceHandle, error) {
	if l == nil || l.checker == nil || l.credentials == nil || l.data == nil || l.quotas == nil {
		return nil, errors.New("plugin health checker, credential issuer, data directories, and quotas are required")
	}
	if info.DataManifest == nil || info.DataManifest.UninstallPolicy == "" || info.DataManifest.RollbackPolicy == "" {
		return nil, errors.New("managed plugin requires an explicit data lifecycle policy")
	}
	entry, pluginDir, address, err := resolveManagedProcess(info)
	if err != nil {
		return nil, err
	}
	processLease, err := l.quotas.Acquire(info.ID, quota.ResourceProcess)
	if err != nil {
		return nil, err
	}
	releaseProcessLease := true
	defer func() {
		if releaseProcessLease {
			processLease.Release()
		}
	}()
	dataDir, err := l.data.Prepare(info.ID)
	if err != nil {
		return nil, fmt.Errorf("prepare plugin data directory: %w", err)
	}
	credential, err := l.credentials.Issue(info.ID)
	if err != nil {
		return nil, fmt.Errorf("issue plugin host credential: %w", err)
	}
	if strings.TrimSpace(credential.HostURL) == "" || strings.TrimSpace(credential.Token) == "" {
		l.credentials.Revoke(credential.Token)
		return nil, errors.New("plugin host credential is incomplete")
	}
	revokeCredential := true
	defer func() {
		if revokeCredential {
			l.credentials.Revoke(credential.Token)
		}
	}()
	command := exec.Command(entry)
	command.Dir = pluginDir
	policy := l.quotas.Policy()
	command.Env = managedProcessEnvironment(info.ID, address, pluginDir, dataDir, credential, policy)
	command.Stdin = nil
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("start plugin backend: %w", err)
	}

	wait := make(chan error, 1)
	go func() {
		wait <- command.Wait()
		close(wait)
	}()
	limitHandle, err := applyManagedProcessLimits(command.Process, policy)
	if err != nil {
		_ = command.Process.Kill()
		<-wait
		return nil, fmt.Errorf("apply plugin process limits: %w", err)
	}

	ticker := time.NewTicker(l.pollInterval)
	defer ticker.Stop()
	probeInfo := info
	probeInfo.State = StateEnabled
	for {
		if report := l.checker.Check(contextOrBackground(ctx), probeInfo); report.Ready() {
			revokeCredential = false
			releaseProcessLease = false
			return newManagedProcessHandle(command.Process, wait, probeInfo, l.checker, l.pollInterval, l.credentials, credential.Token, processLease, limitHandle), nil
		}
		select {
		case processErr := <-wait:
			_ = limitHandle.Close()
			return nil, fmt.Errorf("plugin backend exited before readiness: %w", normalizeProcessExit(processErr))
		case <-contextOrBackground(ctx).Done():
			_ = command.Process.Kill()
			<-wait
			_ = limitHandle.Close()
			return nil, contextOrBackground(ctx).Err()
		case <-ticker.C:
		}
	}
}

func resolveManagedProcess(info Info) (entry string, pluginDir string, address string, err error) {
	pluginDir, err = cleanAbsoluteDir(info.Source, "plugin source")
	if err != nil {
		return "", "", "", err
	}
	base, health, err := validateManagedServiceURLs(info.ServiceBaseURL, info.ServiceHealthURL)
	if err != nil {
		return "", "", "", err
	}
	address = base.Host
	entry, err = resolveManagedBackendEntry(pluginDir, info.ID)
	if err != nil {
		return "", "", "", err
	}
	if !sameURLOrigin(base, health) {
		return "", "", "", errors.New("plugin service and health URLs must use the same loopback origin")
	}
	return entry, pluginDir, address, nil
}

func resolveManagedBackendEntry(pluginDir string, pluginID string) (string, error) {
	relative := managedBackendRelativePath(pluginID)
	entry := filepath.Join(pluginDir, filepath.FromSlash(relative))
	if !sameOrChildPath(pluginDir, entry) {
		return "", errors.New("plugin backend entry escapes plugin directory")
	}
	fileInfo, err := os.Lstat(entry)
	if err != nil {
		return "", fmt.Errorf("inspect plugin backend entry: %w", err)
	}
	if fileInfo.Mode()&os.ModeSymlink != 0 || !fileInfo.Mode().IsRegular() {
		return "", errors.New("plugin backend entry must be a regular file")
	}
	if runtime.GOOS != "windows" && fileInfo.Mode().Perm()&0o111 == 0 {
		return "", errors.New("plugin backend entry is not executable")
	}
	return entry, nil
}

func validateManagedServiceURLs(baseRaw string, healthRaw string) (*url.URL, *url.URL, error) {
	base, err := url.Parse(strings.TrimSpace(baseRaw))
	if err != nil {
		return nil, nil, errors.New("plugin service URL is invalid")
	}
	health, err := url.Parse(strings.TrimSpace(healthRaw))
	if err != nil {
		return nil, nil, errors.New("plugin health URL is invalid")
	}
	for _, candidate := range []*url.URL{base, health} {
		if candidate.Scheme != "http" || candidate.Host == "" || candidate.User != nil || candidate.RawQuery != "" || candidate.Fragment != "" {
			return nil, nil, errors.New("managed plugin service URLs must be plain loopback HTTP URLs")
		}
		host := strings.TrimSpace(candidate.Hostname())
		ip := net.ParseIP(host)
		if !strings.EqualFold(host, "localhost") && (ip == nil || !ip.IsLoopback()) {
			return nil, nil, errors.New("managed plugin service URLs must use a loopback host")
		}
	}
	return base, health, nil
}

func sameURLOrigin(left *url.URL, right *url.URL) bool {
	return left != nil && right != nil && strings.EqualFold(left.Scheme, right.Scheme) && strings.EqualFold(left.Host, right.Host)
}

func managedBackendRelativePath(pluginID string) string {
	name := strings.TrimSpace(pluginID) + "-server"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.ToSlash(filepath.Join(managedBackendDirectory, name))
}

func managedProcessEnvironment(pluginID string, address string, pluginDir string, dataDir string, credential ProcessCredential, policy quota.Policy) []string {
	allowed := []string{"PATH", "SystemRoot", "WINDIR", "TEMP", "TMP", "HOME", "USERPROFILE"}
	environment := make([]string, 0, len(allowed)+6)
	for _, key := range allowed {
		if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
			environment = append(environment, key+"="+value)
		}
	}
	environment = append(environment,
		"SKOLL_PLUGIN_ID="+strings.TrimSpace(pluginID),
		"SKOLL_PLUGIN_ADDRESS="+strings.TrimSpace(address),
		"SKOLL_PLUGIN_DIR="+filepath.Clean(pluginDir),
		"SKOLL_PLUGIN_DATA_DIR="+filepath.Clean(dataDir),
		"SKOLL_PLUGIN_HOST_URL="+strings.TrimSpace(credential.HostURL),
		"SKOLL_PLUGIN_HOST_TOKEN="+strings.TrimSpace(credential.Token),
		fmt.Sprintf("SKOLL_PLUGIN_MEMORY_LIMIT_BYTES=%d", policy.ProcessMemoryBytes),
		fmt.Sprintf("SKOLL_PLUGIN_MAX_PROCS=%d", policy.ProcessMaxProcs),
		fmt.Sprintf("GOMEMLIMIT=%dB", policy.ProcessMemoryBytes),
		fmt.Sprintf("GOMAXPROCS=%d", policy.ProcessMaxProcs),
	)
	return environment
}

type managedProcessHandle struct {
	process         *os.Process
	wait            <-chan error
	info            Info
	checker         HealthChecker
	pollInterval    time.Duration
	done            chan error
	finishOnce      sync.Once
	stopOnce        sync.Once
	mu              sync.RWMutex
	stopping        bool
	credentials     ProcessCredentialIssuer
	credentialToken string
	revokeOnce      sync.Once
	processLease    *quota.Lease
	limitHandle     managedProcessLimitHandle
}

func newManagedProcessHandle(process *os.Process, wait <-chan error, info Info, checker HealthChecker, pollInterval time.Duration, credentials ProcessCredentialIssuer, credentialToken string, processLease *quota.Lease, limitHandle managedProcessLimitHandle) *managedProcessHandle {
	handle := &managedProcessHandle{
		process: process, wait: wait, info: info, checker: checker,
		pollInterval: pollInterval, done: make(chan error, 1), credentials: credentials,
		credentialToken: credentialToken, processLease: processLease, limitHandle: limitHandle,
	}
	go handle.monitor()
	return handle
}

func (h *managedProcessHandle) Done() <-chan error { return h.done }

func (h *managedProcessHandle) Stop(ctx context.Context) error {
	if h == nil {
		return nil
	}
	h.setStopping()
	var signalErr error
	h.stopOnce.Do(func() {
		if runtime.GOOS == "windows" {
			signalErr = h.process.Kill()
		} else {
			signalErr = h.process.Signal(os.Interrupt)
		}
	})
	if signalErr != nil && !errors.Is(signalErr, os.ErrProcessDone) {
		return signalErr
	}
	select {
	case <-h.done:
		return nil
	case <-contextOrBackground(ctx).Done():
		return contextOrBackground(ctx).Err()
	}
}

func (h *managedProcessHandle) ForceStop() error {
	if h == nil {
		return nil
	}
	h.setStopping()
	if err := h.process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}
	<-h.done
	return nil
}

func (h *managedProcessHandle) monitor() {
	ticker := time.NewTicker(h.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case err := <-h.wait:
			if h.isStopping() {
				err = nil
			}
			h.finish(err)
			return
		case <-ticker.C:
			if h.isStopping() {
				continue
			}
			report := h.checker.Check(context.Background(), h.info)
			if report.Ready() {
				continue
			}
			_ = h.process.Kill()
			<-h.wait
			h.finish(fmt.Errorf("plugin service unhealthy: %s", report.Code))
			return
		}
	}
}

func (h *managedProcessHandle) setStopping() {
	h.mu.Lock()
	h.stopping = true
	h.mu.Unlock()
}

func (h *managedProcessHandle) isStopping() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.stopping
}

func (h *managedProcessHandle) finish(err error) {
	h.finishOnce.Do(func() {
		h.revokeOnce.Do(func() {
			if h.credentials != nil {
				h.credentials.Revoke(h.credentialToken)
			}
		})
		if h.limitHandle != nil {
			_ = h.limitHandle.Close()
		}
		if h.processLease != nil {
			h.processLease.Release()
		}
		h.done <- err
		close(h.done)
	})
}

func normalizeProcessExit(err error) error {
	if err == nil {
		return errors.New("process exited")
	}
	return err
}

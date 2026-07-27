package plugintest

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	pluginruntime "github.com/tinboxw/skoll/internal/plugin"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type RuntimeOptions struct {
	PluginID     string
	Services     *Services
	JWTSecret    string
	DataRoot     string
	StartTimeout time.Duration
	StopTimeout  time.Duration
}

type Runtime struct {
	mu         sync.Mutex
	pluginID   string
	services   *Services
	jwtSecret  string
	manager    *pluginruntime.RuntimeManager
	gateway    *pluginruntime.HostGateway
	supervisor *pluginruntime.ServiceSupervisor
	closed     bool
}

func NewRuntime(options RuntimeOptions) (*Runtime, error) {
	pluginID := strings.ToLower(strings.TrimSpace(options.PluginID))
	if pluginID == "" {
		return nil, errors.New("fixture runtime plugin identity is required")
	}
	if options.Services == nil {
		return nil, errors.New("fixture runtime services are required")
	}
	secret := strings.TrimSpace(options.JWTSecret)
	if secret == "" {
		secret = DefaultJWTSecret
	}
	dataRoot := strings.TrimSpace(options.DataRoot)
	if dataRoot == "" {
		return nil, errors.New("fixture runtime data root is required")
	}
	gateway, err := pluginruntime.NewHostGateway(func(requested string) (pluginsdk.HostServices, error) {
		if requested != pluginID {
			return pluginsdk.HostServices{}, fmt.Errorf("fixture runtime rejected plugin %q", requested)
		}
		return options.Services.Host(requested)
	}, secret, time.Minute)
	if err != nil {
		return nil, fmt.Errorf("start fixture host gateway: %w", err)
	}
	dataDirectories, err := pluginruntime.NewPluginDataDirectories(dataRoot)
	if err != nil {
		_ = gateway.Close()
		return nil, fmt.Errorf("prepare fixture plugin data: %w", err)
	}
	launcher := pluginruntime.NewManagedProcessLauncher(
		pluginruntime.NewHTTPHealthChecker(time.Second), 25*time.Millisecond, gateway, dataDirectories,
	)
	return &Runtime{
		pluginID: pluginID, services: options.Services, jwtSecret: secret,
		manager:    pluginruntime.NewRuntimeManager(pluginruntime.NewFileLoader(), pluginruntime.NewTopologicalResolver()),
		gateway:    gateway,
		supervisor: pluginruntime.NewServiceSupervisor(launcher, nil, options.StartTimeout, options.StopTimeout),
	}, nil
}

func (r *Runtime) Install(path string) (pluginruntime.Info, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ready(); err != nil {
		return pluginruntime.Info{}, err
	}
	info, err := r.manager.Install(path)
	if err != nil {
		return pluginruntime.Info{}, err
	}
	if info.ID != r.pluginID {
		return pluginruntime.Info{}, fmt.Errorf("fixture runtime expected plugin %q, got %q", r.pluginID, info.ID)
	}
	return info, nil
}

func (r *Runtime) Enable(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ready(); err != nil {
		return err
	}
	info, err := r.manager.Get(r.pluginID)
	if err != nil {
		return err
	}
	if err := r.manager.Enable(r.pluginID); err != nil {
		return err
	}
	info, err = r.manager.Get(r.pluginID)
	if err != nil {
		_ = r.manager.Disable(r.pluginID)
		return err
	}
	if err := r.supervisor.Start(ctx, info); err != nil {
		_ = r.manager.Disable(r.pluginID)
		return err
	}
	return nil
}

func (r *Runtime) Restart(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ready(); err != nil {
		return err
	}
	info, err := r.manager.Get(r.pluginID)
	if err != nil {
		return err
	}
	if info.State != pluginruntime.StateEnabled {
		return errors.New("fixture runtime restart requires an enabled plugin")
	}
	if err := r.supervisor.Stop(ctx, r.pluginID); err != nil {
		return err
	}
	return r.supervisor.Start(ctx, info)
}

func (r *Runtime) Disable(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ready(); err != nil {
		return err
	}
	if err := r.manager.Disable(r.pluginID); err != nil {
		return err
	}
	return r.supervisor.Stop(ctx, r.pluginID)
}

func (r *Runtime) Uninstall(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ready(); err != nil {
		return err
	}
	info, err := r.manager.Get(r.pluginID)
	if err != nil {
		return err
	}
	if info.State == pluginruntime.StateEnabled {
		if err := r.manager.Disable(r.pluginID); err != nil {
			return err
		}
	}
	if err := r.supervisor.Stop(ctx, r.pluginID); err != nil {
		return err
	}
	return r.manager.Uninstall(r.pluginID)
}

func (r *Runtime) Info() (pluginruntime.Info, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ready(); err != nil {
		return pluginruntime.Info{}, err
	}
	return r.manager.Get(r.pluginID)
}

func (r *Runtime) Service() (pluginruntime.ServiceSnapshot, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return pluginruntime.ServiceSnapshot{}, false
	}
	return r.supervisor.Snapshot(r.pluginID)
}

func (r *Runtime) UserToken(ttl time.Duration) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ready(); err != nil {
		return "", err
	}
	return r.services.Identity.Token(r.jwtSecret, ttl, r.services.Clock.Now())
}

func (r *Runtime) Manager() pluginruntime.Manager {
	return r.manager
}

func (r *Runtime) Close(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil
	}
	r.closed = true
	return errors.Join(r.supervisor.Shutdown(ctx), r.gateway.Close())
}

func (r *Runtime) ready() error {
	if r == nil || r.closed {
		return errors.New("fixture runtime is closed")
	}
	return nil
}

func ReserveLoopbackAddress() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("reserve fixture loopback address: %w", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		return "", fmt.Errorf("release fixture loopback address: %w", err)
	}
	return address, nil
}

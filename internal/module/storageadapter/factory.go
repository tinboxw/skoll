package storageadapter

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/tinboxw/skoll/internal/module/storageadapter/contracts"
	"github.com/tinboxw/skoll/internal/module/storageadapter/memory"
	"github.com/tinboxw/skoll/internal/module/storageadapter/persistent"
)

type Constructor func() (contracts.Adapter, error)

const (
	ModeMemory   = "memory"
	ModeMySQL    = "mysql"
	ModePostgres = "postgres"
)

var (
	constructorsMu sync.RWMutex
	constructors   = map[string]Constructor{}
)

func init() {
	MustRegisterMode(ModeMemory, func() (contracts.Adapter, error) {
		return memory.NewAdapter()
	})
	MustRegisterMode(ModeMySQL, func() (contracts.Adapter, error) {
		cfg, err := persistent.ResolveBootstrapConfig(ModeMySQL, nil)
		if err != nil {
			return nil, err
		}
		return persistent.NewMySQLAdapter(cfg)
	})
	MustRegisterMode(ModePostgres, func() (contracts.Adapter, error) {
		cfg, err := persistent.ResolveBootstrapConfig(ModePostgres, nil)
		if err != nil {
			return nil, err
		}
		return persistent.NewPostgresAdapter(cfg)
	})
}

func RegisterMode(mode string, constructor Constructor) error {
	resolved := strings.ToLower(strings.TrimSpace(mode))
	if resolved == "" {
		return fmt.Errorf("storageadapter.RegisterMode: empty mode")
	}
	if constructor == nil {
		return fmt.Errorf("storageadapter.RegisterMode: nil constructor for mode %q", resolved)
	}

	constructorsMu.Lock()
	defer constructorsMu.Unlock()
	if _, exists := constructors[resolved]; exists {
		return fmt.Errorf("storageadapter.RegisterMode: mode %q already registered", resolved)
	}
	constructors[resolved] = constructor
	return nil
}

func MustRegisterMode(mode string, constructor Constructor) {
	if err := RegisterMode(mode, constructor); err != nil {
		panic(err)
	}
}

func resolveConstructor(mode string) (Constructor, bool) {
	constructorsMu.RLock()
	defer constructorsMu.RUnlock()
	c, ok := constructors[mode]
	return c, ok
}

func registeredModes() []string {
	constructorsMu.RLock()
	defer constructorsMu.RUnlock()
	out := make([]string, 0, len(constructors))
	for mode := range constructors {
		out = append(out, mode)
	}
	sort.Strings(out)
	return out
}

func NewByMode(mode string) (contracts.Adapter, error) {
	resolved := strings.ToLower(strings.TrimSpace(mode))
	if resolved == "" {
		resolved = ModeMemory
	}

	constructor, ok := resolveConstructor(resolved)
	if !ok {
		return nil, fmt.Errorf("unsupported storage adapter mode %q, valid: %s", resolved, strings.Join(registeredModes(), "|"))
	}
	return constructor()
}

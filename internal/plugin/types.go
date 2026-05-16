package plugin

import (
	"errors"
	"time"
)

var (
	ErrPluginNotFound        = errors.New("plugin not found")
	ErrPluginAlreadyExists   = errors.New("plugin already exists")
	ErrPluginInvalidState    = errors.New("plugin state transition is invalid")
	ErrPluginManifestBroken  = errors.New("plugin manifest is invalid")
	ErrPluginDependency      = errors.New("plugin dependency error")
	ErrPluginSystemProtected = errors.New("system builtin plugin cannot be disabled or uninstalled")
)

type State string

const (
	StateInstalled   State = "installed"
	StateEnabled     State = "enabled"
	StateDisabled    State = "disabled"
	StateUninstalled State = "uninstalled"
)

type Dependency struct {
	ID      string
	Version string
}

type UIMode string

const (
	UIModeBackendOnly  UIMode = "backend_only"
	UIModeFrontendOnly UIMode = "frontend_only"
	UIModeMonolith     UIMode = "monolith"
	UIModeSeparated    UIMode = "separated"
)

type Info struct {
	ID            string
	Name          string
	Version       string
	Description   string
	ConfigJSON    string
	Dependencies  []Dependency
	Permissions   []string
	State         State
	InstalledAt   time.Time
	EnabledAt     *time.Time
	Source        string
	UIMode        UIMode
	FrontendEntry string
	SystemBuiltin bool
}

func (i Info) ValidateManifest() error {
	if i.ID == "" || i.Name == "" || i.Version == "" {
		return ErrPluginManifestBroken
	}

	for _, dep := range i.Dependencies {
		if dep.ID == "" {
			return ErrPluginManifestBroken
		}
	}

	mode := i.UIMode
	if mode == "" {
		mode = UIModeBackendOnly
	}
	if mode != UIModeBackendOnly && mode != UIModeFrontendOnly && mode != UIModeMonolith && mode != UIModeSeparated {
		return ErrPluginManifestBroken
	}

	return nil
}

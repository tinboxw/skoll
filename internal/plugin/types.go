package plugin

import (
	"errors"
	"time"
)

var (
	ErrPluginNotFound       = errors.New("plugin not found")
	ErrPluginAlreadyExists  = errors.New("plugin already exists")
	ErrPluginInvalidState   = errors.New("plugin state transition is invalid")
	ErrPluginManifestBroken = errors.New("plugin manifest is invalid")
	ErrPluginDependency     = errors.New("plugin dependency error")
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

type Info struct {
	ID           string
	Name         string
	Version      string
	Description  string
	Dependencies []Dependency
	Permissions  []string
	State        State
	InstalledAt  time.Time
	EnabledAt    *time.Time
	Source       string
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

	return nil
}

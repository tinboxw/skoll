package plugin

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const pluginDataArchiveDirectory = ".archive"

// ProcessDataDirectory prepares the stable host-owned persistence root for a plugin process.
type ProcessDataDirectory interface {
	Prepare(pluginID string) (string, error)
}

// PluginDataDirectories owns active and archived plugin file data outside package directories.
type PluginDataDirectories struct {
	mu    sync.Mutex
	root  string
	nowFn func() time.Time
}

func NewPluginDataDirectories(root string) (*PluginDataDirectories, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("plugin data root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve plugin data root: %w", err)
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return nil, fmt.Errorf("create plugin data root: %w", err)
	}
	if err := requireManagedDirectory(abs, "plugin data root"); err != nil {
		return nil, err
	}
	return &PluginDataDirectories{root: filepath.Clean(abs), nowFn: time.Now}, nil
}

func (d *PluginDataDirectories) Prepare(pluginID string) (string, error) {
	if d == nil {
		return "", errors.New("plugin data directories are required")
	}
	pluginID, err := validatePluginDataID(pluginID)
	if err != nil {
		return "", err
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	active := filepath.Join(d.root, pluginID)
	if !sameOrChildPath(d.root, active) {
		return "", errors.New("plugin data directory escapes managed root")
	}
	if err := os.MkdirAll(active, 0o700); err != nil {
		return "", fmt.Errorf("create plugin data directory: %w", err)
	}
	if err := requireManagedDirectory(active, "plugin data directory"); err != nil {
		return "", err
	}
	return active, nil
}

// Uninstall applies the one policy declared by the current plugin manifest.
func (d *PluginDataDirectories) Uninstall(pluginID string, policy DataUninstallPolicy) error {
	if d == nil {
		return errors.New("plugin data directories are required")
	}
	pluginID, err := validatePluginDataID(pluginID)
	if err != nil {
		return err
	}
	switch policy {
	case DataUninstallRetain, DataUninstallArchive, DataUninstallDrop:
	default:
		return errors.New("plugin uninstall requires an explicit data policy")
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	active := filepath.Join(d.root, pluginID)
	if policy == DataUninstallRetain {
		return validateExistingManagedDirectory(active)
	}
	if err := validateExistingManagedDirectory(active); err != nil {
		return err
	}
	if _, err := os.Lstat(active); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("inspect plugin data directory: %w", err)
	}

	if policy == DataUninstallDrop {
		if err := os.RemoveAll(active); err != nil {
			return fmt.Errorf("drop plugin data directory: %w", err)
		}
		return ensurePathAbsent(active)
	}

	archiveParent := filepath.Join(d.root, pluginDataArchiveDirectory, pluginID)
	if err := os.MkdirAll(archiveParent, 0o700); err != nil {
		return fmt.Errorf("create plugin data archive: %w", err)
	}
	if err := requireManagedDirectory(archiveParent, "plugin data archive"); err != nil {
		return err
	}
	destination := filepath.Join(archiveParent, d.nowFn().UTC().Format("20060102T150405.000000000Z"))
	if err := os.Rename(active, destination); err != nil {
		return fmt.Errorf("archive plugin data directory: %w", err)
	}
	return ensurePathAbsent(active)
}

func validatePluginDataID(pluginID string) (string, error) {
	pluginID = strings.TrimSpace(pluginID)
	if !appIDPattern.MatchString(pluginID) {
		return "", errors.New("plugin data identity is invalid")
	}
	return pluginID, nil
}

func requireManagedDirectory(path, label string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect %s: %w", label, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("%s must be a real directory", label)
	}
	return nil
}

func validateExistingManagedDirectory(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect plugin data directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("plugin data directory must be a real directory")
	}
	return nil
}

func ensurePathAbsent(path string) error {
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	return errors.New("plugin data directory still exists after lifecycle operation")
}

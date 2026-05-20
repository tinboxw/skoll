package plugin

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestValidatePluginManifestsUnderPluginsDir(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to resolve caller path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	pluginsRoot := filepath.Join(repoRoot, "plugins")

	if _, err := os.Stat(pluginsRoot); err != nil {
		t.Fatalf("plugins directory not found: %v", err)
	}

	manifestDirs, err := discoverPluginDirs(pluginsRoot)
	if err != nil {
		t.Fatalf("discover plugin manifests: %v", err)
	}
	if len(manifestDirs) == 0 {
		t.Fatalf("no plugin manifests found under %s", pluginsRoot)
	}

	loader := NewFileLoader()
	coreVersion := strings.TrimSpace(os.Getenv("SKOLL_CORE_VERSION"))
	if coreVersion == "" {
		coreVersion = "1.0.0"
	}
	for _, dir := range manifestDirs {
		info, err := loader.Load(dir)
		if err != nil {
			t.Fatalf("invalid plugin manifest at %s: %v", dir, err)
		}
		if err := info.ValidateCompatibility(coreVersion); err != nil {
			t.Fatalf("plugin compatibility check failed at %s: %v", dir, err)
		}
	}
}

func discoverPluginDirs(root string) ([]string, error) {
	paths := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d == nil || !d.IsDir() {
			return nil
		}

		name := d.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" {
			if path == root {
				return nil
			}
			return filepath.SkipDir
		}

		manifest := filepath.Join(path, "plugin.yaml")
		if fi, err := os.Stat(manifest); err == nil && !fi.IsDir() {
			paths = append(paths, filepath.Clean(path))
			return filepath.SkipDir
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

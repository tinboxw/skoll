package plugin

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type MetadataLoader interface {
	Load(path string) (Info, error)
}

type FileLoader struct{}

func NewFileLoader() *FileLoader {
	return &FileLoader{}
}

func (l *FileLoader) Load(path string) (Info, error) {
	manifestPath, err := resolveManifestPath(path)
	if err != nil {
		return Info{}, err
	}

	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return Info{}, fmt.Errorf("read plugin manifest: %w", err)
	}

	info, err := parseManifest(raw)
	if err != nil {
		return Info{}, err
	}

	if err := info.ValidateManifest(); err != nil {
		return Info{}, err
	}

	return info, nil
}

func resolveManifestPath(path string) (string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	if fi.IsDir() {
		p := filepath.Join(path, "plugin.yaml")
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("plugin.yaml not found under %s", path)
		}
		return p, nil
	}

	if filepath.Base(path) != "plugin.yaml" {
		return "", errors.New("manifest file must be named plugin.yaml")
	}

	return path, nil
}

func parseManifest(raw []byte) (Info, error) {
	var info Info

	const (
		sectionRoot = "root"
		sectionDeps = "deps"
		sectionPerm = "perm"
	)

	section := sectionRoot
	var currentDep *Dependency

	scanner := bufio.NewScanner(strings.NewReader(string(raw)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		switch line {
		case "dependencies:":
			if currentDep != nil {
				info.Dependencies = append(info.Dependencies, *currentDep)
				currentDep = nil
			}
			section = sectionDeps
			continue
		case "permissions:":
			if currentDep != nil {
				info.Dependencies = append(info.Dependencies, *currentDep)
				currentDep = nil
			}
			section = sectionPerm
			continue
		}

		if strings.HasPrefix(line, "-") {
			item := strings.TrimSpace(strings.TrimPrefix(line, "-"))
			switch section {
			case sectionDeps:
				if currentDep != nil {
					info.Dependencies = append(info.Dependencies, *currentDep)
				}
				currentDep = &Dependency{}
				if strings.HasPrefix(item, "id:") {
					currentDep.ID = parseScalar(strings.TrimSpace(strings.TrimPrefix(item, "id:")))
				} else {
					currentDep.ID = parseScalar(item)
				}
			case sectionPerm:
				info.Permissions = append(info.Permissions, parseScalar(item))
			}
			continue
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = parseScalar(strings.TrimSpace(value))

		switch key {
		case "id":
			if section == sectionDeps {
				if currentDep == nil {
					currentDep = &Dependency{}
				}
				currentDep.ID = value
			} else {
				info.ID = value
			}
		case "name":
			info.Name = value
		case "version":
			if section == sectionDeps {
				if currentDep == nil {
					currentDep = &Dependency{}
				}
				currentDep.Version = value
			} else {
				info.Version = value
			}
		case "description":
			info.Description = value
		case "ui_mode":
			info.UIMode = UIMode(value)
		case "frontend_entry":
			info.FrontendEntry = value
		}
	}

	if err := scanner.Err(); err != nil {
		return Info{}, fmt.Errorf("scan manifest: %w", err)
	}

	if currentDep != nil {
		info.Dependencies = append(info.Dependencies, *currentDep)
	}
	if info.UIMode == "" {
		info.UIMode = UIModeBackendOnly
	}

	return info, nil
}

func parseScalar(v string) string {
	v = strings.TrimSpace(v)
	v = strings.Trim(v, "\"")
	v = strings.Trim(v, "'")
	return v
}

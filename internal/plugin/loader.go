package plugin

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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
		case "api_version":
			info.APIVersion = value
		case "compatibility_skoll", "compatibility.skoll":
			info.CompatibilitySkoll = value
		case "service_base_url", "service.base_url":
			info.ServiceBaseURL = value
		case "service_health_url", "service.health_url":
			info.ServiceHealthURL = value
		case "migration_version", "migrations.version":
			info.MigrationVersion = value
		case "description":
			info.Description = value
		case "ui_mode":
			info.UIMode = UIMode(value)
		case "level":
			info.Level = Level(value)
		case "app_id":
			info.AppID = value
		case "mount_policy":
			info.MountPolicy = MountPolicy(value)
		case "frontend_entry":
			info.FrontendEntry = value
		case "vendor":
			info.Vendor = value
		case "vendor_url":
			info.VendorURL = value
		case "sign_algo":
			if info.Signature == nil {
				info.Signature = &Signature{}
			}
			info.Signature.Algorithm = SignatureAlgorithm(value)
		case "sign_timestamp":
			if info.Signature == nil {
				info.Signature = &Signature{}
			}
			if t, err := time.Parse(time.RFC3339, value); err == nil {
				info.Signature.Timestamp = t
			}
		case "sign_value":
			if info.Signature == nil {
				info.Signature = &Signature{}
			}
			info.Signature.Value = value
		case "vendor_pubkey":
			if info.Signature == nil {
				info.Signature = &Signature{}
			}
			info.Signature.PublicKey = value
			info.Signature.VendorID = info.Vendor
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
	if info.Level == "" {
		info.Level = LevelSystem
	}
	if info.MountPolicy == "" {
		info.MountPolicy = MountPolicyAdmin
	}
	info.FrontendEntry = ResolveFrontendEntry(info)

	return info, nil
}

func parseScalar(v string) string {
	v = strings.TrimSpace(v)
	v = strings.Trim(v, "\"")
	v = strings.Trim(v, "'")
	return v
}

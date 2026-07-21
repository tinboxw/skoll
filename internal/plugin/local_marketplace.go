package plugin

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type LocalMarketplaceService struct {
	loader MetadataLoader
}

type LocalMarketplaceCatalog struct {
	PluginsRoot string                 `json:"pluginsRoot"`
	Items       []LocalMarketplaceItem `json:"items"`
}

type LocalMarketplaceItem struct {
	ID               string                      `json:"id"`
	Name             string                      `json:"name"`
	Version          string                      `json:"version"`
	Description      string                      `json:"description,omitempty"`
	ManifestPath     string                      `json:"manifestPath,omitempty"`
	PackagePath      string                      `json:"packagePath,omitempty"`
	PackageDigest    string                      `json:"packageDigest,omitempty"`
	PackageSizeBytes int64                       `json:"packageSizeBytes,omitempty"`
	Installable      bool                        `json:"installable"`
	Signature        LocalMarketplaceSignature   `json:"signature"`
	Risk             LocalMarketplaceRiskSummary `json:"risk"`
}

type LocalMarketplaceSignature struct {
	Status    string `json:"status"`
	Algorithm string `json:"algorithm,omitempty"`
	VendorID  string `json:"vendorId,omitempty"`
	SignedAt  string `json:"signedAt,omitempty"`
}

type LocalMarketplaceRiskSummary struct {
	Level       string   `json:"level"`
	Permissions []string `json:"permissions,omitempty"`
	Migrations  []string `json:"migrations,omitempty"`
	Data        []string `json:"data,omitempty"`
	APIs        []string `json:"apis,omitempty"`
	Audit       []string `json:"audit,omitempty"`
	Network     []string `json:"network,omitempty"`
	Assets      []string `json:"assets,omitempty"`
}

func NewLocalMarketplaceService(loader MetadataLoader) *LocalMarketplaceService {
	if loader == nil {
		loader = NewFileLoader()
	}
	return &LocalMarketplaceService{loader: loader}
}

func (s *LocalMarketplaceService) List(root string) (LocalMarketplaceCatalog, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return LocalMarketplaceCatalog{}, fmt.Errorf("%w: plugins root is required", ErrMarketplaceIndexInvalid)
	}
	cleanRoot := filepath.Clean(root)
	dirs, err := discoverLocalMarketplacePluginDirs(cleanRoot)
	if err != nil {
		return LocalMarketplaceCatalog{}, err
	}

	items := make([]LocalMarketplaceItem, 0, len(dirs))
	byRelease := map[string]int{}
	for _, dir := range dirs {
		info, err := s.loader.Load(dir)
		if err != nil {
			return LocalMarketplaceCatalog{}, fmt.Errorf("load local marketplace plugin %s: %w", dir, err)
		}
		item := localMarketplaceItemFromInfo(info, dir)
		byRelease[localMarketplaceReleaseKey(item.ID, item.Version)] = len(items)
		items = append(items, item)
	}

	packages, err := discoverLocalMarketplacePackages(cleanRoot)
	if err != nil {
		return LocalMarketplaceCatalog{}, err
	}
	for _, pkg := range packages {
		id, version := parseLocalPackageName(pkg)
		digest, size, err := packageDigest(pkg)
		if err != nil {
			return LocalMarketplaceCatalog{}, err
		}
		key := localMarketplaceReleaseKey(id, version)
		if idx, ok := byRelease[key]; ok {
			items[idx].PackagePath = filepath.Clean(pkg)
			items[idx].PackageDigest = digest
			items[idx].PackageSizeBytes = size
			items[idx].Installable = true
			continue
		}
		items = append(items, LocalMarketplaceItem{
			ID:               id,
			Name:             id,
			Version:          version,
			PackagePath:      filepath.Clean(pkg),
			PackageDigest:    digest,
			PackageSizeBytes: size,
			Installable:      true,
			Signature:        LocalMarketplaceSignature{Status: "unknown"},
			Risk:             LocalMarketplaceRiskSummary{Level: "unknown"},
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].ID != items[j].ID {
			return items[i].ID < items[j].ID
		}
		return normalizeMarketplaceVersion(items[i].Version) < normalizeMarketplaceVersion(items[j].Version)
	})
	return LocalMarketplaceCatalog{PluginsRoot: cleanRoot, Items: items}, nil
}

func localMarketplaceItemFromInfo(info Info, dir string) LocalMarketplaceItem {
	return LocalMarketplaceItem{
		ID:           strings.TrimSpace(info.ID),
		Name:         strings.TrimSpace(info.Name),
		Version:      strings.TrimSpace(info.Version),
		Description:  strings.TrimSpace(info.Description),
		ManifestPath: filepath.Join(filepath.Clean(dir), "plugin.yaml"),
		Signature:    localMarketplaceSignature(info),
		Risk:         localMarketplaceRisk(info),
	}
}

func localMarketplaceSignature(info Info) LocalMarketplaceSignature {
	if info.Signature == nil {
		return LocalMarketplaceSignature{Status: "unsigned"}
	}
	status := "signed"
	if strings.TrimSpace(info.Signature.Value) == "" || info.Signature.Timestamp.IsZero() {
		status = "incomplete"
	}
	signedAt := ""
	if !info.Signature.Timestamp.IsZero() {
		signedAt = info.Signature.Timestamp.UTC().Format("2006-01-02T15:04:05Z")
	}
	return LocalMarketplaceSignature{
		Status:    status,
		Algorithm: string(info.Signature.Algorithm),
		VendorID:  strings.TrimSpace(info.Signature.VendorID),
		SignedAt:  signedAt,
	}
}

func localMarketplaceRisk(info Info) LocalMarketplaceRiskSummary {
	risk := LocalMarketplaceRiskSummary{Level: "low"}
	highest := 0
	for _, permission := range info.PermissionResources {
		key := strings.TrimSpace(permission.Key)
		if key != "" {
			risk.Permissions = append(risk.Permissions, key)
		}
		level := riskRank(strings.TrimSpace(permission.Risk))
		if level > highest {
			highest = level
		}
	}
	if strings.TrimSpace(info.MigrationVersion) != "" {
		risk.Migrations = append(risk.Migrations, "migration:"+strings.TrimSpace(info.MigrationVersion))
		if highest < 1 {
			highest = 1
		}
	}
	if info.DataManifest != nil {
		namespace := strings.TrimSpace(info.DataManifest.Namespace)
		if namespace != "" {
			risk.Data = append(risk.Data, "namespace:"+namespace)
		}
		for _, table := range info.DataManifest.Tables {
			name := strings.TrimSpace(table.Name)
			if name != "" {
				risk.Data = append(risk.Data, "table:"+name)
			}
		}
		if info.DataManifest.UninstallPolicy == DataUninstallDrop {
			risk.Data = append(risk.Data, "uninstall:drop")
			highest = 3
		} else if len(info.DataManifest.Tables) > 0 && highest < 2 {
			highest = 2
		}
	}
	if info.APIContract != nil {
		for _, route := range info.APIContract.Routes {
			method := strings.ToUpper(strings.TrimSpace(route.Method))
			path := NormalizeEntryPath(route.Path)
			if method != "" && path != "" {
				risk.APIs = append(risk.APIs, method+" "+path)
			}
		}
		if len(info.APIContract.Routes) > 0 && highest < 1 {
			highest = 1
		}
	}
	risk.Audit = append(risk.Audit, info.AuditActions()...)
	if strings.TrimSpace(info.ServiceBaseURL) != "" || strings.TrimSpace(info.ServiceHealthURL) != "" {
		if strings.TrimSpace(info.ServiceBaseURL) != "" {
			risk.Network = append(risk.Network, info.ServiceBaseURL)
		}
		if strings.TrimSpace(info.ServiceHealthURL) != "" {
			risk.Network = append(risk.Network, info.ServiceHealthURL)
		}
		if highest < 2 {
			highest = 2
		}
	}
	if info.UIMode != "" && info.UIMode != UIModeBackendOnly {
		risk.Assets = append(risk.Assets, string(info.UIMode))
	}
	risk.Level = riskLevelFromRank(highest)
	sort.Strings(risk.Permissions)
	sort.Strings(risk.Migrations)
	sort.Strings(risk.Data)
	sort.Strings(risk.APIs)
	sort.Strings(risk.Audit)
	sort.Strings(risk.Network)
	sort.Strings(risk.Assets)
	return risk
}

func discoverLocalMarketplacePluginDirs(root string) ([]string, error) {
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	dirs := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "_dist" {
			if path == root {
				return nil
			}
			return filepath.SkipDir
		}
		manifest := filepath.Join(path, "plugin.yaml")
		if stat, err := os.Stat(manifest); err == nil && !stat.IsDir() {
			dirs = append(dirs, filepath.Clean(path))
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(dirs)
	return dirs, nil
}

func discoverLocalMarketplacePackages(root string) ([]string, error) {
	dist := filepath.Join(root, "_dist")
	entries, err := os.ReadDir(dist)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	packages := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".zip") {
			continue
		}
		packages = append(packages, filepath.Join(dist, entry.Name()))
	}
	sort.Strings(packages)
	return packages, nil
}

func parseLocalPackageName(path string) (id, version string) {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if left, right, ok := strings.Cut(base, "-"); ok {
		parts := strings.Split(base, "-")
		for i := len(parts) - 1; i > 0; i-- {
			candidateVersion := strings.Join(parts[i:], "-")
			if marketplaceSemverPattern.MatchString(candidateVersion) {
				return strings.Join(parts[:i], "-"), candidateVersion
			}
		}
		return left, right
	}
	return base, "0.0.0"
}

func packageDigest(path string) (digest string, size int64, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	hasher := sha256.New()
	size, err = io.Copy(hasher, file)
	if err != nil {
		return "", 0, err
	}
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil)), size, nil
}

func localMarketplaceReleaseKey(id, version string) string {
	return strings.TrimSpace(strings.ToLower(id)) + "@" + normalizeMarketplaceVersion(version)
}

func riskRank(level string) int {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "critical":
		return 3
	case "high":
		return 2
	case "medium":
		return 1
	default:
		return 0
	}
}

func riskLevelFromRank(rank int) string {
	switch rank {
	case 3:
		return "critical"
	case 2:
		return "high"
	case 1:
		return "medium"
	default:
		return "low"
	}
}

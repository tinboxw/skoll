package plugin

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const SignatureAssetManifestPath = ".skoll/signature-assets.json"

var ErrSignatureCoverage = errors.New("plugin signature coverage failed")

type SignatureCoveragePolicy struct {
	AssetManifestPath string
}

type SignatureAssetManifest struct {
	SchemaVersion string                `json:"schema_version"`
	Files         []SignatureAssetEntry `json:"files"`
}

type SignatureAssetEntry struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256,omitempty"`
	Digest string `json:"digest,omitempty"`
}

type SignatureCoverageReport struct {
	PluginID            string   `json:"pluginId"`
	ManifestPath        string   `json:"manifestPath"`
	ManifestDigest      string   `json:"manifestDigest"`
	AssetManifestPath   string   `json:"assetManifestPath"`
	AssetManifestDigest string   `json:"assetManifestDigest"`
	BackendEntry        string   `json:"backendEntry,omitempty"`
	FrontendAssets      []string `json:"frontendAssets,omitempty"`
	CoveredFiles        []string `json:"coveredFiles"`
	Missing             []string `json:"missing,omitempty"`
	Mismatched          []string `json:"mismatched,omitempty"`
}

func CheckSignatureCoverage(pluginDir string, info Info, policy SignatureCoveragePolicy) (SignatureCoverageReport, error) {
	pluginDir = strings.TrimSpace(pluginDir)
	if pluginDir == "" {
		return SignatureCoverageReport{}, fmt.Errorf("%w: plugin dir is required", ErrSignatureCoverage)
	}
	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	manifestDigest, err := fileSHA256(manifestPath)
	if err != nil {
		return SignatureCoverageReport{}, fmt.Errorf("%w: manifest digest: %v", ErrSignatureCoverage, err)
	}
	assetManifestRel := strings.TrimSpace(policy.AssetManifestPath)
	if assetManifestRel == "" {
		assetManifestRel = SignatureAssetManifestPath
	}
	assetManifestPath, err := safePluginRelativePath(pluginDir, assetManifestRel)
	if err != nil {
		return SignatureCoverageReport{}, fmt.Errorf("%w: asset manifest path: %v", ErrSignatureCoverage, err)
	}
	assetManifestDigest, err := fileSHA256(assetManifestPath)
	if err != nil {
		return SignatureCoverageReport{}, fmt.Errorf("%w: asset manifest digest: %v", ErrSignatureCoverage, err)
	}
	assetManifest, err := loadSignatureAssetManifest(assetManifestPath)
	if err != nil {
		return SignatureCoverageReport{}, err
	}
	entries, err := normalizeSignatureAssetEntries(assetManifest)
	if err != nil {
		return SignatureCoverageReport{}, err
	}

	required, backendEntry, frontendAssets, err := requiredSignatureCoverageFiles(pluginDir, info)
	if err != nil {
		return SignatureCoverageReport{}, err
	}
	missing := make([]string, 0)
	mismatched := make([]string, 0)
	for _, rel := range required {
		expected, ok := entries[rel]
		if !ok {
			missing = append(missing, rel)
			continue
		}
		abs, err := safePluginRelativePath(pluginDir, rel)
		if err != nil {
			mismatched = append(mismatched, rel)
			continue
		}
		actual, err := fileSHA256(abs)
		if err != nil || actual != expected {
			mismatched = append(mismatched, rel)
		}
	}
	covered := make([]string, 0, len(entries))
	for rel := range entries {
		covered = append(covered, rel)
	}
	sort.Strings(covered)

	report := SignatureCoverageReport{
		PluginID:            strings.TrimSpace(info.ID),
		ManifestPath:        "plugin.yaml",
		ManifestDigest:      manifestDigest,
		AssetManifestPath:   filepath.ToSlash(assetManifestRel),
		AssetManifestDigest: assetManifestDigest,
		BackendEntry:        backendEntry,
		FrontendAssets:      frontendAssets,
		CoveredFiles:        covered,
		Missing:             missing,
		Mismatched:          mismatched,
	}
	if len(missing) > 0 || len(mismatched) > 0 {
		return report, fmt.Errorf("%w: missing=%v mismatched=%v", ErrSignatureCoverage, missing, mismatched)
	}
	return report, nil
}

func loadSignatureAssetManifest(path string) (SignatureAssetManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return SignatureAssetManifest{}, fmt.Errorf("%w: read asset manifest: %v", ErrSignatureCoverage, err)
	}
	var manifest SignatureAssetManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return SignatureAssetManifest{}, fmt.Errorf("%w: parse asset manifest: %v", ErrSignatureCoverage, err)
	}
	if strings.TrimSpace(manifest.SchemaVersion) != "v1" {
		return SignatureAssetManifest{}, fmt.Errorf("%w: asset manifest schema_version must be v1", ErrSignatureCoverage)
	}
	return manifest, nil
}

func normalizeSignatureAssetEntries(manifest SignatureAssetManifest) (map[string]string, error) {
	if len(manifest.Files) == 0 {
		return nil, fmt.Errorf("%w: asset manifest files are required", ErrSignatureCoverage)
	}
	out := make(map[string]string, len(manifest.Files))
	for _, entry := range manifest.Files {
		rel, err := normalizeSignatureAssetPath(entry.Path)
		if err != nil {
			return nil, err
		}
		digest := strings.TrimSpace(entry.SHA256)
		if digest == "" {
			digest = strings.TrimSpace(entry.Digest)
		}
		if !marketplaceSHA256Pattern.MatchString(digest) {
			return nil, fmt.Errorf("%w: invalid sha256 for %s", ErrSignatureCoverage, rel)
		}
		if _, exists := out[rel]; exists {
			return nil, fmt.Errorf("%w: duplicate asset entry %s", ErrSignatureCoverage, rel)
		}
		out[rel] = digest
	}
	return out, nil
}

func requiredSignatureCoverageFiles(pluginDir string, info Info) (required []string, backendEntry string, frontendAssets []string, err error) {
	required = []string{"plugin.yaml"}
	mode := info.UIMode
	if mode == "" {
		mode = UIModeBackendOnly
	}
	if mode == UIModeBackendOnly || mode == UIModeMonolith || mode == UIModeSeparated {
		backendEntry = discoverBackendEntry(pluginDir)
		if backendEntry != "" {
			required = append(required, backendEntry)
		}
	}
	if mode == UIModeFrontendOnly || mode == UIModeMonolith || mode == UIModeSeparated {
		frontendAssets, err = discoverFrontendAssets(pluginDir, mode)
		if err != nil {
			return nil, "", nil, err
		}
		required = append(required, frontendAssets...)
	}
	required = uniqueSortedStrings(required)
	return required, backendEntry, frontendAssets, nil
}

func discoverBackendEntry(pluginDir string) string {
	candidates := []string{
		"backend/main.go",
		"backend/cmd/main.go",
		"backend/server.go",
		"main.go",
	}
	for _, rel := range candidates {
		if stat, err := os.Stat(filepath.Join(pluginDir, filepath.FromSlash(rel))); err == nil && !stat.IsDir() {
			return rel
		}
	}
	return ""
}

func discoverFrontendAssets(pluginDir string, mode UIMode) ([]string, error) {
	roots := []string{}
	if mode == UIModeSeparated {
		roots = append(roots, "frontend/dist", "frontend")
	} else {
		roots = append(roots, "static", "frontend/dist")
	}
	out := make([]string, 0)
	for _, root := range roots {
		absRoot := filepath.Join(pluginDir, filepath.FromSlash(root))
		if stat, err := os.Stat(absRoot); err != nil || !stat.IsDir() {
			continue
		}
		err := filepath.WalkDir(absRoot, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				name := d.Name()
				if strings.HasPrefix(name, ".") || name == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}
			rel, err := filepath.Rel(pluginDir, path)
			if err != nil {
				return err
			}
			out = append(out, filepath.ToSlash(rel))
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("%w: scan frontend assets: %v", ErrSignatureCoverage, err)
		}
		break
	}
	sort.Strings(out)
	return out, nil
}

func normalizeSignatureAssetPath(path string) (string, error) {
	v := filepath.ToSlash(strings.TrimSpace(path))
	if v == "" || strings.HasPrefix(v, "/") || strings.Contains(v, "../") || v == ".." {
		return "", fmt.Errorf("%w: invalid asset path %q", ErrSignatureCoverage, path)
	}
	return v, nil
}

func safePluginRelativePath(root string, rel string) (string, error) {
	normalized, err := normalizeSignatureAssetPath(rel)
	if err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(filepath.Join(rootAbs, filepath.FromSlash(normalized)))
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(relative, "..") || filepath.IsAbs(relative) {
		return "", fmt.Errorf("%w: asset path escapes plugin dir", ErrSignatureCoverage)
	}
	return targetAbs, nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil)), nil
}

func uniqueSortedStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

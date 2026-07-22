package plugin

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	packageChecksumSuffix = ".sha256"
	maxPackageFileSize    = 128 << 20
	maxPackageTotalSize   = 512 << 20
)

var stablePackageTime = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)

type PackageResult struct {
	PluginID     string
	Version      string
	ArtifactPath string
	ChecksumPath string
	SHA256       string
}

// BuildPackage validates a current plugin directory and creates a deterministic ZIP and SHA-256 sidecar.
func BuildPackage(sourceDir, outputDir string, loader MetadataLoader) (PackageResult, error) {
	if loader == nil {
		loader = NewFileLoader()
	}
	sourceDir, err := cleanAbsoluteDir(sourceDir, "plugin source")
	if err != nil {
		return PackageResult{}, err
	}
	info, err := loader.Load(sourceDir)
	if err != nil {
		return PackageResult{}, fmt.Errorf("validate plugin source: %w", err)
	}
	outputDir = strings.TrimSpace(outputDir)
	if outputDir == "" {
		return PackageResult{}, errors.New("package output directory is required")
	}
	outputDir, err = filepath.Abs(outputDir)
	if err != nil {
		return PackageResult{}, fmt.Errorf("resolve package output directory: %w", err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return PackageResult{}, fmt.Errorf("create package output directory: %w", err)
	}

	artifactName := fmt.Sprintf("%s-%s.zip", info.ID, info.Version)
	artifactPath := filepath.Join(outputDir, artifactName)
	checksumPath := artifactPath + packageChecksumSuffix
	files, err := packageFiles(sourceDir, outputDir)
	if err != nil {
		return PackageResult{}, err
	}
	if strings.TrimSpace(info.ServiceBaseURL) != "" {
		if _, err := resolveManagedBackendEntry(sourceDir, info.ID); err != nil {
			return PackageResult{}, err
		}
	}
	if err := writePackageArchive(sourceDir, artifactPath, files, info.ID); err != nil {
		return PackageResult{}, err
	}
	digest, err := filePackageSHA256(artifactPath)
	if err != nil {
		return PackageResult{}, err
	}
	if err := writeAtomicFile(checksumPath, []byte(digest+"  "+artifactName+"\n"), 0o644); err != nil {
		return PackageResult{}, fmt.Errorf("write package checksum: %w", err)
	}
	return PackageResult{
		PluginID: info.ID, Version: info.Version, ArtifactPath: artifactPath,
		ChecksumPath: checksumPath, SHA256: digest,
	}, nil
}

// VerifyPackage enforces the current checksum sidecar contract and returns the verified digest.
func VerifyPackage(artifactPath, checksumPath string) (string, error) {
	artifactPath = strings.TrimSpace(artifactPath)
	checksumPath = strings.TrimSpace(checksumPath)
	if artifactPath == "" || checksumPath == "" {
		return "", errors.New("artifact and checksum paths are required")
	}
	raw, err := os.ReadFile(checksumPath)
	if err != nil {
		return "", fmt.Errorf("read package checksum: %w", err)
	}
	fields := strings.Fields(strings.TrimSpace(string(raw)))
	if len(fields) != 2 || fields[1] != filepath.Base(artifactPath) || len(fields[0]) != sha256.Size*2 {
		return "", errors.New("invalid package checksum sidecar")
	}
	if _, err := hex.DecodeString(fields[0]); err != nil {
		return "", errors.New("invalid package checksum digest")
	}
	actual, err := filePackageSHA256(artifactPath)
	if err != nil {
		return "", err
	}
	if actual != strings.ToLower(fields[0]) {
		return "", errors.New("plugin package checksum mismatch")
	}
	return actual, nil
}

// InstallPackage verifies and extracts an artifact into the current plugin directory contract.
func InstallPackage(artifactPath, checksumPath, pluginsRoot string, loader MetadataLoader) (string, Info, error) {
	if loader == nil {
		loader = NewFileLoader()
	}
	if _, err := VerifyPackage(artifactPath, checksumPath); err != nil {
		return "", Info{}, err
	}
	pluginsRoot = strings.TrimSpace(pluginsRoot)
	if pluginsRoot == "" {
		return "", Info{}, errors.New("plugins root is required")
	}
	pluginsRoot, err := filepath.Abs(pluginsRoot)
	if err != nil {
		return "", Info{}, fmt.Errorf("resolve plugins root: %w", err)
	}
	if err := os.MkdirAll(pluginsRoot, 0o755); err != nil {
		return "", Info{}, fmt.Errorf("create plugins root: %w", err)
	}
	tempDir, err := os.MkdirTemp(pluginsRoot, ".skoll-install-")
	if err != nil {
		return "", Info{}, fmt.Errorf("create package staging directory: %w", err)
	}
	keepTemp := false
	defer func() {
		if !keepTemp {
			_ = os.RemoveAll(tempDir)
		}
	}()
	if err := extractPackageArchive(artifactPath, tempDir); err != nil {
		return "", Info{}, err
	}
	info, err := loader.Load(tempDir)
	if err != nil {
		return "", Info{}, fmt.Errorf("validate extracted plugin: %w", err)
	}
	if strings.TrimSpace(info.ServiceBaseURL) != "" {
		if _, err := resolveManagedBackendEntry(tempDir, info.ID); err != nil {
			return "", Info{}, fmt.Errorf("validate managed backend entry: %w", err)
		}
	}
	targetDir := filepath.Join(pluginsRoot, info.ID)
	if _, err := os.Stat(targetDir); err == nil {
		return "", Info{}, fmt.Errorf("plugin install target already exists: %s", targetDir)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", Info{}, fmt.Errorf("inspect plugin install target: %w", err)
	}
	if err := os.Rename(tempDir, targetDir); err != nil {
		return "", Info{}, fmt.Errorf("publish plugin package: %w", err)
	}
	keepTemp = true
	installed, err := loader.Load(targetDir)
	if err != nil {
		_ = os.RemoveAll(targetDir)
		return "", Info{}, fmt.Errorf("validate installed plugin: %w", err)
	}
	return targetDir, installed, nil
}

func packageFiles(sourceDir, outputDir string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == sourceDir {
			return nil
		}
		if sameOrChildPath(outputDir, path) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		name := entry.Name()
		if name == ".git" || name == "node_modules" {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported package file type: %s", path)
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan plugin package: %w", err)
	}
	sort.Strings(files)
	if !containsString(files, "plugin.yaml") {
		return nil, errors.New("plugin package must contain plugin.yaml")
	}
	return files, nil
}

func writePackageArchive(sourceDir, artifactPath string, files []string, pluginID string) error {
	temp, err := os.CreateTemp(filepath.Dir(artifactPath), ".skoll-package-*.zip")
	if err != nil {
		return fmt.Errorf("create plugin package: %w", err)
	}
	tempPath := temp.Name()
	committed := false
	defer func() {
		_ = temp.Close()
		if !committed {
			_ = os.Remove(tempPath)
		}
	}()
	zw := zip.NewWriter(temp)
	for _, rel := range files {
		header := &zip.FileHeader{Name: rel, Method: zip.Deflate}
		header.SetModTime(stablePackageTime)
		mode := os.FileMode(0o644)
		if rel == managedBackendRelativePath(pluginID) {
			mode = 0o755
		}
		header.SetMode(mode)
		entry, err := zw.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("create package entry %s: %w", rel, err)
		}
		source, err := os.Open(filepath.Join(sourceDir, filepath.FromSlash(rel)))
		if err != nil {
			return fmt.Errorf("open package entry %s: %w", rel, err)
		}
		_, copyErr := io.Copy(entry, source)
		closeErr := source.Close()
		if copyErr != nil {
			return fmt.Errorf("write package entry %s: %w", rel, copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close package entry %s: %w", rel, closeErr)
		}
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("finalize plugin package: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close plugin package: %w", err)
	}
	if err := os.Remove(artifactPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("replace plugin package: %w", err)
	}
	if err := os.Rename(tempPath, artifactPath); err != nil {
		return fmt.Errorf("publish plugin package: %w", err)
	}
	committed = true
	return nil
}

func extractPackageArchive(artifactPath, targetDir string) error {
	reader, err := zip.OpenReader(artifactPath)
	if err != nil {
		return fmt.Errorf("open plugin package: %w", err)
	}
	defer reader.Close()
	var total uint64
	for _, entry := range reader.File {
		name := entry.Name
		if name == "" || strings.Contains(name, "\\") || strings.HasPrefix(name, "/") {
			return fmt.Errorf("invalid plugin package entry: %q", name)
		}
		clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(name)))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
			return fmt.Errorf("invalid plugin package entry: %q", name)
		}
		if entry.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("plugin package symlink is not allowed: %s", name)
		}
		if entry.UncompressedSize64 > maxPackageFileSize {
			return fmt.Errorf("plugin package entry exceeds size limit: %s", name)
		}
		total += entry.UncompressedSize64
		if total > maxPackageTotalSize {
			return errors.New("plugin package exceeds total size limit")
		}
		target := filepath.Join(targetDir, filepath.FromSlash(clean))
		if !sameOrChildPath(targetDir, target) {
			return fmt.Errorf("plugin package entry escapes target: %s", name)
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := entry.Open()
		if err != nil {
			return fmt.Errorf("open plugin package entry %s: %w", name, err)
		}
		mode := os.FileMode(0o644)
		if entry.Mode().Perm()&0o111 != 0 {
			mode = 0o755
		}
		file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
		if err != nil {
			_ = rc.Close()
			return fmt.Errorf("create plugin package entry %s: %w", name, err)
		}
		_, copyErr := io.Copy(file, io.LimitReader(rc, maxPackageFileSize+1))
		fileCloseErr := file.Close()
		rcCloseErr := rc.Close()
		if copyErr != nil || fileCloseErr != nil || rcCloseErr != nil {
			return fmt.Errorf("extract plugin package entry %s: %w", name, errors.Join(copyErr, fileCloseErr, rcCloseErr))
		}
	}
	return nil
}

func cleanAbsoluteDir(path, label string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("%s directory is required", label)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s directory: %w", label, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("inspect %s directory: %w", label, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s path must be a directory", label)
	}
	return filepath.Clean(abs), nil
}

func sameOrChildPath(root, target string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func containsString(items []string, wanted string) bool {
	index := sort.SearchStrings(items, wanted)
	return index < len(items) && items[index] == wanted
}

func filePackageSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open package for checksum: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("calculate package checksum: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func writeAtomicFile(path string, content []byte, mode fs.FileMode) error {
	temp, err := os.CreateTemp(filepath.Dir(path), ".skoll-checksum-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	committed := false
	defer func() {
		_ = temp.Close()
		if !committed {
			_ = os.Remove(tempPath)
		}
	}()
	if err := temp.Chmod(mode); err != nil {
		return err
	}
	if _, err := temp.Write(content); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	committed = true
	return nil
}

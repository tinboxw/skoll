package pluginmgr

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

var ErrPluginNotFound = errors.New("plugin not found")
var ErrInvalidPluginVersion = errors.New("invalid plugin version")
var ErrPluginSignatureInvalid = errors.New("plugin signature verification failed")
var ErrPluginDependencyUnsatisfied = errors.New("plugin dependency precheck failed")

type VersionCheckResult struct {
	Name            string `json:"name"`
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	UpdateAvailable bool   `json:"update_available"`
}

type Manifest struct {
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	Hooks        []string     `json:"hooks"`
	Enabled      bool         `json:"enabled"`
	PackageURL   string       `json:"package_url,omitempty"`
	PackageHash  string       `json:"package_hash,omitempty"`
	Signature    string       `json:"signature,omitempty"`
	Dependencies []Dependency `json:"dependencies,omitempty"`
	InstalledAt  time.Time    `json:"installed_at"`
}

type Dependency struct {
	Name       string `json:"name"`
	MinVersion string `json:"min_version"`
}

type UpgradeResult struct {
	Name            string `json:"name"`
	PreviousVersion string `json:"previous_version"`
	TargetVersion   string `json:"target_version"`
	Succeeded       bool   `json:"succeeded"`
	RolledBack      bool   `json:"rolled_back"`
	Reason          string `json:"reason,omitempty"`
}

type Service struct {
	mu    sync.RWMutex
	items map[string]Manifest
}

func NewService() *Service {
	return &Service{items: make(map[string]Manifest)}
}

func (s *Service) Install(name, version string, hooks []string) Manifest {
	item, _ := s.install(name, version, "", "", "", nil, hooks)
	return item
}

func (s *Service) InstallPackage(name, version, packageURL, packageHash string, hooks []string) (Manifest, error) {
	return s.InstallPackageVerified(name, version, packageURL, packageHash, "", nil, hooks)
}

func (s *Service) InstallPackageVerified(name, version, packageURL, packageHash, signature string, dependencies []Dependency, hooks []string) (Manifest, error) {
	if signature != "" && !verifySignature(packageHash, signature) {
		return Manifest{}, ErrPluginSignatureInvalid
	}
	if err := s.precheckDependencies(dependencies); err != nil {
		return Manifest{}, err
	}
	return s.install(name, version, packageURL, packageHash, signature, dependencies, hooks)
}

func (s *Service) install(name, version, packageURL, packageHash, signature string, dependencies []Dependency, hooks []string) (Manifest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	packageURL = strings.TrimSpace(packageURL)
	packageHash = strings.TrimSpace(packageHash)
	if !isValidVersion(version) {
		return Manifest{}, ErrInvalidPluginVersion
	}
	if packageURL != "" && packageHash == "" {
		return Manifest{}, errors.New("package_hash is required when package_url is set")
	}
	if packageURL == "" && packageHash != "" {
		return Manifest{}, errors.New("package_url is required when package_hash is set")
	}
	item := Manifest{
		Name:         name,
		Version:      version,
		Hooks:        append([]string(nil), hooks...),
		Enabled:      true,
		PackageURL:   packageURL,
		PackageHash:  packageHash,
		Signature:    strings.TrimSpace(signature),
		Dependencies: cloneDependencies(dependencies),
		InstalledAt:  time.Now().UTC(),
	}
	s.items[name] = item
	return item, nil
}

func (s *Service) UpgradePackage(name, targetVersion, packageURL, packageHash, signature string, dependencies []Dependency, hooks []string) (UpgradeResult, error) {
	current, err := s.Get(name)
	if err != nil {
		return UpgradeResult{}, err
	}
	result := UpgradeResult{Name: current.Name, PreviousVersion: current.Version, TargetVersion: strings.TrimSpace(targetVersion)}

	_, err = s.InstallPackageVerified(name, targetVersion, packageURL, packageHash, signature, dependencies, hooks)
	if err != nil {
		s.mu.Lock()
		s.items[name] = current
		s.mu.Unlock()
		result.Succeeded = false
		result.RolledBack = true
		result.Reason = err.Error()
		return result, nil
	}

	result.Succeeded = true
	return result, nil
}

func (s *Service) Get(name string) (Manifest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[strings.TrimSpace(name)]
	if !ok {
		return Manifest{}, ErrPluginNotFound
	}
	return item, nil
}

func (s *Service) List() []Manifest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Manifest, 0, len(s.items))
	for _, item := range s.items {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) Enable(name string) (Manifest, error) {
	return s.setEnabled(name, true)
}

func (s *Service) Disable(name string) (Manifest, error) {
	return s.setEnabled(name, false)
}

func (s *Service) CheckVersion(name, latestVersion string) (VersionCheckResult, error) {
	current, err := s.Get(name)
	if err != nil {
		return VersionCheckResult{}, err
	}
	latestVersion = strings.TrimSpace(latestVersion)
	if !isValidVersion(latestVersion) {
		return VersionCheckResult{}, ErrInvalidPluginVersion
	}
	cmp := compareVersion(current.Version, latestVersion)
	return VersionCheckResult{
		Name:            current.Name,
		CurrentVersion:  current.Version,
		LatestVersion:   latestVersion,
		UpdateAvailable: cmp < 0,
	}, nil
}

func (s *Service) setEnabled(name string, enabled bool) (Manifest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := strings.TrimSpace(name)
	item, ok := s.items[key]
	if !ok {
		return Manifest{}, ErrPluginNotFound
	}
	item.Enabled = enabled
	s.items[key] = item
	return item, nil
}

func isValidVersion(version string) bool {
	v := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(version, "v"), "V"))
	if v == "" {
		return false
	}
	parts := strings.Split(v, ".")
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, ch := range p {
			if ch < '0' || ch > '9' {
				return false
			}
		}
	}
	return true
}

func compareVersion(current, latest string) int {
	c := parseVersionParts(current)
	l := parseVersionParts(latest)
	maxLen := len(c)
	if len(l) > maxLen {
		maxLen = len(l)
	}
	for i := 0; i < maxLen; i++ {
		cv := 0
		if i < len(c) {
			cv = c[i]
		}
		lv := 0
		if i < len(l) {
			lv = l[i]
		}
		if cv < lv {
			return -1
		}
		if cv > lv {
			return 1
		}
	}
	return 0
}

func parseVersionParts(version string) []int {
	v := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(version, "v"), "V"))
	parts := strings.Split(v, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		var n int
		for _, ch := range p {
			n = n*10 + int(ch-'0')
		}
		out = append(out, n)
	}
	return out
}

func verifySignature(packageHash, signature string) bool {
	packageHash = strings.TrimSpace(packageHash)
	signature = strings.TrimSpace(signature)
	if packageHash == "" || signature == "" {
		return false
	}
	return signature == "sig:"+packageHash
}

func (s *Service) precheckDependencies(dependencies []Dependency) error {
	if len(dependencies) == 0 {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, dep := range dependencies {
		depName := strings.TrimSpace(dep.Name)
		depMin := strings.TrimSpace(dep.MinVersion)
		if depName == "" || depMin == "" {
			continue
		}
		manifest, ok := s.items[depName]
		if !ok {
			return fmt.Errorf("%w: dependency %s not installed", ErrPluginDependencyUnsatisfied, depName)
		}
		if compareVersion(manifest.Version, depMin) < 0 {
			return fmt.Errorf("%w: dependency %s requires >= %s", ErrPluginDependencyUnsatisfied, depName, depMin)
		}
	}
	return nil
}

func cloneDependencies(in []Dependency) []Dependency {
	if len(in) == 0 {
		return nil
	}
	out := make([]Dependency, len(in))
	copy(out, in)
	return out
}

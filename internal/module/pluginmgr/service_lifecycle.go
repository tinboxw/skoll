package pluginmgr

import (
	"errors"
	"sort"
	"strings"
	"time"
)

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

func (s *Service) Remove(name string) LifecycleResult {
	name = strings.TrimSpace(name)
	if name == "" {
		return LifecycleResult{Name: name, Action: "remove", Succeeded: false, Message: "name is required"}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[name]; !ok {
		return LifecycleResult{Name: name, Action: "remove", Succeeded: true, Idempotent: true, Message: "plugin already absent"}
	}
	delete(s.items, name)
	return LifecycleResult{Name: name, Action: "remove", Succeeded: true}
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

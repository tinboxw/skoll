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
var ErrHookNotFound = errors.New("hook not found")
var ErrHookNamespaceRequired = errors.New("hook namespace is required")
var ErrHookNameRequired = errors.New("hook name is required")

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

type LifecycleResult struct {
	Name       string `json:"name"`
	Action     string `json:"action"`
	Succeeded  bool   `json:"succeeded"`
	Idempotent bool   `json:"idempotent"`
	Message    string `json:"message,omitempty"`
}

type CompatibilityResult struct {
	Name       string    `json:"name"`
	Version    string    `json:"version"`
	Compatible bool      `json:"compatible"`
	Blockers   []string  `json:"blockers,omitempty"`
	CheckedAt  time.Time `json:"checked_at"`
}

type HookRegistration struct {
	Name          string    `json:"name"`
	Namespace     string    `json:"namespace"`
	Version       string    `json:"version"`
	Enabled       bool      `json:"enabled"`
	Order         int       `json:"order"`
	TimeoutMillis int       `json:"timeout_millis"`
	RetryLimit    int       `json:"retry_limit"`
	DeadLetter    bool      `json:"dead_letter"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Service struct {
	mu    sync.RWMutex
	items map[string]Manifest
	hooks map[string]HookRegistration
}

func NewService() *Service {
	return &Service{items: make(map[string]Manifest), hooks: make(map[string]HookRegistration)}
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

func (s *Service) CheckCompatibility(name, version string, dependencies []Dependency) CompatibilityResult {
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	result := CompatibilityResult{Name: name, Version: version, CheckedAt: time.Now().UTC()}
	blockers := make([]string, 0)

	if name == "" {
		blockers = append(blockers, "name is required")
	}
	if version == "" {
		blockers = append(blockers, "version is required")
	} else if !isValidVersion(version) {
		blockers = append(blockers, "version is invalid")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, dep := range dependencies {
		depName := strings.TrimSpace(dep.Name)
		depMin := strings.TrimSpace(dep.MinVersion)
		if depName == "" {
			continue
		}
		if depName == name {
			blockers = append(blockers, "self dependency is not allowed")
			continue
		}
		manifest, ok := s.items[depName]
		if !ok {
			blockers = append(blockers, fmt.Sprintf("dependency %s not installed", depName))
			continue
		}
		if depMin != "" && !isValidVersion(depMin) {
			blockers = append(blockers, fmt.Sprintf("dependency %s has invalid min_version", depName))
			continue
		}
		if depMin != "" && compareVersion(manifest.Version, depMin) < 0 {
			blockers = append(blockers, fmt.Sprintf("dependency %s requires >= %s", depName, depMin))
		}
	}

	sort.Strings(blockers)
	result.Blockers = blockers
	result.Compatible = len(blockers) == 0
	return result
}

func (s *Service) RegisterHook(name, namespace, version string, order, timeoutMillis, retryLimit int, deadLetter bool) (HookRegistration, error) {
	name = strings.TrimSpace(name)
	namespace = strings.TrimSpace(namespace)
	version = strings.TrimSpace(version)
	if name == "" {
		return HookRegistration{}, ErrHookNameRequired
	}
	if namespace == "" {
		return HookRegistration{}, ErrHookNamespaceRequired
	}
	if version == "" {
		version = "1.0.0"
	}
	if !isValidVersion(version) {
		return HookRegistration{}, ErrInvalidPluginVersion
	}
	if order < 0 {
		order = 0
	}
	if timeoutMillis <= 0 {
		timeoutMillis = 500
	}
	if timeoutMillis > 60000 {
		timeoutMillis = 60000
	}
	if retryLimit < 0 {
		retryLimit = 0
	}
	if retryLimit > 10 {
		retryLimit = 10
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	item := HookRegistration{
		Name:          name,
		Namespace:     namespace,
		Version:       version,
		Enabled:       true,
		Order:         order,
		TimeoutMillis: timeoutMillis,
		RetryLimit:    retryLimit,
		DeadLetter:    deadLetter,
		UpdatedAt:     time.Now().UTC(),
	}
	s.hooks[hookKey(namespace, name)] = item
	return item, nil
}

func (s *Service) ListHooks() []HookRegistration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]HookRegistration, 0, len(s.hooks))
	for _, item := range s.hooks {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Namespace == out[j].Namespace {
			if out[i].Order == out[j].Order {
				return out[i].Name < out[j].Name
			}
			return out[i].Order < out[j].Order
		}
		return out[i].Namespace < out[j].Namespace
	})
	return out
}

func (s *Service) SetHookEnabled(name, namespace string, enabled bool) (HookRegistration, error) {
	return s.updateHook(name, namespace, func(item HookRegistration) HookRegistration {
		item.Enabled = enabled
		return item
	})
}

func (s *Service) SetHookOrder(name, namespace string, order int) (HookRegistration, error) {
	if order < 0 {
		order = 0
	}
	return s.updateHook(name, namespace, func(item HookRegistration) HookRegistration {
		item.Order = order
		return item
	})
}

func (s *Service) SetHookRuntimePolicy(name, namespace string, timeoutMillis, retryLimit int, deadLetter bool) (HookRegistration, error) {
	if timeoutMillis <= 0 {
		timeoutMillis = 500
	}
	if timeoutMillis > 60000 {
		timeoutMillis = 60000
	}
	if retryLimit < 0 {
		retryLimit = 0
	}
	if retryLimit > 10 {
		retryLimit = 10
	}
	return s.updateHook(name, namespace, func(item HookRegistration) HookRegistration {
		item.TimeoutMillis = timeoutMillis
		item.RetryLimit = retryLimit
		item.DeadLetter = deadLetter
		return item
	})
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

func (s *Service) updateHook(name, namespace string, fn func(HookRegistration) HookRegistration) (HookRegistration, error) {
	name = strings.TrimSpace(name)
	namespace = strings.TrimSpace(namespace)
	if name == "" {
		return HookRegistration{}, ErrHookNameRequired
	}
	if namespace == "" {
		return HookRegistration{}, ErrHookNamespaceRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	key := hookKey(namespace, name)
	item, ok := s.hooks[key]
	if !ok {
		return HookRegistration{}, ErrHookNotFound
	}
	item = fn(item)
	item.UpdatedAt = time.Now().UTC()
	s.hooks[key] = item
	return item, nil
}

func hookKey(namespace, name string) string {
	return strings.TrimSpace(namespace) + "::" + strings.TrimSpace(name)
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

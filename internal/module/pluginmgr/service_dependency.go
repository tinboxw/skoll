package pluginmgr

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

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

func (s *Service) SolveDependencies(items []DependencySolveItem) DependencySolveResult {
	resolved := make(map[string]string)
	conflicts := make([]string, 0)

	s.mu.RLock()
	for name, manifest := range s.items {
		resolved[name] = manifest.Version
	}
	s.mu.RUnlock()

	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		version := strings.TrimSpace(item.Version)
		if name == "" {
			conflicts = append(conflicts, "candidate name is required")
			continue
		}
		if version == "" || !isValidVersion(version) {
			conflicts = append(conflicts, fmt.Sprintf("candidate %s has invalid version", name))
			continue
		}
		if existing, ok := resolved[name]; ok && existing != version {
			conflicts = append(conflicts, fmt.Sprintf("version conflict for %s: %s vs %s", name, existing, version))
			continue
		}
		resolved[name] = version
	}

	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		for _, dep := range item.Dependencies {
			depName := strings.TrimSpace(dep.Name)
			depMin := strings.TrimSpace(dep.MinVersion)
			if depName == "" {
				continue
			}
			depVersion, ok := resolved[depName]
			if !ok {
				conflicts = append(conflicts, fmt.Sprintf("dependency missing for %s: %s", name, depName))
				continue
			}
			if depMin != "" {
				if !isValidVersion(depMin) {
					conflicts = append(conflicts, fmt.Sprintf("dependency %s of %s has invalid min_version", depName, name))
					continue
				}
				if compareVersion(depVersion, depMin) < 0 {
					conflicts = append(conflicts, fmt.Sprintf("dependency version conflict for %s: %s requires %s >= %s", name, depName, depVersion, depMin))
				}
			}
		}
	}

	resolvedList := make([]string, 0, len(resolved))
	for name, version := range resolved {
		resolvedList = append(resolvedList, fmt.Sprintf("%s@%s", name, version))
	}
	sort.Strings(resolvedList)
	conflicts = normalizeStringList(conflicts)

	return DependencySolveResult{
		Deterministic: true,
		Resolved:      resolvedList,
		Conflicts:     conflicts,
	}
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

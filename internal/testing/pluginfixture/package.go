package pluginfixture

import (
	"fmt"
	"strings"

	pluginruntime "github.com/tinboxw/skoll/internal/plugin"
)

type InstalledPackage struct {
	Build     pluginruntime.PackageResult
	Directory string
	Info      pluginruntime.Info
}

type PackageRunner struct {
	Loader pluginruntime.MetadataLoader
}

func NewPackageRunner() PackageRunner {
	return PackageRunner{Loader: pluginruntime.NewFileLoader()}
}

func (r PackageRunner) BuildAndInstall(sourceDir, outputDir, pluginsRoot string) (InstalledPackage, error) {
	loader := r.loader()
	build, err := pluginruntime.BuildPackage(sourceDir, outputDir, loader)
	if err != nil {
		return InstalledPackage{}, fmt.Errorf("build fixture plugin package: %w", err)
	}
	return r.install(build, pluginsRoot, loader)
}

func (r PackageRunner) Install(artifactPath, checksumPath, pluginsRoot string) (InstalledPackage, error) {
	loader := r.loader()
	digest, err := pluginruntime.VerifyPackage(artifactPath, checksumPath)
	if err != nil {
		return InstalledPackage{}, fmt.Errorf("verify fixture plugin package: %w", err)
	}
	return r.install(pluginruntime.PackageResult{
		ArtifactPath: artifactPath,
		ChecksumPath: checksumPath,
		SHA256:       digest,
	}, pluginsRoot, loader)
}

func (r PackageRunner) install(build pluginruntime.PackageResult, pluginsRoot string, loader pluginruntime.MetadataLoader) (InstalledPackage, error) {
	directory, info, err := pluginruntime.InstallPackage(build.ArtifactPath, build.ChecksumPath, pluginsRoot, loader)
	if err != nil {
		return InstalledPackage{}, fmt.Errorf("install fixture plugin package: %w", err)
	}
	if build.PluginID != "" && build.PluginID != info.ID {
		return InstalledPackage{}, fmt.Errorf("fixture package identity changed from %q to %q", build.PluginID, info.ID)
	}
	if build.Version != "" && build.Version != info.Version {
		return InstalledPackage{}, fmt.Errorf("fixture package version changed from %q to %q", build.Version, info.Version)
	}
	build.PluginID = info.ID
	build.Version = info.Version
	return InstalledPackage{Build: build, Directory: directory, Info: info}, nil
}

func (r PackageRunner) loader() pluginruntime.MetadataLoader {
	if r.Loader == nil {
		return pluginruntime.NewFileLoader()
	}
	return r.Loader
}

func ValidateInstalledPackage(installed InstalledPackage) error {
	if strings.TrimSpace(installed.Directory) == "" || strings.TrimSpace(installed.Info.ID) == "" {
		return fmt.Errorf("installed fixture plugin is incomplete")
	}
	if installed.Build.PluginID != installed.Info.ID || installed.Build.Version != installed.Info.Version {
		return fmt.Errorf("installed fixture package metadata is inconsistent")
	}
	return nil
}

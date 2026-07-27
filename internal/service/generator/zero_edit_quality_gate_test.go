//go:build pluginquality

package generator

import "testing"

func TestGeneratedPluginZeroEditQualityGate(t *testing.T) {
	t.Setenv("SKOLL_GENERATOR_PLUGIN_E2E", "1")
	t.Setenv("SKOLL_GENERATOR_PLUGIN_FRONTEND_E2E", "1")

	t.Run("package-lifecycle", TestGeneratedPluginBuildPackageAndInstallWithoutSourceEdits)
	t.Run("frontend-browser", TestGeneratedPluginFrontendBuildAndBrowserMatrix)
}

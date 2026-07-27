package generator

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGeneratedPluginFrontendUsesCurrentPublicContracts(t *testing.T) {
	result, err := NewService().DryRun(context.Background(), DryRunInput{
		Spec: mustPluginSpec(t), BatchID: "ff4-generated-plugin-contract", ActorID: "generator-test", MigrationTimestamp: "20260722_010203",
	})
	if err != nil {
		t.Fatalf("DryRun() error = %v", err)
	}

	packageJSON := findPlan(t, result.Files, "examples/plugins/pharma-oa/web/package.json").GeneratedContent
	vite := findPlan(t, result.Files, "examples/plugins/pharma-oa/web/vite.config.ts").GeneratedContent
	api := findPlan(t, result.Files, "examples/plugins/pharma-oa/web/src/utils/api.ts").GeneratedContent
	host := findPlan(t, result.Files, "examples/plugins/pharma-oa/web/src/skoll-host.ts").GeneratedContent
	view := findPlan(t, result.Files, "examples/plugins/pharma-oa/web/src/views/Product/index.vue").GeneratedContent
	all := packageJSON + vite + api + host + view
	for _, marker := range []string{
		`"@skoll/business-ui"`,
		`"@skoll/plugin-sdk"`,
		`check-plugin-frontend-bundle.mjs`,
		`@skoll/business-ui/core`,
		`getPluginHost`,
		`requiredCapabilities: ["request"]`,
		`"skoll:locale"`,
		`"skoll:theme"`,
		`"skoll:lifecycle"`,
		`BusinessWorkspace`,
		`BusinessFilterBar`,
		`BusinessList`,
		`BusinessCommandBar`,
	} {
		if !strings.Contains(all, marker) {
			t.Fatalf("generated frontend public contract missing %q", marker)
		}
	}
	for _, forbidden := range []string{
		"window.__SKOLL_HOST__",
		"__SKOLL_PLUGIN_PERMISSIONS__",
		`../../components/Common`,
		`../../permissions/button`,
		`v-permission`,
		`fetch(prefix`,
	} {
		if strings.Contains(all, forbidden) {
			t.Fatalf("generated frontend contains obsolete contract %q", forbidden)
		}
	}
	for _, removedPath := range []string{
		"examples/plugins/pharma-oa/web/src/components/Common/index.ts",
		"examples/plugins/pharma-oa/web/src/i18n/index.ts",
		"examples/plugins/pharma-oa/web/src/permissions/button.ts",
		"examples/plugins/pharma-oa/web/src/router/generated_product.ts",
	} {
		if hasPlanPath(result.Files, removedPath) {
			t.Fatalf("generator still emits obsolete file %s", removedPath)
		}
	}
}

func TestGeneratedPluginFrontendBuildAndBrowserMatrix(t *testing.T) {
	if testing.Short() || os.Getenv("SKOLL_GENERATOR_PLUGIN_FRONTEND_E2E") != "1" {
		t.Skip("set SKOLL_GENERATOR_PLUGIN_FRONTEND_E2E=1 to run generated plugin frontend build and browser matrix")
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	webModules := filepath.Join(repoRoot, "web", "node_modules")
	if stat, err := os.Stat(webModules); err != nil || !stat.IsDir() {
		t.Skip("web/node_modules is required for generated plugin frontend E2E")
	}
	result, err := NewService().DryRun(context.Background(), DryRunInput{
		Spec: mustPluginSpec(t), BatchID: "ff4-generated-plugin-browser", ActorID: "generator-test", MigrationTimestamp: "20260722_010203",
	})
	if err != nil {
		t.Fatalf("DryRun() error = %v", err)
	}
	pluginDir := materializeGeneratedPlugin(t, result, "pharma-oa")
	linkGeneratedFrontendWorkspace(t, repoRoot, pluginDir)
	runGeneratedGoTests(t, pluginDir, generatedGoWorkspace(t, repoRoot, pluginDir))
	pluginWeb := filepath.Join(pluginDir, "web")
	linkNodeModules(t, webModules, filepath.Join(pluginWeb, "node_modules"))
	before := generatedSourceHashes(t, pluginDir)

	npm := "npm"
	if runtime.GOOS == "windows" {
		npm = "npm.cmd"
	}
	build := exec.Command(npm, "run", "build")
	build.Dir = pluginWeb
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("generated plugin frontend build failed: %v\n%s", err, output)
	} else {
		t.Logf("generated plugin frontend build:\n%s", output)
	}
	after := generatedSourceHashes(t, pluginDir)
	if strings.Join(before, "\n") != strings.Join(after, "\n") {
		t.Fatalf("generated source changed during frontend build\nbefore=%v\nafter=%v", before, after)
	}

	node, err := exec.LookPath("node")
	if err != nil {
		t.Skipf("node is not installed: %v", err)
	}
	browser := exec.Command(node, filepath.Join(repoRoot, "web", "scripts", "ff4-generated-plugin-browser.mjs"), pluginWeb, "pharma-oa")
	browser.Dir = filepath.Join(repoRoot, "web")
	if output, err := browser.CombinedOutput(); err != nil {
		t.Fatalf("generated plugin browser matrix failed: %v\n%s", err, output)
	} else {
		t.Logf("generated plugin browser matrix:\n%s", output)
	}
}

func hasPlanPath(files []FilePlan, path string) bool {
	for _, file := range files {
		if file.Path == path {
			return true
		}
	}
	return false
}

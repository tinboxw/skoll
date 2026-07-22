package generator

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestGeneratedFrontendTypechecksAndBuilds(t *testing.T) {
	if testing.Short() {
		t.Skip("generated frontend build is disabled in short mode")
	}
	if os.Getenv("SKOLL_GENERATOR_FRONTEND_BUILD") != "1" {
		t.Skip("set SKOLL_GENERATOR_FRONTEND_BUILD=1 to run generated frontend typecheck, build, and browser smoke")
	}
	webRoot, err := filepath.Abs(filepath.Join("..", "..", "..", "web"))
	if err != nil {
		t.Fatalf("resolve web workspace: %v", err)
	}
	toolExtension := ""
	if runtime.GOOS == "windows" {
		toolExtension = ".cmd"
	}
	vueTSC := filepath.Join(webRoot, "node_modules", ".bin", "vue-tsc"+toolExtension)
	vite := filepath.Join(webRoot, "node_modules", ".bin", "vite"+toolExtension)
	if _, err := os.Stat(vueTSC); err != nil {
		t.Skipf("vue-tsc is not installed: %v", err)
	}
	if _, err := os.Stat(vite); err != nil {
		t.Skipf("vite is not installed: %v", err)
	}

	result, err := NewService().DryRun(context.Background(), DryRunInput{
		Spec:               loadDemoProductSpec(t),
		BatchID:            "pr3-generated-frontend-build",
		ActorID:            "generator-test",
		MigrationTimestamp: "20260722_010203",
	})
	if err != nil {
		t.Fatalf("DryRun() error = %v", err)
	}

	root, err := os.MkdirTemp(webRoot, ".pr3-generated-ui-")
	if err != nil {
		t.Fatalf("create generated frontend workspace: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })

	for _, path := range []string{
		"web/src/api/demo_product.ts",
		"web/src/stores/demo_product.ts",
		"web/src/i18n/generated_demo_product.ts",
		"web/src/router/generated_demo_product.ts",
		"web/src/views/DemoProduct/index.vue",
	} {
		plan := findPlan(t, result.Files, path)
		writeGeneratedFrontendFile(t, root, strings.TrimPrefix(path, "web/"), plan.GeneratedContent)
	}
	writeGeneratedFrontendSupport(t, root)

	runGeneratedFrontendCommand(t, webRoot, vueTSC, "--noEmit", "-p", filepath.Join(root, "tsconfig.json"))
	runGeneratedFrontendCommand(t, webRoot, vite, "build", root, "--config", filepath.Join(root, "vite.config.ts"))
	runGeneratedFrontendBrowserSmoke(t, root, vite)
}

func writeGeneratedFrontendSupport(t *testing.T, root string) {
	t.Helper()
	files := map[string]string{
		"index.html":                     `<div id="app"></div><script type="module" src="/src/main.ts"></script>`,
		"tsconfig.json":                  `{"compilerOptions":{"target":"ES2020","module":"ESNext","moduleResolution":"Bundler","strict":true,"skipLibCheck":true,"lib":["ES2020","DOM","DOM.Iterable"]},"include":["src/**/*.ts","src/**/*.vue"]}`,
		"vite.config.ts":                 `import { defineConfig } from "vite"; import vue from "@vitejs/plugin-vue"; export default defineConfig({ plugins: [vue()] });`,
		"src/main.ts":                    `import { createApp } from "vue"; import { createPinia } from "pinia"; import ElementPlus from "element-plus"; import Page from "./views/DemoProduct/index.vue"; createApp(Page).use(createPinia()).use(ElementPlus).mount("#app");`,
		"src/utils/api.ts":               `export type ApiResponse<T> = { code: string; message: string; data: T }; export async function apiGet<T>(_url: string): Promise<T> { throw new Error("fixture"); } export async function apiPost<T>(_url: string, _body?: unknown): Promise<T> { throw new Error("fixture"); } export async function apiPut<T>(_url: string, _body?: unknown): Promise<T> { throw new Error("fixture"); } export async function apiDelete<T>(_url: string): Promise<T> { throw new Error("fixture"); }`,
		"src/utils/common.ts":            `export function toErrorMessage(error: unknown): string { return error instanceof Error ? error.message : String(error); }`,
		"src/i18n/index.ts":              `import { computed } from "vue"; export function useI18n() { return { locale: computed(() => "en-US") }; }`,
		"src/permissions/button.ts":      `export function useButtonAccess() { return { can: (_permission: string): boolean => true }; }`,
		"src/components/Common/index.ts": `import { defineComponent, h } from "vue"; export type DataTableColumn = { key: string; label: string; minWidth?: number }; const component = () => defineComponent({ setup(_props, { slots }) { return () => h("section", { class: "fixture-component" }, [slots.default?.(), slots.actions?.({ row: { id: "fixture" } }), slots.pagination?.(), slots.footer?.()]); } }); export const ConfirmAction = component(); export const DataTable = component(); export const DetailDrawer = component(); export const FilterBar = component(); export const PageShell = component();`,
	}
	for path, content := range files {
		writeGeneratedFrontendFile(t, root, path, content)
	}
}

func writeGeneratedFrontendFile(t *testing.T, root, relativePath, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create directory for %s: %v", relativePath, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", relativePath, err)
	}
}

func runGeneratedFrontendCommand(t *testing.T, workdir, command string, args ...string) {
	t.Helper()
	cmd := exec.Command(command, args...)
	cmd.Dir = workdir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated frontend command failed: %s %s\n%s\n%v", command, strings.Join(args, " "), output, err)
	}
}

func runGeneratedFrontendBrowserSmoke(t *testing.T, root, vite string) {
	t.Helper()
	chrome := findHeadlessChrome()
	if chrome == "" {
		t.Skip("Chrome or Chromium is not installed")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve browser smoke port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	var serverOutput bytes.Buffer
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skipf("node is not installed: %v", err)
	}
	viteEntry := filepath.Join(filepath.Dir(filepath.Dir(vite)), "vite", "bin", "vite.js")
	server := exec.Command(node, viteEntry, "preview", root, "--host", "127.0.0.1", "--port", fmt.Sprint(port), "--strictPort")
	server.Stdout = &serverOutput
	server.Stderr = &serverOutput
	if err := server.Start(); err != nil {
		t.Fatalf("start generated frontend preview: %v", err)
	}
	t.Cleanup(func() {
		if server.Process != nil {
			_ = server.Process.Kill()
		}
		_ = server.Wait()
	})

	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(15 * time.Second)
	for {
		response, requestErr := http.Get(url)
		if requestErr == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("generated frontend preview did not become ready: %v\n%s", requestErr, serverOutput.String())
		}
		time.Sleep(200 * time.Millisecond)
	}

	desktopContext, cancelDesktop := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelDesktop()
	dump := exec.CommandContext(desktopContext, chrome,
		"--headless=new", "--disable-gpu", "--no-sandbox", "--disable-dev-shm-usage", "--no-first-run",
		"--user-data-dir="+filepath.Join(root, "chrome-desktop"), "--window-size=1440,900", "--virtual-time-budget=3000", "--dump-dom", url,
	)
	dom, err := dump.CombinedOutput()
	if err != nil {
		t.Fatalf("desktop browser smoke failed: %v\n%s", err, dom)
	}
	for _, marker := range []string{"generated-page", "generated-pagination", "el-button"} {
		if !bytes.Contains(dom, []byte(marker)) {
			t.Fatalf("desktop browser smoke is missing %q\n%s", marker, dom)
		}
	}

	screenshot := filepath.Join(root, "generated-ui-390x844.png")
	narrowContext, cancelNarrow := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelNarrow()
	narrow := exec.CommandContext(narrowContext, chrome,
		"--headless=new", "--disable-gpu", "--no-sandbox", "--disable-dev-shm-usage", "--no-first-run",
		"--user-data-dir="+filepath.Join(root, "chrome-narrow"), "--window-size=390,844", "--virtual-time-budget=3000", "--screenshot="+screenshot, url,
	)
	if output, err := narrow.CombinedOutput(); err != nil {
		t.Fatalf("narrow browser smoke failed: %v\n%s", err, output)
	}
	info, err := os.Stat(screenshot)
	if err != nil || info.Size() < 1024 {
		t.Fatalf("narrow browser screenshot is missing or empty: %v", err)
	}
}

func findHeadlessChrome() string {
	candidates := []string{"google-chrome", "chromium", "chromium-browser", "chrome", "msedge"}
	if runtime.GOOS == "windows" {
		candidates = append([]string{
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
			`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		}, candidates...)
	}
	for _, candidate := range candidates {
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}
	return ""
}

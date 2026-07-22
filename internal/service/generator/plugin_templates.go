package generator

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	domaingenerator "github.com/tinboxw/skoll/internal/domain/generator"
)

func renderPluginGoMod(spec domaingenerator.GeneratorSpec) string {
	if spec.Document != nil {
		return fmt.Sprintf("module example.com/skoll-plugins/%s\n\ngo 1.24\n\nrequire github.com/tinboxw/skoll v0.0.0\n\nreplace github.com/tinboxw/skoll => ../../..\n", spec.Plugin.ID)
	}
	return fmt.Sprintf("module example.com/skoll-plugins/%s\n\ngo 1.24\n", spec.Plugin.ID)
}

func renderPluginBackendServer(spec domaingenerator.GeneratorSpec) string {
	template := `package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const (
	apiPath = {{API_PATH}}
	defaultAddress = {{DEFAULT_ADDRESS}}
)

type memoryStore struct {
	mu sync.RWMutex
	sequence uint64
	items map[string]map[string]any
}

func main() {
	store := &memoryStore{items: make(map[string]map[string]any)}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ready", "plugin": {{PLUGIN_ID}}})
	})
	mux.HandleFunc(apiPath, store.collection)
	mux.HandleFunc(apiPath+"/", store.item)
	address := strings.TrimSpace(os.Getenv("SKOLL_PLUGIN_ADDRESS"))
	if address == "" { address = defaultAddress }
	log.Printf("%s listening on %s", {{PLUGIN_ID}}, address)
	log.Fatal(http.ListenAndServe(address, mux))
}

func (s *memoryStore) collection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if offset < 0 { offset = 0 }
		if limit <= 0 { limit = 20 }
		keyword := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("keyword")))
		s.mu.RLock()
		ids := make([]string, 0, len(s.items))
		for id, item := range s.items {
			if keyword == "" || strings.Contains(strings.ToLower(fmt.Sprint(item)), keyword) { ids = append(ids, id) }
		}
		sort.Strings(ids)
		items := make([]map[string]any, 0, limit)
		for index := offset; index < len(ids) && len(items) < limit; index++ { items = append(items, cloneItem(s.items[ids[index]])) }
		s.mu.RUnlock()
		writeOK(w, map[string]any{"items": items, "offset": offset, "limit": limit})
	case http.MethodPost:
		input, ok := decodeItem(w, r)
		if !ok { return }
		s.mu.Lock()
		id := strings.TrimSpace(fmt.Sprint(input[{{PRIMARY_FIELD}}]))
		if id == "" || id == "<nil>" { s.sequence++; id = fmt.Sprintf("generated-%d", s.sequence); input[{{PRIMARY_FIELD}}] = id }
		if _, exists := s.items[id]; exists { s.mu.Unlock(); writeError(w, http.StatusConflict, "already_exists"); return }
		s.items[id] = cloneItem(input)
		item := cloneItem(input)
		s.mu.Unlock()
		writeJSON(w, http.StatusCreated, envelope(map[string]any{"item": item}))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *memoryStore) item(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, apiPath+"/"))
	if id == "" || strings.Contains(id, "/") { http.NotFound(w, r); return }
	switch r.Method {
	case http.MethodGet:
		s.mu.RLock(); item, exists := s.items[id]; item = cloneItem(item); s.mu.RUnlock()
		if !exists { writeError(w, http.StatusNotFound, "not_found"); return }
		writeOK(w, map[string]any{"item": item})
	case http.MethodPut:
		input, ok := decodeItem(w, r); if !ok { return }
		s.mu.Lock()
		current, exists := s.items[id]
		if !exists { s.mu.Unlock(); writeError(w, http.StatusNotFound, "not_found"); return }
		for key, value := range input { current[key] = value }
		current[{{PRIMARY_FIELD}}] = id
		item := cloneItem(current); s.mu.Unlock()
		writeOK(w, map[string]any{"item": item})
	case http.MethodDelete:
		s.mu.Lock(); _, exists := s.items[id]; delete(s.items, id); s.mu.Unlock()
		if !exists { writeError(w, http.StatusNotFound, "not_found"); return }
		writeOK(w, nil)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func decodeItem(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	var input map[string]any
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil { writeError(w, http.StatusBadRequest, "invalid_request"); return nil, false }
	if input == nil { input = make(map[string]any) }
	return input, true
}

func cloneItem(item map[string]any) map[string]any {
	if item == nil { return nil }
	out := make(map[string]any, len(item)); for key, value := range item { out[key] = value }; return out
}

func envelope(data any) map[string]any { return map[string]any{"code": "ok", "message": "", "data": data} }
func writeOK(w http.ResponseWriter, data any) { writeJSON(w, http.StatusOK, envelope(data)) }
func writeError(w http.ResponseWriter, status int, code string) { writeJSON(w, status, map[string]any{"code": code, "message": code, "data": nil}) }
func writeJSON(w http.ResponseWriter, status int, payload any) { w.Header().Set("Content-Type", "application/json"); w.WriteHeader(status); _ = json.NewEncoder(w).Encode(payload) }
`
	return strings.NewReplacer(
		"{{API_PATH}}", fmt.Sprintf("%q", pluginAPIBasePath(spec)),
		"{{DEFAULT_ADDRESS}}", fmt.Sprintf("%q", pluginServiceAddress(spec.Plugin.ServiceBaseURL)),
		"{{PLUGIN_ID}}", fmt.Sprintf("%q", spec.Plugin.ID),
		"{{PRIMARY_FIELD}}", fmt.Sprintf("%q", primaryField(spec).Name),
	).Replace(template)
}

func pluginServiceAddress(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err == nil && parsed.Host != "" {
		return parsed.Host
	}
	return "127.0.0.1:19090"
}

func renderPluginFrontendPackage(spec domaingenerator.GeneratorSpec) string {
	dependencies := map[string]string{"element-plus": "^2.14.1", "lucide-vue-next": "^1.0.0", "pinia": "^2.1.7", "vue": "^3.4.38", "vue-router": "^4.4.3"}
	if spec.Document != nil {
		dependencies["@skoll/document-ui"] = "file:../../../../packages/skoll-document-ui"
	}
	payload := map[string]any{
		"name": "@skoll-plugins/" + spec.Plugin.ID, "private": true, "version": spec.Plugin.Version, "type": "module",
		"scripts":         map[string]string{"build": "vue-tsc --noEmit && vite build", "dev": "vite"},
		"dependencies":    dependencies,
		"devDependencies": map[string]string{"@vitejs/plugin-vue": "^5.2.4", "typescript": "^5.9.3", "vite": "^5.4.21", "vue-tsc": "^3.3.4"},
	}
	raw, _ := json.MarshalIndent(payload, "", "  ")
	return string(raw) + "\n"
}

func renderPluginFrontendTSConfig(spec domaingenerator.GeneratorSpec) string {
	if spec.Document != nil {
		return `{"compilerOptions":{"target":"ES2020","module":"ESNext","moduleResolution":"Bundler","strict":true,"skipLibCheck":true,"types":["vite/client"],"lib":["ES2020","DOM","DOM.Iterable"],"baseUrl":".","paths":{"@skoll/document-ui":["../../../../packages/skoll-document-ui/src/index.ts"]}},"include":["src/**/*.ts","src/**/*.vue"]}` + "\n"
	}
	return `{"compilerOptions":{"target":"ES2020","module":"ESNext","moduleResolution":"Bundler","strict":true,"skipLibCheck":true,"types":["vite/client"],"lib":["ES2020","DOM","DOM.Iterable"]},"include":["src/**/*.ts","src/**/*.vue"]}` + "\n"
}

func renderPluginFrontendVite(spec domaingenerator.GeneratorSpec) string {
	if spec.Document != nil {
		return `import { fileURLToPath, URL } from "node:url";
import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

export default defineConfig({
  base: "./",
  plugins: [vue()],
  resolve: { alias: { "@skoll/document-ui": fileURLToPath(new URL("../../../../packages/skoll-document-ui/src/index.ts", import.meta.url)) } },
  build: { outDir: "dist", emptyOutDir: true }
});
`
	}
	return `import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

export default defineConfig({ base: "./", plugins: [vue()], build: { outDir: "dist", emptyOutDir: true } });
`
}

func renderPluginFrontendIndex(spec domaingenerator.GeneratorSpec) string {
	return fmt.Sprintf("<!doctype html>\n<html lang=\"en\"><head><meta charset=\"UTF-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>%s</title></head><body><div id=\"app\"></div><script type=\"module\" src=\"/src/main.ts\"></script></body></html>\n", spec.Plugin.Name)
}

func renderPluginFrontendMain(spec domaingenerator.GeneratorSpec) string {
	if spec.Document != nil {
		return renderDocumentPluginFrontendMain(spec)
	}
	return fmt.Sprintf(`import { createApp } from "vue";
import { createPinia } from "pinia";
import ElementPlus from "element-plus";
import "element-plus/dist/index.css";
import Page from "./views/%s/index.vue";
import "./styles.css";

const app = createApp(Page);
app.use(createPinia());
app.use(ElementPlus);
app.directive("permission", { mounted(element, binding) { if (!globalThis.__SKOLL_PLUGIN_PERMISSIONS__?.includes("*") && !globalThis.__SKOLL_PLUGIN_PERMISSIONS__?.includes(String(binding.value))) element.remove(); } });
app.mount("#app");

declare global { var __SKOLL_PLUGIN_PERMISSIONS__: string[] | undefined; }
`, spec.Table.DomainName)
}

func renderPluginFrontendStyles() string {
	return `:root { --layout-gap: 16px; --color-primary: #176b5b; --color-text-muted: #64748b; --color-border: #d7dee7; --color-surface: #fff; --radius-md: 6px; font-family: Inter, system-ui, sans-serif; color: #17212b; background: #f5f7fa; }
* { box-sizing: border-box; }
body { margin: 0; padding: 20px; }
button, input, textarea { font: inherit; }
@media (max-width: 760px) { body { padding: 12px; } }
`
}

func renderPluginFrontendAPISupport(spec domaingenerator.GeneratorSpec) string {
	if spec.Document != nil {
		return renderDocumentPluginFrontendAPISupport(spec)
	}
	return `export type ApiResponse<T> = { code: string; message: string; data: T };
const prefix = (import.meta.env.VITE_SKOLL_API_PREFIX || "/skoll").replace(/\/$/, "");
async function request<T>(path: string, init: RequestInit): Promise<T> { const response = await fetch(prefix + path, { ...init, headers: { "Content-Type": "application/json", ...(init.headers || {}) } }); const payload = await response.json(); if (!response.ok) throw new Error(payload?.message || "request_failed"); return payload as T; }
export const apiGet = <T>(path: string): Promise<T> => request<T>(path, { method: "GET" });
export const apiPost = <T>(path: string, body?: unknown): Promise<T> => request<T>(path, { method: "POST", body: body === undefined ? undefined : JSON.stringify(body) });
export const apiPut = <T>(path: string, body?: unknown): Promise<T> => request<T>(path, { method: "PUT", body: body === undefined ? undefined : JSON.stringify(body) });
export const apiDelete = <T>(path: string): Promise<T> => request<T>(path, { method: "DELETE" });
`
}

func renderPluginFrontendCommonSupport() string {
	return `export function toErrorMessage(error: unknown): string { return error instanceof Error ? error.message : String(error); }
`
}

func renderPluginFrontendI18nSupport() string {
	return `import { computed, ref } from "vue";
const current = ref(navigator.language === "en-US" ? "en-US" : "zh-CN");
export function useI18n() { return { locale: computed(() => current.value) }; }
`
}

func renderPluginFrontendPermissionSupport() string {
	return `export function useButtonAccess() { return { can: (permission: string): boolean => globalThis.__SKOLL_PLUGIN_PERMISSIONS__?.includes("*") === true || globalThis.__SKOLL_PLUGIN_PERMISSIONS__?.includes(permission) === true }; }
declare global { var __SKOLL_PLUGIN_PERMISSIONS__: string[] | undefined; }
`
}

func renderPluginFrontendUISupport() string {
	return `import { defineComponent, h } from "vue";
export type DataTableColumn = { key: string; label: string; width?: string | number; minWidth?: string | number };
export const PageShell = defineComponent({ props: { title: String, description: String, forbidden: Boolean, forbiddenDescription: String, loading: Boolean, error: String }, setup(props, { slots }) { return () => h("section", { class: "page-shell" }, [h("header", [h("h2", props.title), h("p", props.description), slots.actions?.()]), props.forbidden ? h("p", props.forbiddenDescription) : props.loading ? h("p", "Loading") : props.error ? h("p", props.error) : slots.default?.()]); } });
export const FilterBar = defineComponent({ setup(_, { slots }) { return () => h("form", { class: "filter-bar", onSubmit: (event: Event) => event.preventDefault() }, [slots.default?.(), slots.actions?.()]); } });
export const DataTable = defineComponent({
  props: { rows: { type: Array, default: () => [] }, columns: { type: Array, default: () => [] }, loading: Boolean, error: String, emptyText: String, ariaLabel: String },
  setup(props, { slots }) {
    return () => {
      const columns = props.columns as DataTableColumn[];
      const headers = h("thead", [h("tr", columns.map(column => h("th", column.label)))]);
      const rows = (props.rows as Record<string, unknown>[]).map(row => {
        const cells = columns.map(column => h("td", String(row[column.key] ?? "-")));
        cells.push(h("td", slots.actions?.({ row })));
        return h("tr", cells);
      });
      return h("section", { class: "data-table", "aria-label": props.ariaLabel }, [props.error ? h("p", props.error) : null, h("table", [headers, h("tbody", rows)]), slots.pagination?.()]);
    };
  }
});
export const DetailDrawer = defineComponent({ props: { modelValue: Boolean, title: String, loading: Boolean }, emits: ["update:modelValue"], setup(props, { slots }) { return () => props.modelValue ? h("aside", { class: "detail-drawer" }, [h("h3", props.title), props.loading ? h("p", "Loading") : slots.default?.(), slots.footer?.()]) : null; } });
export const ConfirmAction = defineComponent({ props: { label: String, message: String, title: String, loading: Boolean }, emits: ["confirm"], setup(props, { emit }) { return () => h("button", { type: "button", disabled: props.loading, onClick: () => { if (globalThis.confirm(props.message || props.title || "Confirm")) emit("confirm"); } }, props.label); } });
`
}

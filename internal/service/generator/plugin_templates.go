package generator

import (
	"bytes"
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
	dependencies := map[string]string{
		"@skoll/business-ui": "file:../../../../packages/skoll-business-ui",
		"@skoll/plugin-sdk":  "file:../../../../packages/skoll-plugin-sdk",
		"element-plus":       "^2.14.1",
		"lucide-vue-next":    "^1.0.0",
		"pinia":              "^2.1.7",
		"vue":                "^3.5.34",
	}
	if spec.Document != nil {
		dependencies["@skoll/document-ui"] = "file:../../../../packages/skoll-document-ui"
	}
	payload := map[string]any{
		"name": "@skoll-plugins/" + spec.Plugin.ID, "private": true, "version": spec.Plugin.Version, "type": "module",
		"scripts": map[string]string{
			"build":        "vue-tsc --noEmit && vite build && npm run check:bundle",
			"check:bundle": "node ../../../../scripts/check-plugin-frontend-bundle.mjs dist",
			"dev":          "vite",
		},
		"dependencies": dependencies,
		"devDependencies": map[string]string{
			"@vitejs/plugin-vue":      "^5.2.4",
			"typescript":              "^5.9.3",
			"unplugin-vue-components": "^0.27.5",
			"vite":                    "^5.4.21",
			"vue-tsc":                 "^3.3.4",
		},
	}
	raw, _ := json.MarshalIndent(payload, "", "  ")
	return string(raw) + "\n"
}

func renderPluginFrontendTSConfig(_ domaingenerator.GeneratorSpec) string {
	return `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "strict": true,
    "skipLibCheck": true,
    "baseUrl": ".",
    "paths": {
      "@skoll/business-ui": ["../../../../packages/skoll-business-ui/src/index.ts"],
      "@skoll/business-ui/core": ["../../../../packages/skoll-business-ui/src/core.ts"],
      "@skoll/document-ui": ["../../../../packages/skoll-document-ui/src/index.ts"],
      "@skoll/plugin-sdk": ["../../../../packages/skoll-plugin-sdk/src/index.ts"]
    },
    "types": ["vite/client"],
    "lib": ["ES2022", "DOM", "DOM.Iterable"]
  },
  "include": ["src/**/*.ts", "src/**/*.vue", "vite.config.ts"]
}
`
}

func renderPluginFrontendVite(_ domaingenerator.GeneratorSpec) string {
	return `import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import Components from "unplugin-vue-components/vite";
import { ElementPlusResolver } from "unplugin-vue-components/resolvers";

const source = (path: string) => decodeURIComponent(new URL(path, import.meta.url).pathname)
  .replace(/^\/([A-Za-z]:\/)/, "$1");

export default defineConfig({
  base: "./",
  resolve: {
    alias: {
      "@skoll/business-ui/core": source("../../../../packages/skoll-business-ui/src/core.ts"),
      "@skoll/business-ui": source("../../../../packages/skoll-business-ui/src/index.ts"),
      "@skoll/document-ui": source("../../../../packages/skoll-document-ui/src/index.ts"),
      "@skoll/plugin-sdk": source("../../../../packages/skoll-plugin-sdk/src/index.ts")
    },
    dedupe: ["vue", "element-plus", "@vueuse/core"]
  },
  plugins: [vue(), Components({ resolvers: [ElementPlusResolver({ importStyle: false })], dts: false })],
  build: {
    outDir: "dist",
    emptyOutDir: true,
    manifest: true,
    target: "es2022",
    cssCodeSplit: true,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes("node_modules/lucide-vue-next")) return "icons";
          if (id.includes("node_modules/vue") || id.includes("node_modules/@vue")) return "vue";
          return undefined;
        }
      }
    }
  }
});
`
}

func renderPluginFrontendIndex(spec domaingenerator.GeneratorSpec) string {
	return fmt.Sprintf("<!doctype html>\n<html lang=\"zh-CN\"><head><meta charset=\"UTF-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>%s</title></head><body><div id=\"app\"></div><script type=\"module\" src=\"/src/main.ts\"></script></body></html>\n", spec.Plugin.Name)
}

func renderPluginFrontendMain(spec domaingenerator.GeneratorSpec) string {
	component := spec.Table.DomainName
	return fmt.Sprintf(`import { createApp } from "vue";
import { createPinia } from "pinia";
import Page from "./views/%s/index.vue";
import "./skoll-host";
import "./styles.css";
import "element-plus/theme-chalk/base.css";
import "element-plus/theme-chalk/el-alert.css";
import "element-plus/theme-chalk/el-button.css";
import "element-plus/theme-chalk/el-date-picker.css";
import "element-plus/theme-chalk/el-descriptions.css";
import "element-plus/theme-chalk/el-descriptions-item.css";
import "element-plus/theme-chalk/el-drawer.css";
import "element-plus/theme-chalk/el-form.css";
import "element-plus/theme-chalk/el-form-item.css";
import "element-plus/theme-chalk/el-input.css";
import "element-plus/theme-chalk/el-input-number.css";
import "element-plus/theme-chalk/el-message.css";
import "element-plus/theme-chalk/el-empty.css";
import "element-plus/theme-chalk/el-option.css";
import "element-plus/theme-chalk/el-popconfirm.css";
import "element-plus/theme-chalk/el-popover.css";
import "element-plus/theme-chalk/el-select.css";
import "element-plus/theme-chalk/el-skeleton.css";
import "element-plus/theme-chalk/el-skeleton-item.css";
import "element-plus/theme-chalk/el-switch.css";
import "element-plus/theme-chalk/el-tab-pane.css";
import "element-plus/theme-chalk/el-table.css";
import "element-plus/theme-chalk/el-table-column.css";
import "element-plus/theme-chalk/el-tag.css";
import "element-plus/theme-chalk/el-tabs.css";
import "element-plus/theme-chalk/el-timeline.css";
import "element-plus/theme-chalk/el-timeline-item.css";
import "element-plus/theme-chalk/el-tooltip.css";

const app = createApp(Page);
app.use(createPinia());
app.mount("#app");
`, component)
}

func renderPluginFrontendStyles() string {
	return `:root { font-family: Inter, "Segoe UI", system-ui, sans-serif; color: var(--color-text); background: var(--color-bg); }
* { box-sizing: border-box; }
html, body, #app { min-height: 100%; }
body { margin: 0; padding: var(--content-padding); background: var(--color-bg); }
button, input, textarea { font: inherit; }
@media (max-width: 760px) { body { padding: 12px; } }
`
}

func renderPluginFrontendAPISupport(spec domaingenerator.GeneratorSpec) string {
	return fmt.Sprintf(`import { getPluginHost, type PluginHostRequestOptions } from "@skoll/plugin-sdk";

export type ApiResponse<T> = { code: string; message: string; data: T };
const host = () => getPluginHost({ pluginId: %q, requiredCapabilities: ["request"] });
const request = <T>(path: string, options: PluginHostRequestOptions): Promise<T> => host().request<T>(path, options);

export const apiGet = <T>(path: string): Promise<T> => request<T>(path, { method: "GET" });
export const apiPost = <T>(path: string, body?: unknown): Promise<T> => request<T>(path, { method: "POST", body });
export const apiPut = <T>(path: string, body?: unknown): Promise<T> => request<T>(path, { method: "PUT", body });
export const apiDelete = <T>(path: string): Promise<T> => request<T>(path, { method: "DELETE" });
`, spec.Plugin.ID)
}

func renderPluginFrontendCommonSupport() string {
	return `export function toErrorMessage(error: unknown): string { return error instanceof Error ? error.message : String(error); }
`
}

func renderBusinessPluginFrontendView(spec domaingenerator.GeneratorSpec) string {
	module := spec.Module.Package
	typeName := spec.Table.DomainName
	storeName := exportedName(module)
	prefix := "generated." + module
	var b bytes.Buffer
	fmt.Fprintf(&b, "<template>\n")
	fmt.Fprintf(&b, "\t<BusinessWorkspace\n\t\tclass=\"plugin-generated-page\"\n\t\t:title=\"t('%s.title')\"\n\t\t:description=\"t('%s.description')\"\n\t\t:locale=\"locale\"\n\t\t:state=\"workspaceState\"\n\t\t:state-description=\"workspaceStateDescription\"\n\t>\n", prefix, prefix)
	fmt.Fprintf(&b, "\t\t<template #actions>\n\t\t\t<BusinessCommandBar :commands=\"commands\" :locale=\"locale\" @command=\"runCommand\" />\n\t\t</template>\n")
	fmt.Fprintf(&b, "\t\t<template #filters>\n\t\t\t<BusinessFilterBar v-model=\"filters\" :fields=\"filterFields\" :locale=\"locale\" :busy=\"store.listStatus === 'loading'\" @search=\"loadPage(0)\" @reset=\"loadPage(0)\" />\n\t\t</template>\n")
	fmt.Fprintf(&b, "\t\t<template #stateActions>\n\t\t\t<el-button v-if=\"canRead && lifecycle.state === 'enabled'\" :icon=\"RefreshCw\" @click=\"loadPage(0)\">{{ t('%s.retry') }}</el-button>\n\t\t</template>\n", prefix)
	fmt.Fprintf(&b, "\t\t<BusinessList\n\t\t\t:items=\"store.items\"\n\t\t\t:columns=\"columns\"\n\t\t\t:state=\"listState\"\n\t\t\t:state-description=\"tableError\"\n\t\t\t:can-previous=\"store.offset > 0\"\n\t\t\t:can-next=\"store.hasMore\"\n\t\t\t:action-width=\"132\"\n\t\t\t:locale=\"locale\"\n\t\t\t@open=\"openDetail\"\n\t\t\t@previous=\"loadPage(Math.max(0, store.offset - store.limit))\"\n\t\t\t@next=\"loadPage(store.offset + store.limit)\"\n\t\t>\n")
	fmt.Fprintf(&b, "\t\t\t<template #actions=\"{ row }\">\n\t\t\t\t<div class=\"plugin-generated-row-actions\">\n")
	fmt.Fprintf(&b, "\t\t\t\t\t<el-tooltip :content=\"t('%s.view')\"><el-button link :icon=\"Eye\" :aria-label=\"t('%s.view')\" @click=\"openDetail(row)\" /></el-tooltip>\n", prefix, prefix)
	fmt.Fprintf(&b, "\t\t\t\t\t<el-tooltip v-if=\"canUpdate\" :content=\"t('%s.edit')\"><el-button link type=\"primary\" :icon=\"Pencil\" :aria-label=\"t('%s.edit')\" @click=\"openEdit(row)\" /></el-tooltip>\n", prefix, prefix)
	fmt.Fprintf(&b, "\t\t\t\t\t<el-popconfirm v-if=\"canDelete\" :title=\"t('%s.deleteConfirm')\" :confirm-button-text=\"t('%s.delete')\" :cancel-button-text=\"t('%s.cancel')\" @confirm=\"remove(String(row.id))\">\n", prefix, prefix, prefix)
	fmt.Fprintf(&b, "\t\t\t\t\t\t<template #reference><el-button link type=\"danger\" :icon=\"Trash2\" :loading=\"deletingId === String(row.id)\" :aria-label=\"t('%s.delete')\" /></template>\n", prefix)
	fmt.Fprintf(&b, "\t\t\t\t\t</el-popconfirm>\n\t\t\t\t</div>\n\t\t\t</template>\n\t\t</BusinessList>\n\t</BusinessWorkspace>\n\n")
	fmt.Fprintf(&b, "\t<el-drawer v-model=\"detailOpen\" :title=\"t('%s.detail')\" size=\"min(560px, 100%%)\">\n", prefix)
	fmt.Fprintf(&b, "\t\t<el-alert v-if=\"store.detailError\" type=\"error\" :title=\"store.detailError\" show-icon :closable=\"false\" />\n\t\t<el-descriptions v-else-if=\"store.selected\" :column=\"1\" border>\n")
	for _, fieldName := range spec.Page.List.Columns {
		if field, ok := spec.FieldByName(fieldName); ok {
			fmt.Fprintf(&b, "\t\t\t<el-descriptions-item :label=\"t('%s.field.%s')\">{{ displayValue(store.selected.%s) }}</el-descriptions-item>\n", prefix, field.Name, field.Name)
		}
	}
	fmt.Fprintf(&b, "\t\t</el-descriptions>\n\t</el-drawer>\n\n")
	fmt.Fprintf(&b, "\t<el-drawer v-model=\"formOpen\" :title=\"formTitle\" size=\"min(560px, 100%%)\">\n\t\t<el-alert v-if=\"store.mutationError\" type=\"error\" :title=\"store.mutationError\" show-icon :closable=\"false\" />\n\t\t<el-form ref=\"formRef\" :model=\"form\" :rules=\"rules\" label-position=\"top\" @submit.prevent>\n")
	for _, fieldName := range spec.Page.Form.Fields {
		if field, ok := spec.FieldByName(fieldName); ok {
			renderGeneratedFormControl(&b, field, prefix)
		}
	}
	fmt.Fprintf(&b, "\t\t</el-form>\n\t\t<template #footer>\n\t\t\t<el-button @click=\"formOpen = false\">{{ t('%s.cancel') }}</el-button>\n\t\t\t<el-button type=\"primary\" :icon=\"Save\" :loading=\"store.mutationStatus === 'loading'\" @click=\"save\">{{ t('%s.save') }}</el-button>\n\t\t</template>\n\t</el-drawer>\n", prefix, prefix)
	fmt.Fprintf(&b, "</template>\n\n<script setup lang=\"ts\">\n")
	fmt.Fprintf(&b, "import { BusinessCommandBar, BusinessFilterBar, BusinessList, BusinessWorkspace, type BusinessCommand, type BusinessFilterField, type BusinessFilterModel, type BusinessListColumn, type BusinessRecord } from \"@skoll/business-ui/core\";\n")
	fmt.Fprintf(&b, "import { ElMessage, type FormInstance, type FormRules } from \"element-plus\";\n")
	fmt.Fprintf(&b, "import { Eye, Pencil, Plus, RefreshCw, Save, Trash2 } from \"lucide-vue-next\";\n")
	fmt.Fprintf(&b, "import { computed, onMounted, reactive, ref, watch } from \"vue\";\n\n")
	fmt.Fprintf(&b, "import { translate%s } from \"../../i18n/generated_%s\";\n", typeName, module)
	fmt.Fprintf(&b, "import { useSkollHost } from \"../../skoll-host\";\n")
	fmt.Fprintf(&b, "import { use%sStore } from \"../../stores/%s\";\n", storeName, module)
	fmt.Fprintf(&b, "import type { %sInput } from \"../../api/%s\";\n\n", typeName, module)
	fmt.Fprintf(&b, "const store = use%sStore();\n", storeName)
	fmt.Fprintf(&b, "const { locale, lifecycle, can } = useSkollHost();\n")
	fmt.Fprintf(&b, "const readPermission = %q;\nconst createPermission = %q;\nconst updatePermission = %q;\nconst deletePermission = %q;\n", spec.Permissions.ReadKey, spec.Permissions.CreateKey, spec.Permissions.UpdateKey, spec.Permissions.DeleteKey)
	fmt.Fprintf(&b, "const canRead = computed(() => can(readPermission));\nconst canCreate = computed(() => can(createPermission));\nconst canUpdate = computed(() => can(updatePermission));\nconst canDelete = computed(() => can(deletePermission));\n")
	fmt.Fprintf(&b, "const filters = ref<BusinessFilterModel>({ keyword: \"\" });\nconst detailOpen = ref(false);\nconst formOpen = ref(false);\nconst editingId = ref<string | null>(null);\nconst deletingId = ref(\"\");\nconst formRef = ref<FormInstance>();\nconst form = reactive<%sInput>({});\n", typeName)
	fmt.Fprintf(&b, "const t = (key: string): string => translate%s(locale.value, key);\n", typeName)
	fmt.Fprintf(&b, "const workspaceState = computed<\"ready\" | \"loading\" | \"error\" | \"forbidden\" | \"conflict\">(() => {\n\tif (lifecycle.value.state === \"disabled\") return \"forbidden\";\n\tif (lifecycle.value.state === \"degraded\") return \"conflict\";\n\tif (!canRead.value) return \"forbidden\";\n\tif (store.listStatus === \"loading\" && store.items.length === 0) return \"loading\";\n\tif (store.listStatus === \"error\" && store.items.length === 0) return \"error\";\n\treturn \"ready\";\n});\n")
	fmt.Fprintf(&b, "const workspaceStateDescription = computed(() => workspaceState.value === \"error\" ? store.listError : \"\");\n")
	fmt.Fprintf(&b, "const listState = computed<\"ready\" | \"loading\" | \"error\">(() => store.listStatus === \"loading\" ? \"loading\" : store.listStatus === \"error\" ? \"error\" : \"ready\");\n")
	fmt.Fprintf(&b, "const tableError = computed(() => store.items.length > 0 ? store.listError : \"\");\n")
	fmt.Fprintf(&b, "const commands = computed<BusinessCommand[]>(() => {\n\tconst items: BusinessCommand[] = [{ id: \"refresh\", label: t(%q), icon: RefreshCw, loading: store.listStatus === \"loading\" }];\n\tif (canCreate.value) items.push({ id: \"create\", label: t(%q), icon: Plus, tone: \"primary\" });\n\treturn items;\n});\n", prefix+".refresh", prefix+".create")
	fmt.Fprintf(&b, "const filterFields = computed<BusinessFilterField[]>(() => [{ key: \"keyword\", label: t(%q), type: \"search\", placeholder: t(%q), clearable: true }]);\n", prefix+".search", prefix+".searchPlaceholder")
	fmt.Fprintf(&b, "const columns = computed<BusinessListColumn[]>(() => [\n")
	for _, fieldName := range spec.Page.List.Columns {
		if field, ok := spec.FieldByName(fieldName); ok {
			fmt.Fprintf(&b, "\t{ key: %q, label: t(%q), minWidth: 140 },\n", field.Name, prefix+".field."+field.Name)
		}
	}
	fmt.Fprintf(&b, "]);\n")
	fmt.Fprintf(&b, "const formTitle = computed(() => t(editingId.value ? %q : %q));\n", prefix+".editTitle", prefix+".createTitle")
	fmt.Fprintf(&b, "const rules: FormRules = {\n")
	for _, fieldName := range spec.Page.Form.Fields {
		if field, ok := spec.FieldByName(fieldName); ok && (field.Required || len(field.Validation) > 0) {
			fmt.Fprintf(&b, "\t%s: [", field.Name)
			if field.Required {
				fmt.Fprintf(&b, "{ required: true, message: t(%q), trigger: \"blur\" }", prefix+".validation."+field.Name)
			}
			for _, rule := range field.Validation {
				if rule.Type == domaingenerator.ValidationPattern && strings.TrimSpace(rule.Value) != "" {
					if field.Required {
						fmt.Fprintf(&b, ", ")
					}
					fmt.Fprintf(&b, "{ pattern: new RegExp(%q), message: t(%q), trigger: \"blur\" }", rule.Value, prefix+".validation."+field.Name)
				}
			}
			fmt.Fprintf(&b, "],\n")
		}
	}
	fmt.Fprintf(&b, "};\n\n")
	fmt.Fprintf(&b, "function runCommand(command: BusinessCommand): void {\n\tif (command.id === \"create\") openCreate();\n\tif (command.id === \"refresh\") void loadPage(store.offset);\n}\n\n")
	fmt.Fprintf(&b, "async function loadPage(offset: number): Promise<void> {\n\tif (!canRead.value || lifecycle.value.state !== \"enabled\") return;\n\ttry {\n\t\tawait store.load({ keyword: String(filters.value.keyword || \"\").trim() || undefined, offset, limit: store.limit });\n\t} catch {\n\t\t// Shared business states render the store error.\n\t}\n}\n\n")
	fmt.Fprintf(&b, "async function openDetail(row: BusinessRecord): Promise<void> {\n\tdetailOpen.value = true;\n\ttry { await store.loadOne(String(row.id)); } catch { /* The drawer renders the store error. */ }\n}\n\n")
	fmt.Fprintf(&b, "function openCreate(): void {\n\teditingId.value = null;\n\tresetForm();\n\tstore.clearMutationState();\n\tformOpen.value = true;\n}\n\n")
	fmt.Fprintf(&b, "function openEdit(row: BusinessRecord): void {\n\teditingId.value = String(row.id);\n\tresetForm();\n\tObject.assign(form, row);\n\tstore.clearMutationState();\n\tformOpen.value = true;\n}\n\n")
	fmt.Fprintf(&b, "async function save(): Promise<void> {\n\tif (!await formRef.value?.validate().catch(() => false)) return;\n\ttry {\n\t\tif (editingId.value) await store.update(editingId.value, { ...form }); else await store.create({ ...form });\n\t\tformOpen.value = false;\n\t\tElMessage.success(t(%q));\n\t} catch { /* The form keeps the backend error visible. */ }\n}\n\n", prefix+".saved")
	fmt.Fprintf(&b, "async function remove(id: string): Promise<void> {\n\tdeletingId.value = id;\n\ttry { await store.remove(id); ElMessage.success(t(%q)); } catch { ElMessage.error(store.mutationError); } finally { deletingId.value = \"\"; }\n}\n\n", prefix+".deleted")
	fmt.Fprintf(&b, "function resetForm(): void {\n\tfor (const key of Object.keys(form)) delete form[key];\n")
	for _, fieldName := range spec.Page.Form.Fields {
		if field, ok := spec.FieldByName(fieldName); ok {
			fmt.Fprintf(&b, "\tform.%s = %s;\n", field.Name, tsFieldDefault(field))
		}
	}
	fmt.Fprintf(&b, "}\n\nfunction displayValue(value: unknown): string {\n\tif (value === null || value === undefined || value === \"\") return \"-\";\n\tif (typeof value === \"object\") return JSON.stringify(value);\n\treturn String(value);\n}\n\n")
	fmt.Fprintf(&b, "watch(() => lifecycle.value.state, (state) => { if (state === \"enabled\" && canRead.value && store.listStatus === \"idle\") void loadPage(0); });\n")
	fmt.Fprintf(&b, "onMounted(() => { if (canRead.value && lifecycle.value.state === \"enabled\") void loadPage(0); });\n")
	fmt.Fprintf(&b, "</script>\n\n<style scoped>\n.plugin-generated-row-actions { display: flex; align-items: center; gap: 4px; }\n:deep(.el-drawer__body) { padding: 16px; }\n:deep(.el-input-number), :deep(.el-date-editor) { width: 100%%; }\n@media (max-width: 560px) { .plugin-generated-row-actions { flex-wrap: wrap; } }\n</style>\n")
	return b.String()
}

func renderPluginFrontendHostSupport(spec domaingenerator.GeneratorSpec) string {
	return fmt.Sprintf(`import type { BusinessLocale } from "@skoll/business-ui/core";
import {
  getPluginHost,
  type PluginHostLifecycle,
  type PluginHostTheme
} from "@skoll/plugin-sdk";
import { readonly, ref } from "vue";

const host = getPluginHost({
  pluginId: %q,
  requiredCapabilities: ["request", "permissions", "locale", "theme", "lifecycle"]
});
const locale = ref<BusinessLocale>(host.locale === "en-US" ? "en-US" : "zh-CN");
const theme = ref<PluginHostTheme>(host.theme);
const lifecycle = ref<PluginHostLifecycle>(host.lifecycle);

function applyTheme(next: PluginHostTheme): void {
  const root = document.documentElement;
  root.dataset.theme = next.colorScheme;
  root.dataset.density = next.density;
  root.style.colorScheme = next.colorScheme;
  Object.entries(next.tokens).forEach(([name, value]) => root.style.setProperty(name, value));
}

document.documentElement.lang = locale.value;
applyTheme(theme.value);
window.addEventListener("skoll:locale", (event) => {
  const next = (event as CustomEvent<{ locale?: string }>).detail?.locale;
  locale.value = next === "en-US" ? "en-US" : "zh-CN";
  document.documentElement.lang = locale.value;
});
window.addEventListener("skoll:theme", (event) => {
  const next = (event as CustomEvent<PluginHostTheme>).detail;
  if (next) {
    theme.value = next;
    applyTheme(next);
  }
});
window.addEventListener("skoll:lifecycle", (event) => {
  const next = (event as CustomEvent<PluginHostLifecycle>).detail;
  if (next) lifecycle.value = next;
});

export function useSkollHost() {
  return {
    host,
    locale: readonly(locale),
    theme: readonly(theme),
    lifecycle: readonly(lifecycle),
    can: (permission: string): boolean => host.permissions.has(permission)
  };
}
`, spec.Plugin.ID)
}

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
	return fmt.Sprintf("module example.com/skoll-plugins/%s\n\ngo 1.24\n\nrequire github.com/tinboxw/skoll v0.0.0\n\nreplace github.com/tinboxw/skoll => ../../..\n", spec.Plugin.ID)
}

func renderPluginBackendServer(spec domaingenerator.GeneratorSpec) string {
	template := `package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/tinboxw/skoll/pkg/pluginclient"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const defaultAddress = {{DEFAULT_ADDRESS}}

func main() {
	client, err := pluginclient.FromEnvironment()
	if err != nil { log.Fatal(err) }
	host, err := client.HostServices()
	if err != nil { log.Fatal(err) }
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ready", "plugin": generatedPluginID})
	})
	mux.HandleFunc(generatedAPIBasePath, func(w http.ResponseWriter, r *http.Request) { collection(w, r, host) })
	mux.HandleFunc(generatedAPIBasePath+"/", func(w http.ResponseWriter, r *http.Request) { item(w, r, host) })
	address := strings.TrimSpace(os.Getenv(pluginclient.EnvironmentPluginAddress))
	if address == "" { address = defaultAddress }
	log.Printf("%s listening on %s", generatedPluginID, address)
	log.Fatal(http.ListenAndServe(address, mux))
}

func collection(w http.ResponseWriter, r *http.Request, host pluginsdk.HostServices) {
	switch r.Method {
	case http.MethodGet:
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 { limit = 20 }
		if limit > pluginsdk.MaxDataQueryLimit { limit = pluginsdk.MaxDataQueryLimit }
		query := pluginsdk.DataQuery{
			Table: generatedLogicalTable, Fields: generatedFields,
			Scope: pluginsdk.DataScopeIntent{Permission: generatedPermissions["read"]},
			Sort: []pluginsdk.DataSort{{Field: generatedPrimaryField, Direction: pluginsdk.DataSortAscending}},
			Page: pluginsdk.DataPageRequest{Cursor: strings.TrimSpace(r.URL.Query().Get("cursor")), Limit: limit},
		}
		if keyword := strings.TrimSpace(r.URL.Query().Get("keyword")); keyword != "" && {{SEARCH_FIELD}} != "" {
			value := pluginsdk.DataValue{Type: pluginsdk.DataValueString, Value: keyword}
			query.Filter = &pluginsdk.DataFilter{Field: {{SEARCH_FIELD}}, Operator: pluginsdk.DataOperatorContains, Value: &value}
		}
		page, err := host.DataStore.Query(requestContext(r), query)
		if err != nil { writeError(w, err); return }
		items := make([]map[string]any, 0, len(page.Records))
		for _, record := range page.Records { items = append(items, recordJSON(record)) }
		writeOK(w, map[string]any{"items": items, "nextCursor": page.NextCursor, "hasMore": page.HasMore, "limit": limit})
	case http.MethodPost:
		input, err := decodeItem(r)
		if err != nil { writeError(w, err); return }
		key, values, err := mutationValues(input)
		if err != nil { writeError(w, err); return }
		var result pluginsdk.DataMutationResult
		err = host.Transactions.Within(requestContext(r), func(tx pluginsdk.Transaction) error {
			var mutationErr error
			result, mutationErr = host.DataStore.Mutate(tx.Context(), pluginsdk.DataMutation{
				Table: generatedLogicalTable, Operation: pluginsdk.DataMutationInsert,
				Scope: pluginsdk.DataScopeIntent{Permission: generatedPermissions["create"]},
				Key: key, Values: values, Returning: generatedFields,
				IdempotencyKey: key[generatedPrimaryField].Value + ".create",
			})
			return mutationErr
		})
		if err != nil { writeError(w, err); return }
		writeJSON(w, http.StatusCreated, envelope(map[string]any{"item": mutationRecord(result)}))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func item(w http.ResponseWriter, r *http.Request, host pluginsdk.HostServices) {
	id := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, generatedAPIBasePath+"/"))
	if id == "" || strings.Contains(id, "/") { http.NotFound(w, r); return }
	switch r.Method {
	case http.MethodGet:
		value := pluginsdk.DataValue{Type: generatedFieldTypes[generatedPrimaryField], Value: id}
		page, err := host.DataStore.Query(requestContext(r), pluginsdk.DataQuery{
			Table: generatedLogicalTable, Fields: generatedFields,
			Scope: pluginsdk.DataScopeIntent{Permission: generatedPermissions["read"]},
			Filter: &pluginsdk.DataFilter{Field: generatedPrimaryField, Operator: pluginsdk.DataOperatorEqual, Value: &value},
			Sort: []pluginsdk.DataSort{{Field: generatedPrimaryField, Direction: pluginsdk.DataSortAscending}},
			Page: pluginsdk.DataPageRequest{Limit: 1},
		})
		if err != nil { writeError(w, err); return }
		if len(page.Records) == 0 { writeError(w, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorNotFound, generatedPrimaryField, "record not found", false)); return }
		writeOK(w, map[string]any{"item": recordJSON(page.Records[0])})
	case http.MethodPut:
		input, err := decodeItem(r)
		if err != nil { writeError(w, err); return }
		input[generatedPrimaryField] = id
		key, values, err := mutationValues(input)
		if err != nil { writeError(w, err); return }
		result, err := host.DataStore.Mutate(requestContext(r), pluginsdk.DataMutation{
			Table: generatedLogicalTable, Operation: pluginsdk.DataMutationUpdate,
			Scope: pluginsdk.DataScopeIntent{Permission: generatedPermissions["update"]},
			Key: key, Values: values, Returning: generatedFields,
			IdempotencyKey: id + ".update." + requestIdempotencySuffix(r),
		})
		if err != nil { writeError(w, err); return }
		writeOK(w, map[string]any{"item": mutationRecord(result)})
	case http.MethodDelete:
		key := map[string]pluginsdk.DataValue{generatedPrimaryField: {Type: generatedFieldTypes[generatedPrimaryField], Value: id}}
		_, err := host.DataStore.Mutate(requestContext(r), pluginsdk.DataMutation{
			Table: generatedLogicalTable, Operation: pluginsdk.DataMutationDelete,
			Scope: pluginsdk.DataScopeIntent{Permission: generatedPermissions["delete"]},
			Key: key, IdempotencyKey: id + ".delete." + requestIdempotencySuffix(r),
		})
		if err != nil { writeError(w, err); return }
		writeOK(w, nil)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func decodeItem(r *http.Request) (map[string]any, error) {
	var input map[string]any
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&input); err != nil { return nil, fmt.Errorf("decode request: %w", err) }
	if input == nil { return nil, errors.New("request body is required") }
	for field := range input {
		if _, declared := generatedFieldTypes[field]; !declared { return nil, fmt.Errorf("field %s is not declared", field) }
	}
	return input, nil
}

func mutationValues(input map[string]any) (map[string]pluginsdk.DataValue, map[string]pluginsdk.DataValue, error) {
	key := make(map[string]pluginsdk.DataValue, 1)
	values := make(map[string]pluginsdk.DataValue, len(input))
	for field, raw := range input {
		value, err := dataValue(field, raw)
		if err != nil { return nil, nil, err }
		if field == generatedPrimaryField { key[field] = value } else { values[field] = value }
	}
	if key[generatedPrimaryField].Value == "" { return nil, nil, fmt.Errorf("%s is required", generatedPrimaryField) }
	if len(values) == 0 { return nil, nil, errors.New("at least one mutable field is required") }
	return key, values, nil
}

func dataValue(field string, raw any) (pluginsdk.DataValue, error) {
	valueType, exists := generatedFieldTypes[field]
	if !exists { return pluginsdk.DataValue{}, fmt.Errorf("field %s is not declared", field) }
	var value string
	switch valueType {
	case pluginsdk.DataValueBoolean:
		boolean, ok := raw.(bool); if !ok { return pluginsdk.DataValue{}, fmt.Errorf("%s must be boolean", field) }
		value = strconv.FormatBool(boolean)
	case pluginsdk.DataValueJSON:
		encoded, err := json.Marshal(raw); if err != nil { return pluginsdk.DataValue{}, fmt.Errorf("%s: %w", field, err) }; value = string(encoded)
	default:
		value = strings.TrimSpace(fmt.Sprint(raw))
	}
	out := pluginsdk.DataValue{Type: valueType, Value: value}
	if err := out.Validate(); err != nil { return pluginsdk.DataValue{}, fmt.Errorf("%s: %w", field, err) }
	return out, nil
}

func recordJSON(record pluginsdk.DataRecord) map[string]any {
	out := make(map[string]any, len(record.Values)+1)
	for field, value := range record.Values { out[field] = jsonValue(value) }
	out["version"] = record.Version
	return out
}

func mutationRecord(result pluginsdk.DataMutationResult) map[string]any {
	if result.Record == nil { return map[string]any{} }
	return recordJSON(*result.Record)
}

func jsonValue(value pluginsdk.DataValue) any {
	switch value.Type {
	case pluginsdk.DataValueInteger:
		parsed, _ := strconv.ParseInt(value.Value, 10, 64); return parsed
	case pluginsdk.DataValueDecimal:
		return json.Number(value.Value)
	case pluginsdk.DataValueBoolean:
		parsed, _ := strconv.ParseBool(value.Value); return parsed
	case pluginsdk.DataValueJSON:
		var parsed any
		if json.Unmarshal([]byte(value.Value), &parsed) == nil { return parsed }
	}
	return value.Value
}

func requestContext(r *http.Request) context.Context {
	if r == nil { return context.Background() }
	return pluginclient.WithUserToken(r.Context(), strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")))
}

func requestIdempotencySuffix(r *http.Request) string {
	value := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if value == "" { value = "request" }
	return value
}

func envelope(data any) map[string]any { return map[string]any{"code": "ok", "message": "", "data": data} }
func writeOK(w http.ResponseWriter, data any) { writeJSON(w, http.StatusOK, envelope(data)) }
func writeError(w http.ResponseWriter, err error) {
	status, code := http.StatusBadRequest, "invalid_request"
	var datastoreErr *pluginsdk.DataStoreError
	if errors.As(err, &datastoreErr) {
		code = string(datastoreErr.Code)
		switch datastoreErr.Code {
		case pluginsdk.DataStoreErrorForbidden: status = http.StatusForbidden
		case pluginsdk.DataStoreErrorNotFound: status = http.StatusNotFound
		case pluginsdk.DataStoreErrorConflict: status = http.StatusConflict
		case pluginsdk.DataStoreErrorLimitExceeded: status = http.StatusRequestEntityTooLarge
		case pluginsdk.DataStoreErrorUnsupported: status = http.StatusUnprocessableEntity
		case pluginsdk.DataStoreErrorUnavailable: status = http.StatusServiceUnavailable
		}
	}
	writeJSON(w, status, map[string]any{"code": code, "message": err.Error(), "data": nil})
}
func writeJSON(w http.ResponseWriter, status int, payload any) { w.Header().Set("Content-Type", "application/json"); w.WriteHeader(status); _ = json.NewEncoder(w).Encode(payload) }
`
	return strings.NewReplacer(
		"{{DEFAULT_ADDRESS}}", fmt.Sprintf("%q", pluginServiceAddress(spec.Plugin.ServiceBaseURL)),
		"{{SEARCH_FIELD}}", fmt.Sprintf("%q", pluginSearchField(spec)),
	).Replace(template)
}

func pluginServiceAddress(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err == nil && parsed.Host != "" {
		return parsed.Host
	}
	return "127.0.0.1:19090"
}

func pluginSearchField(spec domaingenerator.GeneratorSpec) string {
	for _, name := range spec.Page.List.Filters {
		field, ok := spec.FieldByName(name)
		if ok && (field.Type == domaingenerator.FieldTypeString || field.Type == domaingenerator.FieldTypeText) {
			return field.Name
		}
	}
	return ""
}

func renderPluginFrontendStore(spec domaingenerator.GeneratorSpec) string {
	module := spec.Module.Package
	typeName := spec.Table.DomainName
	storeName := exportedName(module)
	return fmt.Sprintf(`import { defineStore } from "pinia";
import {
  create%s,
  delete%s,
  get%s,
  list%s,
  update%s,
  type %s,
  type %sInput,
  type %sListQuery
} from "../api/%s";
import { toErrorMessage } from "../utils/common";

type LoadStatus = "idle" | "loading" | "success" | "error";

export const use%sStore = defineStore("%s", {
  state: () => ({
    items: [] as %s[],
    selected: null as %s | null,
    listStatus: "idle" as LoadStatus,
    detailStatus: "idle" as LoadStatus,
    mutationStatus: "idle" as LoadStatus,
    listError: "",
    detailError: "",
    mutationError: "",
    keyword: "",
    cursor: "",
    cursorHistory: [] as string[],
    nextCursor: "",
    limit: 20,
    hasMore: false
  }),
  getters: {
    isLoading: (state): boolean => state.listStatus === "loading" || state.detailStatus === "loading" || state.mutationStatus === "loading",
    canPrevious: (state): boolean => state.cursorHistory.length > 0
  },
  actions: {
    async load(query: %sListQuery = {}): Promise<void> {
      this.listStatus = "loading";
      this.listError = "";
      try {
        const page = await list%s({ keyword: query.keyword, cursor: query.cursor, limit: query.limit ?? this.limit });
        this.items = page.items;
        this.keyword = query.keyword ?? "";
        this.cursor = query.cursor ?? "";
        this.nextCursor = page.nextCursor;
        this.limit = page.limit;
        this.hasMore = page.hasMore;
        this.listStatus = "success";
      } catch (error) {
        this.listStatus = "error";
        this.listError = toErrorMessage(error);
        throw error;
      }
    },
    async search(keyword: string): Promise<void> {
      this.cursorHistory = [];
      await this.load({ keyword, cursor: "", limit: this.limit });
    },
    async refresh(): Promise<void> {
      await this.load({ keyword: this.keyword, cursor: this.cursor, limit: this.limit });
    },
    async next(): Promise<void> {
      if (!this.hasMore || !this.nextCursor) return;
      this.cursorHistory.push(this.cursor);
      await this.load({ keyword: this.keyword, cursor: this.nextCursor, limit: this.limit });
    },
    async previous(): Promise<void> {
      const cursor = this.cursorHistory.pop();
      if (cursor === undefined) return;
      await this.load({ keyword: this.keyword, cursor, limit: this.limit });
    },
    async loadOne(id: string): Promise<%s> {
      this.detailStatus = "loading";
      this.detailError = "";
      try {
        const item = await get%s(id);
        this.selected = item;
        this.detailStatus = "success";
        return item;
      } catch (error) {
        this.detailStatus = "error";
        this.detailError = toErrorMessage(error);
        throw error;
      }
    },
    async create(input: %sInput): Promise<%s> {
      return this.runMutation(() => create%s(input));
    },
    async update(id: string, input: %sInput): Promise<%s> {
      return this.runMutation(() => update%s(id, input));
    },
    async remove(id: string): Promise<void> {
      this.mutationStatus = "loading";
      this.mutationError = "";
      try {
        await delete%s(id);
        this.items = this.items.filter((item) => String(item.%s) !== id);
        if (this.selected && String(this.selected.%s) === id) this.selected = null;
        this.mutationStatus = "success";
      } catch (error) {
        this.mutationStatus = "error";
        this.mutationError = toErrorMessage(error);
        throw error;
      }
    },
    async runMutation(operation: () => Promise<%s>): Promise<%s> {
      this.mutationStatus = "loading";
      this.mutationError = "";
      try {
        const item = await operation();
        const index = this.items.findIndex((current) => String(current.%s) === String(item.%s));
        if (index >= 0) this.items.splice(index, 1, item); else this.items.unshift(item);
        this.selected = item;
        this.mutationStatus = "success";
        return item;
      } catch (error) {
        this.mutationStatus = "error";
        this.mutationError = toErrorMessage(error);
        throw error;
      }
    },
    clearMutationState(): void {
      this.mutationStatus = "idle";
      this.mutationError = "";
    }
  }
});
`, typeName, typeName, typeName, typeName, typeName, typeName, typeName, typeName, module,
		storeName, module, typeName, typeName, typeName, typeName, typeName, typeName, typeName, typeName,
		typeName, typeName, typeName, typeName, typeName, primaryField(spec).Name, primaryField(spec).Name,
		typeName, typeName, primaryField(spec).Name, primaryField(spec).Name)
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

func renderPluginFrontendAPISupport(_ domaingenerator.GeneratorSpec) string {
	return `import { getPluginHost, type PluginHostRequestOptions } from "@skoll/plugin-sdk";
import { pluginContract } from "../contract";

export type ApiResponse<T> = { code: string; message: string; data: T };
const host = () => getPluginHost({ pluginId: pluginContract.plugin.id, requiredCapabilities: ["request"] });
const request = <T>(path: string, options: PluginHostRequestOptions): Promise<T> => host().request<T>(path, options);

export const apiGet = <T>(path: string): Promise<T> => request<T>(path, { method: "GET" });
export const apiPost = <T>(path: string, body?: unknown): Promise<T> => request<T>(path, { method: "POST", body });
export const apiPut = <T>(path: string, body?: unknown): Promise<T> => request<T>(path, { method: "PUT", body });
export const apiDelete = <T>(path: string): Promise<T> => request<T>(path, { method: "DELETE" });
`
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
	fmt.Fprintf(&b, "\t\t<template #filters>\n\t\t\t<BusinessFilterBar v-model=\"filters\" :fields=\"filterFields\" :locale=\"locale\" :busy=\"store.listStatus === 'loading'\" @search=\"searchFirst\" @reset=\"searchFirst\" />\n\t\t</template>\n")
	fmt.Fprintf(&b, "\t\t<template #stateActions>\n\t\t\t<el-button v-if=\"canRead && lifecycle.state === 'enabled'\" :icon=\"RefreshCw\" @click=\"searchFirst\">{{ t('%s.retry') }}</el-button>\n\t\t</template>\n", prefix)
	fmt.Fprintf(&b, "\t\t<BusinessList\n\t\t\t:items=\"store.items\"\n\t\t\t:columns=\"columns\"\n\t\t\t:state=\"listState\"\n\t\t\t:state-description=\"tableError\"\n\t\t\t:can-previous=\"store.canPrevious\"\n\t\t\t:can-next=\"store.hasMore\"\n\t\t\t:action-width=\"132\"\n\t\t\t:locale=\"locale\"\n\t\t\t@open=\"openDetail\"\n\t\t\t@previous=\"previousPage\"\n\t\t\t@next=\"nextPage\"\n\t\t>\n")
	fmt.Fprintf(&b, "\t\t\t<template #actions=\"{ row }\">\n\t\t\t\t<div class=\"plugin-generated-row-actions\">\n")
	fmt.Fprintf(&b, "\t\t\t\t\t<el-tooltip :content=\"t('%s.view')\"><el-button link :icon=\"Eye\" :aria-label=\"t('%s.view')\" @click=\"openDetail(row)\" /></el-tooltip>\n", prefix, prefix)
	fmt.Fprintf(&b, "\t\t\t\t\t<el-tooltip v-if=\"canUpdate\" :content=\"t('%s.edit')\"><el-button link type=\"primary\" :icon=\"Pencil\" :aria-label=\"t('%s.edit')\" @click=\"openEdit(row)\" /></el-tooltip>\n", prefix, prefix)
	fmt.Fprintf(&b, "\t\t\t\t\t<el-popconfirm v-if=\"canDelete\" :title=\"t('%s.deleteConfirm')\" :confirm-button-text=\"t('%s.delete')\" :cancel-button-text=\"t('%s.cancel')\" @confirm=\"remove(recordID(row))\">\n", prefix, prefix, prefix)
	fmt.Fprintf(&b, "\t\t\t\t\t\t<template #reference><el-button link type=\"danger\" :icon=\"Trash2\" :loading=\"deletingId === recordID(row)\" :aria-label=\"t('%s.delete')\" /></template>\n", prefix)
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
	fmt.Fprintf(&b, "import { pluginContract } from \"../../contract\";\n")
	fmt.Fprintf(&b, "import { useSkollHost } from \"../../skoll-host\";\n")
	fmt.Fprintf(&b, "import { use%sStore } from \"../../stores/%s\";\n", storeName, module)
	fmt.Fprintf(&b, "import type { %sInput } from \"../../api/%s\";\n\n", typeName, module)
	fmt.Fprintf(&b, "const store = use%sStore();\n", storeName)
	fmt.Fprintf(&b, "const { locale, lifecycle, can } = useSkollHost();\n")
	fmt.Fprintf(&b, "const { readKey: readPermission, createKey: createPermission, updateKey: updatePermission, deleteKey: deletePermission } = pluginContract.permissions;\n")
	fmt.Fprintf(&b, "const primaryField = pluginContract.data.primaryField;\n")
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
	fmt.Fprintf(&b, "function runCommand(command: BusinessCommand): void {\n\tif (command.id === \"create\") openCreate();\n\tif (command.id === \"refresh\") void refreshPage();\n}\n\n")
	fmt.Fprintf(&b, "const searchKeyword = (): string => String(filters.value.keyword || \"\").trim();\n")
	fmt.Fprintf(&b, "async function searchFirst(): Promise<void> {\n\tif (!canLoad()) return;\n\ttry { await store.search(searchKeyword()); } catch { /* Shared business states render the store error. */ }\n}\n")
	fmt.Fprintf(&b, "async function refreshPage(): Promise<void> {\n\tif (!canLoad()) return;\n\ttry { await store.refresh(); } catch { /* Shared business states render the store error. */ }\n}\n")
	fmt.Fprintf(&b, "async function nextPage(): Promise<void> {\n\tif (!canLoad()) return;\n\ttry { await store.next(); } catch { /* Shared business states render the store error. */ }\n}\n")
	fmt.Fprintf(&b, "async function previousPage(): Promise<void> {\n\tif (!canLoad()) return;\n\ttry { await store.previous(); } catch { /* Shared business states render the store error. */ }\n}\n")
	fmt.Fprintf(&b, "function canLoad(): boolean { return canRead.value && lifecycle.value.state === \"enabled\"; }\n")
	fmt.Fprintf(&b, "function recordID(row: BusinessRecord): string { return String((row as Record<string, unknown>)[primaryField] ?? \"\"); }\n\n")
	fmt.Fprintf(&b, "async function openDetail(row: BusinessRecord): Promise<void> {\n\tdetailOpen.value = true;\n\ttry { await store.loadOne(recordID(row)); } catch { /* The drawer renders the store error. */ }\n}\n\n")
	fmt.Fprintf(&b, "function openCreate(): void {\n\teditingId.value = null;\n\tresetForm();\n\tstore.clearMutationState();\n\tformOpen.value = true;\n}\n\n")
	fmt.Fprintf(&b, "function openEdit(row: BusinessRecord): void {\n\teditingId.value = recordID(row);\n\tresetForm();\n\tObject.assign(form, row);\n\tstore.clearMutationState();\n\tformOpen.value = true;\n}\n\n")
	fmt.Fprintf(&b, "async function save(): Promise<void> {\n\tif (!await formRef.value?.validate().catch(() => false)) return;\n\ttry {\n\t\tif (editingId.value) await store.update(editingId.value, { ...form }); else await store.create({ ...form });\n\t\tformOpen.value = false;\n\t\tElMessage.success(t(%q));\n\t} catch { /* The form keeps the backend error visible. */ }\n}\n\n", prefix+".saved")
	fmt.Fprintf(&b, "async function remove(id: string): Promise<void> {\n\tdeletingId.value = id;\n\ttry { await store.remove(id); ElMessage.success(t(%q)); } catch { ElMessage.error(store.mutationError); } finally { deletingId.value = \"\"; }\n}\n\n", prefix+".deleted")
	fmt.Fprintf(&b, "function resetForm(): void {\n\tconst mutableForm = form as Record<string, unknown>;\n\tfor (const key of Object.keys(mutableForm)) delete mutableForm[key];\n")
	for _, fieldName := range spec.Page.Form.Fields {
		if field, ok := spec.FieldByName(fieldName); ok {
			fmt.Fprintf(&b, "\tform.%s = %s;\n", field.Name, tsFieldDefault(field))
		}
	}
	fmt.Fprintf(&b, "}\n\nfunction displayValue(value: unknown): string {\n\tif (value === null || value === undefined || value === \"\") return \"-\";\n\tif (typeof value === \"object\") return JSON.stringify(value);\n\treturn String(value);\n}\n\n")
	fmt.Fprintf(&b, "watch(() => lifecycle.value.state, (state) => { if (state === \"enabled\" && canRead.value && store.listStatus === \"idle\") void searchFirst(); });\n")
	fmt.Fprintf(&b, "onMounted(() => { if (canRead.value && lifecycle.value.state === \"enabled\") void searchFirst(); });\n")
	fmt.Fprintf(&b, "</script>\n\n<style scoped>\n.plugin-generated-row-actions { display: flex; align-items: center; gap: 4px; }\n:deep(.el-drawer__body) { padding: 16px; }\n:deep(.el-input-number), :deep(.el-date-editor) { width: 100%%; }\n@media (max-width: 560px) { .plugin-generated-row-actions { flex-wrap: wrap; } }\n</style>\n")
	return b.String()
}

func renderPluginFrontendHostSupport(_ domaingenerator.GeneratorSpec) string {
	return `import type { BusinessLocale } from "@skoll/business-ui/core";
import {
  getPluginHost,
  type PluginHostLifecycle,
  type PluginHostTheme
} from "@skoll/plugin-sdk";
import { readonly, ref } from "vue";
import { pluginContract } from "./contract";

const host = getPluginHost({
  pluginId: pluginContract.plugin.id,
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
`
}

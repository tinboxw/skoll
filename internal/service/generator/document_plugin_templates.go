package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	domaingenerator "github.com/tinboxw/skoll/internal/domain/generator"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

func renderDocumentSchemaJSON(spec domaingenerator.GeneratorSpec) string {
	schema := generatedDocumentSchema(spec)
	raw, _ := json.MarshalIndent(schema, "", "  ")
	return string(raw) + "\n"
}

func generatedDocumentSchema(spec domaingenerator.GeneratorSpec) pluginsdk.DocumentSchema {
	header := make([]pluginsdk.DocumentFieldSchema, 0, len(spec.Fields)-1)
	for _, field := range spec.Fields {
		if field.PrimaryKey {
			continue
		}
		header = append(header, pluginsdk.DocumentFieldSchema{
			Key: field.Name, Label: field.Label, Type: documentFieldType(field.Type), Required: field.Required,
		})
	}
	return pluginsdk.DocumentSchema{
		Key: spec.Document.SchemaKey, Name: spec.Document.SchemaName, Version: 1, InitialState: "draft",
		Header: header, Lines: []pluginsdk.DocumentLineSchema{},
		States: []pluginsdk.DocumentStateSchema{
			{Key: "draft", Name: "Draft"},
			{Key: "submitted", Name: "Submitted"},
			{Key: "approved", Name: "Approved", Terminal: true},
			{Key: "rejected", Name: "Rejected", Terminal: true},
		},
		Actions: []pluginsdk.DocumentActionSchema{
			{Key: "submit", Name: "Submit", From: []string{"draft"}, To: "submitted"},
			{Key: "approve", Name: "Approve", From: []string{"submitted"}, To: "approved"},
			{Key: "reject", Name: "Reject", From: []string{"submitted"}, To: "rejected", RequiresComment: true},
		},
	}
}

func documentFieldType(fieldType domaingenerator.FieldType) pluginsdk.DocumentFieldType {
	switch fieldType {
	case domaingenerator.FieldTypeText:
		return pluginsdk.DocumentFieldText
	case domaingenerator.FieldTypeInt:
		return pluginsdk.DocumentFieldInteger
	case domaingenerator.FieldTypeDecimal:
		return pluginsdk.DocumentFieldDecimal
	case domaingenerator.FieldTypeBool:
		return pluginsdk.DocumentFieldBoolean
	case domaingenerator.FieldTypeTime:
		return pluginsdk.DocumentFieldDateTime
	case domaingenerator.FieldTypeJSON:
		return pluginsdk.DocumentFieldJSON
	default:
		return pluginsdk.DocumentFieldString
	}
}

func renderDocumentFrontendSchema(spec domaingenerator.GeneratorSpec) string {
	return "import type { DocumentSchema } from \"@skoll/document-ui\";\n\nexport const documentSchema: DocumentSchema = " + strings.TrimSpace(renderDocumentSchemaJSON(spec)) + ";\n"
}

func renderDocumentPluginBackendServer(spec domaingenerator.GeneratorSpec) string {
	template := `package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/tinboxw/skoll/pkg/pluginclient"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const (
	apiPath = {{API_PATH}}
	defaultAddress = {{DEFAULT_ADDRESS}}
	tenantID = "tenant-demo"
	definitionID = {{DEFINITION_ID}}
	titleField = {{TITLE_FIELD}}
	numberPrefix = {{NUMBER_PREFIX}}
	readPermission = {{READ_PERMISSION}}
	createPermission = {{CREATE_PERMISSION}}
	managePermission = {{MANAGE_PERMISSION}}
)

type documentServer struct {
	host pluginsdk.HostServices
	schema pluginsdk.DocumentSchema
	mu sync.RWMutex
	sequence uint64
	drafts map[string]pluginsdk.DocumentDraft
}

func main() {
	client, err := pluginclient.FromEnvironment()
	if err != nil { log.Fatal(err) }
	host, err := client.HostServices()
	if err != nil { log.Fatal(err) }
	schema := documentSchema()
	if err := ensureWorkflowDefinition(context.Background(), host.Workflows); err != nil { log.Fatal(err) }
	server := &documentServer{host: host, schema: schema, drafts: make(map[string]pluginsdk.DocumentDraft)}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, http.StatusOK, map[string]any{"status": "ready", "plugin": {{PLUGIN_ID}}, "documentType": schema.Key}) })
	mux.HandleFunc(apiPath, server.collection)
	mux.HandleFunc(apiPath+"/", server.item)
	address := strings.TrimSpace(os.Getenv(pluginclient.EnvironmentPluginAddress))
	if address == "" { address = defaultAddress }
	log.Printf("%s listening on %s", {{PLUGIN_ID}}, address)
	log.Fatal(http.ListenAndServe(address, mux))
}

func documentSchema() pluginsdk.DocumentSchema {
	var schema pluginsdk.DocumentSchema
	if err := json.Unmarshal([]byte({{SCHEMA_JSON}}), &schema); err != nil { panic(err) }
	if err := schema.Validate(); err != nil { panic(err) }
	return schema
}

func ensureWorkflowDefinition(ctx context.Context, workflows pluginsdk.WorkflowService) error {
	current, err := workflows.GetDefinition(ctx, definitionID)
	if err == nil && current.ID != "" {
		if current.Status == pluginsdk.WorkflowDefinitionPublished { return nil }
		_, err = workflows.PublishDefinition(ctx, current.ID)
		return err
	}
	created, err := workflows.CreateDefinition(ctx, pluginsdk.WorkflowDefinitionInput{
		ID: definitionID, Key: definitionID, Name: {{SCHEMA_NAME}}+" Approval", Version: 1,
		Nodes: []pluginsdk.WorkflowNode{
			{ID: "start", Key: "start", Name: "Start", Type: pluginsdk.WorkflowNodeStart},
			{ID: "approval", Key: "approval", Name: "Approval", Type: pluginsdk.WorkflowNodeApproval, AssigneeIDs: []string{"approver"}},
			{ID: "end", Key: "end", Name: "End", Type: pluginsdk.WorkflowNodeEnd},
		},
		Transitions: []pluginsdk.WorkflowTransition{{From: "start", To: "approval"}, {From: "approval", To: "end"}},
	})
	if err != nil { return fmt.Errorf("create workflow definition: %w", err) }
	_, err = workflows.PublishDefinition(ctx, created.ID)
	return err
}

func (s *documentServer) collection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		states := r.URL.Query()["state"]
		page, err := s.host.Documents.Search(requestContext(r), pluginsdk.DocumentSearchInput{
			TenantID: tenantID, Permission: permission(readPermission, "read"), Types: []string{s.schema.Key}, States: states,
			Text: strings.TrimSpace(r.URL.Query().Get("keyword")), SortField: pluginsdk.DocumentSearchSortUpdatedAt, Direction: pluginsdk.DocumentSearchDescending, Limit: 50,
		})
		writeResult(w, page, err)
	case http.MethodPost:
		var input map[string]any
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil { writeError(w, http.StatusBadRequest, err); return }
		draft, err := s.createDraft(input)
		if err != nil { writeError(w, http.StatusBadRequest, err); return }
		writeJSON(w, http.StatusCreated, envelope(draft))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *documentServer) item(w http.ResponseWriter, r *http.Request) {
	relative := strings.Trim(strings.TrimPrefix(r.URL.Path, apiPath), "/")
	parts := strings.Split(relative, "/")
	if len(parts) == 1 && parts[0] == "export" && r.Method == http.MethodGet { s.export(w, r); return }
	if len(parts) == 1 && r.Method == http.MethodGet {
		result, err := s.host.Documents.Get(requestContext(r), pluginsdk.DocumentWorkflowGetInput{TenantID: tenantID, Permission: permission(readPermission, "read"), DocumentID: parts[0]})
		writeResult(w, result, err); return
	}
	if len(parts) != 2 || r.Method != http.MethodPost { http.NotFound(w, r); return }
	switch parts[1] {
	case "submit": s.submit(w, r, parts[0])
	case "approve": s.approve(w, r, parts[0])
	default: http.NotFound(w, r)
	}
}

func (s *documentServer) createDraft(input map[string]any) (pluginsdk.DocumentDraft, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	id := strings.TrimSpace(fmt.Sprint(input[{{PRIMARY_FIELD}}]))
	if id == "" || id == "<nil>" { id = fmt.Sprintf("document-%06d", s.sequence) }
	if _, exists := s.drafts[id]; exists { return pluginsdk.DocumentDraft{}, fmt.Errorf("document already exists") }
	number := strings.TrimSpace(fmt.Sprint(input["number"]))
	if number == "" || number == "<nil>" { number = fmt.Sprintf("%s-%06d", numberPrefix, s.sequence) }
	title := strings.TrimSpace(fmt.Sprint(input[titleField]))
	values := make(map[string]pluginsdk.DocumentValue, len(s.schema.Header))
	for _, field := range s.schema.Header {
		raw, exists := input[field.Key]
		if !exists { continue }
		values[field.Key] = pluginsdk.DocumentValue{Type: field.Type, Value: valueString(raw)}
	}
	draft := pluginsdk.DocumentDraft{ID: id, Type: s.schema.Key, SchemaVersion: s.schema.Version, Number: number, Title: title, Header: values, Lines: map[string][]pluginsdk.DocumentLine{}}
	s.drafts[id] = draft
	return draft, nil
}

func (s *documentServer) submit(w http.ResponseWriter, r *http.Request, id string) {
	s.mu.RLock(); draft, exists := s.drafts[id]; s.mu.RUnlock()
	if !exists { writeError(w, http.StatusNotFound, fmt.Errorf("draft not found")); return }
	var result pluginsdk.DocumentWorkflowResult
	err := s.host.Transactions.Within(requestContext(r), func(tx pluginsdk.Transaction) error {
		var err error
		result, err = s.host.Documents.Submit(tx.Context(), pluginsdk.DocumentWorkflowSubmitInput{
			TenantID: tenantID, Permission: permission(createPermission, "submit"), Schema: s.schema, Draft: draft,
			DefinitionID: definitionID, InstanceID: id+"-workflow", IdempotencyKey: id+"-submit-v1",
		})
		return err
	})
	writeResult(w, result, err)
}

func (s *documentServer) approve(w http.ResponseWriter, r *http.Request, id string) {
	current, err := s.host.Documents.Get(requestContext(r), pluginsdk.DocumentWorkflowGetInput{TenantID: tenantID, Permission: permission(readPermission, "read"), DocumentID: id})
	if err != nil { writeError(w, http.StatusBadRequest, err); return }
	taskID := ""
	for _, task := range current.Workflow.Tasks { if task.Status == pluginsdk.WorkflowTaskPending { taskID = task.ID; break } }
	if taskID == "" { writeError(w, http.StatusConflict, fmt.Errorf("pending approval task not found")); return }
	var result pluginsdk.DocumentWorkflowResult
	err = s.host.Transactions.Within(requestContext(r), func(tx pluginsdk.Transaction) error {
		var actionErr error
		result, actionErr = s.host.Documents.Act(tx.Context(), pluginsdk.DocumentWorkflowActionInput{
			TenantID: tenantID, Permission: permission(managePermission, "approve"), DocumentID: id,
			Action: pluginsdk.DocumentWorkflowApprove, ExpectedVersion: current.Document.Version, TaskID: taskID,
			IdempotencyKey: id+"-approve-v"+strconv.FormatInt(current.Document.Version, 10),
		})
		return actionErr
	})
	writeResult(w, result, err)
}

func (s *documentServer) export(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock(); s.sequence++; sequence := s.sequence; s.mu.Unlock()
	jobID := fmt.Sprintf("document-export-%06d", sequence)
	job, err := s.host.Documents.Export(requestContext(r), pluginsdk.DocumentExportInput{
		JobID: jobID, IdempotencyKey: jobID, Format: pluginsdk.DocumentExportCSV, MaxRows: 1000,
		Search: pluginsdk.DocumentSearchInput{TenantID: tenantID, Permission: permission(readPermission, "read"), Types: []string{s.schema.Key}, SortField: pluginsdk.DocumentSearchSortUpdatedAt, Direction: pluginsdk.DocumentSearchDescending, Limit: 50},
	})
	writeResult(w, job, err)
}

func permission(resource, action string) pluginsdk.Permission { return pluginsdk.Permission{Resource: resource, Action: action} }
func requestContext(r *http.Request) context.Context {
	if r == nil { return context.Background() }
	token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	return pluginclient.WithUserToken(r.Context(), token)
}
func valueString(value any) string { if value == nil { return "" }; if raw, ok := value.(string); ok { return raw }; encoded, _ := json.Marshal(value); return string(encoded) }
func envelope(data any) map[string]any { return map[string]any{"code": "ok", "message": "", "data": data} }
func writeResult(w http.ResponseWriter, data any, err error) { if err != nil { writeError(w, http.StatusBadRequest, err); return }; writeJSON(w, http.StatusOK, envelope(data)) }
func writeError(w http.ResponseWriter, status int, err error) { writeJSON(w, status, map[string]any{"code": "request_failed", "message": err.Error(), "data": nil}) }
func writeJSON(w http.ResponseWriter, status int, payload any) { w.Header().Set("Content-Type", "application/json"); w.WriteHeader(status); _ = json.NewEncoder(w).Encode(payload) }
`
	return strings.NewReplacer(
		"{{API_PATH}}", fmt.Sprintf("%q", pluginAPIBasePath(spec)),
		"{{DEFAULT_ADDRESS}}", fmt.Sprintf("%q", pluginServiceAddress(spec.Plugin.ServiceBaseURL)),
		"{{PLUGIN_ID}}", fmt.Sprintf("%q", spec.Plugin.ID),
		"{{DEFINITION_ID}}", fmt.Sprintf("%q", spec.Document.DefinitionID),
		"{{TITLE_FIELD}}", fmt.Sprintf("%q", spec.Document.TitleField),
		"{{NUMBER_PREFIX}}", fmt.Sprintf("%q", spec.Document.NumberPrefix),
		"{{READ_PERMISSION}}", fmt.Sprintf("%q", spec.Permissions.ReadKey),
		"{{CREATE_PERMISSION}}", fmt.Sprintf("%q", spec.Permissions.CreateKey),
		"{{MANAGE_PERMISSION}}", fmt.Sprintf("%q", spec.Permissions.ManageKey),
		"{{PRIMARY_FIELD}}", fmt.Sprintf("%q", primaryField(spec).Name),
		"{{SCHEMA_JSON}}", fmt.Sprintf("%q", strings.TrimSpace(renderDocumentSchemaJSON(spec))),
		"{{SCHEMA_NAME}}", fmt.Sprintf("%q", spec.Document.SchemaName),
	).Replace(template)
}

func renderDocumentPluginFrontendAPI(spec domaingenerator.GeneratorSpec) string {
	base := pluginAPIBasePath(spec)
	return fmt.Sprintf(`import type { DocumentDraft, DocumentSearchPage, DocumentWorkflowResult } from "@skoll/document-ui";
import { apiGet, apiPost, type ApiResponse } from "../utils/api";

const basePath = %q;
export const searchDocuments = (keyword = "", state = "") => apiGet<ApiResponse<DocumentSearchPage>>(basePath + "?keyword=" + encodeURIComponent(keyword) + (state ? "&state=" + encodeURIComponent(state) : ""));
export const createDocument = (input: Record<string, unknown>) => apiPost<ApiResponse<DocumentDraft>>(basePath, input);
export const submitDocument = (id: string) => apiPost<ApiResponse<DocumentWorkflowResult>>(basePath + "/" + encodeURIComponent(id) + "/submit");
export const getDocument = (id: string) => apiGet<ApiResponse<DocumentWorkflowResult>>(basePath + "/" + encodeURIComponent(id));
export const approveDocument = (id: string) => apiPost<ApiResponse<DocumentWorkflowResult>>(basePath + "/" + encodeURIComponent(id) + "/approve");
export const exportDocuments = () => apiGet<ApiResponse<{ ID: string; Kind: string; Status: string }>>(basePath + "/export");
`, base)
}

func renderDocumentPluginFrontendAPISupport(spec domaingenerator.GeneratorSpec) string {
	return fmt.Sprintf(`export type ApiResponse<T> = { code: string; message: string; data: T };
type HostRequest = { method?: "GET" | "POST" | "PUT" | "DELETE"; body?: unknown };
type SkollHost = { pluginId: string; request<T>(path: string, options?: HostRequest): Promise<T> };

declare global { interface Window { __SKOLL_HOST__?: SkollHost } }

function host(): SkollHost {
  const current = window.__SKOLL_HOST__;
  if (!current || current.pluginId !== %q) throw new Error("Skoll plugin host context is unavailable");
  return current;
}

export const apiGet = <T>(path: string): Promise<T> => host().request<T>(path);
export const apiPost = <T>(path: string, body?: unknown): Promise<T> => host().request<T>(path, { method: "POST", body });
export const apiPut = <T>(path: string, body?: unknown): Promise<T> => host().request<T>(path, { method: "PUT", body });
export const apiDelete = <T>(path: string): Promise<T> => host().request<T>(path, { method: "DELETE" });
`, spec.Plugin.ID)
}

func renderDocumentPluginFrontendMain(spec domaingenerator.GeneratorSpec) string {
	return fmt.Sprintf(`import { createApp } from "vue";
import { createPinia } from "pinia";
import {
  ElAlert, ElButton, ElDatePicker, ElDrawer, ElEmpty, ElForm, ElFormItem, ElInput, ElOption,
  ElSelect, ElSkeleton, ElSwitch, ElTabPane, ElTable, ElTableColumn, ElTabs, ElTag, ElTimeline,
  ElTimelineItem, ElTooltip
} from "element-plus";
import "element-plus/dist/index.css";
import Page from "./views/%s/index.vue";
import "./styles.css";

const app = createApp(Page);
app.use(createPinia());
for (const component of [ElAlert, ElButton, ElDatePicker, ElDrawer, ElEmpty, ElForm, ElFormItem, ElInput, ElOption, ElSelect, ElSkeleton, ElSwitch, ElTabPane, ElTable, ElTableColumn, ElTabs, ElTag, ElTimeline, ElTimelineItem, ElTooltip]) {
  app.component(component.name!, component);
}
app.mount("#app");
`, spec.Table.DomainName)
}

func renderDocumentPluginFrontendStore(spec domaingenerator.GeneratorSpec) string {
	return fmt.Sprintf(`import { defineStore } from "pinia";
import type { DocumentSearchPage, DocumentWorkflowResult } from "@skoll/document-ui";
import { getDocument, searchDocuments } from "../api/%s";

export const use%sStore = defineStore(%q, {
  state: () => ({ page: { items: [], hasMore: false } as DocumentSearchPage, selected: null as DocumentWorkflowResult | null }),
  actions: {
    async search(keyword = "", state = "") { this.page = (await searchDocuments(keyword, state)).data; },
    async open(id: string) { this.selected = (await getDocument(id)).data; }
  }
});
`, spec.Module.Package, exportedName(spec.Module.Name)+"Document", spec.Plugin.ID+"-documents")
}

func renderDocumentPluginFrontendView(spec domaingenerator.GeneratorSpec) string {
	var initial bytes.Buffer
	for _, field := range generatedDocumentSchema(spec).Header {
		fmt.Fprintf(&initial, "    %q: { type: %q, value: \"\" },\n", field.Key, field.Type)
	}
	return fmt.Sprintf(`<script setup lang="ts">
import { DocumentDetail, DocumentForm, DocumentList, type DocumentActionRequest, type DocumentDraft, type DocumentSearchPage, type DocumentWorkflowResult } from "@skoll/document-ui";
import { ElMessage } from "element-plus";
import { ref } from "vue";
import { approveDocument, createDocument, exportDocuments, getDocument, searchDocuments, submitDocument } from "../../api/%s";
import { documentSchema } from "../../document-schema";

const loading = ref(false), busy = ref(false), error = ref(""), query = ref(""), state = ref("");
const page = ref<DocumentSearchPage>({ items: [], hasMore: false });
const editorOpen = ref(false), detailOpen = ref(false), selected = ref<DocumentWorkflowResult | null>(null);
const draft = ref<DocumentDraft>(emptyDraft());

function emptyDraft(): DocumentDraft { return { id: "", type: documentSchema.key, schemaVersion: documentSchema.version, number: "AUTO", title: "", header: {
%s  }, lines: {} }; }
async function refresh() { loading.value = true; error.value = ""; try { page.value = (await searchDocuments(query.value, state.value)).data; } catch (cause) { error.value = String(cause); } finally { loading.value = false; } }
function openCreate() { draft.value = emptyDraft(); editorOpen.value = true; }
async function createAndSubmit(value: DocumentDraft) {
  busy.value = true; error.value = "";
  try {
    const body = Object.fromEntries(Object.entries(value.header).map(([key, item]) => [key, item.value || ""]));
    const created = (await createDocument({ ...body, number: value.number === "AUTO" ? "" : value.number })).data;
    selected.value = (await submitDocument(created.id)).data;
    editorOpen.value = false; detailOpen.value = true; await refresh(); ElMessage.success("Document submitted");
  } catch (cause) { error.value = String(cause); } finally { busy.value = false; }
}
async function openDocument(id: string) { busy.value = true; try { selected.value = (await getDocument(id)).data; detailOpen.value = true; } finally { busy.value = false; } }
async function act(request: DocumentActionRequest) { if (!selected.value || request.action !== "approve") return; busy.value = true; try { selected.value = (await approveDocument(selected.value.document.id)).data; await refresh(); ElMessage.success("Document approved"); } finally { busy.value = false; } }
async function runExport() { const job = (await exportDocuments()).data; ElMessage.success("Export scheduled: " + job.ID); }
void refresh();
</script>

<template>
  <main class="document-workspace">
    <DocumentList v-model:query="query" v-model:state-filter="state" :schema="documentSchema" :page="page" :loading="loading" :error="error" can-create can-export @search="refresh" @refresh="refresh" @create="openCreate" @open="item => openDocument(item.id)" @export="runExport" />
    <el-drawer v-model="editorOpen" title="Create document" size="min(720px, 100%%)"><DocumentForm v-model="draft" :schema="documentSchema" :busy="busy" :show-save="false" @submit="createAndSubmit" @cancel="editorOpen = false" /></el-drawer>
    <el-drawer v-model="detailOpen" :title="selected?.document.title || documentSchema.name" size="min(860px, 100%%)"><DocumentDetail v-if="selected" :schema="documentSchema" :document="selected.document" :workflow="selected.workflow" :busy="busy" :available-actions="selected.document.state === 'submitted' ? ['approve'] : []" can-export @action="act" @export="runExport" /></el-drawer>
  </main>
</template>

<style scoped>.document-workspace { min-width: 0; } :deep(.el-drawer__body) { padding: 16px; }</style>
`, spec.Module.Package, initial.String())
}

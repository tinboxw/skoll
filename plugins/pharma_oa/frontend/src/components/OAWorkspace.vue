<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue";
import { ElMessage } from "element-plus/es/components/message/index.mjs";
import { ElMessageBox } from "element-plus/es/components/message-box/index.mjs";
import {
  Bell,
  Check,
  CircleAlert,
  Clock3,
  Edit3,
  Eye,
  FileText,
  Forward,
  MessageSquare,
  Paperclip,
  Plus,
  RefreshCw,
  RotateCcw,
  Search,
  Send,
  X
} from "lucide-vue-next";
import { api } from "../api";
import { t } from "../i18n";
import { createOAEditorForm, hydrateOAEditorForm, toOARequestWriteInput, validateOAEditorForm } from "../oa-form";
import type { OAEditorForm } from "../oa-form";
import type {
  OARequest,
  OARequestDetail,
  OARequestStatus,
  OARequestType,
  OAWorkflowAction,
  OAWorkspaceMode,
  Scope
} from "../types";
import OARequestEditor from "./OARequestEditor.vue";

type ActionKind = "approve" | "reject" | "withdraw" | "cancel" | "delegate" | "remind";

const props = defineProps<{ mode: OAWorkspaceMode; scope: Scope }>();
const loading = ref(false);
const saving = ref(false);
const error = ref("");
const denied = ref(false);
const keyword = ref("");
const requestType = ref<OARequestType | "">("");
const status = ref<OARequestStatus | "">("");
const items = ref<OARequest[]>([]);
const total = ref(0);
const selected = ref<OARequest | null>(null);
const detail = ref<OARequestDetail | null>(null);
const editorOpen = ref(false);
const detailOpen = ref(false);
const actionKind = ref<ActionKind | null>(null);
const form = ref<OAEditorForm>(createOAEditorForm());
const editing = ref<OARequest | null>(null);
const commentText = ref("");
const attachmentInput = ref<HTMLInputElement | null>(null);
const actionForm = reactive({ comment: "", targetId: "", targetName: "", runAt: "" });

const isInbox = computed(() => props.mode === "inbox");
const draftCount = computed(() => items.value.filter((item) => item.status === "draft").length);
const pendingCount = computed(() => items.value.filter((item) => item.status === "pending").length);
const finishedCount = computed(() => items.value.filter((item) => ["approved", "rejected", "withdrawn", "canceled"].includes(item.status)).length);
const currentTask = computed(() => detail.value?.workflow?.tasks.find((task) => task.status === "pending"));
const actionDialogOpen = computed({
  get: () => actionKind.value !== null,
  set: (open: boolean) => { if (!open) actionKind.value = null; }
});
const actionTitle = computed(() => {
  if (actionKind.value === "approve") return t("approveRequest");
  if (actionKind.value === "reject") return t("rejectRequest");
  if (actionKind.value === "withdraw") return t("withdrawRequest");
  if (actionKind.value === "cancel") return t("cancelRequest");
  if (actionKind.value === "delegate") return t("delegateRequest");
  return t("scheduleReminder");
});
const requestTypes = computed<Array<{ value: OARequestType; label: string }>>(() => [
  { value: "leave", label: t("leaveRequest") },
  { value: "expense", label: t("expenseRequest") },
  { value: "procurement", label: t("procurementRequest") },
  { value: "contract", label: t("contractRequest") },
  { value: "custom", label: t("customRequest") }
]);

onMounted(() => {
  if (isInbox.value) status.value = "pending";
  void loadRequests();
});

watch(() => props.mode, () => {
  status.value = isInbox.value ? "pending" : "";
  selected.value = null;
  detail.value = null;
  void loadRequests();
});

let searchTimer: number | undefined;
watch([keyword, requestType, status], () => {
  window.clearTimeout(searchTimer);
  searchTimer = window.setTimeout(() => void loadRequests(), 250);
});

async function loadRequests(): Promise<void> {
  if (loading.value) return;
  loading.value = true;
  error.value = "";
  denied.value = false;
  try {
    const page = await api.oaRequests({
      keyword: keyword.value.trim(),
      requestType: requestType.value,
      status: isInbox.value ? "pending" : status.value,
      limit: 200
    });
    items.value = page.items ?? [];
    total.value = page.total ?? items.value.length;
  } catch (reason) {
    error.value = errorMessage(reason);
    denied.value = /forbidden|permission|denied|权限|拒绝/i.test(error.value);
  } finally {
    loading.value = false;
  }
}

function openCreate(): void {
  editing.value = null;
  form.value = createOAEditorForm();
  editorOpen.value = true;
}

function openEdit(item: OARequest): void {
  if (item.status !== "draft") return;
  editing.value = item;
  form.value = hydrateOAEditorForm(item);
  editorOpen.value = true;
}

async function saveEditor(submit: boolean): Promise<void> {
  if (!validateOAEditorForm(form.value) || saving.value) {
    ElMessage.warning(t("requiredFields"));
    return;
  }
  saving.value = true;
  error.value = "";
  try {
    const write = toOARequestWriteInput(form.value, props.scope, editing.value?.version);
    const saved = editing.value
      ? await api.updateOARequest(editing.value.id, write)
      : await api.createOARequest(write);
    let item = saved.item;
    if (submit) {
      const submitted = await api.submitOARequest(item.id, item.version);
      item = submitted.item;
      ElMessage.success(t("requestSubmitted"));
    } else {
      ElMessage.success(t("requestSaved"));
    }
    replaceItem(item);
    editorOpen.value = false;
    await loadRequests();
  } catch (reason) {
    error.value = errorMessage(reason);
    ElMessage.error(error.value);
  } finally {
    saving.value = false;
  }
}

async function submitItem(item: OARequest): Promise<void> {
  try {
    await ElMessageBox.confirm(t("confirmSubmit"), t("submitRequest"), {
      type: "warning",
      confirmButtonText: t("submitRequest"),
      cancelButtonText: t("cancel")
    });
  } catch {
    return;
  }
  saving.value = true;
  try {
    const result = await api.submitOARequest(item.id, item.version);
    replaceItem(result.item);
    ElMessage.success(t("requestSubmitted"));
    await loadRequests();
  } catch (reason) {
    showMutationError(reason);
  } finally {
    saving.value = false;
  }
}

async function openDetail(item: OARequest): Promise<void> {
  selected.value = item;
  detailOpen.value = true;
  error.value = "";
  try {
    detail.value = await api.oaRequest(item.id);
    selected.value = detail.value.item;
    replaceItem(detail.value.item);
  } catch (reason) {
    error.value = errorMessage(reason);
    ElMessage.error(error.value);
  }
}

function openAction(kind: ActionKind, item?: OARequest): void {
  if (item && selected.value?.id !== item.id) {
    void openDetail(item).then(() => openAction(kind));
    return;
  }
  Object.assign(actionForm, { comment: "", targetId: "", targetName: "", runAt: defaultReminderTime() });
  actionKind.value = kind;
}

async function runAction(): Promise<void> {
  const item = detail.value?.item ?? selected.value;
  const kind = actionKind.value;
  if (!item || !kind || saving.value) return;
  if (["reject", "withdraw", "cancel"].includes(kind) && !actionForm.comment.trim()) {
    ElMessage.warning(t("formRequired"));
    return;
  }
  if (kind === "delegate" && (!actionForm.targetId.trim() || !actionForm.targetName.trim() || !currentTask.value)) {
    ElMessage.warning(t("formRequired"));
    return;
  }
  saving.value = true;
  try {
    let result;
    if (kind === "approve") {
      result = await api.approveOARequest(item.id, requireTaskID(), item.version, actionForm.comment.trim());
    } else if (kind === "reject") {
      result = await api.rejectOARequest(item.id, requireTaskID(), item.version, actionForm.comment.trim());
    } else if (kind === "withdraw") {
      result = await api.withdrawOARequest(item.id, item.version, actionForm.comment.trim());
    } else if (kind === "cancel") {
      result = await api.cancelOARequest(item.id, item.version, actionForm.comment.trim());
    } else if (kind === "delegate") {
      result = await api.delegateOARequest(item.id, requireTaskID(), item.version, actionForm.targetId.trim(), actionForm.targetName.trim(), actionForm.comment.trim());
    } else {
      const runAt = new Date(actionForm.runAt);
      const reminded = await api.remindOARequest(item.id, item.version, runAt.toISOString());
      result = { item: reminded.item, workflow: detail.value?.workflow };
    }
    detail.value = { item: result.item, workflow: result.workflow };
    selected.value = result.item;
    replaceItem(result.item);
    actionKind.value = null;
    ElMessage.success(kind === "remind" ? t("reminderScheduled") : t("actionCompleted"));
    await loadRequests();
  } catch (reason) {
    showMutationError(reason);
  } finally {
    saving.value = false;
  }
}

async function addComment(): Promise<void> {
  const item = detail.value?.item;
  const content = commentText.value.trim();
  if (!item || !content || saving.value) return;
  saving.value = true;
  try {
    const result = await api.commentOARequest(item.id, item.version, content);
    if (detail.value) detail.value = { ...detail.value, item: result.item };
    selected.value = result.item;
    replaceItem(result.item);
    commentText.value = "";
    ElMessage.success(t("commentAdded"));
  } catch (reason) {
    showMutationError(reason);
  } finally {
    saving.value = false;
  }
}

function chooseAttachment(): void {
  attachmentInput.value?.click();
}

async function addAttachment(event: Event): Promise<void> {
  const item = detail.value?.item;
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = "";
  if (!item || !file || saving.value) return;
  if (file.size > 5 * 1024 * 1024) {
    ElMessage.error(t("fileLimit"));
    return;
  }
  saving.value = true;
  try {
    const contentBase64 = await readBase64(file);
    const result = await api.attachOARequest(item.id, item.version, file.name, contentBase64);
    if (detail.value) detail.value = { ...detail.value, item: result.item };
    selected.value = result.item;
    replaceItem(result.item);
    ElMessage.success(t("attachmentAdded"));
  } catch (reason) {
    showMutationError(reason);
  } finally {
    saving.value = false;
  }
}

function replaceItem(item: OARequest): void {
  const index = items.value.findIndex((current) => current.id === item.id);
  if (index >= 0) items.value.splice(index, 1, item);
  else items.value.unshift(item);
}

function requireTaskID(): string {
  if (!currentTask.value) throw new Error("CURRENT_APPROVAL_TASK_UNAVAILABLE");
  return currentTask.value.id;
}

function showMutationError(reason: unknown): void {
  error.value = errorMessage(reason);
  ElMessage.error(error.value);
}

function errorMessage(reason: unknown): string {
  return reason instanceof Error ? reason.message : t("failed");
}

function requestTypeLabel(value: OARequestType): string {
  return requestTypes.value.find((item) => item.value === value)?.label ?? value;
}

function statusLabel(value: string): string {
  const key = value === "running" ? "running" : value as "draft" | "pending" | "approved" | "rejected" | "withdrawn" | "canceled";
  return t(key);
}

function statusType(value: OARequestStatus): "success" | "warning" | "danger" | "info" {
  if (value === "approved") return "success";
  if (value === "draft" || value === "pending") return "warning";
  if (value === "rejected" || value === "canceled") return "danger";
  return "info";
}

function timelineLabel(action: OAWorkflowAction): string {
  return t(action.type);
}

function formSummary(item: OARequest): string {
  const data = item.formData;
  if (item.requestType === "leave") return `${data.leaveType ?? "-"} · ${data.startDate ?? "-"} - ${data.endDate ?? "-"}`;
  if (item.requestType === "expense") return `${data.category ?? "-"} · CNY ${data.amount ?? 0}`;
  if (item.requestType === "procurement") return `${data.purpose ?? "-"} · CNY ${data.amount ?? 0}`;
  if (item.requestType === "contract") return `${data.counterparty ?? "-"} · CNY ${data.amount ?? 0}`;
  return String(data.content ?? data.formKey ?? "-");
}

function formatTime(value?: string): string {
  if (!value) return "-";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

function formatBytes(size: number): string {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / 1024 / 1024).toFixed(1)} MB`;
}

function defaultReminderTime(): string {
  const date = new Date(Date.now() + 24 * 60 * 60 * 1000);
  date.setSeconds(0, 0);
  const offset = date.getTimezoneOffset() * 60000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 16);
}

function readBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result).split(",")[1] ?? "");
    reader.onerror = () => reject(reader.error);
    reader.readAsDataURL(file);
  });
}
</script>

<template>
  <section class="oa-workspace" :data-mode="mode">
    <header class="module-heading oa-heading">
      <div>
        <p>{{ t("brand") }} · {{ isInbox ? t("approvalInbox") : t("requestCenter") }}</p>
        <h2>{{ t("oaRequestTitle") }}</h2>
        <span>{{ isInbox ? t("oaInboxDescription") : t("oaRequestDescription") }}</span>
      </div>
      <div class="heading-actions">
        <el-tooltip :content="t('refresh')">
          <el-button :icon="RefreshCw" circle :loading="loading" :aria-label="t('refresh')" @click="loadRequests" />
        </el-tooltip>
        <el-button v-if="!isInbox" type="primary" :icon="Plus" @click="openCreate">{{ t("newRequest") }}</el-button>
      </div>
    </header>

    <section class="metrics oa-metrics" :aria-label="t('oaRequestTitle')">
      <article><span>{{ t("total") }}</span><strong>{{ total }}</strong><FileText /></article>
      <article><span>{{ t("draftCount") }}</span><strong>{{ draftCount }}</strong><Edit3 /></article>
      <article><span>{{ t("pendingCount") }}</span><strong>{{ pendingCount }}</strong><Clock3 /></article>
      <article><span>{{ t("finishedCount") }}</span><strong>{{ finishedCount }}</strong><Check /></article>
    </section>

    <section class="toolbar oa-toolbar">
      <el-input v-model="keyword" :prefix-icon="Search" :placeholder="t('searchRequests')" clearable />
      <el-select v-model="requestType" :placeholder="t('allRequestTypes')" clearable>
        <el-option v-for="option in requestTypes" :key="option.value" :label="option.label" :value="option.value" />
      </el-select>
      <el-select v-if="!isInbox" v-model="status" :placeholder="t('allRequestStatuses')" clearable>
        <el-option :label="t('draft')" value="draft" />
        <el-option :label="t('pending')" value="pending" />
        <el-option :label="t('approved')" value="approved" />
        <el-option :label="t('rejected')" value="rejected" />
        <el-option :label="t('withdrawn')" value="withdrawn" />
        <el-option :label="t('canceled')" value="canceled" />
      </el-select>
    </section>

    <section v-if="loading" class="state-panel" aria-live="polite"><el-skeleton :rows="6" animated /><span>{{ t("loading") }}</span></section>
    <section v-else-if="error && items.length === 0" class="state-panel state-error" role="alert">
      <CircleAlert />
      <strong>{{ denied ? t("denied") : t("failed") }}</strong>
      <p>{{ denied ? t("deniedHint") : error }}</p>
      <el-button @click="loadRequests">{{ t("retry") }}</el-button>
    </section>
    <section v-else-if="items.length === 0" class="state-panel">
      <el-empty :description="isInbox ? t('noInbox') : t('noRequests')" />
      <p>{{ isInbox ? t("noInboxHint") : t("noRequestsHint") }}</p>
      <el-button v-if="!isInbox" type="primary" :icon="Plus" @click="openCreate">{{ t("newRequest") }}</el-button>
    </section>

    <section v-else class="data-region oa-data-region">
      <el-table :data="items" stripe class="desktop-table" row-key="id" @row-dblclick="openDetail">
        <el-table-column :label="t('requestTitle')" min-width="250">
          <template #default="{ row }">
            <button class="identity-link" type="button" @click="openDetail(row)">
              <strong>{{ row.title }}</strong><span>{{ requestTypeLabel(row.requestType) }} · {{ row.id }}</span>
            </button>
          </template>
        </el-table-column>
        <el-table-column :label="t('requestContent')" min-width="250"><template #default="{ row }">{{ formSummary(row) }}</template></el-table-column>
        <el-table-column :label="t('currentApprover')" min-width="170"><template #default="{ row }">{{ row.approverName }}</template></el-table-column>
        <el-table-column :label="t('createdTime')" width="175"><template #default="{ row }">{{ formatTime(row.createdAt) }}</template></el-table-column>
        <el-table-column :label="t('status')" width="120"><template #default="{ row }"><el-tag :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag></template></el-table-column>
        <el-table-column :label="t('actions')" width="190" fixed="right">
          <template #default="{ row }">
            <el-tooltip :content="t('openDetail')"><el-button :icon="Eye" text circle :aria-label="t('openDetail')" @click="openDetail(row)" /></el-tooltip>
            <el-tooltip v-if="row.status === 'draft' && !isInbox" :content="t('edit')"><el-button :icon="Edit3" text circle :aria-label="t('edit')" @click="openEdit(row)" /></el-tooltip>
            <el-tooltip v-if="row.status === 'draft' && !isInbox" :content="t('submitRequest')"><el-button :icon="Send" text circle type="primary" :aria-label="t('submitRequest')" @click="submitItem(row)" /></el-tooltip>
            <el-tooltip v-if="row.status === 'pending' && isInbox" :content="t('approveRequest')"><el-button :icon="Check" text circle type="success" :aria-label="t('approveRequest')" @click="openAction('approve', row)" /></el-tooltip>
            <el-tooltip v-if="row.status === 'pending' && isInbox" :content="t('rejectRequest')"><el-button :icon="X" text circle type="danger" :aria-label="t('rejectRequest')" @click="openAction('reject', row)" /></el-tooltip>
          </template>
        </el-table-column>
      </el-table>

      <div class="mobile-records">
        <article v-for="item in items" :key="item.id" class="record-card oa-record-card">
          <button type="button" @click="openDetail(item)"><strong>{{ item.title }}</strong><span>{{ statusLabel(item.status) }}</span></button>
          <p>{{ requestTypeLabel(item.requestType) }} · {{ formSummary(item) }}</p>
          <div><span>{{ t("currentApprover") }}</span><strong>{{ item.approverName }}</strong></div>
          <footer>
            <el-button :icon="Eye" @click="openDetail(item)">{{ t("detail") }}</el-button>
            <el-button v-if="item.status === 'draft' && !isInbox" type="primary" :icon="Edit3" @click="openEdit(item)">{{ t("edit") }}</el-button>
            <el-button v-if="item.status === 'pending' && isInbox" type="success" :icon="Check" @click="openAction('approve', item)">{{ t("approveRequest") }}</el-button>
          </footer>
        </article>
      </div>
    </section>

    <el-drawer v-model="editorOpen" :title="editing ? t('edit') : t('newRequest')" size="min(820px, 100vw)" destroy-on-close>
      <OARequestEditor v-model="form" />
      <template #footer>
        <el-button @click="editorOpen = false">{{ t("cancel") }}</el-button>
        <el-button :loading="saving" @click="saveEditor(false)">{{ t("saveDraft") }}</el-button>
        <el-button type="primary" :icon="Send" :loading="saving" @click="saveEditor(true)">{{ t("submitRequest") }}</el-button>
      </template>
    </el-drawer>

    <el-drawer v-model="detailOpen" :title="t('requestDetail')" size="min(720px, 100vw)" destroy-on-close>
      <div v-if="detail" class="oa-detail">
        <header class="oa-detail-header">
          <div><span>{{ requestTypeLabel(detail.item.requestType) }}</span><h3>{{ detail.item.title }}</h3></div>
          <el-tag :type="statusType(detail.item.status)" size="large">{{ statusLabel(detail.item.status) }}</el-tag>
        </header>
        <dl class="oa-detail-grid">
          <div><dt>{{ t("currentApprover") }}</dt><dd>{{ detail.item.approverName }} · {{ detail.item.approverId }}</dd></div>
          <div><dt>{{ t("createdTime") }}</dt><dd>{{ formatTime(detail.item.createdAt) }}</dd></div>
          <div><dt>{{ t("submittedTime") }}</dt><dd>{{ formatTime(detail.item.submittedAt) }}</dd></div>
          <div><dt>{{ t("reminderTime") }}</dt><dd>{{ formatTime(detail.item.reminderAt) }}</dd></div>
          <div class="wide"><dt>{{ t("requestDescription") }}</dt><dd>{{ detail.item.description || t("noData") }}</dd></div>
          <div class="wide"><dt>{{ t("requestContent") }}</dt><dd>{{ formSummary(detail.item) }}</dd></div>
        </dl>

        <section v-if="detail.item.status === 'pending'" class="oa-action-band">
          <el-button v-if="isInbox" type="success" :icon="Check" @click="openAction('approve')">{{ t("approveRequest") }}</el-button>
          <el-button v-if="isInbox" type="danger" :icon="X" @click="openAction('reject')">{{ t("rejectRequest") }}</el-button>
          <el-button v-if="isInbox" :icon="Forward" @click="openAction('delegate')">{{ t("delegateRequest") }}</el-button>
          <el-button v-if="!isInbox" :icon="RotateCcw" @click="openAction('withdraw')">{{ t("withdrawRequest") }}</el-button>
          <el-button v-if="!isInbox" type="danger" :icon="X" @click="openAction('cancel')">{{ t("cancelRequest") }}</el-button>
          <el-button :icon="Bell" @click="openAction('remind')">{{ t("scheduleReminder") }}</el-button>
        </section>

        <section class="oa-detail-section">
          <h4>{{ t("workflowTimeline") }}</h4>
          <p v-if="!detail.workflow?.timeline.length" class="muted-empty">{{ t("noTimeline") }}</p>
          <ol v-else class="oa-timeline">
            <li v-for="event in detail.workflow.timeline" :key="event.id">
              <span class="timeline-dot" aria-hidden="true" />
              <div><strong>{{ timelineLabel(event) }}</strong><small>{{ formatTime(event.createdAt) }}</small><p>{{ event.actor.name || event.actor.id }}<template v-if="event.target.id"> → {{ event.target.name || event.target.id }}</template></p><p v-if="event.comment">{{ event.comment }}</p></div>
            </li>
          </ol>
        </section>

        <section class="oa-detail-section">
          <div class="section-title"><h4>{{ t("attachments") }}</h4><el-button :icon="Paperclip" :loading="saving" @click="chooseAttachment">{{ t("addAttachment") }}</el-button></div>
          <input ref="attachmentInput" class="sr-only" type="file" @change="addAttachment" />
          <p v-if="!detail.item.attachments.length" class="muted-empty">{{ t("noAttachments") }}</p>
          <ul v-else class="attachment-list"><li v-for="file in detail.item.attachments" :key="file.requestKey"><Paperclip /><span>{{ file.name }}</span><small>{{ formatBytes(file.size) }}</small></li></ul>
        </section>

        <section class="oa-detail-section">
          <h4>{{ t("comments") }}</h4>
          <p v-if="!detail.item.comments.length" class="muted-empty">{{ t("noComments") }}</p>
          <ul v-else class="comment-list"><li v-for="comment in detail.item.comments" :key="comment.id"><MessageSquare /><div><strong>{{ comment.actorId }}</strong><small>{{ formatTime(comment.createdAt) }}</small><p>{{ comment.content }}</p></div></li></ul>
          <div class="comment-composer"><el-input v-model="commentText" type="textarea" :rows="2" maxlength="2000" :placeholder="t('commentPlaceholder')" /><el-button type="primary" :icon="Send" :loading="saving" :disabled="!commentText.trim()" @click="addComment">{{ t("sendComment") }}</el-button></div>
        </section>
      </div>
    </el-drawer>

    <el-dialog v-model="actionDialogOpen" :title="actionTitle" width="min(520px, calc(100vw - 28px))" destroy-on-close align-center>
      <el-form label-position="top" @submit.prevent="runAction">
        <template v-if="actionKind === 'delegate'">
          <el-form-item :label="t('delegateTargetId')" required><el-input v-model="actionForm.targetId" maxlength="128" /></el-form-item>
          <el-form-item :label="t('delegateTargetName')" required><el-input v-model="actionForm.targetName" maxlength="200" /></el-form-item>
        </template>
        <el-form-item v-if="actionKind === 'remind'" :label="t('reminderTime')" required><el-date-picker v-model="actionForm.runAt" type="datetime" value-format="YYYY-MM-DDTHH:mm" /></el-form-item>
        <el-form-item v-else :label="actionKind === 'reject' ? t('rejectionReason') : actionKind === 'withdraw' ? t('withdrawalReason') : actionKind === 'cancel' ? t('cancellationReason') : t('actionComment')" :required="['reject','withdraw','cancel'].includes(actionKind ?? '')">
          <el-input v-model="actionForm.comment" type="textarea" :rows="4" maxlength="1000" show-word-limit />
        </el-form-item>
      </el-form>
      <template #footer><el-button @click="actionKind = null">{{ t("cancel") }}</el-button><el-button type="primary" :loading="saving" @click="runAction">{{ t("confirm") }}</el-button></template>
    </el-dialog>
  </section>
</template>

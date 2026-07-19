<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { CalendarPlus, Check, Eye, FileText, Pencil, RefreshCw, Upload, X } from "lucide-vue-next";
import { ElMessage, ElMessageBox, type UploadFile } from "element-plus";
import { DataTable, DetailDrawer, PageShell, type DataTableColumn } from "../../components/Common";
import { uploadFile } from "../../files/api";
import { useButtonAccess } from "../../permissions/button";
import {
	cancelCustomerFollowUp,
	completeCustomerFollowUp,
	createCustomerFollowUp,
	listCustomerFollowUps,
	listCustomers,
	updateCustomerFollowUp,
	type CustomerFollowUp,
	type CustomerFollowUpAttachment,
	type CustomerFollowUpChannel,
	type CustomerFollowUpStatus,
	type PharmaCustomer
} from "../../pharma-oa/api";
import { useUserStore } from "../../stores/user";
import { toErrorMessage } from "../../utils/common";

type FollowUpRow = Record<string, unknown> & CustomerFollowUp & { customerText: string; scheduleText: string; channelText: string };

const access = useButtonAccess();
const userStore = useUserStore();
const loading = ref(false), saving = ref(false), completing = ref(false), actionId = ref("");
const error = ref(""), keyword = ref(""), statusFilter = ref<CustomerFollowUpStatus | "">("");
const planOpen = ref(false), completeOpen = ref(false), detailOpen = ref(false);
const items = ref<CustomerFollowUp[]>([]), customers = ref<PharmaCustomer[]>([]), selected = ref<CustomerFollowUp | null>(null);
const selectedFile = ref<File | null>(null), completionFile = ref<File | null>(null);
const editingId = ref("");
const actorId = computed(() => userStore.profile?.id?.trim() || "system");
const canRead = computed(() => access.can("pharma_oa.customer_follow_up.read"));
const canCreate = computed(() => access.can("pharma_oa.customer_follow_up.create"));
const canUpdate = computed(() => access.can("pharma_oa.customer_follow_up.update"));
const canComplete = computed(() => access.can("pharma_oa.customer_follow_up.complete"));
const canCancel = computed(() => access.can("pharma_oa.customer_follow_up.cancel"));
const form = reactive<{ customerId: string; contactName: string; channel: CustomerFollowUpChannel; scheduledAt: Date | null; nextAction: string; attachments: CustomerFollowUpAttachment[] }>({ customerId: "", contactName: "", channel: "onsite", scheduledAt: null, nextAction: "", attachments: [] });
const completion = reactive({ summary: "", nextAction: "", attachments: [] as CustomerFollowUpAttachment[] });
const rows = computed<FollowUpRow[]>(() => items.value.map((item) => ({ ...item, customerText: `${item.customerCode} · ${item.customerName}`, scheduleText: formatDateTime(item.scheduledAt), channelText: channelLabel(item.channel) })));
const columns: DataTableColumn[] = [
	{ key: "customerText", label: "Customer", minWidth: 220 },
	{ key: "contactName", label: "Contact", minWidth: 130 },
	{ key: "channelText", label: "Channel", width: 115 },
	{ key: "scheduleText", label: "Visit plan", minWidth: 170 },
	{ key: "nextAction", label: "Next action", minWidth: 190 },
	{ key: "status", label: "Status", width: 120 }
];
const plannedCount = computed(() => items.value.filter((item) => item.status === "planned").length);
const completedCount = computed(() => items.value.filter((item) => item.status === "completed").length);

onMounted(() => void refresh());

async function refresh() {
	if (!canRead.value) return;
	loading.value = true;
	error.value = "";
	try {
		const [followUps, customerItems] = await Promise.all([
			listCustomerFollowUps({ keyword: keyword.value, status: statusFilter.value }),
			listCustomers({ status: "active" })
		]);
		items.value = followUps;
		customers.value = customerItems;
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		loading.value = false;
	}
}

function openCreate() {
	editingId.value = "";
	Object.assign(form, { customerId: customers.value[0]?.id || "", contactName: "", channel: "onsite", scheduledAt: new Date(Date.now() + 86400000), nextAction: "", attachments: [] });
	selectedFile.value = null;
	planOpen.value = true;
}

function openEdit(item: CustomerFollowUp) {
	editingId.value = item.id;
	Object.assign(form, { customerId: item.customerId, contactName: item.contactName || "", channel: item.channel, scheduledAt: new Date(item.scheduledAt), nextAction: item.nextAction || "", attachments: [...item.attachments] });
	selectedFile.value = null;
	planOpen.value = true;
}

function onPlanFile(file: UploadFile) { selectedFile.value = file.raw || null; }
function onCompletionFile(file: UploadFile) { completionFile.value = file.raw || null; }

async function uploadAttachment(file: File | null, resourceId: string): Promise<CustomerFollowUpAttachment[]> {
	if (!file) return [];
	const uploaded = await uploadFile({ file, name: file.name, visibility: "private", ownerType: "user", ownerId: actorId.value, sourceModule: "pharma_oa", sourcePluginId: "pharma_oa", metadata: { resource: "customer_follow_up", resourceId } });
	return [{ fileId: uploaded.id, fileName: uploaded.name, size: uploaded.size }];
}

async function savePlan() {
	if (!form.customerId || !form.scheduledAt || (!editingId.value && !canCreate.value) || (editingId.value && !canUpdate.value)) return;
	saving.value = true;
	error.value = "";
	try {
		const uploaded = await uploadAttachment(selectedFile.value, editingId.value || "new");
		const body = { customerId: form.customerId, contactName: form.contactName.trim(), channel: form.channel, scheduledAt: form.scheduledAt.toISOString(), nextAction: form.nextAction.trim(), attachments: uploaded.length > 0 ? uploaded : form.attachments };
		const item = editingId.value ? await updateCustomerFollowUp(editingId.value, body) : await createCustomerFollowUp(body);
		replaceItem(item, !editingId.value);
		planOpen.value = false;
		ElMessage.success(editingId.value ? "Visit plan updated" : "Customer follow-up planned");
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		saving.value = false;
	}
}

function openComplete(item: CustomerFollowUp) {
	selected.value = item;
	Object.assign(completion, { summary: "", nextAction: item.nextAction || "", attachments: [...item.attachments] });
	completionFile.value = null;
	completeOpen.value = true;
}

async function completeSelected() {
	if (!selected.value || !completion.summary.trim() || !canComplete.value) return;
	completing.value = true;
	error.value = "";
	try {
		const uploaded = await uploadAttachment(completionFile.value, selected.value.id);
		const item = await completeCustomerFollowUp(selected.value.id, { summary: completion.summary.trim(), nextAction: completion.nextAction.trim(), attachments: uploaded.length > 0 ? uploaded : completion.attachments });
		replaceItem(item);
		completeOpen.value = false;
		ElMessage.success("Follow-up completed");
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		completing.value = false;
	}
}

async function cancel(item: CustomerFollowUp) {
	if (!canCancel.value || item.status !== "planned") return;
	const result = await ElMessageBox.prompt("Record why this visit plan is being cancelled.", `Cancel ${item.customerCode}`, { type: "warning", confirmButtonText: "Cancel plan", inputValidator: (value) => Boolean(value.trim()) || "Reason is required" });
	actionId.value = item.id;
	error.value = "";
	try {
		replaceItem(await cancelCustomerFollowUp(item.id, result.value));
		ElMessage.success("Visit plan cancelled");
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		actionId.value = "";
	}
}

function replaceItem(item: CustomerFollowUp, prepend = false) {
	items.value = prepend ? [item, ...items.value] : items.value.map((current) => current.id === item.id ? item : current);
	if (selected.value?.id === item.id) selected.value = item;
}
function openDetail(item: CustomerFollowUp) { selected.value = item; detailOpen.value = true; }
function statusType(status: CustomerFollowUpStatus): "success" | "warning" | "info" { return status === "completed" ? "success" : status === "planned" ? "warning" : "info"; }
function channelLabel(channel: CustomerFollowUpChannel) { return ({ onsite: "On-site", phone: "Phone", online: "Online", email: "Email" })[channel]; }
function formatDateTime(value: string) { return new Date(value).toLocaleString([], { dateStyle: "medium", timeStyle: "short" }); }
function formatBytes(size: number) { return size < 1024 ? `${size} B` : `${(size / 1024).toFixed(1)} KB`; }
</script>

<template>
	<PageShell title="Customer Follow-Ups" :loading="loading" :error="error" :forbidden="!canRead" forbidden-title="No follow-up access" forbidden-description="This page requires pharma_oa.customer_follow_up.read permission.">
		<template #actions>
			<el-input v-model="keyword" clearable placeholder="Search customer or contact" class="filter-control" @keyup.enter="refresh" />
			<el-select v-model="statusFilter" clearable placeholder="All statuses" class="filter-control" @change="refresh"><el-option label="Planned" value="planned" /><el-option label="Completed" value="completed" /><el-option label="Cancelled" value="cancelled" /></el-select>
			<el-tooltip content="Refresh"><el-button circle :icon="RefreshCw" :loading="loading" @click="refresh" /></el-tooltip>
			<el-button type="primary" :icon="CalendarPlus" :disabled="!canCreate || customers.length === 0" @click="openCreate">Plan follow-up</el-button>
		</template>
		<section class="summary-band"><div><span>Total</span><strong>{{ items.length }}</strong></div><div><span>Planned</span><strong>{{ plannedCount }}</strong></div><div><span>Completed</span><strong>{{ completedCount }}</strong></div></section>
		<DataTable :rows="rows" :columns="columns" row-key="id" :loading="loading" :error="error" empty-text="No follow-ups match the current filters.">
			<template #cell-customerText="{ row }"><strong>{{ row.customerText }}</strong></template>
			<template #cell-status="{ row }"><el-tag :type="statusType(row.status as CustomerFollowUpStatus)">{{ row.status }}</el-tag></template>
			<template #actions="{ row }"><el-tooltip content="View details"><el-button circle :icon="Eye" @click="openDetail(row as CustomerFollowUp)" /></el-tooltip><el-tooltip v-if="row.status === 'planned'" content="Edit plan"><el-button circle :icon="Pencil" :disabled="!canUpdate" @click="openEdit(row as CustomerFollowUp)" /></el-tooltip><el-tooltip v-if="row.status === 'planned'" content="Complete"><el-button circle type="success" :icon="Check" :disabled="!canComplete" @click="openComplete(row as CustomerFollowUp)" /></el-tooltip><el-tooltip v-if="row.status === 'planned'" content="Cancel"><el-button circle type="danger" :icon="X" :loading="actionId === row.id" :disabled="!canCancel" @click="cancel(row as CustomerFollowUp)" /></el-tooltip></template>
		</DataTable>

		<DetailDrawer v-model="planOpen" :title="editingId ? 'Edit visit plan' : 'Plan customer follow-up'" size="50%">
			<el-form label-position="top"><div class="form-grid"><el-form-item label="Customer" required><el-select v-model="form.customerId" filterable :disabled="Boolean(editingId)" placeholder="Select customer"><el-option v-for="customer in customers" :key="customer.id" :label="`${customer.code} · ${customer.name}`" :value="customer.id" /></el-select></el-form-item><el-form-item label="Contact"><el-input v-model="form.contactName" maxlength="80" /></el-form-item></div><div class="form-grid"><el-form-item label="Channel" required><el-segmented v-model="form.channel" :options="[{label:'On-site',value:'onsite'},{label:'Phone',value:'phone'},{label:'Online',value:'online'},{label:'Email',value:'email'}]" /></el-form-item><el-form-item label="Scheduled time" required><el-date-picker v-model="form.scheduledAt" type="datetime" /></el-form-item></div><el-form-item label="Next action"><el-input v-model="form.nextAction" type="textarea" :rows="3" maxlength="500" show-word-limit /></el-form-item><el-form-item label="Attachment"><el-upload :auto-upload="false" :limit="1" :on-change="onPlanFile" accept=".pdf,.doc,.docx,.png,.jpg,.jpeg"><el-button :icon="Upload">Select file</el-button></el-upload><small v-if="!selectedFile && form.attachments[0]" class="retained-file">Keeping {{ form.attachments[0].fileName }}</small></el-form-item></el-form>
			<template #footer><el-button :disabled="saving" @click="planOpen = false">Close</el-button><el-button type="primary" :loading="saving" :disabled="!form.customerId || !form.scheduledAt" @click="savePlan">Save plan</el-button></template>
		</DetailDrawer>

		<DetailDrawer v-model="completeOpen" title="Complete follow-up" size="48%"><el-form label-position="top"><el-form-item label="Summary" required><el-input v-model="completion.summary" type="textarea" :rows="5" maxlength="2000" show-word-limit /></el-form-item><el-form-item label="Next action"><el-input v-model="completion.nextAction" type="textarea" :rows="3" maxlength="500" /></el-form-item><el-form-item label="Evidence"><el-upload :auto-upload="false" :limit="1" :on-change="onCompletionFile" accept=".pdf,.doc,.docx,.png,.jpg,.jpeg"><el-button :icon="Upload">Select file</el-button></el-upload></el-form-item></el-form><template #footer><el-button :disabled="completing" @click="completeOpen = false">Close</el-button><el-button type="success" :icon="Check" :loading="completing" :disabled="!completion.summary.trim()" @click="completeSelected">Complete</el-button></template></DetailDrawer>

		<DetailDrawer v-model="detailOpen" :title="selected ? `${selected.customerCode} · ${selected.customerName}` : 'Follow-up'" size="48%"><template v-if="selected"><div class="detail-status"><el-tag :type="statusType(selected.status)">{{ selected.status }}</el-tag><span>{{ channelLabel(selected.channel) }}</span></div><dl class="detail-grid"><div><dt>Scheduled</dt><dd>{{ formatDateTime(selected.scheduledAt) }}</dd></div><div><dt>Contact</dt><dd>{{ selected.contactName || 'Not specified' }}</dd></div><div><dt>Owner</dt><dd>{{ selected.ownerId }}</dd></div><div><dt>Organization</dt><dd>{{ selected.organizationId }}</dd></div><div class="wide"><dt>Summary</dt><dd>{{ selected.summary || 'Not completed' }}</dd></div><div class="wide"><dt>Next action</dt><dd>{{ selected.nextAction || 'None' }}</dd></div><div v-if="selected.cancelReason" class="wide"><dt>Cancel reason</dt><dd>{{ selected.cancelReason }}</dd></div></dl><section v-if="selected.attachments.length" class="files"><h3>Attachments</h3><div v-for="file in selected.attachments" :key="file.fileId" class="file-row"><FileText :size="18" /><div><strong>{{ file.fileName }}</strong><small>{{ formatBytes(file.size) }} · {{ file.fileId }}</small></div></div></section></template><template #footer><el-button @click="detailOpen = false">Close</el-button><el-button v-if="selected?.status === 'planned'" type="success" :icon="Check" :disabled="!canComplete" @click="openComplete(selected)">Complete</el-button></template></DetailDrawer>
	</PageShell>
</template>

<style scoped>
.filter-control{width:190px}.summary-band{display:flex;gap:36px;padding:4px 0 16px;border-bottom:1px solid var(--el-border-color-lighter)}.summary-band div{display:grid;gap:3px}.summary-band span{color:var(--el-text-color-secondary);font-size:13px}.summary-band strong{font-size:22px}.form-grid,.detail-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px}.detail-status{display:flex;align-items:center;gap:10px;margin-bottom:18px}.detail-grid{margin:0}.detail-grid div{min-width:0}.detail-grid .wide{grid-column:1/-1}.detail-grid dt{color:var(--el-text-color-secondary);font-size:12px}.detail-grid dd{margin:5px 0 0;overflow-wrap:anywhere;white-space:pre-wrap}.files{margin-top:24px;border-top:1px solid var(--el-border-color-lighter)}.files h3{font-size:15px}.file-row{display:flex;align-items:center;gap:10px;padding:10px 0}.file-row div{display:grid;min-width:0}.file-row small,.retained-file{color:var(--el-text-color-secondary);overflow-wrap:anywhere}.retained-file{display:block;margin-top:8px}@media(max-width:760px){.filter-control{width:100%}.summary-band{gap:18px;justify-content:space-between}.summary-band strong{font-size:19px}.form-grid,.detail-grid{grid-template-columns:1fr}.detail-grid .wide{grid-column:auto}}
</style>

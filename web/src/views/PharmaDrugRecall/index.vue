<script setup lang="ts">
import { useI18n } from "../../i18n";

import { computed, onMounted, reactive, ref } from "vue";
import { CheckCircle2, Eye, PackageSearch, Plus, RefreshCw, Truck } from "lucide-vue-next";
import { ElMessage, ElMessageBox } from "element-plus";
import { DataTable, DetailDrawer, PageShell, type DataTableColumn } from "../../components/Common";
import { useButtonAccess } from "../../permissions/button";
import {
	completeDrugRecallTask,
	createDrugRecall,
	listDrugRecallBatches,
	listDrugRecalls,
	listQualityComplaints,
	previewDrugRecallScope,
	type DrugRecall,
	type DrugRecallBatch,
	type DrugRecallScope,
	type DrugRecallStatus,
	type DrugRecallTask,
	type QualityComplaint
} from "../../pharma-oa/api";
import { useUserStore } from "../../stores/user";
import { toErrorMessage } from "../../utils/common";

const { t, valueLabel } = useI18n();

type RecallRow = Record<string, unknown> & DrugRecall & { scopeText: string; progressText: string; initiatedText: string };

const access = useButtonAccess();
const userStore = useUserStore();
const loading = ref(false);
const saving = ref(false);
const scopeLoading = ref(false);
const actionId = ref("");
const error = ref("");
const keyword = ref("");
const statusFilter = ref<DrugRecallStatus | "">("");
const createOpen = ref(false);
const detailOpen = ref(false);
const items = ref<DrugRecall[]>([]);
const batches = ref<DrugRecallBatch[]>([]);
const complaints = ref<QualityComplaint[]>([]);
const scope = ref<DrugRecallScope[]>([]);
const selected = ref<DrugRecall | null>(null);
const form = reactive({ number: "", title: "", reason: "", batchId: "", sourceComplaintId: "" });

const canRead = computed(() => access.can("pharma_oa.drug_recall.read"));
const canCreate = computed(() => access.can("pharma_oa.drug_recall.create"));
const canComplete = computed(() => access.can("pharma_oa.drug_recall.complete"));
const actorId = computed(() => userStore.profile?.id?.trim() || "system");
const canSubmit = computed(() => canCreate.value && scope.value.length > 0 && Boolean(form.number.trim() && form.title.trim() && form.reason.trim() && form.batchId));
const rows = computed<RecallRow[]>(() => items.value.map((item) => {
	const completed = item.tasks.filter((task) => task.status === "completed").length;
	return { ...item, scopeText: `${item.productName} / ${item.batchNo}`, progressText: `${completed} / ${item.tasks.length}`, initiatedText: new Date(item.initiatedAt).toLocaleString() };
}));
const columns: DataTableColumn[] = [
	{ key: "number", label: t("pharma.drugRecall.recall"), minWidth: 145 },
	{ key: "title", label: t("pharma.drugRecall.titleColumn"), minWidth: 210 },
	{ key: "scopeText", label: t("pharma.drugRecall.productBatch"), minWidth: 230 },
	{ key: "progressText", label: t("pharma.drugRecall.tasks"), width: 100 },
	{ key: "status", label: t("pharma.drugRecall.status"), width: 110 },
	{ key: "initiatedText", label: t("pharma.drugRecall.initiated"), minWidth: 170 }
];
const activeCount = computed(() => items.value.filter((item) => item.status === "active").length);
const completedCount = computed(() => items.value.filter((item) => item.status === "completed").length);
const affectedCount = computed(() => items.value.reduce((total, item) => total + item.tasks.length, 0));
const scopeQuantity = computed(() => scope.value.reduce((total, item) => total + item.quantity, 0));

onMounted(() => void refresh());

async function refresh(): Promise<void> {
	if (!canRead.value) return;
	loading.value = true;
	error.value = "";
	try {
		const [recalls, batchItems, complaintItems] = await Promise.all([
			listDrugRecalls({ keyword: keyword.value, status: statusFilter.value }),
			listDrugRecallBatches(),
			listQualityComplaints()
		]);
		items.value = recalls;
		batches.value = batchItems;
		complaints.value = complaintItems;
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		loading.value = false;
	}
}

function openCreate(): void {
	Object.assign(form, { number: "", title: "", reason: "", batchId: "", sourceComplaintId: "" });
	scope.value = [];
	createOpen.value = true;
}

async function onBatchChange(): Promise<void> {
	scope.value = [];
	if (!form.batchId) return;
	scopeLoading.value = true;
	error.value = "";
	try {
		scope.value = await previewDrugRecallScope(form.batchId);
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		scopeLoading.value = false;
	}
}

async function save(): Promise<void> {
	if (!canSubmit.value) return;
	saving.value = true;
	error.value = "";
	try {
		const item = await createDrugRecall({ number: form.number.trim(), title: form.title.trim(), reason: form.reason.trim(), batchId: form.batchId, sourceComplaintId: form.sourceComplaintId || undefined, actorId: actorId.value });
		items.value.unshift(item);
		createOpen.value = false;
		selected.value = item;
		detailOpen.value = true;
		ElMessage.success(t("pharma.drugRecall.created"));
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		saving.value = false;
	}
}

function openDetail(item: DrugRecall): void {
	selected.value = item;
	detailOpen.value = true;
}

function replaceItem(item: DrugRecall): void {
	items.value = items.value.map((current) => current.id === item.id ? item : current);
	if (selected.value?.id === item.id) selected.value = item;
}

async function completeTask(item: DrugRecall, task: DrugRecallTask): Promise<void> {
	if (!canComplete.value || task.status === "completed") return;
	let note = "";
	try {
		const result = await ElMessageBox.prompt(t("pharma.drugRecall.recordCustomerNotificationQuarantineReturnOrDisposalEvidence"), `${t("pharma.drugRecall.completeCustomerTask")}: ${task.customerName}`, {
			type: "warning",
			confirmButtonText: t("pharma.drugRecall.completeCustomerTask"),
			inputType: "textarea",
			inputValidator: (value) => Boolean(value.trim()) || t("pharma.common.completionNoteRequired")
		});
		note = result.value;
	} catch {
		return;
	}
	actionId.value = task.id;
	error.value = "";
	try {
		replaceItem(await completeDrugRecallTask(item.id, task.id, note, actorId.value));
		ElMessage.success(t("pharma.drugRecall.taskCompleted"));
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		actionId.value = "";
	}
}

function statusType(status: DrugRecallStatus): "success" | "warning" {
	return status === "completed" ? "success" : "warning";
}

function taskType(status: DrugRecallTask["status"]): "success" | "info" {
	return status === "completed" ? "success" : "info";
}
</script>

<template>
	<PageShell :title="t('pharma.drugRecall.title')" :description="t('pharma.drugRecall.description')" :loading="loading" :error="error" :forbidden="!canRead" :forbidden-title="t('pharma.drugRecall.noDrugRecallAccess')" :forbidden-description="t('pharma.drugRecall.thisPageRequiresPharmaOaDrugRecallReadPermission')">
		<template #actions>
			<el-input v-model="keyword" clearable :placeholder="t('pharma.drugRecall.search')" class="filter-control" @keyup.enter="refresh" />
			<el-select v-model="statusFilter" clearable :placeholder="t('pharma.drugRecall.allStatuses')" class="filter-control" @change="refresh"><el-option :label="t('pharma.drugRecall.active')" value="active" /><el-option :label="t('pharma.drugRecall.completed')" value="completed" /></el-select>
			<el-tooltip :content="t('pharma.drugRecall.refresh')"><el-button :icon="RefreshCw" circle :loading="loading" @click="refresh" /></el-tooltip>
			<el-button type="primary" :icon="Plus" :disabled="!canCreate" @click="openCreate">{{ t("pharma.drugRecall.newRecall") }}</el-button>
		</template>

		<section class="summary-band">
			<div><span>{{ t("pharma.drugRecall.total") }}</span><strong>{{ items.length }}</strong></div><div><span>{{ t("pharma.drugRecall.active") }}</span><strong>{{ activeCount }}</strong></div><div><span>{{ t("pharma.drugRecall.completed") }}</span><strong>{{ completedCount }}</strong></div><div><span>{{ t("pharma.drugRecall.customerTasks") }}</span><strong>{{ affectedCount }}</strong></div>
		</section>

		<DataTable :rows="rows" :columns="columns" row-key="id" :loading="loading" :error="error" :empty-text="t('pharma.drugRecall.empty')">
			<template #cell-number="{ row }"><strong>{{ row.number }}</strong></template>
			<template #cell-status="{ row }"><el-tag :type="statusType(row.status as DrugRecallStatus)">{{ valueLabel(row.status) }}</el-tag></template>
			<template #actions="{ row }"><el-tooltip :content="t('pharma.drugRecall.viewProcessingTrace')"><el-button circle :icon="Eye" @click="openDetail(row as DrugRecall)" /></el-tooltip></template>
		</DataTable>

		<DetailDrawer v-model="createOpen" :title="t('pharma.drugRecall.newRecallTitle')" size="56%">
			<el-form label-position="top">
				<div class="form-grid"><el-form-item :label="t('pharma.drugRecall.recallNumber')" required><el-input v-model="form.number" maxlength="64" /></el-form-item><el-form-item :label="t('pharma.drugRecall.titleColumn')" required><el-input v-model="form.title" maxlength="160" /></el-form-item></div>
				<el-form-item :label="t('pharma.drugRecall.recallReason')" required><el-input v-model="form.reason" type="textarea" :rows="4" maxlength="2000" show-word-limit /></el-form-item>
				<div class="form-grid">
					<el-form-item :label="t('pharma.drugRecall.affectedBatch')" required><el-select v-model="form.batchId" filterable :placeholder="t('pharma.drugRecall.selectInventoryBatch')" @change="onBatchChange"><el-option v-for="batch in batches" :key="batch.id" :label="`${batch.batchNo} / ${batch.productId} / expires ${new Date(batch.expiresAt).toLocaleDateString()}`" :value="batch.id" /></el-select></el-form-item>
					<el-form-item :label="t('pharma.drugRecall.sourceComplaint')"><el-select v-model="form.sourceComplaintId" clearable filterable :placeholder="t('pharma.drugRecall.optional')"><el-option v-for="complaint in complaints" :key="complaint.id" :label="`${complaint.number} - ${complaint.title}`" :value="complaint.id" /></el-select></el-form-item>
				</div>
				<section class="scope-section" v-loading="scopeLoading">
					<header><div><PackageSearch :size="18" /><h3>{{ t("pharma.drugRecall.affectedCustomers") }}</h3></div><span>{{ scope.length }} {{ t("pharma.drugRecall.customerCountSeparator") }} {{ scopeQuantity }} {{ t("pharma.drugRecall.units") }}</span></header>
					<el-empty v-if="form.batchId && !scopeLoading && scope.length === 0" :image-size="64" :description="t('pharma.drugRecall.noOutboundCustomers')" />
					<div v-for="entry in scope" :key="entry.customerId" class="scope-row"><div><strong>{{ entry.customerName }}</strong><small>{{ entry.customerId }}</small></div><div class="scope-evidence"><span>{{ entry.quantity }} {{ t("pharma.drugRecall.units") }}</span><small>{{ entry.outboundIds.join(", ") }}</small></div></div>
				</section>
			</el-form>
			<template #footer><el-button :disabled="saving" @click="createOpen = false">{{ t("pharma.drugRecall.cancel") }}</el-button><el-button type="danger" :icon="Truck" :loading="saving" :disabled="!canSubmit" @click="save">{{ t("pharma.drugRecall.createRecall") }}</el-button></template>
		</DetailDrawer>

		<DetailDrawer v-model="detailOpen" :title="selected ? `${selected.number} - ${selected.title}` : t('pharma.drugRecall.title')" size="58%">
			<template v-if="selected">
				<div class="detail-status"><el-tag :type="statusType(selected.status)">{{ valueLabel(selected.status) }}</el-tag><span>{{ selected.productName }} / {{ selected.batchNo }}</span></div>
				<p class="reason">{{ selected.reason }}</p>
				<dl class="detail-grid"><div><dt>{{ t("pharma.drugRecall.batchId") }}</dt><dd>{{ selected.batchId }}</dd></div><div><dt>{{ t("pharma.drugRecall.sourceComplaint") }}</dt><dd>{{ selected.sourceComplaintId || '-' }}</dd></div><div><dt>{{ t("pharma.drugRecall.initiatedBy") }}</dt><dd>{{ selected.initiatedBy }}</dd></div><div><dt>{{ t("pharma.drugRecall.initiatedAt") }}</dt><dd>{{ new Date(selected.initiatedAt).toLocaleString() }}</dd></div></dl>
				<section class="task-section"><h3>{{ t("pharma.drugRecall.customerTrace") }}</h3><div v-for="task in selected.tasks" :key="task.id" class="task-row"><div class="task-main"><div><strong>{{ task.customerName }}</strong><el-tag size="small" :type="taskType(task.status)">{{ valueLabel(task.status) }}</el-tag></div><small>{{ task.quantity }} {{ t("pharma.drugRecall.unitSeparator") }} {{ task.outboundIds.join(", ") }}</small><p v-if="task.completionNote">{{ task.completionNote }}</p><small v-if="task.completedAt">{{ task.completedBy }} / {{ new Date(task.completedAt).toLocaleString() }}</small></div><el-tooltip v-if="task.status === 'pending'" :content="t('pharma.drugRecall.completeCustomerTask')"><el-button circle type="success" :icon="CheckCircle2" :loading="actionId === task.id" :disabled="!canComplete" @click="completeTask(selected, task)" /></el-tooltip></div></section>
			</template>
			<template #footer><el-button @click="detailOpen = false">{{ t("pharma.drugRecall.close") }}</el-button></template>
		</DetailDrawer>
	</PageShell>
</template>

<style scoped>
.filter-control{width:180px}.summary-band{display:flex;gap:32px;padding:4px 0 16px;border-bottom:1px solid var(--el-border-color-lighter)}.summary-band div{display:grid;gap:3px;min-width:82px}.summary-band span{color:var(--el-text-color-secondary);font-size:13px}.summary-band strong{font-size:22px}.form-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}.scope-section{min-height:110px;margin-top:8px;padding-top:16px;border-top:1px solid var(--el-border-color-lighter)}.scope-section header{display:flex;align-items:center;justify-content:space-between;gap:12px;margin-bottom:8px;color:var(--el-text-color-secondary)}.scope-section header div{display:flex;align-items:center;gap:8px}.scope-section h3,.task-section h3{margin:0;font-size:15px;color:var(--el-text-color-primary)}.scope-row,.task-row{display:flex;align-items:center;justify-content:space-between;gap:16px;padding:12px 0;border-bottom:1px solid var(--el-border-color-lighter)}.scope-row>div,.scope-evidence,.task-main{display:grid;gap:4px;min-width:0}.scope-row small,.task-main small{color:var(--el-text-color-secondary);overflow-wrap:anywhere}.scope-evidence{text-align:right}.detail-status{display:flex;align-items:center;gap:10px;margin-bottom:16px}.reason{margin:0 0 20px;line-height:1.65;white-space:pre-wrap}.detail-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:16px;margin:0}.detail-grid div{min-width:0}.detail-grid dt{color:var(--el-text-color-secondary);font-size:12px}.detail-grid dd{margin:5px 0 0;overflow-wrap:anywhere}.task-section{margin-top:24px;padding-top:16px;border-top:1px solid var(--el-border-color-lighter)}.task-main>div{display:flex;align-items:center;gap:8px}.task-main p{margin:4px 0;line-height:1.55;white-space:pre-wrap}@media(max-width:760px){.filter-control{width:100%}.summary-band{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px}.summary-band strong{font-size:19px}.form-grid,.detail-grid{grid-template-columns:1fr}.scope-section header,.scope-row,.task-row{align-items:flex-start}.scope-section header,.scope-row{flex-direction:column}.scope-evidence{text-align:left}.task-row .el-button{flex:0 0 auto}}
</style>

<script setup lang="ts">
import { useI18n } from "../../i18n";

import { computed, onMounted, reactive, ref } from "vue";
import { Check, Eye, FileText, Plus, RefreshCw, Upload, X } from "lucide-vue-next";
import { ElMessage, ElMessageBox, type UploadFile } from "element-plus";
import { DataTable, DetailDrawer, PageShell, type DataTableColumn } from "../../components/Common";
import { uploadFile } from "../../files/api";
import { useButtonAccess } from "../../permissions/button";
import {
	createQualityComplaint,
	listCustomers,
	listProducts,
	listQualityComplaintBatches,
	listQualityComplaints,
	rejectQualityComplaint,
	resolveQualityComplaint,
	type PharmaCustomer,
	type PharmaProduct,
	type QualityComplaint,
	type QualityComplaintBatch,
	type QualityComplaintStatus
} from "../../pharma-oa/api";
import { useUserStore } from "../../stores/user";
import { toErrorMessage } from "../../utils/common";

const { t, valueLabel } = useI18n();

type ComplaintRow = Record<string, unknown> & QualityComplaint & { relationText: string; createdText: string };

const access = useButtonAccess();
const userStore = useUserStore();
const loading = ref(false);
const saving = ref(false);
const batchLoading = ref(false);
const actionId = ref("");
const error = ref("");
const keyword = ref("");
const statusFilter = ref<QualityComplaintStatus | "">("");
const createOpen = ref(false);
const detailOpen = ref(false);
const items = ref<QualityComplaint[]>([]);
const customers = ref<PharmaCustomer[]>([]);
const products = ref<PharmaProduct[]>([]);
const batches = ref<QualityComplaintBatch[]>([]);
const selected = ref<QualityComplaint | null>(null);
const selectedFile = ref<File | null>(null);
const uploadedFileId = ref("");

const canRead = computed(() => access.can("pharma_oa.quality_complaint.read"));
const canCreate = computed(() => access.can("pharma_oa.quality_complaint.create"));
const canResolve = computed(() => access.can("pharma_oa.quality_complaint.resolve"));
const canReject = computed(() => access.can("pharma_oa.quality_complaint.reject"));
const actorId = computed(() => userStore.profile?.id?.trim() || "system");
const form = reactive({ number: "", title: "", description: "", customerId: "", productId: "", batchId: "", handlerId: "" });
const canSubmit = computed(() => canCreate.value && Boolean(form.number.trim() && form.title.trim() && form.description.trim() && form.customerId && form.productId && form.batchId && form.handlerId.trim() && selectedFile.value));
const rows = computed<ComplaintRow[]>(() => items.value.map((item) => ({
	...item,
	relationText: `${item.productName} / ${item.batchNo}`,
	createdText: item.meta?.createdAt ? new Date(item.meta.createdAt).toLocaleString() : "-"
})));
const columns: DataTableColumn[] = [
	{ key: "number", label: t("pharma.qualityComplaint.complaint"), minWidth: 150 },
	{ key: "title", label: t("pharma.qualityComplaint.titleColumn"), minWidth: 220 },
	{ key: "customerName", label: t("pharma.qualityComplaint.customer"), minWidth: 180 },
	{ key: "relationText", label: t("pharma.qualityComplaint.productBatchLabel"), minWidth: 230 },
	{ key: "handlerId", label: t("pharma.qualityComplaint.handler"), minWidth: 140 },
	{ key: "status", label: t("pharma.qualityComplaint.status"), width: 120 },
	{ key: "createdText", label: t("pharma.qualityComplaint.registeredColumn"), minWidth: 170 }
];
const pendingCount = computed(() => items.value.filter((item) => item.status === "pending").length);
const resolvedCount = computed(() => items.value.filter((item) => item.status === "resolved").length);
const rejectedCount = computed(() => items.value.filter((item) => item.status === "rejected").length);

onMounted(() => void refresh());

async function refresh(): Promise<void> {
	if (!canRead.value) return;
	loading.value = true;
	error.value = "";
	try {
		const [complaints, customerItems, productItems] = await Promise.all([
			listQualityComplaints({ keyword: keyword.value, status: statusFilter.value }),
			listCustomers({ status: "active" }),
			listProducts({ status: "active" })
		]);
		items.value = complaints;
		customers.value = customerItems;
		products.value = productItems;
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		loading.value = false;
	}
}

function openCreate(): void {
	Object.assign(form, { number: "", title: "", description: "", customerId: "", productId: "", batchId: "", handlerId: "" });
	batches.value = [];
	selectedFile.value = null;
	uploadedFileId.value = "";
	createOpen.value = true;
}

async function onProductChange(): Promise<void> {
	form.batchId = "";
	batches.value = [];
	if (!form.productId) return;
	batchLoading.value = true;
	error.value = "";
	try {
		batches.value = await listQualityComplaintBatches(form.productId);
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		batchLoading.value = false;
	}
}

function onFileSelected(file: UploadFile): void {
	selectedFile.value = file.raw || null;
	uploadedFileId.value = "";
}

async function save(): Promise<void> {
	if (!canSubmit.value || !selectedFile.value) return;
	saving.value = true;
	error.value = "";
	try {
		if (!uploadedFileId.value) {
			const uploaded = await uploadFile({
				file: selectedFile.value,
				name: selectedFile.value.name,
				visibility: "private",
				ownerType: "user",
				ownerId: actorId.value,
				sourceModule: "pharma_oa",
				sourcePluginId: "pharma_oa",
				metadata: { resource: "quality_complaint", complaintNumber: form.number.trim() }
			});
			uploadedFileId.value = uploaded.id;
		}
		const item = await createQualityComplaint({
			number: form.number.trim(),
			title: form.title.trim(),
			description: form.description.trim(),
			customerId: form.customerId,
			productId: form.productId,
			batchId: form.batchId,
			reporterId: actorId.value,
			handlerId: form.handlerId.trim(),
			attachmentIds: [uploadedFileId.value]
		});
		items.value.unshift(item);
		createOpen.value = false;
		ElMessage.success(t("pharma.qualityComplaint.registered"));
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		saving.value = false;
	}
}

function openDetail(item: QualityComplaint): void {
	selected.value = item;
	detailOpen.value = true;
}

function replaceItem(item: QualityComplaint): void {
	items.value = items.value.map((current) => current.id === item.id ? item : current);
	if (selected.value?.id === item.id) selected.value = item;
}

async function resolveComplaint(item: QualityComplaint): Promise<void> {
	if (!canResolve.value || item.status !== "pending") return;
	let conclusion = "";
	try {
		const result = await ElMessageBox.prompt(t("pharma.qualityComplaint.recordTheInvestigationResultAndCorrectiveAction"), `${t("pharma.qualityComplaint.resolve")}: ${item.number}`, {
			type: "warning",
			confirmButtonText: t("pharma.qualityComplaint.resolve"),
			inputType: "textarea",
			inputValidator: (value) => Boolean(value.trim()) || t("pharma.common.conclusionRequired")
		});
		conclusion = result.value;
	} catch {
		return;
	}
	actionId.value = item.id;
	error.value = "";
	try {
		replaceItem(await resolveQualityComplaint(item.id, conclusion, actorId.value));
		ElMessage.success(t("pharma.qualityComplaint.resolved"));
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		actionId.value = "";
	}
}

async function rejectComplaint(item: QualityComplaint): Promise<void> {
	if (!canReject.value || item.status !== "pending") return;
	let conclusion = "";
	try {
		const result = await ElMessageBox.prompt(t("pharma.qualityComplaint.recordWhyTheComplaintIsRejected"), `${t("pharma.qualityComplaint.reject")}: ${item.number}`, {
			type: "warning",
			confirmButtonText: t("pharma.qualityComplaint.reject"),
			inputType: "textarea",
			inputValidator: (value) => Boolean(value.trim()) || t("pharma.common.conclusionRequired")
		});
		conclusion = result.value;
	} catch {
		return;
	}
	actionId.value = item.id;
	error.value = "";
	try {
		replaceItem(await rejectQualityComplaint(item.id, conclusion, actorId.value));
		ElMessage.success(t("pharma.qualityComplaint.rejected"));
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		actionId.value = "";
	}
}

function statusType(status: QualityComplaintStatus): "success" | "warning" | "danger" {
	return status === "resolved" ? "success" : status === "rejected" ? "danger" : "warning";
}

function formatBytes(size: number): string {
	return size < 1024 ? `${size} B` : `${(size / 1024).toFixed(1)} KB`;
}
</script>

<template>
	<PageShell :title="t('pharma.qualityComplaint.title')" :description="t('pharma.qualityComplaint.description')" :loading="loading" :error="error" :forbidden="!canRead" :forbidden-title="t('pharma.qualityComplaint.noQualityComplaintAccess')" :forbidden-description="t('pharma.qualityComplaint.thisPageRequiresPharmaOaQualityComplaintReadPermission')">
		<template #actions>
			<el-input v-model="keyword" clearable :placeholder="t('pharma.qualityComplaint.search')" class="filter-control" @keyup.enter="refresh" />
			<el-select v-model="statusFilter" clearable :placeholder="t('pharma.qualityComplaint.allStatuses')" class="filter-control" @change="refresh">
				<el-option :label="t('pharma.qualityComplaint.pending')" value="pending" />
				<el-option :label="t('pharma.qualityComplaint.statusResolved')" value="resolved" />
				<el-option :label="t('pharma.qualityComplaint.statusRejected')" value="rejected" />
			</el-select>
			<el-tooltip :content="t('pharma.qualityComplaint.refresh')"><el-button :icon="RefreshCw" circle :loading="loading" @click="refresh" /></el-tooltip>
			<el-button type="primary" :icon="Plus" :disabled="!canCreate" @click="openCreate">{{ t("pharma.qualityComplaint.newComplaint") }}</el-button>
		</template>

		<section class="summary-band">
			<div><span>{{ t("pharma.qualityComplaint.total") }}</span><strong>{{ items.length }}</strong></div>
			<div><span>{{ t("pharma.qualityComplaint.pending") }}</span><strong>{{ pendingCount }}</strong></div>
			<div><span>{{ t("pharma.qualityComplaint.statusResolved") }}</span><strong>{{ resolvedCount }}</strong></div>
			<div><span>{{ t("pharma.qualityComplaint.statusRejected") }}</span><strong>{{ rejectedCount }}</strong></div>
		</section>

		<DataTable :rows="rows" :columns="columns" row-key="id" :loading="loading" :error="error" :empty-text="t('pharma.qualityComplaint.empty')">
			<template #cell-number="{ row }"><strong>{{ row.number }}</strong></template>
			<template #cell-status="{ row }"><el-tag :type="statusType(row.status as QualityComplaintStatus)">{{ valueLabel(row.status) }}</el-tag></template>
			<template #actions="{ row }">
				<el-tooltip :content="t('pharma.qualityComplaint.viewDetails')"><el-button circle :icon="Eye" @click="openDetail(row as QualityComplaint)" /></el-tooltip>
				<el-tooltip v-if="row.status === 'pending'" :content="t('pharma.qualityComplaint.resolve')"><el-button circle type="success" :icon="Check" :loading="actionId === row.id" :disabled="!canResolve" @click="resolveComplaint(row as QualityComplaint)" /></el-tooltip>
				<el-tooltip v-if="row.status === 'pending'" :content="t('pharma.qualityComplaint.reject')"><el-button circle type="danger" :icon="X" :loading="actionId === row.id" :disabled="!canReject" @click="rejectComplaint(row as QualityComplaint)" /></el-tooltip>
			</template>
		</DataTable>

		<DetailDrawer v-model="createOpen" :title="t('pharma.qualityComplaint.newComplaintTitle')" size="54%">
			<el-form label-position="top">
				<div class="form-grid">
					<el-form-item :label="t('pharma.qualityComplaint.complaintNumber')" required><el-input v-model="form.number" maxlength="64" /></el-form-item>
					<el-form-item :label="t('pharma.qualityComplaint.titleColumn')" required><el-input v-model="form.title" maxlength="160" /></el-form-item>
				</div>
				<el-form-item :label="t('pharma.qualityComplaint.issueDescription')" required><el-input v-model="form.description" type="textarea" :rows="4" maxlength="2000" show-word-limit /></el-form-item>
				<div class="form-grid">
					<el-form-item :label="t('pharma.qualityComplaint.customer')" required><el-select v-model="form.customerId" filterable :placeholder="t('pharma.qualityComplaint.selectCustomer')"><el-option v-for="customer in customers" :key="customer.id" :label="`${customer.code} - ${customer.name}`" :value="customer.id" /></el-select></el-form-item>
					<el-form-item :label="t('pharma.qualityComplaint.handlerUserId')" required><el-input v-model="form.handlerId" :placeholder="t('pharma.qualityComplaint.workflowAssignee')" /></el-form-item>
				</div>
				<div class="form-grid">
					<el-form-item :label="t('pharma.qualityComplaint.product')" required><el-select v-model="form.productId" filterable :placeholder="t('pharma.qualityComplaint.selectProduct')" @change="onProductChange"><el-option v-for="product in products" :key="product.id" :label="`${product.code} - ${product.name} ${product.spec}`" :value="product.id" /></el-select></el-form-item>
					<el-form-item :label="t('pharma.qualityComplaint.productBatch')" required><el-select v-model="form.batchId" filterable :loading="batchLoading" :disabled="!form.productId" :placeholder="t('pharma.qualityComplaint.selectBatch')"><el-option v-for="batch in batches" :key="batch.id" :label="`${batch.batchNo} - expires ${new Date(batch.expiresAt).toLocaleDateString()}`" :value="batch.id" /></el-select><small v-if="form.productId && !batchLoading && batches.length === 0" class="field-note">{{ t("pharma.qualityComplaint.noInventoryBatch") }}</small></el-form-item>
				</div>
				<el-form-item :label="t('pharma.qualityComplaint.evidence')" required><el-upload :auto-upload="false" :limit="1" :on-change="onFileSelected" accept=".pdf,.doc,.docx,.png,.jpg,.jpeg"><el-button :icon="Upload">{{ t("pharma.qualityComplaint.selectFile") }}</el-button></el-upload></el-form-item>
			</el-form>
			<template #footer><el-button :disabled="saving" @click="createOpen = false">{{ t("pharma.qualityComplaint.cancel") }}</el-button><el-button type="primary" :loading="saving" :disabled="!canSubmit" @click="save">{{ t("pharma.qualityComplaint.registerAndStart") }}</el-button></template>
		</DetailDrawer>

		<DetailDrawer v-model="detailOpen" :title="selected ? `${selected.number} - ${selected.title}` : t('pharma.qualityComplaint.complaint')" size="52%">
			<template v-if="selected">
				<div class="detail-status"><el-tag :type="statusType(selected.status)">{{ valueLabel(selected.status) }}</el-tag><span>{{ selected.customerName }}</span></div>
				<p class="description">{{ selected.description }}</p>
				<dl class="detail-grid">
					<div><dt>{{ t("pharma.qualityComplaint.product") }}</dt><dd>{{ selected.productName }}</dd></div><div><dt>{{ t("pharma.qualityComplaint.batch") }}</dt><dd>{{ selected.batchNo }} / {{ selected.batchId }}</dd></div>
					<div><dt>{{ t("pharma.qualityComplaint.reporter") }}</dt><dd>{{ selected.reporterId }}</dd></div><div><dt>{{ t("pharma.qualityComplaint.handler") }}</dt><dd>{{ selected.handlerId }}</dd></div>
					<div><dt>{{ t("pharma.qualityComplaint.workflow") }}</dt><dd>{{ selected.workflowInstanceId }}</dd></div><div><dt>{{ t("pharma.qualityComplaint.conclusionActor") }}</dt><dd>{{ selected.resolvedBy || selected.rejectedBy || '-' }}</dd></div>
				</dl>
				<section v-if="selected.conclusion" class="conclusion"><h3>{{ t("pharma.qualityComplaint.conclusion") }}</h3><p>{{ selected.conclusion }}</p></section>
				<section class="files"><h3>{{ t("pharma.qualityComplaint.evidenceSection") }}</h3><div v-for="file in selected.attachments" :key="file.fileId" class="file-row"><FileText :size="18" /><div><strong>{{ file.fileName }}</strong><small>{{ file.mime }} / {{ formatBytes(file.size) }} / {{ file.fileId }}</small></div></div></section>
			</template>
			<template #footer><el-button @click="detailOpen = false">{{ t("pharma.qualityComplaint.close") }}</el-button><el-button v-if="selected?.status === 'pending'" type="danger" :icon="X" :disabled="!canReject" :loading="actionId === selected.id" @click="rejectComplaint(selected)">{{ t("pharma.qualityComplaint.reject") }}</el-button><el-button v-if="selected?.status === 'pending'" type="success" :icon="Check" :disabled="!canResolve" :loading="actionId === selected.id" @click="resolveComplaint(selected)">{{ t("pharma.qualityComplaint.resolve") }}</el-button></template>
		</DetailDrawer>
	</PageShell>
</template>

<style scoped>
.filter-control{width:180px}.summary-band{display:flex;gap:32px;padding:4px 0 16px;border-bottom:1px solid var(--el-border-color-lighter)}.summary-band div{display:grid;gap:3px;min-width:72px}.summary-band span{color:var(--el-text-color-secondary);font-size:13px}.summary-band strong{font-size:22px}.form-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}.field-note{display:block;margin-top:6px;color:var(--el-text-color-secondary)}.detail-status{display:flex;align-items:center;gap:10px;margin-bottom:16px}.description{margin:0 0 20px;line-height:1.65;white-space:pre-wrap}.detail-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:16px;margin:0}.detail-grid div{min-width:0}.detail-grid dt{color:var(--el-text-color-secondary);font-size:12px}.detail-grid dd{margin:5px 0 0;overflow-wrap:anywhere}.conclusion,.files{margin-top:24px;padding-top:16px;border-top:1px solid var(--el-border-color-lighter)}.conclusion h3,.files h3{margin:0 0 10px;font-size:15px}.conclusion p{margin:0;line-height:1.6;white-space:pre-wrap}.file-row{display:flex;align-items:center;gap:10px;padding:8px 0}.file-row div{display:grid;min-width:0}.file-row small{color:var(--el-text-color-secondary);overflow-wrap:anywhere}@media(max-width:760px){.filter-control{width:100%}.summary-band{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px}.form-grid,.detail-grid{grid-template-columns:1fr}.summary-band strong{font-size:19px}}
</style>

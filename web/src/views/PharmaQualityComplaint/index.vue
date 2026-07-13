<script setup lang="ts">
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
	{ key: "number", label: "Complaint", minWidth: 150 },
	{ key: "title", label: "Title", minWidth: 220 },
	{ key: "customerName", label: "Customer", minWidth: 180 },
	{ key: "relationText", label: "Product / batch", minWidth: 230 },
	{ key: "handlerId", label: "Handler", minWidth: 140 },
	{ key: "status", label: "Status", width: 120 },
	{ key: "createdText", label: "Registered", minWidth: 170 }
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
			listCustomers({ status: "active", scope: { includeAll: true } }),
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
		ElMessage.success("Complaint registered and workflow started");
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
		const result = await ElMessageBox.prompt("Record the investigation result and corrective action.", `Resolve ${item.number}`, {
			type: "warning",
			confirmButtonText: "Resolve",
			inputType: "textarea",
			inputValidator: (value) => Boolean(value.trim()) || "Conclusion is required"
		});
		conclusion = result.value;
	} catch {
		return;
	}
	actionId.value = item.id;
	error.value = "";
	try {
		replaceItem(await resolveQualityComplaint(item.id, conclusion, actorId.value));
		ElMessage.success("Complaint resolved");
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
		const result = await ElMessageBox.prompt("Record why the complaint is rejected.", `Reject ${item.number}`, {
			type: "warning",
			confirmButtonText: "Reject",
			inputType: "textarea",
			inputValidator: (value) => Boolean(value.trim()) || "Conclusion is required"
		});
		conclusion = result.value;
	} catch {
		return;
	}
	actionId.value = item.id;
	error.value = "";
	try {
		replaceItem(await rejectQualityComplaint(item.id, conclusion, actorId.value));
		ElMessage.success("Complaint rejected");
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
	<PageShell title="Quality Complaints" description="Register product quality issues, route investigations, and retain batch evidence." :loading="loading" :error="error" :forbidden="!canRead" forbidden-title="No quality complaint access" forbidden-description="This page requires pharma_oa.quality_complaint.read permission.">
		<template #actions>
			<el-input v-model="keyword" clearable placeholder="Search complaints" class="filter-control" @keyup.enter="refresh" />
			<el-select v-model="statusFilter" clearable placeholder="All statuses" class="filter-control" @change="refresh">
				<el-option label="Pending" value="pending" />
				<el-option label="Resolved" value="resolved" />
				<el-option label="Rejected" value="rejected" />
			</el-select>
			<el-tooltip content="Refresh"><el-button :icon="RefreshCw" circle :loading="loading" @click="refresh" /></el-tooltip>
			<el-button type="primary" :icon="Plus" :disabled="!canCreate" @click="openCreate">New complaint</el-button>
		</template>

		<section class="summary-band">
			<div><span>Total</span><strong>{{ items.length }}</strong></div>
			<div><span>Pending</span><strong>{{ pendingCount }}</strong></div>
			<div><span>Resolved</span><strong>{{ resolvedCount }}</strong></div>
			<div><span>Rejected</span><strong>{{ rejectedCount }}</strong></div>
		</section>

		<DataTable :rows="rows" :columns="columns" row-key="id" :loading="loading" :error="error" empty-text="No quality complaints match the current filters.">
			<template #cell-number="{ row }"><strong>{{ row.number }}</strong></template>
			<template #cell-status="{ row }"><el-tag :type="statusType(row.status as QualityComplaintStatus)">{{ row.status }}</el-tag></template>
			<template #actions="{ row }">
				<el-tooltip content="View details"><el-button circle :icon="Eye" @click="openDetail(row as QualityComplaint)" /></el-tooltip>
				<el-tooltip v-if="row.status === 'pending'" content="Resolve"><el-button circle type="success" :icon="Check" :loading="actionId === row.id" :disabled="!canResolve" @click="resolveComplaint(row as QualityComplaint)" /></el-tooltip>
				<el-tooltip v-if="row.status === 'pending'" content="Reject"><el-button circle type="danger" :icon="X" :loading="actionId === row.id" :disabled="!canReject" @click="rejectComplaint(row as QualityComplaint)" /></el-tooltip>
			</template>
		</DataTable>

		<DetailDrawer v-model="createOpen" title="New quality complaint" size="54%">
			<el-form label-position="top">
				<div class="form-grid">
					<el-form-item label="Complaint number" required><el-input v-model="form.number" maxlength="64" /></el-form-item>
					<el-form-item label="Title" required><el-input v-model="form.title" maxlength="160" /></el-form-item>
				</div>
				<el-form-item label="Issue description" required><el-input v-model="form.description" type="textarea" :rows="4" maxlength="2000" show-word-limit /></el-form-item>
				<div class="form-grid">
					<el-form-item label="Customer" required><el-select v-model="form.customerId" filterable placeholder="Select customer"><el-option v-for="customer in customers" :key="customer.id" :label="`${customer.code} - ${customer.name}`" :value="customer.id" /></el-select></el-form-item>
					<el-form-item label="Handler user ID" required><el-input v-model="form.handlerId" placeholder="Workflow assignee" /></el-form-item>
				</div>
				<div class="form-grid">
					<el-form-item label="Product" required><el-select v-model="form.productId" filterable placeholder="Select product" @change="onProductChange"><el-option v-for="product in products" :key="product.id" :label="`${product.code} - ${product.name} ${product.spec}`" :value="product.id" /></el-select></el-form-item>
					<el-form-item label="Product batch" required><el-select v-model="form.batchId" filterable :loading="batchLoading" :disabled="!form.productId" placeholder="Select batch"><el-option v-for="batch in batches" :key="batch.id" :label="`${batch.batchNo} - expires ${new Date(batch.expiresAt).toLocaleDateString()}`" :value="batch.id" /></el-select><small v-if="form.productId && !batchLoading && batches.length === 0" class="field-note">No inventory batch is available for this product.</small></el-form-item>
				</div>
				<el-form-item label="Complaint evidence" required><el-upload :auto-upload="false" :limit="1" :on-change="onFileSelected" accept=".pdf,.doc,.docx,.png,.jpg,.jpeg"><el-button :icon="Upload">Select file</el-button></el-upload></el-form-item>
			</el-form>
			<template #footer><el-button :disabled="saving" @click="createOpen = false">Cancel</el-button><el-button type="primary" :loading="saving" :disabled="!canSubmit" @click="save">Register and start workflow</el-button></template>
		</DetailDrawer>

		<DetailDrawer v-model="detailOpen" :title="selected ? `${selected.number} - ${selected.title}` : 'Quality complaint'" size="52%">
			<template v-if="selected">
				<div class="detail-status"><el-tag :type="statusType(selected.status)">{{ selected.status }}</el-tag><span>{{ selected.customerName }}</span></div>
				<p class="description">{{ selected.description }}</p>
				<dl class="detail-grid">
					<div><dt>Product</dt><dd>{{ selected.productName }}</dd></div><div><dt>Batch</dt><dd>{{ selected.batchNo }} / {{ selected.batchId }}</dd></div>
					<div><dt>Reporter</dt><dd>{{ selected.reporterId }}</dd></div><div><dt>Handler</dt><dd>{{ selected.handlerId }}</dd></div>
					<div><dt>Workflow</dt><dd>{{ selected.workflowInstanceId }}</dd></div><div><dt>Conclusion actor</dt><dd>{{ selected.resolvedBy || selected.rejectedBy || '-' }}</dd></div>
				</dl>
				<section v-if="selected.conclusion" class="conclusion"><h3>Conclusion</h3><p>{{ selected.conclusion }}</p></section>
				<section class="files"><h3>Evidence</h3><div v-for="file in selected.attachments" :key="file.fileId" class="file-row"><FileText :size="18" /><div><strong>{{ file.fileName }}</strong><small>{{ file.mime }} / {{ formatBytes(file.size) }} / {{ file.fileId }}</small></div></div></section>
			</template>
			<template #footer><el-button @click="detailOpen = false">Close</el-button><el-button v-if="selected?.status === 'pending'" type="danger" :icon="X" :disabled="!canReject" :loading="actionId === selected.id" @click="rejectComplaint(selected)">Reject</el-button><el-button v-if="selected?.status === 'pending'" type="success" :icon="Check" :disabled="!canResolve" :loading="actionId === selected.id" @click="resolveComplaint(selected)">Resolve</el-button></template>
		</DetailDrawer>
	</PageShell>
</template>

<style scoped>
.filter-control{width:180px}.summary-band{display:flex;gap:32px;padding:4px 0 16px;border-bottom:1px solid var(--el-border-color-lighter)}.summary-band div{display:grid;gap:3px;min-width:72px}.summary-band span{color:var(--el-text-color-secondary);font-size:13px}.summary-band strong{font-size:22px}.form-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}.field-note{display:block;margin-top:6px;color:var(--el-text-color-secondary)}.detail-status{display:flex;align-items:center;gap:10px;margin-bottom:16px}.description{margin:0 0 20px;line-height:1.65;white-space:pre-wrap}.detail-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:16px;margin:0}.detail-grid div{min-width:0}.detail-grid dt{color:var(--el-text-color-secondary);font-size:12px}.detail-grid dd{margin:5px 0 0;overflow-wrap:anywhere}.conclusion,.files{margin-top:24px;padding-top:16px;border-top:1px solid var(--el-border-color-lighter)}.conclusion h3,.files h3{margin:0 0 10px;font-size:15px}.conclusion p{margin:0;line-height:1.6;white-space:pre-wrap}.file-row{display:flex;align-items:center;gap:10px;padding:8px 0}.file-row div{display:grid;min-width:0}.file-row small{color:var(--el-text-color-secondary);overflow-wrap:anywhere}@media(max-width:760px){.filter-control{width:100%}.summary-band{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px}.form-grid,.detail-grid{grid-template-columns:1fr}.summary-band strong{font-size:19px}}
</style>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { AlarmClock, ExternalLink, RefreshCw } from "lucide-vue-next";
import { ElMessage, ElMessageBox } from "element-plus";
import { useRouter } from "vue-router";

import { DataTable, PageShell, type DataTableColumn } from "../../components/Common";
import { useButtonAccess } from "../../permissions/button";
import {
	listQualifications,
	scanQualificationExpiry,
	type QualificationRecord,
	type QualificationStatus,
	type QualificationSubjectType
} from "../../pharma-oa/api";
import { useUserStore } from "../../stores/user";
import { toErrorMessage } from "../../utils/common";

type QualificationRow = Record<string, unknown> & QualificationRecord & {
	subjectText: string;
	expiryText: string;
	attachmentText: string;
	reminderText: string;
};

const access = useButtonAccess();
const router = useRouter();
const userStore = useUserStore();
const loading = ref(false);
const scanning = ref(false);
const error = ref("");
const keyword = ref("");
const subjectFilter = ref<QualificationSubjectType | "">("");
const statusFilter = ref<QualificationStatus | "">("");
const expiryDays = ref(30);
const items = ref<QualificationRecord[]>([]);

const canRead = computed(() => access.can("pharma_oa.qualification.read"));
const canScan = computed(() => access.can("pharma_oa.qualification.expiry.run"));
const actorId = computed(() => userStore.profile?.id?.trim() || "system");
const expiredCount = computed(() => items.value.filter((item) => item.status === "expired").length);
const expiringCount = computed(() => items.value.filter((item) => item.status === "expiring").length);
const reminderCount = computed(() => items.value.filter((item) => Boolean(item.reminderNotificationId)).length);
const rows = computed<QualificationRow[]>(() => items.value.map((item) => ({
	...item,
	subjectText: `${subjectLabel(item.subjectType)}: ${item.subjectName}`,
	expiryText: formatExpiry(item.expiresAt),
	attachmentText: String(item.attachmentCount),
	reminderText: item.reminderNotificationId ? "Created" : "Not created"
})));
const columns: DataTableColumn[] = [
	{ key: "subjectText", label: "Subject", minWidth: 220 },
	{ key: "subjectCode", label: "Code", minWidth: 130 },
	{ key: "qualificationName", label: "Qualification", minWidth: 190 },
	{ key: "number", label: "Number", minWidth: 150 },
	{ key: "expiryText", label: "Expires", width: 135 },
	{ key: "status", label: "Status", width: 120 },
	{ key: "attachmentText", label: "Files", width: 80, align: "center" },
	{ key: "reminderText", label: "Reminder", width: 125 }
];

onMounted(() => void refresh());

async function refresh() {
	if (!canRead.value) return;
	loading.value = true;
	error.value = "";
	try {
		items.value = await listQualifications({
			keyword: keyword.value,
			subjectType: subjectFilter.value,
			status: statusFilter.value,
			days: expiryDays.value
		});
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		loading.value = false;
	}
}

async function scanExpiry() {
	if (!canScan.value || scanning.value) return;
	await ElMessageBox.confirm(
		`Scan qualifications expired or expiring within ${expiryDays.value} days and create reminders?`,
		"Run qualification expiry scan",
		{ type: "warning", confirmButtonText: "Run scan" }
	);
	scanning.value = true;
	error.value = "";
	try {
		const result = await scanQualificationExpiry(expiryDays.value, actorId.value);
		await refresh();
		ElMessage.success(`${result.createdCount} new reminder${result.createdCount === 1 ? "" : "s"}`);
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		scanning.value = false;
	}
}

async function openSubject(item: QualificationRecord) {
	await router.push(item.targetPath);
}

function subjectLabel(subjectType: QualificationSubjectType): string {
	return subjectType.charAt(0).toUpperCase() + subjectType.slice(1);
}

function formatExpiry(value: string): string {
	if (!value || value.startsWith("0001-")) return "No expiry";
	return new Date(value).toLocaleDateString();
}

function statusType(status: QualificationStatus): "success" | "warning" | "danger" | "info" {
	if (status === "valid") return "success";
	if (status === "expiring") return "warning";
	if (status === "expired") return "danger";
	return "info";
}
</script>

<template>
	<PageShell
		title="Qualification Management"
		:loading="loading"
		:error="error"
		:forbidden="!canRead"
		forbidden-title="No qualification access"
		forbidden-description="This page requires pharma_oa.qualification.read permission."
	>
		<template #actions>
			<el-input v-model="keyword" clearable placeholder="Search qualifications" class="filter-control" @keyup.enter="refresh" />
			<el-select v-model="subjectFilter" clearable placeholder="All subjects" class="filter-control" @change="refresh">
				<el-option label="Employees" value="employee" />
				<el-option label="Suppliers" value="supplier" />
				<el-option label="Customers" value="customer" />
			</el-select>
			<el-select v-model="statusFilter" clearable placeholder="All statuses" class="filter-control" @change="refresh">
				<el-option label="Valid" value="valid" />
				<el-option label="Expiring" value="expiring" />
				<el-option label="Expired" value="expired" />
				<el-option label="Permanent" value="permanent" />
			</el-select>
			<el-input-number v-model="expiryDays" :min="1" :max="365" controls-position="right" class="days-control" aria-label="Expiry window in days" @change="refresh" />
			<el-tooltip content="Refresh"><el-button circle :icon="RefreshCw" :loading="loading" @click="refresh" /></el-tooltip>
			<el-button :icon="AlarmClock" :loading="scanning" :disabled="!canScan || loading" @click="scanExpiry">Expiry scan</el-button>
		</template>

		<section class="summary-band" aria-label="Qualification summary">
			<div><span>Total</span><strong>{{ items.length }}</strong></div>
			<div><span>Expired</span><strong>{{ expiredCount }}</strong></div>
			<div><span>Expiring</span><strong>{{ expiringCount }}</strong></div>
			<div><span>Reminders</span><strong>{{ reminderCount }}</strong></div>
		</section>

		<DataTable :rows="rows" :columns="columns" row-key="id" :loading="loading" :error="error" empty-text="No qualifications match the current filters.">
			<template #cell-subjectText="{ row }"><strong>{{ row.subjectText }}</strong></template>
			<template #cell-status="{ row }"><el-tag :type="statusType(row.status as QualificationStatus)">{{ row.status }}</el-tag></template>
			<template #cell-reminderText="{ row }"><el-tag :type="row.reminderNotificationId ? 'success' : 'info'" effect="plain">{{ row.reminderText }}</el-tag></template>
			<template #actions="{ row }">
				<el-tooltip content="Open subject ledger"><el-button circle :icon="ExternalLink" @click="openSubject(row as QualificationRecord)" /></el-tooltip>
			</template>
		</DataTable>
	</PageShell>
</template>

<style scoped>
.filter-control{width:180px}.days-control{width:126px}.summary-band{display:flex;gap:32px;padding:4px 0 16px;border-bottom:1px solid var(--el-border-color-lighter)}.summary-band div{display:grid;gap:3px;min-width:72px}.summary-band span{color:var(--el-text-color-secondary);font-size:13px}.summary-band strong{font-size:22px}@media(max-width:760px){.filter-control,.days-control{width:100%}.summary-band{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:16px}.summary-band strong{font-size:19px}}
</style>

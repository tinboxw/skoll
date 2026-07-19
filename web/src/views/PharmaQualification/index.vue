<script setup lang="ts">
import { useI18n } from "../../i18n";

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

const { t, valueLabel } = useI18n();

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
	reminderText: item.reminderNotificationId ? valueLabel("created") : t("pharma.qualification.notCreated")
})));
const columns: DataTableColumn[] = [
	{ key: "subjectText", label: t("pharma.qualification.subject"), minWidth: 220 },
	{ key: "subjectCode", label: t("pharma.qualification.code"), minWidth: 130 },
	{ key: "qualificationName", label: t("pharma.qualification.qualification"), minWidth: 190 },
	{ key: "number", label: t("pharma.qualification.number"), minWidth: 150 },
	{ key: "expiryText", label: t("pharma.qualification.expires"), width: 135 },
	{ key: "status", label: t("pharma.qualification.status"), width: 120 },
	{ key: "attachmentText", label: t("pharma.qualification.files"), width: 80, align: "center" },
	{ key: "reminderText", label: t("pharma.qualification.reminder"), width: 125 }
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
		`${t("pharma.qualification.runQualificationExpiryScan")}: ${expiryDays.value} ${valueLabel("day")}?`,
		t("pharma.qualification.runQualificationExpiryScan"),
		{ type: "warning", confirmButtonText: t("pharma.qualification.runQualificationExpiryScan") }
	);
	scanning.value = true;
	error.value = "";
	try {
		const result = await scanQualificationExpiry(expiryDays.value, actorId.value);
		await refresh();
		ElMessage.success(`${result.createdCount} ${t("pharma.common.createdReminders")}`);
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
		:title="t('pharma.qualification.title')"
		:loading="loading"
		:error="error"
		:forbidden="!canRead"
		:forbidden-title="t('pharma.qualification.noQualificationAccess')"
		:forbidden-description="t('pharma.qualification.thisPageRequiresPharmaOaQualificationReadPermission')"
	>
		<template #actions>
			<el-input v-model="keyword" clearable :placeholder="t('pharma.qualification.search')" class="filter-control" @keyup.enter="refresh" />
			<el-select v-model="subjectFilter" clearable :placeholder="t('pharma.qualification.allSubjects')" class="filter-control" @change="refresh">
				<el-option :label="t('pharma.qualification.employees')" value="employee" />
				<el-option :label="t('pharma.qualification.suppliers')" value="supplier" />
				<el-option :label="t('pharma.qualification.customers')" value="customer" />
			</el-select>
			<el-select v-model="statusFilter" clearable :placeholder="t('pharma.qualification.allStatuses')" class="filter-control" @change="refresh">
				<el-option :label="t('pharma.qualification.valid')" value="valid" />
				<el-option :label="t('pharma.qualification.expiring')" value="expiring" />
				<el-option :label="t('pharma.qualification.expired')" value="expired" />
				<el-option :label="t('pharma.qualification.permanent')" value="permanent" />
			</el-select>
			<el-input-number v-model="expiryDays" :min="1" :max="365" controls-position="right" class="days-control" :aria-label="t('pharma.qualification.expiryWindowDays')" @change="refresh" />
			<el-tooltip :content="t('pharma.qualification.refresh')"><el-button circle :icon="RefreshCw" :loading="loading" @click="refresh" /></el-tooltip>
			<el-button :icon="AlarmClock" :loading="scanning" :disabled="!canScan || loading" @click="scanExpiry">{{ t("pharma.qualification.expiryScan") }}</el-button>
		</template>

		<section class="summary-band" :aria-label="t('pharma.qualification.summary')">
			<div><span>{{ t("pharma.qualification.total") }}</span><strong>{{ items.length }}</strong></div>
			<div><span>{{ t("pharma.qualification.expired") }}</span><strong>{{ expiredCount }}</strong></div>
			<div><span>{{ t("pharma.qualification.expiring") }}</span><strong>{{ expiringCount }}</strong></div>
			<div><span>{{ t("pharma.qualification.reminders") }}</span><strong>{{ reminderCount }}</strong></div>
		</section>

		<DataTable :rows="rows" :columns="columns" row-key="id" :loading="loading" :error="error" :empty-text="t('pharma.qualification.empty')">
			<template #cell-subjectText="{ row }"><strong>{{ row.subjectText }}</strong></template>
			<template #cell-status="{ row }"><el-tag :type="statusType(row.status as QualificationStatus)">{{ valueLabel(row.status) }}</el-tag></template>
			<template #cell-reminderText="{ row }"><el-tag :type="row.reminderNotificationId ? 'success' : 'info'" effect="plain">{{ row.reminderText }}</el-tag></template>
			<template #actions="{ row }">
				<el-tooltip :content="t('pharma.qualification.openSubjectLedger')"><el-button circle :icon="ExternalLink" @click="openSubject(row as QualificationRecord)" /></el-tooltip>
			</template>
		</DataTable>
	</PageShell>
</template>

<style scoped>
.filter-control{width:180px}.days-control{width:126px}.summary-band{display:flex;gap:32px;padding:4px 0 16px;border-bottom:1px solid var(--el-border-color-lighter)}.summary-band div{display:grid;gap:3px;min-width:72px}.summary-band span{color:var(--el-text-color-secondary);font-size:13px}.summary-band strong{font-size:22px}@media(max-width:760px){.filter-control,.days-control{width:100%}.summary-band{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:16px}.summary-band strong{font-size:19px}}
</style>

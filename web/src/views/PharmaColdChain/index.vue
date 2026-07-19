<script setup lang="ts">
import { useI18n } from "../../i18n";

import { computed, onMounted, reactive, ref } from "vue";
import { Activity, AlertTriangle, Eye, Plus, RefreshCw, RotateCcw, Thermometer } from "lucide-vue-next";
import { ElMessage, ElMessageBox } from "element-plus";
import { useRoute } from "vue-router";
import { DataTable, DetailDrawer, PageShell, type DataTableColumn } from "../../components/Common";
import { useButtonAccess } from "../../permissions/button";
import {
	createColdChainRecord,
	listColdChainAnomalies,
	listColdChainContexts,
	listColdChainJobs,
	listColdChainRecords,
	retryColdChainScan,
	runColdChainScan,
	type ColdChainAnomaly,
	type ColdChainAnomalyStatus,
	type ColdChainContext,
	type ColdChainJob,
	type ColdChainRecord
} from "../../pharma-oa/api";
import { useUserStore } from "../../stores/user";
import { toErrorMessage } from "../../utils/common";

const { t, valueLabel } = useI18n();

type AnomalyRow = Record<string, unknown> & ColdChainAnomaly & { readingText: string; thresholdText: string; locationText: string; observedText: string; reasonsText: string };
type RecordRow = Record<string, unknown> & ColdChainRecord & { readingText: string; thresholdText: string; locationText: string; recordedText: string };
type JobRow = Record<string, unknown> & ColdChainJob & { resultText: string; policyText: string; completedText: string };

const access = useButtonAccess();
const userStore = useUserStore();
const route = useRoute();
const loading = ref(false);
const saving = ref(false);
const scanning = ref(false);
const actionId = ref("");
const error = ref("");
const activeTab = ref("anomalies");
const activeOnly = ref(true);
const batchFilter = ref(typeof route.query.batchId === "string" ? route.query.batchId : "");
const createOpen = ref(false);
const scanOpen = ref(false);
const detailOpen = ref(false);
const contexts = ref<ColdChainContext[]>([]);
const records = ref<ColdChainRecord[]>([]);
const anomalies = ref<ColdChainAnomaly[]>([]);
const jobs = ref<ColdChainJob[]>([]);
const selected = ref<ColdChainAnomaly | null>(null);
const form = reactive({ balanceId: "", temperatureCelsius: 5, humidityPercent: 50, source: "manual", recordedAt: new Date() });
const scanForm = reactive({ minHumidityPercent: 30, maxHumidityPercent: 70, recipientId: "" });

const canRead = computed(() => access.can("pharma_oa.cold_chain.read"));
const canCreate = computed(() => access.can("pharma_oa.cold_chain.create"));
const canRun = computed(() => access.can("pharma_oa.cold_chain.run"));
const actorId = computed(() => userStore.profile?.id?.trim() || "system");
const canSave = computed(() => canCreate.value && Boolean(form.balanceId && form.source.trim() && form.recordedAt) && form.humidityPercent >= 0 && form.humidityPercent <= 100);
const canScan = computed(() => canRun.value && Boolean(scanForm.recipientId.trim()) && scanForm.minHumidityPercent >= 0 && scanForm.maxHumidityPercent <= 100 && scanForm.minHumidityPercent <= scanForm.maxHumidityPercent);
const activeAnomalies = computed(() => anomalies.value.filter((item) => item.status === "active"));
const highRiskCount = computed(() => activeAnomalies.value.filter((item) => item.risk === "high").length);
const failedJobs = computed(() => jobs.value.filter((item) => item.status === "failed").length);
const batchOptions = computed(() => Array.from(new Map(contexts.value.map((item) => [item.batchId, item])).values()));
const selectedContext = computed(() => contexts.value.find((item) => item.balanceId === form.balanceId));

const anomalyRows = computed<AnomalyRow[]>(() => anomalies.value.map((item) => ({ ...item, readingText: `${item.temperatureCelsius.toFixed(1)} C / ${item.humidityPercent.toFixed(1)}%`, thresholdText: `${item.minCelsius}-${item.maxCelsius} C / ${item.minHumidityPercent}-${item.maxHumidityPercent}%`, locationText: `${item.warehouseId} / ${item.locationId}`, observedText: new Date(item.lastSeenAt).toLocaleString(), reasonsText: item.reasons.map(reasonLabel).join(", ") })));
const recordRows = computed<RecordRow[]>(() => records.value.map((item) => ({ ...item, readingText: `${item.temperatureCelsius.toFixed(1)} C / ${item.humidityPercent.toFixed(1)}%`, thresholdText: `${item.minCelsius}-${item.maxCelsius} C`, locationText: `${item.warehouseId} / ${item.locationId}`, recordedText: new Date(item.recordedAt).toLocaleString() })));
const jobRows = computed<JobRow[]>(() => jobs.value.map((item) => ({ ...item, resultText: `${item.matchedCount} matched / ${item.createdCount} new / ${item.resolvedCount} resolved`, policyText: `${item.policy.minHumidityPercent}-${item.policy.maxHumidityPercent}% -> ${item.policy.recipientId}`, completedText: item.completedAt ? new Date(item.completedAt).toLocaleString() : "-" })));
const anomalyColumns: DataTableColumn[] = [{ key: "batchNo", label: t("pharma.coldChain.batch"), minWidth: 130 }, { key: "readingText", label: t("pharma.coldChain.latestReading"), minWidth: 160 }, { key: "reasonsText", label: t("pharma.coldChain.riskReason"), minWidth: 210 }, { key: "locationText", label: t("pharma.coldChain.location"), minWidth: 210 }, { key: "risk", label: t("pharma.coldChain.riskColumn"), width: 100 }, { key: "status", label: t("pharma.coldChain.status"), width: 110 }, { key: "observedText", label: t("pharma.coldChain.observed"), minWidth: 170 }];
const recordColumns: DataTableColumn[] = [{ key: "batchNo", label: t("pharma.coldChain.batch"), minWidth: 130 }, { key: "readingText", label: t("pharma.coldChain.reading"), minWidth: 150 }, { key: "thresholdText", label: t("pharma.coldChain.temperatureLimit"), minWidth: 150 }, { key: "locationText", label: t("pharma.coldChain.location"), minWidth: 210 }, { key: "source", label: t("pharma.coldChain.source"), minWidth: 130 }, { key: "recordedBy", label: t("pharma.coldChain.recordedBy"), minWidth: 130 }, { key: "recordedText", label: t("pharma.coldChain.recorded"), minWidth: 170 }];
const jobColumns: DataTableColumn[] = [{ key: "id", label: t("pharma.coldChain.job"), minWidth: 145 }, { key: "status", label: t("pharma.coldChain.status"), width: 110 }, { key: "resultText", label: t("pharma.coldChain.result"), minWidth: 235 }, { key: "policyText", label: t("pharma.coldChain.humidityPolicy"), minWidth: 220 }, { key: "retryCount", label: t("pharma.coldChain.retries"), width: 90 }, { key: "completedText", label: t("pharma.coldChain.completed"), minWidth: 170 }];

onMounted(() => void refresh());

async function refresh(): Promise<void> {
	if (!canRead.value) return;
	loading.value = true;
	error.value = "";
	try {
		const [contextItems, recordItems, anomalyItems, jobItems] = await Promise.all([listColdChainContexts(), listColdChainRecords({ batchId: batchFilter.value, limit: 200 }), listColdChainAnomalies(activeOnly.value), listColdChainJobs()]);
		contexts.value = contextItems;
		records.value = recordItems;
		anomalies.value = anomalyItems;
		jobs.value = jobItems;
		const requested = typeof route.query.anomalyId === "string" ? route.query.anomalyId : "";
		if (requested) {
			const match = anomalyItems.find((item) => item.id === requested);
			if (match) openDetail(match);
		}
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		loading.value = false;
	}
}

function openCreate(): void {
	Object.assign(form, { balanceId: "", temperatureCelsius: 5, humidityPercent: 50, source: "manual", recordedAt: new Date() });
	createOpen.value = true;
}

async function saveRecord(): Promise<void> {
	if (!canSave.value) return;
	saving.value = true;
	error.value = "";
	try {
		await createColdChainRecord({ balanceId: form.balanceId, temperatureCelsius: form.temperatureCelsius, humidityPercent: form.humidityPercent, source: form.source.trim(), recordedAt: form.recordedAt.toISOString(), actorId: actorId.value });
		createOpen.value = false;
		activeTab.value = "readings";
		await refresh();
		ElMessage.success(t("pharma.coldChain.readingRecorded"));
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		saving.value = false;
	}
}

async function runScan(): Promise<void> {
	if (!canScan.value) return;
	try {
		await ElMessageBox.confirm(t("pharma.coldChain.theScanCanCreateOrResolveNotificationCenterRemindersFromc1639dda"), t("pharma.coldChain.runColdChainScan"), { type: "warning", confirmButtonText: t("pharma.coldChain.runScan") });
	} catch {
		return;
	}
	scanning.value = true;
	error.value = "";
	try {
		await runColdChainScan({ minHumidityPercent: scanForm.minHumidityPercent, maxHumidityPercent: scanForm.maxHumidityPercent, recipientId: scanForm.recipientId.trim(), actorId: actorId.value });
		scanOpen.value = false;
		activeTab.value = "anomalies";
		await refresh();
		ElMessage.success(t("pharma.coldChain.scanCompleted"));
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		scanning.value = false;
	}
}

async function retryJob(job: ColdChainJob): Promise<void> {
	if (!canRun.value || job.status !== "failed") return;
	try {
		await ElMessageBox.confirm(`${t("pharma.coldChain.retryFailedScan")}: ${job.id}?`, t("pharma.coldChain.retryFailedScan"), { type: "warning", confirmButtonText: t("pharma.coldChain.retryFailedScan") });
	} catch {
		return;
	}
	actionId.value = job.id;
	error.value = "";
	try {
		await retryColdChainScan(job.id, actorId.value);
		await refresh();
		ElMessage.success(t("pharma.coldChain.scanRetried"));
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		actionId.value = "";
	}
}

function openDetail(item: ColdChainAnomaly): void {
	selected.value = item;
	detailOpen.value = true;
}

function reasonLabel(value: string): string {
	return valueLabel(value);
}

function statusType(status: ColdChainAnomalyStatus): "success" | "warning" { return status === "resolved" ? "success" : "warning"; }
function riskType(risk: ColdChainAnomaly["risk"]): "danger" | "warning" { return risk === "high" ? "danger" : "warning"; }
function jobType(status: ColdChainJob["status"]): "success" | "danger" | "warning" | "info" { return status === "succeeded" ? "success" : status === "failed" ? "danger" : status === "running" ? "warning" : "info"; }
</script>

<template>
	<PageShell :title="t('pharma.coldChain.title')" :description="t('pharma.coldChain.description')" :loading="loading" :error="error" :forbidden="!canRead" :forbidden-title="t('pharma.coldChain.noColdChainAccess')" :forbidden-description="t('pharma.coldChain.thisPageRequiresPharmaOaColdChainReadPermission')">
		<template #actions>
			<el-select v-model="batchFilter" clearable filterable :placeholder="t('pharma.coldChain.allBatches')" class="filter-control" @change="refresh"><el-option v-for="item in batchOptions" :key="item.batchId" :label="`${item.batchNo} / ${item.productId}`" :value="item.batchId" /></el-select>
			<el-tooltip :content="t('pharma.coldChain.refresh')"><el-button circle :icon="RefreshCw" :loading="loading" @click="refresh" /></el-tooltip>
			<el-button :icon="Activity" :disabled="!canRun" @click="scanOpen = true">{{ t("pharma.coldChain.runScan") }}</el-button>
			<el-button type="primary" :icon="Plus" :disabled="!canCreate" @click="openCreate">{{ t("pharma.coldChain.newReading") }}</el-button>
		</template>

		<section class="summary-band"><div><span>{{ t("pharma.coldChain.readings") }}</span><strong>{{ records.length }}</strong></div><div><span>{{ t("pharma.coldChain.activeAnomalies") }}</span><strong>{{ activeAnomalies.length }}</strong></div><div><span>{{ t("pharma.coldChain.highRisk") }}</span><strong>{{ highRiskCount }}</strong></div><div><span>{{ t("pharma.coldChain.failedJobs") }}</span><strong>{{ failedJobs }}</strong></div></section>

		<el-tabs v-model="activeTab" class="work-tabs">
			<el-tab-pane :label="t('pharma.coldChain.anomalies')" name="anomalies">
				<div class="tab-toolbar"><el-checkbox v-model="activeOnly" :label="t('pharma.coldChain.activeOnly')" @change="refresh" /></div>
				<DataTable :rows="anomalyRows" :columns="anomalyColumns" row-key="id" :loading="loading" :error="error" :empty-text="t('pharma.coldChain.emptyAnomalies')">
					<template #cell-batchNo="{ row }"><strong>{{ row.batchNo }}</strong></template><template #cell-risk="{ row }"><el-tag :type="riskType(row.risk as ColdChainAnomaly['risk'])">{{ valueLabel(row.risk) }}</el-tag></template><template #cell-status="{ row }"><el-tag :type="statusType(row.status as ColdChainAnomalyStatus)">{{ valueLabel(row.status) }}</el-tag></template><template #actions="{ row }"><el-tooltip :content="t('pharma.coldChain.viewAnomalyTrace')"><el-button circle :icon="Eye" @click="openDetail(row as AnomalyRow)" /></el-tooltip></template>
				</DataTable>
			</el-tab-pane>
			<el-tab-pane :label="t('pharma.coldChain.readings')" name="readings"><DataTable :rows="recordRows" :columns="recordColumns" row-key="id" :loading="loading" :error="error" :empty-text="t('pharma.coldChain.emptyReadings')"><template #cell-batchNo="{ row }"><strong>{{ row.batchNo }}</strong></template></DataTable></el-tab-pane>
			<el-tab-pane :label="t('pharma.coldChain.scanJobs')" name="jobs"><DataTable :rows="jobRows" :columns="jobColumns" row-key="id" :loading="loading" :error="error" :empty-text="t('pharma.coldChain.emptyJobs')"><template #cell-status="{ row }"><el-tag :type="jobType(row.status as ColdChainJob['status'])">{{ valueLabel(row.status) }}</el-tag></template><template #actions="{ row }"><el-tooltip v-if="row.status === 'failed'" :content="t('pharma.coldChain.retryFailedScan')"><el-button circle type="warning" :icon="RotateCcw" :loading="actionId === row.id" :disabled="!canRun" @click="retryJob(row as JobRow)" /></el-tooltip></template></DataTable></el-tab-pane>
		</el-tabs>

		<DetailDrawer v-model="createOpen" :title="t('pharma.coldChain.newReadingTitle')" size="52%">
			<el-form label-position="top"><el-form-item :label="t('pharma.coldChain.inventoryBatchAndLocation')" required><el-select v-model="form.balanceId" filterable :placeholder="t('pharma.coldChain.selectEligibleStock')"><el-option v-for="item in contexts" :key="item.balanceId" :label="`${item.batchNo} / ${item.warehouseId} / ${item.locationId} / ${item.quantity} units`" :value="item.balanceId" /></el-select></el-form-item><div class="form-grid"><el-form-item :label="t('pharma.coldChain.temperature')" required><el-input-number v-model="form.temperatureCelsius" :min="-100" :max="100" :precision="1" controls-position="right" /></el-form-item><el-form-item :label="t('pharma.coldChain.humidity')" required><el-input-number v-model="form.humidityPercent" :min="0" :max="100" :precision="1" controls-position="right" /></el-form-item><el-form-item :label="t('pharma.coldChain.source')" required><el-input v-model="form.source" maxlength="120" /></el-form-item><el-form-item :label="t('pharma.coldChain.recordedAt')" required><el-date-picker v-model="form.recordedAt" type="datetime" /></el-form-item></div><div v-if="selectedContext" class="limit-band"><Thermometer :size="18" /><span>{{ t("pharma.coldChain.effectiveTemperatureLimit") }}</span><strong>{{ selectedContext.minCelsius }} {{ t("pharma.coldChain.to") }} {{ selectedContext.maxCelsius }} {{ t("pharma.coldChain.c") }}</strong></div></el-form>
			<template #footer><el-button :disabled="saving" @click="createOpen = false">{{ t("pharma.coldChain.cancel") }}</el-button><el-button type="primary" :icon="Thermometer" :loading="saving" :disabled="!canSave" @click="saveRecord">{{ t("pharma.coldChain.recordReading") }}</el-button></template>
		</DetailDrawer>

		<DetailDrawer v-model="scanOpen" :title="t('pharma.coldChain.scanTitle')" size="46%">
			<el-form label-position="top"><div class="form-grid"><el-form-item :label="t('pharma.coldChain.minimumHumidity')" required><el-input-number v-model="scanForm.minHumidityPercent" :min="0" :max="100" :precision="1" controls-position="right" /></el-form-item><el-form-item :label="t('pharma.coldChain.maximumHumidity')" required><el-input-number v-model="scanForm.maxHumidityPercent" :min="0" :max="100" :precision="1" controls-position="right" /></el-form-item></div><el-form-item :label="t('pharma.coldChain.reminderRecipient')" required><el-input v-model="scanForm.recipientId" maxlength="120" /></el-form-item></el-form>
			<template #footer><el-button :disabled="scanning" @click="scanOpen = false">{{ t("pharma.coldChain.cancel") }}</el-button><el-button type="warning" :icon="AlertTriangle" :loading="scanning" :disabled="!canScan" @click="runScan">{{ t("pharma.coldChain.runScan") }}</el-button></template>
		</DetailDrawer>

		<DetailDrawer v-model="detailOpen" :title="selected ? `${t('pharma.coldChain.viewAnomalyTrace')} / ${selected.batchNo}` : t('pharma.coldChain.viewAnomalyTrace')" size="54%">
			<template v-if="selected"><div class="detail-status"><el-tag :type="riskType(selected.risk)">{{ valueLabel(selected.risk) }} {{ t("pharma.coldChain.risk") }}</el-tag><el-tag :type="statusType(selected.status)">{{ valueLabel(selected.status) }}</el-tag></div><dl class="detail-grid"><div><dt>{{ t("pharma.coldChain.latestReading") }}</dt><dd>{{ selected.temperatureCelsius }} {{ t("pharma.coldChain.temperatureSeparator") }} {{ selected.humidityPercent }}%</dd></div><div><dt>{{ t("pharma.coldChain.allowedRange") }}</dt><dd>{{ selected.minCelsius }}-{{ selected.maxCelsius }} {{ t("pharma.coldChain.temperatureSeparator") }} {{ selected.minHumidityPercent }}-{{ selected.maxHumidityPercent }}%</dd></div><div><dt>{{ t("pharma.coldChain.productBatch") }}</dt><dd>{{ selected.productId }} / {{ selected.batchNo }}</dd></div><div><dt>{{ t("pharma.coldChain.warehouseLocation") }}</dt><dd>{{ selected.warehouseId }} / {{ selected.areaId }} / {{ selected.locationId }}</dd></div><div><dt>{{ t("pharma.coldChain.notification") }}</dt><dd>{{ selected.notificationId }}</dd></div><div><dt>{{ t("pharma.coldChain.lastObserved") }}</dt><dd>{{ new Date(selected.lastSeenAt).toLocaleString() }}</dd></div></dl><section class="reason-list"><h3>{{ t("pharma.coldChain.riskReasons") }}</h3><div v-for="reason in selected.reasons" :key="reason"><AlertTriangle :size="16" /><span>{{ reasonLabel(reason) }}</span></div></section></template>
			<template #footer><el-button @click="detailOpen = false">{{ t("pharma.coldChain.close") }}</el-button></template>
		</DetailDrawer>
	</PageShell>
</template>

<style scoped>
.filter-control{width:210px}.summary-band{display:flex;gap:32px;padding:4px 0 16px;border-bottom:1px solid var(--el-border-color-lighter)}.summary-band div{display:grid;gap:3px;min-width:100px}.summary-band span{color:var(--el-text-color-secondary);font-size:13px}.summary-band strong{font-size:22px}.work-tabs{margin-top:10px}.tab-toolbar{display:flex;justify-content:flex-end;margin-bottom:10px}.form-grid,.detail-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}.form-grid :deep(.el-input-number),.form-grid :deep(.el-date-editor){width:100%}.limit-band{display:flex;align-items:center;gap:8px;padding:12px 0;border-top:1px solid var(--el-border-color-lighter);color:var(--el-text-color-secondary)}.limit-band strong{margin-left:auto;color:var(--el-text-color-primary)}.detail-status{display:flex;gap:8px;margin-bottom:18px}.detail-grid{margin:0}.detail-grid div{min-width:0}.detail-grid dt{color:var(--el-text-color-secondary);font-size:12px}.detail-grid dd{margin:5px 0 0;overflow-wrap:anywhere}.reason-list{margin-top:22px;padding-top:16px;border-top:1px solid var(--el-border-color-lighter)}.reason-list h3{margin:0 0 10px;font-size:15px}.reason-list div{display:flex;align-items:center;gap:8px;padding:8px 0;color:var(--el-color-danger)}@media(max-width:760px){.filter-control{width:100%}.summary-band{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px}.summary-band strong{font-size:19px}.form-grid,.detail-grid{grid-template-columns:1fr}.limit-band{align-items:flex-start;flex-wrap:wrap}.limit-band strong{width:100%;margin-left:26px}}
</style>

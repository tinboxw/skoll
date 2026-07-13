<script setup lang="ts">
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
const anomalyColumns: DataTableColumn[] = [{ key: "batchNo", label: "Batch", minWidth: 130 }, { key: "readingText", label: "Latest reading", minWidth: 160 }, { key: "reasonsText", label: "Risk reason", minWidth: 210 }, { key: "locationText", label: "Location", minWidth: 210 }, { key: "risk", label: "Risk", width: 100 }, { key: "status", label: "Status", width: 110 }, { key: "observedText", label: "Observed", minWidth: 170 }];
const recordColumns: DataTableColumn[] = [{ key: "batchNo", label: "Batch", minWidth: 130 }, { key: "readingText", label: "Reading", minWidth: 150 }, { key: "thresholdText", label: "Temperature limit", minWidth: 150 }, { key: "locationText", label: "Location", minWidth: 210 }, { key: "source", label: "Source", minWidth: 130 }, { key: "recordedBy", label: "Recorded by", minWidth: 130 }, { key: "recordedText", label: "Recorded", minWidth: 170 }];
const jobColumns: DataTableColumn[] = [{ key: "id", label: "Job", minWidth: 145 }, { key: "status", label: "Status", width: 110 }, { key: "resultText", label: "Result", minWidth: 235 }, { key: "policyText", label: "Humidity policy", minWidth: 220 }, { key: "retryCount", label: "Retries", width: 90 }, { key: "completedText", label: "Completed", minWidth: 170 }];

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
		ElMessage.success("Cold-chain reading recorded");
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		saving.value = false;
	}
}

async function runScan(): Promise<void> {
	if (!canScan.value) return;
	try {
		await ElMessageBox.confirm("The scan can create or resolve notification-center reminders from the latest reading at each stock location.", "Run cold-chain scan", { type: "warning", confirmButtonText: "Run scan" });
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
		ElMessage.success("Cold-chain scan completed");
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		scanning.value = false;
	}
}

async function retryJob(job: ColdChainJob): Promise<void> {
	if (!canRun.value || job.status !== "failed") return;
	try {
		await ElMessageBox.confirm(`Retry ${job.id}? The same anomaly keys and notification identities will be reused.`, "Retry failed scan", { type: "warning", confirmButtonText: "Retry" });
	} catch {
		return;
	}
	actionId.value = job.id;
	error.value = "";
	try {
		await retryColdChainScan(job.id, actorId.value);
		await refresh();
		ElMessage.success("Cold-chain scan retried");
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
	return ({ temperature_below_min: "Temperature below minimum", temperature_above_max: "Temperature above maximum", humidity_below_min: "Humidity below minimum", humidity_above_max: "Humidity above maximum" } as Record<string, string>)[value] || value;
}

function statusType(status: ColdChainAnomalyStatus): "success" | "warning" { return status === "resolved" ? "success" : "warning"; }
function riskType(risk: ColdChainAnomaly["risk"]): "danger" | "warning" { return risk === "high" ? "danger" : "warning"; }
function jobType(status: ColdChainJob["status"]): "success" | "danger" | "warning" | "info" { return status === "succeeded" ? "success" : status === "failed" ? "danger" : status === "running" ? "warning" : "info"; }
</script>

<template>
	<PageShell title="Cold-Chain Records" description="Capture controlled readings and trace batch-level compliance risk." :loading="loading" :error="error" :forbidden="!canRead" forbidden-title="No cold-chain access" forbidden-description="This page requires pharma_oa.cold_chain.read permission.">
		<template #actions>
			<el-select v-model="batchFilter" clearable filterable placeholder="All batches" class="filter-control" @change="refresh"><el-option v-for="item in batchOptions" :key="item.batchId" :label="`${item.batchNo} / ${item.productId}`" :value="item.batchId" /></el-select>
			<el-tooltip content="Refresh"><el-button circle :icon="RefreshCw" :loading="loading" @click="refresh" /></el-tooltip>
			<el-button :icon="Activity" :disabled="!canRun" @click="scanOpen = true">Run scan</el-button>
			<el-button type="primary" :icon="Plus" :disabled="!canCreate" @click="openCreate">New reading</el-button>
		</template>

		<section class="summary-band"><div><span>Readings</span><strong>{{ records.length }}</strong></div><div><span>Active anomalies</span><strong>{{ activeAnomalies.length }}</strong></div><div><span>High risk</span><strong>{{ highRiskCount }}</strong></div><div><span>Failed jobs</span><strong>{{ failedJobs }}</strong></div></section>

		<el-tabs v-model="activeTab" class="work-tabs">
			<el-tab-pane label="Anomalies" name="anomalies">
				<div class="tab-toolbar"><el-checkbox v-model="activeOnly" label="Active only" @change="refresh" /></div>
				<DataTable :rows="anomalyRows" :columns="anomalyColumns" row-key="id" :loading="loading" :error="error" empty-text="No cold-chain anomalies match the current filters.">
					<template #cell-batchNo="{ row }"><strong>{{ row.batchNo }}</strong></template><template #cell-risk="{ row }"><el-tag :type="riskType(row.risk as ColdChainAnomaly['risk'])">{{ row.risk }}</el-tag></template><template #cell-status="{ row }"><el-tag :type="statusType(row.status as ColdChainAnomalyStatus)">{{ row.status }}</el-tag></template><template #actions="{ row }"><el-tooltip content="View anomaly trace"><el-button circle :icon="Eye" @click="openDetail(row as AnomalyRow)" /></el-tooltip></template>
				</DataTable>
			</el-tab-pane>
			<el-tab-pane label="Readings" name="readings"><DataTable :rows="recordRows" :columns="recordColumns" row-key="id" :loading="loading" :error="error" empty-text="No cold-chain readings match the current filters."><template #cell-batchNo="{ row }"><strong>{{ row.batchNo }}</strong></template></DataTable></el-tab-pane>
			<el-tab-pane label="Scan jobs" name="jobs"><DataTable :rows="jobRows" :columns="jobColumns" row-key="id" :loading="loading" :error="error" empty-text="No cold-chain scan jobs have run."><template #cell-status="{ row }"><el-tag :type="jobType(row.status as ColdChainJob['status'])">{{ row.status }}</el-tag></template><template #actions="{ row }"><el-tooltip v-if="row.status === 'failed'" content="Retry failed scan"><el-button circle type="warning" :icon="RotateCcw" :loading="actionId === row.id" :disabled="!canRun" @click="retryJob(row as JobRow)" /></el-tooltip></template></DataTable></el-tab-pane>
		</el-tabs>

		<DetailDrawer v-model="createOpen" title="New cold-chain reading" size="52%">
			<el-form label-position="top"><el-form-item label="Inventory batch and location" required><el-select v-model="form.balanceId" filterable placeholder="Select eligible stock"><el-option v-for="item in contexts" :key="item.balanceId" :label="`${item.batchNo} / ${item.warehouseId} / ${item.locationId} / ${item.quantity} units`" :value="item.balanceId" /></el-select></el-form-item><div class="form-grid"><el-form-item label="Temperature (C)" required><el-input-number v-model="form.temperatureCelsius" :min="-100" :max="100" :precision="1" controls-position="right" /></el-form-item><el-form-item label="Humidity (%)" required><el-input-number v-model="form.humidityPercent" :min="0" :max="100" :precision="1" controls-position="right" /></el-form-item><el-form-item label="Source" required><el-input v-model="form.source" maxlength="120" /></el-form-item><el-form-item label="Recorded at" required><el-date-picker v-model="form.recordedAt" type="datetime" /></el-form-item></div><div v-if="selectedContext" class="limit-band"><Thermometer :size="18" /><span>Effective temperature limit</span><strong>{{ selectedContext.minCelsius }} to {{ selectedContext.maxCelsius }} C</strong></div></el-form>
			<template #footer><el-button :disabled="saving" @click="createOpen = false">Cancel</el-button><el-button type="primary" :icon="Thermometer" :loading="saving" :disabled="!canSave" @click="saveRecord">Record reading</el-button></template>
		</DetailDrawer>

		<DetailDrawer v-model="scanOpen" title="Cold-chain anomaly scan" size="46%">
			<el-form label-position="top"><div class="form-grid"><el-form-item label="Minimum humidity (%)" required><el-input-number v-model="scanForm.minHumidityPercent" :min="0" :max="100" :precision="1" controls-position="right" /></el-form-item><el-form-item label="Maximum humidity (%)" required><el-input-number v-model="scanForm.maxHumidityPercent" :min="0" :max="100" :precision="1" controls-position="right" /></el-form-item></div><el-form-item label="Reminder recipient" required><el-input v-model="scanForm.recipientId" maxlength="120" /></el-form-item></el-form>
			<template #footer><el-button :disabled="scanning" @click="scanOpen = false">Cancel</el-button><el-button type="warning" :icon="AlertTriangle" :loading="scanning" :disabled="!canScan" @click="runScan">Run scan</el-button></template>
		</DetailDrawer>

		<DetailDrawer v-model="detailOpen" :title="selected ? `Cold-chain anomaly / ${selected.batchNo}` : 'Cold-chain anomaly'" size="54%">
			<template v-if="selected"><div class="detail-status"><el-tag :type="riskType(selected.risk)">{{ selected.risk }} risk</el-tag><el-tag :type="statusType(selected.status)">{{ selected.status }}</el-tag></div><dl class="detail-grid"><div><dt>Latest reading</dt><dd>{{ selected.temperatureCelsius }} C / {{ selected.humidityPercent }}%</dd></div><div><dt>Allowed range</dt><dd>{{ selected.minCelsius }}-{{ selected.maxCelsius }} C / {{ selected.minHumidityPercent }}-{{ selected.maxHumidityPercent }}%</dd></div><div><dt>Product / batch</dt><dd>{{ selected.productId }} / {{ selected.batchNo }}</dd></div><div><dt>Warehouse / location</dt><dd>{{ selected.warehouseId }} / {{ selected.areaId }} / {{ selected.locationId }}</dd></div><div><dt>Notification</dt><dd>{{ selected.notificationId }}</dd></div><div><dt>Last observed</dt><dd>{{ new Date(selected.lastSeenAt).toLocaleString() }}</dd></div></dl><section class="reason-list"><h3>Risk reasons</h3><div v-for="reason in selected.reasons" :key="reason"><AlertTriangle :size="16" /><span>{{ reasonLabel(reason) }}</span></div></section></template>
			<template #footer><el-button @click="detailOpen = false">Close</el-button></template>
		</DetailDrawer>
	</PageShell>
</template>

<style scoped>
.filter-control{width:210px}.summary-band{display:flex;gap:32px;padding:4px 0 16px;border-bottom:1px solid var(--el-border-color-lighter)}.summary-band div{display:grid;gap:3px;min-width:100px}.summary-band span{color:var(--el-text-color-secondary);font-size:13px}.summary-band strong{font-size:22px}.work-tabs{margin-top:10px}.tab-toolbar{display:flex;justify-content:flex-end;margin-bottom:10px}.form-grid,.detail-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}.form-grid :deep(.el-input-number),.form-grid :deep(.el-date-editor){width:100%}.limit-band{display:flex;align-items:center;gap:8px;padding:12px 0;border-top:1px solid var(--el-border-color-lighter);color:var(--el-text-color-secondary)}.limit-band strong{margin-left:auto;color:var(--el-text-color-primary)}.detail-status{display:flex;gap:8px;margin-bottom:18px}.detail-grid{margin:0}.detail-grid div{min-width:0}.detail-grid dt{color:var(--el-text-color-secondary);font-size:12px}.detail-grid dd{margin:5px 0 0;overflow-wrap:anywhere}.reason-list{margin-top:22px;padding-top:16px;border-top:1px solid var(--el-border-color-lighter)}.reason-list h3{margin:0 0 10px;font-size:15px}.reason-list div{display:flex;align-items:center;gap:8px;padding:8px 0;color:var(--el-color-danger)}@media(max-width:760px){.filter-control{width:100%}.summary-band{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px}.summary-band strong{font-size:19px}.form-grid,.detail-grid{grid-template-columns:1fr}.limit-band{align-items:flex-start;flex-wrap:wrap}.limit-band strong{width:100%;margin-left:26px}}
</style>

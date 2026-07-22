<script setup lang="ts">
import { useI18n } from "../../i18n";

import { computed, onMounted, ref } from "vue";
import { AlertTriangle, ArrowUpRight, BadgeCheck, ChartNoAxesCombined, ClipboardCheck, RefreshCw, UsersRound } from "lucide-vue-next";
import { useRouter } from "vue-router";
import { PageShell, StateBlock } from "../../components/Common";
import { useButtonAccess } from "../../permissions/button";
import { getBusinessMetrics, type BusinessMetricsBucket, type BusinessMetricsSnapshot } from "../../pharma-oa/api";
import { toErrorMessage } from "../../utils/common";

const { t, valueLabel } = useI18n();

const router = useRouter();
const access = useButtonAccess();
const loading = ref(false);
const error = ref("");
const snapshot = ref<BusinessMetricsSnapshot | null>(null);
const bucket = ref<BusinessMetricsBucket>("day");
const qualificationDays = ref(30);
const now = new Date();
const start = new Date(now);
start.setDate(start.getDate() - 29);
const dateRange = ref<[Date, Date]>([start, now]);
let requestSequence = 0;

const canRead = computed(() => access.can("pharma_oa.business_metrics.read"));
const isEmpty = computed(() => {
	const item = snapshot.value;
	return !!item && item.stockAlerts.active === 0 && item.qualifications.expired + item.qualifications.expiring === 0 && item.approvalEfficiency.total === 0 && item.customerFollowUps.total === 0 && item.salesTrend.orderCount === 0;
});
const qualificationTotal = computed(() => (snapshot.value?.qualifications.expired ?? 0) + (snapshot.value?.qualifications.expiring ?? 0));
const maxSalesAmount = computed(() => Math.max(0, ...(snapshot.value?.salesTrend.series ?? []).map((item) => item.amountCents)));
const metricItems = computed(() => [
	{ key: "alerts", label: t("pharma.dashboard.activeStockAlerts"), value: String(snapshot.value?.stockAlerts.active ?? 0), detail: `${snapshot.value?.stockAlerts.lowStock ?? 0} ${t("pharma.dashboard.lowStock")} · ${snapshot.value?.stockAlerts.nearExpiry ?? 0} ${t("pharma.dashboard.nearExpiry")}`, icon: AlertTriangle, tone: "danger", path: "/skoll/pharma-oa/warehouses" },
	{ key: "qualifications", label: t("pharma.dashboard.qualificationRisk"), value: String(qualificationTotal.value), detail: `${snapshot.value?.qualifications.expired ?? 0} ${t("pharma.qualification.expired")} · ${snapshot.value?.qualifications.expiring ?? 0} ${t("pharma.qualification.expiring")}`, icon: BadgeCheck, tone: "warning", path: "/skoll/pharma-oa/qualifications" },
	{ key: "approvals", label: t("pharma.dashboard.approvalCompletion"), value: percent(snapshot.value?.approvalEfficiency.completionRate ?? 0), detail: `${snapshot.value?.approvalEfficiency.averageCompletionHours ?? 0} ${t("pharma.dashboard.averageHours")} · ${snapshot.value?.approvalEfficiency.pending ?? 0} ${t("pharma.dashboard.pending")}`, icon: ClipboardCheck, tone: "success", path: "/skoll/workflow" },
	{ key: "followups", label: t("pharma.dashboard.followUpCompletion"), value: percent(snapshot.value?.customerFollowUps.completionRate ?? 0), detail: `${snapshot.value?.customerFollowUps.overdue ?? 0} ${t("pharma.dashboard.overdue")} · ${snapshot.value?.customerFollowUps.planned ?? 0} ${t("pharma.dashboard.planned")}`, icon: UsersRound, tone: "primary", path: "/skoll/pharma-oa/customer-follow-ups" }
]);

onMounted(() => void refresh());

async function refresh(): Promise<void> {
	if (!canRead.value || loading.value) return;
	const sequence = ++requestSequence;
	loading.value = true;
	error.value = "";
	try {
		const [from, to] = dateRange.value;
		const result = await getBusinessMetrics({ from: startOfDay(from).toISOString(), to: endOfDay(to).toISOString(), bucket: bucket.value, qualificationDays: qualificationDays.value });
		if (sequence === requestSequence) snapshot.value = result;
	} catch (cause) {
		if (sequence === requestSequence) error.value = toErrorMessage(cause);
	} finally {
		if (sequence === requestSequence) loading.value = false;
	}
}

function startOfDay(value: Date): Date { const next = new Date(value); next.setHours(0, 0, 0, 0); return next; }
function endOfDay(value: Date): Date { const next = new Date(value); next.setHours(23, 59, 59, 999); return next; }
function percent(value: number): string { return `${value.toFixed(value % 1 === 0 ? 0 : 1)}%`; }
function money(value: number): string { return new Intl.NumberFormat(undefined, { style: "currency", currency: "CNY", maximumFractionDigits: 0 }).format(value / 100); }
function pointLabel(value: string): string {
	const date = new Date(value);
	return bucket.value === "month" ? date.toLocaleDateString(undefined, { month: "short" }) : date.toLocaleDateString(undefined, { month: "numeric", day: "numeric" });
}
function barHeight(value: number): string { return maxSalesAmount.value <= 0 ? "0%" : `${Math.max(4, Math.round(value / maxSalesAmount.value * 100))}%`; }
function rangeLabel(): string {
	const item = snapshot.value;
	if (!item) return "";
	return `${new Date(item.window.from).toLocaleDateString()} - ${new Date(item.window.to).toLocaleDateString()}`;
}
function open(path: string): void { void router.push(path); }
</script>

<template>
	<PageShell :title="t('pharma.dashboard.pharmaOaDashboard')" :description="t('pharma.dashboard.operationsComplianceApprovalsCustomerActivityAndSalesInO8e02a8e6')" :loading="loading" :error="error" :forbidden="!canRead" :forbidden-title="t('pharma.dashboard.noDashboardAccess')" :forbidden-description="t('pharma.dashboard.thisPageRequiresPharmaOaBusinessMetricsReadPermission')">
		<template #meta><span v-if="snapshot" class="updated">{{ t("pharma.dashboard.updated") }} {{ new Date(snapshot.generatedAt).toLocaleTimeString() }}</span></template>
		<template #actions><el-tooltip :content="t('pharma.dashboard.refresh')"><el-button circle :aria-label="t('pharma.dashboard.refreshDashboard')" :icon="RefreshCw" :loading="loading" @click="refresh" /></el-tooltip></template>
		<template #stateActions><el-button :icon="RefreshCw" @click="refresh">{{ t("pharma.dashboard.retry") }}</el-button></template>

		<section class="filters" :aria-label="t('pharma.dashboard.dashboardWindow')">
			<el-date-picker v-model="dateRange" type="daterange" :aria-label="t('pharma.dashboard.dashboardWindow')" :range-separator="t('pharma.dashboard.toSeparator')" :start-placeholder="t('pharma.dashboard.from')" :end-placeholder="t('pharma.dashboard.to')" :clearable="false" :disabled="loading" />
			<el-segmented v-model="bucket" :options="[{ label: valueLabel('day'), value: 'day' }, { label: valueLabel('week'), value: 'week' }, { label: valueLabel('month'), value: 'month' }]" :disabled="loading" />
			<label><span>{{ t("pharma.dashboard.qualificationHorizon") }}</span><el-input-number v-model="qualificationDays" :min="1" :max="365" :step="5" :disabled="loading" /></label>
			<el-button type="primary" :icon="ChartNoAxesCombined" :loading="loading" @click="refresh">{{ t("pharma.dashboard.apply") }}</el-button>
		</section>

		<StateBlock v-if="isEmpty" :description="t('pharma.dashboard.noBusinessActivityOrActiveOperationalRiskInThisWindow')" />
		<template v-else-if="snapshot">
			<section class="metric-grid" :aria-label="t('pharma.dashboard.keyBusinessMetrics')">
				<article v-for="item in metricItems" :key="item.key" class="metric" :class="`metric--${item.tone}`">
					<div class="metric__heading"><component :is="item.icon" :size="19" /><span>{{ item.label }}</span><el-tooltip :content="`${t('pharma.dashboard.openMetric')}${item.label}`"><el-button text circle :aria-label="`${t('pharma.dashboard.openMetric')}${item.label}`" :icon="ArrowUpRight" @click="open(item.path)" /></el-tooltip></div>
					<strong>{{ item.value }}</strong><p>{{ item.detail }}</p>
				</article>
			</section>

			<section class="sales-band">
				<header><div><span>{{ t("pharma.dashboard.salesTrend") }}</span><strong>{{ money(snapshot.salesTrend.amountCents) }}</strong><p>{{ snapshot.salesTrend.orderCount }} {{ t("pharma.dashboard.orders") }} {{ rangeLabel() }}</p></div><el-tooltip :content="t('pharma.dashboard.openSales')"><el-button circle :aria-label="t('pharma.dashboard.openSales')" :icon="ArrowUpRight" @click="open('/skoll/pharma-oa/sales')" /></el-tooltip></header>
				<div v-if="snapshot.salesTrend.orderCount" class="chart" role="img" :aria-label="`${t('pharma.dashboard.salesTrendAria')} ${snapshot.salesTrend.orderCount}`">
					<div v-for="point in snapshot.salesTrend.series" :key="point.startedAt" class="bar-column"><div class="bar-value">{{ point.orderCount || '' }}</div><div class="bar-track"><div class="bar" :style="{ height: barHeight(point.amountCents) }" :title="`${pointLabel(point.startedAt)}: ${money(point.amountCents)}`" /></div><span>{{ pointLabel(point.startedAt) }}</span></div>
				</div>
				<StateBlock v-else :description="t('pharma.dashboard.noSalesOrdersInThisWindow')" />
			</section>

			<section class="detail-grid">
				<div><h3>{{ t("pharma.dashboard.approvalFlow") }}</h3><dl><div><dt>{{ t("pharma.dashboard.approved") }}</dt><dd>{{ snapshot.approvalEfficiency.approved }}</dd></div><div><dt>{{ t("pharma.dashboard.rejected") }}</dt><dd>{{ snapshot.approvalEfficiency.rejected }}</dd></div><div><dt>{{ t("pharma.dashboard.pending") }}</dt><dd>{{ snapshot.approvalEfficiency.pending }}</dd></div></dl></div>
				<div><h3>{{ t("pharma.dashboard.qualificationSubjects") }}</h3><dl><div><dt>{{ t("pharma.dashboard.employees") }}</dt><dd>{{ snapshot.qualifications.employee }}</dd></div><div><dt>{{ t("pharma.dashboard.suppliers") }}</dt><dd>{{ snapshot.qualifications.supplier }}</dd></div><div><dt>{{ t("pharma.dashboard.customers") }}</dt><dd>{{ snapshot.qualifications.customer }}</dd></div></dl></div>
				<div><h3>{{ t("pharma.dashboard.stockAlertMix") }}</h3><dl><div><dt>{{ t("pharma.dashboard.lowStock") }}</dt><dd>{{ snapshot.stockAlerts.lowStock }}</dd></div><div><dt>{{ t("pharma.dashboard.overStock") }}</dt><dd>{{ snapshot.stockAlerts.overStock }}</dd></div><div><dt>{{ t("pharma.dashboard.nearExpiry") }}</dt><dd>{{ snapshot.stockAlerts.nearExpiry }}</dd></div></dl></div>
			</section>
		</template>
	</PageShell>
</template>

<style scoped>
.updated{color:var(--el-text-color-secondary);font-size:12px;white-space:nowrap}.filters{display:grid;grid-template-columns:minmax(270px,1.4fr) auto minmax(200px,.8fr) auto;gap:10px;align-items:end;padding:12px;border:1px solid var(--el-border-color-lighter);border-radius:6px;background:var(--el-bg-color)}.filters :deep(.el-date-editor){width:100%}.filters label{display:grid;gap:5px;min-width:0}.filters label span{color:var(--el-text-color-secondary);font-size:12px}.filters :deep(.el-input-number){width:100%}.metric-grid{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));border:1px solid var(--el-border-color-lighter);border-radius:6px;overflow:hidden}.metric{min-width:0;padding:14px;border-right:1px solid var(--el-border-color-lighter);background:var(--el-bg-color)}.metric:last-child{border-right:0}.metric__heading{display:grid;grid-template-columns:auto minmax(0,1fr) auto;align-items:center;gap:8px;color:var(--el-text-color-secondary);font-size:13px}.metric__heading span{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.metric>strong{display:block;margin-top:14px;font-size:26px;line-height:1}.metric p{margin:7px 0 0;color:var(--el-text-color-secondary);font-size:12px;overflow-wrap:anywhere}.metric--danger .metric__heading>svg{color:var(--el-color-danger)}.metric--warning .metric__heading>svg{color:var(--el-color-warning)}.metric--success .metric__heading>svg{color:var(--el-color-success)}.metric--primary .metric__heading>svg{color:var(--el-color-primary)}.sales-band{display:grid;grid-template-columns:minmax(190px,240px) minmax(0,1fr);gap:20px;padding:16px 0;border-top:1px solid var(--el-border-color-lighter);border-bottom:1px solid var(--el-border-color-lighter)}.sales-band>header{display:flex;justify-content:space-between;gap:12px}.sales-band header span{color:var(--el-text-color-secondary);font-size:13px}.sales-band header strong{display:block;margin-top:10px;font-size:25px}.sales-band header p{margin:7px 0 0;color:var(--el-text-color-secondary);font-size:12px}.chart{display:grid;grid-auto-flow:column;grid-auto-columns:minmax(28px,1fr);align-items:end;gap:6px;height:210px;min-width:0;overflow-x:auto;padding:4px 2px}.bar-column{display:grid;grid-template-rows:18px 150px 24px;gap:4px;align-items:end;min-width:28px;text-align:center}.bar-value{color:var(--el-text-color-secondary);font-size:11px}.bar-track{height:150px;display:flex;align-items:flex-end;background:var(--el-fill-color-lighter);border-radius:3px 3px 0 0;overflow:hidden}.bar{width:100%;background:var(--el-color-primary);border-radius:3px 3px 0 0;transition:height .2s ease}.bar-column>span{font-size:10px;color:var(--el-text-color-secondary);white-space:nowrap}.detail-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:0;border-bottom:1px solid var(--el-border-color-lighter)}.detail-grid>div{padding:4px 18px 14px;border-right:1px solid var(--el-border-color-lighter)}.detail-grid>div:first-child{padding-left:0}.detail-grid>div:last-child{border-right:0}.detail-grid h3{margin:0 0 12px;font-size:14px}.detail-grid dl{display:grid;gap:9px;margin:0}.detail-grid dl div{display:flex;justify-content:space-between;gap:12px}.detail-grid dt{color:var(--el-text-color-secondary)}.detail-grid dd{margin:0;font-weight:700}@media(max-width:1050px){.filters{grid-template-columns:1fr 1fr}.metric-grid{grid-template-columns:repeat(2,minmax(0,1fr))}.metric:nth-child(2){border-right:0}.metric:nth-child(-n+2){border-bottom:1px solid var(--el-border-color-lighter)}}@media(max-width:760px){.filters{grid-template-columns:1fr}.filters :deep(.el-segmented){width:100%}.metric-grid{grid-template-columns:1fr}.metric{border-right:0;border-bottom:1px solid var(--el-border-color-lighter)}.metric:last-child{border-bottom:0}.sales-band{grid-template-columns:1fr}.chart{height:190px}.bar-column{grid-template-rows:18px 130px 24px}.bar-track{height:130px}.detail-grid{grid-template-columns:1fr}.detail-grid>div,.detail-grid>div:first-child{padding:12px 0;border-right:0;border-bottom:1px solid var(--el-border-color-lighter)}.detail-grid>div:last-child{border-bottom:0}}
</style>

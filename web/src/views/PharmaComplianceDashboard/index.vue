<script setup lang="ts">
import { useI18n } from "../../i18n";

import { computed, onMounted, ref } from "vue";
import { Download, ExternalLink, Eye, RefreshCw, Search, ShieldAlert, X } from "lucide-vue-next";
import { ElMessage } from "element-plus";
import { useRoute, useRouter } from "vue-router";
import { DataTable, DetailDrawer, PageShell, type DataTableColumn } from "../../components/Common";
import { downloadBlob } from "../../composables/useDownloadBlob";
import { useButtonAccess } from "../../permissions/button";
import {
	exportComplianceDashboard,
	getComplianceDashboard,
	type ComplianceDashboardQuery,
	type ComplianceDashboardSnapshot,
	type ComplianceRisk,
	type ComplianceRiskItem,
	type ComplianceSource
} from "../../pharma-oa/api";
import { toErrorMessage } from "../../utils/common";

const { t, valueLabel } = useI18n();

type RiskRow = Record<string, unknown> & ComplianceRiskItem & {
	sourceText: string;
	observedText: string;
	actionText: string;
};

const route = useRoute();
const router = useRouter();
const access = useButtonAccess();
const loading = ref(false);
const exporting = ref(false);
const error = ref("");
const detailOpen = ref(false);
const selected = ref<ComplianceRiskItem | null>(null);
const snapshot = ref<ComplianceDashboardSnapshot | null>(null);
const keyword = ref(typeof route.query.keyword === "string" ? route.query.keyword : "");
const risk = ref<ComplianceRisk | "">(route.query.risk === "high" || route.query.risk === "medium" ? route.query.risk : "");
const source = ref<ComplianceSource | "">(isComplianceSource(route.query.source) ? route.query.source : "");

const canRead = computed(() => access.can("pharma_oa.compliance_dashboard.read"));
const canExport = computed(() => access.can("pharma_oa.compliance_dashboard.export"));
const summary = computed(() => snapshot.value?.summary ?? { total: 0, high: 0, medium: 0, qualifications: 0, complaints: 0, recalls: 0, coldChain: 0 });
const sourceStats = computed(() => [
	{ key: "qualification", label: t("pharma.complianceDashboard.qualifications"), value: summary.value.qualifications },
	{ key: "quality_complaint", label: t("pharma.complianceDashboard.complaints"), value: summary.value.complaints },
	{ key: "drug_recall", label: t("pharma.complianceDashboard.recalls"), value: summary.value.recalls },
	{ key: "cold_chain", label: t("pharma.complianceDashboard.coldChain"), value: summary.value.coldChain }
]);
const rows = computed<RiskRow[]>(() => (snapshot.value?.items ?? []).map((item) => ({
	...item,
	sourceText: sourceLabel(item.source),
	observedText: new Date(item.observedAt).toLocaleString(),
	actionText: item.totalActions > 1 ? `${item.pendingActions} / ${item.totalActions}` : String(item.pendingActions)
})));
const columns: DataTableColumn[] = [
	{ key: "risk", label: t("pharma.complianceDashboard.riskColumn"), width: 90 },
	{ key: "sourceText", label: t("pharma.complianceDashboard.source"), minWidth: 145 },
	{ key: "reference", label: t("pharma.complianceDashboard.reference"), minWidth: 145 },
	{ key: "title", label: t("pharma.complianceDashboard.issue"), minWidth: 210 },
	{ key: "subject", label: t("pharma.complianceDashboard.subject"), minWidth: 205 },
	{ key: "batchNo", label: t("pharma.complianceDashboard.batch"), minWidth: 125 },
	{ key: "actionText", label: t("pharma.complianceDashboard.openActions"), width: 120 },
	{ key: "observedText", label: t("pharma.complianceDashboard.observedDue"), minWidth: 175 }
];

onMounted(() => void refresh());

function currentQuery(): ComplianceDashboardQuery {
	return { keyword: keyword.value, risk: risk.value, source: source.value, limit: 200 };
}

async function refresh(): Promise<void> {
	if (!canRead.value) return;
	loading.value = true;
	error.value = "";
	try {
		snapshot.value = await getComplianceDashboard(currentQuery());
		const requested = typeof route.query.riskId === "string" ? route.query.riskId : "";
		if (requested) {
			const match = snapshot.value.items.find((item) => item.id === requested);
			if (match) openDetail(match);
		}
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		loading.value = false;
	}
}

async function exportCSV(): Promise<void> {
	if (!canExport.value || exporting.value) return;
	exporting.value = true;
	error.value = "";
	try {
		const blob = await exportComplianceDashboard(currentQuery());
		downloadBlob({ blob, filename: `pharma-oa-compliance-risks-${new Date().toISOString().slice(0, 10)}.csv` });
		ElMessage.success(t("pharma.complianceDashboard.exportStarted"));
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		exporting.value = false;
	}
}

function clearFilters(): void {
	keyword.value = "";
	risk.value = "";
	source.value = "";
	void refresh();
}

function selectSource(value: string): void {
	source.value = source.value === value ? "" : value as ComplianceSource;
	void refresh();
}

function openDetail(item: ComplianceRiskItem): void {
	selected.value = item;
	detailOpen.value = true;
}

async function openSource(item: ComplianceRiskItem): Promise<void> {
	detailOpen.value = false;
	await router.push(item.targetPath);
}

function sourceLabel(value: ComplianceSource): string {
	return valueLabel(value);
}

function riskType(value: ComplianceRisk): "danger" | "warning" {
	return value === "high" ? "danger" : "warning";
}

function isComplianceSource(value: unknown): value is ComplianceSource {
	return value === "qualification" || value === "quality_complaint" || value === "drug_recall" || value === "cold_chain";
}
</script>

<template>
	<PageShell :title="t('pharma.complianceDashboard.title')" :description="t('pharma.complianceDashboard.description')" :loading="loading" :error="error" :forbidden="!canRead" :forbidden-title="t('pharma.complianceDashboard.noComplianceDashboardAccess')" :forbidden-description="t('pharma.complianceDashboard.thisPageRequiresPharmaOaComplianceDashboardReadPermission')">
		<template #actions>
			<el-tooltip :content="t('pharma.complianceDashboard.refresh56e3badc')"><el-button circle :aria-label="t('pharma.complianceDashboard.refresh')" :icon="RefreshCw" :loading="loading" @click="refresh" /></el-tooltip>
			<el-button :icon="Download" :loading="exporting" :disabled="!canExport || loading" @click="exportCSV">{{ t("pharma.complianceDashboard.exportEvidence") }}</el-button>
		</template>

		<section class="summary-band" :aria-label="t('pharma.complianceDashboard.riskSummary')">
			<div><span>{{ t("pharma.complianceDashboard.activeRisks") }}</span><strong>{{ summary.total }}</strong></div>
			<div class="high"><span>{{ t("pharma.complianceDashboard.highRisk") }}</span><strong>{{ summary.high }}</strong></div>
			<div><span>{{ t("pharma.complianceDashboard.mediumRisk") }}</span><strong>{{ summary.medium }}</strong></div>
			<div><span>{{ t("pharma.complianceDashboard.matchedView") }}</span><strong>{{ snapshot?.matchedCount ?? 0 }}</strong></div>
		</section>

		<section class="source-band" :aria-label="t('pharma.complianceDashboard.riskSources')">
			<button v-for="item in sourceStats" :key="item.key" type="button" :class="{ active: source === item.key }" @click="selectSource(item.key)"><span>{{ item.label }}</span><strong>{{ item.value }}</strong></button>
		</section>

		<div class="filter-band">
			<el-input v-model="keyword" clearable :placeholder="t('pharma.complianceDashboard.keywordPlaceholder')" :prefix-icon="Search" @keyup.enter="refresh" @clear="refresh" />
			<el-select v-model="source" clearable :placeholder="t('pharma.complianceDashboard.allSources')" @change="refresh">
				<el-option :label="t('pharma.complianceDashboard.qualifications')" value="qualification" />
				<el-option :label="t('pharma.complianceDashboard.qualityComplaints')" value="quality_complaint" />
				<el-option :label="t('pharma.complianceDashboard.drugRecalls')" value="drug_recall" />
				<el-option :label="t('pharma.complianceDashboard.coldChain')" value="cold_chain" />
			</el-select>
			<el-segmented v-model="risk" :options="[{ label: t('pharma.complianceDashboard.allRisk'), value: '' }, { label: valueLabel('high'), value: 'high' }, { label: valueLabel('medium'), value: 'medium' }]" @change="refresh" />
			<el-tooltip :content="t('pharma.complianceDashboard.clearFilters')"><el-button circle :aria-label="t('pharma.complianceDashboard.clearFilters')" :icon="X" :disabled="!keyword && !risk && !source" @click="clearFilters" /></el-tooltip>
		</div>

		<DataTable :rows="rows" :columns="columns" row-key="id" :loading="loading" :error="error" :empty-text="t('pharma.complianceDashboard.empty')">
			<template #cell-risk="{ row }"><el-tag :type="riskType(row.risk as ComplianceRisk)">{{ valueLabel(row.risk) }}</el-tag></template>
			<template #cell-reference="{ row }"><strong>{{ row.reference }}</strong></template>
			<template #cell-batchNo="{ row }">{{ row.batchNo || "-" }}</template>
			<template #actions="{ row }">
				<el-tooltip :content="t('pharma.complianceDashboard.viewTrace')"><el-button circle :aria-label="t('pharma.complianceDashboard.viewTrace')" :icon="Eye" @click="openDetail(row as RiskRow)" /></el-tooltip>
				<el-tooltip :content="t('pharma.complianceDashboard.openSource')"><el-button circle type="primary" :aria-label="t('pharma.complianceDashboard.openSource')" :icon="ExternalLink" @click="openSource(row as RiskRow)" /></el-tooltip>
			</template>
		</DataTable>

		<DetailDrawer v-model="detailOpen" :title="selected ? `${sourceLabel(selected.source)} / ${selected.reference}` : t('pharma.complianceDashboard.sourceTrace')" size="54%">
			<template v-if="selected">
				<div class="detail-status"><el-tag :type="riskType(selected.risk)">{{ valueLabel(selected.risk) }} {{ t("pharma.complianceDashboard.risk") }}</el-tag><el-tag>{{ valueLabel(selected.status) }}</el-tag></div>
				<div class="risk-heading"><ShieldAlert :size="22" /><div><h3>{{ selected.title }}</h3><p>{{ selected.subject }}</p></div></div>
				<dl class="detail-grid">
					<div><dt>{{ t("pharma.complianceDashboard.sourceId") }}</dt><dd>{{ selected.sourceId }}</dd></div>
					<div><dt>{{ t("pharma.complianceDashboard.observedDue") }}</dt><dd>{{ new Date(selected.observedAt).toLocaleString() }}</dd></div>
					<div><dt>{{ t("pharma.complianceDashboard.batch") }}</dt><dd>{{ selected.batchNo || "Not batch-specific" }}</dd></div>
					<div><dt>{{ t("pharma.complianceDashboard.openActions") }}</dt><dd>{{ selected.pendingActions }} {{ t("pharma.complianceDashboard.of") }} {{ selected.totalActions }}</dd></div>
				</dl>
				<section class="trace-list"><h3>{{ t("pharma.complianceDashboard.sourceTrace") }}</h3><dl><div v-for="item in selected.trace" :key="`${item.label}-${item.value}`"><dt>{{ item.label }}</dt><dd>{{ item.value || "-" }}</dd></div></dl></section>
			</template>
			<template #footer><el-button @click="detailOpen = false">{{ t("pharma.complianceDashboard.close") }}</el-button><el-button v-if="selected" type="primary" :icon="ExternalLink" @click="openSource(selected)">{{ t("pharma.complianceDashboard.openSource") }}</el-button></template>
		</DetailDrawer>
	</PageShell>
</template>

<style scoped>
.summary-band{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:24px;padding:4px 0 16px;border-bottom:1px solid var(--el-border-color-lighter)}.summary-band div{display:grid;gap:4px}.summary-band span{color:var(--el-text-color-secondary);font-size:13px}.summary-band strong{font-size:24px;line-height:1.1}.summary-band .high strong{color:var(--el-color-danger)}.source-band{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));border:1px solid var(--el-border-color-lighter);border-radius:6px;overflow:hidden}.source-band button{display:flex;align-items:center;justify-content:space-between;gap:12px;min-width:0;padding:12px 14px;border:0;border-right:1px solid var(--el-border-color-lighter);background:var(--el-bg-color);color:var(--el-text-color-regular);cursor:pointer}.source-band button:last-child{border-right:0}.source-band button:hover,.source-band button.active{background:var(--el-fill-color-light);color:var(--el-color-primary)}.source-band span{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.source-band strong{font-size:17px}.filter-band{display:grid;grid-template-columns:minmax(240px,1fr) 190px auto 40px;gap:10px;align-items:center}.detail-status{display:flex;gap:8px}.risk-heading{display:flex;align-items:flex-start;gap:12px;padding:16px 0;border-bottom:1px solid var(--el-border-color-lighter)}.risk-heading h3{margin:0;font-size:17px}.risk-heading p{margin:5px 0 0;color:var(--el-text-color-secondary);overflow-wrap:anywhere}.detail-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px;margin:0}.detail-grid div,.trace-list dl div{min-width:0}.detail-grid dt,.trace-list dt{color:var(--el-text-color-secondary);font-size:12px}.detail-grid dd,.trace-list dd{margin:5px 0 0;overflow-wrap:anywhere}.trace-list{padding-top:16px;border-top:1px solid var(--el-border-color-lighter)}.trace-list h3{margin:0 0 12px;font-size:15px}.trace-list dl{display:grid;gap:12px;margin:0}.trace-list dl div{display:grid;grid-template-columns:150px minmax(0,1fr);gap:12px;padding-bottom:10px;border-bottom:1px solid var(--el-border-color-extra-light)}@media(max-width:900px){.filter-band{grid-template-columns:1fr 180px}.filter-band :deep(.el-segmented){width:100%}.source-band{grid-template-columns:repeat(2,minmax(0,1fr))}.source-band button:nth-child(2){border-right:0}.source-band button:nth-child(-n+2){border-bottom:1px solid var(--el-border-color-lighter)}}@media(max-width:760px){.summary-band{grid-template-columns:repeat(2,minmax(0,1fr));gap:16px}.summary-band strong{font-size:20px}.filter-band{grid-template-columns:1fr}.filter-band :deep(.el-button.is-circle){justify-self:start}.source-band{grid-template-columns:1fr}.source-band button{border-right:0;border-bottom:1px solid var(--el-border-color-lighter)}.source-band button:last-child{border-bottom:0}.detail-grid{grid-template-columns:1fr}.trace-list dl div{grid-template-columns:1fr;gap:3px}}
</style>

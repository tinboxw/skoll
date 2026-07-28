<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { RefreshCw, Search } from "lucide-vue-next";
import { useRoute } from "vue-router";

import StateBlock from "../../../components/Common/StateBlock.vue";
import { useI18n } from "../../../i18n";
import { formatDateTime } from "../../../utils/common";
import type { PluginDiagnosticError } from "../api";
import { usePluginDiagnostics } from "../diagnostics";
import { usePluginWorkspace } from "../workspace";

const workspace = usePluginWorkspace();
const diagnostics = usePluginDiagnostics(workspace.pluginId);
const route = useRoute();
const { t } = useI18n();
const correlation = ref(String(route.query.correlation || ""));
const query = computed(() => ({ correlation: correlation.value, limit: 100 }));
const summaryItems = computed(() => {
	const summary = diagnostics.snapshot.value?.summary;
	return [
		{ key: "totalJobs", value: summary?.totalJobs ?? 0 },
		{ key: "activeJobs", value: summary?.activeJobs ?? 0 },
		{ key: "deadLetters", value: summary?.deadLetters ?? 0 },
		{ key: "auditEvents", value: summary?.auditEvents ?? 0 },
		{ key: "failureCount", value: summary?.failureCount ?? 0 }
	];
});

function severityType(value: PluginDiagnosticError["severity"]): "warning" | "danger" {
	return value === "medium" ? "warning" : "danger";
}

function correlationValues(item: PluginDiagnosticError): string[] {
	return Object.values(item.correlation).filter((value): value is string => Boolean(value));
}

onMounted(() => diagnostics.refresh(query.value));
</script>

<template>
	<section class="diagnostics-surface" data-testid="plugin-diagnostics">
		<div class="surface-heading">
			<div><h3>{{ t("plugin.center.diagnosticsTitle") }}</h3><p>{{ t("plugin.center.diagnosticsDescription") }}</p></div>
			<div class="filters">
				<el-input v-model="correlation" clearable :prefix-icon="Search" :placeholder="t('plugin.center.correlationPlaceholder')" @keyup.enter="diagnostics.refresh(query)" />
				<el-tooltip :content="t('plugin.refresh')"><el-button circle :icon="RefreshCw" :aria-label="t('plugin.refresh')" :loading="diagnostics.loading.value" @click="diagnostics.refresh(query)" /></el-tooltip>
			</div>
		</div>
		<el-skeleton v-if="diagnostics.loading.value" :rows="8" animated />
		<StateBlock v-else-if="diagnostics.error.value" type="error" :description="diagnostics.error.value">
			<template #actions><el-button type="primary" :icon="RefreshCw" @click="diagnostics.refresh(query)">{{ t("common.retry") }}</el-button></template>
		</StateBlock>
		<template v-else-if="diagnostics.snapshot.value">
			<div class="summary-band" :aria-label="t('plugin.center.diagnosticsSummary')">
				<div v-for="item in summaryItems" :key="item.key" class="summary-item"><strong>{{ item.value }}</strong><span>{{ t(`plugin.center.summary.${item.key}`) }}</span></div>
			</div>
			<el-alert
				:type="diagnostics.snapshot.value.health.status === 'unhealthy' ? 'error' : 'success'"
				show-icon
				:closable="false"
				:title="t(`plugin.center.health.${diagnostics.snapshot.value.health.status}`)"
				:description="`${diagnostics.snapshot.value.health.code} / ${formatDateTime(diagnostics.snapshot.value.health.checkedAt)}`"
			/>
			<StateBlock v-if="!diagnostics.snapshot.value.errors.length" type="empty" :description="t('plugin.center.noDiagnosticErrors')" />
			<div v-else class="table-scroll">
				<el-table :data="diagnostics.snapshot.value.errors" size="small" stripe border>
					<el-table-column :label="t('plugin.center.severity')" width="105"><template #default="scope"><el-tag :type="severityType(scope.row.severity)" effect="light">{{ t(`plugin.center.severity.${scope.row.severity}`) }}</el-tag></template></el-table-column>
					<el-table-column :label="t('plugin.center.errorCategory')" width="115"><template #default="scope">{{ t(`plugin.center.errorCategory.${scope.row.category}`) }}</template></el-table-column>
					<el-table-column prop="summary" :label="t('plugin.center.errorSummary')" min-width="230" show-overflow-tooltip />
					<el-table-column prop="owner" :label="t('plugin.center.owner')" min-width="135" show-overflow-tooltip />
					<el-table-column prop="stage" :label="t('plugin.center.stage')" min-width="170" show-overflow-tooltip />
					<el-table-column :label="t('plugin.center.retryable')" width="105">
						<template #default="scope"><el-tag :type="scope.row.retryable ? 'warning' : 'info'" effect="plain">{{ t(scope.row.retryable ? 'common.yes' : 'common.no') }}</el-tag></template>
					</el-table-column>
					<el-table-column :label="t('plugin.center.correlation')" min-width="260">
						<template #default="scope"><div class="correlation-list"><el-tag v-for="value in correlationValues(scope.row)" :key="value" effect="plain" size="small">{{ value }}</el-tag></div></template>
					</el-table-column>
					<el-table-column prop="evidenceId" :label="t('plugin.center.evidenceId')" min-width="220" show-overflow-tooltip />
					<el-table-column :label="t('plugin.center.occurredAt')" min-width="170"><template #default="scope">{{ formatDateTime(scope.row.occurredAt) }}</template></el-table-column>
				</el-table>
			</div>
		</template>
	</section>
</template>

<style scoped>
.diagnostics-surface { display: grid; gap: 16px; min-width: 0; }
.surface-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; }
.surface-heading h3, .surface-heading p { margin: 0; }
.surface-heading p { margin-top: 4px; color: var(--color-text-muted); }
.filters { display: grid; grid-template-columns: minmax(240px, 360px) 32px; align-items: center; gap: 8px; }
.summary-band { display: grid; grid-template-columns: repeat(5, minmax(0, 1fr)); border-block: 1px solid var(--color-border); }
.summary-item { display: grid; gap: 2px; padding: 14px 16px; border-right: 1px solid var(--color-border); }
.summary-item:last-child { border-right: 0; }
.summary-item strong { font-size: 22px; line-height: 1.2; }
.summary-item span { color: var(--color-text-muted); font-size: 13px; }
.table-scroll { min-width: 0; overflow-x: auto; }
.table-scroll :deep(.el-table) { min-width: 1520px; }
.correlation-list { display: flex; flex-wrap: wrap; gap: 4px; }
@media (max-width: 760px) { .surface-heading, .filters { display: grid; grid-template-columns: 1fr; } .summary-band { grid-template-columns: repeat(2, minmax(0, 1fr)); } .summary-item { border-bottom: 1px solid var(--color-border); } }
</style>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { RefreshCw, Search } from "lucide-vue-next";
import { useRouter } from "vue-router";

import StateBlock from "../../../components/Common/StateBlock.vue";
import { useI18n } from "../../../i18n";
import { formatDateTime } from "../../../utils/common";
import type { PluginDiagnosticAudit } from "../api";
import { usePluginDiagnostics } from "../diagnostics";
import { usePluginWorkspace } from "../workspace";

const workspace = usePluginWorkspace();
const diagnostics = usePluginDiagnostics(workspace.pluginId);
const router = useRouter();
const { t } = useI18n();
const result = ref<"" | "success" | "failure" | "denied">("");
const correlation = ref("");
const resultOptions = computed(() => ["", "success", "failure", "denied"].map((value) => ({
	value,
	label: value ? t(`plugin.center.auditResult.${value}`) : t("plugin.center.allResults")
})));
const query = computed(() => ({ auditResult: result.value, correlation: correlation.value, limit: 100 }));

function resultType(value: PluginDiagnosticAudit["result"]): "success" | "warning" | "danger" | "info" {
	if (value === "success") return "success";
	if (value === "denied") return "warning";
	if (value === "failure") return "danger";
	return "info";
}

function inspectCorrelation(value: string): void {
	if (!value) return;
	void router.push({ name: "plugin-center-diagnostics", params: { pluginId: workspace.pluginId.value }, query: { correlation: value } });
}

onMounted(() => diagnostics.refresh(query.value));
</script>

<template>
	<section class="audit-surface" data-testid="plugin-audit">
		<div class="surface-heading">
			<div><h3>{{ t("plugin.center.auditTitle") }}</h3><p>{{ t("plugin.center.auditDescription") }}</p></div>
			<div class="filters">
				<el-select v-model="result" :aria-label="t('plugin.center.auditResultFilter')">
					<el-option v-for="option in resultOptions" :key="option.value" :label="option.label" :value="option.value" />
				</el-select>
				<el-input v-model="correlation" clearable :prefix-icon="Search" :placeholder="t('plugin.center.correlationPlaceholder')" @keyup.enter="diagnostics.refresh(query)" />
				<el-tooltip :content="t('plugin.refresh')"><el-button circle :icon="RefreshCw" :aria-label="t('plugin.refresh')" :loading="diagnostics.loading.value" @click="diagnostics.refresh(query)" /></el-tooltip>
			</div>
		</div>
		<el-skeleton v-if="diagnostics.loading.value" :rows="8" animated />
		<StateBlock v-else-if="diagnostics.error.value" type="error" :description="diagnostics.error.value">
			<template #actions><el-button type="primary" :icon="RefreshCw" @click="diagnostics.refresh(query)">{{ t("common.retry") }}</el-button></template>
		</StateBlock>
		<StateBlock v-else-if="!diagnostics.snapshot.value?.audit.length" type="empty" :description="t('plugin.center.noAudit')" />
		<div v-else class="table-scroll">
			<el-table :data="diagnostics.snapshot.value.audit" size="small" stripe border>
				<el-table-column :label="t('plugin.center.auditResultLabel')" width="105"><template #default="scope"><el-tag :type="resultType(scope.row.result)" effect="light">{{ scope.row.result ? t(`plugin.center.auditResult.${scope.row.result}`) : "-" }}</el-tag></template></el-table-column>
				<el-table-column prop="action" :label="t('plugin.center.auditAction')" min-width="200" show-overflow-tooltip />
				<el-table-column :label="t('plugin.center.resource')" min-width="210"><template #default="scope"><span class="resource-cell">{{ scope.row.resourceType }}<small>{{ scope.row.resourceId || "-" }}</small></span></template></el-table-column>
				<el-table-column prop="actorId" :label="t('plugin.center.actor')" min-width="120" show-overflow-tooltip />
				<el-table-column :label="t('plugin.center.correlation')" min-width="190">
					<template #default="scope">
						<el-button v-if="scope.row.traceId" link type="primary" @click="inspectCorrelation(scope.row.traceId)">{{ scope.row.traceId }}</el-button>
						<el-button v-else-if="scope.row.requestId" link type="primary" @click="inspectCorrelation(scope.row.requestId)">{{ scope.row.requestId }}</el-button>
						<span v-else>{{ scope.row.id }}</span>
					</template>
				</el-table-column>
				<el-table-column :label="t('plugin.center.occurredAt')" min-width="170"><template #default="scope">{{ formatDateTime(scope.row.occurredAt) }}</template></el-table-column>
			</el-table>
		</div>
	</section>
</template>

<style scoped>
.audit-surface { display: grid; gap: 16px; min-width: 0; }
.surface-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; }
.surface-heading h3, .surface-heading p { margin: 0; }
.surface-heading p { margin-top: 4px; color: var(--color-text-muted); }
.filters { display: grid; grid-template-columns: 150px minmax(220px, 320px) 32px; align-items: center; gap: 8px; }
.table-scroll { min-width: 0; overflow-x: auto; }
.table-scroll :deep(.el-table) { min-width: 1080px; }
.resource-cell { display: grid; gap: 2px; }
.resource-cell small { color: var(--color-text-muted); }
@media (max-width: 760px) { .surface-heading, .filters { display: grid; grid-template-columns: 1fr; } .filters { width: 100%; } }
</style>

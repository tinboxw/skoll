<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { RefreshCw, RotateCcw } from "lucide-vue-next";

import StateBlock from "../../../components/Common/StateBlock.vue";
import { confirmAction } from "../../../composables/useConfirmAction";
import { useI18n } from "../../../i18n";
import { BUTTON_ACCESS, useButtonAccess } from "../../../permissions/button";
import { formatDateTime } from "../../../utils/common";
import type { PluginDiagnosticJob, PluginDiagnosticJobStatus } from "../api";
import { usePluginDiagnostics } from "../diagnostics";
import { usePluginWorkspace } from "../workspace";

const workspace = usePluginWorkspace();
const diagnostics = usePluginDiagnostics(workspace.pluginId);
const access = useButtonAccess();
const { t } = useI18n();
const status = ref<PluginDiagnosticJobStatus | "">("");
const statusOptions = computed(() => ["", "scheduled", "running", "retry_wait", "succeeded", "dead_letter"].map((value) => ({
	value,
	label: value ? t(`plugin.center.jobStatus.${value}`) : t("plugin.center.allStatuses")
})));
const query = computed(() => ({ jobStatus: status.value, limit: 100 }));

function statusType(value: PluginDiagnosticJobStatus): "success" | "warning" | "danger" | "info" | "primary" {
	if (value === "succeeded") return "success";
	if (value === "dead_letter") return "danger";
	if (value === "running") return "primary";
	if (value === "retry_wait") return "warning";
	return "info";
}

function mayRetry(item: PluginDiagnosticJob): boolean {
	return item.canRetry && access.can(BUTTON_ACCESS.pluginManage) && access.currentRole.value.toLowerCase() === "super_admin";
}

async function retry(item: PluginDiagnosticJob): Promise<void> {
	if (!mayRetry(item)) return;
	const accepted = await confirmAction({
		title: t("plugin.center.retryJobTitle"),
		message: t("plugin.center.retryJobConfirm", { pluginId: workspace.pluginId.value, jobId: item.id, kind: item.kind }),
		confirmText: t("plugin.center.retryJobAction"),
		cancelText: t("common.cancel"),
		type: "warning",
		danger: true
	});
	if (accepted) await diagnostics.retry(item.id, query.value);
}

onMounted(() => diagnostics.refresh(query.value));
</script>

<template>
	<section class="jobs-surface" data-testid="plugin-jobs">
		<div class="surface-heading">
			<div><h3>{{ t("plugin.center.jobsTitle") }}</h3><p>{{ t("plugin.center.jobsDescription") }}</p></div>
			<div class="filters">
				<el-select v-model="status" :aria-label="t('plugin.center.jobStatusFilter')" @change="diagnostics.refresh(query)">
					<el-option v-for="option in statusOptions" :key="option.value" :label="option.label" :value="option.value" />
				</el-select>
				<el-tooltip :content="t('plugin.refresh')"><el-button circle :icon="RefreshCw" :aria-label="t('plugin.refresh')" :loading="diagnostics.loading.value" @click="diagnostics.refresh(query)" /></el-tooltip>
			</div>
		</div>
		<el-alert
			v-if="diagnostics.retryResult.value"
			type="success"
			show-icon
			:closable="false"
			:title="t('plugin.center.retryJobCompleted', { jobId: diagnostics.retryResult.value.retryJob.id })"
			:description="diagnostics.retryResult.value.operationId"
		/>
		<el-skeleton v-if="diagnostics.loading.value" :rows="8" animated />
		<StateBlock v-else-if="diagnostics.error.value" type="error" :description="diagnostics.error.value">
			<template #actions><el-button type="primary" :icon="RefreshCw" @click="diagnostics.refresh(query)">{{ t("common.retry") }}</el-button></template>
		</StateBlock>
		<StateBlock v-else-if="!diagnostics.snapshot.value?.jobs.length" type="empty" :description="t('plugin.center.noJobs')" />
		<div v-else class="table-scroll">
			<el-table :data="diagnostics.snapshot.value.jobs" size="small" stripe border>
				<el-table-column prop="id" :label="t('plugin.center.jobId')" min-width="190" show-overflow-tooltip />
				<el-table-column prop="kind" :label="t('plugin.center.jobKind')" min-width="140" show-overflow-tooltip />
				<el-table-column :label="t('plugin.center.jobStatus')" width="125">
					<template #default="scope"><el-tag :data-testid="`plugin-job-status-${scope.row.status}`" :type="statusType(scope.row.status)" effect="light">{{ t(`plugin.center.jobStatus.${scope.row.status}`) }}</el-tag></template>
				</el-table-column>
				<el-table-column :label="t('plugin.center.attempts')" width="105"><template #default="scope">{{ scope.row.attemptCount }} / {{ scope.row.maxAttempts }}</template></el-table-column>
				<el-table-column prop="lastError" :label="t('plugin.center.lastError')" min-width="210" show-overflow-tooltip />
				<el-table-column :label="t('plugin.center.updatedAt')" min-width="170"><template #default="scope">{{ formatDateTime(scope.row.updatedAt) }}</template></el-table-column>
				<el-table-column :label="t('common.actions')" width="82" fixed="right">
					<template #default="scope">
						<el-tooltip v-if="scope.row.canRetry" :content="mayRetry(scope.row) ? t('plugin.center.retryJobAction') : t('plugin.center.retryJobSuperAdmin')">
							<el-button data-testid="plugin-job-retry" circle :icon="RotateCcw" :aria-label="t('plugin.center.retryJobAction')" :disabled="!mayRetry(scope.row)" :loading="diagnostics.operating.value" @click="retry(scope.row)" />
						</el-tooltip>
					</template>
				</el-table-column>
			</el-table>
		</div>
	</section>
</template>

<style scoped>
.jobs-surface { display: grid; gap: 16px; min-width: 0; }
.surface-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; }
.surface-heading h3, .surface-heading p { margin: 0; }
.surface-heading p { margin-top: 4px; color: var(--color-text-muted); }
.filters { display: flex; align-items: center; gap: 8px; }
.filters .el-select { width: 180px; }
.table-scroll { min-width: 0; overflow-x: auto; }
.table-scroll :deep(.el-table) { min-width: 980px; }
@media (max-width: 620px) { .surface-heading { display: grid; } .filters { width: 100%; } .filters .el-select { flex: 1; } }
</style>

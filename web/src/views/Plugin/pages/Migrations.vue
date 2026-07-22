<script setup lang="ts">
import { computed, ref } from "vue";
import { RefreshCw, RotateCcw } from "lucide-vue-next";

import StateBlock from "../../../components/Common/StateBlock.vue";
import { confirmAction } from "../../../composables/useConfirmAction";
import { useI18n } from "../../../i18n";
import { BUTTON_ACCESS, useButtonAccess } from "../../../permissions/button";
import { formatDateTime } from "../../../utils/common";
import { usePluginDataControl } from "../data-control";
import { usePluginWorkspace } from "../workspace";

const workspace = usePluginWorkspace();
const control = usePluginDataControl(workspace.pluginId);
const access = useButtonAccess();
const { t } = useI18n();
const rollbackLimit = ref(1);
const snapshot = computed(() => control.snapshot.value);
const canOperateRollback = computed(() => Boolean(
	snapshot.value?.actions.canRollback &&
	access.can(BUTTON_ACCESS.pluginManage) &&
	access.currentRole.value.toLowerCase() === "super_admin"
));
const blockedReason = computed(() => snapshot.value?.actions.blockedReason
	? t(`plugin.center.rollbackBlocked.${snapshot.value.actions.blockedReason}`)
	: "");

async function rollback(): Promise<void> {
	if (!snapshot.value || !canOperateRollback.value) return;
	const accepted = await confirmAction({
		title: t("plugin.center.rollbackTitle"),
		message: t("plugin.center.rollbackConfirm", {
			pluginId: workspace.pluginId.value,
			limit: rollbackLimit.value,
			policy: snapshot.value.policy.rollback || "-",
			effect: t(`plugin.center.policy.${snapshot.value.policy.effect}`)
		}),
		confirmText: t("plugin.center.rollbackAction"),
		cancelText: t("common.cancel"),
		type: "error",
		danger: true
	});
	if (!accepted) return;
	await control.rollback(rollbackLimit.value);
	rollbackLimit.value = 1;
}
</script>

<template>
	<section class="migration-surface" data-testid="plugin-migrations">
		<el-skeleton v-if="control.loading.value" :rows="8" animated />
		<StateBlock v-else-if="control.error.value" type="error" :description="control.error.value">
			<template #actions><el-button type="primary" :icon="RefreshCw" @click="control.refresh">{{ t("common.retry") }}</el-button></template>
		</StateBlock>
		<template v-else-if="snapshot">
			<div class="surface-heading">
				<div><h3>{{ t("plugin.center.migrationsTitle") }}</h3><p>{{ t("plugin.center.migrationsDescription") }}</p></div>
				<el-tag :type="snapshot.migration.pending.length > 0 ? 'warning' : 'success'" effect="light">
					{{ snapshot.migration.pending.length > 0 ? t("plugin.center.pendingMigrations", { count: snapshot.migration.pending.length }) : t("plugin.center.migrationsCurrent") }}
				</el-tag>
			</div>
			<el-alert v-if="snapshot.migration.error" type="error" show-icon :closable="false" :title="snapshot.migration.error" />
			<el-alert
				v-if="control.rollbackResult.value"
				type="success"
				show-icon
				:closable="false"
				:title="t('plugin.center.rollbackCompleted', { count: control.rollbackResult.value.rolledBackSteps })"
				:description="`${control.rollbackResult.value.operationId} / ${formatDateTime(control.rollbackResult.value.completedAt)}`"
			/>
			<el-descriptions :column="4" border>
				<el-descriptions-item :label="t('plugin.center.declaredVersion')">{{ snapshot.migration.declaredVersion || "-" }}</el-descriptions-item>
				<el-descriptions-item :label="t('plugin.center.currentMigration')">{{ snapshot.migration.currentVersion || "-" }}</el-descriptions-item>
				<el-descriptions-item :label="t('plugin.center.appliedCount')">{{ snapshot.migration.applied.length }}</el-descriptions-item>
				<el-descriptions-item :label="t('plugin.center.pendingCount')">{{ snapshot.migration.pending.length }}</el-descriptions-item>
			</el-descriptions>

			<div class="migration-grid">
				<section class="migration-list">
					<h3>{{ t("plugin.center.appliedMigrations") }}</h3>
					<StateBlock v-if="snapshot.migration.applied.length === 0" type="empty" :description="t('plugin.center.noAppliedMigrations')" />
					<el-table v-else :data="snapshot.migration.applied" size="small" stripe border>
						<el-table-column prop="version" :label="t('plugin.table.version')" width="90" />
						<el-table-column prop="name" :label="t('plugin.table.name')" min-width="150" show-overflow-tooltip />
						<el-table-column :label="t('plugin.center.appliedAt')" min-width="170"><template #default="scope">{{ formatDateTime(scope.row.appliedAt) }}</template></el-table-column>
					</el-table>
				</section>
				<section class="migration-list">
					<h3>{{ t("plugin.center.pendingMigrationsTitle") }}</h3>
					<StateBlock v-if="snapshot.migration.pending.length === 0" type="empty" :description="t('plugin.center.noPendingMigrations')" />
					<el-table v-else :data="snapshot.migration.pending" size="small" stripe border>
						<el-table-column prop="version" :label="t('plugin.table.version')" width="90" />
						<el-table-column prop="name" :label="t('plugin.table.name')" min-width="150" show-overflow-tooltip />
						<el-table-column prop="checksum" :label="t('plugin.center.checksum')" min-width="170" show-overflow-tooltip />
					</el-table>
				</section>
			</div>

			<section class="rollback-section">
				<div><h3>{{ t("plugin.center.rollbackTitle") }}</h3><p>{{ t("plugin.center.rollbackDescription") }}</p></div>
				<el-alert v-if="!snapshot.actions.canRollback" data-testid="plugin-migration-blocked" type="info" show-icon :closable="false" :title="blockedReason" />
				<el-alert v-else-if="!canOperateRollback" type="warning" show-icon :closable="false" :title="t('plugin.center.rollbackSuperAdmin')" />
				<div v-else class="rollback-controls">
					<el-input-number v-model="rollbackLimit" :min="1" :max="snapshot.actions.rollbackMaxSteps" controls-position="right" :aria-label="t('plugin.center.rollbackSteps')" />
					<el-button data-testid="plugin-migration-rollback" type="danger" :icon="RotateCcw" :loading="control.operating.value" @click="rollback">{{ t("plugin.center.rollbackAction") }}</el-button>
				</div>
				<p class="policy-note">{{ t("plugin.center.rollbackPolicySummary", { rollback: snapshot.policy.rollback || "-", uninstall: snapshot.policy.uninstall || "-", effect: t(`plugin.center.policy.${snapshot.policy.effect}`) }) }}</p>
			</section>
		</template>
	</section>
</template>

<style scoped>
.migration-surface, .migration-list, .rollback-section { display: grid; gap: 16px; min-width: 0; }
.surface-heading { display: flex; justify-content: space-between; align-items: flex-start; gap: 16px; }
.surface-heading h3, .surface-heading p, .migration-list h3, .rollback-section h3, .rollback-section p { margin: 0; }
.surface-heading p, .rollback-section > div > p, .policy-note { margin-top: 4px; color: var(--color-text-muted); }
.migration-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: var(--layout-gap); }
.rollback-section { padding-top: 16px; border-top: 1px solid var(--color-border); }
.rollback-controls { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.policy-note { font-size: 13px; }
@media (max-width: 860px) {
	.surface-heading, .migration-grid { display: grid; grid-template-columns: 1fr; }
	.migration-surface :deep(.el-descriptions__body) { overflow-x: auto; }
	.migration-surface :deep(.el-descriptions__table) { min-width: 680px; }
}
</style>

<script setup lang="ts">
import { computed } from "vue";
import { RefreshCw } from "lucide-vue-next";

import StateBlock from "../../../components/Common/StateBlock.vue";
import { useI18n } from "../../../i18n";
import { formatDataSize, usePluginDataControl } from "../data-control";
import { usePluginWorkspace } from "../workspace";

const workspace = usePluginWorkspace();
const control = usePluginDataControl(workspace.pluginId);
const { t } = useI18n();
const snapshot = computed(() => control.snapshot.value);
</script>

<template>
	<section class="data-surface" data-testid="plugin-data">
		<el-skeleton v-if="control.loading.value" :rows="7" animated />
		<StateBlock v-else-if="control.error.value" type="error" :description="control.error.value">
			<template #actions><el-button type="primary" :icon="RefreshCw" @click="control.refresh">{{ t("common.retry") }}</el-button></template>
		</StateBlock>
		<template v-else-if="snapshot">
			<div class="surface-heading">
				<div><h3>{{ t("plugin.center.dataTitle") }}</h3><p>{{ t("plugin.center.dataDescription") }}</p></div>
				<el-tag :type="snapshot.schema.registered ? 'success' : snapshot.schema.available ? 'warning' : 'info'" effect="light">
					{{ snapshot.schema.registered ? t("plugin.center.schemaRegistered") : snapshot.schema.available ? t("plugin.center.schemaDeclaredInactive") : t("plugin.center.schemaUnavailable") }}
				</el-tag>
			</div>
			<el-descriptions class="responsive-descriptions" :column="3" border>
				<el-descriptions-item :label="t('plugin.center.namespace')">{{ snapshot.schema.namespace || "-" }}</el-descriptions-item>
				<el-descriptions-item :label="t('plugin.center.tableCount')">{{ snapshot.schema.tables.length }}</el-descriptions-item>
				<el-descriptions-item :label="t('plugin.center.storageSize')">{{ formatDataSize(snapshot.schema.totalSizeBytes, snapshot.schema.sizeKnown) }}</el-descriptions-item>
				<el-descriptions-item :label="t('plugin.center.uninstallPolicy')">{{ snapshot.policy.uninstall || "-" }}</el-descriptions-item>
				<el-descriptions-item :label="t('plugin.center.rollbackPolicy')">{{ snapshot.policy.rollback || "-" }}</el-descriptions-item>
				<el-descriptions-item :label="t('plugin.center.policyEffect')">{{ t(`plugin.center.policy.${snapshot.policy.effect}`) }}</el-descriptions-item>
			</el-descriptions>

			<section class="table-section">
				<h3>{{ t("plugin.center.ownedTables") }}</h3>
				<StateBlock v-if="snapshot.schema.tables.length === 0" type="empty" :description="t('plugin.center.noOwnedTables')" />
				<div v-else class="table-scroll">
					<el-table :data="snapshot.schema.tables" stripe border>
						<el-table-column prop="logicalName" :label="t('plugin.center.logicalTable')" min-width="150" />
						<el-table-column prop="physicalName" :label="t('plugin.center.physicalTable')" min-width="230" show-overflow-tooltip />
						<el-table-column :label="t('plugin.center.mutationPolicy')" width="120">
							<template #default="scope"><el-tag data-testid="plugin-table-mutation-policy" effect="plain">{{ t(`plugin.center.mutationPolicy.${scope.row.mutationPolicy}`) }}</el-tag></template>
						</el-table-column>
						<el-table-column :label="t('plugin.center.fields')" min-width="220">
							<template #default="scope"><span class="code-list">{{ scope.row.fields.join(", ") }}</span></template>
						</el-table-column>
						<el-table-column :label="t('plugin.center.primaryKey')" min-width="130">
							<template #default="scope">{{ scope.row.primaryKey.join(", ") || "-" }}</template>
						</el-table-column>
						<el-table-column prop="indexCount" :label="t('plugin.center.indexCount')" width="100" align="right" />
						<el-table-column :label="t('plugin.center.storageState')" width="120">
							<template #default="scope"><el-tag :type="scope.row.exists ? 'success' : 'danger'" effect="light">{{ scope.row.exists ? t("plugin.center.tableReady") : t("plugin.center.tableMissing") }}</el-tag></template>
						</el-table-column>
						<el-table-column :label="t('plugin.center.storageSize')" width="120" align="right">
							<template #default="scope">{{ formatDataSize(scope.row.sizeBytes, scope.row.sizeKnown) }}</template>
						</el-table-column>
					</el-table>
				</div>
			</section>
		</template>
	</section>
</template>

<style scoped>
.data-surface, .table-section { display: grid; gap: 16px; min-width: 0; }
.surface-heading { display: flex; justify-content: space-between; align-items: flex-start; gap: 16px; }
.surface-heading h3, .surface-heading p, .table-section h3 { margin: 0; }
.surface-heading p { margin-top: 4px; color: var(--color-text-muted); }
.table-scroll { min-width: 0; overflow-x: auto; }
.code-list { font-family: var(--font-mono, monospace); font-size: 12px; }
@media (max-width: 760px) {
	.surface-heading { display: grid; }
	.data-surface :deep(.el-descriptions__body) { overflow-x: auto; }
	.data-surface :deep(.el-descriptions__table) { min-width: 620px; }
}
</style>

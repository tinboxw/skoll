<script setup lang="ts">
import { computed } from "vue";

import StateBlock from "../../../components/Common/StateBlock.vue";
import { useI18n } from "../../../i18n";
import { usePluginWorkspace } from "../workspace";

const workspace = usePluginWorkspace();
const { t } = useI18n();
const capabilities = computed(() => workspace.snapshot.value?.capabilities ?? null);
const extensionRows = computed(() => capabilities.value ? Object.entries(capabilities.value.extensions).map(([name, count]) => ({ name, count })) : []);
</script>

<template>
	<div v-if="capabilities" class="capability-surface" data-testid="plugin-capabilities">
		<section class="surface-section">
			<h3>{{ t("plugin.center.hostServices") }}</h3>
			<div class="tag-list"><el-tag v-for="service in capabilities.hostServices" :key="service" effect="plain">{{ service }}</el-tag></div>
		</section>
		<section class="surface-section">
			<h3>{{ t("plugin.center.routes") }}</h3>
			<StateBlock v-if="capabilities.routes.length === 0" type="empty" :description="t('plugin.center.noRoutes')" />
			<el-table v-else :data="capabilities.routes" stripe border>
				<el-table-column prop="method" :label="t('plugin.center.method')" width="100" />
				<el-table-column prop="path" :label="t('plugin.advanced.preflight.path')" min-width="260" show-overflow-tooltip />
				<el-table-column prop="permission" :label="t('plugin.advanced.detail.permissions')" min-width="170" show-overflow-tooltip />
				<el-table-column prop="auditAction" :label="t('plugin.center.auditAction')" min-width="180" show-overflow-tooltip />
				<el-table-column prop="source" :label="t('plugin.center.source')" width="130" />
			</el-table>
		</section>
		<section class="surface-section">
			<h3>{{ t("plugin.advanced.detail.permissions") }}</h3>
			<StateBlock v-if="capabilities.permissions.length === 0" type="empty" :description="t('plugin.center.noPermissions')" />
			<el-table v-else :data="capabilities.permissions" stripe border>
				<el-table-column prop="key" :label="t('plugin.advanced.preflight.key')" min-width="200" />
				<el-table-column prop="module" :label="t('plugin.center.module')" width="140" />
				<el-table-column prop="type" :label="t('plugin.advanced.preflight.type')" width="120" />
				<el-table-column prop="risk" :label="t('plugin.advanced.preflight.risk')" width="110" />
			</el-table>
		</section>
		<section class="surface-grid">
			<div><h3>{{ t("plugin.center.dependencies") }}</h3><el-table :data="capabilities.dependencies" size="small"><el-table-column prop="id" :label="t('plugin.table.id')" /><el-table-column prop="version" :label="t('plugin.table.version')" /></el-table></div>
			<div><h3>{{ t("plugin.center.extensions") }}</h3><el-table :data="extensionRows" size="small"><el-table-column prop="name" :label="t('plugin.table.name')" /><el-table-column prop="count" :label="t('plugin.center.count')" width="100" /></el-table></div>
		</section>
	</div>
</template>

<style scoped>
.capability-surface, .surface-section { display: grid; gap: 12px; }
.surface-section { padding-bottom: 16px; border-bottom: 1px solid var(--color-border); }
.surface-section h3, .surface-grid h3 { margin: 0; }
.tag-list { display: flex; gap: 8px; flex-wrap: wrap; }
.surface-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: var(--layout-gap); }
@media (max-width: 760px) { .surface-grid { grid-template-columns: 1fr; } }
</style>

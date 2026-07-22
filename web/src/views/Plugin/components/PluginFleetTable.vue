<script setup lang="ts">
import { computed } from "vue";
import { ExternalLink, Settings2 } from "lucide-vue-next";

import type { FrontendPluginManifest } from "../../../plugins/types";
import { useI18n } from "../../../i18n";

const props = defineProps<{ items: FrontendPluginManifest[]; operating?: boolean }>();
const emit = defineEmits<{
	(e: "open", pluginId: string): void;
	(e: "visit", plugin: FrontendPluginManifest): void;
}>();
const { t, locale } = useI18n();
const rows = computed(() => props.items);

function displayName(plugin: FrontendPluginManifest): string {
	return locale.value === "en-US" ? plugin.nameEnUS || plugin.name : plugin.nameZhCN || plugin.name;
}

function canVisit(plugin: FrontendPluginManifest): boolean {
	return plugin.enabled !== false && plugin.uiMode !== "backend_only";
}

function openRow(plugin: FrontendPluginManifest): void {
	emit("open", plugin.id);
}
</script>

<template>
	<el-table :data="rows" stripe border class="plugin-fleet-table" data-testid="plugin-fleet-table" @row-click="openRow">
		<el-table-column prop="id" :label="t('plugin.table.id')" min-width="170" show-overflow-tooltip />
		<el-table-column :label="t('plugin.table.name')" min-width="180" show-overflow-tooltip>
			<template #default="{ row }"><strong>{{ displayName(row) }}</strong></template>
		</el-table-column>
		<el-table-column prop="appId" :label="t('plugin.table.appId')" min-width="120">
			<template #default="{ row }">{{ row.appId || row.level || "system" }}</template>
		</el-table-column>
		<el-table-column prop="version" :label="t('plugin.table.version')" width="110" />
		<el-table-column :label="t('plugin.table.status')" width="130">
			<template #default="{ row }">
				<el-tag :type="row.enabled === false ? 'info' : 'success'" effect="light">
					{{ row.enabled === false ? t('plugin.status.disabled') : t('plugin.status.enabled') }}
				</el-tag>
			</template>
		</el-table-column>
		<el-table-column :label="t('plugin.center.runtimeObservation')" min-width="150">
			<template #default="{ row }">
				<span class="observation-copy">{{ row.enabled === false ? t('plugin.center.state.disabled') : t('plugin.center.inspectRuntime') }}</span>
			</template>
		</el-table-column>
		<el-table-column :label="t('common.actions')" width="124" fixed="right">
			<template #default="{ row }">
				<div class="row-actions">
					<el-tooltip :content="t('plugin.center.openControl')">
						<el-button circle :icon="Settings2" :aria-label="t('plugin.center.openControl')" :disabled="operating" @click.stop="emit('open', row.id)" />
					</el-tooltip>
					<el-tooltip :content="t('plugin.action.visit')">
						<el-button circle :icon="ExternalLink" :aria-label="t('plugin.action.visit')" :disabled="operating || !canVisit(row)" @click.stop="emit('visit', row)" />
					</el-tooltip>
				</div>
			</template>
		</el-table-column>
	</el-table>
</template>

<style scoped>
.plugin-fleet-table { width: 100%; cursor: pointer; }
.row-actions { display: flex; gap: 6px; }
.observation-copy { color: var(--color-text-muted); font-size: 0.86rem; }
</style>

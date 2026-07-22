<script setup lang="ts">
import { computed } from "vue";
import { RefreshCw } from "lucide-vue-next";
import { useRoute, useRouter } from "vue-router";

import PageShell from "../../../components/Common/PageShell.vue";
import { useI18n } from "../../../i18n";
import { providePluginWorkspace } from "../workspace";
import { runtimeTagType } from "../model";
import PluginLifecycleCommands from "./PluginLifecycleCommands.vue";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const pluginId = computed(() => String(route.params.pluginId || "").trim());
const workspace = providePluginWorkspace(pluginId);
const plugin = computed(() => workspace.snapshot.value?.plugin ?? null);
const activeTab = computed(() => String(route.name || "plugin-center-overview"));
const tabs = computed(() => [
	{ name: "plugin-center-overview", label: t("plugin.center.nav.overview") },
	{ name: "plugin-center-runtime", label: t("plugin.center.nav.runtime") },
	{ name: "plugin-center-capabilities", label: t("plugin.center.nav.capabilities") },
	{ name: "plugin-center-settings", label: t("plugin.center.nav.settings") }
]);

function changeTab(name: string | number): void {
	void router.push({ name: String(name), params: { pluginId: pluginId.value } });
}
</script>

<template>
	<PageShell
		:title="plugin ? `${plugin.name} / ${plugin.id}` : pluginId"
		:description="t('plugin.center.workspaceDescription')"
		:loading="workspace.loading.value"
		:error="workspace.error.value"
	>
		<template #meta>
			<el-tag v-if="workspace.snapshot.value" :type="runtimeTagType(workspace.runtimeState.value)" effect="light" data-testid="plugin-runtime-state">
				{{ t(`plugin.center.state.${workspace.runtimeState.value}`) }}
			</el-tag>
		</template>
		<template #actions>
			<PluginLifecycleCommands />
			<el-tooltip :content="t('plugin.refresh')">
				<el-button circle :icon="RefreshCw" :aria-label="t('plugin.refresh')" :loading="workspace.loading.value" @click="workspace.refresh" />
			</el-tooltip>
		</template>
		<template #stateActions>
			<el-button type="primary" :icon="RefreshCw" @click="workspace.refresh">{{ t("common.retry") }}</el-button>
		</template>

		<nav class="workspace-navigation" :aria-label="t('plugin.center.workspaceNavigation')">
			<el-tabs :model-value="activeTab" stretch @tab-change="changeTab">
				<el-tab-pane v-for="tab in tabs" :key="tab.name" :name="tab.name" :label="tab.label" />
			</el-tabs>
		</nav>
		<router-view />
	</PageShell>
</template>

<style scoped>
.workspace-navigation { min-width: 0; border-bottom: 1px solid var(--color-border); }
.workspace-navigation :deep(.el-tabs__header) { margin: 0; }
@media (max-width: 620px) {
	.workspace-navigation { overflow-x: auto; }
	.workspace-navigation :deep(.el-tabs__nav) { min-width: 520px; }
}
</style>

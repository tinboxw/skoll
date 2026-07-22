<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { PackagePlus, RefreshCw, Store } from "lucide-vue-next";
import { useRouter } from "vue-router";

import PageShell from "../../components/Common/PageShell.vue";
import StateBlock from "../../components/Common/StateBlock.vue";
import { useI18n } from "../../i18n";
import { syncBackendPlugins } from "../../plugins";
import type { FrontendPluginManifest } from "../../plugins/types";
import { resolvePluginEntryPath, usePluginStore } from "../../stores/plugins";
import { toErrorMessage } from "../../utils/common";
import PluginFleetTable from "./components/PluginFleetTable.vue";

const router = useRouter();
const store = usePluginStore();
const { t } = useI18n();
const keyword = ref("");
const state = ref("");
const currentPage = ref(1);
const pageSize = 20;
const error = ref("");
const filtered = computed(() => {
	const value = keyword.value.trim().toLowerCase();
	return store.items.filter((plugin) => {
		const matchesKeyword = !value || [plugin.id, plugin.name, plugin.nameZhCN, plugin.nameEnUS, plugin.appId, plugin.version].some((field) => String(field ?? "").toLowerCase().includes(value));
		const matchesState = !state.value || (state.value === "enabled" ? plugin.enabled !== false : plugin.enabled === false);
		return matchesKeyword && matchesState;
	});
});
const paged = computed(() => {
	const start = (currentPage.value - 1) * pageSize;
	return filtered.value.slice(start, start + pageSize);
});

watch([keyword, state], () => { currentPage.value = 1; });
watch(() => filtered.value.length, (total) => {
	const lastPage = Math.max(1, Math.ceil(total / pageSize));
	if (currentPage.value > lastPage) currentPage.value = lastPage;
});

async function refresh(): Promise<void> {
	error.value = "";
	try { await syncBackendPlugins(router, store); }
	catch (reason) { error.value = toErrorMessage(reason); }
}

function visit(plugin: FrontendPluginManifest): void {
	if (plugin.uiOpenMode === "standalone") {
		window.open(`/skoll/v1/plugins/${encodeURIComponent(plugin.id)}/page`, "_blank", "noopener,noreferrer");
		return;
	}
	const path = resolvePluginEntryPath(plugin);
	if (path) void router.push(path);
}

onMounted(() => { if (!store.syncedFromServer) void refresh(); });
</script>

<template>
	<PageShell :title="t('plugin.title')" :description="t('plugin.center.fleetDescription')" :loading="store.isSyncing" :error="error || store.lastSyncError || ''">
		<template #actions>
			<el-button :icon="Store" @click="router.push('/skoll/plugin-center/marketplace')">{{ t("plugin.advanced.market.title") }}</el-button>
			<el-button :icon="PackagePlus" @click="router.push('/skoll/plugin-center/install')">{{ t("plugin.action.install") }}</el-button>
			<el-tooltip :content="t('plugin.refresh')"><el-button circle type="primary" :icon="RefreshCw" :aria-label="t('plugin.refresh')" @click="refresh" /></el-tooltip>
		</template>
		<template #stateActions><el-button type="primary" :icon="RefreshCw" @click="refresh">{{ t("common.retry") }}</el-button></template>
		<div class="fleet-toolbar">
			<el-input v-model="keyword" clearable :placeholder="t('plugin.filter.keywordPlaceholder')" />
			<el-select v-model="state" clearable :placeholder="t('plugin.filter.allStatus')">
				<el-option :label="t('plugin.status.enabled')" value="enabled" />
				<el-option :label="t('plugin.status.disabled')" value="disabled" />
			</el-select>
		</div>
		<StateBlock v-if="filtered.length === 0" type="empty" :description="t('plugin.filter.empty')" />
		<template v-else>
			<PluginFleetTable :items="paged" :operating="store.isSyncing" @open="(id) => router.push(`/skoll/plugin-center/${id}/overview`)" @visit="visit" />
			<el-pagination
				v-if="filtered.length > pageSize"
				v-model:current-page="currentPage"
				class="fleet-pagination"
				:page-size="pageSize"
				:total="filtered.length"
				layout="total, prev, pager, next"
			/>
		</template>
	</PageShell>
</template>

<style scoped>
.fleet-toolbar { display: grid; grid-template-columns: minmax(220px, 1fr) minmax(160px, 240px); gap: 8px; }
.fleet-pagination { justify-content: flex-end; margin-top: 12px; flex-wrap: wrap; }
@media (max-width: 620px) { .fleet-toolbar { grid-template-columns: 1fr; } }
</style>

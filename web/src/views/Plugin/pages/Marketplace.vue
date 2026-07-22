<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { PackagePlus, RefreshCw } from "lucide-vue-next";
import { useRouter } from "vue-router";

import PageShell from "../../../components/Common/PageShell.vue";
import StateBlock from "../../../components/Common/StateBlock.vue";
import { useI18n } from "../../../i18n";
import { toErrorMessage } from "../../../utils/common";
import { listMarketplace, type MarketplacePlugin } from "../api";

const router = useRouter();
const { t } = useI18n();
const items = ref<MarketplacePlugin[]>([]);
const keyword = ref("");
const loading = ref(false);
const error = ref("");
const filtered = computed(() => {
	const value = keyword.value.trim().toLowerCase();
	return value ? items.value.filter((item) => [item.id, item.name, item.version, item.description].some((field) => String(field ?? "").toLowerCase().includes(value))) : items.value;
});

async function load(): Promise<void> {
	loading.value = true; error.value = "";
	try { items.value = await listMarketplace(); }
	catch (reason) { error.value = toErrorMessage(reason); }
	finally { loading.value = false; }
}

function prepare(item: MarketplacePlugin): void {
	const path = item.manifestPath || item.packagePath;
	if (path) void router.push({ path: "/skoll/plugin-center/install", query: { path } });
}
onMounted(load);
</script>

<template>
	<PageShell :title="t('plugin.advanced.market.title')" :description="t('plugin.center.marketplaceDescription')" :error="error" :loading="loading">
		<template #actions>
			<el-button @click="router.push('/skoll/plugin-center')">{{ t("plugin.center.backToFleet") }}</el-button>
			<el-button type="primary" :icon="RefreshCw" @click="load">{{ t("plugin.refresh") }}</el-button>
		</template>
		<el-input v-model="keyword" clearable :placeholder="t('plugin.advanced.market.searchPlaceholder')" />
		<StateBlock v-if="filtered.length === 0" type="empty" :description="t('plugin.advanced.market.empty')" />
		<el-table v-else :data="filtered" stripe border data-testid="plugin-marketplace-table">
			<el-table-column prop="id" :label="t('plugin.table.id')" min-width="170" />
			<el-table-column prop="name" :label="t('plugin.table.name')" min-width="180" />
			<el-table-column prop="version" :label="t('plugin.table.version')" width="110" />
			<el-table-column :label="t('plugin.advanced.market.risk')" width="120"><template #default="{ row }"><el-tag type="warning" effect="plain">{{ row.risk.level }}</el-tag></template></el-table-column>
			<el-table-column :label="t('plugin.advanced.market.signature')" width="140"><template #default="{ row }">{{ row.signature.status }}</template></el-table-column>
			<el-table-column :label="t('common.actions')" width="100" fixed="right"><template #default="{ row }"><el-tooltip :content="t('plugin.action.install')"><el-button circle :icon="PackagePlus" :aria-label="t('plugin.action.install')" :disabled="!(row.manifestPath || row.packagePath)" @click="prepare(row)" /></el-tooltip></template></el-table-column>
		</el-table>
	</PageShell>
</template>

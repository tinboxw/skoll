<script setup lang="ts">
import { computed } from "vue";
import { Activity, Boxes, Settings2 } from "lucide-vue-next";
import { useRouter } from "vue-router";

import MetricStrip, { type MetricStripItem } from "../../../components/Common/MetricStrip.vue";
import { useI18n } from "../../../i18n";
import { formatTimestamp, runtimeTagType } from "../model";
import { usePluginWorkspace } from "../workspace";

const workspace = usePluginWorkspace();
const router = useRouter();
const { t } = useI18n();
const snapshot = computed(() => workspace.snapshot.value);
const metrics = computed<MetricStripItem[]>(() => snapshot.value ? [
	{ label: t("plugin.table.version"), value: snapshot.value.plugin.version || "-", note: snapshot.value.capabilities.apiVersion || "-" },
	{ label: t("plugin.signal.health"), value: t(`plugin.center.state.${workspace.runtimeState.value}`), note: snapshot.value.runtime.health?.code || "-", tone: workspace.runtimeState.value === "ready" ? "default" : "warning" },
	{ label: t("plugin.center.routes"), value: snapshot.value.capabilities.routes.length, note: `${snapshot.value.capabilities.permissions.length} ${t("plugin.advanced.detail.permissions")}` },
	{ label: t("plugin.center.hostServices"), value: snapshot.value.capabilities.hostServices.length, note: formatTimestamp(snapshot.value.capturedAt) }
] : []);

function open(name: string): void {
	void router.push({ name, params: { pluginId: workspace.pluginId.value } });
}
</script>

<template>
	<div v-if="snapshot" class="overview-surface" data-testid="plugin-overview">
		<MetricStrip :items="metrics" />
		<section class="surface-section">
			<div class="section-heading">
				<div><h3>{{ t("plugin.center.attention") }}</h3><p>{{ t("plugin.center.attentionDescription") }}</p></div>
				<el-tag :type="runtimeTagType(workspace.runtimeState.value)" effect="light">{{ t(`plugin.center.state.${workspace.runtimeState.value}`) }}</el-tag>
			</div>
			<el-alert v-if="workspace.runtimeState.value !== 'ready'" type="warning" :closable="false" show-icon :title="snapshot.runtime.health?.code || t('plugin.center.healthUnavailable')" />
			<el-alert v-else type="success" :closable="false" show-icon :title="t('plugin.center.noAttention')" />
		</section>
		<section class="surface-section">
			<h3>{{ t("plugin.center.identity") }}</h3>
			<el-descriptions :column="2" border>
				<el-descriptions-item :label="t('plugin.table.id')">{{ snapshot.plugin.id }}</el-descriptions-item>
				<el-descriptions-item :label="t('plugin.table.version')">{{ snapshot.plugin.version }}</el-descriptions-item>
				<el-descriptions-item :label="t('plugin.table.mode')">{{ snapshot.plugin.uiMode || "-" }}</el-descriptions-item>
				<el-descriptions-item :label="t('plugin.table.appId')">{{ snapshot.plugin.appId || snapshot.plugin.level || "system" }}</el-descriptions-item>
				<el-descriptions-item :label="t('plugin.center.installedAt')">{{ formatTimestamp(snapshot.runtime.installedAt) }}</el-descriptions-item>
				<el-descriptions-item :label="t('plugin.center.enabledAt')">{{ formatTimestamp(snapshot.runtime.enabledAt) }}</el-descriptions-item>
			</el-descriptions>
		</section>
		<div class="overview-links">
			<el-button :icon="Activity" @click="open('plugin-center-runtime')">{{ t("plugin.center.nav.runtime") }}</el-button>
			<el-button :icon="Boxes" @click="open('plugin-center-capabilities')">{{ t("plugin.center.nav.capabilities") }}</el-button>
			<el-button :icon="Settings2" @click="open('plugin-center-settings')">{{ t("plugin.center.nav.settings") }}</el-button>
		</div>
	</div>
</template>

<style scoped>
.overview-surface { display: grid; gap: var(--layout-gap); }
.surface-section { display: grid; gap: 12px; padding: 16px 0; border-bottom: 1px solid var(--color-border); }
.surface-section h3, .surface-section p { margin: 0; }
.surface-section p { margin-top: 4px; color: var(--color-text-muted); }
.section-heading { display: flex; justify-content: space-between; gap: 16px; align-items: flex-start; }
.overview-links { display: flex; gap: 8px; flex-wrap: wrap; }
@media (max-width: 620px) { .section-heading { display: grid; } }
</style>

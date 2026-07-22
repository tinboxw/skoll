<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";

import { useI18n } from "../../../i18n";
import { deriveRuntimeState, formatTimestamp, runtimeTagType } from "../model";
import { usePluginWorkspace } from "../workspace";

const workspace = usePluginWorkspace();
const { t } = useI18n();
const now = ref(Date.now());
let timer: ReturnType<typeof setInterval> | undefined;
onMounted(() => { timer = setInterval(() => { now.value = Date.now(); }, 5000); });
onBeforeUnmount(() => { if (timer) clearInterval(timer); });
const snapshot = computed(() => workspace.snapshot.value);
const state = computed(() => deriveRuntimeState(snapshot.value, now.value));
</script>

<template>
	<section v-if="snapshot" class="runtime-surface" data-testid="plugin-runtime">
		<div class="runtime-heading">
			<div><h3>{{ t("plugin.center.runtimeTitle") }}</h3><p>{{ t("plugin.center.runtimeDescription") }}</p></div>
			<el-tag :type="runtimeTagType(state)" effect="light" size="large">{{ t(`plugin.center.state.${state}`) }}</el-tag>
		</div>
		<el-alert v-if="state === 'stale'" type="warning" show-icon :closable="false" :title="t('plugin.center.staleObservation')" />
		<el-alert v-else-if="state === 'crashed'" type="error" show-icon :closable="false" :title="t('plugin.center.crashedObservation')" />
		<el-alert v-else-if="state === 'degraded'" type="warning" show-icon :closable="false" :title="t('plugin.center.degradedObservation')" />
		<el-descriptions :column="2" border>
			<el-descriptions-item :label="t('plugin.center.lifecycleState')">{{ snapshot.runtime.state }}</el-descriptions-item>
			<el-descriptions-item :label="t('plugin.signal.health')">{{ snapshot.runtime.health?.status || t('plugin.center.notObserved') }}</el-descriptions-item>
			<el-descriptions-item :label="t('plugin.center.healthCode')">{{ snapshot.runtime.health?.code || "-" }}</el-descriptions-item>
			<el-descriptions-item :label="t('plugin.center.latency')">{{ snapshot.runtime.health ? `${snapshot.runtime.health.latencyMillis} ms` : "-" }}</el-descriptions-item>
			<el-descriptions-item :label="t('plugin.center.checkedAt')">{{ formatTimestamp(snapshot.runtime.health?.checkedAt) }}</el-descriptions-item>
			<el-descriptions-item :label="t('plugin.center.staleAfter')">{{ formatTimestamp(snapshot.staleAfter) }}</el-descriptions-item>
			<el-descriptions-item :label="t('plugin.center.serviceBaseUrl')">{{ snapshot.runtime.serviceBaseUrl || "-" }}</el-descriptions-item>
			<el-descriptions-item :label="t('plugin.center.serviceHealthUrl')">{{ snapshot.runtime.serviceHealthUrl || "-" }}</el-descriptions-item>
			<el-descriptions-item :label="t('plugin.center.httpStatus')">{{ snapshot.runtime.health?.httpStatus || "-" }}</el-descriptions-item>
			<el-descriptions-item :label="t('plugin.center.capturedAt')">{{ formatTimestamp(snapshot.capturedAt) }}</el-descriptions-item>
		</el-descriptions>
	</section>
</template>

<style scoped>
.runtime-surface { display: grid; gap: 16px; }
.runtime-heading { display: flex; justify-content: space-between; gap: 16px; align-items: flex-start; }
.runtime-heading h3, .runtime-heading p { margin: 0; }
.runtime-heading p { margin-top: 4px; color: var(--color-text-muted); }
@media (max-width: 620px) { .runtime-heading { display: grid; } }
</style>

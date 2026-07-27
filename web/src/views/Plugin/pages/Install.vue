<script setup lang="ts">
import { computed, ref } from "vue";
import { CheckCircle2, PackagePlus } from "lucide-vue-next";
import { useRoute, useRouter } from "vue-router";

import PageShell from "../../../components/Common/PageShell.vue";
import { useI18n } from "../../../i18n";
import { toErrorMessage } from "../../../utils/common";
import { installPlugin, validatePluginInstall, type PluginInstallPreflight } from "../api";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const path = ref(typeof route.query.path === "string" ? route.query.path : "");
const validatedPath = ref("");
const preflight = ref<PluginInstallPreflight | null>(null);
const loading = ref(false);
const error = ref("");
const installed = ref(false);
const canInstall = computed(() => preflight.value?.status === "pass" && validatedPath.value === path.value.trim());
const permissionChanges = computed(() => preflight.value ? [preflight.value.permissions.add, preflight.value.permissions.update, preflight.value.permissions.conflict].reduce((sum, values) => sum + (values?.length ?? 0), 0) : 0);

async function validate(): Promise<void> {
	if (!path.value.trim()) return;
	loading.value = true; error.value = ""; installed.value = false; preflight.value = null;
	try { preflight.value = await validatePluginInstall(path.value.trim()); validatedPath.value = path.value.trim(); }
	catch (reason) { error.value = toErrorMessage(reason); validatedPath.value = ""; }
	finally { loading.value = false; }
}

async function install(): Promise<void> {
	if (!canInstall.value) return;
	loading.value = true; error.value = "";
	try { await installPlugin(path.value.trim()); installed.value = true; }
	catch (reason) { error.value = toErrorMessage(reason); }
	finally { loading.value = false; }
}
</script>

<template>
	<PageShell :title="t('plugin.installTitle')" :description="t('plugin.installDesc')" :error="error" :loading="loading">
		<template #actions><el-button @click="router.push('/skoll/plugin-center')">{{ t("plugin.center.backToFleet") }}</el-button></template>
		<div class="install-workflow" data-testid="plugin-install-workflow">
			<div class="install-input">
				<el-input v-model="path" clearable :placeholder="t('plugin.pathPlaceholder')" @input="preflight = null" />
				<el-button :icon="CheckCircle2" :disabled="!path.trim()" @click="validate">{{ t("plugin.action.validate") }}</el-button>
			</div>
			<el-alert v-if="installed" type="success" show-icon :closable="false" :title="t('plugin.center.installComplete')" />
			<section v-if="preflight" class="preflight-result">
				<div class="preflight-heading">
					<div><h3>{{ preflight.plugin.name }} / {{ preflight.plugin.id }}</h3><p>{{ preflight.plugin.version }} · {{ validatedPath }}</p></div>
					<div class="preflight-tags"><el-tag :type="preflight.status === 'pass' ? 'success' : 'danger'">{{ preflight.status }}</el-tag><el-tag type="warning" effect="plain">{{ preflight.risk.level }}</el-tag><el-tag effect="plain">{{ preflight.signature.status }}</el-tag></div>
				</div>
				<el-alert v-if="preflight.blockers?.length" type="error" show-icon :closable="false" :title="preflight.blockers.join('; ')" />
				<el-alert v-if="preflight.warnings?.length" type="warning" show-icon :closable="false" :title="preflight.warnings.join('; ')" />
				<el-descriptions class="responsive-descriptions" :column="3" border>
					<el-descriptions-item :label="t('plugin.advanced.preflight.permissionDiff')">{{ permissionChanges }}</el-descriptions-item>
					<el-descriptions-item :label="t('plugin.advanced.preflight.hostCapabilities')">{{ preflight.resources.hostCapabilities?.length ?? 0 }}</el-descriptions-item>
					<el-descriptions-item :label="t('plugin.advanced.preflight.configFields')">{{ preflight.config.fieldCount }}</el-descriptions-item>
					<el-descriptions-item :label="t('plugin.advanced.preflight.migrations')">{{ preflight.migration.pending?.length ?? 0 }}</el-descriptions-item>
				</el-descriptions>
				<div v-if="preflight.resources.hostCapabilities?.length" class="capability-list">
					<strong>{{ t("plugin.advanced.preflight.hostCapabilities") }}</strong>
					<div class="preflight-tags"><el-tag v-for="capability in preflight.resources.hostCapabilities" :key="capability" effect="plain">{{ capability }}</el-tag></div>
				</div>
				<div class="install-command"><el-button type="primary" :icon="PackagePlus" :disabled="!canInstall" @click="install">{{ t("plugin.action.install") }}</el-button></div>
			</section>
		</div>
	</PageShell>
</template>

<style scoped>
.install-workflow, .preflight-result { display: grid; gap: 16px; }
.install-input { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: 8px; }
.preflight-result { padding-top: 16px; border-top: 1px solid var(--color-border); }
.preflight-heading { display: flex; justify-content: space-between; gap: 16px; }
.preflight-heading h3, .preflight-heading p { margin: 0; }
.preflight-heading p { margin-top: 4px; color: var(--color-text-muted); }
.capability-list { display: grid; gap: 8px; }
.preflight-tags, .install-command { display: flex; gap: 8px; flex-wrap: wrap; }
.install-command { justify-content: flex-end; }
@media (max-width: 620px) { .install-input, .preflight-heading { grid-template-columns: 1fr; display: grid; } }
</style>

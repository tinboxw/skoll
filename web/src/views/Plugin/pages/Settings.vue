<script setup lang="ts">
import { computed, ref, watch } from "vue";

import SchemaForm from "../../../components/Common/SchemaForm.vue";
import StateBlock from "../../../components/Common/StateBlock.vue";
import { useI18n } from "../../../i18n";
import { applyConfigDefaults, normalizePluginConfigSchema } from "../../../plugins/config-schema";
import { toErrorMessage } from "../../../utils/common";
import { getPluginConfig, savePluginConfig } from "../api";
import { usePluginWorkspace } from "../workspace";

const workspace = usePluginWorkspace();
const { t, locale } = useI18n();
const loading = ref(false);
const saving = ref(false);
const error = ref("");
const valid = ref(true);
const config = ref<Record<string, unknown>>({});
const schema = computed(() => normalizePluginConfigSchema(workspace.snapshot.value?.plugin.configSchema) ?? null);

async function load(): Promise<void> {
	loading.value = true;
	error.value = "";
	try {
		const result = await getPluginConfig(workspace.pluginId.value);
		config.value = applyConfigDefaults(result.config, normalizePluginConfigSchema(result.configSchema) ?? schema.value);
	} catch (reason) { error.value = toErrorMessage(reason); }
	finally { loading.value = false; }
}

async function save(): Promise<void> {
	if (!valid.value) return;
	saving.value = true;
	error.value = "";
	try { await savePluginConfig(workspace.pluginId.value, config.value); }
	catch (reason) { error.value = toErrorMessage(reason); }
	finally { saving.value = false; }
}

watch(() => workspace.pluginId.value, load, { immediate: true });
</script>

<template>
	<section class="settings-surface" data-testid="plugin-settings">
		<el-alert v-if="error" type="error" :closable="false" show-icon :title="error" />
		<el-skeleton v-if="loading" :rows="6" animated />
		<StateBlock v-else-if="!schema || (schema.fields?.length ?? 0) === 0" type="empty" :description="t('plugin.advanced.detail.noConfigSchema')" />
		<template v-else>
			<SchemaForm v-model="config" v-model:valid="valid" :schema="schema" :locale="locale" :disabled="saving" />
			<div class="form-actions"><el-button type="primary" :loading="saving" :disabled="!valid" @click="save">{{ t("common.save") }}</el-button></div>
		</template>
	</section>
</template>

<style scoped>
.settings-surface { display: grid; gap: 16px; max-width: 920px; }
.form-actions { display: flex; justify-content: flex-end; }
</style>

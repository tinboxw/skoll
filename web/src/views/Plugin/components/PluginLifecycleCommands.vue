<script setup lang="ts">
import { computed } from "vue";
import { CirclePlay, Pause, Trash2 } from "lucide-vue-next";
import { useRouter } from "vue-router";

import { confirmAction } from "../../../composables/useConfirmAction";
import { useI18n } from "../../../i18n";
import { useButtonAccess, BUTTON_ACCESS } from "../../../permissions/button";
import { usePluginWorkspace } from "../workspace";

const workspace = usePluginWorkspace();
const router = useRouter();
const { t } = useI18n();
const access = useButtonAccess();
const plugin = computed(() => workspace.snapshot.value?.plugin ?? null);
const state = computed(() => workspace.snapshot.value?.runtime.state ?? "");
const canEnable = computed(() => access.can(BUTTON_ACCESS.pluginEnable));
const canDisable = computed(() => access.can(BUTTON_ACCESS.pluginDisable));
const canUninstall = computed(() => access.can(BUTTON_ACCESS.pluginUninstall));

async function execute(action: "enable" | "disable" | "uninstall"): Promise<void> {
	const accepted = action === "enable" || await confirmAction({
		title: t("plugin.table.actions"),
		message: `${t(`plugin.action.${action}`)}: ${workspace.pluginId.value}`,
		confirmText: t(`plugin.action.${action}`),
		cancelText: t("common.cancel"),
		type: action === "uninstall" ? "error" : "warning",
		danger: true
	});
	if (!accepted) return;
	await workspace.runLifecycle(action);
	if (action === "uninstall") await router.replace("/skoll/plugin-center");
}
</script>

<template>
	<div v-if="plugin && (canEnable || canDisable || canUninstall)" class="lifecycle-commands" data-testid="plugin-lifecycle-commands">
		<el-button v-if="state !== 'enabled' && canEnable" :icon="CirclePlay" type="success" :loading="workspace.operating.value" @click="execute('enable')">
			{{ t("plugin.action.enable") }}
		</el-button>
		<el-button v-else-if="!plugin.systemBuiltin && canDisable" :icon="Pause" :loading="workspace.operating.value" @click="execute('disable')">
			{{ t("plugin.action.disable") }}
		</el-button>
		<el-button v-if="!plugin.systemBuiltin && canUninstall" :icon="Trash2" type="danger" plain :loading="workspace.operating.value" @click="execute('uninstall')">
			{{ t("plugin.action.uninstall") }}
		</el-button>
	</div>
</template>

<style scoped>
.lifecycle-commands { display: flex; gap: 8px; flex-wrap: wrap; }
</style>

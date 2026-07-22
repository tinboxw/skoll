import { computed, inject, provide, ref, watch, type ComputedRef, type InjectionKey, type Ref } from "vue";

import { getPluginControl, runPluginLifecycle, type PluginControlSnapshot } from "./api";
import { deriveRuntimeState, type PluginRuntimeState } from "./model";
import { toErrorMessage } from "../../utils/common";

export type PluginWorkspaceContext = {
	pluginId: Ref<string>;
	snapshot: Ref<PluginControlSnapshot | null>;
	loading: Ref<boolean>;
	operating: Ref<boolean>;
	error: Ref<string>;
	runtimeState: ComputedRef<PluginRuntimeState>;
	refresh: () => Promise<void>;
	runLifecycle: (action: "enable" | "disable" | "uninstall") => Promise<void>;
};

const workspaceKey: InjectionKey<PluginWorkspaceContext> = Symbol("plugin-workspace");

export function providePluginWorkspace(pluginId: Ref<string>): PluginWorkspaceContext {
	const snapshot = ref<PluginControlSnapshot | null>(null);
	const loading = ref(false);
	const operating = ref(false);
	const error = ref("");

	async function refresh(): Promise<void> {
		if (!pluginId.value) return;
		loading.value = true;
		error.value = "";
		try {
			snapshot.value = await getPluginControl(pluginId.value);
		} catch (reason) {
			snapshot.value = null;
			error.value = toErrorMessage(reason);
		} finally {
			loading.value = false;
		}
	}

	async function runLifecycle(action: "enable" | "disable" | "uninstall"): Promise<void> {
		operating.value = true;
		error.value = "";
		try {
			await runPluginLifecycle(pluginId.value, action);
			if (action !== "uninstall") await refresh();
		} catch (reason) {
			error.value = toErrorMessage(reason);
			throw reason;
		} finally {
			operating.value = false;
		}
	}

	const context: PluginWorkspaceContext = {
		pluginId,
		snapshot,
		loading,
		operating,
		error,
		runtimeState: computed(() => deriveRuntimeState(snapshot.value)),
		refresh,
		runLifecycle
	};
	provide(workspaceKey, context);
	watch(pluginId, refresh, { immediate: true });
	return context;
}

export function usePluginWorkspace(): PluginWorkspaceContext {
	const context = inject(workspaceKey);
	if (!context) throw new Error("plugin workspace context is unavailable");
	return context;
}

import { ref, watch, type Ref } from "vue";

import { toErrorMessage } from "../../utils/common";
import { getPluginDataControl, rollbackPluginMigration, type PluginDataControlSnapshot, type PluginMigrationRollbackResult } from "./api";

export function usePluginDataControl(pluginId: Ref<string>) {
	const snapshot = ref<PluginDataControlSnapshot | null>(null);
	const rollbackResult = ref<PluginMigrationRollbackResult | null>(null);
	const loading = ref(false);
	const operating = ref(false);
	const error = ref("");

	async function refresh(): Promise<void> {
		if (!pluginId.value) return;
		loading.value = true;
		error.value = "";
		try {
			snapshot.value = await getPluginDataControl(pluginId.value);
		} catch (reason) {
			snapshot.value = null;
			error.value = toErrorMessage(reason);
		} finally {
			loading.value = false;
		}
	}

	async function rollback(limit: number): Promise<void> {
		operating.value = true;
		error.value = "";
		try {
			const result = await rollbackPluginMigration(pluginId.value, limit);
			rollbackResult.value = result;
			snapshot.value = result.snapshot;
		} catch (reason) {
			error.value = toErrorMessage(reason);
			throw reason;
		} finally {
			operating.value = false;
		}
	}

	watch(pluginId, refresh, { immediate: true });
	return { snapshot, rollbackResult, loading, operating, error, refresh, rollback };
}

export function formatDataSize(bytes: number, known: boolean): string {
	if (!known) return "-";
	if (bytes < 1024) return `${bytes} B`;
	if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
	if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
	return `${(bytes / 1024 / 1024 / 1024).toFixed(1)} GB`;
}

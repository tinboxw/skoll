import { ref, type Ref } from "vue";

import { toErrorMessage } from "../../utils/common";
import {
	getPluginDiagnostics,
	retryPluginDeadLetter,
	type PluginDiagnosticQuery,
	type PluginDiagnosticsSnapshot,
	type PluginJobRetryResult
} from "./api";

export function usePluginDiagnostics(pluginId: Ref<string>) {
	const snapshot = ref<PluginDiagnosticsSnapshot | null>(null);
	const loading = ref(false);
	const operating = ref(false);
	const error = ref("");
	const retryResult = ref<PluginJobRetryResult | null>(null);

	async function refresh(query: PluginDiagnosticQuery = {}): Promise<void> {
		if (!pluginId.value) return;
		loading.value = true;
		error.value = "";
		try {
			snapshot.value = await getPluginDiagnostics(pluginId.value, query);
		} catch (cause) {
			error.value = toErrorMessage(cause);
		} finally {
			loading.value = false;
		}
	}

	async function retry(jobId: string, query: PluginDiagnosticQuery = {}): Promise<void> {
		operating.value = true;
		error.value = "";
		try {
			retryResult.value = await retryPluginDeadLetter(pluginId.value, jobId);
			await refresh(query);
		} catch (cause) {
			error.value = toErrorMessage(cause);
		} finally {
			operating.value = false;
		}
	}

	return { snapshot, loading, operating, error, retryResult, refresh, retry };
}

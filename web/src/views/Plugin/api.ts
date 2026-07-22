import type { BackendPluginRecord, PluginConfigSchema } from "../../plugins/types";
import type { ApiResponse } from "../../utils/api";
import { apiDelete, apiGet, apiPost, apiPut } from "../../utils/api";

export type PluginHealthReport = {
	pluginId: string;
	status: "healthy" | "unhealthy" | "not_applicable";
	code: string;
	checkedAt: string;
	latencyMillis: number;
	httpStatus?: number;
};

export type PluginControlRoute = {
	method: string;
	path: string;
	summary?: string;
	permission?: string;
	auditAction?: string;
	source: string;
};

export type PluginControlPermission = {
	key: string;
	type?: string;
	module?: string;
	name?: string;
	risk?: string;
};

export type PluginControlSnapshot = {
	plugin: BackendPluginRecord;
	capturedAt: string;
	staleAfter: string;
	runtime: {
		state: string;
		installedAt: string;
		enabledAt?: string;
		serviceBaseUrl?: string;
		serviceHealthUrl?: string;
		health?: PluginHealthReport;
	};
	capabilities: {
		apiVersion?: string;
		migrationVersion?: string;
		hostServices: string[];
		permissions: PluginControlPermission[];
		routes: PluginControlRoute[];
		dependencies: Array<{ id: string; version?: string }>;
		extensions: {
			routes: number;
			middlewares: number;
			events: number;
			menus: number;
			widgets: number;
			settings: number;
		};
	};
};

export type PluginDataControlSnapshot = {
	pluginId: string;
	capturedAt: string;
	state: string;
	schema: {
		available: boolean;
		registered: boolean;
		namespace?: string;
		tables: Array<{
			logicalName: string;
			physicalName: string;
			fields: string[];
			primaryKey: string[];
			indexCount: number;
			exists: boolean;
			sizeBytes: number;
			sizeKnown: boolean;
		}>;
		totalSizeBytes: number;
		sizeKnown: boolean;
	};
	migration: {
		declaredVersion?: string;
		currentVersion: number;
		applied: PluginMigrationStep[];
		pending: PluginMigrationStep[];
		error?: string;
	};
	policy: {
		uninstall?: string;
		rollback?: string;
		effect: string;
	};
	actions: {
		canRollback: boolean;
		rollbackMaxSteps: number;
		blockedReason?: string;
	};
};

export type PluginMigrationStep = {
	version: number;
	name: string;
	checksum: string;
	appliedAt?: string;
};

export type PluginMigrationRollbackResult = {
	operationId: string;
	completedAt: string;
	rolledBackSteps: number;
	snapshot: PluginDataControlSnapshot;
};

export type PluginInstallPreflight = {
	status: "pass" | "blocked";
	plugin: { id: string; name: string; version: string; source: string };
	blockers?: string[];
	warnings?: string[];
	permissions: { add?: unknown[]; update?: unknown[]; conflict?: unknown[] };
	menus: { add?: unknown[]; update?: unknown[]; conflict?: unknown[] };
	config: { hasSchema: boolean; fieldCount: number };
	migration: { version?: string; pending?: unknown[]; applied?: unknown[]; error?: string };
	signature: { status: string; algorithm?: string; vendorId?: string };
	risk: { level: string; summary?: string[] };
};

export type MarketplacePlugin = {
	id: string;
	name: string;
	version: string;
	description?: string;
	manifestPath?: string;
	packagePath?: string;
	installable: boolean;
	signature: { status: string; algorithm?: string };
	risk: { level: string };
};

export async function getPluginControl(pluginId: string): Promise<PluginControlSnapshot> {
	const response = await apiGet<ApiResponse<PluginControlSnapshot>>(`/v1/plugins/${encodeURIComponent(pluginId)}/control`);
	return response.data;
}

export async function getPluginDataControl(pluginId: string): Promise<PluginDataControlSnapshot> {
	const response = await apiGet<ApiResponse<PluginDataControlSnapshot>>(`/v1/plugins/${encodeURIComponent(pluginId)}/data-control`);
	return response.data;
}

export async function rollbackPluginMigration(pluginId: string, limit: number): Promise<PluginMigrationRollbackResult> {
	const response = await apiPost<ApiResponse<PluginMigrationRollbackResult>>(`/v1/plugins/${encodeURIComponent(pluginId)}/migrations/rollback`, {
		limit,
		confirmPluginId: pluginId
	});
	return response.data;
}

export async function runPluginLifecycle(pluginId: string, action: "enable" | "disable" | "uninstall"): Promise<void> {
	const path = `/v1/plugins/${encodeURIComponent(pluginId)}`;
	if (action === "uninstall") {
		await apiDelete<ApiResponse<unknown>>(path);
		return;
	}
	await apiPost<ApiResponse<unknown>>(`${path}/${action}`);
}

export async function validatePluginInstall(path: string): Promise<PluginInstallPreflight> {
	const response = await apiPost<ApiResponse<PluginInstallPreflight>>("/v1/plugins/preflight", { path });
	return response.data;
}

export async function installPlugin(path: string): Promise<void> {
	await apiPost<ApiResponse<unknown>>("/v1/plugins/install", { path });
}

export async function listMarketplace(pluginsRoot = "plugins"): Promise<MarketplacePlugin[]> {
	const query = new URLSearchParams({ pluginsRoot });
	const response = await apiGet<ApiResponse<{ items?: MarketplacePlugin[] }>>(`/v1/plugins/marketplace/local?${query.toString()}`);
	return response.data?.items ?? [];
}

export async function getPluginConfig(pluginId: string): Promise<{ config: Record<string, unknown>; configSchema: PluginConfigSchema | null }> {
	const response = await apiGet<ApiResponse<{ config?: Record<string, unknown>; configSchema?: PluginConfigSchema | null }>>(`/v1/plugins/${encodeURIComponent(pluginId)}/config`);
	return { config: response.data?.config ?? {}, configSchema: response.data?.configSchema ?? null };
}

export async function savePluginConfig(pluginId: string, config: Record<string, unknown>): Promise<void> {
	await apiPut<ApiResponse<unknown>>(`/v1/plugins/${encodeURIComponent(pluginId)}/config`, { config });
}

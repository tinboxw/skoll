import { apiGet, apiPost, type ApiResponse } from "../utils/api";

export type PermissionResourceType = "api" | "menu" | "button" | "data_scope" | "plugin";
export type PermissionRisk = "low" | "medium" | "high" | "critical";

export type PermissionResource = {
	key: string;
	type: PermissionResourceType;
	module: string;
	source: string;
	name: string;
	risk: PermissionRisk;
	metadata: Record<string, string>;
	enabled: boolean;
};

export type PermissionResourceInput = {
	key: string;
	type: PermissionResourceType;
	module: string;
	source: string;
	name: string;
	risk?: PermissionRisk;
	metadata?: Record<string, string>;
};

export type PermissionListQuery = {
	type?: PermissionResourceType;
	source?: string;
	enabled?: boolean;
	offset?: number;
	limit?: number;
};

export type PermissionListResult = {
	items: PermissionResource[];
	offset: number;
	limit: number;
};

export type PermissionDetailResult = {
	item: PermissionResource;
};

export type PermissionStateResult = {
	key: string;
	enabled: boolean;
};

export type PermissionDiffRequest = {
	source: string;
	desired: PermissionResourceInput[];
};

export type PermissionDiffResult = {
	added: PermissionResource[];
	updated: PermissionResource[];
	removed: PermissionResource[];
};

export async function listPermissionResources(query: PermissionListQuery = {}): Promise<PermissionListResult> {
	const payload = await apiGet<ApiResponse<PermissionListResult>>(`/v1/permissions${buildPermissionQuery(query)}`);
	return normalizePermissionListResult(payload.data);
}

export async function getPermissionResource(key: string): Promise<PermissionResource> {
	const payload = await apiGet<ApiResponse<PermissionDetailResult>>(`/v1/permissions/${encodeURIComponent(key)}`);
	return normalizePermissionResource(payload.data.item);
}

export async function enablePermissionResource(key: string): Promise<PermissionStateResult> {
	const payload = await apiPost<ApiResponse<PermissionStateResult>>(`/v1/permissions/${encodeURIComponent(key)}/enable`);
	return normalizePermissionState(payload.data);
}

export async function disablePermissionResource(key: string): Promise<PermissionStateResult> {
	const payload = await apiPost<ApiResponse<PermissionStateResult>>(`/v1/permissions/${encodeURIComponent(key)}/disable`);
	return normalizePermissionState(payload.data);
}

export async function diffPermissionResources(request: PermissionDiffRequest): Promise<PermissionDiffResult> {
	const payload = await apiPost<ApiResponse<PermissionDiffResult>>("/v1/permissions/diff", {
		source: request.source,
		desired: request.desired.map(normalizePermissionInput)
	});
	return {
		added: normalizePermissionResources(payload.data.added),
		updated: normalizePermissionResources(payload.data.updated),
		removed: normalizePermissionResources(payload.data.removed)
	};
}

function buildPermissionQuery(query: PermissionListQuery): string {
	const params = new URLSearchParams();
	if (query.type) {
		params.set("type", query.type);
	}
	if (query.source?.trim()) {
		params.set("source", query.source.trim());
	}
	if (typeof query.enabled === "boolean") {
		params.set("enabled", String(query.enabled));
	}
	if (typeof query.offset === "number" && Number.isFinite(query.offset)) {
		params.set("offset", String(Math.max(0, Math.trunc(query.offset))));
	}
	if (typeof query.limit === "number" && Number.isFinite(query.limit)) {
		params.set("limit", String(Math.max(0, Math.trunc(query.limit))));
	}
	const raw = params.toString();
	return raw ? `?${raw}` : "";
}

function normalizePermissionListResult(raw: PermissionListResult): PermissionListResult {
	return {
		items: normalizePermissionResources(raw.items),
		offset: Number.isFinite(raw.offset) ? raw.offset : 0,
		limit: Number.isFinite(raw.limit) ? raw.limit : 0
	};
}

function normalizePermissionResources(items: PermissionResource[] | undefined): PermissionResource[] {
	return Array.isArray(items) ? items.map(normalizePermissionResource) : [];
}

function normalizePermissionResource(raw: PermissionResource): PermissionResource {
	return {
		key: String(raw.key ?? "").trim(),
		type: normalizePermissionType(raw.type),
		module: String(raw.module ?? "").trim(),
		source: String(raw.source ?? "").trim(),
		name: String(raw.name ?? "").trim(),
		risk: normalizePermissionRisk(raw.risk),
		metadata: normalizeMetadata(raw.metadata),
		enabled: Boolean(raw.enabled)
	};
}

function normalizePermissionInput(raw: PermissionResourceInput): PermissionResourceInput {
	return {
		key: raw.key.trim(),
		type: raw.type,
		module: raw.module.trim(),
		source: raw.source.trim(),
		name: raw.name.trim(),
		risk: raw.risk,
		metadata: normalizeMetadata(raw.metadata ?? {})
	};
}

function normalizePermissionState(raw: PermissionStateResult): PermissionStateResult {
	return {
		key: String(raw.key ?? "").trim(),
		enabled: Boolean(raw.enabled)
	};
}

function normalizeMetadata(raw: Record<string, string> | undefined): Record<string, string> {
	const out: Record<string, string> = {};
	for (const [key, value] of Object.entries(raw ?? {})) {
		const normalizedKey = key.trim().toLowerCase();
		if (normalizedKey !== "") {
			out[normalizedKey] = String(value).trim();
		}
	}
	return out;
}

function normalizePermissionType(type: PermissionResourceType): PermissionResourceType {
	switch (type) {
		case "menu":
		case "button":
		case "data_scope":
		case "plugin":
			return type;
		default:
			return "api";
	}
}

function normalizePermissionRisk(risk: PermissionRisk): PermissionRisk {
	switch (risk) {
		case "medium":
		case "high":
		case "critical":
			return risk;
		default:
			return "low";
	}
}

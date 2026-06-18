import { ApiError, apiGet, type ApiResponse } from "../utils/api";
import { API_BASE_PREFIX } from "../utils/api-base-prefix";
import { getToken } from "../utils/auth";

export type AuditEventType = "operation" | "login" | "error" | "plugin" | "security";
export type AuditEventResult = "success" | "failure" | "denied";
export type AuditEventRisk = "low" | "medium" | "high" | "critical";

export type AuditRef = {
	type: string;
	id: string;
	name?: string;
};

export type AuditTraceContext = {
	traceId?: string;
	requestId?: string;
	method?: string;
	path?: string;
	ip?: string;
	userAgent?: string;
};

export type AuditEvent = {
	id: string;
	type: AuditEventType;
	action: string;
	actor: AuditRef;
	resource: AuditRef;
	result: AuditEventResult;
	risk: AuditEventRisk;
	trace: AuditTraceContext;
	metadata?: Record<string, unknown>;
	occurredAt: string;
};

export type AuditEventDiff = {
	before?: Record<string, unknown>;
	after?: Record<string, unknown>;
	summary?: string[];
};

export type AuditEventDetail = AuditEvent & {
	sourceData?: Record<string, unknown>;
	diff?: AuditEventDiff;
};

export type AuditEventListQuery = {
	type?: AuditEventType;
	actorId?: string;
	action?: string;
	resourceType?: string;
	resourceId?: string;
	result?: AuditEventResult;
	risk?: AuditEventRisk;
	from?: string;
	to?: string;
	offset?: number;
	limit?: number;
};

export type AuditEventListResult = {
	items: AuditEvent[];
	offset: number;
	limit: number;
	total?: number;
};

export type AuditEventDetailResult = {
	item: AuditEventDetail;
};

export async function listAuditEvents(query: AuditEventListQuery = {}): Promise<AuditEventListResult> {
	const payload = await apiGet<ApiResponse<AuditEventListResult>>(`/v1/audit${buildAuditQuery(query)}`);
	return normalizeAuditEventListResult(payload.data);
}

export async function getAuditEvent(id: string): Promise<AuditEventDetail> {
	const payload = await apiGet<ApiResponse<AuditEventDetailResult>>(`/v1/audit/${encodeURIComponent(id)}`);
	return normalizeAuditEventDetail(payload.data.item);
}

export async function exportAuditEvents(query: AuditEventListQuery = {}): Promise<Blob> {
	const response = await fetch(`${API_BASE_PREFIX}/v1/audit/export${buildAuditQuery(query)}`, {
		method: "GET",
		headers: buildAuthHeaders()
	});
	if (!response.ok) {
		throw await apiErrorFromResponse(response);
	}
	return response.blob();
}

export function buildAuditQuery(query: AuditEventListQuery): string {
	const params = new URLSearchParams();
	setStringParam(params, "type", query.type);
	setStringParam(params, "actorId", query.actorId);
	setStringParam(params, "action", query.action);
	setStringParam(params, "resourceType", query.resourceType);
	setStringParam(params, "resourceId", query.resourceId);
	setStringParam(params, "result", query.result);
	setStringParam(params, "risk", query.risk);
	setStringParam(params, "from", query.from);
	setStringParam(params, "to", query.to);
	setNumberParam(params, "offset", query.offset, 0);
	setNumberParam(params, "limit", query.limit, 200);
	const raw = params.toString();
	return raw ? `?${raw}` : "";
}

function buildAuthHeaders(): Headers {
	const headers = new Headers();
	const token = getToken().trim();
	if (token !== "") {
		headers.set("Authorization", token.toLowerCase().startsWith("bearer ") ? token : `Bearer ${token}`);
	}
	return headers;
}

async function apiErrorFromResponse(response: Response): Promise<ApiError> {
	let payload: Partial<ApiResponse<unknown>> | null = null;
	try {
		payload = (await response.json()) as Partial<ApiResponse<unknown>>;
	} catch {
		payload = null;
	}
	const message = typeof payload?.message === "string" && payload.message.trim() !== ""
		? payload.message
		: `request failed: ${response.status}`;
	const code = typeof payload?.code === "string" ? payload.code : "";
	return new ApiError(message, response.status, code);
}

function setStringParam(params: URLSearchParams, key: string, value: string | undefined): void {
	const normalized = value?.trim() ?? "";
	if (normalized !== "") {
		params.set(key, normalized);
	}
}

function setNumberParam(params: URLSearchParams, key: string, value: number | undefined, max: number): void {
	if (typeof value !== "number" || !Number.isFinite(value)) {
		return;
	}
	const normalized = Math.max(0, Math.trunc(value));
	params.set(key, String(max > 0 ? Math.min(normalized, max) : normalized));
}

function normalizeAuditEventListResult(raw: AuditEventListResult): AuditEventListResult {
	return {
		items: Array.isArray(raw.items) ? raw.items.map(normalizeAuditEvent) : [],
		offset: Number.isFinite(raw.offset) ? raw.offset : 0,
		limit: Number.isFinite(raw.limit) ? raw.limit : 50,
		total: typeof raw.total === "number" && Number.isFinite(raw.total) ? raw.total : undefined
	};
}

function normalizeAuditEventDetail(raw: AuditEventDetail): AuditEventDetail {
	const event = normalizeAuditEvent(raw);
	return {
		...event,
		sourceData: normalizeUnknownRecord(raw.sourceData),
		diff: normalizeAuditEventDiff(raw.diff)
	};
}

function normalizeAuditEvent(raw: AuditEvent): AuditEvent {
	return {
		id: String(raw.id ?? "").trim(),
		type: normalizeEventType(raw.type),
		action: String(raw.action ?? "").trim(),
		actor: normalizeAuditRef(raw.actor),
		resource: normalizeAuditRef(raw.resource),
		result: normalizeEventResult(raw.result),
		risk: normalizeEventRisk(raw.risk),
		trace: normalizeTrace(raw.trace),
		metadata: normalizeUnknownRecord(raw.metadata),
		occurredAt: String(raw.occurredAt ?? "").trim()
	};
}

function normalizeAuditRef(raw: AuditRef | undefined): AuditRef {
	return {
		type: String(raw?.type ?? "").trim(),
		id: String(raw?.id ?? "").trim(),
		name: String(raw?.name ?? "").trim() || undefined
	};
}

function normalizeTrace(raw: AuditTraceContext | undefined): AuditTraceContext {
	return {
		traceId: String(raw?.traceId ?? "").trim() || undefined,
		requestId: String(raw?.requestId ?? "").trim() || undefined,
		method: String(raw?.method ?? "").trim() || undefined,
		path: String(raw?.path ?? "").trim() || undefined,
		ip: String(raw?.ip ?? "").trim() || undefined,
		userAgent: String(raw?.userAgent ?? "").trim() || undefined
	};
}

function normalizeAuditEventDiff(raw: AuditEventDiff | undefined): AuditEventDiff | undefined {
	if (!raw || typeof raw !== "object") {
		return undefined;
	}
	return {
		before: normalizeUnknownRecord(raw.before),
		after: normalizeUnknownRecord(raw.after),
		summary: Array.isArray(raw.summary) ? raw.summary.map((item) => String(item ?? "")) : undefined
	};
}

function normalizeUnknownRecord(raw: unknown): Record<string, unknown> | undefined {
	if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
		return undefined;
	}
	return raw as Record<string, unknown>;
}

function normalizeEventType(value: unknown): AuditEventType {
	return value === "operation" || value === "login" || value === "error" || value === "plugin" || value === "security"
		? value
		: "operation";
}

function normalizeEventResult(value: unknown): AuditEventResult {
	return value === "success" || value === "failure" || value === "denied" ? value : "success";
}

function normalizeEventRisk(value: unknown): AuditEventRisk {
	return value === "low" || value === "medium" || value === "high" || value === "critical" ? value : "low";
}

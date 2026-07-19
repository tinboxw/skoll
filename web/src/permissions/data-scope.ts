import { apiGet, type ApiResponse } from "../utils/api";

export type AuthorizedDataScope = "self" | "department" | "department_tree" | "all";

export type AuthorizedDataScopeDecision = {
	scope: AuthorizedDataScope;
	all: boolean;
	userIds: string[];
	departmentIds: string[];
};

export async function resolveAuthorizedDataScope(resource: string, action: string): Promise<AuthorizedDataScopeDecision> {
	const params = new URLSearchParams({ resource: resource.trim(), action: action.trim() });
	const payload = await apiGet<ApiResponse<unknown>>(`/v1/rbac/data-scope?${params.toString()}`);
	return normalizeAuthorizedDataScope(payload.data);
}

function normalizeAuthorizedDataScope(value: unknown): AuthorizedDataScopeDecision {
	if (!value || typeof value !== "object") {
		throw new Error("invalid_data_scope_response");
	}
	const row = value as Record<string, unknown>;
	const scope = String(row.scope ?? "").trim();
	if (!isAuthorizedDataScope(scope)) {
		throw new Error("invalid_data_scope_response");
	}
	return {
		scope,
		all: scope === "all" && row.all === true,
		userIds: cleanIDs(row.userIds),
		departmentIds: cleanIDs(row.departmentIds)
	};
}

function isAuthorizedDataScope(value: string): value is AuthorizedDataScope {
	return value === "self" || value === "department" || value === "department_tree" || value === "all";
}

function cleanIDs(value: unknown): string[] {
	if (!Array.isArray(value)) {
		return [];
	}
	return [...new Set(value.filter((item): item is string => typeof item === "string").map((item) => item.trim()).filter(Boolean))];
}

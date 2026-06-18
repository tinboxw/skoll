import { apiGet, apiPatch, apiPost, apiPut, type ApiResponse } from "../utils/api";

export type MenuNodeRecord = {
	key: string;
	parentKey?: string;
	source: string;
	name: string;
	path: string;
	component?: string;
	icon?: string;
	sort: number;
	visible: boolean;
	requiredRoles?: string[];
	requiredPermissions?: string[];
};

export type MenuTreeQuery = {
	parentKey?: string;
	source?: string;
	visible?: boolean;
	roles?: string[];
	permissions?: string[];
};

export type MenuTreeResult = {
	items: MenuNodeRecord[];
};

export type MenuSaveRequest = {
	items: MenuNodeRecord[];
};

export type MenuReorderRequest = {
	parentKey: string;
	orderedKeys: string[];
};

export type MenuReorderResult = {
	parentKey: string;
	orderedKeys: string[];
};

export type MenuVisibilityRequest = {
	key: string;
	visible: boolean;
};

export type MenuVisibilityResult = {
	item: MenuNodeRecord;
};

export async function getMenuTree(query: MenuTreeQuery = {}): Promise<MenuNodeRecord[]> {
	const payload = await apiGet<ApiResponse<MenuTreeResult>>(`/v1/menus/tree${buildMenuQuery(query)}`);
	return normalizeMenuNodes(payload.data.items);
}

export async function saveMenuNodes(items: MenuNodeRecord[]): Promise<MenuNodeRecord[]> {
	const payload = await apiPut<ApiResponse<MenuTreeResult>>("/v1/menus", {
		items: normalizeMenuNodes(items)
	} satisfies MenuSaveRequest);
	return normalizeMenuNodes(payload.data.items);
}

export async function reorderMenuNodes(request: MenuReorderRequest): Promise<MenuReorderResult> {
	const payload = await apiPost<ApiResponse<MenuReorderResult>>("/v1/menus/reorder", {
		parentKey: request.parentKey.trim(),
		orderedKeys: normalizeStringList(request.orderedKeys)
	});
	return {
		parentKey: String(payload.data.parentKey ?? "").trim(),
		orderedKeys: normalizeStringList(payload.data.orderedKeys)
	};
}

export async function setMenuVisibility(request: MenuVisibilityRequest): Promise<MenuNodeRecord> {
	const payload = await apiPatch<ApiResponse<MenuVisibilityResult>>("/v1/menus/visibility", {
		key: request.key.trim(),
		visible: request.visible
	});
	return normalizeMenuNode(payload.data.item);
}

function buildMenuQuery(query: MenuTreeQuery): string {
	const params = new URLSearchParams();
	if (query.parentKey?.trim()) {
		params.set("parentKey", query.parentKey.trim());
	}
	if (query.source?.trim()) {
		params.set("source", query.source.trim());
	}
	if (typeof query.visible === "boolean") {
		params.set("visible", String(query.visible));
	}
	if (query.roles && query.roles.length > 0) {
		params.set("roles", normalizeStringList(query.roles).join(","));
	}
	if (query.permissions && query.permissions.length > 0) {
		params.set("permissions", normalizeStringList(query.permissions).join(","));
	}
	const raw = params.toString();
	return raw ? `?${raw}` : "";
}

export function normalizeMenuNodes(items: MenuNodeRecord[] | undefined): MenuNodeRecord[] {
	return Array.isArray(items) ? items.map(normalizeMenuNode).filter((item) => item.key !== "" && item.path.startsWith("/")) : [];
}

export function normalizeMenuNode(raw: MenuNodeRecord): MenuNodeRecord {
	return {
		key: String(raw.key ?? "").trim(),
		parentKey: String(raw.parentKey ?? "").trim() || undefined,
		source: String(raw.source ?? "").trim(),
		name: String(raw.name ?? "").trim(),
		path: String(raw.path ?? "").trim(),
		component: String(raw.component ?? "").trim() || undefined,
		icon: String(raw.icon ?? "").trim() || undefined,
		sort: Number.isFinite(raw.sort) ? Number(raw.sort) : 0,
		visible: raw.visible !== false,
		requiredRoles: normalizeStringList(raw.requiredRoles),
		requiredPermissions: normalizeStringList(raw.requiredPermissions)
	};
}

function normalizeStringList(items?: string[]): string[] {
	return Array.isArray(items)
		? items.map((item) => String(item ?? "").trim()).filter((item) => item !== "")
		: [];
}

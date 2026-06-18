import type { RouteLocationNormalized, RouteMeta, RouteRecordRaw } from "vue-router";

import { canAccess, type AccessRule } from "./access";

const AUTH_PLUGIN_PREFIX = "/skoll/plugins/auth";

export type RouteAccessMeta = AccessRule & {
	public?: boolean;
};

export function isPublicRoute(to: Pick<RouteLocationNormalized, "meta" | "path">): boolean {
	return to.meta.public === true || to.path.startsWith(AUTH_PLUGIN_PREFIX);
}

export function canAccessRoute(to: Pick<RouteLocationNormalized, "meta">, currentRole: string, permissions: string[]): boolean {
	const rule = normalizeRouteAccessMeta(to.meta);
	if (rule.roles.length === 0 && rule.permissions.length === 0) {
		return true;
	}
	return canAccess(rule, currentRole, permissions);
}

export function withRouteAccessMeta(route: RouteRecordRaw, meta: RouteAccessMeta): RouteRecordRaw {
	const normalized = normalizeRouteAccessMeta(meta);
	if (normalized.roles.length === 0 && normalized.permissions.length === 0 && meta.public !== true) {
		return route;
	}
	return {
		...route,
		meta: {
			...(route.meta ?? {}),
			...(meta.public === true ? { public: true } : {}),
			...(normalized.roles.length > 0 ? { roles: normalized.roles } : {}),
			...(normalized.permissions.length > 0 ? { permissions: normalized.permissions } : {}),
			mode: normalized.mode
		}
	};
}

export function normalizeRouteAccessMeta(meta: RouteMeta | RouteAccessMeta): Required<AccessRule> {
	return {
		roles: compact(Array.isArray(meta.roles) ? meta.roles : []),
		permissions: compact(Array.isArray(meta.permissions) ? meta.permissions : []),
		mode: meta.mode === "any" ? "any" : "all"
	};
}

function compact(values: string[]): string[] {
	return values.map((item) => String(item || "").trim()).filter((item) => item !== "");
}

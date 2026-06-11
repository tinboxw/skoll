import { computed } from "vue";

import { useUserStore } from "../stores/user";
import { hasPermissionValue } from "./implied";

export type AccessRule = {
	roles?: string[];
	permissions?: string[];
	mode?: "all" | "any";
};

export type AccessDirectiveValue = string | string[] | AccessRule;

export function canAccess(rule: AccessDirectiveValue | null | undefined, currentRole: string, permissions: string[]): boolean {
	if (!rule) {
		return true;
	}

	const normalized = normalizeAccessRule(rule);
	if (currentRole === "super_admin") {
		return true;
	}
	if (normalized.roles.length > 0 && !normalized.roles.includes(currentRole)) {
		return false;
	}
	if (normalized.permissions.length === 0) {
		return true;
	}

	const permissionSet = new Set(permissions);
	if (normalized.mode === "any") {
		return normalized.permissions.some((permission) => hasPermissionValue(permissionSet, permission));
	}
	return normalized.permissions.every((permission) => hasPermissionValue(permissionSet, permission));
}

export function useAccess() {
	const userStore = useUserStore();
	const currentRole = computed(() => userStore.profile?.role ?? "");
	const permissions = computed(() => userStore.permissions);

	return {
		can: (rule: AccessDirectiveValue | null | undefined): boolean => canAccess(rule, currentRole.value, permissions.value),
		currentRole,
		permissions
	};
}

function normalizeAccessRule(rule: AccessDirectiveValue): { roles: string[]; permissions: string[]; mode: "all" | "any" } {
	if (typeof rule === "string") {
		return { roles: [], permissions: compact([rule]), mode: "all" };
	}
	if (Array.isArray(rule)) {
		return { roles: [], permissions: compact(rule), mode: "all" };
	}
	return {
		roles: compact(rule.roles ?? []),
		permissions: compact(rule.permissions ?? []),
		mode: rule.mode === "any" ? "any" : "all"
	};
}

function compact(values: string[]): string[] {
	return values.map((item) => String(item || "").trim()).filter((item) => item !== "");
}

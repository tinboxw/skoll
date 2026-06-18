import { computed } from "vue";

import { useUserStore } from "../stores/user";
import { canAccess, type AccessDirectiveValue } from "./access";

export const BUTTON_ACCESS = {
	pluginRead: "plugin.read",
	pluginManage: "plugin.manage",
	roleCreate: "role.create",
	roleUpdate: "role.update",
	roleDelete: "role.delete",
	roleManage: "role.manage",
	permissionManage: "permission.manage",
	userCreate: "user.create",
	userUpdate: "user.update",
	userDelete: "user.delete"
} as const satisfies Record<string, AccessDirectiveValue>;

export type ButtonAccessKey = keyof typeof BUTTON_ACCESS;
export type ButtonAccessRule = ButtonAccessKey | AccessDirectiveValue;

export function canUseButton(rule: ButtonAccessRule | null | undefined, currentRole: string, permissions: string[]): boolean {
	return canAccess(resolveButtonAccessRule(rule), currentRole, permissions);
}

export function useButtonAccess() {
	const userStore = useUserStore();
	const currentRole = computed(() => userStore.profile?.role ?? "");
	const permissions = computed(() => userStore.permissions);

	return {
		can: (rule: ButtonAccessRule | null | undefined): boolean => canUseButton(rule, currentRole.value, permissions.value),
		disabled: (rule: ButtonAccessRule | null | undefined, baseDisabled = false): boolean => baseDisabled || !canUseButton(rule, currentRole.value, permissions.value),
		currentRole,
		permissions
	};
}

function resolveButtonAccessRule(rule: ButtonAccessRule | null | undefined): AccessDirectiveValue | null | undefined {
	if (typeof rule === "string" && rule in BUTTON_ACCESS) {
		return BUTTON_ACCESS[rule as ButtonAccessKey];
	}
	return rule;
}

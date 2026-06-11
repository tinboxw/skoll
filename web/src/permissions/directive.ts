import type { App, DirectiveBinding } from "vue";

import { canAccess, type AccessDirectiveValue } from "./access";
import { getStoredPermissions, getStoredUserRole } from "../stores/user";

type PermissionElement = HTMLElement & {
	_skollPermissionDisplay?: string;
};

export function installPermissionDirective(app: App): void {
	app.directive("permission", {
		mounted(el: PermissionElement, binding: DirectiveBinding<AccessDirectiveValue>) {
			el._skollPermissionDisplay = el.style.display;
			applyPermissionState(el, binding.value);
		},
		updated(el: PermissionElement, binding: DirectiveBinding<AccessDirectiveValue>) {
			applyPermissionState(el, binding.value);
		}
	});
}

function applyPermissionState(el: PermissionElement, value: AccessDirectiveValue): void {
	const allowed = canAccess(value, getStoredUserRole(), getStoredPermissions());
	el.style.display = allowed ? el._skollPermissionDisplay ?? "" : "none";
}

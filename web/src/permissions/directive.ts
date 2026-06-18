import type { App, DirectiveBinding } from "vue";

import { getStoredPermissions, getStoredUserRole } from "../stores/user";
import { canUseButton, type ButtonAccessRule } from "./button";

type PermissionElement = HTMLElement & {
	_skollPermissionDisplay?: string;
};

export function installPermissionDirective(app: App): void {
	app.directive("permission", {
		mounted(el: PermissionElement, binding: DirectiveBinding<ButtonAccessRule>) {
			el._skollPermissionDisplay = el.style.display;
			applyPermissionState(el, binding.value);
		},
		updated(el: PermissionElement, binding: DirectiveBinding<ButtonAccessRule>) {
			applyPermissionState(el, binding.value);
		}
	});
}

function applyPermissionState(el: PermissionElement, value: ButtonAccessRule): void {
	const allowed = canUseButton(value, getStoredUserRole(), getStoredPermissions());
	el.style.display = allowed ? el._skollPermissionDisplay ?? "" : "none";
}

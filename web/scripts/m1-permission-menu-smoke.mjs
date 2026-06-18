import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { resolve } from "node:path";

const root = resolve(fileURLToPath(new URL("..", import.meta.url)));

const checks = [
	{
		name: "sidebar uses registry-backed navigation store",
		file: "src/stores/navigation.ts",
		include: ["getMenuTree", "menuNodesToSystemMenus", "saveMenuNodes"]
	},
	{
		name: "app sidebar consumes navigation store menus",
		file: "src/App.vue",
		include: ["navigationStore.systemMenus", "buildSidebarItems"]
	},
	{
		name: "route guard uses unified route access",
		file: "src/router/index.ts",
		include: ["canAccessRoute", "isPublicRoute"],
		exclude: ["canAccess({"]
	},
	{
		name: "plugin routes use unified access metadata",
		file: "src/plugins/index.ts",
		include: ["withRouteAccessMeta"]
	},
	{
		name: "button directive uses button access utility",
		file: "src/permissions/directive.ts",
		include: ["canUseButton"]
	},
	{
		name: "views use button access utility",
		file: "src/views/User/list.vue",
		include: ["useButtonAccess", "BUTTON_ACCESS"],
		exclude: ["useAccess", "access.can"]
	},
	{
		name: "menu save requires confirmation",
		file: "src/views/Menu/index.vue",
		include: ["confirmAction", "menu.editor.saveConfirm", "saveSystemMenus"]
	},
	{
		name: "permission matrix uses catalog store and confirmations",
		file: "src/views/Permission/index.vue",
		include: ["usePermissionStore", "permission.grantConfirm", "permission.revokeConfirm", "permission.savePoliciesConfirm"],
		exclude: ["BASE_PERMISSION_CATALOG", "BUILTIN_ROLE_DEFAULTS"]
	}
];

let failures = 0;

for (const check of checks) {
	const path = resolve(root, check.file);
	const content = readFileSync(path, "utf8");
	const missing = (check.include ?? []).filter((needle) => !content.includes(needle));
	const present = (check.exclude ?? []).filter((needle) => content.includes(needle));
	if (missing.length > 0 || present.length > 0) {
		failures += 1;
		console.error(`[FAIL] ${check.name}`);
		for (const needle of missing) {
			console.error(`  missing: ${needle}`);
		}
		for (const needle of present) {
			console.error(`  forbidden: ${needle}`);
		}
		continue;
	}
	console.log(`[PASS] ${check.name}`);
}

if (failures > 0) {
	process.exitCode = 1;
}

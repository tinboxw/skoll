import type { FrontendPluginManifest } from "../plugins/types";
import { canAccess } from "../permissions/access";
import type { SystemMenuRecord } from "../stores/navigation";

const ADMIN_BASE = "/skoll";
const SIDEBAR_ICONS = new Set(["dashboard", "users", "roles", "permissions", "menus", "audit", "plugins", "settings"]);

export type SidebarItem = {
	label: string;
	to: string;
	icon: string;
	order: number;
	requiredRoles?: string[];
	requiredPermissions?: string[];
	source: "system" | "plugin";
};

type BuildSidebarInput = {
	t: (key: string) => string;
	systemMenus: SystemMenuRecord[];
	plugins: FrontendPluginManifest[];
	currentRole: string;
	currentLocale: string;
	permissions: string[];
	resolvePluginLabel: (item: FrontendPluginManifest) => string;
};

const SYSTEM_MENU: Array<Omit<SidebarItem, "label" | "source"> & { labelKey: string }> = [
	{ labelKey: "menu.dashboard", to: `${ADMIN_BASE}/dashboard`, icon: "dashboard", order: 10 },
	{ labelKey: "menu.users", to: `${ADMIN_BASE}/user`, icon: "users", order: 20, requiredPermissions: ["user.read"] },
	{ labelKey: "menu.roles", to: `${ADMIN_BASE}/role`, icon: "roles", order: 30, requiredPermissions: ["role.read"] },
	{ labelKey: "menu.permissions", to: `${ADMIN_BASE}/permission`, icon: "permissions", order: 40, requiredPermissions: ["permission.manage"] },
	{ labelKey: "menu.menus", to: `${ADMIN_BASE}/menu`, icon: "menus", order: 45, requiredPermissions: ["system.manage"] },
	{ labelKey: "menu.dictionaries", to: `${ADMIN_BASE}/dictionary`, icon: "settings", order: 47, requiredPermissions: ["dict.read"] },
	{ labelKey: "menu.organization", to: `${ADMIN_BASE}/organization`, icon: "users", order: 48, requiredPermissions: ["org.read"] },
	{ labelKey: "menu.audit", to: `${ADMIN_BASE}/audit`, icon: "audit", order: 50, requiredPermissions: ["audit.read"] },
	{ labelKey: "menu.plugins", to: `${ADMIN_BASE}/plugin`, icon: "plugins", order: 60, requiredPermissions: ["plugin.read"] },
	{ labelKey: "menu.settings", to: `${ADMIN_BASE}/setting`, icon: "settings", order: 70, requiredPermissions: ["system.manage"] }
];

export function buildSidebarItems(input: BuildSidebarInput): SidebarItem[] {
	const systemItems = input.systemMenus.length > 0
		? flattenSystemMenus(input.systemMenus, input.t)
		: SYSTEM_MENU.map((item): SidebarItem => ({
			label: input.t(item.labelKey),
			to: item.to,
			icon: item.icon,
			order: item.order,
			requiredRoles: item.requiredRoles,
			requiredPermissions: item.requiredPermissions,
			source: "system"
		}));

	const pluginItems = input.plugins
		.filter((item) => item.enabled !== false && item.uiMode !== "backend_only" && item.uiNavPosition === "sidebar" && item.uiOpenMode !== "standalone")
		.map((item, index): SidebarItem => ({
			label: resolvePluginMenuLabel(item, input.currentLocale, input.resolvePluginLabel),
			to: resolvePluginMenuPath(item),
			icon: resolvePluginMenuIcon(item.uiMenu?.icon),
			order: typeof item.uiMenu?.order === "number" ? item.uiMenu.order : 1000 + index,
			requiredRoles: item.uiMenu?.requiredRoles,
			requiredPermissions: item.uiMenu?.requiredPermissions,
			source: "plugin"
		}));

	return [...systemItems, ...pluginItems]
		.filter((item) => isMenuAllowed(item, input.currentRole, input.permissions))
		.sort((a, b) => a.order - b.order || a.label.localeCompare(b.label));
}

function flattenSystemMenus(items: SystemMenuRecord[], t: (key: string) => string, baseOrder = 0): SidebarItem[] {
	const out: SidebarItem[] = [];
	for (const item of items) {
		if (item.visible === false) {
			continue;
		}
		out.push({
			label: resolveSystemMenuLabel(item, t),
			to: item.path,
			icon: resolvePluginMenuIcon(item.icon),
			order: baseOrder + item.order,
			requiredRoles: item.requiredRoles,
			requiredPermissions: item.requiredPermissions,
			source: "system"
		});
		if (Array.isArray(item.children) && item.children.length > 0) {
			out.push(...flattenSystemMenus(item.children, t, baseOrder + item.order + 0.01));
		}
	}
	return out;
}

function resolveSystemMenuLabel(item: SystemMenuRecord, t: (key: string) => string): string {
	const translated = t(`menu.${item.id}`);
	if (translated !== `menu.${item.id}`) {
		return translated;
	}
	return item.label;
}

function resolvePluginMenuLabel(item: FrontendPluginManifest, currentLocale: string, resolvePluginLabel: (item: FrontendPluginManifest) => string): string {
	const zh = String(item.uiMenu?.labelZhCN || "").trim();
	const en = String(item.uiMenu?.labelEnUS || "").trim();
	if (currentLocale === "zh-CN" && zh) {
		return zh;
	}
	if (currentLocale === "en-US" && en) {
		return en;
	}
	const menuLabel = String(item.uiMenu?.label || "").trim();
	return menuLabel || resolvePluginLabel(item);
}

function resolvePluginMenuPath(item: FrontendPluginManifest): string {
	const menuPath = String(item.uiMenu?.path || "").trim();
	if (menuPath !== "") {
		return menuPath;
	}
	return typeof item.entryPath === "string" && item.entryPath.trim() !== "" ? item.entryPath : `${ADMIN_BASE}/plugins/${item.id}`;
}

function resolvePluginMenuIcon(icon?: string): string {
	const value = String(icon || "").trim();
	return SIDEBAR_ICONS.has(value) ? value : "plugins";
}

function isMenuAllowed(item: SidebarItem, currentRole: string, permissions: string[]): boolean {
	return canAccess({
		roles: item.requiredRoles,
		permissions: item.requiredPermissions
	}, currentRole, permissions);
}

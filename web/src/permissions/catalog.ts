export type PermissionCatalogItem = {
	key: string;
	resource: string;
	action: string;
	label: string;
};

export const BUILTIN_ROLE_DEFAULTS: Record<string, string[]> = {
	super_admin: ["*", "user.read", "user.create", "user.update", "user.delete", "role.read", "role.create", "role.update", "role.delete", "role.manage", "permission.manage", "plugin.read", "plugin.manage", "audit.read", "system.read", "system.manage", "dict.read", "dict.manage", "org.read", "org.manage"],
	dept_admin: ["user.read", "user.update", "role.read", "plugin.read", "audit.read", "dict.read", "org.read"],
	operator: ["user.read", "role.read", "plugin.read", "dict.read", "org.read"],
	user: ["user.read"]
};

export const BASE_PERMISSION_CATALOG: PermissionCatalogItem[] = [
	{ key: "user.read", resource: "user", action: "read", label: "User Read" },
	{ key: "user.create", resource: "user", action: "create", label: "User Create" },
	{ key: "user.update", resource: "user", action: "update", label: "User Update" },
	{ key: "user.delete", resource: "user", action: "delete", label: "User Delete" },
	{ key: "role.read", resource: "role", action: "read", label: "Role Read" },
	{ key: "role.create", resource: "role", action: "create", label: "Role Create" },
	{ key: "role.update", resource: "role", action: "update", label: "Role Update" },
	{ key: "role.delete", resource: "role", action: "delete", label: "Role Delete" },
	{ key: "role.manage", resource: "role", action: "manage", label: "Role Manage" },
	{ key: "permission.manage", resource: "permission", action: "manage", label: "Permission Manage" },
	{ key: "plugin.read", resource: "plugin", action: "read", label: "Plugin Read" },
	{ key: "plugin.manage", resource: "plugin", action: "manage", label: "Plugin Manage" },
	{ key: "plugin.install", resource: "plugin", action: "install", label: "Plugin Install" },
	{ key: "plugin.enable", resource: "plugin", action: "enable", label: "Plugin Enable" },
	{ key: "plugin.disable", resource: "plugin", action: "disable", label: "Plugin Disable" },
	{ key: "plugin.uninstall", resource: "plugin", action: "uninstall", label: "Plugin Uninstall" },
	{ key: "audit.read", resource: "audit", action: "read", label: "Audit Read" },
	{ key: "system.read", resource: "system", action: "read", label: "System Read" },
	{ key: "system.manage", resource: "system", action: "manage", label: "System Manage" },
	{ key: "dict.read", resource: "dictionary", action: "read", label: "Dictionary Read" },
	{ key: "dict.manage", resource: "dictionary", action: "manage", label: "Dictionary Manage" },
	{ key: "org.read", resource: "organization", action: "read", label: "Organization Read" },
	{ key: "org.manage", resource: "organization", action: "manage", label: "Organization Manage" }
];

export const BASE_PERMISSION_OPTIONS = BASE_PERMISSION_CATALOG.map((item) => item.key);

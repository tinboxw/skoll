const IMPLIED_PERMISSIONS: Record<string, string[]> = {
	"user.read": ["*", "user.create", "user.update", "user.delete"],
	"user.create": ["*"],
	"user.update": ["*"],
	"user.delete": ["*"],
	"role.read": ["*", "role.create", "role.update", "role.delete", "role.manage"],
	"role.create": ["*", "role.manage"],
	"role.update": ["*", "role.manage"],
	"role.delete": ["*", "role.manage"],
	"role.manage": ["*"],
	"permission.manage": ["*"],
	"plugin.read": ["*", "plugin.manage"],
	"plugin.manage": ["*"],
	"audit.read": ["*"],
	"system.read": ["*", "system.manage"],
	"system.manage": ["*"],
	"dict.read": ["*", "dict.manage"],
	"dict.manage": ["*"],
	"org.read": ["*", "org.manage"],
	"org.manage": ["*"]
};

export function hasPermissionValue(permissions: string[] | Set<string>, permission: string): boolean {
	const required = permission.trim();
	if (required === "") {
		return true;
	}

	const permissionSet = permissions instanceof Set ? permissions : new Set(permissions);
	if (permissionSet.has(required)) {
		return true;
	}
	return (IMPLIED_PERMISSIONS[required] ?? []).some((item) => permissionSet.has(item));
}

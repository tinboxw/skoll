package gormrepo

func AllModels() []any {
	return []any{
		&UserModel{},
		&RoleModel{},
		&BindingModel{},
		&PolicyRuleModel{},
		&SystemSettingModel{},
		&PluginModel{},
		&PluginRouteModel{},
		&PluginReleaseModel{},
		&PermissionResourceModel{},
		&MenuNodeModel{},
		&AuditRecordModel{},
		&AuditEventModel{},
	}
}

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
		&FileObjectModel{},
		&DictionaryTypeModel{},
		&DictionaryItemModel{},
		&DepartmentModel{},
		&PositionModel{},
		&UserAssignmentModel{},
		&AuditRecordModel{},
		&AuditEventModel{},
	}
}

package rbac

func Allows(binding Binding, resource, action string) bool {
	for _, p := range binding.Permissions {
		if p.Resource == resource && p.Action == action {
			return true
		}
	}
	return false
}

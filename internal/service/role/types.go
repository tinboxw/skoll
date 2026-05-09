package role

type CreateRoleInput struct {
	Name        string
	Key         string
	Description string
	Permissions []string
	BuiltIn     bool
}

type ListInput struct {
	Offset int
	Limit  int
}

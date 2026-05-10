package role

type CreateRoleInput struct {
	Name        string
	Key         string
	Description string
	Permissions []string
	BuiltIn     bool
}

type UpdateRoleInput struct {
	ID          string
	Name        string
	Key         string
	Description string
	Permissions []string
}

type ListInput struct {
	Offset int
	Limit  int
}

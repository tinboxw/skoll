package rbac

type ScopeType string

const (
	ScopeAll         ScopeType = "all"
	ScopeDepartment  ScopeType = "department"
	ScopeDepartmentS ScopeType = "department_and_sub"
	ScopeSelf        ScopeType = "self"
)

type DataScope struct {
	Type         ScopeType
	DepartmentID string
}

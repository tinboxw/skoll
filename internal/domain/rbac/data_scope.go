package rbac

import "fmt"

type DataScope string

const (
	DataScopeSelf     DataScope = "self"
	DataScopeDept     DataScope = "dept"
	DataScopeDeptTree DataScope = "dept_tree"
	DataScopeAll      DataScope = "all"
	DataScopeCustom   DataScope = "custom"
)

func (s DataScope) Validate() error {
	switch s {
	case DataScopeSelf, DataScopeDept, DataScopeDeptTree, DataScopeAll, DataScopeCustom:
		return nil
	default:
		return fmt.Errorf("unsupported data scope: %s", s)
	}
}

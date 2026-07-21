package rbac

import (
	"fmt"
	"strings"
)

type DataScope string

const (
	DataScopeSelf           DataScope = "self"
	DataScopeDepartment     DataScope = "department"
	DataScopeDepartmentTree DataScope = "department_tree"
	DataScopeAll            DataScope = "all"
	DataScopeCustom         DataScope = "custom"
)

func (s DataScope) Validate() error {
	switch NormalizeDataScope(s) {
	case DataScopeSelf, DataScopeDepartment, DataScopeDepartmentTree, DataScopeAll, DataScopeCustom:
		return nil
	default:
		return fmt.Errorf("unsupported data scope: %s", s)
	}
}

func NormalizeDataScope(scope DataScope) DataScope {
	switch DataScope(strings.ToLower(strings.TrimSpace(string(scope)))) {
	case DataScopeSelf:
		return DataScopeSelf
	case DataScopeDepartment:
		return DataScopeDepartment
	case DataScopeDepartmentTree:
		return DataScopeDepartmentTree
	case DataScopeAll:
		return DataScopeAll
	case DataScopeCustom:
		return DataScopeCustom
	default:
		return DataScope(strings.ToLower(strings.TrimSpace(string(scope))))
	}
}

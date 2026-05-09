package mysql

import "github.com/tinboxw/skoll/internal/store/memory"

type RoleStore = memory.RoleStore

func NewRoleStore() *RoleStore {
	return memory.NewRoleStore()
}

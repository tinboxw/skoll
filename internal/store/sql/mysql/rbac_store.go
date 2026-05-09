package mysql

import "github.com/tinboxw/skoll/internal/store/memory"

type RBACStore = memory.RBACStore

func NewRBACStore() *RBACStore {
	return memory.NewRBACStore()
}

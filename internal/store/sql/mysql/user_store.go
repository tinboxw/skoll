package mysql

import "github.com/tinboxw/skoll/internal/store/memory"

type UserStore = memory.UserStore

func NewUserStore() *UserStore {
	return memory.NewUserStore()
}

package gormrepo

import (
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo/model"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo/store"
	"gorm.io/gorm"
)

type UserStore = store.UserStore
type RoleStore = store.RoleStore
type SystemStore = store.SystemStore
type RBACStore = store.RBACStore
type PluginStore = store.PluginStore

func NewUserStore(db *gorm.DB) *UserStore {
	return store.NewUserStore(db)
}

func NewRoleStore(db *gorm.DB, normalizeKey model.Normalizer) *RoleStore {
	return store.NewRoleStore(db, normalizeKey)
}

func NewSystemStore(db *gorm.DB, normalizeKey model.Normalizer) *SystemStore {
	return store.NewSystemStore(db, normalizeKey)
}

func NewRBACStore(db *gorm.DB) *RBACStore {
	return store.NewRBACStore(db)
}

func NewPluginStore(db *gorm.DB) *PluginStore {
	return store.NewPluginStore(db)
}

func AllModels() []any {
	return model.AllModels()
}

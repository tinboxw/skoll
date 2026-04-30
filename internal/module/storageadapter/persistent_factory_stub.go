package storageadapter

import "github.com/tinboxw/skoll/internal/module/storageadapter/persistent"

func newMySQLAdapter(cfg PersistentBootstrapConfig) (Adapter, error) {
	return persistent.NewMySQLAdapter(cfg)
}

func newPostgresAdapter(cfg PersistentBootstrapConfig) (Adapter, error) {
	return persistent.NewPostgresAdapter(cfg)
}

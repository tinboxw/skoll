package storageadapter

import "github.com/tinboxw/skoll/internal/module/storageadapter/persistent"

const (
	EnvStorageDSN         = persistent.EnvStorageDSN
	EnvStorageMySQLDSN    = persistent.EnvStorageMySQLDSN
	EnvStoragePostgresDSN = persistent.EnvStoragePostgresDSN
)

type PersistentBootstrapConfig = persistent.BootstrapConfig

func ResolvePersistentBootstrapConfig(mode string, lookupEnv func(string) string) (PersistentBootstrapConfig, error) {
	return persistent.ResolveBootstrapConfig(mode, lookupEnv)
}

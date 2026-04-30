package persistent

import (
	"fmt"
	"os"
	"strings"
)

const (
	EnvStorageDSN         = "SKOLL_STORAGE_DSN"
	EnvStorageMySQLDSN    = "SKOLL_STORAGE_MYSQL_DSN"
	EnvStoragePostgresDSN = "SKOLL_STORAGE_POSTGRES_DSN"
)

type BootstrapConfig struct {
	Mode      string
	DSN       string
	SourceEnv string
}

func ResolveBootstrapConfig(mode string, lookupEnv func(string) string) (BootstrapConfig, error) {
	resolved := strings.ToLower(strings.TrimSpace(mode))
	if resolved != "mysql" && resolved != "postgres" {
		return BootstrapConfig{}, fmt.Errorf("persistent bootstrap config supports only mysql|postgres, got %q", mode)
	}

	if lookupEnv == nil {
		lookupEnv = os.Getenv
	}

	envKeys := dsnEnvKeys(resolved)
	for _, key := range envKeys {
		dsn := strings.TrimSpace(lookupEnv(key))
		if dsn == "" {
			continue
		}
		return BootstrapConfig{Mode: resolved, DSN: dsn, SourceEnv: key}, nil
	}

	return BootstrapConfig{}, fmt.Errorf("storage adapter mode %q requires DSN via %s", resolved, strings.Join(envKeys, " or "))
}

func dsnEnvKeys(mode string) []string {
	switch mode {
	case "mysql":
		return []string{EnvStorageMySQLDSN, EnvStorageDSN}
	case "postgres":
		return []string{EnvStoragePostgresDSN, EnvStorageDSN}
	default:
		return nil
	}
}

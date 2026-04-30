package storageadapter

import "github.com/tinboxw/skoll/internal/module/storageadapter/contracts"

// Keep root package API stable while implementations move into subpackages.
type Adapter = contracts.Adapter

type UserRepository = contracts.UserRepository
type RoleRepository = contracts.RoleRepository
type MenuRepository = contracts.MenuRepository
type AuditRepository = contracts.AuditRepository
type ConfigRepository = contracts.ConfigRepository
type DictionaryRepository = contracts.DictionaryRepository
type FileRepository = contracts.FileRepository
type JobRepository = contracts.JobRepository
type GeneratorRepository = contracts.GeneratorRepository
type PluginRepository = contracts.PluginRepository
type RBACRepository = contracts.RBACRepository
type APIRegistryRepository = contracts.APIRegistryRepository
type ReleaseRepository = contracts.ReleaseRepository

package persistent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/gorm"

	"github.com/tinboxw/skoll/internal/module/fileservice"
	"github.com/tinboxw/skoll/internal/module/storageadapter/contracts"
	"github.com/tinboxw/skoll/internal/module/storageadapter/persistent/db"
)

// Adapter is the SQL-backed implementation of contracts.Adapter. It wires the
// 13 module repositories over a shared *gorm.DB and a local file backend.
type Adapter struct {
	db *gorm.DB

	closeFn func() error

	users    contracts.UserRepository
	roles    contracts.RoleRepository
	menus    contracts.MenuRepository
	audit    contracts.AuditRepository
	configs  contracts.ConfigRepository
	dicts    contracts.DictionaryRepository
	files    contracts.FileRepository
	jobs     contracts.JobRepository
	gen      contracts.GeneratorRepository
	plugins  contracts.PluginRepository
	rbac     contracts.RBACRepository
	apis     contracts.APIRegistryRepository
	releases contracts.ReleaseRepository
}

type repositorySet struct {
	users    contracts.UserRepository
	roles    contracts.RoleRepository
	menus    contracts.MenuRepository
	audit    contracts.AuditRepository
	configs  contracts.ConfigRepository
	dicts    contracts.DictionaryRepository
	files    contracts.FileRepository
	jobs     contracts.JobRepository
	gen      contracts.GeneratorRepository
	plugins  contracts.PluginRepository
	rbac     contracts.RBACRepository
	apis     contracts.APIRegistryRepository
	releases contracts.ReleaseRepository
}

// NewAdapter constructs a SQL-backed adapter over an already-opened gorm DB.
// It runs AutoMigrate for every repository's owned models and then constructs
// each repository, hydrating any in-memory state from the database.
//
// The supplied uploadDir is used for file blob storage. If empty, a directory
// under os.TempDir() is used.
func NewAdapter(gdb *gorm.DB, uploadDir string) (*Adapter, error) {
	return newAdapter(gdb, uploadDir, nil)
}

func newOwnedAdapter(gdb *gorm.DB, uploadDir string) (*Adapter, error) {
	if gdb == nil {
		return nil, fmt.Errorf("persistent.newOwnedAdapter: nil db")
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, fmt.Errorf("persistent.newOwnedAdapter: extract sql.DB: %w", err)
	}
	return newAdapter(gdb, uploadDir, sqlDB.Close)
}

func newAdapter(gdb *gorm.DB, uploadDir string, closeFn func() error) (*Adapter, error) {
	if gdb == nil {
		return nil, fmt.Errorf("persistent.NewAdapter: nil db")
	}
	if err := migrateRepositoryModels(gdb); err != nil {
		return nil, fmt.Errorf("persistent.NewAdapter: migrate: %w", err)
	}

	repos, err := buildRepositories(gdb, uploadDir)
	if err != nil {
		return nil, fmt.Errorf("persistent.NewAdapter: build repositories: %w", err)
	}

	return &Adapter{
		db:       gdb,
		closeFn:  closeFn,
		users:    repos.users,
		roles:    repos.roles,
		menus:    repos.menus,
		audit:    repos.audit,
		configs:  repos.configs,
		dicts:    repos.dicts,
		files:    repos.files,
		jobs:     repos.jobs,
		gen:      repos.gen,
		plugins:  repos.plugins,
		rbac:     repos.rbac,
		apis:     repos.apis,
		releases: repos.releases,
	}, nil
}

func defaultUploadDir() string {
	return filepath.Join(os.TempDir(), "skoll-uploads")
}

func migrateRepositoryModels(gdb *gorm.DB) error {
	// Materialise every repo's owned models for a single AutoMigrate pass.
	// Repos that need to read existing rows during construction (rbac, user,
	// job, plugin, file) require their tables to exist beforehand.
	models := []any{}
	models = append(models, (&AuditRepository{}).Models()...)
	models = append(models, (&ConfigRepository{}).Models()...)
	models = append(models, (&DictionaryRepository{}).Models()...)
	models = append(models, (&RoleRepository{}).Models()...)
	models = append(models, (&MenuRepository{}).Models()...)
	models = append(models, (&RBACRepository{}).Models()...)
	models = append(models, (&UserRepository{}).Models()...)
	models = append(models, (&JobRepository{}).Models()...)
	models = append(models, (&PluginRepository{}).Models()...)
	models = append(models, (&FileRepository{}).Models()...)
	models = append(models, (&ReleaseRepository{}).Models()...)
	models = append(models, (&APIRegistryRepository{}).Models()...)
	return db.Migrate(gdb, models...)
}

func buildRepositories(gdb *gorm.DB, uploadDir string) (repositorySet, error) {
	if uploadDir == "" {
		uploadDir = defaultUploadDir()
	}
	backend, err := fileservice.NewLocalBackend(uploadDir)
	if err != nil {
		return repositorySet{}, fmt.Errorf("backend: %w", err)
	}

	rbacRepo, err := NewRBACRepository(gdb)
	if err != nil {
		return repositorySet{}, fmt.Errorf("rbac: %w", err)
	}
	userRepo, err := NewUserRepository(gdb)
	if err != nil {
		return repositorySet{}, fmt.Errorf("user: %w", err)
	}
	jobRepo, err := NewJobRepository(gdb)
	if err != nil {
		return repositorySet{}, fmt.Errorf("job: %w", err)
	}
	pluginRepo, err := NewPluginRepository(gdb)
	if err != nil {
		return repositorySet{}, fmt.Errorf("plugin: %w", err)
	}
	fileRepo, err := NewFileRepository(gdb, backend)
	if err != nil {
		return repositorySet{}, fmt.Errorf("file: %w", err)
	}
	releaseRepo, err := NewReleaseRepository(gdb)
	if err != nil {
		return repositorySet{}, fmt.Errorf("release: %w", err)
	}

	return repositorySet{
		users:    userRepo,
		roles:    NewRoleRepository(gdb),
		menus:    NewMenuRepository(gdb),
		audit:    NewAuditRepository(gdb),
		configs:  NewConfigRepository(gdb),
		dicts:    NewDictionaryRepository(gdb),
		files:    fileRepo,
		jobs:     jobRepo,
		gen:      NewGeneratorRepository(),
		plugins:  pluginRepo,
		rbac:     rbacRepo,
		apis:     NewAPIRegistryRepository(gdb),
		releases: releaseRepo,
	}, nil
}

func (a *Adapter) Users() contracts.UserRepository              { return a.users }
func (a *Adapter) Roles() contracts.RoleRepository              { return a.roles }
func (a *Adapter) Menus() contracts.MenuRepository              { return a.menus }
func (a *Adapter) Audit() contracts.AuditRepository             { return a.audit }
func (a *Adapter) Configs() contracts.ConfigRepository          { return a.configs }
func (a *Adapter) Dictionaries() contracts.DictionaryRepository { return a.dicts }
func (a *Adapter) Files() contracts.FileRepository              { return a.files }
func (a *Adapter) Jobs() contracts.JobRepository                { return a.jobs }
func (a *Adapter) Generators() contracts.GeneratorRepository    { return a.gen }
func (a *Adapter) Plugins() contracts.PluginRepository          { return a.plugins }
func (a *Adapter) RBAC() contracts.RBACRepository               { return a.rbac }
func (a *Adapter) APIs() contracts.APIRegistryRepository        { return a.apis }
func (a *Adapter) Releases() contracts.ReleaseRepository        { return a.releases }

func (a *Adapter) Close(context.Context) error {
	if a.closeFn == nil {
		return nil
	}
	return a.closeFn()
}

// DB returns the underlying gorm database. Useful for advanced operations
// (custom queries, transaction probes) but should not be used to bypass the
// repository contracts in regular flow.
func (a *Adapter) DB() *gorm.DB { return a.db }

var _ contracts.Lifecycle = (*Adapter)(nil)

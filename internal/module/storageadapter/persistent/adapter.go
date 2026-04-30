package persistent

import (
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
	if gdb == nil {
		return nil, fmt.Errorf("persistent.NewAdapter: nil db")
	}
	if uploadDir == "" {
		uploadDir = filepath.Join(os.TempDir(), "skoll-uploads")
	}
	backend, err := fileservice.NewLocalBackend(uploadDir)
	if err != nil {
		return nil, fmt.Errorf("persistent.NewAdapter: backend: %w", err)
	}

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
	if err := db.Migrate(gdb, models...); err != nil {
		return nil, fmt.Errorf("persistent.NewAdapter: migrate: %w", err)
	}

	rbacRepo, err := NewRBACRepository(gdb)
	if err != nil {
		return nil, fmt.Errorf("persistent.NewAdapter: rbac: %w", err)
	}
	userRepo, err := NewUserRepository(gdb)
	if err != nil {
		return nil, fmt.Errorf("persistent.NewAdapter: user: %w", err)
	}
	jobRepo, err := NewJobRepository(gdb)
	if err != nil {
		return nil, fmt.Errorf("persistent.NewAdapter: job: %w", err)
	}
	pluginRepo, err := NewPluginRepository(gdb)
	if err != nil {
		return nil, fmt.Errorf("persistent.NewAdapter: plugin: %w", err)
	}
	fileRepo, err := NewFileRepository(gdb, backend)
	if err != nil {
		return nil, fmt.Errorf("persistent.NewAdapter: file: %w", err)
	}
	releaseRepo, err := NewReleaseRepository(gdb)
	if err != nil {
		return nil, fmt.Errorf("persistent.NewAdapter: release: %w", err)
	}

	return &Adapter{
		db:       gdb,
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

// DB returns the underlying gorm database. Useful for advanced operations
// (custom queries, transaction probes) but should not be used to bypass the
// repository contracts in regular flow.
func (a *Adapter) DB() *gorm.DB { return a.db }

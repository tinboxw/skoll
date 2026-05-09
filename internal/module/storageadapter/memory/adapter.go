package memory

import (
	"context"
	"os"
	"path/filepath"

	"github.com/tinboxw/skoll/internal/module/apiregistry"
	"github.com/tinboxw/skoll/internal/module/audit"
	"github.com/tinboxw/skoll/internal/module/config"
	"github.com/tinboxw/skoll/internal/module/dictionary"
	"github.com/tinboxw/skoll/internal/module/fileservice"
	"github.com/tinboxw/skoll/internal/module/jobscheduler"
	"github.com/tinboxw/skoll/internal/module/menu"
	"github.com/tinboxw/skoll/internal/module/modgenerator"
	"github.com/tinboxw/skoll/internal/module/pluginmgr"
	"github.com/tinboxw/skoll/internal/module/rbac"
	"github.com/tinboxw/skoll/internal/module/releasegov"
	"github.com/tinboxw/skoll/internal/module/role"
	"github.com/tinboxw/skoll/internal/module/storageadapter/contracts"
	"github.com/tinboxw/skoll/internal/module/user"
)

type Adapter struct {
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

func NewAdapter() (*Adapter, error) {
	backend, err := fileservice.NewLocalBackend(filepath.Join(os.TempDir(), "skoll-uploads"))
	if err != nil {
		return nil, err
	}
	return &Adapter{
		users:    user.NewService(),
		roles:    role.NewService(),
		menus:    menu.NewService(),
		audit:    audit.NewService(),
		configs:  config.NewService(),
		dicts:    dictionary.NewService(),
		files:    fileservice.NewService(backend),
		jobs:     jobscheduler.NewService(),
		gen:      modgenerator.NewService(),
		plugins:  pluginmgr.NewService(),
		rbac:     rbac.NewService(),
		apis:     apiregistry.NewService(),
		releases: releasegov.NewService(),
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

func (a *Adapter) Close(context.Context) error { return nil }

package storageadapter

import (
	"github.com/tinboxw/skoll/internal/module/apiregistry"
	"github.com/tinboxw/skoll/internal/module/audit"
	"github.com/tinboxw/skoll/internal/module/config"
	"github.com/tinboxw/skoll/internal/module/dictionary"
	"github.com/tinboxw/skoll/internal/module/menu"
	"github.com/tinboxw/skoll/internal/module/rbac"
	"github.com/tinboxw/skoll/internal/module/role"
	"github.com/tinboxw/skoll/internal/module/user"
)

// Adapter groups module repositories behind a single storage boundary.
type Adapter interface {
	Users() UserRepository
	Roles() RoleRepository
	Menus() MenuRepository
	Audit() AuditRepository
	Configs() ConfigRepository
	Dictionaries() DictionaryRepository
	RBAC() RBACRepository
	APIs() APIRegistryRepository
}

type UserRepository interface {
	Create(name, email string) user.User
	Get(id int64) (user.User, error)
	List() []user.User
}

type RoleRepository interface {
	Create(name string, permissions []string) role.Role
	Get(id int64) (role.Role, error)
	List() []role.Role
}

type MenuRepository interface {
	Create(title, path string, order int) menu.Item
	Get(id int64) (menu.Item, error)
	List() []menu.Item
}

type AuditRepository interface {
	Append(actor, action, target string) audit.Record
	Recent(limit int) []audit.Record
	Query(q audit.Query) audit.QueryResult
}

type ConfigRepository interface {
	Set(key, value, description string) config.Entry
	Get(key string) (config.Entry, error)
	List() []config.Entry
}

type DictionaryRepository interface {
	Create(itemType, label, value string, sortOrder int, enabled bool) dictionary.Item
	Get(id int64) (dictionary.Item, error)
	List() []dictionary.Item
	ListByType(itemType string) []dictionary.Item
}

type RBACRepository interface {
	SetRoleMenus(roleID int64, menuIDs []int64) []int64
	GetRoleMenus(roleID int64) []int64
	SetRoleAPIs(roleID int64, apis []string) []string
	GetRoleAPIs(roleID int64) []string
}

type APIRegistryRepository interface {
	RegisterMany(entries []string)
	Exists(entry string) bool
	List() []string
}

type InMemoryAdapter struct {
	users   UserRepository
	roles   RoleRepository
	menus   MenuRepository
	audit   AuditRepository
	configs ConfigRepository
	dicts   DictionaryRepository
	rbac    RBACRepository
	apis    APIRegistryRepository
}

func NewInMemoryAdapter() *InMemoryAdapter {
	return &InMemoryAdapter{
		users:   user.NewService(),
		roles:   role.NewService(),
		menus:   menu.NewService(),
		audit:   audit.NewService(),
		configs: config.NewService(),
		dicts:   dictionary.NewService(),
		rbac:    rbac.NewService(),
		apis:    apiregistry.NewService(),
	}
}

func (a *InMemoryAdapter) Users() UserRepository {
	return a.users
}

func (a *InMemoryAdapter) Roles() RoleRepository {
	return a.roles
}

func (a *InMemoryAdapter) Menus() MenuRepository {
	return a.menus
}

func (a *InMemoryAdapter) Audit() AuditRepository {
	return a.audit
}

func (a *InMemoryAdapter) Configs() ConfigRepository {
	return a.configs
}

func (a *InMemoryAdapter) Dictionaries() DictionaryRepository {
	return a.dicts
}

func (a *InMemoryAdapter) RBAC() RBACRepository {
	return a.rbac
}

func (a *InMemoryAdapter) APIs() APIRegistryRepository {
	return a.apis
}

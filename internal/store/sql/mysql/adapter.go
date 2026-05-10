package mysql

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/tinboxw/skoll/internal/repository"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Adapter struct {
	dsn  string
	db   *gorm.DB
	user repository.UserRepository
	role repository.RoleRepository
	rbac repository.RBACRepository
	sys  repository.SystemRepository
}

func NewAdapter(dsn string) (*Adapter, error) {
	if storesql.NormalizeDSN(dsn) == "" {
		return nil, fmt.Errorf("mysql dsn is required")
	}

	resolvedDSN, err := resolveMySQLDSN(dsn)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(mysql.Open(resolvedDSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open mysql connection: %w", err)
	}

	if err := db.AutoMigrate(&userModel{}, &roleModel{}, &bindingModel{}, &policyRuleModel{}, &systemSettingModel{}); err != nil {
		return nil, fmt.Errorf("auto migrate mysql schema: %w", err)
	}

	return &Adapter{
		dsn:  resolvedDSN,
		db:   db,
		user: NewUserStore(db),
		role: NewRoleStore(db),
		rbac: NewRBACStore(db),
		sys:  NewSystemStore(db),
	}, nil
}

func (a *Adapter) DSN() string { return a.dsn }
func (a *Adapter) UserRepository() repository.UserRepository {
	return a.user
}
func (a *Adapter) RoleRepository() repository.RoleRepository {
	return a.role
}
func (a *Adapter) RBACRepository() repository.RBACRepository {
	return a.rbac
}

func (a *Adapter) SystemRepository() repository.SystemRepository {
	return a.sys
}

func resolveMySQLDSN(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if !strings.HasPrefix(value, "mysql://") {
		return value, nil
	}

	u, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("parse mysql dsn: %w", err)
	}
	if u.Host == "" {
		return "", fmt.Errorf("mysql host is required")
	}

	username := ""
	password := ""
	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
	}
	if username == "" {
		return "", fmt.Errorf("mysql username is required")
	}
	database := strings.TrimPrefix(u.Path, "/")
	if database == "" {
		return "", fmt.Errorf("mysql database is required")
	}

	query := u.Query()
	if query.Get("parseTime") == "" {
		query.Set("parseTime", "true")
	}
	if query.Get("charset") == "" {
		query.Set("charset", "utf8mb4")
	}
	queryString := query.Encode()

	credential := username
	if password != "" {
		credential += ":" + password
	}
	return fmt.Sprintf("%s@tcp(%s)/%s?%s", credential, u.Host, database, queryString), nil
}

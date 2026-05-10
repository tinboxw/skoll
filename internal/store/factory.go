package store

import (
	"fmt"
	"strings"

	"github.com/tinboxw/skoll/internal/repository"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
	"github.com/tinboxw/skoll/internal/store/memory"
	"github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/internal/store/sql/mysql"
	"github.com/tinboxw/skoll/internal/store/sql/postgres"
)

type Mode string

const (
	ModeMemory   Mode = "memory"
	ModeMySQL    Mode = "mysql"
	ModePostgres Mode = "postgres"
)

type Options struct {
	Mode          Mode
	PrimaryDSN    string
	ClickHouseDSN string
}

type Bundle struct {
	Users      repository.UserRepository
	Roles      repository.RoleRepository
	RBAC       repository.RBACRepository
	Audit      repository.AuditRepository
	System     repository.SystemRepository
	UnitOfWork repository.UnitOfWork
}

func NewBundle(opts Options) (*Bundle, error) {
	switch Mode(strings.ToLower(strings.TrimSpace(string(opts.Mode)))) {
	case ModeMemory:
		users := memory.NewUserStore()
		roles := memory.NewRoleStore()
		rbac := memory.NewRBACStore()
		audit := clickhouse.NewAuditStore()
		system := memory.NewSystemStore()
		return &Bundle{Users: users, Roles: roles, RBAC: rbac, Audit: audit, System: system, UnitOfWork: sql.NewUnitOfWork()}, nil
	case ModeMySQL:
		primary, err := mysql.NewAdapter(opts.PrimaryDSN)
		if err != nil {
			return nil, err
		}
		audit, err := clickhouse.NewAdapter(opts.ClickHouseDSN)
		if err != nil {
			return nil, err
		}
		return &Bundle{
			Users:      primary.UserRepository(),
			Roles:      primary.RoleRepository(),
			RBAC:       primary.RBACRepository(),
			Audit:      audit.AuditRepository(),
			System:     primary.SystemRepository(),
			UnitOfWork: sql.NewUnitOfWork(),
		}, nil
	case ModePostgres:
		primary, err := postgres.NewAdapter(opts.PrimaryDSN)
		if err != nil {
			return nil, err
		}
		audit, err := clickhouse.NewAdapter(opts.ClickHouseDSN)
		if err != nil {
			return nil, err
		}
		return &Bundle{
			Users:      primary.UserRepository(),
			Roles:      primary.RoleRepository(),
			RBAC:       memory.NewRBACStore(),
			Audit:      audit.AuditRepository(),
			System:     primary.SystemRepository(),
			UnitOfWork: sql.NewUnitOfWork(),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported store mode: %q", opts.Mode)
	}
}

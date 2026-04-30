package app

import (
	"net/http"
	"time"

	adminapiregistry "github.com/tinboxw/skoll/internal/app/admin/apiregistry"
	adminauditlog "github.com/tinboxw/skoll/internal/app/admin/auditlog"
	adminauth "github.com/tinboxw/skoll/internal/app/admin/auth"
	adminconfigdict "github.com/tinboxw/skoll/internal/app/admin/configdict"
	admincontracts "github.com/tinboxw/skoll/internal/app/admin/contracts"
	admindashboard "github.com/tinboxw/skoll/internal/app/admin/dashboard"
	admindbops "github.com/tinboxw/skoll/internal/app/admin/dbops"
	adminfiles "github.com/tinboxw/skoll/internal/app/admin/files"
	admingenerator "github.com/tinboxw/skoll/internal/app/admin/generator"
	adminhardening "github.com/tinboxw/skoll/internal/app/admin/hardening"
	adminjobs "github.com/tinboxw/skoll/internal/app/admin/jobs"
	adminmenu "github.com/tinboxw/skoll/internal/app/admin/menu"
	adminplugins "github.com/tinboxw/skoll/internal/app/admin/plugins"
	adminrbac "github.com/tinboxw/skoll/internal/app/admin/rbac"
	adminreleasegov "github.com/tinboxw/skoll/internal/app/admin/releasegov"
)

var adminModuleStartTime = time.Now().UTC()

func MountAdminModuleRoutes(mux *http.ServeMux, services admincontracts.AdminModuleServices, wrapper func(http.Handler) http.Handler) {
	if mux == nil {
		return
	}
	if services.Users == nil ||
		services.Roles == nil ||
		services.Menus == nil ||
		services.Audit == nil ||
		services.Configs == nil ||
		services.Dictionaries == nil ||
		services.Files == nil ||
		services.Jobs == nil ||
		services.Generator == nil ||
		services.Plugins == nil ||
		services.RBAC == nil ||
		services.APIs == nil ||
		services.Releases == nil {
		return
	}

	hardeningH := adminhardening.NewHandler(services.Audit)
	adminauth.NewHandler(services.Users, services.Audit).Register(mux, wrapper, services.APIs)
	adminrbac.NewHandler(services.Roles, services.Menus, services.RBAC, services.APIs, services.Audit).Register(mux, wrapper, services.APIs)
	adminconfigdict.NewHandler(services.Configs, services.Dictionaries, services.Audit).Register(mux, wrapper, services.APIs)
	admindashboard.NewHandler(
		func() any { return admindashboard.CollectSystemStatus(services) },
		func() any { return admindashboard.CollectRuntimeMetrics() },
		func() any { return admindashboard.CollectNodeHealth(services) },
		func(r *http.Request) any {
			authSession := admindashboard.CollectAuthSessionContext(r)
			authObservability := admindashboard.CollectAuthObservability()
			jwtSession := admindashboard.CollectJWTSessionBootstrap(r, time.Now().UTC())
			return admindashboard.AggregateResponse{
				Contract:             admindashboard.CollectContractDescriptor(),
				GeneratedAtUnixSec:   time.Now().UTC().Unix(),
				AuthSession:          authSession,
				AuthObservability:    authObservability,
				AuthActionability:    admindashboard.CollectAuthActionability(authSession, authObservability),
				JWTSessionBootstrap:  jwtSession,
				Status:               admindashboard.CollectSystemStatus(services),
				RuntimeMetrics:       admindashboard.CollectRuntimeMetrics(),
				NodeHealth:           admindashboard.CollectNodeHealth(services),
				SchedulerReliability: services.Jobs.ReliabilitySnapshot(time.Now().UTC()),
				HardeningPosture:     hardeningH.Snapshot(),
			}
		},
	).Register(mux, wrapper, services.APIs)

	adminmenu.NewHandler(services.Menus).Register(mux, wrapper, services.APIs)
	adminauditlog.NewHandler(services.Audit).Register(mux, wrapper, services.APIs)
	adminfiles.NewHandler(services.Files).Register(mux, wrapper, services.APIs)
	adminjobs.NewHandler(services.Jobs, services.Audit).Register(mux, wrapper, services.APIs)
	admingenerator.NewHandler(services.Generator).Register(mux, wrapper, services.APIs)
	adminplugins.NewHandler(services.Plugins, services.Audit).Register(mux, wrapper, services.APIs)
	admindbops.NewHandler(services.Audit).Register(mux, wrapper, services.APIs)
	hardeningH.Register(mux, wrapper, services.APIs)
	adminapiregistry.NewHandler(services.APIs).Register(mux, wrapper)
	adminreleasegov.NewHandler(services.Releases, services.Audit).Register(mux, wrapper, services.APIs)
}

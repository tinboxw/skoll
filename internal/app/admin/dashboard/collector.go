package dashboard

import (
	"net/http"
	"time"

	admincontracts "github.com/tinboxw/skoll/internal/app/admin/contracts"
)

type AggregateProviders struct {
	SchedulerReliability func(now time.Time) any
	HardeningPosture     func() any
}

type Collector struct {
	services admincontracts.AdminModuleServices
	now      func() time.Time
}

func NewCollector(services admincontracts.AdminModuleServices, now func() time.Time) *Collector {
	if now == nil {
		now = time.Now
	}
	return &Collector{services: services, now: now}
}

func (c *Collector) ContractDescriptor() ContractDescriptor {
	return CollectContractDescriptor()
}

func (c *Collector) SystemStatus() SystemStatusResponse {
	return CollectSystemStatus(c.services)
}

func (c *Collector) RuntimeMetrics() RuntimeMetricsResponse {
	return CollectRuntimeMetrics()
}

func (c *Collector) NodeHealth() NodeHealthResponse {
	return CollectNodeHealth(c.services)
}

func (c *Collector) AuthSessionContext(r *http.Request) AuthSessionContext {
	return CollectAuthSessionContext(r)
}

func (c *Collector) AuthObservability() AuthObservability {
	return CollectAuthObservability()
}

func (c *Collector) AuthActionability(authSession AuthSessionContext, authObs AuthObservability) AuthActionability {
	return CollectAuthActionability(authSession, authObs)
}

func (c *Collector) JWTSessionBootstrap(r *http.Request) JWTSessionBootstrap {
	return CollectJWTSessionBootstrap(r, c.now().UTC())
}

func (c *Collector) Aggregate(r *http.Request, providers AggregateProviders) AggregateResponse {
	now := c.now().UTC()
	authSession := c.AuthSessionContext(r)
	authObservability := c.AuthObservability()

	var schedulerReliability any
	if providers.SchedulerReliability != nil {
		schedulerReliability = providers.SchedulerReliability(now)
	}

	var hardeningPosture any
	if providers.HardeningPosture != nil {
		hardeningPosture = providers.HardeningPosture()
	}

	return AggregateResponse{
		Contract:             c.ContractDescriptor(),
		GeneratedAtUnixSec:   now.Unix(),
		AuthSession:          authSession,
		AuthObservability:    authObservability,
		AuthActionability:    c.AuthActionability(authSession, authObservability),
		JWTSessionBootstrap:  c.JWTSessionBootstrap(r),
		Status:               c.SystemStatus(),
		RuntimeMetrics:       c.RuntimeMetrics(),
		NodeHealth:           c.NodeHealth(),
		SchedulerReliability: schedulerReliability,
		HardeningPosture:     hardeningPosture,
	}
}

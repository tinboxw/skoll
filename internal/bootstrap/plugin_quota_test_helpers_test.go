package bootstrap

import (
	"time"

	"github.com/tinboxw/skoll/internal/plugin/quota"
	"github.com/tinboxw/skoll/pkg/config"
)

func testPluginQuotaPolicy() quota.Policy {
	limit := quota.Limit{RatePerSecond: 10000, Burst: 10000, MaxConcurrent: 128}
	return quota.Policy{
		Request: limit, HostCall: limit, Query: limit, Mutation: limit, Event: limit,
		Job: limit, Export: limit, Storage: limit, Process: quota.Limit{RatePerSecond: 100, Burst: 100, MaxConcurrent: 1},
		MaxPendingEvents: 100, MaxPendingJobs: 100,
		MaxFileBytes: 16 << 20, MaxStorageBytes: 512 << 20,
		MaxRequestBytes: 8 << 20, MaxResponseBytes: 16 << 20, RequestTimeout: 10 * time.Second,
		ProcessMemoryBytes: 512 << 20, ProcessMaxProcs: 2,
	}
}

func testPluginQuotaConfig() config.PluginQuotaConfig {
	return config.PluginQuotaConfig{
		RequestRate: 10000, RequestBurst: 10000, RequestConcurrency: 128,
		HostCallRate: 10000, HostCallBurst: 10000, HostCallConcurrency: 128,
		QueryRate: 10000, QueryBurst: 10000, QueryConcurrency: 128,
		MutationRate: 10000, MutationBurst: 10000, MutationConcurrency: 128,
		EventRate: 10000, EventBurst: 10000, EventConcurrency: 128,
		JobRate: 10000, JobBurst: 10000, JobConcurrency: 128,
		ExportRate: 10000, ExportBurst: 10000, ExportConcurrency: 128,
		StorageRate: 10000, StorageBurst: 10000, StorageConcurrency: 128,
		ProcessRate: 100, ProcessBurst: 100, ProcessConcurrency: 1,
		MaxPendingEvents: 100, MaxPendingJobs: 100,
		MaxFileBytes: 16 << 20, MaxStorageBytes: 512 << 20,
		MaxRequestBytes: 8 << 20, MaxResponseBytes: 16 << 20, RequestTimeout: 10 * time.Second,
		ProcessMemoryBytes: 512 << 20, ProcessMaxProcs: 2,
	}
}

func newTestPluginQuotaController() *quota.Controller {
	controller, err := quota.NewController(testPluginQuotaPolicy())
	if err != nil {
		panic(err)
	}
	return controller
}

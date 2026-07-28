package hostservice

import (
	"time"

	"github.com/tinboxw/skoll/internal/plugin/quota"
)

func hostServiceTestQuotaPolicy() quota.Policy {
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

func newHostServiceTestQuotaController() *quota.Controller {
	controller, err := quota.NewController(hostServiceTestQuotaPolicy())
	if err != nil {
		panic(err)
	}
	return controller
}

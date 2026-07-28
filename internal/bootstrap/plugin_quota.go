package bootstrap

import (
	"bytes"
	"errors"
	"math"
	"net/http"
	"strconv"
	"sync"

	httpHandler "github.com/tinboxw/skoll/internal/handler/http"
	"github.com/tinboxw/skoll/internal/plugin/quota"
	"github.com/tinboxw/skoll/pkg/config"
)

func pluginQuotaPolicy(in config.PluginQuotaConfig) quota.Policy {
	return quota.Policy{
		Request:  quota.Limit{RatePerSecond: in.RequestRate, Burst: in.RequestBurst, MaxConcurrent: in.RequestConcurrency},
		HostCall: quota.Limit{RatePerSecond: in.HostCallRate, Burst: in.HostCallBurst, MaxConcurrent: in.HostCallConcurrency},
		Query:    quota.Limit{RatePerSecond: in.QueryRate, Burst: in.QueryBurst, MaxConcurrent: in.QueryConcurrency},
		Mutation: quota.Limit{RatePerSecond: in.MutationRate, Burst: in.MutationBurst, MaxConcurrent: in.MutationConcurrency},
		Event:    quota.Limit{RatePerSecond: in.EventRate, Burst: in.EventBurst, MaxConcurrent: in.EventConcurrency},
		Job:      quota.Limit{RatePerSecond: in.JobRate, Burst: in.JobBurst, MaxConcurrent: in.JobConcurrency},
		Export:   quota.Limit{RatePerSecond: in.ExportRate, Burst: in.ExportBurst, MaxConcurrent: in.ExportConcurrency},
		Storage:  quota.Limit{RatePerSecond: in.StorageRate, Burst: in.StorageBurst, MaxConcurrent: in.StorageConcurrency},
		Process:  quota.Limit{RatePerSecond: in.ProcessRate, Burst: in.ProcessBurst, MaxConcurrent: in.ProcessConcurrency},

		MaxPendingEvents: in.MaxPendingEvents, MaxPendingJobs: in.MaxPendingJobs,
		MaxFileBytes: in.MaxFileBytes, MaxStorageBytes: in.MaxStorageBytes,
		MaxRequestBytes: in.MaxRequestBytes, MaxResponseBytes: in.MaxResponseBytes,
		RequestTimeout: in.RequestTimeout, ProcessMemoryBytes: in.ProcessMemoryBytes, ProcessMaxProcs: in.ProcessMaxProcs,
	}
}

func writePluginQuotaError(w http.ResponseWriter, err error) {
	var quotaErr *quota.Error
	if errors.As(err, &quotaErr) {
		seconds := int(math.Ceil(quotaErr.RetryAfter.Seconds()))
		if seconds < 1 {
			seconds = 1
		}
		w.Header().Set("Retry-After", strconv.Itoa(seconds))
		w.Header().Set("X-Skoll-Quota-Resource", string(quotaErr.Resource))
	}
	httpHandler.WriteMessage(w, http.StatusTooManyRequests, "plugin_quota_exceeded", "插件资源配额已用尽，请稍后重试")
}

type boundedPluginResponse struct {
	destination http.ResponseWriter
	maxBytes    int64

	mu       sync.Mutex
	header   http.Header
	body     bytes.Buffer
	status   int
	overflow bool
}

func newBoundedPluginResponse(destination http.ResponseWriter, maxBytes int64) *boundedPluginResponse {
	return &boundedPluginResponse{destination: destination, maxBytes: maxBytes, header: make(http.Header)}
}

func (w *boundedPluginResponse) Header() http.Header {
	return w.header
}

func (w *boundedPluginResponse) WriteHeader(status int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.status == 0 {
		w.status = status
	}
}

func (w *boundedPluginResponse) Write(value []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if int64(w.body.Len())+int64(len(value)) > w.maxBytes {
		w.overflow = true
		return 0, errors.New("plugin response exceeds quota")
	}
	return w.body.Write(value)
}

func (w *boundedPluginResponse) Flush() {}

func (w *boundedPluginResponse) commit() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.overflow {
		httpHandler.WriteMessage(w.destination, http.StatusBadGateway, "plugin_response_too_large", "插件响应超过资源配额")
		return
	}
	for key, values := range w.header {
		w.destination.Header()[key] = append([]string(nil), values...)
	}
	status := w.status
	if status == 0 {
		status = http.StatusOK
	}
	w.destination.WriteHeader(status)
	_, _ = w.destination.Write(w.body.Bytes())
}

var _ http.Flusher = (*boundedPluginResponse)(nil)

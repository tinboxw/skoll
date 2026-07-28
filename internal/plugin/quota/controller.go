package quota

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

type Resource string

const (
	ResourceRequest  Resource = "request"
	ResourceHostCall Resource = "host_call"
	ResourceQuery    Resource = "query"
	ResourceMutation Resource = "mutation"
	ResourceEvent    Resource = "event"
	ResourceJob      Resource = "job"
	ResourceExport   Resource = "export"
	ResourceStorage  Resource = "storage"
	ResourceProcess  Resource = "process"
)

var resources = []Resource{
	ResourceRequest,
	ResourceHostCall,
	ResourceQuery,
	ResourceMutation,
	ResourceEvent,
	ResourceJob,
	ResourceExport,
	ResourceStorage,
	ResourceProcess,
}

type Limit struct {
	RatePerSecond float64
	Burst         int
	MaxConcurrent int
}

func (l Limit) Validate(resource Resource) error {
	if l.RatePerSecond <= 0 {
		return fmt.Errorf("%s rate must be positive", resource)
	}
	if l.Burst <= 0 {
		return fmt.Errorf("%s burst must be positive", resource)
	}
	if l.MaxConcurrent <= 0 {
		return fmt.Errorf("%s concurrency must be positive", resource)
	}
	return nil
}

type Policy struct {
	Request  Limit
	HostCall Limit
	Query    Limit
	Mutation Limit
	Event    Limit
	Job      Limit
	Export   Limit
	Storage  Limit
	Process  Limit

	MaxPendingEvents   int
	MaxPendingJobs     int
	MaxFileBytes       int64
	MaxStorageBytes    int64
	MaxRequestBytes    int64
	MaxResponseBytes   int64
	RequestTimeout     time.Duration
	ProcessMemoryBytes int64
	ProcessMaxProcs    int
}

func (p Policy) Validate() error {
	for _, resource := range resources {
		if err := p.Limit(resource).Validate(resource); err != nil {
			return err
		}
	}
	if p.MaxPendingEvents < 1 || p.MaxPendingEvents > 199 {
		return errors.New("pending event limit must be between 1 and 199")
	}
	if p.MaxPendingJobs < 1 || p.MaxPendingJobs > 499 {
		return errors.New("pending job limit must be between 1 and 499")
	}
	if p.MaxFileBytes <= 0 || p.MaxStorageBytes < p.MaxFileBytes {
		return errors.New("storage limits require positive file bytes within total bytes")
	}
	if p.MaxRequestBytes <= 0 || p.MaxResponseBytes <= 0 {
		return errors.New("plugin request and response byte limits must be positive")
	}
	if p.RequestTimeout <= 0 {
		return errors.New("plugin request timeout must be positive")
	}
	if p.ProcessMemoryBytes <= 0 || p.ProcessMaxProcs <= 0 {
		return errors.New("plugin process memory and processor limits must be positive")
	}
	return nil
}

func (p Policy) Limit(resource Resource) Limit {
	switch resource {
	case ResourceRequest:
		return p.Request
	case ResourceHostCall:
		return p.HostCall
	case ResourceQuery:
		return p.Query
	case ResourceMutation:
		return p.Mutation
	case ResourceEvent:
		return p.Event
	case ResourceJob:
		return p.Job
	case ResourceExport:
		return p.Export
	case ResourceStorage:
		return p.Storage
	case ResourceProcess:
		return p.Process
	default:
		return Limit{}
	}
}

type Error struct {
	PluginID   string
	Resource   Resource
	RetryAfter time.Duration
}

func (e *Error) Error() string {
	return fmt.Sprintf("plugin %s %s quota exceeded", e.PluginID, e.Resource)
}

func (e *Error) Retryable() bool {
	return true
}

type Snapshot struct {
	Resource       Resource  `json:"resource"`
	RatePerSecond  float64   `json:"ratePerSecond"`
	Burst          int       `json:"burst"`
	MaxConcurrent  int       `json:"maxConcurrent"`
	Active         int       `json:"active"`
	Available      int       `json:"available"`
	Rejected       uint64    `json:"rejected"`
	Reserved       int64     `json:"reserved"`
	LastRejectedAt time.Time `json:"lastRejectedAt,omitempty"`
}

type bucket struct {
	tokens         float64
	lastRefill     time.Time
	active         int
	rejected       uint64
	lastRejectedAt time.Time
	reserved       int64
}

type Controller struct {
	policy Policy
	now    func() time.Time
	mu     sync.Mutex
	state  map[string]map[Resource]*bucket
}

func NewController(policy Policy) (*Controller, error) {
	return newController(policy, time.Now)
}

func newController(policy Policy, now func() time.Time) (*Controller, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	if now == nil {
		return nil, errors.New("plugin quota clock is required")
	}
	return &Controller{policy: policy, now: now, state: make(map[string]map[Resource]*bucket)}, nil
}

func (c *Controller) Policy() Policy {
	if c == nil {
		return Policy{}
	}
	return c.policy
}

func (c *Controller) Acquire(pluginID string, resource Resource) (*Lease, error) {
	if c == nil {
		return nil, errors.New("plugin quota controller is required")
	}
	pluginID = strings.ToLower(strings.TrimSpace(pluginID))
	if pluginID == "" {
		return nil, errors.New("plugin quota identity is required")
	}
	limit := c.policy.Limit(resource)
	if err := limit.Validate(resource); err != nil {
		return nil, err
	}
	now := c.now().UTC()
	c.mu.Lock()
	item := c.bucket(pluginID, resource, limit, now)
	c.refill(item, limit, now)
	if item.active >= limit.MaxConcurrent || item.tokens < 1 {
		item.rejected++
		item.lastRejectedAt = now
		retryAfter := time.Duration(math.Ceil((1-item.tokens)/limit.RatePerSecond*1000)) * time.Millisecond
		if retryAfter < time.Millisecond {
			retryAfter = time.Millisecond
		}
		c.mu.Unlock()
		return nil, &Error{PluginID: pluginID, Resource: resource, RetryAfter: retryAfter}
	}
	item.tokens--
	item.active++
	c.mu.Unlock()
	return &Lease{controller: c, pluginID: pluginID, resource: resource}, nil
}

func (c *Controller) Snapshot(pluginID string) []Snapshot {
	if c == nil {
		return []Snapshot{}
	}
	pluginID = strings.ToLower(strings.TrimSpace(pluginID))
	now := c.now().UTC()
	out := make([]Snapshot, 0, len(resources))
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, resource := range resources {
		limit := c.policy.Limit(resource)
		item := c.bucket(pluginID, resource, limit, now)
		c.refill(item, limit, now)
		out = append(out, Snapshot{
			Resource: resource, RatePerSecond: limit.RatePerSecond, Burst: limit.Burst,
			MaxConcurrent: limit.MaxConcurrent, Active: item.active,
			Available: int(math.Floor(item.tokens)), Rejected: item.rejected,
			Reserved: item.reserved, LastRejectedAt: item.lastRejectedAt,
		})
	}
	return out
}

func (c *Controller) Reserve(pluginID string, resource Resource, current, amount, maximum int64) (*Reservation, error) {
	if c == nil {
		return nil, errors.New("plugin quota controller is required")
	}
	pluginID = strings.ToLower(strings.TrimSpace(pluginID))
	if pluginID == "" || current < 0 || amount <= 0 || maximum <= 0 {
		return nil, errors.New("plugin capacity reservation is invalid")
	}
	now := c.now().UTC()
	limit := c.policy.Limit(resource)
	c.mu.Lock()
	item := c.bucket(pluginID, resource, limit, now)
	if current+item.reserved+amount > maximum {
		item.rejected++
		item.lastRejectedAt = now
		c.mu.Unlock()
		return nil, &Error{PluginID: pluginID, Resource: resource, RetryAfter: time.Second}
	}
	item.reserved += amount
	c.mu.Unlock()
	return &Reservation{controller: c, pluginID: pluginID, resource: resource, amount: amount}, nil
}

func (c *Controller) bucket(pluginID string, resource Resource, limit Limit, now time.Time) *bucket {
	byResource := c.state[pluginID]
	if byResource == nil {
		byResource = make(map[Resource]*bucket)
		c.state[pluginID] = byResource
	}
	item := byResource[resource]
	if item == nil {
		item = &bucket{tokens: float64(limit.Burst), lastRefill: now}
		byResource[resource] = item
	}
	return item
}

func (c *Controller) refill(item *bucket, limit Limit, now time.Time) {
	elapsed := now.Sub(item.lastRefill).Seconds()
	if elapsed > 0 {
		item.tokens = math.Min(float64(limit.Burst), item.tokens+elapsed*limit.RatePerSecond)
		item.lastRefill = now
	}
}

func (c *Controller) release(pluginID string, resource Resource) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if item := c.state[pluginID][resource]; item != nil && item.active > 0 {
		item.active--
	}
}

func (c *Controller) releaseReservation(pluginID string, resource Resource, amount int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if item := c.state[pluginID][resource]; item != nil {
		item.reserved -= amount
		if item.reserved < 0 {
			item.reserved = 0
		}
	}
}

type Lease struct {
	once       sync.Once
	controller *Controller
	pluginID   string
	resource   Resource
}

func (l *Lease) Release() {
	if l == nil || l.controller == nil {
		return
	}
	l.once.Do(func() { l.controller.release(l.pluginID, l.resource) })
}

type Reservation struct {
	once       sync.Once
	controller *Controller
	pluginID   string
	resource   Resource
	amount     int64
}

func (r *Reservation) Release() {
	if r == nil || r.controller == nil {
		return
	}
	r.once.Do(func() { r.controller.releaseReservation(r.pluginID, r.resource, r.amount) })
}

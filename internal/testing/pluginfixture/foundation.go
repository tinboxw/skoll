package pluginfixture

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"github.com/tinboxw/skoll/pkg/security"
)

const DefaultJWTSecret = "skoll-plugin-fixture-current-contract-secret"

type Clock struct {
	mu  sync.RWMutex
	now time.Time
}

func NewClock(now time.Time) *Clock {
	return &Clock{now: now.UTC()}
}

func (c *Clock) Now() time.Time {
	if c == nil {
		return time.Time{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.now
}

func (c *Clock) Set(now time.Time) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.now = now.UTC()
	c.mu.Unlock()
}

func (c *Clock) Advance(duration time.Duration) time.Time {
	if c == nil {
		return time.Time{}
	}
	c.mu.Lock()
	c.now = c.now.Add(duration)
	now := c.now
	c.mu.Unlock()
	return now
}

type Identity struct {
	Subject        string
	TenantID       string
	OrganizationID string
	Role           string
	Roles          []string
}

func (i Identity) Token(secret string, ttl time.Duration, now time.Time) (string, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "", errors.New("fixture JWT secret is required")
	}
	identity := security.JWTIdentity{
		Subject: strings.TrimSpace(i.Subject), OrganizationID: strings.TrimSpace(i.OrganizationID),
		OrganizationPath: []string{strings.TrimSpace(i.TenantID), strings.TrimSpace(i.OrganizationID)},
		Role:             strings.TrimSpace(i.Role), Roles: append([]string(nil), i.Roles...),
	}
	if identity.Role == "" {
		identity.Role = "operator"
	}
	if len(identity.Roles) == 0 {
		identity.Roles = []string{identity.Role}
	}
	return security.SignJWT(secret, identity, ttl, now.UTC())
}

func (i Identity) Scope() (pluginsdk.ScopePredicate, error) {
	return pluginsdk.NewScopePredicate(pluginsdk.TrustedScope{
		SubjectID: strings.TrimSpace(i.Subject), TenantIDs: []string{strings.TrimSpace(i.TenantID)},
		OwnerIDs: []string{strings.TrimSpace(i.Subject)}, OrganizationIDs: []string{strings.TrimSpace(i.OrganizationID)},
	})
}

type FailurePlan struct {
	mu     sync.Mutex
	next   map[string][]error
	sticky map[string]error
}

func NewFailurePlan() *FailurePlan {
	return &FailurePlan{next: make(map[string][]error), sticky: make(map[string]error)}
}

func (p *FailurePlan) FailNext(operation string, failures ...error) {
	if p == nil {
		return
	}
	operation = strings.TrimSpace(operation)
	p.mu.Lock()
	for _, failure := range failures {
		if failure != nil {
			p.next[operation] = append(p.next[operation], failure)
		}
	}
	p.mu.Unlock()
}

func (p *FailurePlan) Set(operation string, failure error) {
	if p == nil {
		return
	}
	operation = strings.TrimSpace(operation)
	p.mu.Lock()
	if failure == nil {
		delete(p.sticky, operation)
	} else {
		p.sticky[operation] = failure
	}
	p.mu.Unlock()
}

func (p *FailurePlan) Clear(operation string) {
	if p == nil {
		return
	}
	operation = strings.TrimSpace(operation)
	p.mu.Lock()
	delete(p.next, operation)
	delete(p.sticky, operation)
	p.mu.Unlock()
}

func (p *FailurePlan) take(operation string) error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if queued := p.next[operation]; len(queued) > 0 {
		failure := queued[0]
		if len(queued) == 1 {
			delete(p.next, operation)
		} else {
			p.next[operation] = queued[1:]
		}
		return failure
	}
	return p.sticky[operation]
}

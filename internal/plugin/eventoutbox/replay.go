package eventoutbox

import (
	"context"
	"errors"
	"strings"
	"time"
)

type ReplayAuthorizer interface {
	AuthorizeEventReplay(context.Context, string, Record) error
}

type ReplayAuditSink interface {
	RecordEventReplay(context.Context, string, Record) error
}

type ReplayOptions struct {
	Now func() time.Time
}

type Replayer struct {
	store      Store
	authorizer ReplayAuthorizer
	audit      ReplayAuditSink
	now        func() time.Time
}

func NewReplayer(store Store, authorizer ReplayAuthorizer, audit ReplayAuditSink, options ReplayOptions) (*Replayer, error) {
	if store == nil || authorizer == nil || audit == nil {
		return nil, errors.New("plugin event replay dependencies are required")
	}
	if options.Now == nil {
		options.Now = func() time.Time { return time.Now().UTC() }
	}
	return &Replayer{
		store: store, authorizer: authorizer, audit: audit,
		now: options.Now,
	}, nil
}

func (r *Replayer) Replay(ctx context.Context, actorID, eventID string) (Record, error) {
	if r == nil || r.store == nil || r.authorizer == nil || r.audit == nil {
		return Record{}, errors.New("plugin event replay is not configured")
	}
	actorID = strings.TrimSpace(actorID)
	eventID = strings.TrimSpace(eventID)
	if actorID == "" || eventID == "" {
		return Record{}, errors.New("plugin event replay actor and event identities are required")
	}
	current, err := r.store.Get(ctx, eventID)
	if err != nil {
		return Record{}, err
	}
	if current.Status != StatusDeadLetter {
		return Record{}, ErrNotDeadLetter
	}
	if err := r.authorizer.AuthorizeEventReplay(ctx, actorID, current); err != nil {
		return Record{}, err
	}
	replayed, err := r.store.Replay(ctx, eventID, r.now().UTC())
	if err != nil {
		return Record{}, err
	}
	if err := r.audit.RecordEventReplay(ctx, actorID, replayed); err != nil {
		return Record{}, err
	}
	return replayed, nil
}

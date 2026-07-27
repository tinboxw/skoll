package hostservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/plugin/eventoutbox"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type EventPublicationResolver func(string) ([]pluginsdk.EventPublicationDeclaration, error)

type eventService struct {
	pluginID string
	resolve  EventPublicationResolver
	store    eventoutbox.Store
	now      func() time.Time
}

func NewEventService(pluginID string, resolve EventPublicationResolver, store eventoutbox.Store, now func() time.Time) (pluginsdk.EventService, error) {
	pluginID = strings.ToLower(strings.TrimSpace(pluginID))
	if pluginID == "" || resolve == nil || store == nil {
		return nil, errors.New("plugin event service dependencies are required")
	}
	if now == nil {
		now = time.Now
	}
	return &eventService{pluginID: pluginID, resolve: resolve, store: store, now: now}, nil
}

func (s *eventService) Publish(ctx context.Context, publication pluginsdk.EventPublication) (pluginsdk.EventEnvelope, error) {
	declaration, err := s.declaration(publication.Name, publication.SchemaVersion)
	if err != nil {
		return pluginsdk.EventEnvelope{}, err
	}
	if err = publication.Validate(declaration); err != nil {
		return pluginsdk.EventEnvelope{}, err
	}
	if storesql.DBFromContext(ctx) == nil {
		return pluginsdk.EventEnvelope{}, pluginsdk.NewEventError(
			pluginsdk.EventErrorTransactionRequired, "transaction",
			"event publication requires an active host transaction", false,
		)
	}
	requestJSON, err := json.Marshal(publication)
	if err != nil {
		return pluginsdk.EventEnvelope{}, pluginsdk.NewEventError(pluginsdk.EventErrorInvalidRequest, "publication", "event publication cannot be encoded", false)
	}
	requestHash := sha256.Sum256(requestJSON)
	identityHash := sha256.Sum256([]byte(s.pluginID + "\x00" + publication.IdempotencyKey))
	now := s.now().UTC()
	record := eventoutbox.Record{
		Envelope: pluginsdk.EventEnvelope{
			ID: "event-" + hex.EncodeToString(identityHash[:]), Publisher: s.pluginID,
			Name: declaration.Name, SchemaVersion: declaration.SchemaVersion,
			PayloadType: declaration.PayloadType, Scope: publication.Scope,
			CorrelationID: publication.CorrelationID, CausationID: publication.CausationID,
			Subject: publication.Subject, Payload: publication.Payload, OccurredAt: now,
		},
		IdempotencyKey: publication.IdempotencyKey,
		RequestHash:    hex.EncodeToString(requestHash[:]),
		Status:         eventoutbox.StatusPending,
		MaxAttempts:    eventoutbox.DefaultMaxAttempts,
		NextAttemptAt:  now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	stored, _, err := s.store.Enqueue(ctx, record)
	if errors.Is(err, eventoutbox.ErrIdentityConflict) {
		return pluginsdk.EventEnvelope{}, pluginsdk.NewEventError(
			pluginsdk.EventErrorConflict, "idempotencyKey",
			"event identity is already bound to another publication", false,
		)
	}
	if err != nil {
		return pluginsdk.EventEnvelope{}, fmt.Errorf("persist plugin event outbox: %w", err)
	}
	return stored.Envelope, nil
}

func (s *eventService) declaration(name string, version uint32) (pluginsdk.EventPublicationDeclaration, error) {
	declarations, err := s.resolve(s.pluginID)
	if err != nil {
		return pluginsdk.EventPublicationDeclaration{}, fmt.Errorf("resolve plugin event declarations: %w", err)
	}
	name = strings.ToLower(strings.TrimSpace(name))
	for _, declaration := range declarations {
		if strings.ToLower(strings.TrimSpace(declaration.Name)) == name && declaration.SchemaVersion == version {
			return declaration, nil
		}
	}
	return pluginsdk.EventPublicationDeclaration{}, pluginsdk.NewEventError(
		pluginsdk.EventErrorUndeclaredPublication, "name",
		"plugin did not declare this event publication and schema version", false,
	)
}

var _ pluginsdk.EventService = (*eventService)(nil)

package plugin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/plugin/eventinbox"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type currentEventCatalog interface {
	Get(string) (Info, error)
	List() []Info
}

type CurrentEventRouterOptions struct {
	WorkerID      string
	LeaseDuration time.Duration
	Now           func() time.Time
}

type CurrentEventRouter struct {
	catalog       currentEventCatalog
	inbox         eventinbox.Store
	delivery      EventDeliveryClient
	workerID      string
	leaseDuration time.Duration
	now           func() time.Time
}

func NewCurrentEventRouter(catalog currentEventCatalog, inbox eventinbox.Store, delivery EventDeliveryClient, options CurrentEventRouterOptions) (*CurrentEventRouter, error) {
	if catalog == nil || inbox == nil || delivery == nil {
		return nil, errors.New("current plugin event router dependencies are required")
	}
	options.WorkerID = strings.TrimSpace(options.WorkerID)
	if options.WorkerID == "" {
		return nil, errors.New("current plugin event router worker identity is required")
	}
	if options.LeaseDuration <= 0 {
		options.LeaseDuration = 30 * time.Second
	}
	if options.Now == nil {
		options.Now = func() time.Time { return time.Now().UTC() }
	}
	return &CurrentEventRouter{
		catalog: catalog, inbox: inbox, delivery: delivery, workerID: options.WorkerID,
		leaseDuration: options.LeaseDuration, now: options.Now,
	}, nil
}

func (r *CurrentEventRouter) Deliver(ctx context.Context, envelope pluginsdk.EventEnvelope) error {
	if r == nil || r.catalog == nil || r.inbox == nil || r.delivery == nil {
		return errors.New("current plugin event router is not configured")
	}
	publisher, err := r.catalog.Get(strings.TrimSpace(envelope.Publisher))
	if err != nil || publisher.State != StateEnabled {
		return pluginsdk.NewEventError(pluginsdk.EventErrorUndeclaredPublication, "publisher", "event publisher is not enabled", false)
	}
	publication, ok := currentPublication(publisher, envelope)
	if !ok {
		return pluginsdk.NewEventError(pluginsdk.EventErrorUndeclaredPublication, "envelope", "event publication is not declared by publisher", false)
	}
	if err := envelope.Validate(publication); err != nil {
		return err
	}

	var deliveryErrors []error
	for _, subscriber := range r.catalog.List() {
		if subscriber.State != StateEnabled {
			continue
		}
		for _, subscription := range subscriber.EventSubscriptions() {
			if !currentSubscriptionMatches(subscription, envelope) {
				continue
			}
			delivery := pluginsdk.EventDelivery{
				DeliveryID: currentDeliveryID(envelope.ID, subscriber.ID, subscription.Handler),
				Subscriber: subscriber.ID, Handler: subscription.Handler, Envelope: envelope,
			}
			if err := delivery.Validate(subscription); err != nil {
				deliveryErrors = append(deliveryErrors, err)
				continue
			}
			now := r.now().UTC()
			claim, claimed, err := r.inbox.Claim(ctx, eventinbox.ClaimInput{
				Delivery: delivery, WorkerID: r.workerID, Now: now, LeaseDuration: r.leaseDuration,
			})
			if err != nil {
				deliveryErrors = append(deliveryErrors, fmt.Errorf("claim event delivery %s: %w", delivery.DeliveryID, err))
				continue
			}
			if !claimed {
				continue
			}
			if err := r.delivery.Deliver(ctx, subscriber, delivery); err != nil {
				_, failErr := r.inbox.Fail(ctx, eventinbox.TransitionInput{
					DeliveryID: delivery.DeliveryID, LeaseToken: claim.LeaseToken, Error: err.Error(), Now: r.now().UTC(),
				})
				deliveryErrors = append(deliveryErrors, errors.Join(err, failErr))
				continue
			}
			if _, err := r.inbox.Complete(ctx, eventinbox.TransitionInput{
				DeliveryID: delivery.DeliveryID, LeaseToken: claim.LeaseToken, Now: r.now().UTC(),
			}); err != nil {
				deliveryErrors = append(deliveryErrors, err)
			}
		}
	}
	return errors.Join(deliveryErrors...)
}

func currentPublication(info Info, envelope pluginsdk.EventEnvelope) (pluginsdk.EventPublicationDeclaration, bool) {
	for _, publication := range info.EventPublications() {
		if publication.Name == envelope.Name && publication.SchemaVersion == envelope.SchemaVersion &&
			publication.PayloadType == envelope.PayloadType {
			return publication, true
		}
	}
	return pluginsdk.EventPublicationDeclaration{}, false
}

func currentSubscriptionMatches(subscription EventSubscription, envelope pluginsdk.EventEnvelope) bool {
	if subscription.Publisher != envelope.Publisher || subscription.Name != envelope.Name {
		return false
	}
	for _, version := range subscription.SchemaVersions {
		if version == envelope.SchemaVersion {
			return true
		}
	}
	return false
}

func currentDeliveryID(eventID, subscriber, handler string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(eventID) + "\x00" + strings.TrimSpace(subscriber) + "\x00" + strings.TrimSpace(handler)))
	return "delivery-" + hex.EncodeToString(sum[:])
}

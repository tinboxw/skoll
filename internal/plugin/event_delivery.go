package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/event"
)

const pluginEventEndpoint = "/_skoll/events"

type EventDeliveryClient interface {
	Deliver(ctx context.Context, info Info, subscription EventSubscription, evt event.BusinessEvent) error
}

type EventDeliveryEnvelope struct {
	DeliveryID string            `json:"deliveryId"`
	PluginID   string            `json:"pluginId"`
	Handler    string            `json:"handler"`
	EventID    string            `json:"eventId"`
	EventName  string            `json:"eventName"`
	Source     string            `json:"source,omitempty"`
	Subject    EventSubject      `json:"subject"`
	Payload    map[string]any    `json:"payload"`
	Metadata   map[string]string `json:"metadata"`
	OccurredAt time.Time         `json:"occurredAt"`
}

type EventSubject struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id,omitempty"`
}

type HTTPEventDeliveryClient struct {
	client *http.Client
}

func NewHTTPEventDeliveryClient(client *http.Client, timeout time.Duration) *HTTPEventDeliveryClient {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if client == nil {
		client = &http.Client{}
	}
	configured := *client
	if configured.Timeout <= 0 {
		configured.Timeout = timeout
	}
	configured.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &HTTPEventDeliveryClient{client: &configured}
}

func (c *HTTPEventDeliveryClient) Deliver(ctx context.Context, info Info, subscription EventSubscription, evt event.BusinessEvent) error {
	if c == nil || c.client == nil {
		return errors.New("plugin event delivery client is not configured")
	}
	if err := evt.Validate(); err != nil {
		return err
	}
	pluginID := strings.TrimSpace(info.ID)
	handler := strings.TrimSpace(subscription.Handler)
	if pluginID == "" || handler == "" {
		return ErrPluginManifestBroken
	}
	target, err := pluginEventURL(info.ServiceBaseURL)
	if err != nil {
		return err
	}
	handlerName := pluginID + ":" + handler
	envelope := EventDeliveryEnvelope{
		DeliveryID: event.BusinessDeliveryID(evt.ID, handlerName),
		PluginID:   pluginID,
		Handler:    handler,
		EventID:    evt.ID,
		EventName:  evt.EventName,
		Source:     evt.Source,
		Subject:    EventSubject{Type: evt.SubjectType, ID: evt.SubjectID},
		Payload:    evt.Payload,
		Metadata:   evt.Metadata,
		OccurredAt: evt.OccurredAt.UTC(),
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("encode plugin event delivery: %w", err)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build plugin event delivery: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", envelope.DeliveryID)
	req.Header.Set("X-Skoll-Plugin-ID", pluginID)
	req.Header.Set("X-Skoll-Event-Handler", handler)

	response, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("plugin event delivery failed: %w", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("plugin event delivery returned status %d", response.StatusCode)
	}
	return nil
}

func pluginEventURL(raw string) (*url.URL, error) {
	target, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || target == nil || target.Host == "" || (target.Scheme != "http" && target.Scheme != "https") {
		return nil, ErrPluginManifestBroken
	}
	if target.User != nil || target.RawQuery != "" || target.Fragment != "" {
		return nil, ErrPluginManifestBroken
	}
	target.Path = strings.TrimRight(target.Path, "/") + pluginEventEndpoint
	target.RawPath = ""
	return target, nil
}

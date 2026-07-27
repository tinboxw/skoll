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

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const pluginEventEndpoint = "/_skoll/events"

type EventDeliveryClient interface {
	Deliver(ctx context.Context, info Info, delivery pluginsdk.EventDelivery) error
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

func (c *HTTPEventDeliveryClient) Deliver(ctx context.Context, info Info, delivery pluginsdk.EventDelivery) error {
	if c == nil || c.client == nil {
		return errors.New("plugin event delivery client is not configured")
	}
	pluginID := strings.TrimSpace(info.ID)
	if pluginID == "" || pluginID != strings.TrimSpace(delivery.Subscriber) ||
		strings.TrimSpace(delivery.Handler) == "" || strings.TrimSpace(delivery.DeliveryID) == "" {
		return ErrPluginManifestBroken
	}
	target, err := pluginEventURL(info.ServiceBaseURL)
	if err != nil {
		return err
	}
	body, err := json.Marshal(delivery)
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
	req.Header.Set("Idempotency-Key", delivery.DeliveryID)
	req.Header.Set("X-Skoll-Plugin-ID", pluginID)
	req.Header.Set("X-Skoll-Event-Handler", delivery.Handler)
	req.Header.Set("X-Skoll-Event-Publisher", delivery.Envelope.Publisher)

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

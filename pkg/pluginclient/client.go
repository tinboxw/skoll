package pluginclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const maxHostResponseBytes = 32 << 20

type Options struct {
	PluginID   string
	HostURL    string
	HostToken  string
	HTTPClient *http.Client
}

type Client struct {
	pluginID string
	baseURL  string
	token    string
	http     *http.Client
}

type Error struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *Error) Error() string {
	if e == nil {
		return "plugin host call failed"
	}
	return fmt.Sprintf("plugin host call failed: status=%d code=%s message=%s", e.StatusCode, e.Code, e.Message)
}

func FromEnvironment() (*Client, error) {
	return New(Options{
		PluginID: os.Getenv(EnvironmentPluginID), HostURL: os.Getenv(EnvironmentHostURL),
		HostToken: os.Getenv(EnvironmentHostToken),
	})
}

func New(options Options) (*Client, error) {
	pluginID := strings.TrimSpace(options.PluginID)
	token := strings.TrimSpace(options.HostToken)
	if pluginID == "" || token == "" {
		return nil, errors.New("plugin host identity and token are required")
	}
	parsed, err := url.Parse(strings.TrimSpace(options.HostURL))
	if err != nil || parsed.Scheme != "http" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("plugin host URL must be plain loopback HTTP")
	}
	host := strings.TrimSpace(parsed.Hostname())
	ip := net.ParseIP(host)
	if !strings.EqualFold(host, "localhost") && (ip == nil || !ip.IsLoopback()) {
		return nil, errors.New("plugin host URL must use a loopback host")
	}
	client := options.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{pluginID: pluginID, baseURL: strings.TrimRight(parsed.String(), "/"), token: token, http: client}, nil
}

func (c *Client) HostServices() (pluginsdk.HostServices, error) {
	if c == nil {
		return pluginsdk.HostServices{}, errors.New("plugin host client is required")
	}
	host := pluginsdk.HostServices{
		PluginID:     c.pluginID,
		Transactions: transactionService{client: c}, DataScopes: dataScopeService{client: c},
		DataStore: dataStoreService{client: c},
		Files:     fileService{client: c}, Audit: auditService{client: c}, Config: configService{client: c},
		Secrets: secretService{client: c}, Workflows: workflowService{client: c}, Jobs: jobService{client: c},
	}
	if err := host.Validate(); err != nil {
		return pluginsdk.HostServices{}, err
	}
	return host, nil
}

type userTokenContextKey struct{}
type transactionContextKey struct{}

func WithUserToken(ctx context.Context, token string) context.Context {
	return context.WithValue(contextOrBackground(ctx), userTokenContextKey{}, strings.TrimSpace(token))
}

func (c *Client) call(ctx context.Context, capability, operation string, input any, output any) error {
	if c == nil || c.http == nil {
		return errors.New("plugin host client is not configured")
	}
	body, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("encode plugin host request: %w", err)
	}
	endpoint := c.baseURL + "/" + HostAPIVersion + "/" + capability + "/" + operation
	request, err := http.NewRequestWithContext(contextOrBackground(ctx), http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(AuthorizationHeader, "Bearer "+c.token)
	if userToken, _ := request.Context().Value(userTokenContextKey{}).(string); userToken != "" {
		request.Header.Set(UserTokenHeader, userToken)
	}
	if transactionID, _ := request.Context().Value(transactionContextKey{}).(string); transactionID != "" {
		request.Header.Set(TransactionHeader, transactionID)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("call plugin host: %w", err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxHostResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read plugin host response: %w", err)
	}
	if len(raw) > maxHostResponseBytes {
		return errors.New("plugin host response exceeds size limit")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var failure ErrorResponse
		_ = json.Unmarshal(raw, &failure)
		return &Error{StatusCode: response.StatusCode, Code: failure.Code, Message: failure.Message}
	}
	if output == nil || len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, output); err != nil {
		return fmt.Errorf("decode plugin host response: %w", err)
	}
	return nil
}

func contextOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

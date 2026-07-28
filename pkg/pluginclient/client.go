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
		DataStore:       dataStoreService{client: c},
		Events:          eventService{client: c},
		DocumentNumbers: documentNumberService{client: c},
		Documents:       documentWorkflowService{client: c},
		Files:           fileService{client: c}, Audit: auditService{client: c}, Config: configService{client: c},
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

func BindRequestContext(ctx context.Context, token string, header http.Header) (context.Context, error) {
	ctx = WithUserToken(ctx, token)
	operation := pluginsdk.OperationContext{
		CorrelationID: strings.TrimSpace(header.Get(CorrelationHeader)),
		RequestID:     strings.TrimSpace(header.Get(RequestIDHeader)),
		TraceID:       strings.TrimSpace(header.Get(TraceIDHeader)),
	}
	if operation.CorrelationID == "" {
		generated, err := pluginsdk.NewOperationContext(operation.RequestID, operation.TraceID)
		if err != nil {
			return nil, err
		}
		operation = generated
	}
	return pluginsdk.WithOperationContext(ctx, operation)
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
	ctx, operationContext, err := pluginsdk.EnsureOperationContext(contextOrBackground(ctx))
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
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
	request.Header.Set(CorrelationHeader, operationContext.CorrelationID)
	if operationContext.RequestID != "" {
		request.Header.Set(RequestIDHeader, operationContext.RequestID)
	}
	if operationContext.TraceID != "" {
		request.Header.Set(TraceIDHeader, operationContext.TraceID)
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
		if err := decodeResponseJSON(raw, &failure); err != nil || strings.TrimSpace(failure.Code) == "" || strings.TrimSpace(failure.Message) == "" {
			return &Error{StatusCode: response.StatusCode, Code: "host_response_invalid", Message: "plugin host returned an invalid error response"}
		}
		if capability == "datastore" {
			if code, ok := parseDataStoreErrorCode(failure.Code); ok {
				return pluginsdk.NewDataStoreError(code, failure.Field, failure.Message, failure.Retryable)
			}
		}
		if capability == "events" {
			if code, ok := parseEventErrorCode(failure.Code); ok {
				return pluginsdk.NewEventError(code, failure.Field, failure.Message, failure.Retryable)
			}
		}
		if capability == "document-numbers" {
			if code, ok := parseDocumentNumberErrorCode(failure.Code); ok {
				return pluginsdk.NewDocumentNumberError(code, failure.Field, failure.Message, failure.Retryable)
			}
		}
		if capability == "documents" {
			if code, ok := parseDocumentWorkflowErrorCode(failure.Code); ok {
				return pluginsdk.NewDocumentWorkflowError(code, failure.Field, failure.Message, failure.Retryable)
			}
		}
		return &Error{StatusCode: response.StatusCode, Code: failure.Code, Message: failure.Message}
	}
	if output == nil || len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	if err := decodeResponseJSON(raw, output); err != nil {
		return fmt.Errorf("decode plugin host response: %w", err)
	}
	return nil
}

func parseEventErrorCode(value string) (pluginsdk.EventErrorCode, bool) {
	code := pluginsdk.EventErrorCode(strings.TrimSpace(value))
	switch code {
	case pluginsdk.EventErrorInvalidRequest, pluginsdk.EventErrorUndeclaredPublication,
		pluginsdk.EventErrorUndeclaredSubscription, pluginsdk.EventErrorUnsupportedSchema,
		pluginsdk.EventErrorScopeMismatch, pluginsdk.EventErrorPayloadLimit,
		pluginsdk.EventErrorConflict, pluginsdk.EventErrorTransactionRequired,
		pluginsdk.EventErrorUnavailable:
		return code, true
	default:
		return "", false
	}
}

func parseDocumentWorkflowErrorCode(value string) (pluginsdk.DocumentWorkflowErrorCode, bool) {
	code := pluginsdk.DocumentWorkflowErrorCode(strings.TrimSpace(value))
	switch code {
	case pluginsdk.DocumentWorkflowErrorInvalidRequest, pluginsdk.DocumentWorkflowErrorForbidden,
		pluginsdk.DocumentWorkflowErrorNotFound, pluginsdk.DocumentWorkflowErrorConflict,
		pluginsdk.DocumentWorkflowErrorTransactionRequired, pluginsdk.DocumentWorkflowErrorUnavailable:
		return code, true
	default:
		return "", false
	}
}

func parseDocumentNumberErrorCode(value string) (pluginsdk.DocumentNumberErrorCode, bool) {
	code := pluginsdk.DocumentNumberErrorCode(strings.TrimSpace(value))
	switch code {
	case pluginsdk.DocumentNumberErrorInvalidRequest, pluginsdk.DocumentNumberErrorForbidden,
		pluginsdk.DocumentNumberErrorConflict, pluginsdk.DocumentNumberErrorTransactionRequired,
		pluginsdk.DocumentNumberErrorExhausted, pluginsdk.DocumentNumberErrorUnavailable:
		return code, true
	default:
		return "", false
	}
}

func decodeResponseJSON(raw []byte, output any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("plugin host response contains trailing JSON")
		}
		return err
	}
	return nil
}

func parseDataStoreErrorCode(value string) (pluginsdk.DataStoreErrorCode, bool) {
	code := pluginsdk.DataStoreErrorCode(strings.TrimSpace(value))
	switch code {
	case pluginsdk.DataStoreErrorInvalidRequest, pluginsdk.DataStoreErrorForbidden,
		pluginsdk.DataStoreErrorNotFound, pluginsdk.DataStoreErrorConflict,
		pluginsdk.DataStoreErrorLimitExceeded, pluginsdk.DataStoreErrorUnsupported,
		pluginsdk.DataStoreErrorUnavailable:
		return code, true
	default:
		return "", false
	}
}

func contextOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

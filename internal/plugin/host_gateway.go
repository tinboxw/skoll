package plugin

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginclient"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"github.com/tinboxw/skoll/pkg/security"
)

const (
	maxHostRequestBytes       = 32 << 20
	defaultHostTransactionTTL = 30 * time.Second
)

var errRemoteTransactionRollback = errors.New("remote plugin transaction rolled back")

type ProcessCredential struct {
	HostURL string
	Token   string
}

type ProcessCredentialIssuer interface {
	Issue(pluginID string) (ProcessCredential, error)
	Revoke(token string)
}

type HostServicesFactory func(pluginID string) (pluginsdk.HostServices, error)

type hostCredential struct {
	pluginID string
	host     pluginsdk.HostServices
}

type hostTransaction struct {
	id            string
	credentialKey [32]byte
	pluginID      string
	ctx           context.Context
	decision      chan bool
	done          chan error
	timer         *time.Timer
	callMu        sync.Mutex
}

// HostGateway is the loopback-only, lifecycle-authenticated host API for managed plugin processes.
type HostGateway struct {
	listener  net.Listener
	server    *http.Server
	url       string
	factory   HostServicesFactory
	jwtSecret string
	txTTL     time.Duration

	mu           sync.RWMutex
	credentials  map[[32]byte]hostCredential
	transactions map[string]*hostTransaction
	closeOnce    sync.Once
}

func NewHostGateway(factory HostServicesFactory, jwtSecret string, transactionTTL time.Duration) (*HostGateway, error) {
	if factory == nil {
		return nil, errors.New("plugin host services factory is required")
	}
	if strings.TrimSpace(jwtSecret) == "" {
		return nil, errors.New("plugin host JWT secret is required")
	}
	if transactionTTL <= 0 {
		transactionTTL = defaultHostTransactionTTL
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen for plugin host gateway: %w", err)
	}
	gateway := &HostGateway{
		listener: listener, url: "http://" + listener.Addr().String(), factory: factory,
		jwtSecret: jwtSecret, txTTL: transactionTTL,
		credentials: make(map[[32]byte]hostCredential), transactions: make(map[string]*hostTransaction),
	}
	gateway.server = &http.Server{Handler: gateway, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = gateway.server.Serve(listener) }()
	return gateway, nil
}

func (g *HostGateway) URL() string {
	if g == nil {
		return ""
	}
	return g.url
}

func (g *HostGateway) Issue(pluginID string) (ProcessCredential, error) {
	if g == nil {
		return ProcessCredential{}, errors.New("plugin host gateway is not configured")
	}
	pluginID = strings.ToLower(strings.TrimSpace(pluginID))
	if pluginID == "" {
		return ProcessCredential{}, errors.New("plugin host identity is required")
	}
	host, err := g.factory(pluginID)
	if err != nil {
		return ProcessCredential{}, fmt.Errorf("build plugin host services: %w", err)
	}
	if err := host.Validate(); err != nil {
		return ProcessCredential{}, err
	}
	if host.PluginID != pluginID {
		return ProcessCredential{}, errors.New("plugin host identity mismatch")
	}
	token, err := randomHostSecret(32)
	if err != nil {
		return ProcessCredential{}, err
	}
	key := sha256.Sum256([]byte(token))
	g.mu.Lock()
	g.credentials[key] = hostCredential{pluginID: pluginID, host: host}
	g.mu.Unlock()
	return ProcessCredential{HostURL: g.url, Token: token}, nil
}

func (g *HostGateway) Revoke(token string) {
	if g == nil || strings.TrimSpace(token) == "" {
		return
	}
	key := sha256.Sum256([]byte(strings.TrimSpace(token)))
	g.mu.Lock()
	delete(g.credentials, key)
	ids := make([]string, 0)
	for id, tx := range g.transactions {
		if tx != nil && tx.credentialKey == key {
			ids = append(ids, id)
		}
	}
	g.mu.Unlock()
	for _, id := range ids {
		_ = g.finishTransaction(key, id, false)
	}
}

func (g *HostGateway) Close() error {
	if g == nil {
		return nil
	}
	var closeErr error
	g.closeOnce.Do(func() {
		g.mu.Lock()
		keys := make([][32]byte, 0, len(g.credentials))
		for key := range g.credentials {
			keys = append(keys, key)
		}
		g.credentials = make(map[[32]byte]hostCredential)
		ids := make([]string, 0, len(g.transactions))
		for id := range g.transactions {
			ids = append(ids, id)
		}
		g.mu.Unlock()
		for _, id := range ids {
			for _, key := range keys {
				_ = g.finishTransaction(key, id, false)
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		closeErr = g.server.Shutdown(ctx)
	})
	return closeErr
}

func (g *HostGateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeHostError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	contentType := strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0])
	if !strings.EqualFold(contentType, "application/json") || r.URL.RawQuery != "" {
		writeHostError(w, http.StatusBadRequest, "host_request_invalid")
		return
	}
	if !requestFromLoopback(r.RemoteAddr) {
		writeHostError(w, http.StatusForbidden, "loopback_required")
		return
	}
	credentialKey, credential, ok := g.authenticate(r.Header.Get(pluginclient.AuthorizationHeader))
	if !ok {
		writeHostError(w, http.StatusUnauthorized, "plugin_identity_invalid")
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/"), "/"), "/")
	if len(parts) != 3 || parts[0] != pluginclient.HostAPIVersion {
		writeHostError(w, http.StatusNotFound, "host_operation_not_found")
		return
	}
	if r.ContentLength > maxHostRequestBytes {
		writeHostError(w, http.StatusRequestEntityTooLarge, "host_request_too_large")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxHostRequestBytes)

	if parts[1] == "transactions" {
		g.serveTransaction(w, r, credentialKey, credential, parts[2])
		return
	}
	ctx, unlock, err := g.callContext(r, credentialKey, credential.pluginID)
	if err != nil {
		writeHostError(w, http.StatusForbidden, "host_context_invalid")
		return
	}
	if unlock != nil {
		defer unlock()
	}
	result, err := dispatchHostCall(ctx, credential.host, parts[1], parts[2], json.NewDecoder(r.Body))
	if err != nil {
		writeHostCallError(w, err)
		return
	}
	writeHostJSON(w, http.StatusOK, result)
}

func (g *HostGateway) callContext(r *http.Request, key [32]byte, pluginID string) (context.Context, func(), error) {
	ctx := r.Context()
	if txID := strings.TrimSpace(r.Header.Get(pluginclient.TransactionHeader)); txID != "" {
		g.mu.RLock()
		tx := g.transactions[txID]
		g.mu.RUnlock()
		if tx == nil || tx.credentialKey != key || tx.pluginID != pluginID {
			return nil, nil, errors.New("plugin transaction is invalid")
		}
		tx.callMu.Lock()
		ctx = tx.ctx
		unlock := tx.callMu.Unlock
		ctx, err := g.withUserClaims(ctx, r.Header.Get(pluginclient.UserTokenHeader))
		if err != nil {
			unlock()
			return nil, nil, err
		}
		return ctx, unlock, nil
	}
	ctx, err := g.withUserClaims(ctx, r.Header.Get(pluginclient.UserTokenHeader))
	return ctx, nil, err
}

func (g *HostGateway) withUserClaims(ctx context.Context, token string) (context.Context, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return ctx, nil
	}
	claims, err := security.ParseJWT(g.jwtSecret, token)
	if err != nil {
		return nil, err
	}
	return security.WithJWTClaimsContext(ctx, claims), nil
}

func (g *HostGateway) authenticate(header string) ([32]byte, hostCredential, bool) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return [32]byte{}, hostCredential{}, false
	}
	key := sha256.Sum256([]byte(strings.TrimSpace(strings.TrimPrefix(header, prefix))))
	g.mu.RLock()
	credential, ok := g.credentials[key]
	g.mu.RUnlock()
	return key, credential, ok
}

func (g *HostGateway) serveTransaction(w http.ResponseWriter, r *http.Request, key [32]byte, credential hostCredential, operation string) {
	switch operation {
	case "start":
		if strings.TrimSpace(r.Header.Get(pluginclient.TransactionHeader)) != "" {
			writeHostError(w, http.StatusConflict, "nested_transaction_forbidden")
			return
		}
		if err := decodeHostInput(json.NewDecoder(r.Body), &struct{}{}); err != nil {
			writeHostError(w, http.StatusBadRequest, "transaction_start_invalid")
			return
		}
		id, err := g.startTransaction(key, credential)
		if err != nil {
			writeHostError(w, http.StatusConflict, "transaction_start_failed")
			return
		}
		writeHostJSON(w, http.StatusCreated, pluginclient.TransactionStartResponse{ID: id})
	case "finish":
		id := strings.TrimSpace(r.Header.Get(pluginclient.TransactionHeader))
		var input pluginclient.TransactionFinishRequest
		if id == "" || decodeHostInput(json.NewDecoder(r.Body), &input) != nil {
			writeHostError(w, http.StatusBadRequest, "transaction_finish_invalid")
			return
		}
		if err := g.finishTransaction(key, id, input.Commit); err != nil {
			writeHostError(w, http.StatusConflict, "transaction_finish_failed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeHostError(w, http.StatusNotFound, "host_operation_not_found")
	}
}

func (g *HostGateway) startTransaction(key [32]byte, credential hostCredential) (string, error) {
	id, err := randomHostSecret(24)
	if err != nil {
		return "", err
	}
	ready := make(chan context.Context, 1)
	tx := &hostTransaction{id: id, credentialKey: key, pluginID: credential.pluginID, decision: make(chan bool, 1), done: make(chan error, 1)}
	go func() {
		err := credential.host.Transactions.Within(context.Background(), func(active pluginsdk.Transaction) error {
			if active == nil || active.Context() == nil {
				return errors.New("plugin transaction context is unavailable")
			}
			ready <- active.Context()
			if commit := <-tx.decision; !commit {
				return errRemoteTransactionRollback
			}
			return nil
		})
		if errors.Is(err, errRemoteTransactionRollback) {
			err = nil
		}
		tx.done <- err
	}()
	select {
	case tx.ctx = <-ready:
	case err := <-tx.done:
		if err == nil {
			err = errors.New("plugin transaction ended before ready")
		}
		return "", err
	case <-time.After(5 * time.Second):
		tx.decision <- false
		return "", errors.New("plugin transaction start timed out")
	}
	g.mu.Lock()
	g.transactions[id] = tx
	g.mu.Unlock()
	tx.timer = time.AfterFunc(g.txTTL, func() { _ = g.finishTransaction(key, id, false) })
	return id, nil
}

func (g *HostGateway) finishTransaction(key [32]byte, id string, commit bool) error {
	g.mu.Lock()
	tx := g.transactions[id]
	if tx == nil || tx.credentialKey != key {
		g.mu.Unlock()
		return errors.New("plugin transaction is invalid")
	}
	delete(g.transactions, id)
	g.mu.Unlock()
	if tx.timer != nil {
		tx.timer.Stop()
	}
	tx.callMu.Lock()
	defer tx.callMu.Unlock()
	tx.decision <- commit
	return <-tx.done
}

func randomHostSecret(size int) (string, error) {
	raw := make([]byte, size)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate plugin host credential: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func requestFromLoopback(remote string) bool {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func writeHostJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	var raw []byte
	if value != nil {
		encoded, err := json.Marshal(value)
		if err != nil || len(encoded) > maxHostRequestBytes {
			status = http.StatusInternalServerError
			raw, _ = json.Marshal(pluginclient.ErrorResponse{Code: "host_response_invalid", Message: "plugin host request failed"})
		} else {
			raw = encoded
		}
	}
	w.WriteHeader(status)
	if raw != nil {
		_, _ = w.Write(raw)
	}
}
func writeHostError(w http.ResponseWriter, status int, code string) {
	writeHostJSON(w, status, pluginclient.ErrorResponse{Code: code, Message: "plugin host request failed"})
}

func writeHostCallError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeHostError(w, http.StatusRequestEntityTooLarge, "host_request_too_large")
		return
	}
	var datastoreErr *pluginsdk.DataStoreError
	if errors.As(err, &datastoreErr) {
		status := http.StatusUnprocessableEntity
		switch datastoreErr.Code {
		case pluginsdk.DataStoreErrorInvalidRequest:
			status = http.StatusBadRequest
		case pluginsdk.DataStoreErrorForbidden:
			status = http.StatusForbidden
		case pluginsdk.DataStoreErrorNotFound:
			status = http.StatusNotFound
		case pluginsdk.DataStoreErrorConflict:
			status = http.StatusConflict
		case pluginsdk.DataStoreErrorLimitExceeded:
			status = http.StatusUnprocessableEntity
		case pluginsdk.DataStoreErrorUnavailable:
			status = http.StatusServiceUnavailable
		}
		writeHostJSON(w, status, pluginclient.ErrorResponse{
			Code: string(datastoreErr.Code), Field: datastoreErr.Field,
			Message: datastoreErr.Message, Retryable: datastoreErr.Retryable,
		})
		return
	}
	var numberErr *pluginsdk.DocumentNumberError
	if !errors.As(err, &numberErr) {
		writeHostError(w, http.StatusUnprocessableEntity, "host_call_failed")
		return
	}
	status := http.StatusUnprocessableEntity
	switch numberErr.Code {
	case pluginsdk.DocumentNumberErrorInvalidRequest:
		status = http.StatusBadRequest
	case pluginsdk.DocumentNumberErrorForbidden:
		status = http.StatusForbidden
	case pluginsdk.DocumentNumberErrorConflict:
		status = http.StatusConflict
	case pluginsdk.DocumentNumberErrorTransactionRequired:
		status = http.StatusConflict
	case pluginsdk.DocumentNumberErrorExhausted:
		status = http.StatusUnprocessableEntity
	case pluginsdk.DocumentNumberErrorUnavailable:
		status = http.StatusServiceUnavailable
	}
	writeHostJSON(w, status, pluginclient.ErrorResponse{
		Code: string(numberErr.Code), Field: numberErr.Field,
		Message: numberErr.Message, Retryable: numberErr.Retryable,
	})
}

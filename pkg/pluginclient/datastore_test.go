package pluginclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

func TestDataStoreClientPreservesTypedContractsAndScope(t *testing.T) {
	query := clientQuery()
	mutation := clientMutation()
	page := pluginsdk.DataPage{
		Records: []pluginsdk.DataRecord{{Values: map[string]pluginsdk.DataValue{
			"id":     {Type: pluginsdk.DataValueString, Value: "product-1"},
			"amount": {Type: pluginsdk.DataValueDecimal, Value: "19.95"},
			"meta":   {Type: pluginsdk.DataValueJSON, Value: `{"coldChain":true}`},
		}, Version: 7}},
		NextCursor: "cursor-next", HasMore: true,
	}
	mutationResult := pluginsdk.DataMutationResult{RowsAffected: 1, Record: &page.Records[0]}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(AuthorizationHeader) != "Bearer host-token" || r.Header.Get(UserTokenHeader) != "user-token" {
			t.Errorf("headers=%v", r.Header)
		}
		switch r.URL.Path {
		case "/v1/datastore/query":
			var received pluginsdk.DataQuery
			if err := json.NewDecoder(r.Body).Decode(&received); err != nil || !reflect.DeepEqual(received, query) {
				t.Errorf("query=%+v err=%v", received, err)
			}
			writeClientJSON(t, w, http.StatusOK, page)
		case "/v1/datastore/mutate":
			var received pluginsdk.DataMutation
			if err := json.NewDecoder(r.Body).Decode(&received); err != nil || !reflect.DeepEqual(received, mutation) {
				t.Errorf("mutation=%+v err=%v", received, err)
			}
			writeClientJSON(t, w, http.StatusOK, mutationResult)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	host := newClientHost(t, server)
	ctx := WithUserToken(context.Background(), "user-token")

	gotPage, err := host.DataStore.Query(ctx, query)
	if err != nil || !reflect.DeepEqual(gotPage, page) {
		t.Fatalf("page=%+v err=%v", gotPage, err)
	}
	gotMutation, err := host.DataStore.Mutate(ctx, mutation)
	if err != nil || !reflect.DeepEqual(gotMutation, mutationResult) {
		t.Fatalf("mutation=%+v err=%v", gotMutation, err)
	}
}

func TestDataStoreClientRestoresStableErrorsAndRejectsInvalidResponses(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      any
		wantCode  pluginsdk.DataStoreErrorCode
		wantField string
	}{
		{
			name: "host conflict", status: http.StatusConflict,
			body:     ErrorResponse{Code: "conflict", Field: "expectedVersion", Message: "record changed concurrently"},
			wantCode: pluginsdk.DataStoreErrorConflict, wantField: "expectedVersion",
		},
		{
			name: "invalid success page", status: http.StatusOK,
			body: pluginsdk.DataPage{Records: []pluginsdk.DataRecord{{Values: map[string]pluginsdk.DataValue{
				"id": {Type: pluginsdk.DataValueString, Value: "product-1"},
			}, Version: 1}}},
			wantCode: pluginsdk.DataStoreErrorUnavailable, wantField: "response",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeClientJSON(t, w, test.status, test.body)
			}))
			defer server.Close()
			_, err := newClientHost(t, server).DataStore.Query(context.Background(), clientQuery())
			var datastoreErr *pluginsdk.DataStoreError
			if !errors.As(err, &datastoreErr) || datastoreErr.Code != test.wantCode || datastoreErr.Field != test.wantField {
				t.Fatalf("error=%v", err)
			}
		})
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeClientJSON(t, w, http.StatusBadGateway, map[string]any{"unexpected": true})
	}))
	defer server.Close()
	_, err := newClientHost(t, server).DataStore.Query(context.Background(), clientQuery())
	var clientErr *Error
	if !errors.As(err, &clientErr) || clientErr.Code != "host_response_invalid" {
		t.Fatalf("error=%v", err)
	}

	strictServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"records":[],"hasMore":false,"unexpected":true}`))
	}))
	defer strictServer.Close()
	if _, err = newClientHost(t, strictServer).DataStore.Query(context.Background(), clientQuery()); err == nil {
		t.Fatal("unknown success response field was accepted")
	}
}

func TestDataStoreClientPropagatesTransactionIdentity(t *testing.T) {
	const transactionID = "transaction-7"
	var mu sync.Mutex
	dataCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/transactions/start":
			writeClientJSON(t, w, http.StatusCreated, TransactionStartResponse{ID: transactionID})
		case "/v1/datastore/query":
			if r.Header.Get(TransactionHeader) != transactionID {
				t.Errorf("transaction header=%q", r.Header.Get(TransactionHeader))
			}
			mu.Lock()
			dataCalls++
			mu.Unlock()
			writeClientJSON(t, w, http.StatusOK, pluginsdk.DataPage{})
		case "/v1/transactions/finish":
			if r.Header.Get(TransactionHeader) != transactionID {
				t.Errorf("finish transaction header=%q", r.Header.Get(TransactionHeader))
			}
			var finish TransactionFinishRequest
			if err := json.NewDecoder(r.Body).Decode(&finish); err != nil || !finish.Commit {
				t.Errorf("finish=%+v err=%v", finish, err)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	host := newClientHost(t, server)
	err := host.Transactions.Within(context.Background(), func(tx pluginsdk.Transaction) error {
		_, queryErr := host.DataStore.Query(tx.Context(), clientQuery())
		return queryErr
	})
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if dataCalls != 1 {
		t.Fatalf("datastore calls=%d", dataCalls)
	}
}

func TestDataStoreClientHonorsCancellationAndDeadline(t *testing.T) {
	started := make(chan struct{}, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		started <- struct{}{}
		time.Sleep(100 * time.Millisecond)
		writeClientJSON(t, w, http.StatusOK, pluginsdk.DataPage{})
	}))
	defer server.Close()
	host := newClientHost(t, server)

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := host.DataStore.Query(canceled, clientQuery())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled error=%v", err)
	}

	deadline, stop := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer stop()
	_, err = host.DataStore.Query(deadline, clientQuery())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline error=%v", err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("deadline request did not reach the host")
	}
}

func newClientHost(t *testing.T, server *httptest.Server) pluginsdk.HostServices {
	t.Helper()
	client, err := New(Options{PluginID: "medical_oa", HostURL: server.URL, HostToken: "host-token", HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	host, err := client.HostServices()
	if err != nil {
		t.Fatal(err)
	}
	return host
}

func clientQuery() pluginsdk.DataQuery {
	return pluginsdk.DataQuery{
		Table: "products", Fields: []string{"id", "amount", "meta"},
		Scope: pluginsdk.DataScopeIntent{
			Permission: pluginsdk.Permission{Resource: "medical_oa.product", Action: "read"},
			Filter:     pluginsdk.ScopeFilter{TenantIDs: []string{"tenant-a"}, OrganizationIDs: []string{"org-a"}},
		},
		Filter: &pluginsdk.DataFilter{Field: "amount", Operator: pluginsdk.DataOperatorGreaterOrEq, Value: &pluginsdk.DataValue{Type: pluginsdk.DataValueDecimal, Value: "10.00"}},
		Sort:   []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 25, Cursor: "cursor-current"},
	}
}

func clientMutation() pluginsdk.DataMutation {
	version := int64(6)
	return pluginsdk.DataMutation{
		Table: "products", Operation: pluginsdk.DataMutationUpdate,
		Scope: pluginsdk.DataScopeIntent{
			Permission: pluginsdk.Permission{Resource: "medical_oa.product", Action: "write"},
			Filter:     pluginsdk.ScopeFilter{TenantIDs: []string{"tenant-a"}, OrganizationIDs: []string{"org-a"}, OwnerIDs: []string{"employee-1"}},
		},
		Key: map[string]pluginsdk.DataValue{"id": {Type: pluginsdk.DataValueString, Value: "product-1"}},
		Values: map[string]pluginsdk.DataValue{
			"amount": {Type: pluginsdk.DataValueDecimal, Value: "19.95"},
			"meta":   {Type: pluginsdk.DataValueJSON, Value: `{"coldChain":true}`},
		},
		Returning: []string{"id", "amount", "meta"}, IdempotencyKey: "update-product-1", ExpectedVersion: &version,
	}
}

func writeClientJSON(t *testing.T, w http.ResponseWriter, status int, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Errorf("encode response: %v", err)
	}
}

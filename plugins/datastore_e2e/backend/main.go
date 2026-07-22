package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/tinboxw/skoll/pkg/pluginclient"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type createRecordRequest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func main() {
	client, err := pluginclient.FromEnvironment()
	if err != nil {
		panic(err)
	}
	host, err := client.HostServices()
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mux.HandleFunc("POST /v1/plugins/datastore_e2e/api/records", func(w http.ResponseWriter, r *http.Request) {
		var input createRecordRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil || strings.TrimSpace(input.ID) == "" || strings.TrimSpace(input.Name) == "" {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		ctx := pluginclient.WithUserToken(r.Context(), bearerToken(r))
		var result pluginsdk.DataMutationResult
		err := host.Transactions.Within(ctx, func(tx pluginsdk.Transaction) error {
			var mutationErr error
			result, mutationErr = host.DataStore.Mutate(tx.Context(), pluginsdk.DataMutation{
				Table: "records", Operation: pluginsdk.DataMutationInsert,
				Scope:     pluginsdk.DataScopeIntent{Permission: pluginsdk.Permission{Resource: "datastore_e2e.record", Action: "write"}},
				Key:       map[string]pluginsdk.DataValue{"id": {Type: pluginsdk.DataValueString, Value: strings.TrimSpace(input.ID)}},
				Values:    map[string]pluginsdk.DataValue{"name": {Type: pluginsdk.DataValueString, Value: strings.TrimSpace(input.Name)}},
				Returning: []string{"id", "name"}, IdempotencyKey: strings.TrimSpace(input.ID) + ".create",
			})
			return mutationErr
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, result)
	})
	mux.HandleFunc("GET /v1/plugins/datastore_e2e/api/records", func(w http.ResponseWriter, r *http.Request) {
		queryRecords(w, r, host, pluginsdk.ScopeFilter{})
	})
	mux.HandleFunc("GET /v1/plugins/datastore_e2e/api/records/forged", func(w http.ResponseWriter, r *http.Request) {
		queryRecords(w, r, host, pluginsdk.ScopeFilter{TenantIDs: []string{"tenant-b"}})
	})

	server := &http.Server{Addr: os.Getenv(pluginclient.EnvironmentPluginAddress), Handler: mux}
	go func() { _ = server.ListenAndServe() }()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	<-signals
	_ = server.Close()
}

func queryRecords(w http.ResponseWriter, r *http.Request, host pluginsdk.HostServices, filter pluginsdk.ScopeFilter) {
	ctx := pluginclient.WithUserToken(r.Context(), bearerToken(r))
	page, err := host.DataStore.Query(ctx, pluginsdk.DataQuery{
		Table: "records", Fields: []string{"id", "name"},
		Scope: pluginsdk.DataScopeIntent{Permission: pluginsdk.Permission{Resource: "datastore_e2e.record", Action: "read"}, Filter: filter},
		Sort:  []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 50},
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func bearerToken(r *http.Request) string {
	return strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	var datastoreErr *pluginsdk.DataStoreError
	if errors.As(err, &datastoreErr) {
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
			status = http.StatusRequestEntityTooLarge
		case pluginsdk.DataStoreErrorUnsupported:
			status = http.StatusUnprocessableEntity
		case pluginsdk.DataStoreErrorUnavailable:
			status = http.StatusServiceUnavailable
		}
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

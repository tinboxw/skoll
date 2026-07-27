package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/tinboxw/skoll/pkg/pluginclient"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type createRecordRequest struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	Quantity int64  `json:"quantity"`
}

type adjustRecordRequest struct {
	ID             string `json:"id"`
	Delta          int64  `json:"delta"`
	IdempotencyKey string `json:"idempotencyKey"`
}

type ledgerRequest struct {
	ID       string `json:"id"`
	RecordID string `json:"recordId"`
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
		status := strings.TrimSpace(input.Status)
		if status == "" {
			status = "active"
		}
		var result pluginsdk.DataMutationResult
		err := host.Transactions.Within(ctx, func(tx pluginsdk.Transaction) error {
			var mutationErr error
			result, mutationErr = host.DataStore.Mutate(tx.Context(), pluginsdk.DataMutation{
				Table: "records", Operation: pluginsdk.DataMutationInsert,
				Scope: pluginsdk.DataScopeIntent{Permission: pluginsdk.Permission{Resource: "datastore_e2e.record", Action: "write"}},
				Key:   map[string]pluginsdk.DataValue{"id": {Type: pluginsdk.DataValueString, Value: strings.TrimSpace(input.ID)}},
				Values: map[string]pluginsdk.DataValue{
					"name":     {Type: pluginsdk.DataValueString, Value: strings.TrimSpace(input.Name)},
					"status":   {Type: pluginsdk.DataValueString, Value: status},
					"quantity": {Type: pluginsdk.DataValueInteger, Value: formatInteger(input.Quantity)},
				},
				Returning: []string{"id", "name", "status", "quantity"}, IdempotencyKey: strings.TrimSpace(input.ID) + ".create",
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
	mux.HandleFunc("POST /v1/plugins/datastore_e2e/api/records/adjust", func(w http.ResponseWriter, r *http.Request) {
		var input adjustRecordRequest
		if err := decodeStrict(r, &input); err != nil || strings.TrimSpace(input.ID) == "" || input.Delta == 0 || strings.TrimSpace(input.IdempotencyKey) == "" {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		ctx := pluginclient.WithUserToken(r.Context(), bearerToken(r))
		result, err := host.DataStore.Mutate(ctx, pluginsdk.DataMutation{
			Table: "records", Operation: pluginsdk.DataMutationAdjust,
			Scope: pluginsdk.DataScopeIntent{Permission: pluginsdk.Permission{Resource: "datastore_e2e.record", Action: "write"}},
			Key:   map[string]pluginsdk.DataValue{"id": {Type: pluginsdk.DataValueString, Value: strings.TrimSpace(input.ID)}},
			Adjustment: &pluginsdk.DataAdjustment{
				Field: "quantity", Delta: pluginsdk.DataValue{Type: pluginsdk.DataValueInteger, Value: formatInteger(input.Delta)},
			},
			Returning: []string{"id", "quantity"}, IdempotencyKey: strings.TrimSpace(input.IdempotencyKey),
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	})
	mux.HandleFunc("GET /v1/plugins/datastore_e2e/api/records/aggregate", func(w http.ResponseWriter, r *http.Request) {
		ctx := pluginclient.WithUserToken(r.Context(), bearerToken(r))
		page, err := host.DataStore.Aggregate(ctx, pluginsdk.DataAggregateQuery{
			Table: "records",
			Scope: pluginsdk.DataScopeIntent{Permission: pluginsdk.Permission{Resource: "datastore_e2e.record", Action: "read"}},
			Metrics: []pluginsdk.DataAggregateMetric{
				{Operation: pluginsdk.DataAggregateCount},
				{Operation: pluginsdk.DataAggregateSum, Field: "quantity"},
			},
			GroupBy: []string{"status"}, Page: pluginsdk.DataPageRequest{Limit: 50},
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, page)
	})
	mux.HandleFunc("POST /v1/plugins/datastore_e2e/api/records/rollback", func(w http.ResponseWriter, r *http.Request) {
		var input createRecordRequest
		if err := decodeStrict(r, &input); err != nil || strings.TrimSpace(input.ID) == "" || strings.TrimSpace(input.Name) == "" {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		ctx := pluginclient.WithUserToken(r.Context(), bearerToken(r))
		err := host.Transactions.Within(ctx, func(tx pluginsdk.Transaction) error {
			_, mutationErr := host.DataStore.Mutate(tx.Context(), pluginsdk.DataMutation{
				Table: "records", Operation: pluginsdk.DataMutationInsert,
				Scope: pluginsdk.DataScopeIntent{Permission: pluginsdk.Permission{Resource: "datastore_e2e.record", Action: "write"}},
				Key:   map[string]pluginsdk.DataValue{"id": {Type: pluginsdk.DataValueString, Value: strings.TrimSpace(input.ID)}},
				Values: map[string]pluginsdk.DataValue{
					"name":     {Type: pluginsdk.DataValueString, Value: strings.TrimSpace(input.Name)},
					"status":   {Type: pluginsdk.DataValueString, Value: "rollback"},
					"quantity": {Type: pluginsdk.DataValueInteger, Value: "1"},
				},
				IdempotencyKey: strings.TrimSpace(input.ID) + ".rollback",
			})
			if mutationErr != nil {
				return mutationErr
			}
			return errors.New("forced rollback")
		})
		writeError(w, err)
	})
	mux.HandleFunc("POST /v1/plugins/datastore_e2e/api/ledger", func(w http.ResponseWriter, r *http.Request) {
		var input ledgerRequest
		if err := decodeStrict(r, &input); err != nil || strings.TrimSpace(input.ID) == "" || strings.TrimSpace(input.RecordID) == "" {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		ctx := pluginclient.WithUserToken(r.Context(), bearerToken(r))
		result, err := host.DataStore.Mutate(ctx, pluginsdk.DataMutation{
			Table: "ledger_entries", Operation: pluginsdk.DataMutationInsert,
			Scope: pluginsdk.DataScopeIntent{Permission: pluginsdk.Permission{Resource: "datastore_e2e.ledger", Action: "write"}},
			Key:   map[string]pluginsdk.DataValue{"id": {Type: pluginsdk.DataValueString, Value: strings.TrimSpace(input.ID)}},
			Values: map[string]pluginsdk.DataValue{
				"record_id": {Type: pluginsdk.DataValueString, Value: strings.TrimSpace(input.RecordID)},
			},
			Returning: []string{"id", "record_id"}, IdempotencyKey: strings.TrimSpace(input.ID) + ".append",
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, result)
	})
	mux.HandleFunc("DELETE /v1/plugins/datastore_e2e/api/ledger", func(w http.ResponseWriter, r *http.Request) {
		var input ledgerRequest
		if err := decodeStrict(r, &input); err != nil || strings.TrimSpace(input.ID) == "" {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		ctx := pluginclient.WithUserToken(r.Context(), bearerToken(r))
		_, err := host.DataStore.Mutate(ctx, pluginsdk.DataMutation{
			Table: "ledger_entries", Operation: pluginsdk.DataMutationDelete,
			Scope:          pluginsdk.DataScopeIntent{Permission: pluginsdk.Permission{Resource: "datastore_e2e.ledger", Action: "write"}},
			Key:            map[string]pluginsdk.DataValue{"id": {Type: pluginsdk.DataValueString, Value: strings.TrimSpace(input.ID)}},
			IdempotencyKey: strings.TrimSpace(input.ID) + ".delete",
		})
		if err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	server := &http.Server{Addr: os.Getenv(pluginclient.EnvironmentPluginAddress), Handler: mux}
	go func() { _ = server.ListenAndServe() }()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	<-signals
	_ = server.Close()
}

func decodeStrict(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func formatInteger(value int64) string {
	return strconv.FormatInt(value, 10)
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

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginclient"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const (
	pluginID = "equipment_maintenance"
	apiBase  = "/v1/plugins/equipment_maintenance/api"
)

type Host struct {
	Transactions pluginsdk.TransactionService
	DataScopes   pluginsdk.DataScopeService
	Files        pluginsdk.FileService
	Audit        pluginsdk.AuditService
	Config       pluginsdk.ConfigService
	Workflows    pluginsdk.WorkflowService
	Jobs         pluginsdk.JobService
}

type Server struct {
	store *Store
	host  Host
	nowFn func() time.Time
	bind  func(context.Context, string) context.Context
}

func NewServer(store *Store, host Host) (*Server, error) {
	if store == nil {
		return nil, errors.New("equipment maintenance store is required")
	}
	if host.Transactions == nil || host.DataScopes == nil || host.Files == nil || host.Audit == nil || host.Config == nil || host.Workflows == nil || host.Jobs == nil {
		return nil, errors.New("complete equipment maintenance host services are required")
	}
	return &Server{store: store, host: host, nowFn: time.Now, bind: pluginclient.WithUserToken}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("POST /_skoll/events", s.handleEvent)
	mux.HandleFunc("GET "+apiBase+"/assets", s.listAssets)
	mux.HandleFunc("POST "+apiBase+"/assets", s.createAsset)
	mux.HandleFunc("GET "+apiBase+"/assets/{id}", s.getAsset)
	mux.HandleFunc("PUT "+apiBase+"/assets/{id}", s.updateAsset)
	mux.HandleFunc("POST "+apiBase+"/assets/{id}/retire", s.retireAsset)
	mux.HandleFunc("GET "+apiBase+"/work-orders", s.listWorkOrders)
	mux.HandleFunc("POST "+apiBase+"/work-orders", s.createWorkOrder)
	mux.HandleFunc("GET "+apiBase+"/work-orders/{id}", s.getWorkOrder)
	mux.HandleFunc("PUT "+apiBase+"/work-orders/{id}", s.updateWorkOrder)
	mux.HandleFunc("POST "+apiBase+"/work-orders/{id}/dispatch", s.dispatchWorkOrder)
	mux.HandleFunc("POST "+apiBase+"/work-orders/{id}/start", s.startWorkOrder)
	mux.HandleFunc("POST "+apiBase+"/work-orders/{id}/submit", s.submitWorkOrder)
	mux.HandleFunc("POST "+apiBase+"/work-orders/{id}/complete", s.completeWorkOrder)
	mux.HandleFunc("POST "+apiBase+"/work-orders/{id}/close", s.closeWorkOrder)
	mux.HandleFunc("GET "+apiBase+"/maintenance-plans", s.listPlans)
	mux.HandleFunc("POST "+apiBase+"/maintenance-plans", s.createPlan)
	mux.HandleFunc("PUT "+apiBase+"/maintenance-plans/{id}", s.updatePlan)
	mux.HandleFunc("GET "+apiBase+"/inspections", s.listInspections)
	mux.HandleFunc("POST "+apiBase+"/inspections", s.createInspection)
	mux.HandleFunc("GET "+apiBase+"/inspections/{id}", s.getInspection)
	mux.HandleFunc("POST "+apiBase+"/inspections/{id}/complete", s.completeInspection)
	mux.HandleFunc("GET "+apiBase+"/spare-parts", s.listSpareParts)
	mux.HandleFunc("POST "+apiBase+"/spare-parts", s.createSparePart)
	mux.HandleFunc("PUT "+apiBase+"/spare-parts/{id}", s.updateSparePart)
	mux.HandleFunc("GET "+apiBase+"/spare-movements", s.listSpareMovements)
	mux.HandleFunc("POST "+apiBase+"/spare-movements", s.createSpareMovement)
	mux.HandleFunc("GET "+apiBase+"/dashboard", s.dashboard)
	mux.HandleFunc("GET "+apiBase+"/dashboard/trends", s.dashboardTrends)
	mux.HandleFunc("GET "+apiBase+"/jobs", s.listJobs)
	mux.HandleFunc("POST "+apiBase+"/jobs/maintenance-due-scan", s.scheduleMaintenanceScan)
	mux.HandleFunc("POST "+apiBase+"/jobs/spare-stock-scan", s.scheduleStockScan)
	return mux
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, envelope(map[string]any{"status": "ready", "pluginId": pluginID}))
}

func (s *Server) requestContext(r *http.Request) (context.Context, error) {
	value := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(value) < 8 || !strings.EqualFold(value[:7], "Bearer ") {
		return nil, errors.New("bearer token is required")
	}
	token := strings.TrimSpace(value[7:])
	if token == "" {
		return nil, errors.New("bearer token is required")
	}
	return s.bind(r.Context(), token), nil
}

func (s *Server) scope(r *http.Request, permission string) (context.Context, pluginsdk.ScopePredicate, error) {
	ctx, err := s.requestContext(r)
	if err != nil {
		return nil, pluginsdk.ScopePredicate{}, err
	}
	resource, action, ok := strings.Cut(permission, ":")
	if !ok {
		index := strings.LastIndex(permission, ".")
		if index <= 0 || index == len(permission)-1 {
			return nil, pluginsdk.ScopePredicate{}, errors.New("permission is invalid")
		}
		resource, action = permission[:index], permission[index+1:]
	}
	predicate, err := s.host.DataScopes.Resolve(ctx, pluginsdk.Permission{Resource: resource, Action: action})
	if err != nil {
		return nil, pluginsdk.ScopePredicate{}, err
	}
	if predicate.Denied() {
		return nil, pluginsdk.ScopePredicate{}, errors.New("trusted data scope denied")
	}
	return ctx, predicate, nil
}

func trustedWriteScope(predicate pluginsdk.ScopePredicate, requested Scope) (Scope, error) {
	requested.OwnerID = predicate.SubjectID()
	constrained := predicate.Constrain(pluginsdk.ScopeFilter{
		TenantIDs: singleValue(requested.TenantID), OwnerIDs: []string{requested.OwnerID}, OrganizationIDs: singleValue(requested.OrganizationID),
	})
	if constrained.Denied() {
		return Scope{}, errors.New("requested data scope is not allowed")
	}
	tenantID, err := chooseScopeID(requested.TenantID, constrained.TenantIDs(), constrained.AllTenants(), "tenantId")
	if err != nil {
		return Scope{}, err
	}
	organizationID, err := chooseScopeID(requested.OrganizationID, constrained.OrganizationIDs(), constrained.AllOrganizations(), "organizationId")
	if err != nil {
		return Scope{}, err
	}
	return Scope{TenantID: tenantID, OrganizationID: organizationID, OwnerID: predicate.SubjectID()}, nil
}

func chooseScopeID(requested string, trusted []string, all bool, label string) (string, error) {
	requested = strings.TrimSpace(requested)
	if requested != "" {
		if all || contains(trusted, requested) {
			return requested, nil
		}
		return "", fmt.Errorf("%s is outside trusted scope", label)
	}
	if len(trusted) == 1 {
		return trusted[0], nil
	}
	return "", fmt.Errorf("%s is required", label)
}

func allowed(predicate pluginsdk.ScopePredicate, scope Scope) bool {
	return predicate.Allows(pluginsdk.ScopedRecord{TenantID: scope.TenantID, OwnerID: scope.OwnerID, OrganizationID: scope.OrganizationID})
}

func (s *Server) transact(ctx context.Context, fn func(context.Context) error) error {
	return s.host.Transactions.Within(ctx, func(tx pluginsdk.Transaction) error { return fn(tx.Context()) })
}

func (s *Server) audit(ctx context.Context, action, resource, resourceID string, risk pluginsdk.AuditRisk, detail map[string]any) error {
	_, err := s.host.Audit.Record(ctx, pluginsdk.AuditEntry{
		Action: action, Resource: resource, ResourceID: resourceID, Result: pluginsdk.AuditResultSuccess, Risk: risk, Detail: detail,
	})
	return err
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return false
	}
	return true
}

func pagination(r *http.Request) (int, int, error) {
	offset, err := parseInt(r.URL.Query().Get("offset"), 0, 0, 1000000)
	if err != nil {
		return 0, 0, fmt.Errorf("offset: %w", err)
	}
	limit, err := parseInt(r.URL.Query().Get("limit"), 50, 1, 200)
	if err != nil {
		return 0, 0, fmt.Errorf("limit: %w", err)
	}
	return offset, limit, nil
}

func parseInt(raw string, defaultValue, minimum, maximum int) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		return 0, fmt.Errorf("must be between %d and %d", minimum, maximum)
	}
	return value, nil
}

func page[T any](items []T, offset, limit int) []T {
	if offset >= len(items) {
		return []T{}
	}
	end := min(len(items), offset+limit)
	return items[offset:end]
}

func sortByUpdated[T any](items []T, updated func(T) time.Time) {
	sort.Slice(items, func(i, j int) bool { return updated(items[i]).After(updated(items[j])) })
}

func singleValue(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return []string{strings.TrimSpace(value)}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func required(value, label string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required", label)
	}
	return value, nil
}

func envelope(data any) map[string]any {
	return map[string]any{"code": "ok", "message": "", "data": data}
}
func writeOK(w http.ResponseWriter, data any) { writeJSON(w, http.StatusOK, envelope(data)) }
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"code": code, "message": message, "data": nil})
}
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

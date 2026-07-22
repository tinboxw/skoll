package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type sparePartInput struct {
	TenantID        string  `json:"tenantId"`
	OrganizationID  string  `json:"organizationId"`
	SKU             string  `json:"sku"`
	Name            string  `json:"name"`
	Unit            string  `json:"unit"`
	MinimumQuantity float64 `json:"minimumQuantity"`
	Status          string  `json:"status"`
}

type spareMovementInput struct {
	TenantID       string  `json:"tenantId"`
	OrganizationID string  `json:"organizationId"`
	SparePartID    string  `json:"sparePartId"`
	WorkOrderID    string  `json:"workOrderId"`
	MovementType   string  `json:"movementType"`
	Quantity       float64 `json:"quantity"`
	IdempotencyKey string  `json:"idempotencyKey"`
}

func (s *Server) listSpareParts(w http.ResponseWriter, r *http.Request) {
	_, predicate, err := s.scope(r, "equipment_maintenance.spare_part.read")
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	offset, limit, err := pagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_pagination", err.Error())
		return
	}
	keyword := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("keyword")))
	items := make([]SparePart, 0)
	_ = s.store.View(func(state State) error {
		for _, item := range state.SpareParts {
			if allowed(predicate, item.Scope) && (keyword == "" || strings.Contains(strings.ToLower(item.SKU+" "+item.Name), keyword)) {
				items = append(items, item)
			}
		}
		return nil
	})
	sortByUpdated(items, func(item SparePart) time.Time { return item.UpdatedAt })
	writeOK(w, map[string]any{"items": page(items, offset, limit), "total": len(items), "offset": offset, "limit": limit})
}

func (s *Server) createSparePart(w http.ResponseWriter, r *http.Request) {
	ctx, predicate, err := s.scope(r, "equipment_maintenance.spare_part.manage")
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	var input sparePartInput
	if !decodeJSON(w, r, &input) {
		return
	}
	scope, err := trustedWriteScope(predicate, Scope{TenantID: input.TenantID, OrganizationID: input.OrganizationID})
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	input.SKU, err = required(input.SKU, "sku")
	if err == nil {
		input.Name, err = required(input.Name, "name")
	}
	if err == nil {
		input.Unit, err = required(input.Unit, "unit")
	}
	if err != nil || input.MinimumQuantity < 0 {
		writeError(w, http.StatusBadRequest, "invalid_spare_part", "sku, name, unit, and non-negative minimumQuantity are required")
		return
	}
	now := s.nowFn().UTC()
	var created SparePart
	err = s.transact(ctx, func(tx context.Context) error {
		return s.store.Update(func(state *State) error {
			for _, current := range state.SpareParts {
				if current.TenantID == scope.TenantID && strings.EqualFold(current.SKU, input.SKU) {
					return errors.New("spare-part SKU already exists")
				}
			}
			status := strings.ToLower(strings.TrimSpace(input.Status))
			if status == "" {
				status = "active"
			}
			created = SparePart{ID: nextID(state, "spare"), Scope: scope, SKU: input.SKU, Name: input.Name, Unit: input.Unit, MinimumQuantity: input.MinimumQuantity, Status: status, CreatedAt: now, UpdatedAt: now}
			state.SpareParts[created.ID] = created
			return s.audit(tx, "equipment_maintenance.spare_part.create", "spare_part", created.ID, pluginsdk.AuditRiskMedium, map[string]any{"sku": created.SKU})
		})
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "spare_part_create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, envelope(map[string]any{"item": created}))
}

func (s *Server) updateSparePart(w http.ResponseWriter, r *http.Request) {
	ctx, predicate, err := s.scope(r, "equipment_maintenance.spare_part.manage")
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	var input sparePartInput
	if !decodeJSON(w, r, &input) {
		return
	}
	input.SKU, err = required(input.SKU, "sku")
	if err == nil {
		input.Name, err = required(input.Name, "name")
	}
	if err == nil {
		input.Unit, err = required(input.Unit, "unit")
	}
	if err != nil || input.MinimumQuantity < 0 {
		writeError(w, http.StatusBadRequest, "invalid_spare_part", "sku, name, unit, and non-negative minimumQuantity are required")
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	var updated SparePart
	err = s.transact(ctx, func(tx context.Context) error {
		return s.store.Update(func(state *State) error {
			current, ok := state.SpareParts[id]
			if !ok || !allowed(predicate, current.Scope) {
				return errors.New("spare part not found")
			}
			for otherID, other := range state.SpareParts {
				if otherID != id && other.TenantID == current.TenantID && strings.EqualFold(other.SKU, input.SKU) {
					return errors.New("spare-part SKU already exists")
				}
			}
			current.SKU, current.Name, current.Unit, current.MinimumQuantity = input.SKU, input.Name, input.Unit, input.MinimumQuantity
			if status := strings.ToLower(strings.TrimSpace(input.Status)); status != "" {
				current.Status = status
			}
			current.UpdatedAt = s.nowFn().UTC()
			state.SpareParts[id], updated = current, current
			return s.audit(tx, "equipment_maintenance.spare_part.update", "spare_part", id, pluginsdk.AuditRiskMedium, nil)
		})
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "spare_part_update_failed", err.Error())
		return
	}
	writeOK(w, map[string]any{"item": updated})
}

func (s *Server) listSpareMovements(w http.ResponseWriter, r *http.Request) {
	_, predicate, err := s.scope(r, "equipment_maintenance.spare_part.read")
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	offset, limit, err := pagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_pagination", err.Error())
		return
	}
	partID := strings.TrimSpace(r.URL.Query().Get("sparePartId"))
	items := make([]SpareMovement, 0)
	_ = s.store.View(func(state State) error {
		for _, item := range state.SpareMovements {
			if allowed(predicate, item.Scope) && (partID == "" || item.SparePartID == partID) {
				items = append(items, item)
			}
		}
		return nil
	})
	sort.Slice(items, func(i, j int) bool { return items[i].OccurredAt.After(items[j].OccurredAt) })
	writeOK(w, map[string]any{"items": page(items, offset, limit), "total": len(items), "offset": offset, "limit": limit})
}

func (s *Server) createSpareMovement(w http.ResponseWriter, r *http.Request) {
	ctx, predicate, err := s.scope(r, "equipment_maintenance.spare_part.move")
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	var input spareMovementInput
	if !decodeJSON(w, r, &input) {
		return
	}
	scope, err := trustedWriteScope(predicate, Scope{TenantID: input.TenantID, OrganizationID: input.OrganizationID})
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	input.SparePartID, err = required(input.SparePartID, "sparePartId")
	if err == nil {
		input.IdempotencyKey, err = required(input.IdempotencyKey, "idempotencyKey")
	}
	typeName := strings.ToLower(strings.TrimSpace(input.MovementType))
	if err != nil || input.Quantity <= 0 || !contains([]string{"inbound", "outbound", "usage", "adjustment"}, typeName) {
		writeError(w, http.StatusBadRequest, "invalid_movement", "valid type, positive quantity, and idempotencyKey are required")
		return
	}
	now := s.nowFn().UTC()
	var movement SpareMovement
	var part SparePart
	duplicate := false
	err = s.transact(ctx, func(tx context.Context) error {
		return s.store.Update(func(state *State) error {
			key := scope.TenantID + "|" + input.IdempotencyKey
			if movementID, ok := state.MovementKeys[key]; ok {
				movement, duplicate = state.SpareMovements[movementID], true
				part = state.SpareParts[movement.SparePartID]
				return nil
			}
			current, ok := state.SpareParts[input.SparePartID]
			if !ok || !allowed(predicate, current.Scope) || current.Status != "active" {
				return errors.New("active spare part not found")
			}
			delta := input.Quantity
			if typeName == "outbound" || typeName == "usage" {
				delta = -input.Quantity
			}
			if current.Quantity+delta < 0 {
				return errors.New("movement would produce negative stock")
			}
			if input.WorkOrderID != "" {
				order, ok := state.WorkOrders[input.WorkOrderID]
				if !ok || !allowed(predicate, order.Scope) {
					return errors.New("work order not found")
				}
			}
			current.Quantity, current.UpdatedAt = current.Quantity+delta, now
			state.SpareParts[current.ID], part = current, current
			movement = SpareMovement{ID: nextID(state, "movement"), Scope: scope, SparePartID: current.ID, WorkOrderID: strings.TrimSpace(input.WorkOrderID), MovementType: typeName, Quantity: input.Quantity, IdempotencyKey: input.IdempotencyKey, OccurredAt: now, CreatedAt: now}
			state.SpareMovements[movement.ID], state.MovementKeys[key] = movement, movement.ID
			return s.audit(tx, "equipment_maintenance.spare_movement.create", "spare_movement", movement.ID, pluginsdk.AuditRiskHigh, map[string]any{"sparePartId": current.ID, "movementType": typeName, "quantity": input.Quantity})
		})
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "movement_create_failed", err.Error())
		return
	}
	status := http.StatusCreated
	if duplicate {
		status = http.StatusOK
	}
	writeJSON(w, status, envelope(map[string]any{"item": movement, "sparePart": part, "duplicate": duplicate}))
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	ctx, predicate, err := s.scope(r, "equipment_maintenance.dashboard.read")
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	threshold, err := s.lowStockThreshold(ctx)
	if err != nil {
		writeError(w, http.StatusBadGateway, "config_failed", err.Error())
		return
	}
	now := s.nowFn().UTC()
	metrics := map[string]int{"assets": 0, "openWorkOrders": 0, "overduePlans": 0, "lowStockParts": 0, "completedInspections": 0}
	_ = s.store.View(func(state State) error {
		for _, item := range state.Assets {
			if allowed(predicate, item.Scope) && item.Status != "retired" {
				metrics["assets"]++
			}
		}
		for _, item := range state.WorkOrders {
			if allowed(predicate, item.Scope) && item.Status != "closed" {
				metrics["openWorkOrders"]++
			}
		}
		for _, item := range state.Plans {
			if allowed(predicate, item.Scope) && item.Status == "active" && item.NextRunAt.Before(now) {
				metrics["overduePlans"]++
			}
		}
		for _, item := range state.SpareParts {
			if allowed(predicate, item.Scope) && item.Status == "active" && item.Quantity <= max(item.MinimumQuantity, threshold) {
				metrics["lowStockParts"]++
			}
		}
		for _, item := range state.Inspections {
			if allowed(predicate, item.Scope) && item.Status == "completed" {
				metrics["completedInspections"]++
			}
		}
		return nil
	})
	writeOK(w, map[string]any{"metrics": metrics, "generatedAt": now})
}

func (s *Server) dashboardTrends(w http.ResponseWriter, r *http.Request) {
	_, predicate, err := s.scope(r, "equipment_maintenance.dashboard.read")
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	now := s.nowFn().UTC()
	buckets := make([]map[string]any, 0, 6)
	for index := 5; index >= 0; index-- {
		month := now.AddDate(0, -index, 0).Format("2006-01")
		buckets = append(buckets, map[string]any{"month": month, "closedWorkOrders": 0, "inspections": 0})
	}
	_ = s.store.View(func(state State) error {
		for _, order := range state.WorkOrders {
			if allowed(predicate, order.Scope) && order.ClosedAt != nil {
				incrementBucket(buckets, order.ClosedAt.Format("2006-01"), "closedWorkOrders")
			}
		}
		for _, inspection := range state.Inspections {
			if allowed(predicate, inspection.Scope) && inspection.Status == "completed" {
				incrementBucket(buckets, inspection.InspectedAt.Format("2006-01"), "inspections")
			}
		}
		return nil
	})
	writeOK(w, map[string]any{"items": buckets})
}

func incrementBucket(items []map[string]any, month, key string) {
	for _, item := range items {
		if item["month"] == month {
			item[key] = item[key].(int) + 1
			return
		}
	}
}

func (s *Server) lowStockThreshold(ctx context.Context) (float64, error) {
	values, err := s.host.Config.Get(ctx)
	if err != nil {
		return 0, err
	}
	value, ok := values["equipment_maintenance.low_stock_threshold"]
	if !ok {
		return 5, nil
	}
	switch typed := value.(type) {
	case float64:
		return typed, nil
	case int:
		return float64(typed), nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, errors.New("low stock threshold is invalid")
		}
		return parsed, nil
	default:
		return 0, errors.New("low stock threshold is invalid")
	}
}

func (s *Server) listJobs(w http.ResponseWriter, r *http.Request) {
	ctx, _, err := s.scope(r, "equipment_maintenance.job.read")
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	jobs, err := s.host.Jobs.List(ctx, pluginsdk.JobQuery{Kind: strings.TrimSpace(r.URL.Query().Get("kind")), Limit: 100})
	if err != nil {
		writeError(w, http.StatusBadGateway, "jobs_failed", err.Error())
		return
	}
	writeOK(w, map[string]any{"items": jobs})
}

func (s *Server) scheduleMaintenanceScan(w http.ResponseWriter, r *http.Request) {
	s.scheduleScan(w, r, "maintenance.due_scan")
}
func (s *Server) scheduleStockScan(w http.ResponseWriter, r *http.Request) {
	s.scheduleScan(w, r, "spare.low_stock_scan")
}

func (s *Server) scheduleScan(w http.ResponseWriter, r *http.Request, kind string) {
	ctx, predicate, err := s.scope(r, "equipment_maintenance.job.run")
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	var input struct {
		TenantID       string `json:"tenantId"`
		OrganizationID string `json:"organizationId"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	scope, err := trustedWriteScope(predicate, Scope{TenantID: input.TenantID, OrganizationID: input.OrganizationID})
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	date := s.nowFn().UTC().Format("2006-01-02")
	payload, _ := json.Marshal(map[string]any{"tenantId": scope.TenantID, "organizationId": scope.OrganizationID, "requestedBy": scope.OwnerID, "date": date})
	key := strings.ReplaceAll(kind, ".", "-") + ":" + scope.TenantID + ":" + date
	job, err := s.host.Jobs.Schedule(ctx, pluginsdk.JobScheduleInput{ID: key, Kind: kind, IdempotencyKey: key, Payload: payload, RunAt: s.nowFn().UTC(), MaxAttempts: 5})
	if err != nil {
		writeError(w, http.StatusBadGateway, "job_schedule_failed", err.Error())
		return
	}
	if err := s.audit(ctx, "equipment_maintenance.job."+map[string]string{"maintenance.due_scan": "maintenance_scan", "spare.low_stock_scan": "stock_scan"}[kind], "job", job.ID, pluginsdk.AuditRiskHigh, map[string]any{"kind": kind}); err != nil {
		writeError(w, http.StatusBadGateway, "audit_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, envelope(map[string]any{"item": job}))
}

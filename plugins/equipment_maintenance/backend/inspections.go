package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type planInput struct {
	TenantID       string    `json:"tenantId"`
	OrganizationID string    `json:"organizationId"`
	AssetID        string    `json:"assetId"`
	Name           string    `json:"name"`
	IntervalDays   int       `json:"intervalDays"`
	NextRunAt      time.Time `json:"nextRunAt"`
	Status         string    `json:"status"`
}

type inspectionInput struct {
	TenantID       string `json:"tenantId"`
	OrganizationID string `json:"organizationId"`
	PlanID         string `json:"planId"`
	WorkOrderID    string `json:"workOrderId"`
}

type inspectionCompleteInput struct {
	Result        string   `json:"result"`
	Finding       string   `json:"finding"`
	AttachmentIDs []string `json:"attachmentIds"`
}

func (s *Server) listPlans(w http.ResponseWriter, r *http.Request) {
	_, predicate, err := s.scope(r, "equipment_maintenance.inspection.read")
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	offset, limit, err := pagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_pagination", err.Error())
		return
	}
	status := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	items := make([]MaintenancePlan, 0)
	_ = s.store.View(func(state State) error {
		for _, item := range state.Plans {
			if allowed(predicate, item.Scope) && (status == "" || item.Status == status) {
				items = append(items, item)
			}
		}
		return nil
	})
	sortByUpdated(items, func(item MaintenancePlan) time.Time { return item.UpdatedAt })
	writeOK(w, map[string]any{"items": page(items, offset, limit), "total": len(items), "offset": offset, "limit": limit})
}

func (s *Server) createPlan(w http.ResponseWriter, r *http.Request) {
	ctx, predicate, err := s.scope(r, "equipment_maintenance.inspection.plan")
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	var input planInput
	if !decodeJSON(w, r, &input) {
		return
	}
	scope, err := trustedWriteScope(predicate, Scope{TenantID: input.TenantID, OrganizationID: input.OrganizationID})
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	input.AssetID, err = required(input.AssetID, "assetId")
	if err == nil {
		input.Name, err = required(input.Name, "name")
	}
	if err != nil || input.IntervalDays < 1 || input.NextRunAt.IsZero() {
		writeError(w, http.StatusBadRequest, "invalid_plan", "assetId, name, intervalDays, and nextRunAt are required")
		return
	}
	now := s.nowFn().UTC()
	var created MaintenancePlan
	err = s.transact(ctx, func(tx context.Context) error {
		return s.store.Update(func(state *State) error {
			asset, ok := state.Assets[input.AssetID]
			if !ok || !allowed(predicate, asset.Scope) || asset.Status == "retired" {
				return errors.New("active asset not found")
			}
			status := strings.ToLower(strings.TrimSpace(input.Status))
			if status == "" {
				status = "active"
			}
			created = MaintenancePlan{ID: nextID(state, "plan"), Scope: scope, AssetID: input.AssetID, Name: input.Name, IntervalDays: input.IntervalDays, NextRunAt: input.NextRunAt.UTC(), Status: status, CreatedAt: now, UpdatedAt: now}
			state.Plans[created.ID] = created
			return s.audit(tx, "equipment_maintenance.plan.create", "maintenance_plan", created.ID, pluginsdk.AuditRiskMedium, map[string]any{"assetId": created.AssetID})
		})
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "plan_create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, envelope(map[string]any{"item": created}))
}

func (s *Server) updatePlan(w http.ResponseWriter, r *http.Request) {
	ctx, predicate, err := s.scope(r, "equipment_maintenance.inspection.plan")
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	var input planInput
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Name, err = required(input.Name, "name")
	if err != nil || input.IntervalDays < 1 || input.NextRunAt.IsZero() {
		writeError(w, http.StatusBadRequest, "invalid_plan", "name, intervalDays, and nextRunAt are required")
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	var updated MaintenancePlan
	err = s.transact(ctx, func(tx context.Context) error {
		return s.store.Update(func(state *State) error {
			current, ok := state.Plans[id]
			if !ok || !allowed(predicate, current.Scope) {
				return errors.New("maintenance plan not found")
			}
			current.Name, current.IntervalDays, current.NextRunAt = input.Name, input.IntervalDays, input.NextRunAt.UTC()
			if status := strings.ToLower(strings.TrimSpace(input.Status)); status != "" {
				current.Status = status
			}
			current.UpdatedAt = s.nowFn().UTC()
			state.Plans[id], updated = current, current
			return s.audit(tx, "equipment_maintenance.plan.update", "maintenance_plan", id, pluginsdk.AuditRiskMedium, nil)
		})
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "plan_update_failed", err.Error())
		return
	}
	writeOK(w, map[string]any{"item": updated})
}

func (s *Server) listInspections(w http.ResponseWriter, r *http.Request) {
	_, predicate, err := s.scope(r, "equipment_maintenance.inspection.read")
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	offset, limit, err := pagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_pagination", err.Error())
		return
	}
	items := make([]Inspection, 0)
	_ = s.store.View(func(state State) error {
		for _, item := range state.Inspections {
			if allowed(predicate, item.Scope) {
				items = append(items, item)
			}
		}
		return nil
	})
	sortByUpdated(items, func(item Inspection) time.Time { return item.UpdatedAt })
	writeOK(w, map[string]any{"items": page(items, offset, limit), "total": len(items), "offset": offset, "limit": limit})
}

func (s *Server) createInspection(w http.ResponseWriter, r *http.Request) {
	ctx, predicate, err := s.scope(r, "equipment_maintenance.inspection.execute")
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	var input inspectionInput
	if !decodeJSON(w, r, &input) {
		return
	}
	scope, err := trustedWriteScope(predicate, Scope{TenantID: input.TenantID, OrganizationID: input.OrganizationID})
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	input.PlanID, err = required(input.PlanID, "planId")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_inspection", err.Error())
		return
	}
	now := s.nowFn().UTC()
	var created Inspection
	err = s.transact(ctx, func(tx context.Context) error {
		return s.store.Update(func(state *State) error {
			plan, ok := state.Plans[input.PlanID]
			if !ok || !allowed(predicate, plan.Scope) || plan.Status != "active" {
				return errors.New("active maintenance plan not found")
			}
			created = Inspection{ID: nextID(state, "inspection"), Scope: scope, PlanID: plan.ID, AssetID: plan.AssetID, WorkOrderID: strings.TrimSpace(input.WorkOrderID), Status: "in_progress", AttachmentIDs: []string{}, InspectedAt: now, CreatedAt: now, UpdatedAt: now}
			state.Inspections[created.ID] = created
			return s.audit(tx, "equipment_maintenance.inspection.create", "inspection", created.ID, pluginsdk.AuditRiskHigh, map[string]any{"planId": plan.ID})
		})
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "inspection_create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, envelope(map[string]any{"item": created}))
}

func (s *Server) getInspection(w http.ResponseWriter, r *http.Request) {
	_, predicate, err := s.scope(r, "equipment_maintenance.inspection.read")
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	var item Inspection
	found := false
	_ = s.store.View(func(state State) error {
		item, found = state.Inspections[strings.TrimSpace(r.PathValue("id"))]
		found = found && allowed(predicate, item.Scope)
		return nil
	})
	if !found {
		writeError(w, http.StatusNotFound, "inspection_not_found", "inspection not found")
		return
	}
	writeOK(w, map[string]any{"item": item})
}

func (s *Server) completeInspection(w http.ResponseWriter, r *http.Request) {
	ctx, predicate, err := s.scope(r, "equipment_maintenance.inspection.execute")
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	var input inspectionCompleteInput
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Result, err = required(input.Result, "result")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_inspection", err.Error())
		return
	}
	for _, fileID := range input.AttachmentIDs {
		if _, err := s.host.Files.Get(ctx, strings.TrimSpace(fileID)); err != nil {
			writeError(w, http.StatusBadRequest, "attachment_invalid", err.Error())
			return
		}
	}
	id := strings.TrimSpace(r.PathValue("id"))
	var completed Inspection
	err = s.transact(ctx, func(tx context.Context) error {
		return s.store.Update(func(state *State) error {
			current, ok := state.Inspections[id]
			if !ok || !allowed(predicate, current.Scope) {
				return errors.New("inspection not found")
			}
			if current.Status != "in_progress" {
				return errors.New("inspection is not in progress")
			}
			current.Status, current.Result, current.Finding = "completed", strings.TrimSpace(input.Result), strings.TrimSpace(input.Finding)
			current.AttachmentIDs, current.UpdatedAt = append([]string(nil), input.AttachmentIDs...), s.nowFn().UTC()
			state.Inspections[id], completed = current, current
			plan := state.Plans[current.PlanID]
			plan.NextRunAt = current.InspectedAt.AddDate(0, 0, plan.IntervalDays)
			plan.UpdatedAt = current.UpdatedAt
			state.Plans[plan.ID] = plan
			return s.audit(tx, "equipment_maintenance.inspection.complete", "inspection", id, pluginsdk.AuditRiskHigh, map[string]any{"attachments": len(input.AttachmentIDs), "result": input.Result})
		})
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "inspection_complete_failed", err.Error())
		return
	}
	writeOK(w, map[string]any{"item": completed})
}

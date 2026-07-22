package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type assetInput struct {
	TenantID          string     `json:"tenantId"`
	OrganizationID    string     `json:"organizationId"`
	Code              string     `json:"code"`
	Name              string     `json:"name"`
	Category          string     `json:"category"`
	Model             string     `json:"model"`
	SerialNumber      string     `json:"serialNumber"`
	Location          string     `json:"location"`
	NextMaintenanceAt *time.Time `json:"nextMaintenanceAt"`
}

func (s *Server) listAssets(w http.ResponseWriter, r *http.Request) {
	_, predicate, err := s.scope(r, "equipment_maintenance.asset.read")
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
	status := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	items := make([]Asset, 0)
	err = s.store.View(func(state State) error {
		for _, item := range state.Assets {
			if !allowed(predicate, item.Scope) || status != "" && item.Status != status {
				continue
			}
			if keyword != "" && !strings.Contains(strings.ToLower(item.Code+" "+item.Name+" "+item.Location), keyword) {
				continue
			}
			items = append(items, item)
		}
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_failed", err.Error())
		return
	}
	sortByUpdated(items, func(item Asset) time.Time { return item.UpdatedAt })
	writeOK(w, map[string]any{"items": page(items, offset, limit), "total": len(items), "offset": offset, "limit": limit})
}

func (s *Server) createAsset(w http.ResponseWriter, r *http.Request) {
	ctx, predicate, err := s.scope(r, "equipment_maintenance.asset.create")
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	var input assetInput
	if !decodeJSON(w, r, &input) {
		return
	}
	scope, err := trustedWriteScope(predicate, Scope{TenantID: input.TenantID, OrganizationID: input.OrganizationID})
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	input.Code, err = required(input.Code, "code")
	if err == nil {
		input.Name, err = required(input.Name, "name")
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_asset", err.Error())
		return
	}
	now := s.nowFn().UTC()
	var created Asset
	err = s.transact(ctx, func(tx context.Context) error {
		return s.store.Update(func(state *State) error {
			for _, current := range state.Assets {
				if current.TenantID == scope.TenantID && strings.EqualFold(current.Code, input.Code) {
					return errors.New("asset code already exists")
				}
			}
			created = Asset{ID: nextID(state, "asset"), Scope: scope, Code: input.Code, Name: input.Name, Category: strings.TrimSpace(input.Category), Model: strings.TrimSpace(input.Model), SerialNumber: strings.TrimSpace(input.SerialNumber), Location: strings.TrimSpace(input.Location), Status: "active", NextMaintenanceAt: input.NextMaintenanceAt, CreatedAt: now, UpdatedAt: now}
			state.Assets[created.ID] = created
			return s.audit(tx, "equipment_maintenance.asset.create", "asset", created.ID, pluginsdk.AuditRiskMedium, map[string]any{"code": created.Code})
		})
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "asset_create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, envelope(map[string]any{"item": created}))
}

func (s *Server) getAsset(w http.ResponseWriter, r *http.Request) {
	_, predicate, err := s.scope(r, "equipment_maintenance.asset.read")
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	var item Asset
	found := false
	err = s.store.View(func(state State) error {
		item, found = state.Assets[strings.TrimSpace(r.PathValue("id"))]
		found = found && allowed(predicate, item.Scope)
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_failed", err.Error())
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "asset_not_found", "asset not found")
		return
	}
	writeOK(w, map[string]any{"item": item})
}

func (s *Server) updateAsset(w http.ResponseWriter, r *http.Request) {
	ctx, predicate, err := s.scope(r, "equipment_maintenance.asset.update")
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	var input assetInput
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Code, err = required(input.Code, "code")
	if err == nil {
		input.Name, err = required(input.Name, "name")
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_asset", err.Error())
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	var updated Asset
	err = s.transact(ctx, func(tx context.Context) error {
		return s.store.Update(func(state *State) error {
			current, ok := state.Assets[id]
			if !ok || !allowed(predicate, current.Scope) {
				return errors.New("asset not found")
			}
			if current.Status == "retired" {
				return errors.New("retired asset cannot be updated")
			}
			for otherID, other := range state.Assets {
				if otherID != id && other.TenantID == current.TenantID && strings.EqualFold(other.Code, input.Code) {
					return errors.New("asset code already exists")
				}
			}
			current.Code, current.Name, current.Category = input.Code, input.Name, strings.TrimSpace(input.Category)
			current.Model, current.SerialNumber, current.Location = strings.TrimSpace(input.Model), strings.TrimSpace(input.SerialNumber), strings.TrimSpace(input.Location)
			current.NextMaintenanceAt, current.UpdatedAt = input.NextMaintenanceAt, s.nowFn().UTC()
			state.Assets[id], updated = current, current
			return s.audit(tx, "equipment_maintenance.asset.update", "asset", id, pluginsdk.AuditRiskMedium, nil)
		})
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "asset_update_failed", err.Error())
		return
	}
	writeOK(w, map[string]any{"item": updated})
}

func (s *Server) retireAsset(w http.ResponseWriter, r *http.Request) {
	ctx, predicate, err := s.scope(r, "equipment_maintenance.asset.retire")
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	var retired Asset
	err = s.transact(ctx, func(tx context.Context) error {
		return s.store.Update(func(state *State) error {
			current, ok := state.Assets[id]
			if !ok || !allowed(predicate, current.Scope) {
				return errors.New("asset not found")
			}
			for _, order := range state.WorkOrders {
				if order.AssetID == id && order.Status != "closed" {
					return errors.New("asset has an open work order")
				}
			}
			current.Status, current.UpdatedAt = "retired", s.nowFn().UTC()
			state.Assets[id], retired = current, current
			return s.audit(tx, "equipment_maintenance.asset.retire", "asset", id, pluginsdk.AuditRiskHigh, nil)
		})
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "asset_retire_failed", err.Error())
		return
	}
	writeOK(w, map[string]any{"item": retired})
}

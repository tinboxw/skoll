package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const approvalDefinitionID = "equipment-maintenance-cost-approval"

type workOrderInput struct {
	TenantID       string  `json:"tenantId"`
	OrganizationID string  `json:"organizationId"`
	AssetID        string  `json:"assetId"`
	Number         string  `json:"number"`
	Title          string  `json:"title"`
	Priority       string  `json:"priority"`
	EstimatedCost  float64 `json:"estimatedCost"`
}

type dispatchInput struct {
	AssigneeID string `json:"assigneeId"`
}

func (s *Server) listWorkOrders(w http.ResponseWriter, r *http.Request) {
	_, predicate, err := s.scope(r, "equipment_maintenance.work_order.read")
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
	assetID := strings.TrimSpace(r.URL.Query().Get("assetId"))
	keyword := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("keyword")))
	items := make([]WorkOrder, 0)
	err = s.store.View(func(state State) error {
		for _, item := range state.WorkOrders {
			if !allowed(predicate, item.Scope) || status != "" && item.Status != status || assetID != "" && item.AssetID != assetID {
				continue
			}
			if keyword != "" && !strings.Contains(strings.ToLower(item.Number+" "+item.Title), keyword) {
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
	sortByUpdated(items, func(item WorkOrder) time.Time { return item.UpdatedAt })
	writeOK(w, map[string]any{"items": page(items, offset, limit), "total": len(items), "offset": offset, "limit": limit})
}

func (s *Server) createWorkOrder(w http.ResponseWriter, r *http.Request) {
	ctx, predicate, err := s.scope(r, "equipment_maintenance.work_order.create")
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	var input workOrderInput
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
		input.Number, err = required(input.Number, "number")
	}
	if err == nil {
		input.Title, err = required(input.Title, "title")
	}
	if err != nil || input.EstimatedCost < 0 {
		writeError(w, http.StatusBadRequest, "invalid_work_order", "assetId, number, title, and non-negative estimatedCost are required")
		return
	}
	if input.Priority == "" {
		input.Priority = "normal"
	}
	now := s.nowFn().UTC()
	var created WorkOrder
	err = s.transact(ctx, func(tx context.Context) error {
		return s.store.Update(func(state *State) error {
			asset, ok := state.Assets[input.AssetID]
			if !ok || !allowed(predicate, asset.Scope) || asset.Status == "retired" {
				return errors.New("active asset not found")
			}
			for _, current := range state.WorkOrders {
				if current.TenantID == scope.TenantID && strings.EqualFold(current.Number, input.Number) {
					return errors.New("work order number already exists")
				}
			}
			created = WorkOrder{ID: nextID(state, "work"), Scope: scope, AssetID: input.AssetID, Number: input.Number, Title: input.Title, Priority: strings.ToLower(strings.TrimSpace(input.Priority)), Status: "draft", EstimatedCost: input.EstimatedCost, CreatedAt: now, UpdatedAt: now}
			state.WorkOrders[created.ID] = created
			return s.audit(tx, "equipment_maintenance.work_order.create", "work_order", created.ID, pluginsdk.AuditRiskMedium, map[string]any{"assetId": created.AssetID})
		})
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "work_order_create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, envelope(map[string]any{"item": created}))
}

func (s *Server) getWorkOrder(w http.ResponseWriter, r *http.Request) {
	_, predicate, err := s.scope(r, "equipment_maintenance.work_order.read")
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	var item WorkOrder
	found := false
	_ = s.store.View(func(state State) error {
		item, found = state.WorkOrders[strings.TrimSpace(r.PathValue("id"))]
		found = found && allowed(predicate, item.Scope)
		return nil
	})
	if !found {
		writeError(w, http.StatusNotFound, "work_order_not_found", "work order not found")
		return
	}
	writeOK(w, map[string]any{"item": item})
}

func (s *Server) updateWorkOrder(w http.ResponseWriter, r *http.Request) {
	ctx, predicate, err := s.scope(r, "equipment_maintenance.work_order.update")
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	var input workOrderInput
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Title, err = required(input.Title, "title")
	if err != nil || input.EstimatedCost < 0 {
		writeError(w, http.StatusBadRequest, "invalid_work_order", "title and non-negative estimatedCost are required")
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	var updated WorkOrder
	err = s.transact(ctx, func(tx context.Context) error {
		return s.store.Update(func(state *State) error {
			current, ok := state.WorkOrders[id]
			if !ok || !allowed(predicate, current.Scope) {
				return errors.New("work order not found")
			}
			if current.Status != "draft" {
				return errors.New("only draft work orders can be updated")
			}
			current.Title, current.Priority, current.EstimatedCost = input.Title, strings.ToLower(strings.TrimSpace(input.Priority)), input.EstimatedCost
			current.UpdatedAt = s.nowFn().UTC()
			state.WorkOrders[id], updated = current, current
			return s.audit(tx, "equipment_maintenance.work_order.update", "work_order", id, pluginsdk.AuditRiskMedium, nil)
		})
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "work_order_update_failed", err.Error())
		return
	}
	writeOK(w, map[string]any{"item": updated})
}

func (s *Server) dispatchWorkOrder(w http.ResponseWriter, r *http.Request) {
	var input dispatchInput
	if !decodeJSON(w, r, &input) {
		return
	}
	input.AssigneeID = strings.TrimSpace(input.AssigneeID)
	if input.AssigneeID == "" {
		writeError(w, http.StatusBadRequest, "invalid_assignee", "assigneeId is required")
		return
	}
	s.transitionWorkOrder(w, r, "equipment_maintenance.work_order.dispatch", "equipment_maintenance.work_order.dispatch", "draft", "dispatched", pluginsdk.AuditRiskHigh, func(order *WorkOrder, _ time.Time) { order.AssigneeID = input.AssigneeID })
}

func (s *Server) startWorkOrder(w http.ResponseWriter, r *http.Request) {
	s.transitionWorkOrder(w, r, "equipment_maintenance.work_order.execute", "equipment_maintenance.work_order.start", "dispatched", "in_progress", pluginsdk.AuditRiskHigh, func(order *WorkOrder, now time.Time) { order.StartedAt = &now })
}

func (s *Server) completeWorkOrder(w http.ResponseWriter, r *http.Request) {
	s.transitionWorkOrder(w, r, "equipment_maintenance.work_order.execute", "equipment_maintenance.work_order.complete", "approved", "completed", pluginsdk.AuditRiskHigh, func(order *WorkOrder, now time.Time) { order.CompletedAt = &now })
}

func (s *Server) closeWorkOrder(w http.ResponseWriter, r *http.Request) {
	s.transitionWorkOrder(w, r, "equipment_maintenance.work_order.close", "equipment_maintenance.work_order.close", "completed", "closed", pluginsdk.AuditRiskHigh, func(order *WorkOrder, now time.Time) { order.ClosedAt = &now })
}

func (s *Server) transitionWorkOrder(w http.ResponseWriter, r *http.Request, permission, auditAction, from, to string, risk pluginsdk.AuditRisk, mutate func(*WorkOrder, time.Time)) {
	ctx, predicate, err := s.scope(r, permission)
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	var updated WorkOrder
	err = s.transact(ctx, func(tx context.Context) error {
		return s.store.Update(func(state *State) error {
			current, ok := state.WorkOrders[id]
			if !ok || !allowed(predicate, current.Scope) {
				return errors.New("work order not found")
			}
			if current.Status != from {
				return fmt.Errorf("work order must be %s", from)
			}
			now := s.nowFn().UTC()
			current.Status, current.UpdatedAt = to, now
			mutate(&current, now)
			state.WorkOrders[id], updated = current, current
			return s.audit(tx, auditAction, "work_order", id, risk, map[string]any{"from": from, "to": to})
		})
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "work_order_transition_failed", err.Error())
		return
	}
	writeOK(w, map[string]any{"item": updated})
}

func (s *Server) submitWorkOrder(w http.ResponseWriter, r *http.Request) {
	ctx, predicate, err := s.scope(r, "equipment_maintenance.work_order.submit")
	if err != nil {
		writeError(w, http.StatusForbidden, "scope_denied", err.Error())
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	var current WorkOrder
	found := false
	_ = s.store.View(func(state State) error {
		current, found = state.WorkOrders[id]
		found = found && allowed(predicate, current.Scope)
		return nil
	})
	if !found {
		writeError(w, http.StatusNotFound, "work_order_not_found", "work order not found")
		return
	}
	if current.Status != "in_progress" {
		writeError(w, http.StatusBadRequest, "invalid_work_order_state", "work order must be in_progress")
		return
	}
	limit, err := s.approvalLimit(ctx)
	if err != nil {
		writeError(w, http.StatusBadGateway, "config_failed", err.Error())
		return
	}
	if current.EstimatedCost < limit {
		s.transitionWorkOrder(w, r, "equipment_maintenance.work_order.submit", "equipment_maintenance.work_order.submit", "in_progress", "approved", pluginsdk.AuditRiskHigh, func(*WorkOrder, time.Time) {})
		return
	}
	if err := s.ensureApprovalDefinition(ctx, predicate.SubjectID()); err != nil {
		writeError(w, http.StatusBadGateway, "workflow_definition_failed", err.Error())
		return
	}
	var instance pluginsdk.WorkflowInstance
	err = s.transact(ctx, func(tx context.Context) error {
		created, startErr := s.host.Workflows.Start(tx, pluginsdk.WorkflowStartInput{
			ID: "equipment-approval-" + id, DefinitionID: approvalDefinitionID, BusinessType: "equipment.work_order", BusinessID: id, Title: current.Title,
		})
		if startErr != nil {
			return startErr
		}
		instance = created
		return s.store.Update(func(state *State) error {
			order := state.WorkOrders[id]
			if order.Status != "in_progress" {
				return errors.New("work order changed during submission")
			}
			order.Status, order.WorkflowInstanceID, order.UpdatedAt = "awaiting_approval", instance.ID, s.nowFn().UTC()
			state.WorkOrders[id] = order
			return s.audit(tx, "equipment_maintenance.work_order.submit", "work_order", id, pluginsdk.AuditRiskHigh, map[string]any{"workflowInstanceId": instance.ID, "estimatedCost": order.EstimatedCost})
		})
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "workflow_start_failed", err.Error())
		return
	}
	_ = s.store.View(func(state State) error { current = state.WorkOrders[id]; return nil })
	writeOK(w, map[string]any{"item": current, "workflow": instance})
}

func (s *Server) approvalLimit(ctx context.Context) (float64, error) {
	values, err := s.host.Config.Get(ctx)
	if err != nil {
		return 0, err
	}
	value, ok := values["equipment_maintenance.approval_cost_limit"]
	if !ok {
		return 1000, nil
	}
	switch typed := value.(type) {
	case float64:
		return typed, nil
	case int:
		return float64(typed), nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, errors.New("approval cost limit is invalid")
		}
		return parsed, nil
	default:
		return 0, errors.New("approval cost limit is invalid")
	}
}

func (s *Server) ensureApprovalDefinition(ctx context.Context, assigneeID string) error {
	if definition, err := s.host.Workflows.GetDefinition(ctx, approvalDefinitionID); err == nil && definition.Status == pluginsdk.WorkflowDefinitionPublished {
		return nil
	}
	definition, err := s.host.Workflows.CreateDefinition(ctx, pluginsdk.WorkflowDefinitionInput{
		ID: approvalDefinitionID, Key: "work_order_cost_approval", Name: "Work order cost approval", Version: 1,
		Nodes: []pluginsdk.WorkflowNode{
			{ID: "start", Key: "start", Name: "Start", Type: pluginsdk.WorkflowNodeStart},
			{ID: "approval", Key: "approval", Name: "Cost approval", Type: pluginsdk.WorkflowNodeApproval, AssigneeIDs: []string{assigneeID}},
			{ID: "end", Key: "end", Name: "End", Type: pluginsdk.WorkflowNodeEnd},
		},
		Transitions: []pluginsdk.WorkflowTransition{{From: "start", To: "approval"}, {From: "approval", To: "end"}},
	})
	if err != nil {
		existing, getErr := s.host.Workflows.GetDefinition(ctx, approvalDefinitionID)
		if getErr != nil || existing.Status != pluginsdk.WorkflowDefinitionPublished {
			return err
		}
		return nil
	}
	_, err = s.host.Workflows.PublishDefinition(ctx, definition.ID)
	return err
}

func (s *Server) handleEvent(w http.ResponseWriter, r *http.Request) {
	var delivery pluginsdk.EventDelivery
	if !decodeJSON(w, r, &delivery) {
		return
	}
	event := delivery.Envelope
	if delivery.Subscriber != pluginID || delivery.Handler != "onApprovalCompleted" ||
		event.Publisher != "skoll" || event.Name != "approval-completed" || event.SchemaVersion != 1 ||
		strings.TrimSpace(delivery.DeliveryID) == "" {
		writeError(w, http.StatusBadRequest, "invalid_event", "event contract does not match manifest")
		return
	}
	status := strings.ToLower(eventPayloadString(event.Payload["status"]))
	if status != "approved" && status != "rejected" {
		writeError(w, http.StatusBadRequest, "invalid_event", "approval status is invalid")
		return
	}
	businessID := eventPayloadString(event.Payload["businessId"])
	if businessID == "" {
		businessID = strings.TrimSpace(event.Subject.ID)
	}
	err := s.store.Update(func(state *State) error {
		if _, duplicate := state.EventDeliveries[delivery.DeliveryID]; duplicate {
			return nil
		}
		order, ok := state.WorkOrders[businessID]
		if !ok || order.Status != "awaiting_approval" {
			return errors.New("approval work order not found")
		}
		order.Status, order.UpdatedAt = status, s.nowFn().UTC()
		state.WorkOrders[businessID] = order
		state.EventDeliveries[delivery.DeliveryID] = DeliveryRecord{DeliveryID: delivery.DeliveryID, HandledAt: s.nowFn().UTC()}
		return nil
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "event_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func eventPayloadString(value pluginsdk.DataValue) string {
	switch value.Type {
	case pluginsdk.DataValueString, pluginsdk.DataValueInteger, pluginsdk.DataValueDecimal, pluginsdk.DataValueBoolean:
		return strings.TrimSpace(value.Value)
	case pluginsdk.DataValueJSON:
		var decoded any
		if json.Unmarshal([]byte(value.Value), &decoded) == nil {
			return strings.TrimSpace(fmt.Sprint(decoded))
		}
	}
	return ""
}

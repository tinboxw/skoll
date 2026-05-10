package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	usersvc "github.com/tinboxw/skoll/internal/service/user"
	"github.com/tinboxw/skoll/pkg/security"
)

type UserHandler struct {
	service usersvc.Service
}

type batchCreateUsersRequest struct {
	Items  []usersvc.CreateUserInput `json:"items"`
	Atomic bool                      `json:"atomic"`
}

func RegisterUserRoutes(mux *http.ServeMux, service usersvc.Service) {
	if service == nil {
		return
	}
	h := &UserHandler{service: service}
	mux.HandleFunc("POST /v1/users", h.create)
	mux.HandleFunc("POST /v1/users/batch", h.createBatch)
	mux.HandleFunc("GET /v1/users", h.list)
	mux.HandleFunc("GET /v1/users/{id}", h.get)
	mux.HandleFunc("PUT /v1/users/{id}", h.update)
	mux.HandleFunc("PATCH /v1/users/{id}/email", h.updateEmail)
	mux.HandleFunc("DELETE /v1/users/{id}", h.delete)
	mux.HandleFunc("POST /v1/users/{id}/disable", h.disable)
}

func (h *UserHandler) create(w http.ResponseWriter, r *http.Request) {
	var req usersvc.CreateUserInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	entity, err := h.service.Create(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, entity)
}

func (h *UserHandler) createBatch(w http.ResponseWriter, r *http.Request) {
	var req batchCreateUsersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(req.Items) == 0 {
		writeMessage(w, http.StatusBadRequest, "invalid_request", "items are required")
		return
	}

	results, err := h.service.CreateBatch(r.Context(), usersvc.BatchCreateInput{Items: req.Items, Atomic: req.Atomic})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	successCount := 0
	for _, item := range results {
		if item.Success {
			successCount++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"successCount": successCount,
		"failureCount": len(req.Items) - successCount,
		"results":      results,
	})
}

func (h *UserHandler) list(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.service.List(r.Context(), usersvc.ListInput{Offset: offset, Limit: limit})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *UserHandler) get(w http.ResponseWriter, r *http.Request) {
	entity, err := h.service.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if entity == nil {
		writeMessage(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	writeJSON(w, http.StatusOK, entity)
}

func (h *UserHandler) updateEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email   string `json:"email"`
		ActorID string `json:"actorId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	entity, err := h.service.UpdateEmail(r.Context(), usersvc.UpdateEmailInput{ID: r.PathValue("id"), Email: req.Email, ActorID: actorIDFromRequest(r.Context(), req.ActorID)})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, entity)
}

func (h *UserHandler) update(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string `json:"name"`
		Email  string `json:"email"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	entity, err := h.service.Update(r.Context(), usersvc.UpdateUserInput{
		ID:     r.PathValue("id"),
		Name:   req.Name,
		Email:  req.Email,
		Status: req.Status,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, entity)
}

func (h *UserHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeMessage(w, http.StatusOK, "ok", "deleted")
}

func (h *UserHandler) disable(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ActorID string `json:"actorId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if err := h.service.Disable(r.Context(), r.PathValue("id"), actorIDFromRequest(r.Context(), req.ActorID)); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeMessage(w, http.StatusOK, "ok", "disabled")
}

func actorIDFromRequest(ctx context.Context, raw string) string {
	if actorID := strings.TrimSpace(raw); actorID != "" {
		return actorID
	}
	claims, ok := security.JWTClaimsFromContext(ctx)
	if !ok {
		return ""
	}
	return strings.TrimSpace(claims.Subject)
}

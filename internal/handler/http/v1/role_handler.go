package v1

import (
	"encoding/json"
	"net/http"
	"strconv"

	rolesvc "github.com/tinboxw/skoll/internal/service/role"
)

type RoleHandler struct {
	service rolesvc.Service
}

func RegisterRoleRoutes(mux *http.ServeMux, service rolesvc.Service) {
	if service == nil {
		return
	}
	h := &RoleHandler{service: service}
	mux.HandleFunc("POST /v1/roles", h.create)
	mux.HandleFunc("GET /v1/roles", h.list)
	mux.HandleFunc("GET /v1/roles/{id}", h.get)
	mux.HandleFunc("PUT /v1/roles/{id}", h.update)
	mux.HandleFunc("DELETE /v1/roles/{id}", h.delete)
	mux.HandleFunc("POST /v1/roles/{id}/grant", h.grant)
	mux.HandleFunc("POST /v1/roles/{id}/revoke", h.revoke)
}

func (h *RoleHandler) create(w http.ResponseWriter, r *http.Request) {
	var req rolesvc.CreateRoleInput
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

func (h *RoleHandler) list(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.service.List(r.Context(), rolesvc.ListInput{Offset: offset, Limit: limit})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *RoleHandler) get(w http.ResponseWriter, r *http.Request) {
	entity, err := h.service.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if entity == nil {
		writeMessage(w, http.StatusNotFound, "not_found", "role not found")
		return
	}
	writeJSON(w, http.StatusOK, entity)
}

func (h *RoleHandler) update(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string   `json:"name"`
		Key         string   `json:"key"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	entity, err := h.service.Update(r.Context(), rolesvc.UpdateRoleInput{
		ID:          r.PathValue("id"),
		Name:        req.Name,
		Key:         req.Key,
		Description: req.Description,
		Permissions: req.Permissions,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, entity)
}

func (h *RoleHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeMessage(w, http.StatusOK, "ok", "deleted")
}

func (h *RoleHandler) grant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Permission string `json:"permission"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	entity, err := h.service.Grant(r.Context(), r.PathValue("id"), req.Permission)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, entity)
}

func (h *RoleHandler) revoke(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Permission string `json:"permission"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	entity, err := h.service.Revoke(r.Context(), r.PathValue("id"), req.Permission)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, entity)
}

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

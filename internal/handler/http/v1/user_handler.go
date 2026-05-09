package v1

import (
	"encoding/json"
	"net/http"
	"strconv"

	usersvc "github.com/tinboxw/skoll/internal/service/user"
)

type UserHandler struct {
	service usersvc.Service
}

func RegisterUserRoutes(mux *http.ServeMux, service usersvc.Service) {
	if service == nil {
		return
	}
	h := &UserHandler{service: service}
	mux.HandleFunc("POST /v1/users", h.create)
	mux.HandleFunc("GET /v1/users", h.list)
	mux.HandleFunc("GET /v1/users/{id}", h.get)
	mux.HandleFunc("PATCH /v1/users/{id}/email", h.updateEmail)
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
	entity, err := h.service.UpdateEmail(r.Context(), usersvc.UpdateEmailInput{ID: r.PathValue("id"), Email: req.Email, ActorID: req.ActorID})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, entity)
}

func (h *UserHandler) disable(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ActorID string `json:"actorId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if err := h.service.Disable(r.Context(), r.PathValue("id"), req.ActorID); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeMessage(w, http.StatusOK, "ok", "disabled")
}

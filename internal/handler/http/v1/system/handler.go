package system

import (
	"encoding/json"
	"net/http"
	"strconv"

	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	systemsvc "github.com/tinboxw/skoll/internal/service/system"
)

type SystemHandler struct {
	service systemsvc.Service
}

func RegisterSystemRoutes(mux *http.ServeMux, service systemsvc.Service) {
	if service == nil {
		return
	}
	h := &SystemHandler{service: service}
	mux.HandleFunc("GET /v1/system/settings", h.list)
	mux.HandleFunc("GET /v1/system/settings/{key}", h.getByKey)
	mux.HandleFunc("PUT /v1/system/settings/{key}", h.upsert)
}

func (h *SystemHandler) list(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.service.List(r.Context(), systemsvc.ListInput{Offset: offset, Limit: limit})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, items)
}

func (h *SystemHandler) getByKey(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetByKey(r.Context(), r.PathValue("key"))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	if item == nil {
		apiv1.WriteMessage(w, http.StatusNotFound, "not_found", "setting not found")
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, item)
}

func (h *SystemHandler) upsert(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Value     string `json:"value"`
		Encrypted bool   `json:"encrypted"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.Upsert(r.Context(), systemsvc.UpsertInput{
		Key:       r.PathValue("key"),
		Value:     req.Value,
		Encrypted: req.Encrypted,
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, item)
}

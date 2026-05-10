package v1

import (
	"net/http"
	"strconv"

	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
)

type AuditHandler struct {
	service auditsvc.Service
}

func RegisterAuditRoutes(mux *http.ServeMux, service auditsvc.Service) {
	if service == nil {
		return
	}
	h := &AuditHandler{service: service}
	mux.HandleFunc("GET /v1/audit/actors/{actorId}", h.listByActor)
}

func (h *AuditHandler) listByActor(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.service.ListByActor(r.Context(), r.PathValue("actorId"), limit)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

package apiregistry

import (
	"encoding/json"
	"net/http"

	"github.com/tinboxw/skoll/internal/module/audit"
)

type APIRegistryService interface {
	RegisterMany(entries []string)
	Exists(entry string) bool
	List() []string
}

// AuditService is declared to maintain interface parity with other subpackages; not used directly.
type AuditService interface {
	Append(actor, action, target string) audit.Record
}

type Handler struct {
	svc APIRegistryService
}

func NewHandler(svc APIRegistryService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(mux *http.ServeMux, wrapper func(http.Handler) http.Handler) {
	if mux == nil || h.svc == nil {
		return
	}

	// Register the list-APIs route; the route itself is self-describing so we skip re-registering it.
	hd := http.Handler(http.HandlerFunc(h.listAPIs))
	if wrapper != nil {
		hd = wrapper(hd)
	}
	mux.Handle("GET /admin/v1/apis", hd)
}

func (h *Handler) listAPIs(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, listAPIsResponse{Items: h.svc.List()})
}

type listAPIsResponse struct {
	Items []string `json:"items"`
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

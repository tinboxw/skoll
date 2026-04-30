package generator

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/tinboxw/skoll/internal/module/audit"
	"github.com/tinboxw/skoll/internal/module/modgenerator"
)

type GeneratorService interface {
	Generate(module string) (modgenerator.Result, error)
	GenerateWithSchema(module string, schema *modgenerator.FormSchema, templateVersion string) (modgenerator.Result, error)
}

// AuditService is declared to maintain interface parity with other subpackages; not used directly.
type AuditService interface {
	Append(actor, action, target string) audit.Record
}

type APIRegistry interface {
	RegisterMany(entries []string)
}

type Handler struct {
	svc GeneratorService
}

func NewHandler(svc GeneratorService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(mux *http.ServeMux, wrapper func(http.Handler) http.Handler, apis APIRegistry) {
	if mux == nil || h.svc == nil {
		return
	}

	handle := func(pattern string, next http.HandlerFunc) {
		if apis != nil {
			apis.RegisterMany([]string{pattern})
		}
		hd := http.Handler(next)
		if wrapper != nil {
			hd = wrapper(hd)
		}
		mux.Handle(pattern, hd)
	}

	handle("POST /admin/v1/generator/modules", h.generateModule)
}

func (h *Handler) generateModule(w http.ResponseWriter, r *http.Request) {
	var req generateModuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	result, err := h.svc.GenerateWithSchema(strings.TrimSpace(req.Module), req.FormSchema, strings.TrimSpace(req.TemplateVersion))
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, result)
}

type generateModuleRequest struct {
	Module          string                   `json:"module"`
	TemplateVersion string                   `json:"template_version"`
	FormSchema      *modgenerator.FormSchema `json:"form_schema"`
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

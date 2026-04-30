package menu

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/tinboxw/skoll/internal/module/audit"
	"github.com/tinboxw/skoll/internal/module/menu"
)

type MenuService interface {
	Create(title, path string, order int) menu.Item
	Get(id int64) (menu.Item, error)
	List() []menu.Item
}

type APIRegistry interface {
	RegisterMany(entries []string)
}

// AuditService is declared to maintain interface parity with other subpackages; not used directly.
type AuditService interface {
	Append(actor, action, target string) audit.Record
}

type Handler struct {
	svc MenuService
}

func NewHandler(svc MenuService) *Handler {
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

	handle("POST /admin/v1/menus", h.createMenu)
	handle("GET /admin/v1/menus", h.listMenus)
	handle("GET /admin/v1/menus/{id}", h.getMenu)
}

func (h *Handler) createMenu(w http.ResponseWriter, r *http.Request) {
	var req createMenuRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Path = strings.TrimSpace(req.Path)
	if req.Title == "" || req.Path == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "title and path are required"})
		return
	}
	respondJSON(w, http.StatusCreated, h.svc.Create(req.Title, req.Path, req.Order))
}

func (h *Handler) listMenus(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, h.svc.List())
}

func (h *Handler) getMenu(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathInt64(r, "id")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	out, err := h.svc.Get(id)
	if err != nil {
		if err == menu.ErrMenuNotFound {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	respondJSON(w, http.StatusOK, out)
}

type createMenuRequest struct {
	Title string `json:"title"`
	Path  string `json:"path"`
	Order int    `json:"order"`
}

func parsePathInt64(r *http.Request, key string) (int64, error) {
	raw := strings.TrimSpace(r.PathValue(key))
	if raw == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return id, nil
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

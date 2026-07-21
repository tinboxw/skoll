package pharmaoa

import (
	"context"
	"net/http"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	permissionsvc "github.com/tinboxw/skoll/internal/service/permission"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
)

const (
	PermissionDemoSeedRead  = "pharma_oa.seed.read"
	PermissionDemoSeedApply = "pharma_oa.seed.apply"
)

type DemoSeedHandler struct{ service pharmaoasvc.DemoSeedService }

func RegisterDemoSeedRoutes(mux *http.ServeMux, service pharmaoasvc.DemoSeedService) {
	if mux == nil || service == nil {
		return
	}
	h := &DemoSeedHandler{service: service}
	mux.HandleFunc("GET /v1/plugins/pharma_oa/api/demo-seed/status", h.status)
	mux.HandleFunc("POST /v1/plugins/pharma_oa/api/demo-seed/apply", h.apply)
}

func RegisterDemoSeedPermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	for _, item := range []permissionsvc.RegisterResourceInput{
		{Key: PermissionDemoSeedRead, Type: domainpermission.ResourceTypeAPI, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Read demo seed status", Risk: domainpermission.RiskLevelLow},
		{Key: PermissionDemoSeedApply, Type: domainpermission.ResourceTypeButton, Module: "pharma_oa", Source: "plugin.pharma_oa", Name: "Apply demo seed", Risk: domainpermission.RiskLevelMedium},
	} {
		if _, err := service.RegisterResource(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}

func (h *DemoSeedHandler) status(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Status(r.Context(), contractActorIDFromRequest(r, ""))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *DemoSeedHandler) apply(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Apply(r.Context(), contractActorIDFromRequest(r, ""))
	if err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

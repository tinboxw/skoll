package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	"github.com/tinboxw/skoll/pkg/security"
)

const (
	devReleaseOrderStatusPending  = "pending"
	devReleaseOrderStatusApproved = "approved"
	devReleaseOrderStatusRejected = "rejected"
)

type devCreateReleaseOrderRequest struct {
	PluginsRoot    string `json:"pluginsRoot"`
	PluginID       string `json:"pluginId"`
	ReleaseVersion string `json:"releaseVersion"`
	Changelog      string `json:"changelog"`
}

type devReviewReleaseOrderRequest struct {
	PluginsRoot string `json:"pluginsRoot"`
	PluginID    string `json:"pluginId"`
	Comment     string `json:"comment"`
}

type devReleaseOrderRecord struct {
	OrderID        string `json:"orderId"`
	PluginID       string `json:"pluginId"`
	ReleaseVersion string `json:"releaseVersion"`
	OrderStatus    string `json:"orderStatus"`
	Changelog      string `json:"changelog,omitempty"`
	CreatedBy      string `json:"createdBy"`
	CreatedAt      string `json:"createdAt"`
	ApprovedBy     string `json:"approvedBy,omitempty"`
	ApprovedAt     string `json:"approvedAt,omitempty"`
	RejectedBy     string `json:"rejectedBy,omitempty"`
	RejectedAt     string `json:"rejectedAt,omitempty"`
	ReviewComment  string `json:"reviewComment,omitempty"`
}

type devReleaseOrderEnvelope struct {
	Orders []devReleaseOrderRecord `json:"orders"`
}

type devCreateReleaseOrderResponse struct {
	Operation string                `json:"operation"`
	Status    string                `json:"status"`
	Order     devReleaseOrderRecord `json:"order"`
}

type devListReleaseOrderResponse struct {
	Operation   string                  `json:"operation"`
	Status      string                  `json:"status"`
	PluginsRoot string                  `json:"pluginsRoot"`
	PluginID    string                  `json:"pluginId,omitempty"`
	Orders      []devReleaseOrderRecord `json:"orders"`
}

type devReviewReleaseOrderResponse struct {
	Operation string                `json:"operation"`
	Status    string                `json:"status"`
	Order     devReleaseOrderRecord `json:"order"`
}

func (h *PluginHandler) devCreateReleaseOrder(w http.ResponseWriter, r *http.Request) {
	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "super_admin role required")
		return
	}

	var req devCreateReleaseOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	pluginsRoot, err := h.resolveDevPluginsRoot(req.PluginsRoot)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	pluginID := strings.TrimSpace(strings.ToLower(req.PluginID))
	if pluginID == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("pluginId is required"))
		return
	}

	pluginDir, _, err := h.resolveManifestPath(pluginsRoot, pluginID)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	releaseVersion := strings.TrimSpace(req.ReleaseVersion)
	if releaseVersion == "" {
		info, loadErr := h.loader.Load(pluginDir)
		if loadErr != nil {
			apiv1.WriteError(w, http.StatusBadRequest, loadErr)
			return
		}
		releaseVersion = strings.TrimSpace(info.Version)
	}
	if releaseVersion == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("releaseVersion is required"))
		return
	}

	orders, err := h.loadDevReleaseOrders(pluginsRoot, pluginID)
	if err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	actorID := actorIDFromJWT(r)
	now := time.Now().UTC().Format(time.RFC3339)
	order := devReleaseOrderRecord{
		OrderID:        newDevReleaseOrderID(),
		PluginID:       pluginID,
		ReleaseVersion: releaseVersion,
		OrderStatus:    devReleaseOrderStatusPending,
		Changelog:      strings.TrimSpace(req.Changelog),
		CreatedBy:      actorID,
		CreatedAt:      now,
	}
	orders = append(orders, order)
	if err := h.saveDevReleaseOrders(pluginsRoot, pluginID, orders); err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	h.appendAudit(r, "dev_release_order_create", "plugin_release_order", order.OrderID, map[string]any{
		"pluginsRoot":    pluginsRoot,
		"pluginId":       pluginID,
		"releaseVersion": releaseVersion,
		"orderStatus":    order.OrderStatus,
	})
	apiv1.WriteJSON(w, http.StatusCreated, devCreateReleaseOrderResponse{
		Operation: "release_order_create",
		Status:    "ok",
		Order:     order,
	})
}

func (h *PluginHandler) devListReleaseOrders(w http.ResponseWriter, r *http.Request) {
	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "super_admin role required")
		return
	}

	pluginsRoot, err := h.resolveDevPluginsRoot(r.URL.Query().Get("pluginsRoot"))
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	pluginID := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("pluginId")))

	orders := make([]devReleaseOrderRecord, 0, 16)
	if pluginID != "" {
		items, loadErr := h.loadDevReleaseOrders(pluginsRoot, pluginID)
		if loadErr != nil {
			apiv1.WriteError(w, http.StatusInternalServerError, loadErr)
			return
		}
		orders = append(orders, items...)
	} else {
		pluginDirs, dirErr := discoverPluginManifestDirs(pluginsRoot)
		if dirErr != nil {
			apiv1.WriteError(w, http.StatusInternalServerError, dirErr)
			return
		}
		for _, dir := range pluginDirs {
			id := strings.TrimSpace(filepath.Base(dir))
			if id == "" {
				continue
			}
			items, loadErr := h.loadDevReleaseOrders(pluginsRoot, id)
			if loadErr != nil {
				apiv1.WriteError(w, http.StatusInternalServerError, loadErr)
				return
			}
			orders = append(orders, items...)
		}
	}

	sort.SliceStable(orders, func(i, j int) bool {
		return orders[i].CreatedAt > orders[j].CreatedAt
	})

	apiv1.WriteJSON(w, http.StatusOK, devListReleaseOrderResponse{
		Operation:   "release_order_list",
		Status:      "ok",
		PluginsRoot: pluginsRoot,
		PluginID:    pluginID,
		Orders:      orders,
	})
}

func (h *PluginHandler) devApproveReleaseOrder(w http.ResponseWriter, r *http.Request) {
	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "super_admin role required")
		return
	}

	orderID := strings.TrimSpace(r.PathValue("orderId"))
	if orderID == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("orderId is required"))
		return
	}

	var req devReviewReleaseOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	pluginsRoot, err := h.resolveDevPluginsRoot(req.PluginsRoot)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	pluginID := strings.TrimSpace(strings.ToLower(req.PluginID))
	if pluginID == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("pluginId is required"))
		return
	}

	orders, err := h.loadDevReleaseOrders(pluginsRoot, pluginID)
	if err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	index := findDevReleaseOrderIndex(orders, orderID)
	if index < 0 {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("release order not found"))
		return
	}
	if orders[index].OrderStatus != devReleaseOrderStatusPending {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("only pending order can be approved"))
		return
	}

	actorID := actorIDFromJWT(r)
	now := time.Now().UTC().Format(time.RFC3339)
	orders[index].OrderStatus = devReleaseOrderStatusApproved
	orders[index].ApprovedBy = actorID
	orders[index].ApprovedAt = now
	orders[index].ReviewComment = strings.TrimSpace(req.Comment)
	if err := h.saveDevReleaseOrders(pluginsRoot, pluginID, orders); err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	h.appendAudit(r, "dev_release_order_approve", "plugin_release_order", orderID, map[string]any{
		"pluginsRoot": pluginsRoot,
		"pluginId":    pluginID,
		"comment":     orders[index].ReviewComment,
	})
	apiv1.WriteJSON(w, http.StatusOK, devReviewReleaseOrderResponse{
		Operation: "release_order_approve",
		Status:    "ok",
		Order:     orders[index],
	})
}

func (h *PluginHandler) devRejectReleaseOrder(w http.ResponseWriter, r *http.Request) {
	if !h.isSuperAdmin(r) {
		apiv1.WriteMessage(w, http.StatusForbidden, "forbidden", "super_admin role required")
		return
	}

	orderID := strings.TrimSpace(r.PathValue("orderId"))
	if orderID == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("orderId is required"))
		return
	}

	var req devReviewReleaseOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}

	pluginsRoot, err := h.resolveDevPluginsRoot(req.PluginsRoot)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	pluginID := strings.TrimSpace(strings.ToLower(req.PluginID))
	if pluginID == "" {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("pluginId is required"))
		return
	}

	orders, err := h.loadDevReleaseOrders(pluginsRoot, pluginID)
	if err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	index := findDevReleaseOrderIndex(orders, orderID)
	if index < 0 {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("release order not found"))
		return
	}
	if orders[index].OrderStatus != devReleaseOrderStatusPending {
		apiv1.WriteError(w, http.StatusBadRequest, errors.New("only pending order can be rejected"))
		return
	}

	actorID := actorIDFromJWT(r)
	now := time.Now().UTC().Format(time.RFC3339)
	orders[index].OrderStatus = devReleaseOrderStatusRejected
	orders[index].RejectedBy = actorID
	orders[index].RejectedAt = now
	orders[index].ReviewComment = strings.TrimSpace(req.Comment)
	if err := h.saveDevReleaseOrders(pluginsRoot, pluginID, orders); err != nil {
		apiv1.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	h.appendAudit(r, "dev_release_order_reject", "plugin_release_order", orderID, map[string]any{
		"pluginsRoot": pluginsRoot,
		"pluginId":    pluginID,
		"comment":     orders[index].ReviewComment,
	})
	apiv1.WriteJSON(w, http.StatusOK, devReviewReleaseOrderResponse{
		Operation: "release_order_reject",
		Status:    "ok",
		Order:     orders[index],
	})
}

func (h *PluginHandler) loadDevReleaseOrders(pluginsRoot, pluginID string) ([]devReleaseOrderRecord, error) {
	path := devReleaseOrdersPath(pluginsRoot, pluginID)
	h.devMu.Lock()
	defer h.devMu.Unlock()

	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []devReleaseOrderRecord{}, nil
		}
		return nil, err
	}

	envelope := devReleaseOrderEnvelope{}
	if len(bytesTrim(raw)) == 0 {
		return []devReleaseOrderRecord{}, nil
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	return envelope.Orders, nil
}

func (h *PluginHandler) saveDevReleaseOrders(pluginsRoot, pluginID string, orders []devReleaseOrderRecord) error {
	path := devReleaseOrdersPath(pluginsRoot, pluginID)
	h.devMu.Lock()
	defer h.devMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(devReleaseOrderEnvelope{Orders: orders}, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func devReleaseOrdersPath(pluginsRoot, pluginID string) string {
	return filepath.Join(pluginsRoot, pluginID, ".devportal", fmt.Sprintf("%s.release.orders.json", pluginID))
}

func findDevReleaseOrderIndex(orders []devReleaseOrderRecord, orderID string) int {
	for idx := range orders {
		if orders[idx].OrderID == orderID {
			return idx
		}
	}
	return -1
}

func actorIDFromJWT(r *http.Request) string {
	if r == nil {
		return "system"
	}
	claims, ok := security.JWTClaimsFromContext(r.Context())
	if !ok {
		return "system"
	}
	if v := strings.TrimSpace(claims.Subject); v != "" {
		return v
	}
	return "system"
}

func newDevReleaseOrderID() string {
	unixNano := time.Now().UTC().UnixNano()
	return "ro-" + strconv.FormatInt(unixNano, 36)
}

func bytesTrim(raw []byte) string {
	return strings.TrimSpace(string(raw))
}

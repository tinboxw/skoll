package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const warehouseTable = "warehouses"

var warehouseFields = []string{
	"id", "code", "name", "address", "contact_name", "contact_phone", "status", "disable_reason",
	"tenant_id", "organization_id", "owner_id", "created_at", "updated_at",
}

type warehouse struct {
	ID            string `json:"id"`
	Code          string `json:"code"`
	Name          string `json:"name"`
	Address       string `json:"address"`
	ContactName   string `json:"contactName"`
	ContactPhone  string `json:"contactPhone"`
	Status        string `json:"status"`
	DisableReason string `json:"disableReason,omitempty"`
	Version       int64  `json:"version"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
	scope         employeeScope
}

type warehouseWriteRequest struct {
	TenantID       string `json:"tenantId"`
	OrganizationID string `json:"organizationId"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	Address        string `json:"address"`
	ContactName    string `json:"contactName"`
	ContactPhone   string `json:"contactPhone"`
	Version        int64  `json:"version,omitempty"`
}

type warehouseStatusRequest struct {
	TenantID       string `json:"tenantId"`
	OrganizationID string `json:"organizationId"`
	Reason         string `json:"reason,omitempty"`
	Version        int64  `json:"version"`
}

func (s *server) listWarehouses(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	offset, limit, err := employeePagination(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	filter, err := warehouseFilter(r.URL.Query().Get("keyword"), r.URL.Query().Get("status"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	items, err := s.queryWarehouses(ctx, warehousePermission("read"), filter)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	total := len(items)
	if offset > total {
		offset = total
	}
	end := min(total, offset+limit)
	writeOK(w, map[string]any{"items": items[offset:end], "total": total, "offset": offset, "limit": limit})
}

func (s *server) createWarehouse(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	key, err := mutationKey(r, "warehouse-create")
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var input warehouseWriteRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	if err = validateWarehouseWrite(&input, false); err != nil {
		writeServiceError(w, err)
		return
	}
	scope, err := s.exactWriteScope(ctx, warehousePermission("create"), input.TenantID, input.OrganizationID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	item := warehouse{
		ID: stableWarehouseID(key), Code: input.Code, Name: input.Name, Address: input.Address,
		ContactName: input.ContactName, ContactPhone: input.ContactPhone, Status: "active", scope: scope,
	}
	if err = s.ensureWarehouseCodeUnique(ctx, warehousePermission("create"), scope, item.ID, item.Code); err != nil {
		writeServiceError(w, err)
		return
	}
	var created warehouse
	err = s.transaction(ctx, func(tx context.Context) error {
		result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{
			Table: warehouseTable, Operation: pluginsdk.DataMutationInsert, Scope: warehouseIntent("create", scope),
			Key: map[string]pluginsdk.DataValue{"id": stringValue(item.ID)}, Values: warehouseValues(item),
			Returning: warehouseFields, IdempotencyKey: key,
		})
		if mutationErr != nil {
			return mutationErr
		}
		created, mutationErr = warehouseFromMutation(result)
		if mutationErr != nil {
			return mutationErr
		}
		return s.audit(tx, "pharma_oa.warehouse.create", created.ID, pluginsdk.AuditRiskMedium, warehouseAuditDetail(created))
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeCreated(w, map[string]any{"item": created})
}

func (s *server) updateWarehouse(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	key, err := mutationKey(r, "warehouse-update")
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var input warehouseWriteRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	if err = validateWarehouseWrite(&input, true); err != nil {
		writeServiceError(w, err)
		return
	}
	scope, err := s.exactWriteScope(ctx, warehousePermission("update"), input.TenantID, input.OrganizationID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	current, err := s.getWarehouse(ctx, warehousePermission("update"), scope, r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if input.Version != current.Version && !(current.Version > 1 && input.Version == current.Version-1 && warehouseMatchesWrite(current, input)) {
		writeServiceError(w, newHTTPError(http.StatusConflict, "stale_warehouse", "warehouse version is stale"))
		return
	}
	if err = s.ensureWarehouseCodeUnique(ctx, warehousePermission("update"), scope, current.ID, input.Code); err != nil {
		writeServiceError(w, err)
		return
	}
	current.Code, current.Name, current.Address = input.Code, input.Name, input.Address
	current.ContactName, current.ContactPhone = input.ContactName, input.ContactPhone
	updated, err := s.mutateWarehouse(ctx, "update", key, current, input.Version, pluginsdk.AuditRiskMedium)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, map[string]any{"item": updated})
}

func (s *server) changeWarehouseStatus(enabled bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, err := s.requestContext(r)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		action, targetStatus := "disable", "disabled"
		if enabled {
			action, targetStatus = "enable", "active"
		}
		key, err := mutationKey(r, "warehouse-"+action)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		var input warehouseStatusRequest
		if !decodeJSON(w, r, &input) {
			return
		}
		input.TenantID = strings.TrimSpace(input.TenantID)
		input.OrganizationID = strings.TrimSpace(input.OrganizationID)
		input.Reason = strings.TrimSpace(input.Reason)
		if input.TenantID == "" || input.OrganizationID == "" || input.Version < 1 || (!enabled && input.Reason == "") ||
			len(input.Reason) > 500 || (enabled && input.Reason != "") {
			writeServiceError(w, newHTTPError(http.StatusBadRequest, "invalid_warehouse_status", "scope, current version, and a bounded disable reason are required"))
			return
		}
		scope, err := s.exactWriteScope(ctx, warehousePermission(action), input.TenantID, input.OrganizationID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		current, err := s.getWarehouse(ctx, warehousePermission(action), scope, r.PathValue("id"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		targetReason := input.Reason
		if enabled {
			targetReason = ""
		}
		replayCandidate := current.Version > 1 && input.Version == current.Version-1 &&
			current.Status == targetStatus && current.DisableReason == targetReason
		if input.Version != current.Version && !replayCandidate {
			writeServiceError(w, newHTTPError(http.StatusConflict, "stale_warehouse", "warehouse version is stale"))
			return
		}
		if input.Version == current.Version && current.Status == targetStatus {
			writeServiceError(w, newHTTPError(http.StatusConflict, "warehouse_status_unchanged", "warehouse is already in the requested status"))
			return
		}
		current.Status, current.DisableReason = targetStatus, targetReason
		updated, err := s.mutateWarehouse(ctx, action, key, current, input.Version, pluginsdk.AuditRiskHigh)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeOK(w, map[string]any{"item": updated})
	}
}

func (s *server) mutateWarehouse(ctx context.Context, action, key string, item warehouse, version int64, risk pluginsdk.AuditRisk) (warehouse, error) {
	var updated warehouse
	err := s.transaction(ctx, func(tx context.Context) error {
		result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{
			Table: warehouseTable, Operation: pluginsdk.DataMutationUpdate, Scope: warehouseIntent(action, item.scope),
			Key: map[string]pluginsdk.DataValue{"id": stringValue(item.ID)}, Values: warehouseValues(item),
			Returning: warehouseFields, IdempotencyKey: key, ExpectedVersion: &version,
		})
		if mutationErr != nil {
			return mutationErr
		}
		updated, mutationErr = warehouseFromMutation(result)
		if mutationErr != nil {
			return mutationErr
		}
		return s.audit(tx, "pharma_oa.warehouse."+action, updated.ID, risk, warehouseAuditDetail(updated))
	})
	return updated, err
}

func (s *server) ensureWarehouseCodeUnique(ctx context.Context, permission pluginsdk.Permission, scope employeeScope, excludeID, code string) error {
	codeValue := stringValue(code)
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
		Table: warehouseTable, Fields: warehouseFields, Scope: pluginsdk.DataScopeIntent{Permission: permission, Filter: scopeFilter(scope)},
		Filter: &pluginsdk.DataFilter{Field: "code", Operator: pluginsdk.DataOperatorEqual, Value: &codeValue},
		Sort:   []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 2},
	})
	if err != nil {
		return err
	}
	for _, record := range page.Records {
		item, parseErr := warehouseFromRecord(record)
		if parseErr != nil {
			return parseErr
		}
		if item.ID != excludeID {
			return newHTTPError(http.StatusConflict, "duplicate_warehouse_code", "warehouse code already exists in this organization")
		}
	}
	return nil
}

func (s *server) getWarehouse(ctx context.Context, permission pluginsdk.Permission, scope employeeScope, id string) (warehouse, error) {
	id = strings.TrimSpace(id)
	if id == "" || len(id) > 128 {
		return warehouse{}, newHTTPError(http.StatusBadRequest, "invalid_warehouse", "warehouse id is invalid")
	}
	idValue := stringValue(id)
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
		Table: warehouseTable, Fields: warehouseFields, Scope: pluginsdk.DataScopeIntent{Permission: permission, Filter: scopeFilter(scope)},
		Filter: &pluginsdk.DataFilter{Field: "id", Operator: pluginsdk.DataOperatorEqual, Value: &idValue},
		Sort:   []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 1},
	})
	if err != nil {
		return warehouse{}, err
	}
	if len(page.Records) != 1 {
		return warehouse{}, newHTTPError(http.StatusNotFound, "warehouse_not_found", "warehouse was not found")
	}
	return warehouseFromRecord(page.Records[0])
}

func (s *server) queryWarehouses(ctx context.Context, permission pluginsdk.Permission, filter *pluginsdk.DataFilter) ([]warehouse, error) {
	items, cursor := make([]warehouse, 0), ""
	for pageNumber := 0; pageNumber < 25; pageNumber++ {
		page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
			Table: warehouseTable, Fields: warehouseFields, Scope: pluginsdk.DataScopeIntent{Permission: permission},
			Filter: filter, Sort: []pluginsdk.DataSort{{Field: "updated_at", Direction: pluginsdk.DataSortDescending}},
			Page: pluginsdk.DataPageRequest{Cursor: cursor, Limit: 200},
		})
		if err != nil {
			return nil, err
		}
		for _, record := range page.Records {
			item, parseErr := warehouseFromRecord(record)
			if parseErr != nil {
				return nil, parseErr
			}
			items = append(items, item)
		}
		if !page.HasMore {
			return items, nil
		}
		cursor = page.NextCursor
	}
	return nil, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorLimitExceeded, warehouseTable, "warehouse query exceeds 5000 records", false)
}

func warehouseFilter(keyword, status string) (*pluginsdk.DataFilter, error) {
	filters := make([]pluginsdk.DataFilter, 0, 2)
	keyword = strings.TrimSpace(keyword)
	if len(keyword) > 100 {
		return nil, newHTTPError(http.StatusBadRequest, "invalid_request", "keyword is too long")
	}
	if keyword != "" {
		value := stringValue(keyword)
		filters = append(filters, pluginsdk.DataFilter{Any: []pluginsdk.DataFilter{
			{Field: "code", Operator: pluginsdk.DataOperatorContains, Value: &value},
			{Field: "name", Operator: pluginsdk.DataOperatorContains, Value: &value},
			{Field: "address", Operator: pluginsdk.DataOperatorContains, Value: &value},
			{Field: "contact_name", Operator: pluginsdk.DataOperatorContains, Value: &value},
		}})
	}
	status = strings.TrimSpace(status)
	if status != "" {
		if status != "active" && status != "disabled" {
			return nil, newHTTPError(http.StatusBadRequest, "invalid_warehouse_status", "warehouse status is invalid")
		}
		value := stringValue(status)
		filters = append(filters, pluginsdk.DataFilter{Field: "status", Operator: pluginsdk.DataOperatorEqual, Value: &value})
	}
	if len(filters) == 0 {
		return nil, nil
	}
	if len(filters) == 1 {
		return &filters[0], nil
	}
	return &pluginsdk.DataFilter{All: filters}, nil
}

func validateWarehouseWrite(input *warehouseWriteRequest, update bool) error {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.OrganizationID = strings.TrimSpace(input.OrganizationID)
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	input.Address = strings.TrimSpace(input.Address)
	input.ContactName = strings.TrimSpace(input.ContactName)
	input.ContactPhone = strings.TrimSpace(input.ContactPhone)
	if input.TenantID == "" || input.OrganizationID == "" || input.Code == "" || len(input.Code) > 64 || !validWarehouseCode(input.Code) ||
		input.Name == "" || len(input.Name) > 160 || input.Address == "" || len(input.Address) > 500 ||
		input.ContactName == "" || len(input.ContactName) > 120 || input.ContactPhone == "" || len(input.ContactPhone) > 64 ||
		(update && input.Version < 1) {
		return newHTTPError(http.StatusBadRequest, "invalid_warehouse", "scope, code, name, address, contact, and current version are required")
	}
	return nil
}

func validWarehouseCode(value string) bool {
	for _, character := range value {
		if (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || strings.ContainsRune("._-", character) {
			continue
		}
		return false
	}
	return true
}

func stableWarehouseID(idempotencyKey string) string {
	digest := sha256.Sum256([]byte(pluginID + "\x00warehouse\x00" + idempotencyKey))
	return "warehouse-" + hex.EncodeToString(digest[:12])
}

func warehousePermission(action string) pluginsdk.Permission {
	return pluginsdk.Permission{Resource: "pharma_oa.warehouse", Action: action}
}

func warehouseIntent(action string, scope employeeScope) pluginsdk.DataScopeIntent {
	return pluginsdk.DataScopeIntent{Permission: warehousePermission(action), Filter: scopeFilter(scope)}
}

func warehouseValues(item warehouse) map[string]pluginsdk.DataValue {
	return map[string]pluginsdk.DataValue{
		"code": stringValue(item.Code), "name": stringValue(item.Name), "address": stringValue(item.Address),
		"contact_name": stringValue(item.ContactName), "contact_phone": stringValue(item.ContactPhone),
		"status": stringValue(item.Status), "disable_reason": nullableStringValue(item.DisableReason),
	}
}

func warehouseFromMutation(result pluginsdk.DataMutationResult) (warehouse, error) {
	if result.Record == nil || result.RowsAffected != 1 {
		return warehouse{}, fmt.Errorf("warehouse mutation returned no record")
	}
	return warehouseFromRecord(*result.Record)
}

func warehouseFromRecord(record pluginsdk.DataRecord) (warehouse, error) {
	item := warehouse{
		ID: dataString(record, "id"), Code: dataString(record, "code"), Name: dataString(record, "name"),
		Address: dataString(record, "address"), ContactName: dataString(record, "contact_name"), ContactPhone: dataString(record, "contact_phone"),
		Status: dataString(record, "status"), DisableReason: dataString(record, "disable_reason"),
		Version: record.Version, CreatedAt: dataString(record, "created_at"), UpdatedAt: dataString(record, "updated_at"), scope: recordScope(record),
	}
	if item.ID == "" || item.Code == "" || item.Name == "" || (item.Status != "active" && item.Status != "disabled") {
		return warehouse{}, fmt.Errorf("warehouse record is incomplete")
	}
	return item, nil
}

func warehouseMatchesWrite(item warehouse, input warehouseWriteRequest) bool {
	return item.Code == input.Code && item.Name == input.Name && item.Address == input.Address &&
		item.ContactName == input.ContactName && item.ContactPhone == input.ContactPhone
}

func warehouseAuditDetail(item warehouse) map[string]any {
	return map[string]any{"code": item.Code, "status": item.Status, "organizationId": item.scope.OrganizationID}
}

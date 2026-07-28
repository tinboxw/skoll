package main

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type topologyKind string

const (
	topologyArea     topologyKind = "warehouse_area"
	topologyLocation topologyKind = "warehouse_location"

	warehouseAreaTable     = "warehouse_areas"
	warehouseLocationTable = "warehouse_locations"
)

var topologyDecimal = regexp.MustCompile(`^-?(?:0|[1-9][0-9]{0,2})(?:\.[0-9]{1,3})?$`)

type topologyNode struct {
	ID             string `json:"id"`
	WarehouseID    string `json:"warehouseId"`
	AreaID         string `json:"areaId,omitempty"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	TemperatureMin string `json:"temperatureMin,omitempty"`
	TemperatureMax string `json:"temperatureMax,omitempty"`
	LocationType   string `json:"locationType,omitempty"`
	Status         string `json:"status"`
	DisableReason  string `json:"disableReason,omitempty"`
	Version        int64  `json:"version"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
	scope          employeeScope
}

type topologyWriteRequest struct {
	TenantID       string `json:"tenantId"`
	OrganizationID string `json:"organizationId"`
	WarehouseID    string `json:"warehouseId"`
	AreaID         string `json:"areaId,omitempty"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	TemperatureMin string `json:"temperatureMin,omitempty"`
	TemperatureMax string `json:"temperatureMax,omitempty"`
	LocationType   string `json:"locationType,omitempty"`
	Version        int64  `json:"version,omitempty"`
}

func (s *server) listTopology(kind topologyKind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		filter, err := topologyFilter(kind, r.URL.Query())
		if err != nil {
			writeServiceError(w, err)
			return
		}
		items, err := s.queryTopology(ctx, topologyPermission(kind, "read"), kind, filter)
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
}

func (s *server) createTopology(kind topologyKind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, err := s.requestContext(r)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		key, err := mutationKey(r, string(kind)+"-create")
		if err != nil {
			writeServiceError(w, err)
			return
		}
		var input topologyWriteRequest
		if !decodeJSON(w, r, &input) {
			return
		}
		if err = validateTopologyWrite(kind, &input, false); err != nil {
			writeServiceError(w, err)
			return
		}
		scope, err := s.exactWriteScope(ctx, topologyPermission(kind, "create"), input.TenantID, input.OrganizationID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		item := topologyNode{
			ID: stableTopologyID(kind, key), WarehouseID: input.WarehouseID, AreaID: input.AreaID,
			Code: input.Code, Name: input.Name, TemperatureMin: input.TemperatureMin,
			TemperatureMax: input.TemperatureMax, LocationType: input.LocationType, Status: "active", scope: scope,
		}
		if err = s.validateTopologyParents(ctx, topologyPermission(kind, "create"), kind, scope, item); err != nil {
			writeServiceError(w, err)
			return
		}
		if err = s.ensureTopologyCodeUnique(ctx, topologyPermission(kind, "create"), kind, scope, item.ID, item); err != nil {
			writeServiceError(w, err)
			return
		}
		var created topologyNode
		err = s.transaction(ctx, func(tx context.Context) error {
			result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{
				Table: topologyTable(kind), Operation: pluginsdk.DataMutationInsert,
				Scope: topologyIntent(kind, "create", scope), Key: map[string]pluginsdk.DataValue{"id": stringValue(item.ID)},
				Values: topologyValues(kind, item), Returning: topologyFields(kind), IdempotencyKey: key,
			})
			if mutationErr != nil {
				return mutationErr
			}
			created, mutationErr = topologyFromMutation(kind, result)
			if mutationErr != nil {
				return mutationErr
			}
			return s.audit(tx, "pharma_oa."+string(kind)+".create", created.ID, pluginsdk.AuditRiskMedium, topologyAuditDetail(created))
		})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeCreated(w, map[string]any{"item": created})
	}
}

func (s *server) updateTopology(kind topologyKind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, err := s.requestContext(r)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		key, err := mutationKey(r, string(kind)+"-update")
		if err != nil {
			writeServiceError(w, err)
			return
		}
		var input topologyWriteRequest
		if !decodeJSON(w, r, &input) {
			return
		}
		if err = validateTopologyWrite(kind, &input, true); err != nil {
			writeServiceError(w, err)
			return
		}
		scope, err := s.exactWriteScope(ctx, topologyPermission(kind, "update"), input.TenantID, input.OrganizationID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		current, err := s.getTopology(ctx, topologyPermission(kind, "update"), kind, scope, r.PathValue("id"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		replayCandidate := current.Version > 1 && input.Version == current.Version-1 && topologyMatchesWrite(current, input)
		if input.Version != current.Version && !replayCandidate {
			writeServiceError(w, topologyHTTPError(kind, http.StatusConflict, "stale", "version is stale"))
			return
		}
		next := current
		next.WarehouseID, next.AreaID, next.Code, next.Name = input.WarehouseID, input.AreaID, input.Code, input.Name
		next.TemperatureMin, next.TemperatureMax, next.LocationType = input.TemperatureMin, input.TemperatureMax, input.LocationType
		if !replayCandidate {
			if kind == topologyArea && current.WarehouseID != next.WarehouseID {
				if err = s.ensureTopologyParentUnused(ctx, topologyPermission(kind, "update"), topologyLocation, "area_id", current.ID, false, "warehouse_area_in_use"); err != nil {
					writeServiceError(w, err)
					return
				}
			}
			if err = s.validateTopologyParents(ctx, topologyPermission(kind, "update"), kind, scope, next); err != nil {
				writeServiceError(w, err)
				return
			}
			if err = s.ensureTopologyCodeUnique(ctx, topologyPermission(kind, "update"), kind, scope, current.ID, next); err != nil {
				writeServiceError(w, err)
				return
			}
		}
		updated, err := s.mutateTopology(ctx, kind, "update", key, next, input.Version, pluginsdk.AuditRiskMedium)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeOK(w, map[string]any{"item": updated})
	}
}

func (s *server) changeTopologyStatus(kind topologyKind, enabled bool) http.HandlerFunc {
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
		key, err := mutationKey(r, string(kind)+"-"+action)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		var input warehouseStatusRequest
		if !decodeJSON(w, r, &input) {
			return
		}
		input.TenantID, input.OrganizationID = strings.TrimSpace(input.TenantID), strings.TrimSpace(input.OrganizationID)
		input.Reason = strings.TrimSpace(input.Reason)
		if input.TenantID == "" || input.OrganizationID == "" || input.Version < 1 || (!enabled && input.Reason == "") ||
			len(input.Reason) > 500 || (enabled && input.Reason != "") {
			writeServiceError(w, topologyHTTPError(kind, http.StatusBadRequest, "invalid_status", "scope, current version, and a bounded disable reason are required"))
			return
		}
		permission := topologyPermission(kind, action)
		scope, err := s.exactWriteScope(ctx, permission, input.TenantID, input.OrganizationID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		current, err := s.getTopology(ctx, permission, kind, scope, r.PathValue("id"))
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
			writeServiceError(w, topologyHTTPError(kind, http.StatusConflict, "stale", "version is stale"))
			return
		}
		if input.Version == current.Version && current.Status == targetStatus {
			writeServiceError(w, topologyHTTPError(kind, http.StatusConflict, "status_unchanged", "node is already in the requested status"))
			return
		}
		if !replayCandidate {
			if enabled {
				if err = s.validateTopologyParents(ctx, permission, kind, scope, current); err != nil {
					writeServiceError(w, err)
					return
				}
			} else if kind == topologyArea {
				if err = s.ensureTopologyParentUnused(ctx, permission, topologyLocation, "area_id", current.ID, true, "warehouse_area_in_use"); err != nil {
					writeServiceError(w, err)
					return
				}
			}
		}
		current.Status, current.DisableReason = targetStatus, targetReason
		updated, err := s.mutateTopology(ctx, kind, action, key, current, input.Version, pluginsdk.AuditRiskHigh)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeOK(w, map[string]any{"item": updated})
	}
}

func (s *server) warehouseMovementEligibility(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	areaID, locationID := strings.TrimSpace(r.URL.Query().Get("areaId")), strings.TrimSpace(r.URL.Query().Get("locationId"))
	if areaID == "" || locationID == "" || len(areaID) > 128 || len(locationID) > 128 {
		writeServiceError(w, newHTTPError(http.StatusBadRequest, "invalid_movement_location", "areaId and locationId are required"))
		return
	}
	permission := warehousePermission("movement")
	item, err := s.findWarehouse(ctx, permission, r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	area, err := s.getTopology(ctx, permission, topologyArea, item.scope, areaID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	location, err := s.getTopology(ctx, permission, topologyLocation, item.scope, locationID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if item.Status != "active" {
		writeServiceError(w, newHTTPError(http.StatusConflict, "warehouse_movement_ineligible", "warehouse is disabled"))
		return
	}
	if area.WarehouseID != item.ID || area.Status != "active" {
		writeServiceError(w, newHTTPError(http.StatusConflict, "warehouse_movement_ineligible", "area is disabled or belongs to another warehouse"))
		return
	}
	if location.WarehouseID != item.ID || location.AreaID != area.ID || location.Status != "active" {
		writeServiceError(w, newHTTPError(http.StatusConflict, "warehouse_movement_ineligible", "location is disabled or belongs to another topology branch"))
		return
	}
	writeOK(w, map[string]any{"eligible": true, "warehouse": item, "area": area, "location": location})
}

func (s *server) validateTopologyParents(ctx context.Context, permission pluginsdk.Permission, kind topologyKind, scope employeeScope, item topologyNode) error {
	parent, err := s.getWarehouse(ctx, permission, scope, item.WarehouseID)
	if err != nil {
		return topologyHTTPError(kind, http.StatusUnprocessableEntity, "invalid_parent", "active parent warehouse is required")
	}
	if parent.Status != "active" {
		return topologyHTTPError(kind, http.StatusConflict, "parent_disabled", "parent warehouse is disabled")
	}
	if kind == topologyArea {
		return nil
	}
	area, err := s.getTopology(ctx, permission, topologyArea, scope, item.AreaID)
	if err != nil {
		return topologyHTTPError(kind, http.StatusUnprocessableEntity, "invalid_parent", "active parent area is required")
	}
	if area.WarehouseID != item.WarehouseID {
		return topologyHTTPError(kind, http.StatusConflict, "parent_mismatch", "area does not belong to the selected warehouse")
	}
	if area.Status != "active" {
		return topologyHTTPError(kind, http.StatusConflict, "parent_disabled", "parent area is disabled")
	}
	return nil
}

func (s *server) ensureTopologyParentUnused(ctx context.Context, permission pluginsdk.Permission, child topologyKind, field, parentID string, activeOnly bool, code string) error {
	value, filters := stringValue(parentID), make([]pluginsdk.DataFilter, 0, 2)
	filters = append(filters, pluginsdk.DataFilter{Field: field, Operator: pluginsdk.DataOperatorEqual, Value: &value})
	if activeOnly {
		status := stringValue("active")
		filters = append(filters, pluginsdk.DataFilter{Field: "status", Operator: pluginsdk.DataOperatorEqual, Value: &status})
	}
	filter := pluginsdk.DataFilter{All: filters}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
		Table: topologyTable(child), Fields: []string{"id"}, Scope: pluginsdk.DataScopeIntent{Permission: permission},
		Filter: &filter,
		Sort:   []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 1},
	})
	if err != nil {
		return err
	}
	if len(page.Records) > 0 {
		return newHTTPError(http.StatusConflict, code, "parent topology node still has child records")
	}
	return nil
}

func (s *server) mutateTopology(ctx context.Context, kind topologyKind, action, key string, item topologyNode, version int64, risk pluginsdk.AuditRisk) (topologyNode, error) {
	var updated topologyNode
	err := s.transaction(ctx, func(tx context.Context) error {
		result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{
			Table: topologyTable(kind), Operation: pluginsdk.DataMutationUpdate, Scope: topologyIntent(kind, action, item.scope),
			Key: map[string]pluginsdk.DataValue{"id": stringValue(item.ID)}, Values: topologyValues(kind, item),
			Returning: topologyFields(kind), IdempotencyKey: key, ExpectedVersion: &version,
		})
		if mutationErr != nil {
			return mutationErr
		}
		updated, mutationErr = topologyFromMutation(kind, result)
		if mutationErr != nil {
			return mutationErr
		}
		return s.audit(tx, "pharma_oa."+string(kind)+"."+action, updated.ID, risk, topologyAuditDetail(updated))
	})
	return updated, err
}

func (s *server) ensureTopologyCodeUnique(ctx context.Context, permission pluginsdk.Permission, kind topologyKind, scope employeeScope, excludeID string, item topologyNode) error {
	parentField, parentID := "warehouse_id", item.WarehouseID
	if kind == topologyLocation {
		parentField, parentID = "area_id", item.AreaID
	}
	codeValue, parentValue := stringValue(item.Code), stringValue(parentID)
	filter := pluginsdk.DataFilter{All: []pluginsdk.DataFilter{
		{Field: parentField, Operator: pluginsdk.DataOperatorEqual, Value: &parentValue},
		{Field: "code", Operator: pluginsdk.DataOperatorEqual, Value: &codeValue},
	}}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
		Table: topologyTable(kind), Fields: topologyFields(kind), Scope: pluginsdk.DataScopeIntent{Permission: permission, Filter: scopeFilter(scope)},
		Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 2},
	})
	if err != nil {
		return err
	}
	for _, record := range page.Records {
		existing, parseErr := topologyFromRecord(kind, record)
		if parseErr != nil {
			return parseErr
		}
		if existing.ID != excludeID {
			return topologyHTTPError(kind, http.StatusConflict, "duplicate_code", "code already exists under this parent")
		}
	}
	return nil
}

func (s *server) getTopology(ctx context.Context, permission pluginsdk.Permission, kind topologyKind, scope employeeScope, id string) (topologyNode, error) {
	id = strings.TrimSpace(id)
	if id == "" || len(id) > 128 {
		return topologyNode{}, topologyHTTPError(kind, http.StatusBadRequest, "invalid", "id is invalid")
	}
	idValue := stringValue(id)
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
		Table: topologyTable(kind), Fields: topologyFields(kind), Scope: pluginsdk.DataScopeIntent{Permission: permission, Filter: scopeFilter(scope)},
		Filter: &pluginsdk.DataFilter{Field: "id", Operator: pluginsdk.DataOperatorEqual, Value: &idValue},
		Sort:   []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 1},
	})
	if err != nil {
		return topologyNode{}, err
	}
	if len(page.Records) != 1 {
		return topologyNode{}, topologyHTTPError(kind, http.StatusNotFound, "not_found", "node was not found")
	}
	return topologyFromRecord(kind, page.Records[0])
}

func (s *server) queryTopology(ctx context.Context, permission pluginsdk.Permission, kind topologyKind, filter *pluginsdk.DataFilter) ([]topologyNode, error) {
	items, cursor := make([]topologyNode, 0), ""
	for pageNumber := 0; pageNumber < 25; pageNumber++ {
		page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
			Table: topologyTable(kind), Fields: topologyFields(kind), Scope: pluginsdk.DataScopeIntent{Permission: permission},
			Filter: filter, Sort: []pluginsdk.DataSort{{Field: "updated_at", Direction: pluginsdk.DataSortDescending}},
			Page: pluginsdk.DataPageRequest{Cursor: cursor, Limit: 200},
		})
		if err != nil {
			return nil, err
		}
		for _, record := range page.Records {
			item, parseErr := topologyFromRecord(kind, record)
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
	return nil, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorLimitExceeded, topologyTable(kind), "topology query exceeds 5000 records", false)
}

func (s *server) findWarehouse(ctx context.Context, permission pluginsdk.Permission, id string) (warehouse, error) {
	id = strings.TrimSpace(id)
	if id == "" || len(id) > 128 {
		return warehouse{}, newHTTPError(http.StatusBadRequest, "invalid_warehouse", "warehouse id is invalid")
	}
	value := stringValue(id)
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
		Table: warehouseTable, Fields: warehouseFields, Scope: pluginsdk.DataScopeIntent{Permission: permission},
		Filter: &pluginsdk.DataFilter{Field: "id", Operator: pluginsdk.DataOperatorEqual, Value: &value},
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

func topologyFilter(kind topologyKind, query mapStringValues) (*pluginsdk.DataFilter, error) {
	filters := make([]pluginsdk.DataFilter, 0, 5)
	for _, pair := range []struct {
		field string
		value string
	}{
		{field: "warehouse_id", value: strings.TrimSpace(query.Get("warehouseId"))},
		{field: "area_id", value: strings.TrimSpace(query.Get("areaId"))},
		{field: "status", value: strings.TrimSpace(query.Get("status"))},
	} {
		if pair.field == "area_id" && kind != topologyLocation {
			continue
		}
		if pair.value == "" {
			continue
		}
		if len(pair.value) > 128 || (pair.field == "status" && pair.value != "active" && pair.value != "disabled") {
			return nil, topologyHTTPError(kind, http.StatusBadRequest, "invalid_filter", "topology filter is invalid")
		}
		value := stringValue(pair.value)
		filters = append(filters, pluginsdk.DataFilter{Field: pair.field, Operator: pluginsdk.DataOperatorEqual, Value: &value})
	}
	keyword := strings.TrimSpace(query.Get("keyword"))
	if len(keyword) > 100 {
		return nil, topologyHTTPError(kind, http.StatusBadRequest, "invalid_filter", "keyword is too long")
	}
	if keyword != "" {
		value := stringValue(keyword)
		filters = append(filters, pluginsdk.DataFilter{Any: []pluginsdk.DataFilter{
			{Field: "code", Operator: pluginsdk.DataOperatorContains, Value: &value},
			{Field: "name", Operator: pluginsdk.DataOperatorContains, Value: &value},
		}})
	}
	if len(filters) == 0 {
		return nil, nil
	}
	if len(filters) == 1 {
		return &filters[0], nil
	}
	return &pluginsdk.DataFilter{All: filters}, nil
}

type mapStringValues interface {
	Get(string) string
}

func validateTopologyWrite(kind topologyKind, input *topologyWriteRequest, update bool) error {
	input.TenantID, input.OrganizationID = strings.TrimSpace(input.TenantID), strings.TrimSpace(input.OrganizationID)
	input.WarehouseID, input.AreaID = strings.TrimSpace(input.WarehouseID), strings.TrimSpace(input.AreaID)
	input.Code, input.Name = strings.ToUpper(strings.TrimSpace(input.Code)), strings.TrimSpace(input.Name)
	input.TemperatureMin, input.TemperatureMax = strings.TrimSpace(input.TemperatureMin), strings.TrimSpace(input.TemperatureMax)
	input.LocationType = strings.ToLower(strings.TrimSpace(input.LocationType))
	if input.TenantID == "" || input.OrganizationID == "" || input.WarehouseID == "" || len(input.WarehouseID) > 128 ||
		input.Code == "" || len(input.Code) > 64 || !validWarehouseCode(input.Code) || input.Name == "" || len(input.Name) > 160 ||
		(update && input.Version < 1) {
		return topologyHTTPError(kind, http.StatusBadRequest, "invalid", "scope, parent, code, name, and current version are required")
	}
	switch kind {
	case topologyArea:
		input.AreaID, input.LocationType = "", ""
		if err := normalizeTemperatureRange(input); err != nil {
			return topologyHTTPError(kind, http.StatusBadRequest, "invalid_temperature", err.Error())
		}
	case topologyLocation:
		input.TemperatureMin, input.TemperatureMax = "", ""
		if input.AreaID == "" || len(input.AreaID) > 128 || !validLocationType(input.LocationType) {
			return topologyHTTPError(kind, http.StatusBadRequest, "invalid", "area and a current location type are required")
		}
	default:
		return newHTTPError(http.StatusInternalServerError, "invalid_topology_kind", "topology kind is invalid")
	}
	return nil
}

func normalizeTemperatureRange(input *topologyWriteRequest) error {
	if input.TemperatureMin == "" && input.TemperatureMax == "" {
		return nil
	}
	if input.TemperatureMin == "" || input.TemperatureMax == "" {
		return fmt.Errorf("both temperature bounds are required")
	}
	minimum, err := normalizeTopologyDecimal(input.TemperatureMin)
	if err != nil {
		return fmt.Errorf("minimum temperature is invalid")
	}
	maximum, err := normalizeTopologyDecimal(input.TemperatureMax)
	if err != nil {
		return fmt.Errorf("maximum temperature is invalid")
	}
	minValue, _ := strconv.ParseFloat(minimum, 64)
	maxValue, _ := strconv.ParseFloat(maximum, 64)
	if minValue < -80 || maxValue > 80 || minValue > maxValue {
		return fmt.Errorf("temperature range must be ordered inside -80 to 80")
	}
	input.TemperatureMin, input.TemperatureMax = minimum, maximum
	return nil
}

func normalizeTopologyDecimal(value string) (string, error) {
	if !topologyDecimal.MatchString(value) {
		return "", fmt.Errorf("decimal has invalid precision")
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return "", err
	}
	normalized := strconv.FormatFloat(number, 'f', -1, 64)
	if normalized == "-0" {
		normalized = "0"
	}
	return normalized, nil
}

func validLocationType(value string) bool {
	switch value {
	case "standard", "cold_chain", "quarantine", "defective", "returned":
		return true
	default:
		return false
	}
}

func topologyTable(kind topologyKind) string {
	if kind == topologyArea {
		return warehouseAreaTable
	}
	return warehouseLocationTable
}

func topologyFields(kind topologyKind) []string {
	common := []string{"id", "warehouse_id"}
	if kind == topologyArea {
		common = append(common, "code", "name", "temperature_min", "temperature_max")
	} else {
		common = append(common, "area_id", "code", "name", "location_type")
	}
	return append(common, "status", "disable_reason", "tenant_id", "organization_id", "owner_id", "created_at", "updated_at")
}

func topologyPermission(kind topologyKind, action string) pluginsdk.Permission {
	return pluginsdk.Permission{Resource: "pharma_oa." + string(kind), Action: action}
}

func topologyIntent(kind topologyKind, action string, scope employeeScope) pluginsdk.DataScopeIntent {
	return pluginsdk.DataScopeIntent{Permission: topologyPermission(kind, action), Filter: scopeFilter(scope)}
}

func topologyValues(kind topologyKind, item topologyNode) map[string]pluginsdk.DataValue {
	values := map[string]pluginsdk.DataValue{
		"warehouse_id": stringValue(item.WarehouseID), "code": stringValue(item.Code), "name": stringValue(item.Name),
		"status": stringValue(item.Status), "disable_reason": nullableStringValue(item.DisableReason),
	}
	if kind == topologyArea {
		values["temperature_min"] = nullableDecimalValue(item.TemperatureMin)
		values["temperature_max"] = nullableDecimalValue(item.TemperatureMax)
	} else {
		values["area_id"] = stringValue(item.AreaID)
		values["location_type"] = stringValue(item.LocationType)
	}
	return values
}

func nullableDecimalValue(value string) pluginsdk.DataValue {
	if value == "" {
		return pluginsdk.DataValue{Type: pluginsdk.DataValueNull}
	}
	return decimalValue(value)
}

func topologyFromMutation(kind topologyKind, result pluginsdk.DataMutationResult) (topologyNode, error) {
	if result.Record == nil || result.RowsAffected != 1 {
		return topologyNode{}, fmt.Errorf("%s mutation returned no record", kind)
	}
	return topologyFromRecord(kind, *result.Record)
}

func topologyFromRecord(kind topologyKind, record pluginsdk.DataRecord) (topologyNode, error) {
	item := topologyNode{
		ID: dataString(record, "id"), WarehouseID: dataString(record, "warehouse_id"), AreaID: dataString(record, "area_id"),
		Code: dataString(record, "code"), Name: dataString(record, "name"), TemperatureMin: dataString(record, "temperature_min"),
		TemperatureMax: dataString(record, "temperature_max"), LocationType: dataString(record, "location_type"),
		Status: dataString(record, "status"), DisableReason: dataString(record, "disable_reason"), Version: record.Version,
		CreatedAt: dataString(record, "created_at"), UpdatedAt: dataString(record, "updated_at"), scope: recordScope(record),
	}
	if item.ID == "" || item.WarehouseID == "" || item.Code == "" || item.Name == "" ||
		(item.Status != "active" && item.Status != "disabled") || (kind == topologyLocation && (item.AreaID == "" || !validLocationType(item.LocationType))) {
		return topologyNode{}, fmt.Errorf("%s record is incomplete", kind)
	}
	return item, nil
}

func topologyMatchesWrite(item topologyNode, input topologyWriteRequest) bool {
	return item.WarehouseID == input.WarehouseID && item.AreaID == input.AreaID && item.Code == input.Code &&
		item.Name == input.Name && item.TemperatureMin == input.TemperatureMin &&
		item.TemperatureMax == input.TemperatureMax && item.LocationType == input.LocationType
}

func stableTopologyID(kind topologyKind, key string) string {
	digest := strings.TrimPrefix(stableWarehouseID(string(kind)+"\x00"+key), "warehouse-")
	return strings.ReplaceAll(string(kind), "_", "-") + "-" + digest
}

func topologyAuditDetail(item topologyNode) map[string]any {
	detail := map[string]any{"code": item.Code, "status": item.Status, "organizationId": item.scope.OrganizationID, "warehouseId": item.WarehouseID}
	if item.AreaID != "" {
		detail["areaId"] = item.AreaID
	}
	return detail
}

func topologyHTTPError(kind topologyKind, status int, suffix, message string) error {
	return newHTTPError(status, string(kind)+"_"+suffix, message)
}

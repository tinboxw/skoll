package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const (
	catalogTable = "catalogs"
	productTable = "products"
)

var (
	catalogFields = []string{"id", "catalog_type", "code", "name", "description", "parent_id", "symbol", "decimal_places", "unified_social_credit_code", "license_number", "status", "disable_reason", "tenant_id", "organization_id", "owner_id", "created_at", "updated_at"}
	productFields = []string{"id", "code", "sku", "name", "generic_name", "category_id", "unit_id", "manufacturer_id", "dosage_form", "specification", "approval_number", "barcode", "storage_condition", "temperature_min", "temperature_max", "status", "disable_reason", "tenant_id", "organization_id", "owner_id", "created_at", "updated_at"}
)

type catalogItem struct {
	ID                      string `json:"id"`
	Type                    string `json:"type"`
	Code                    string `json:"code"`
	Name                    string `json:"name"`
	Description             string `json:"description"`
	ParentID                string `json:"parentId,omitempty"`
	Symbol                  string `json:"symbol,omitempty"`
	DecimalPlaces           int64  `json:"decimalPlaces"`
	UnifiedSocialCreditCode string `json:"unifiedSocialCreditCode,omitempty"`
	LicenseNumber           string `json:"licenseNumber,omitempty"`
	Status                  string `json:"status"`
	DisableReason           string `json:"disableReason,omitempty"`
	Version                 int64  `json:"version"`
	CreatedAt               string `json:"createdAt"`
	UpdatedAt               string `json:"updatedAt"`
	scope                   employeeScope
}

type catalogWriteRequest struct {
	TenantID                string `json:"tenantId"`
	OrganizationID          string `json:"organizationId"`
	Code                    string `json:"code"`
	Name                    string `json:"name"`
	Description             string `json:"description"`
	ParentID                string `json:"parentId,omitempty"`
	Symbol                  string `json:"symbol,omitempty"`
	DecimalPlaces           int64  `json:"decimalPlaces,omitempty"`
	UnifiedSocialCreditCode string `json:"unifiedSocialCreditCode,omitempty"`
	LicenseNumber           string `json:"licenseNumber,omitempty"`
	Version                 int64  `json:"version,omitempty"`
}

type product struct {
	ID               string `json:"id"`
	Code             string `json:"code"`
	SKU              string `json:"sku"`
	Name             string `json:"name"`
	GenericName      string `json:"genericName"`
	CategoryID       string `json:"categoryId"`
	UnitID           string `json:"unitId"`
	ManufacturerID   string `json:"manufacturerId"`
	DosageForm       string `json:"dosageForm"`
	Specification    string `json:"specification"`
	ApprovalNumber   string `json:"approvalNumber"`
	Barcode          string `json:"barcode,omitempty"`
	StorageCondition string `json:"storageCondition"`
	TemperatureMin   int64  `json:"temperatureMin"`
	TemperatureMax   int64  `json:"temperatureMax"`
	Status           string `json:"status"`
	DisableReason    string `json:"disableReason,omitempty"`
	Version          int64  `json:"version"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
	scope            employeeScope
}

type productWriteRequest struct {
	TenantID         string `json:"tenantId"`
	OrganizationID   string `json:"organizationId"`
	Code             string `json:"code"`
	SKU              string `json:"sku"`
	Name             string `json:"name"`
	GenericName      string `json:"genericName"`
	CategoryID       string `json:"categoryId"`
	UnitID           string `json:"unitId"`
	ManufacturerID   string `json:"manufacturerId"`
	DosageForm       string `json:"dosageForm"`
	Specification    string `json:"specification"`
	ApprovalNumber   string `json:"approvalNumber"`
	Barcode          string `json:"barcode,omitempty"`
	StorageCondition string `json:"storageCondition"`
	TemperatureMin   int64  `json:"temperatureMin"`
	TemperatureMax   int64  `json:"temperatureMax"`
	Version          int64  `json:"version,omitempty"`
}

func (s *server) listCatalogs(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, err := s.requestContext(r)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		limit, err := catalogPageLimit(r)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		filter, err := catalogFilter(kind, r.URL.Query().Get("keyword"), r.URL.Query().Get("status"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{
			Table: catalogTable, Fields: catalogFields,
			Scope: pluginsdk.DataScopeIntent{Permission: catalogPermission(kind, "read")}, Filter: filter,
			Sort: []pluginsdk.DataSort{{Field: "updated_at", Direction: pluginsdk.DataSortDescending}, {Field: "id", Direction: pluginsdk.DataSortAscending}},
			Page: pluginsdk.DataPageRequest{Cursor: r.URL.Query().Get("cursor"), Limit: limit},
		})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		items := make([]catalogItem, 0, len(page.Records))
		for _, record := range page.Records {
			item, parseErr := catalogFromRecord(record)
			if parseErr != nil {
				writeServiceError(w, parseErr)
				return
			}
			items = append(items, item)
		}
		writeOK(w, map[string]any{"items": items, "pageInfo": map[string]any{"nextCursor": page.NextCursor, "hasMore": page.HasMore, "limit": limit}})
	}
}

func (s *server) createCatalog(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, err := s.requestContext(r)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		key, err := mutationKey(r, kind+"-create")
		if err != nil {
			writeServiceError(w, err)
			return
		}
		var input catalogWriteRequest
		if !decodeJSON(w, r, &input) {
			return
		}
		if err = validateCatalogWrite(kind, &input, false); err != nil {
			writeServiceError(w, err)
			return
		}
		scope, err := s.exactWriteScope(ctx, catalogPermission(kind, "create"), input.TenantID, input.OrganizationID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		item := catalogItem{ID: resourceID(kind, s.newID()), Type: kind, Code: input.Code, Name: input.Name, Description: input.Description, ParentID: input.ParentID, Symbol: input.Symbol, DecimalPlaces: input.DecimalPlaces, UnifiedSocialCreditCode: input.UnifiedSocialCreditCode, LicenseNumber: input.LicenseNumber, Status: "active", scope: scope}
		var created catalogItem
		err = s.transaction(ctx, func(tx context.Context) error {
			if uniqueErr := s.ensureCatalogUnique(tx, kind, catalogPermission(kind, "create"), scope, "", input.Code, input.UnifiedSocialCreditCode); uniqueErr != nil {
				return uniqueErr
			}
			if hierarchyErr := s.validateCategoryParent(tx, catalogPermission(kind, "create"), scope, item.ID, input.ParentID); hierarchyErr != nil {
				return hierarchyErr
			}
			result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{Table: catalogTable, Operation: pluginsdk.DataMutationInsert, Scope: catalogIntent(kind, "create", scope), Key: map[string]pluginsdk.DataValue{"id": stringValue(item.ID)}, Values: catalogValues(item), Returning: catalogFields, IdempotencyKey: key})
			if mutationErr != nil {
				return mutationErr
			}
			created, mutationErr = catalogFromMutation(result)
			if mutationErr != nil {
				return mutationErr
			}
			return s.audit(tx, "pharma_oa."+kind+".create", created.ID, pluginsdk.AuditRiskMedium, map[string]any{"code": created.Code})
		})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeCreated(w, map[string]any{"item": created})
	}
}

func (s *server) updateCatalog(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, err := s.requestContext(r)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		key, err := mutationKey(r, kind+"-update")
		if err != nil {
			writeServiceError(w, err)
			return
		}
		var input catalogWriteRequest
		if !decodeJSON(w, r, &input) {
			return
		}
		if err = validateCatalogWrite(kind, &input, true); err != nil {
			writeServiceError(w, err)
			return
		}
		current, err := s.getCatalogUnscoped(ctx, kind, catalogPermission(kind, "update"), r.PathValue("id"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		if input.ParentID == current.ID {
			writeServiceError(w, newHTTPError(http.StatusBadRequest, "invalid_parent", "catalog cannot reference itself"))
			return
		}
		current.Code, current.Name, current.Description = input.Code, input.Name, input.Description
		current.ParentID, current.Symbol, current.DecimalPlaces = input.ParentID, input.Symbol, input.DecimalPlaces
		current.UnifiedSocialCreditCode, current.LicenseNumber = input.UnifiedSocialCreditCode, input.LicenseNumber
		var updated catalogItem
		err = s.transaction(ctx, func(tx context.Context) error {
			if input.Version != current.Version {
				return newHTTPError(http.StatusConflict, "stale_catalog", "catalog version is stale")
			}
			if uniqueErr := s.ensureCatalogUnique(tx, kind, catalogPermission(kind, "update"), current.scope, current.ID, input.Code, input.UnifiedSocialCreditCode); uniqueErr != nil {
				return uniqueErr
			}
			if hierarchyErr := s.validateCategoryParent(tx, catalogPermission(kind, "update"), current.scope, current.ID, input.ParentID); hierarchyErr != nil {
				return hierarchyErr
			}
			result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{Table: catalogTable, Operation: pluginsdk.DataMutationUpdate, Scope: catalogIntent(kind, "update", current.scope), Key: map[string]pluginsdk.DataValue{"id": stringValue(current.ID)}, Values: catalogValues(current), Returning: catalogFields, IdempotencyKey: key, ExpectedVersion: &input.Version})
			if mutationErr != nil {
				return mutationErr
			}
			updated, mutationErr = catalogFromMutation(result)
			if mutationErr != nil {
				return mutationErr
			}
			return s.audit(tx, "pharma_oa."+kind+".update", updated.ID, pluginsdk.AuditRiskMedium, map[string]any{"code": updated.Code})
		})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeOK(w, map[string]any{"item": updated})
	}
}

func (s *server) changeCatalogStatus(kind string, enabled bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, err := s.requestContext(r)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		action := "disable"
		if enabled {
			action = "enable"
		}
		key, err := mutationKey(r, kind+"-"+action)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		var input partyStatusRequest
		if !decodeJSON(w, r, &input) {
			return
		}
		input.Reason = strings.TrimSpace(input.Reason)
		if input.Version < 1 || (!enabled && input.Reason == "") || len(input.Reason) > 500 {
			writeServiceError(w, newHTTPError(http.StatusBadRequest, "invalid_request", "current version and a bounded disable reason are required"))
			return
		}
		current, err := s.getCatalogUnscoped(ctx, kind, catalogPermission(kind, action), r.PathValue("id"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		if enabled {
			current.Status, current.DisableReason = "active", ""
		} else {
			current.Status, current.DisableReason = "disabled", input.Reason
		}
		var updated catalogItem
		err = s.transaction(ctx, func(tx context.Context) error {
			if input.Version != current.Version {
				return newHTTPError(http.StatusConflict, "stale_catalog", "catalog version is stale")
			}
			if !enabled {
				if referenceErr := s.ensureCatalogUnused(tx, kind, catalogPermission(kind, action), current); referenceErr != nil {
					return referenceErr
				}
			} else if hierarchyErr := s.validateCategoryParent(tx, catalogPermission(kind, action), current.scope, current.ID, current.ParentID); hierarchyErr != nil {
				return hierarchyErr
			}
			result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{Table: catalogTable, Operation: pluginsdk.DataMutationUpdate, Scope: catalogIntent(kind, action, current.scope), Key: map[string]pluginsdk.DataValue{"id": stringValue(current.ID)}, Values: catalogValues(current), Returning: catalogFields, IdempotencyKey: key, ExpectedVersion: &input.Version})
			if mutationErr != nil {
				return mutationErr
			}
			updated, mutationErr = catalogFromMutation(result)
			if mutationErr != nil {
				return mutationErr
			}
			return s.audit(tx, "pharma_oa."+kind+"."+action, updated.ID, pluginsdk.AuditRiskHigh, map[string]any{"code": updated.Code, "status": updated.Status})
		})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeOK(w, map[string]any{"item": updated})
	}
}

func (s *server) listProducts(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	limit, err := catalogPageLimit(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	filter, err := productFilter(r.URL.Query().Get("keyword"), r.URL.Query().Get("status"), r.URL.Query().Get("categoryId"), r.URL.Query().Get("manufacturerId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{Table: productTable, Fields: productFields, Scope: pluginsdk.DataScopeIntent{Permission: productPermission("read")}, Filter: filter, Sort: []pluginsdk.DataSort{{Field: "updated_at", Direction: pluginsdk.DataSortDescending}, {Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Cursor: r.URL.Query().Get("cursor"), Limit: limit}})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	items := make([]product, 0, len(page.Records))
	for _, record := range page.Records {
		item, parseErr := productFromRecord(record)
		if parseErr != nil {
			writeServiceError(w, parseErr)
			return
		}
		items = append(items, item)
	}
	writeOK(w, map[string]any{"items": items, "pageInfo": map[string]any{"nextCursor": page.NextCursor, "hasMore": page.HasMore, "limit": limit}})
}

func (s *server) createProduct(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	key, err := mutationKey(r, "product-create")
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var input productWriteRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	if err = validateProductWrite(&input, false); err != nil {
		writeServiceError(w, err)
		return
	}
	scope, err := s.exactWriteScope(ctx, productPermission("create"), input.TenantID, input.OrganizationID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	item := product{ID: resourceID("product", s.newID()), Code: input.Code, SKU: input.SKU, Name: input.Name, GenericName: input.GenericName, CategoryID: input.CategoryID, UnitID: input.UnitID, ManufacturerID: input.ManufacturerID, DosageForm: input.DosageForm, Specification: input.Specification, ApprovalNumber: input.ApprovalNumber, Barcode: input.Barcode, StorageCondition: input.StorageCondition, TemperatureMin: input.TemperatureMin, TemperatureMax: input.TemperatureMax, Status: "active", scope: scope}
	var created product
	err = s.transaction(ctx, func(tx context.Context) error {
		if uniqueErr := s.ensureProductUnique(tx, productPermission("create"), scope, "", item); uniqueErr != nil {
			return uniqueErr
		}
		if referenceErr := s.validateProductReferences(tx, productPermission("create"), scope, item); referenceErr != nil {
			return referenceErr
		}
		result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{Table: productTable, Operation: pluginsdk.DataMutationInsert, Scope: productIntent("create", scope), Key: map[string]pluginsdk.DataValue{"id": stringValue(item.ID)}, Values: productValues(item), Returning: productFields, IdempotencyKey: key})
		if mutationErr != nil {
			return mutationErr
		}
		created, mutationErr = productFromMutation(result)
		if mutationErr != nil {
			return mutationErr
		}
		return s.audit(tx, "pharma_oa.product.create", created.ID, pluginsdk.AuditRiskMedium, map[string]any{"code": created.Code, "sku": created.SKU})
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeCreated(w, map[string]any{"item": created})
}

func (s *server) updateProduct(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.requestContext(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	key, err := mutationKey(r, "product-update")
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var input productWriteRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	if err = validateProductWrite(&input, true); err != nil {
		writeServiceError(w, err)
		return
	}
	current, err := s.getProduct(ctx, productPermission("update"), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	current.Code, current.SKU, current.Name, current.GenericName = input.Code, input.SKU, input.Name, input.GenericName
	current.CategoryID, current.UnitID, current.ManufacturerID = input.CategoryID, input.UnitID, input.ManufacturerID
	current.DosageForm, current.Specification, current.ApprovalNumber, current.Barcode = input.DosageForm, input.Specification, input.ApprovalNumber, input.Barcode
	current.StorageCondition, current.TemperatureMin, current.TemperatureMax = input.StorageCondition, input.TemperatureMin, input.TemperatureMax
	updated, err := s.mutateProduct(ctx, "update", key, current, input.Version, pluginsdk.AuditRiskMedium, true)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, map[string]any{"item": updated})
}

func (s *server) changeProductStatus(enabled bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, err := s.requestContext(r)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		action := "disable"
		if enabled {
			action = "enable"
		}
		key, err := mutationKey(r, "product-"+action)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		var input partyStatusRequest
		if !decodeJSON(w, r, &input) {
			return
		}
		input.Reason = strings.TrimSpace(input.Reason)
		if input.Version < 1 || (!enabled && input.Reason == "") || len(input.Reason) > 500 {
			writeServiceError(w, newHTTPError(http.StatusBadRequest, "invalid_request", "current version and a bounded disable reason are required"))
			return
		}
		current, err := s.getProduct(ctx, productPermission(action), r.PathValue("id"))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		if enabled {
			current.Status, current.DisableReason = "active", ""
		} else {
			current.Status, current.DisableReason = "disabled", input.Reason
		}
		updated, err := s.mutateProduct(ctx, action, key, current, input.Version, pluginsdk.AuditRiskHigh, enabled)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeOK(w, map[string]any{"item": updated})
	}
}

func (s *server) mutateProduct(ctx context.Context, action, key string, item product, version int64, risk pluginsdk.AuditRisk, validateReferences bool) (product, error) {
	if version != item.Version {
		return product{}, newHTTPError(http.StatusConflict, "stale_product", "product version is stale")
	}
	var updated product
	err := s.transaction(ctx, func(tx context.Context) error {
		if action == "update" {
			if uniqueErr := s.ensureProductUnique(tx, productPermission(action), item.scope, item.ID, item); uniqueErr != nil {
				return uniqueErr
			}
		}
		if validateReferences {
			if referenceErr := s.validateProductReferences(tx, productPermission(action), item.scope, item); referenceErr != nil {
				return referenceErr
			}
		}
		result, mutationErr := s.host.DataStore.Mutate(tx, pluginsdk.DataMutation{Table: productTable, Operation: pluginsdk.DataMutationUpdate, Scope: productIntent(action, item.scope), Key: map[string]pluginsdk.DataValue{"id": stringValue(item.ID)}, Values: productValues(item), Returning: productFields, IdempotencyKey: key, ExpectedVersion: &version})
		if mutationErr != nil {
			return mutationErr
		}
		updated, mutationErr = productFromMutation(result)
		if mutationErr != nil {
			return mutationErr
		}
		return s.audit(tx, "pharma_oa.product."+action, updated.ID, risk, map[string]any{"code": updated.Code, "status": updated.Status})
	})
	return updated, err
}

func (s *server) validateProductReferences(ctx context.Context, permission pluginsdk.Permission, scope employeeScope, item product) error {
	for _, reference := range []struct{ kind, id string }{{"category", item.CategoryID}, {"unit", item.UnitID}, {"manufacturer", item.ManufacturerID}} {
		if _, err := s.getCatalog(ctx, reference.kind, permission, scope, reference.id, true); err != nil {
			return newHTTPError(http.StatusUnprocessableEntity, "invalid_catalog_reference", reference.kind+" reference is missing, disabled, or outside the trusted scope")
		}
	}
	return nil
}

func (s *server) validateCategoryParent(ctx context.Context, permission pluginsdk.Permission, scope employeeScope, currentID, parentID string) error {
	if parentID == "" {
		return nil
	}
	next := parentID
	for depth := 0; depth < 32; depth++ {
		if next == currentID {
			return newHTTPError(http.StatusConflict, "category_cycle", "category hierarchy cannot contain a cycle")
		}
		parent, err := s.getCatalog(ctx, "category", permission, scope, next, true)
		if err != nil {
			return newHTTPError(http.StatusUnprocessableEntity, "invalid_parent", "parent category is missing, disabled, or outside the trusted scope")
		}
		if parent.ParentID == "" {
			return nil
		}
		next = parent.ParentID
	}
	return newHTTPError(http.StatusUnprocessableEntity, "category_depth_exceeded", "category hierarchy exceeds 32 levels")
}

func (s *server) ensureCatalogUnused(ctx context.Context, kind string, permission pluginsdk.Permission, item catalogItem) error {
	field := map[string]string{"category": "category_id", "unit": "unit_id", "manufacturer": "manufacturer_id"}[kind]
	idValue, activeValue := stringValue(item.ID), stringValue("active")
	filter := pluginsdk.DataFilter{All: []pluginsdk.DataFilter{{Field: field, Operator: pluginsdk.DataOperatorEqual, Value: &idValue}, {Field: "status", Operator: pluginsdk.DataOperatorEqual, Value: &activeValue}}}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{Table: productTable, Fields: []string{"id"}, Scope: catalogIntent(kind, permission.Action, item.scope), Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 1}})
	if err != nil {
		return err
	}
	if len(page.Records) > 0 {
		return newHTTPError(http.StatusConflict, "catalog_in_use", "active products still reference this catalog")
	}
	if kind == "category" {
		typeValue := stringValue("category")
		childFilter := pluginsdk.DataFilter{All: []pluginsdk.DataFilter{{Field: "catalog_type", Operator: pluginsdk.DataOperatorEqual, Value: &typeValue}, {Field: "parent_id", Operator: pluginsdk.DataOperatorEqual, Value: &idValue}, {Field: "status", Operator: pluginsdk.DataOperatorEqual, Value: &activeValue}}}
		children, queryErr := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{Table: catalogTable, Fields: []string{"id"}, Scope: catalogIntent(kind, permission.Action, item.scope), Filter: &childFilter, Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 1}})
		if queryErr != nil {
			return queryErr
		}
		if len(children.Records) > 0 {
			return newHTTPError(http.StatusConflict, "catalog_in_use", "active child categories still reference this catalog")
		}
	}
	return nil
}

func (s *server) ensureCatalogUnique(ctx context.Context, kind string, permission pluginsdk.Permission, scope employeeScope, excludeID, code, credit string) error {
	kindValue, codeValue := stringValue(kind), stringValue(code)
	identity := []pluginsdk.DataFilter{{Field: "code", Operator: pluginsdk.DataOperatorEqual, Value: &codeValue}}
	if kind == "manufacturer" && credit != "" {
		creditValue := stringValue(credit)
		identity = append(identity, pluginsdk.DataFilter{Field: "unified_social_credit_code", Operator: pluginsdk.DataOperatorEqual, Value: &creditValue})
	}
	filter := pluginsdk.DataFilter{All: []pluginsdk.DataFilter{{Field: "catalog_type", Operator: pluginsdk.DataOperatorEqual, Value: &kindValue}, {Any: identity}}}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{Table: catalogTable, Fields: catalogFields, Scope: pluginsdk.DataScopeIntent{Permission: permission, Filter: scopeFilter(scope)}, Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 2}})
	if err != nil {
		return err
	}
	for _, record := range page.Records {
		found, parseErr := catalogFromRecord(record)
		if parseErr != nil {
			return parseErr
		}
		if found.ID != excludeID {
			return newHTTPError(http.StatusConflict, "duplicate_catalog", "catalog code or manufacturer credit code already exists")
		}
	}
	return nil
}

func (s *server) ensureProductUnique(ctx context.Context, permission pluginsdk.Permission, scope employeeScope, excludeID string, item product) error {
	code, sku, approval := stringValue(item.Code), stringValue(item.SKU), stringValue(item.ApprovalNumber)
	filter := pluginsdk.DataFilter{Any: []pluginsdk.DataFilter{{Field: "code", Operator: pluginsdk.DataOperatorEqual, Value: &code}, {Field: "sku", Operator: pluginsdk.DataOperatorEqual, Value: &sku}, {Field: "approval_number", Operator: pluginsdk.DataOperatorEqual, Value: &approval}}}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{Table: productTable, Fields: productFields, Scope: pluginsdk.DataScopeIntent{Permission: permission, Filter: scopeFilter(scope)}, Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 3}})
	if err != nil {
		return err
	}
	for _, record := range page.Records {
		found, parseErr := productFromRecord(record)
		if parseErr != nil {
			return parseErr
		}
		if found.ID != excludeID {
			return newHTTPError(http.StatusConflict, "duplicate_product", "product code, SKU, or approval number already exists")
		}
	}
	return nil
}

func (s *server) getCatalogUnscoped(ctx context.Context, kind string, permission pluginsdk.Permission, id string) (catalogItem, error) {
	idValue, typeValue := stringValue(strings.TrimSpace(id)), stringValue(kind)
	filter := pluginsdk.DataFilter{All: []pluginsdk.DataFilter{{Field: "id", Operator: pluginsdk.DataOperatorEqual, Value: &idValue}, {Field: "catalog_type", Operator: pluginsdk.DataOperatorEqual, Value: &typeValue}}}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{Table: catalogTable, Fields: catalogFields, Scope: pluginsdk.DataScopeIntent{Permission: permission}, Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 1}})
	if err != nil {
		return catalogItem{}, err
	}
	if len(page.Records) != 1 {
		return catalogItem{}, newHTTPError(http.StatusNotFound, "catalog_not_found", "catalog was not found")
	}
	return catalogFromRecord(page.Records[0])
}

func (s *server) getCatalog(ctx context.Context, kind string, permission pluginsdk.Permission, scope employeeScope, id string, activeOnly bool) (catalogItem, error) {
	idValue, typeValue := stringValue(strings.TrimSpace(id)), stringValue(kind)
	filters := []pluginsdk.DataFilter{{Field: "id", Operator: pluginsdk.DataOperatorEqual, Value: &idValue}, {Field: "catalog_type", Operator: pluginsdk.DataOperatorEqual, Value: &typeValue}}
	if activeOnly {
		activeValue := stringValue("active")
		filters = append(filters, pluginsdk.DataFilter{Field: "status", Operator: pluginsdk.DataOperatorEqual, Value: &activeValue})
	}
	filter := pluginsdk.DataFilter{All: filters}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{Table: catalogTable, Fields: catalogFields, Scope: pluginsdk.DataScopeIntent{Permission: permission, Filter: scopeFilter(scope)}, Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 1}})
	if err != nil {
		return catalogItem{}, err
	}
	if len(page.Records) != 1 {
		return catalogItem{}, newHTTPError(http.StatusNotFound, "catalog_not_found", "catalog was not found")
	}
	return catalogFromRecord(page.Records[0])
}

func (s *server) getProduct(ctx context.Context, permission pluginsdk.Permission, id string) (product, error) {
	idValue := stringValue(strings.TrimSpace(id))
	filter := pluginsdk.DataFilter{Field: "id", Operator: pluginsdk.DataOperatorEqual, Value: &idValue}
	page, err := s.host.DataStore.Query(ctx, pluginsdk.DataQuery{Table: productTable, Fields: productFields, Scope: pluginsdk.DataScopeIntent{Permission: permission}, Filter: &filter, Sort: []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}}, Page: pluginsdk.DataPageRequest{Limit: 1}})
	if err != nil {
		return product{}, err
	}
	if len(page.Records) != 1 {
		return product{}, newHTTPError(http.StatusNotFound, "product_not_found", "product was not found")
	}
	return productFromRecord(page.Records[0])
}

func catalogFilter(kind, keyword, status string) (*pluginsdk.DataFilter, error) {
	typeValue := stringValue(kind)
	filters := []pluginsdk.DataFilter{{Field: "catalog_type", Operator: pluginsdk.DataOperatorEqual, Value: &typeValue}}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		value := stringValue(keyword)
		filters = append(filters, pluginsdk.DataFilter{Any: []pluginsdk.DataFilter{{Field: "code", Operator: pluginsdk.DataOperatorContains, Value: &value}, {Field: "name", Operator: pluginsdk.DataOperatorContains, Value: &value}, {Field: "symbol", Operator: pluginsdk.DataOperatorContains, Value: &value}, {Field: "unified_social_credit_code", Operator: pluginsdk.DataOperatorContains, Value: &value}, {Field: "license_number", Operator: pluginsdk.DataOperatorContains, Value: &value}}})
	}
	if status = strings.TrimSpace(status); status != "" {
		if status != "active" && status != "disabled" {
			return nil, newHTTPError(http.StatusBadRequest, "invalid_status", "catalog status is invalid")
		}
		value := stringValue(status)
		filters = append(filters, pluginsdk.DataFilter{Field: "status", Operator: pluginsdk.DataOperatorEqual, Value: &value})
	}
	return &pluginsdk.DataFilter{All: filters}, nil
}

func productFilter(keyword, status, categoryID, manufacturerID string) (*pluginsdk.DataFilter, error) {
	filters := make([]pluginsdk.DataFilter, 0, 4)
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		value := stringValue(keyword)
		filters = append(filters, pluginsdk.DataFilter{Any: []pluginsdk.DataFilter{{Field: "code", Operator: pluginsdk.DataOperatorContains, Value: &value}, {Field: "sku", Operator: pluginsdk.DataOperatorContains, Value: &value}, {Field: "name", Operator: pluginsdk.DataOperatorContains, Value: &value}, {Field: "generic_name", Operator: pluginsdk.DataOperatorContains, Value: &value}, {Field: "approval_number", Operator: pluginsdk.DataOperatorContains, Value: &value}, {Field: "barcode", Operator: pluginsdk.DataOperatorContains, Value: &value}}})
	}
	if status = strings.TrimSpace(status); status != "" {
		if status != "active" && status != "disabled" {
			return nil, newHTTPError(http.StatusBadRequest, "invalid_status", "product status is invalid")
		}
		value := stringValue(status)
		filters = append(filters, pluginsdk.DataFilter{Field: "status", Operator: pluginsdk.DataOperatorEqual, Value: &value})
	}
	for field, raw := range map[string]string{"category_id": categoryID, "manufacturer_id": manufacturerID} {
		if raw = strings.TrimSpace(raw); raw != "" {
			value := stringValue(raw)
			filters = append(filters, pluginsdk.DataFilter{Field: field, Operator: pluginsdk.DataOperatorEqual, Value: &value})
		}
	}
	if len(filters) == 0 {
		return nil, nil
	}
	if len(filters) == 1 {
		return &filters[0], nil
	}
	return &pluginsdk.DataFilter{All: filters}, nil
}

func catalogPageLimit(r *http.Request) (int, error) {
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > pluginsdk.MaxDataQueryLimit {
			return 0, newHTTPError(http.StatusBadRequest, "invalid_pagination", "limit must be between 1 and 200")
		}
		return value, nil
	}
	return 100, nil
}

func validateCatalogWrite(kind string, input *catalogWriteRequest, update bool) error {
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.ParentID = strings.TrimSpace(input.ParentID)
	input.Symbol = strings.TrimSpace(input.Symbol)
	input.UnifiedSocialCreditCode = strings.ToUpper(strings.TrimSpace(input.UnifiedSocialCreditCode))
	input.LicenseNumber = strings.TrimSpace(input.LicenseNumber)
	if input.Code == "" || len(input.Code) > 64 || input.Name == "" || len(input.Name) > 255 || len(input.Description) > 512 || (update && input.Version < 1) {
		return newHTTPError(http.StatusBadRequest, "invalid_catalog", "code, name, bounded description, and current version are required")
	}
	switch kind {
	case "category":
		if input.Symbol != "" || input.DecimalPlaces != 0 || input.UnifiedSocialCreditCode != "" || input.LicenseNumber != "" {
			return newHTTPError(http.StatusBadRequest, "invalid_category", "category accepts only hierarchy fields")
		}
	case "unit":
		if input.Symbol == "" || len(input.Symbol) > 32 || input.DecimalPlaces < 0 || input.DecimalPlaces > 6 || input.ParentID != "" || input.UnifiedSocialCreditCode != "" || input.LicenseNumber != "" {
			return newHTTPError(http.StatusBadRequest, "invalid_unit", "unit symbol and decimal places from 0 to 6 are required")
		}
	case "manufacturer":
		if len(input.UnifiedSocialCreditCode) < 8 || len(input.UnifiedSocialCreditCode) > 32 || input.LicenseNumber == "" || len(input.LicenseNumber) > 128 || input.ParentID != "" || input.Symbol != "" || input.DecimalPlaces != 0 {
			return newHTTPError(http.StatusBadRequest, "invalid_manufacturer", "manufacturer credit code and license number are required")
		}
	default:
		return newHTTPError(http.StatusBadRequest, "invalid_catalog_type", "catalog type is invalid")
	}
	return nil
}

func validateProductWrite(input *productWriteRequest, update bool) error {
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.SKU = strings.ToUpper(strings.TrimSpace(input.SKU))
	input.Name = strings.TrimSpace(input.Name)
	input.GenericName = strings.TrimSpace(input.GenericName)
	input.CategoryID = strings.TrimSpace(input.CategoryID)
	input.UnitID = strings.TrimSpace(input.UnitID)
	input.ManufacturerID = strings.TrimSpace(input.ManufacturerID)
	input.DosageForm = strings.TrimSpace(input.DosageForm)
	input.Specification = strings.TrimSpace(input.Specification)
	input.ApprovalNumber = strings.ToUpper(strings.TrimSpace(input.ApprovalNumber))
	input.Barcode = strings.TrimSpace(input.Barcode)
	input.StorageCondition = strings.TrimSpace(input.StorageCondition)
	if input.Code == "" || len(input.Code) > 64 || input.SKU == "" || len(input.SKU) > 64 || input.Name == "" || len(input.Name) > 255 || input.GenericName == "" || input.CategoryID == "" || input.UnitID == "" || input.ManufacturerID == "" || input.DosageForm == "" || input.Specification == "" || input.ApprovalNumber == "" || input.StorageCondition == "" || (update && input.Version < 1) {
		return newHTTPError(http.StatusBadRequest, "invalid_product", "complete product identity, catalog references, pharmaceutical attributes, and current version are required")
	}
	if input.TemperatureMin < -80 || input.TemperatureMax > 80 || input.TemperatureMin > input.TemperatureMax {
		return newHTTPError(http.StatusBadRequest, "invalid_temperature", "storage temperature range must be ordered within -80 to 80 Celsius")
	}
	return nil
}

func catalogPermission(kind, action string) pluginsdk.Permission {
	return pluginsdk.Permission{Resource: "pharma_oa." + kind, Action: action}
}

func productPermission(action string) pluginsdk.Permission {
	return pluginsdk.Permission{Resource: "pharma_oa.product", Action: action}
}

func catalogIntent(kind, action string, scope employeeScope) pluginsdk.DataScopeIntent {
	return pluginsdk.DataScopeIntent{Permission: catalogPermission(kind, action), Filter: scopeFilter(scope)}
}

func productIntent(action string, scope employeeScope) pluginsdk.DataScopeIntent {
	return pluginsdk.DataScopeIntent{Permission: productPermission(action), Filter: scopeFilter(scope)}
}

func scopeFilter(scope employeeScope) pluginsdk.ScopeFilter {
	return pluginsdk.ScopeFilter{TenantIDs: []string{scope.TenantID}, OrganizationIDs: []string{scope.OrganizationID}, OwnerIDs: []string{scope.OwnerID}}
}

func catalogValues(item catalogItem) map[string]pluginsdk.DataValue {
	return map[string]pluginsdk.DataValue{"catalog_type": stringValue(item.Type), "code": stringValue(item.Code), "name": stringValue(item.Name), "description": stringValue(item.Description), "parent_id": nullableStringValue(item.ParentID), "symbol": nullableStringValue(item.Symbol), "decimal_places": integerValue(item.DecimalPlaces), "unified_social_credit_code": nullableStringValue(item.UnifiedSocialCreditCode), "license_number": nullableStringValue(item.LicenseNumber), "status": stringValue(item.Status), "disable_reason": nullableStringValue(item.DisableReason)}
}

func productValues(item product) map[string]pluginsdk.DataValue {
	return map[string]pluginsdk.DataValue{"code": stringValue(item.Code), "sku": stringValue(item.SKU), "name": stringValue(item.Name), "generic_name": stringValue(item.GenericName), "category_id": stringValue(item.CategoryID), "unit_id": stringValue(item.UnitID), "manufacturer_id": stringValue(item.ManufacturerID), "dosage_form": stringValue(item.DosageForm), "specification": stringValue(item.Specification), "approval_number": stringValue(item.ApprovalNumber), "barcode": nullableStringValue(item.Barcode), "storage_condition": stringValue(item.StorageCondition), "temperature_min": integerValue(item.TemperatureMin), "temperature_max": integerValue(item.TemperatureMax), "status": stringValue(item.Status), "disable_reason": nullableStringValue(item.DisableReason)}
}

func integerValue(value int64) pluginsdk.DataValue {
	return pluginsdk.DataValue{Type: pluginsdk.DataValueInteger, Value: strconv.FormatInt(value, 10)}
}

func resourceID(prefix, generated string) string {
	return prefix + "-" + strings.TrimPrefix(generated, "employee-")
}

func catalogFromMutation(result pluginsdk.DataMutationResult) (catalogItem, error) {
	if result.Record == nil || result.RowsAffected != 1 {
		return catalogItem{}, fmt.Errorf("catalog mutation returned no record")
	}
	return catalogFromRecord(*result.Record)
}

func productFromMutation(result pluginsdk.DataMutationResult) (product, error) {
	if result.Record == nil || result.RowsAffected != 1 {
		return product{}, fmt.Errorf("product mutation returned no record")
	}
	return productFromRecord(*result.Record)
}

func catalogFromRecord(record pluginsdk.DataRecord) (catalogItem, error) {
	decimalPlaces, err := parseRecordInteger(record, "decimal_places")
	if err != nil {
		return catalogItem{}, err
	}
	return catalogItem{ID: dataString(record, "id"), Type: dataString(record, "catalog_type"), Code: dataString(record, "code"), Name: dataString(record, "name"), Description: dataString(record, "description"), ParentID: dataString(record, "parent_id"), Symbol: dataString(record, "symbol"), DecimalPlaces: decimalPlaces, UnifiedSocialCreditCode: dataString(record, "unified_social_credit_code"), LicenseNumber: dataString(record, "license_number"), Status: dataString(record, "status"), DisableReason: dataString(record, "disable_reason"), Version: record.Version, CreatedAt: dataString(record, "created_at"), UpdatedAt: dataString(record, "updated_at"), scope: recordScope(record)}, nil
}

func productFromRecord(record pluginsdk.DataRecord) (product, error) {
	minimum, err := parseRecordInteger(record, "temperature_min")
	if err != nil {
		return product{}, err
	}
	maximum, err := parseRecordInteger(record, "temperature_max")
	if err != nil {
		return product{}, err
	}
	return product{ID: dataString(record, "id"), Code: dataString(record, "code"), SKU: dataString(record, "sku"), Name: dataString(record, "name"), GenericName: dataString(record, "generic_name"), CategoryID: dataString(record, "category_id"), UnitID: dataString(record, "unit_id"), ManufacturerID: dataString(record, "manufacturer_id"), DosageForm: dataString(record, "dosage_form"), Specification: dataString(record, "specification"), ApprovalNumber: dataString(record, "approval_number"), Barcode: dataString(record, "barcode"), StorageCondition: dataString(record, "storage_condition"), TemperatureMin: minimum, TemperatureMax: maximum, Status: dataString(record, "status"), DisableReason: dataString(record, "disable_reason"), Version: record.Version, CreatedAt: dataString(record, "created_at"), UpdatedAt: dataString(record, "updated_at"), scope: recordScope(record)}, nil
}

func parseRecordInteger(record pluginsdk.DataRecord, field string) (int64, error) {
	raw := dataString(record, field)
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("decode %s: %w", field, err)
	}
	return value, nil
}

func recordScope(record pluginsdk.DataRecord) employeeScope {
	return employeeScope{TenantID: dataString(record, "tenant_id"), OrganizationID: dataString(record, "organization_id"), OwnerID: dataString(record, "owner_id")}
}

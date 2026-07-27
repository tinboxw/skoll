package datastore

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/repository"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MutationExecutor struct {
	db       *gorm.DB
	uow      repository.UnitOfWork
	registry *SchemaRegistry
	scopes   pluginsdk.DataScopeService
	audit    pluginsdk.AuditService
	dialect  SQLDialect
	now      func() time.Time
}

type mutationScope struct {
	tenantID       string
	organizationID string
	ownerID        string
}

func NewMutationExecutor(db *gorm.DB, uow repository.UnitOfWork, registry *SchemaRegistry, scopes pluginsdk.DataScopeService, audit pluginsdk.AuditService, dialect SQLDialect) (*MutationExecutor, error) {
	if db == nil || uow == nil {
		return nil, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "database", "plugin datastore database is unavailable", true)
	}
	if registry == nil || scopes == nil || audit == nil {
		return nil, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "services", "plugin datastore services are incomplete", true)
	}
	parsed, err := ParseSQLDialect(string(dialect))
	if err != nil {
		return nil, err
	}
	return &MutationExecutor{db: db, uow: uow, registry: registry, scopes: scopes, audit: audit, dialect: parsed, now: time.Now}, nil
}

func (e *MutationExecutor) Mutate(ctx context.Context, pluginID string, mutation pluginsdk.DataMutation) (pluginsdk.DataMutationResult, error) {
	if e == nil || e.db == nil || e.uow == nil || e.registry == nil || e.scopes == nil || e.audit == nil {
		return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "executor", "mutation executor is unavailable", true)
	}
	if ctx == nil {
		return pluginsdk.DataMutationResult{}, invalidSchema("context", "mutation context is required")
	}
	if err := mutation.Validate(); err != nil {
		return pluginsdk.DataMutationResult{}, err
	}
	pluginID = strings.TrimSpace(pluginID)
	if err := validateQueryPermission(pluginID, mutation.Scope.Permission); err != nil {
		return pluginsdk.DataMutationResult{}, err
	}
	table, err := e.registry.Resolve(pluginID, mutation.Table)
	if err != nil {
		return pluginsdk.DataMutationResult{}, err
	}
	if err = validateMutationSchema(table, mutation); err != nil {
		return pluginsdk.DataMutationResult{}, err
	}
	trusted, err := e.scopes.Resolve(ctx, mutation.Scope.Permission)
	if err != nil {
		return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorForbidden, "scope.permission", "trusted data scope could not be resolved", false)
	}
	trusted = trusted.Constrain(mutation.Scope.Filter)
	if err = validateTrustedScope(trusted); err != nil {
		return pluginsdk.DataMutationResult{}, err
	}
	scope, err := exactMutationScope(trusted)
	if err != nil {
		return pluginsdk.DataMutationResult{}, err
	}
	requestHash, err := mutationRequestHash(pluginID, mutation)
	if err != nil {
		return pluginsdk.DataMutationResult{}, err
	}

	var result pluginsdk.DataMutationResult
	err = e.uow.Do(ctx, func(tx repository.Tx) error {
		db := storesql.ResolveDB(tx.Context(), e.db)
		if db == nil {
			return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "database", "transaction database is unavailable", true)
		}
		now := e.now().UTC()
		replayed, stored, reserveErr := reserveMutation(db, pluginID, mutation.IdempotencyKey, requestHash, now)
		if reserveErr != nil {
			return reserveErr
		}
		if replayed {
			result = stored
			return nil
		}

		result, err = e.executeMutation(db, table, scope, mutation, now)
		if err != nil {
			return err
		}
		if _, err = e.audit.Record(tx.Context(), mutationAuditEntry(table, mutation, result)); err != nil {
			return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "audit", "mutation audit could not be recorded", true)
		}
		return completeMutation(db, pluginID, mutation.IdempotencyKey, requestHash, result, now)
	})
	if err != nil {
		var storeErr *pluginsdk.DataStoreError
		if errors.As(err, &storeErr) {
			return pluginsdk.DataMutationResult{}, storeErr
		}
		return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "database", "plugin datastore mutation failed", true)
	}
	return result, nil
}

func (e *MutationExecutor) executeMutation(db *gorm.DB, table ResolvedTable, scope mutationScope, mutation pluginsdk.DataMutation, now time.Time) (pluginsdk.DataMutationResult, error) {
	switch mutation.Operation {
	case pluginsdk.DataMutationInsert:
		return e.insert(db, table, scope, mutation, now)
	case pluginsdk.DataMutationUpdate:
		return e.update(db, table, scope, mutation, now)
	case pluginsdk.DataMutationUpsert:
		current, exists, err := loadMutationRecord(db, e.dialect, table, scope, mutation.Key, nil)
		if err != nil {
			return pluginsdk.DataMutationResult{}, err
		}
		if exists {
			return e.updateCurrent(db, table, scope, mutation, current, now)
		}
		if mutation.ExpectedVersion != nil {
			return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorConflict, "expectedVersion", "upsert target does not exist at the expected version", false)
		}
		return e.insert(db, table, scope, mutation, now)
	case pluginsdk.DataMutationDelete:
		return e.delete(db, table, scope, mutation)
	default:
		return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnsupported, "operation", "mutation operation is unsupported", false)
	}
}

func (e *MutationExecutor) insert(db *gorm.DB, table ResolvedTable, scope mutationScope, mutation pluginsdk.DataMutation, now time.Time) (pluginsdk.DataMutationResult, error) {
	if _, exists, err := loadMutationRecord(db, e.dialect, table, scope, mutation.Key, nil); err != nil {
		return pluginsdk.DataMutationResult{}, err
	} else if exists {
		return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorConflict, "key", "record already exists", false)
	}
	row := map[string]any{
		FieldTenantID: scope.tenantID, FieldOrganizationID: scope.organizationID, FieldOwnerID: scope.ownerID,
		FieldVersion: int64(1), FieldCreatedAt: now, FieldUpdatedAt: now,
	}
	for _, field := range table.PrimaryKey {
		converted, err := convertMutationValue(table.Fields[field], mutation.Key[field], "key."+field)
		if err != nil {
			return pluginsdk.DataMutationResult{}, err
		}
		row[field] = converted
	}
	for _, field := range sortedMutationKeys(mutation.Values) {
		converted, err := convertMutationValue(table.Fields[field], mutation.Values[field], "values."+field)
		if err != nil {
			return pluginsdk.DataMutationResult{}, err
		}
		row[field] = converted
	}
	if err := db.Table(table.PhysicalName).Create(&row).Error; err != nil {
		return pluginsdk.DataMutationResult{}, mutationDatabaseError("key", "record could not be inserted", err)
	}
	return mutationResult(db, e.dialect, table, scope, mutation.Key, mutation.Returning, 1)
}

func (e *MutationExecutor) update(db *gorm.DB, table ResolvedTable, scope mutationScope, mutation pluginsdk.DataMutation, now time.Time) (pluginsdk.DataMutationResult, error) {
	current, exists, err := loadMutationRecord(db, e.dialect, table, scope, mutation.Key, nil)
	if err != nil {
		return pluginsdk.DataMutationResult{}, err
	}
	if !exists {
		return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorNotFound, "key", "record does not exist in the trusted scope", false)
	}
	return e.updateCurrent(db, table, scope, mutation, current, now)
}

func (e *MutationExecutor) updateCurrent(db *gorm.DB, table ResolvedTable, scope mutationScope, mutation pluginsdk.DataMutation, current map[string]any, now time.Time) (pluginsdk.DataMutationResult, error) {
	version, err := databaseVersion(current[FieldVersion])
	if err != nil {
		return pluginsdk.DataMutationResult{}, err
	}
	if mutation.ExpectedVersion != nil && *mutation.ExpectedVersion != version {
		return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorConflict, "expectedVersion", "record version is stale", false)
	}
	updates := map[string]any{FieldUpdatedAt: now, FieldVersion: gorm.Expr(quoteSQLIdentifier(e.dialect, FieldVersion) + " + 1")}
	for _, field := range sortedMutationKeys(mutation.Values) {
		converted, convertErr := convertMutationValue(table.Fields[field], mutation.Values[field], "values."+field)
		if convertErr != nil {
			return pluginsdk.DataMutationResult{}, convertErr
		}
		updates[field] = converted
	}
	where, args := mutationWhere(e.dialect, table, scope, mutation.Key, &version)
	operation := db.Table(table.PhysicalName).Where(where, args...).Updates(updates)
	if operation.Error != nil {
		return pluginsdk.DataMutationResult{}, mutationDatabaseError("values", "record could not be updated", operation.Error)
	}
	if operation.RowsAffected != 1 {
		return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorConflict, "expectedVersion", "record changed concurrently", false)
	}
	return mutationResult(db, e.dialect, table, scope, mutation.Key, mutation.Returning, version+1)
}

func (e *MutationExecutor) delete(db *gorm.DB, table ResolvedTable, scope mutationScope, mutation pluginsdk.DataMutation) (pluginsdk.DataMutationResult, error) {
	current, exists, err := loadMutationRecord(db, e.dialect, table, scope, mutation.Key, mutation.Returning)
	if err != nil {
		return pluginsdk.DataMutationResult{}, err
	}
	if !exists {
		return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorNotFound, "key", "record does not exist in the trusted scope", false)
	}
	version, err := databaseVersion(current[FieldVersion])
	if err != nil {
		return pluginsdk.DataMutationResult{}, err
	}
	if mutation.ExpectedVersion != nil && *mutation.ExpectedVersion != version {
		return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorConflict, "expectedVersion", "record version is stale", false)
	}
	where, args := mutationWhere(e.dialect, table, scope, mutation.Key, &version)
	operation := db.Table(table.PhysicalName).Where(where, args...).Delete(&map[string]any{})
	if operation.Error != nil {
		return pluginsdk.DataMutationResult{}, mutationDatabaseError("key", "record could not be deleted", operation.Error)
	}
	if operation.RowsAffected != 1 {
		return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorConflict, "expectedVersion", "record changed concurrently", false)
	}
	result := pluginsdk.DataMutationResult{RowsAffected: 1}
	if len(mutation.Returning) > 0 {
		record, convertErr := mutationRecord(table, current, mutation.Returning, version)
		if convertErr != nil {
			return pluginsdk.DataMutationResult{}, convertErr
		}
		result.Record = &record
	}
	return result, nil
}

func validateMutationSchema(table ResolvedTable, mutation pluginsdk.DataMutation) error {
	if table.MutationPolicy == TableMutationAppendOnly && mutation.Operation != pluginsdk.DataMutationInsert {
		return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnsupported, "operation", "append-only table accepts insert mutations only", false)
	}
	if len(mutation.Key) != len(table.PrimaryKey) {
		return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, "key", "key must match the declared primary key", false)
	}
	for _, field := range table.PrimaryKey {
		value, exists := mutation.Key[field]
		if !exists {
			return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, "key."+field, "primary key field is missing", false)
		}
		if err := validateMutationValue(table.Fields[field], value, "key."+field); err != nil {
			return err
		}
	}
	for _, field := range sortedMutationKeys(mutation.Key) {
		if !containsField(table.PrimaryKey, field) {
			return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, "key."+field, "field is not part of the declared primary key", false)
		}
	}
	for _, field := range sortedMutationKeys(mutation.Values) {
		schema, exists := table.Fields[field]
		if !exists || schema.HostManaged {
			return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, "values."+field, "field is not plugin managed", false)
		}
		if !schema.Mutable {
			return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, "values."+field, "field is immutable", false)
		}
		if err := validateMutationValue(schema, mutation.Values[field], "values."+field); err != nil {
			return err
		}
	}
	for index, field := range mutation.Returning {
		if _, exists := table.Fields[field]; !exists {
			return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, fmt.Sprintf("returning[%d]", index), "returning field is not declared by the plugin schema", false)
		}
	}
	return nil
}

func exactMutationScope(scope pluginsdk.ScopePredicate) (mutationScope, error) {
	tenantID, err := exactScopeID(scope.TenantIDs(), scope.AllTenants(), "scope.filter.tenantIds")
	if err != nil {
		return mutationScope{}, err
	}
	organizationID, err := exactScopeID(scope.OrganizationIDs(), scope.AllOrganizations(), "scope.filter.organizationIds")
	if err != nil {
		return mutationScope{}, err
	}
	ownerID, err := exactScopeID(scope.OwnerIDs(), scope.AllOwners(), "scope.filter.ownerIds")
	if err != nil {
		return mutationScope{}, err
	}
	return mutationScope{tenantID: tenantID, organizationID: organizationID, ownerID: ownerID}, nil
}

func exactScopeID(ids []string, all bool, field string) (string, error) {
	if all || len(ids) != 1 {
		return "", pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, field, "mutation requires exactly one trusted scope identifier", false)
	}
	return ids[0], nil
}

func mutationWhere(dialect SQLDialect, table ResolvedTable, scope mutationScope, key map[string]pluginsdk.DataValue, version *int64) (string, []any) {
	parts := []string{
		quoteSQLIdentifier(dialect, FieldTenantID) + " = ?",
		quoteSQLIdentifier(dialect, FieldOrganizationID) + " = ?",
		quoteSQLIdentifier(dialect, FieldOwnerID) + " = ?",
	}
	args := []any{scope.tenantID, scope.organizationID, scope.ownerID}
	for _, field := range table.PrimaryKey {
		value, _ := convertMutationValue(table.Fields[field], key[field], "key."+field)
		parts = append(parts, quoteSQLIdentifier(dialect, field)+" = ?")
		args = append(args, value)
	}
	if version != nil {
		parts = append(parts, quoteSQLIdentifier(dialect, FieldVersion)+" = ?")
		args = append(args, *version)
	}
	return strings.Join(parts, " AND "), args
}

func loadMutationRecord(db *gorm.DB, dialect SQLDialect, table ResolvedTable, scope mutationScope, key map[string]pluginsdk.DataValue, returning []string) (map[string]any, bool, error) {
	fields := appendUniqueFields(returning, FieldVersion)
	quoted := make([]string, len(fields))
	for index, field := range fields {
		quoted[index] = quoteSQLIdentifier(dialect, field)
	}
	where, args := mutationWhere(dialect, table, scope, key, nil)
	row := map[string]any{}
	err := db.Table(table.PhysicalName).Select(strings.Join(quoted, ", ")).Where(where, args...).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, mutationDatabaseError("key", "record could not be loaded", err)
	}
	return row, true, nil
}

func mutationResult(db *gorm.DB, dialect SQLDialect, table ResolvedTable, scope mutationScope, key map[string]pluginsdk.DataValue, returning []string, version int64) (pluginsdk.DataMutationResult, error) {
	result := pluginsdk.DataMutationResult{RowsAffected: 1}
	if len(returning) == 0 {
		return result, nil
	}
	row, exists, err := loadMutationRecord(db, dialect, table, scope, key, returning)
	if err != nil {
		return pluginsdk.DataMutationResult{}, err
	}
	if !exists {
		return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorConflict, "key", "mutated record could not be reloaded", false)
	}
	record, err := mutationRecord(table, row, returning, version)
	if err != nil {
		return pluginsdk.DataMutationResult{}, err
	}
	result.Record = &record
	return result, nil
}

func mutationRecord(table ResolvedTable, row map[string]any, fields []string, version int64) (pluginsdk.DataRecord, error) {
	record := pluginsdk.DataRecord{Values: make(map[string]pluginsdk.DataValue, len(fields)), Version: version}
	for _, field := range fields {
		value, exists := row[field]
		if !exists {
			return pluginsdk.DataRecord{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "returning."+field, "database did not return the requested field", true)
		}
		converted, err := databaseDataValue(table.Fields[field], value, "returning."+field)
		if err != nil {
			return pluginsdk.DataRecord{}, err
		}
		record.Values[field] = converted
	}
	return record, nil
}

func reserveMutation(db *gorm.DB, pluginID, key, requestHash string, now time.Time) (bool, pluginsdk.DataMutationResult, error) {
	row := gormrepo.PluginDataMutationModel{PluginID: pluginID, IdempotencyKey: key, RequestHash: requestHash, ResultJSON: "", CreatedAt: now, UpdatedAt: now}
	created := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
	if created.Error != nil {
		return false, pluginsdk.DataMutationResult{}, mutationDatabaseError("idempotencyKey", "idempotency reservation failed", created.Error)
	}
	if created.RowsAffected == 1 {
		return false, pluginsdk.DataMutationResult{}, nil
	}
	var stored gormrepo.PluginDataMutationModel
	if err := db.Where("plugin_id = ? AND idempotency_key = ?", pluginID, key).Take(&stored).Error; err != nil {
		return false, pluginsdk.DataMutationResult{}, mutationDatabaseError("idempotencyKey", "idempotency result could not be loaded", err)
	}
	if stored.RequestHash != requestHash {
		return false, pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorConflict, "idempotencyKey", "idempotency key was used for a different mutation", false)
	}
	if stored.ResultJSON == "" {
		return false, pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "idempotencyKey", "idempotency result is incomplete", true)
	}
	var result pluginsdk.DataMutationResult
	if err := json.Unmarshal([]byte(stored.ResultJSON), &result); err != nil || result.Validate() != nil {
		return false, pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "idempotencyKey", "idempotency result is invalid", true)
	}
	return true, result, nil
}

func completeMutation(db *gorm.DB, pluginID, key, requestHash string, result pluginsdk.DataMutationResult, now time.Time) error {
	raw, err := json.Marshal(result)
	if err != nil {
		return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "idempotencyKey", "idempotency result could not be encoded", true)
	}
	operation := db.Model(&gormrepo.PluginDataMutationModel{}).
		Where("plugin_id = ? AND idempotency_key = ? AND request_hash = ?", pluginID, key, requestHash).
		Updates(map[string]any{"result_json": string(raw), "updated_at": now})
	if operation.Error != nil || operation.RowsAffected != 1 {
		return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "idempotencyKey", "idempotency result could not be committed", true)
	}
	return nil
}

func mutationRequestHash(pluginID string, mutation pluginsdk.DataMutation) (string, error) {
	canonical := mutation
	canonical.Scope.Filter.TenantIDs = sortedScopeIDs(mutation.Scope.Filter.TenantIDs)
	canonical.Scope.Filter.OrganizationIDs = sortedScopeIDs(mutation.Scope.Filter.OrganizationIDs)
	canonical.Scope.Filter.OwnerIDs = sortedScopeIDs(mutation.Scope.Filter.OwnerIDs)
	raw, err := json.Marshal(struct {
		PluginID string                 `json:"pluginId"`
		Mutation pluginsdk.DataMutation `json:"mutation"`
	}{PluginID: pluginID, Mutation: canonical})
	if err != nil {
		return "", pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, "mutation", "mutation could not be canonicalized", false)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func mutationAuditEntry(table ResolvedTable, mutation pluginsdk.DataMutation, result pluginsdk.DataMutationResult) pluginsdk.AuditEntry {
	risk := pluginsdk.AuditRiskMedium
	if mutation.Operation == pluginsdk.DataMutationDelete {
		risk = pluginsdk.AuditRiskHigh
	}
	detail := map[string]any{
		"table": table.LogicalName, "operation": string(mutation.Operation), "rowsAffected": result.RowsAffected,
		"idempotencyHash": shortHash(mutation.IdempotencyKey),
	}
	if result.Record != nil {
		detail["version"] = result.Record.Version
	}
	return pluginsdk.AuditEntry{
		Action: "datastore." + string(mutation.Operation), Resource: "data." + table.LogicalName,
		ResourceID: mutationResourceID(mutation.Key), Result: pluginsdk.AuditResultSuccess, Risk: risk, Detail: detail,
	}
}

func mutationResourceID(key map[string]pluginsdk.DataValue) string {
	raw, _ := json.Marshal(key)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:8])
}

func shortHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:8])
}

func validateMutationValue(field FieldSchema, value pluginsdk.DataValue, path string) error {
	_, err := convertMutationValue(field, value, path)
	return err
}

func convertMutationValue(field FieldSchema, value pluginsdk.DataValue, path string) (any, error) {
	return convertQueryValue(field, value, path, true)
}

func databaseVersion(value any) (int64, error) {
	converted, err := databaseDataValue(FieldSchema{Type: pluginsdk.DataValueInteger}, value, FieldVersion)
	if err != nil {
		return 0, err
	}
	version, err := strconv.ParseInt(converted.Value, 10, 64)
	if err != nil || version < 1 {
		return 0, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, FieldVersion, "database version is invalid", true)
	}
	return version, nil
}

func databaseDataValue(field FieldSchema, value any, path string) (pluginsdk.DataValue, error) {
	if value == nil {
		if !field.Nullable {
			return pluginsdk.DataValue{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, path, "database returned null for a required field", true)
		}
		return pluginsdk.DataValue{Type: pluginsdk.DataValueNull}, nil
	}
	var text string
	switch field.Type {
	case pluginsdk.DataValueString, pluginsdk.DataValueDecimal, pluginsdk.DataValueJSON:
		switch typed := value.(type) {
		case string:
			text = typed
		case []byte:
			text = string(typed)
		default:
			text = fmt.Sprint(typed)
		}
	case pluginsdk.DataValueInteger:
		var err error
		text, err = databaseInteger(value)
		if err != nil {
			return pluginsdk.DataValue{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, path, "database integer is invalid", true)
		}
	case pluginsdk.DataValueBoolean:
		var err error
		text, err = databaseBoolean(value)
		if err != nil {
			return pluginsdk.DataValue{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, path, "database boolean is invalid", true)
		}
	case pluginsdk.DataValueTimestamp:
		var err error
		text, err = databaseTimestamp(value)
		if err != nil {
			return pluginsdk.DataValue{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, path, "database timestamp is invalid", true)
		}
	case pluginsdk.DataValueBytes:
		switch typed := value.(type) {
		case []byte:
			text = base64.StdEncoding.EncodeToString(typed)
		case string:
			text = base64.StdEncoding.EncodeToString([]byte(typed))
		default:
			return pluginsdk.DataValue{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, path, "database bytes are invalid", true)
		}
	default:
		return pluginsdk.DataValue{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnsupported, path, "database value type is unsupported", false)
	}
	converted := pluginsdk.DataValue{Type: field.Type, Value: text}
	if err := converted.Validate(); err != nil {
		return pluginsdk.DataValue{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, path, "database value violates the schema", true)
	}
	return converted, nil
}

func databaseInteger(value any) (string, error) {
	switch typed := value.(type) {
	case int:
		return strconv.Itoa(typed), nil
	case int8, int16, int32, int64:
		return fmt.Sprintf("%d", typed), nil
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", typed), nil
	case float64:
		if typed != math.Trunc(typed) || typed > math.MaxInt64 || typed < math.MinInt64 {
			return "", fmt.Errorf("not an int64")
		}
		return strconv.FormatInt(int64(typed), 10), nil
	case string:
		parsed, err := strconv.ParseInt(typed, 10, 64)
		if err != nil {
			return "", err
		}
		return strconv.FormatInt(parsed, 10), nil
	case []byte:
		return databaseInteger(string(typed))
	default:
		return "", fmt.Errorf("unsupported integer %T", value)
	}
}

func databaseBoolean(value any) (string, error) {
	switch typed := value.(type) {
	case bool:
		return strconv.FormatBool(typed), nil
	case int64:
		if typed == 0 || typed == 1 {
			return strconv.FormatBool(typed == 1), nil
		}
	case string:
		if typed == "1" || strings.EqualFold(typed, "true") {
			return "true", nil
		}
		if typed == "0" || strings.EqualFold(typed, "false") {
			return "false", nil
		}
	case []byte:
		return databaseBoolean(string(typed))
	}
	return "", fmt.Errorf("unsupported boolean %T", value)
}

func databaseTimestamp(value any) (string, error) {
	switch typed := value.(type) {
	case time.Time:
		return typed.UTC().Format(time.RFC3339Nano), nil
	case string:
		parsed, err := time.Parse(time.RFC3339Nano, typed)
		if err != nil {
			for _, layout := range []string{"2006-01-02 15:04:05.999999999Z07:00", "2006-01-02 15:04:05.999999999"} {
				if parsed, err = time.Parse(layout, typed); err == nil {
					return parsed.UTC().Format(time.RFC3339Nano), nil
				}
			}
			return "", err
		}
		return parsed.UTC().Format(time.RFC3339Nano), nil
	case []byte:
		return databaseTimestamp(string(typed))
	default:
		return "", fmt.Errorf("unsupported timestamp %T", value)
	}
}

func mutationDatabaseError(field, message string, err error) error {
	if err == nil {
		return nil
	}
	lower := strings.ToLower(err.Error())
	if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(lower, "duplicate") || strings.Contains(lower, "unique constraint") || strings.Contains(lower, "violates unique") {
		return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorConflict, field, message, false)
	}
	return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, field, message, true)
}

func sortedMutationKeys(values map[string]pluginsdk.DataValue) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedScopeIDs(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func containsField(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

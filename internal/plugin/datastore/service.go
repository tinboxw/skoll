package datastore

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/tinboxw/skoll/internal/repository"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"gorm.io/gorm"
)

// Service binds the public datastore port to one plugin identity.
type Service struct {
	pluginID   string
	db         *gorm.DB
	planner    *QueryPlanner
	aggregator *AggregatePlanner
	mutator    *MutationExecutor
}

func NewService(db *gorm.DB, uow repository.UnitOfWork, registry *SchemaRegistry, scopes pluginsdk.DataScopeService, audit pluginsdk.AuditService, dialect SQLDialect, pluginID string) (*Service, error) {
	planner, err := NewQueryPlanner(registry, scopes, dialect)
	if err != nil {
		return nil, err
	}
	aggregator, err := NewAggregatePlanner(registry, scopes, dialect)
	if err != nil {
		return nil, err
	}
	mutator, err := NewMutationExecutor(db, uow, registry, scopes, audit, dialect)
	if err != nil {
		return nil, err
	}
	if _, err = PluginNamespace(pluginID); err != nil {
		return nil, err
	}
	return &Service{pluginID: pluginID, db: db, planner: planner, aggregator: aggregator, mutator: mutator}, nil
}

func (s *Service) Query(ctx context.Context, query pluginsdk.DataQuery) (pluginsdk.DataPage, error) {
	if s == nil || s.db == nil || s.planner == nil {
		return pluginsdk.DataPage{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "service", "plugin datastore service is unavailable", true)
	}
	plan, err := s.planner.Plan(ctx, s.pluginID, query)
	if err != nil {
		return pluginsdk.DataPage{}, err
	}
	db := storesql.ResolveDB(ctx, s.db)
	if db == nil {
		return pluginsdk.DataPage{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "database", "plugin datastore database is unavailable", true)
	}
	rows, err := db.WithContext(ctx).Raw(plan.SQL, plan.Args...).Rows()
	if err != nil {
		return pluginsdk.DataPage{}, queryDatabaseError(err)
	}
	defer rows.Close()

	typedRows := make([]map[string]pluginsdk.DataValue, 0, plan.FetchLimit)
	for rows.Next() {
		raw := make([]any, len(plan.ScanFields))
		destinations := make([]any, len(raw))
		for index := range raw {
			destinations[index] = &raw[index]
		}
		if err = rows.Scan(destinations...); err != nil {
			return pluginsdk.DataPage{}, queryDatabaseError(err)
		}
		values := make(map[string]pluginsdk.DataValue, len(plan.ScanFields))
		for index, field := range plan.ScanFields {
			converted, convertErr := databaseDataValue(plan.table.Fields[field], raw[index], "records."+field)
			if convertErr != nil {
				return pluginsdk.DataPage{}, convertErr
			}
			values[field] = converted
		}
		typedRows = append(typedRows, values)
	}
	if err = rows.Err(); err != nil {
		return pluginsdk.DataPage{}, queryDatabaseError(err)
	}

	hasMore := len(typedRows) > plan.Limit
	if hasMore {
		typedRows = typedRows[:plan.Limit]
	}
	page := pluginsdk.DataPage{Records: make([]pluginsdk.DataRecord, 0, len(typedRows)), HasMore: hasMore}
	for _, values := range typedRows {
		versionValue := values[FieldVersion]
		version, versionErr := strconv.ParseInt(versionValue.Value, 10, 64)
		if versionErr != nil || version < 1 {
			return pluginsdk.DataPage{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, FieldVersion, "database version is invalid", true)
		}
		record := pluginsdk.DataRecord{Values: make(map[string]pluginsdk.DataValue, len(plan.Fields)), Version: version}
		for _, field := range plan.Fields {
			record.Values[field] = values[field]
		}
		page.Records = append(page.Records, record)
	}
	if hasMore {
		page.NextCursor, err = plan.EncodeCursor(typedRows[len(typedRows)-1])
		if err != nil {
			return pluginsdk.DataPage{}, err
		}
	}
	if err = page.Validate(); err != nil {
		return pluginsdk.DataPage{}, err
	}
	return page, nil
}

func (s *Service) Mutate(ctx context.Context, mutation pluginsdk.DataMutation) (pluginsdk.DataMutationResult, error) {
	if s == nil || s.mutator == nil {
		return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "service", "plugin datastore service is unavailable", true)
	}
	return s.mutator.Mutate(ctx, s.pluginID, mutation)
}

func (s *Service) Aggregate(ctx context.Context, query pluginsdk.DataAggregateQuery) (pluginsdk.DataAggregatePage, error) {
	if s == nil || s.db == nil || s.aggregator == nil {
		return pluginsdk.DataAggregatePage{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "service", "plugin aggregate service is unavailable", true)
	}
	plan, err := s.aggregator.Plan(ctx, s.pluginID, query)
	if err != nil {
		return pluginsdk.DataAggregatePage{}, err
	}
	db := storesql.ResolveDB(ctx, s.db)
	if db == nil {
		return pluginsdk.DataAggregatePage{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "database", "plugin datastore database is unavailable", true)
	}
	rows, err := db.WithContext(ctx).Raw(plan.SQL, plan.Args...).Rows()
	if err != nil {
		return pluginsdk.DataAggregatePage{}, queryDatabaseError(err)
	}
	defer rows.Close()

	page := pluginsdk.DataAggregatePage{
		Metrics: append([]pluginsdk.DataAggregateMetric(nil), plan.Metrics...),
		GroupBy: append([]string(nil), plan.GroupBy...),
		Rows:    make([]pluginsdk.DataAggregateRow, 0, plan.FetchLimit),
	}
	for rows.Next() {
		raw := make([]any, len(plan.GroupBy)+len(plan.Metrics))
		destinations := make([]any, len(raw))
		for index := range raw {
			destinations[index] = &raw[index]
		}
		if err = rows.Scan(destinations...); err != nil {
			return pluginsdk.DataAggregatePage{}, queryDatabaseError(err)
		}
		row := pluginsdk.DataAggregateRow{
			Group:  make(map[string]pluginsdk.DataValue, len(plan.GroupBy)),
			Values: make([]pluginsdk.DataValue, len(plan.Metrics)),
		}
		for index, field := range plan.GroupBy {
			value, convertErr := databaseDataValue(plan.table.Fields[field], raw[index], fmt.Sprintf("rows.group.%s", field))
			if convertErr != nil {
				return pluginsdk.DataAggregatePage{}, convertErr
			}
			row.Group[field] = value
		}
		for index, metric := range plan.Metrics {
			schema := FieldSchema{Type: pluginsdk.DataValueInteger}
			if metric.Operation != pluginsdk.DataAggregateCount {
				schema = plan.table.Fields[metric.Field]
				schema.Nullable = true
			}
			value, convertErr := databaseDataValue(schema, raw[len(plan.GroupBy)+index], fmt.Sprintf("rows.values[%d]", index))
			if convertErr != nil {
				return pluginsdk.DataAggregatePage{}, convertErr
			}
			row.Values[index] = value
		}
		page.Rows = append(page.Rows, row)
	}
	if err = rows.Err(); err != nil {
		return pluginsdk.DataAggregatePage{}, queryDatabaseError(err)
	}
	if len(plan.GroupBy) > 0 && len(page.Rows) > plan.Limit {
		page.Rows = page.Rows[:plan.Limit]
		page.HasMore = true
		page.NextCursor, err = plan.EncodeCursor(page.Rows[len(page.Rows)-1].Group)
		if err != nil {
			return pluginsdk.DataAggregatePage{}, err
		}
	}
	if err = page.Validate(); err != nil {
		return pluginsdk.DataAggregatePage{}, err
	}
	return page, nil
}

func queryDatabaseError(err error) error {
	var datastoreErr *pluginsdk.DataStoreError
	if errors.As(err, &datastoreErr) {
		return err
	}
	return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "query", "plugin datastore query failed", true)
}

var _ pluginsdk.DataStoreService = (*Service)(nil)

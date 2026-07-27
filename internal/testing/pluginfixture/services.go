package pluginfixture

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const (
	OperationTransactionWithin = "transactions.within"
	OperationScopeResolve      = "scopes.resolve"
	OperationDataQuery         = "datastore.query"
	OperationDataMutate        = "datastore.mutate"
	OperationDataAggregate     = "datastore.aggregate"
	OperationEventPublish      = "events.publish"
	OperationNumberPreview     = "document_numbers.preview"
	OperationNumberIssue       = "document_numbers.issue"
	OperationFileStore         = "files.store"
	OperationFileList          = "files.list"
	OperationFileGet           = "files.get"
	OperationFileDownload      = "files.download"
	OperationFileDelete        = "files.delete"
	OperationAuditRecord       = "audit.record"
	OperationConfigGet         = "config.get"
	OperationConfigReplace     = "config.replace"
	OperationSecretGet         = "secrets.get"
	OperationSecretSet         = "secrets.set"
)

var ErrDocumentServiceNotConfigured = errors.New("fixture document service is not configured")

type ServicesOptions struct {
	Clock    *Clock
	Identity Identity
}

type Services struct {
	Clock           *Clock
	Identity        Identity
	Failures        *FailurePlan
	Transactions    *TransactionService
	DataScopes      *DataScopeService
	DataStore       *DataStoreService
	Events          *EventService
	DocumentNumbers *DocumentNumberService
	Documents       *DocumentService
	Files           *FileService
	Audit           *AuditService
	Config          *ConfigService
	Secrets         *SecretService
	Workflows       *WorkflowService
	Jobs            *JobService
}

type HostOverrides struct {
	Transactions    pluginsdk.TransactionService
	DataScopes      pluginsdk.DataScopeService
	DataStore       pluginsdk.DataStoreService
	Events          pluginsdk.EventService
	DocumentNumbers pluginsdk.DocumentNumberService
	Documents       pluginsdk.DocumentService
	Files           pluginsdk.FileService
	Audit           pluginsdk.AuditService
	Config          pluginsdk.ConfigService
	Secrets         pluginsdk.SecretService
	Workflows       pluginsdk.WorkflowService
	Jobs            pluginsdk.JobService
}

func NewServices(options ServicesOptions) (*Services, error) {
	if options.Clock == nil {
		options.Clock = NewClock(time.Now().UTC())
	}
	if strings.TrimSpace(options.Identity.Subject) == "" {
		options.Identity.Subject = "fixture-user"
	}
	if strings.TrimSpace(options.Identity.TenantID) == "" {
		options.Identity.TenantID = "fixture-tenant"
	}
	if strings.TrimSpace(options.Identity.OrganizationID) == "" {
		options.Identity.OrganizationID = "fixture-organization"
	}
	predicate, err := options.Identity.Scope()
	if err != nil {
		return nil, fmt.Errorf("build fixture identity scope: %w", err)
	}
	failures := NewFailurePlan()
	services := &Services{Clock: options.Clock, Identity: options.Identity, Failures: failures}
	services.Transactions = &TransactionService{Failures: failures}
	services.DataScopes = &DataScopeService{Predicate: predicate, Failures: failures}
	services.DataStore = &DataStoreService{Failures: failures}
	services.Events = &EventService{Clock: options.Clock, Failures: failures}
	services.DocumentNumbers = newDocumentNumberService(options.Clock, failures)
	services.Documents = &DocumentService{}
	services.Files = newFileService(options.Clock, failures)
	services.Audit = &AuditService{Clock: options.Clock, Failures: failures}
	services.Config = &ConfigService{Failures: failures, values: make(map[string]any)}
	services.Secrets = &SecretService{Failures: failures, values: make(map[string]string)}
	services.Workflows = newWorkflowService(options.Clock, options.Identity, failures)
	services.Jobs = newJobService(options.Clock, failures)
	return services, nil
}

func (s *Services) Host(pluginID string, overrides ...HostOverrides) (pluginsdk.HostServices, error) {
	if s == nil {
		return pluginsdk.HostServices{}, errors.New("fixture services are required")
	}
	host := pluginsdk.HostServices{
		PluginID:     strings.ToLower(strings.TrimSpace(pluginID)),
		Transactions: s.Transactions, DataScopes: s.DataScopes, DataStore: s.DataStore,
		Events: s.Events, DocumentNumbers: s.DocumentNumbers, Documents: s.Documents,
		Files: s.Files, Audit: s.Audit, Config: s.Config, Secrets: s.Secrets,
		Workflows: s.Workflows, Jobs: s.Jobs,
	}
	for _, override := range overrides {
		applyHostOverrides(&host, override)
	}
	if err := host.Validate(); err != nil {
		return pluginsdk.HostServices{}, err
	}
	return host, nil
}

func applyHostOverrides(host *pluginsdk.HostServices, override HostOverrides) {
	if override.Transactions != nil {
		host.Transactions = override.Transactions
	}
	if override.DataScopes != nil {
		host.DataScopes = override.DataScopes
	}
	if override.DataStore != nil {
		host.DataStore = override.DataStore
	}
	if override.Events != nil {
		host.Events = override.Events
	}
	if override.DocumentNumbers != nil {
		host.DocumentNumbers = override.DocumentNumbers
	}
	if override.Documents != nil {
		host.Documents = override.Documents
	}
	if override.Files != nil {
		host.Files = override.Files
	}
	if override.Audit != nil {
		host.Audit = override.Audit
	}
	if override.Config != nil {
		host.Config = override.Config
	}
	if override.Secrets != nil {
		host.Secrets = override.Secrets
	}
	if override.Workflows != nil {
		host.Workflows = override.Workflows
	}
	if override.Jobs != nil {
		host.Jobs = override.Jobs
	}
}

type fixtureTransaction struct{ ctx context.Context }

func (t fixtureTransaction) Context() context.Context { return t.ctx }

type TransactionService struct {
	Failures *FailurePlan
}

func (s *TransactionService) Within(ctx context.Context, fn func(pluginsdk.Transaction) error) error {
	if err := s.Failures.take(OperationTransactionWithin); err != nil {
		return err
	}
	if fn == nil {
		return errors.New("fixture transaction callback is required")
	}
	return fn(fixtureTransaction{ctx: ctx})
}

type DataScopeService struct {
	mu          sync.RWMutex
	Predicate   pluginsdk.ScopePredicate
	Permissions map[pluginsdk.Permission]pluginsdk.ScopePredicate
	Failures    *FailurePlan
}

func (s *DataScopeService) Resolve(_ context.Context, permission pluginsdk.Permission) (pluginsdk.ScopePredicate, error) {
	if err := s.Failures.take(OperationScopeResolve); err != nil {
		return pluginsdk.ScopePredicate{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if predicate, ok := s.Permissions[permission]; ok {
		return predicate, nil
	}
	return s.Predicate, nil
}

type DataStoreService struct {
	mu            sync.Mutex
	Failures      *FailurePlan
	Queries       []pluginsdk.DataQuery
	Mutations     []pluginsdk.DataMutation
	Aggregates    []pluginsdk.DataAggregateQuery
	QueryFunc     func(context.Context, pluginsdk.DataQuery) (pluginsdk.DataPage, error)
	MutateFunc    func(context.Context, pluginsdk.DataMutation) (pluginsdk.DataMutationResult, error)
	AggregateFunc func(context.Context, pluginsdk.DataAggregateQuery) (pluginsdk.DataAggregatePage, error)
}

func (s *DataStoreService) Query(ctx context.Context, query pluginsdk.DataQuery) (pluginsdk.DataPage, error) {
	if err := s.Failures.take(OperationDataQuery); err != nil {
		return pluginsdk.DataPage{}, err
	}
	s.mu.Lock()
	s.Queries = append(s.Queries, query)
	fn := s.QueryFunc
	s.mu.Unlock()
	if fn != nil {
		return fn(ctx, query)
	}
	return pluginsdk.DataPage{Records: []pluginsdk.DataRecord{}}, nil
}

func (s *DataStoreService) Mutate(ctx context.Context, mutation pluginsdk.DataMutation) (pluginsdk.DataMutationResult, error) {
	if err := s.Failures.take(OperationDataMutate); err != nil {
		return pluginsdk.DataMutationResult{}, err
	}
	s.mu.Lock()
	s.Mutations = append(s.Mutations, mutation)
	fn := s.MutateFunc
	s.mu.Unlock()
	if fn != nil {
		return fn(ctx, mutation)
	}
	if mutation.Operation == pluginsdk.DataMutationDelete {
		return pluginsdk.DataMutationResult{RowsAffected: 1}, nil
	}
	values := make(map[string]pluginsdk.DataValue, len(mutation.Key)+len(mutation.Values))
	for key, value := range mutation.Key {
		values[key] = value
	}
	for key, value := range mutation.Values {
		values[key] = value
	}
	version := int64(1)
	if mutation.ExpectedVersion != nil {
		version = *mutation.ExpectedVersion + 1
	}
	return pluginsdk.DataMutationResult{
		RowsAffected: 1,
		Record:       &pluginsdk.DataRecord{Values: values, Version: version},
	}, nil
}

func (s *DataStoreService) Aggregate(ctx context.Context, query pluginsdk.DataAggregateQuery) (pluginsdk.DataAggregatePage, error) {
	if err := s.Failures.take(OperationDataAggregate); err != nil {
		return pluginsdk.DataAggregatePage{}, err
	}
	s.mu.Lock()
	s.Aggregates = append(s.Aggregates, query)
	fn := s.AggregateFunc
	s.mu.Unlock()
	if fn != nil {
		return fn(ctx, query)
	}
	row := pluginsdk.DataAggregateRow{Group: map[string]pluginsdk.DataValue{}, Values: make([]pluginsdk.DataValue, len(query.Metrics))}
	for index := range query.Metrics {
		row.Values[index] = pluginsdk.DataValue{Type: pluginsdk.DataValueInteger, Value: "0"}
	}
	return pluginsdk.DataAggregatePage{Metrics: query.Metrics, GroupBy: query.GroupBy, Rows: []pluginsdk.DataAggregateRow{row}}, nil
}

type EventService struct {
	mu           sync.Mutex
	Clock        *Clock
	Failures     *FailurePlan
	Publications []pluginsdk.EventPublication
	Envelopes    []pluginsdk.EventEnvelope
}

func (s *EventService) Publish(_ context.Context, publication pluginsdk.EventPublication) (pluginsdk.EventEnvelope, error) {
	if err := s.Failures.take(OperationEventPublish); err != nil {
		return pluginsdk.EventEnvelope{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	envelope := pluginsdk.EventEnvelope{
		ID: "fixture-event-" + strconv.Itoa(len(s.Envelopes)+1), Publisher: "fixture",
		Name: publication.Name, SchemaVersion: publication.SchemaVersion, PayloadType: publication.Name,
		Scope: publication.Scope, CorrelationID: publication.CorrelationID, CausationID: publication.CausationID,
		Subject: publication.Subject, Payload: publication.Payload, OccurredAt: s.Clock.Now(),
	}
	s.Publications = append(s.Publications, publication)
	s.Envelopes = append(s.Envelopes, envelope)
	return envelope, nil
}

type DocumentNumberService struct {
	mu       sync.Mutex
	Clock    *Clock
	Failures *FailurePlan
	next     map[string]int64
	issued   map[string]pluginsdk.DocumentNumberResult
}

func newDocumentNumberService(clock *Clock, failures *FailurePlan) *DocumentNumberService {
	return &DocumentNumberService{Clock: clock, Failures: failures, next: make(map[string]int64), issued: make(map[string]pluginsdk.DocumentNumberResult)}
}

func (s *DocumentNumberService) Preview(_ context.Context, input pluginsdk.DocumentNumberInput) (pluginsdk.DocumentNumberResult, error) {
	if err := s.Failures.take(OperationNumberPreview); err != nil {
		return pluginsdk.DocumentNumberResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.result(input, false), nil
}

func (s *DocumentNumberService) Issue(_ context.Context, input pluginsdk.DocumentNumberInput) (pluginsdk.DocumentNumberResult, error) {
	if err := s.Failures.take(OperationNumberIssue); err != nil {
		return pluginsdk.DocumentNumberResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if result, ok := s.issued[input.IdempotencyKey]; ok {
		result.Duplicate = true
		return result, nil
	}
	result := s.result(input, true)
	s.issued[input.IdempotencyKey] = result
	return result, nil
}

func (s *DocumentNumberService) result(input pluginsdk.DocumentNumberInput, commit bool) pluginsdk.DocumentNumberResult {
	occurredAt := input.OccurredAt.UTC()
	if occurredAt.IsZero() {
		occurredAt = s.Clock.Now()
	}
	period := documentNumberPeriod(input.Rule.Period, occurredAt)
	key := input.TenantID + ":" + input.Rule.DocumentType + ":" + period
	sequence := s.next[key]
	if sequence == 0 {
		sequence = input.Rule.Start
		if sequence == 0 {
			sequence = 1
		}
	}
	if commit {
		s.next[key] = sequence + 1
	}
	parts := []string{input.Rule.Prefix}
	if period != "" {
		parts = append(parts, period)
	}
	parts = append(parts, fmt.Sprintf("%0*d", input.Rule.Width, sequence))
	return pluginsdk.DocumentNumberResult{Number: strings.Join(parts, input.Rule.Separator), PeriodKey: period, Sequence: sequence}
}

func documentNumberPeriod(period pluginsdk.DocumentNumberPeriod, at time.Time) string {
	switch period {
	case pluginsdk.DocumentNumberPeriodYear:
		return at.Format("2006")
	case pluginsdk.DocumentNumberPeriodMonth:
		return at.Format("200601")
	case pluginsdk.DocumentNumberPeriodDay:
		return at.Format("20060102")
	default:
		return ""
	}
}

type storedFile struct {
	object  pluginsdk.FileObject
	content []byte
}

type FileService struct {
	mu       sync.RWMutex
	Clock    *Clock
	Failures *FailurePlan
	files    map[string]storedFile
}

func newFileService(clock *Clock, failures *FailurePlan) *FileService {
	return &FileService{Clock: clock, Failures: failures, files: make(map[string]storedFile)}
}

func (s *FileService) Store(_ context.Context, input pluginsdk.FileWrite) (pluginsdk.FileObject, error) {
	if err := s.Failures.take(OperationFileStore); err != nil {
		return pluginsdk.FileObject{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sum := sha256.Sum256(input.Content)
	id := strings.TrimSpace(input.Key)
	if id == "" {
		id = "fixture-file-" + strconv.Itoa(len(s.files)+1)
	}
	now := s.Clock.Now()
	object := pluginsdk.FileObject{
		ID: id, Key: input.Key, Name: input.Name, Size: int64(len(input.Content)),
		MIME: "application/octet-stream", Hash: hex.EncodeToString(sum[:]), Visibility: input.Visibility,
		Status: "ready", Metadata: cloneStringMap(input.Metadata), CreatedAt: now, UpdatedAt: now,
	}
	s.files[id] = storedFile{object: object, content: append([]byte(nil), input.Content...)}
	return object, nil
}

func (s *FileService) List(_ context.Context, query pluginsdk.FileQuery) ([]pluginsdk.FileObject, error) {
	if err := s.Failures.take(OperationFileList); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.files))
	for id := range s.files {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	items := make([]pluginsdk.FileObject, 0, len(ids))
	for _, id := range ids {
		item := s.files[id].object
		if query.Visibility != "" && item.Visibility != query.Visibility {
			continue
		}
		items = append(items, item)
	}
	start := query.Offset
	if start < 0 {
		start = 0
	}
	if start >= len(items) {
		return []pluginsdk.FileObject{}, nil
	}
	end := len(items)
	if query.Limit > 0 && start+query.Limit < end {
		end = start + query.Limit
	}
	return append([]pluginsdk.FileObject(nil), items[start:end]...), nil
}

func (s *FileService) Get(_ context.Context, id string) (pluginsdk.FileObject, error) {
	if err := s.Failures.take(OperationFileGet); err != nil {
		return pluginsdk.FileObject{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.files[id]
	if !ok {
		return pluginsdk.FileObject{}, fmt.Errorf("fixture file %q not found", id)
	}
	return item.object, nil
}

func (s *FileService) Content(id string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.files[id]
	if !ok {
		return nil, fmt.Errorf("fixture file %q not found", id)
	}
	return append([]byte(nil), item.content...), nil
}

func (s *FileService) Download(_ context.Context, id string) (pluginsdk.FileDownload, error) {
	if err := s.Failures.take(OperationFileDownload); err != nil {
		return pluginsdk.FileDownload{}, err
	}
	if _, err := s.Get(context.Background(), id); err != nil {
		return pluginsdk.FileDownload{}, err
	}
	return pluginsdk.FileDownload{ID: id, URL: "fixture://files/" + id, Method: "GET", ExpiresAt: s.Clock.Now().Add(time.Hour), Headers: map[string]string{}}, nil
}

func (s *FileService) Delete(_ context.Context, id string) error {
	if err := s.Failures.take(OperationFileDelete); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.files[id]; !ok {
		return fmt.Errorf("fixture file %q not found", id)
	}
	delete(s.files, id)
	return nil
}

type AuditRecord struct {
	Entry   pluginsdk.AuditEntry
	Receipt pluginsdk.AuditReceipt
}

type AuditService struct {
	mu       sync.Mutex
	Clock    *Clock
	Failures *FailurePlan
	Records  []AuditRecord
}

func (s *AuditService) Record(_ context.Context, entry pluginsdk.AuditEntry) (pluginsdk.AuditReceipt, error) {
	if err := s.Failures.take(OperationAuditRecord); err != nil {
		return pluginsdk.AuditReceipt{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	receipt := pluginsdk.AuditReceipt{ID: "fixture-audit-" + strconv.Itoa(len(s.Records)+1), OccurredAt: s.Clock.Now()}
	s.Records = append(s.Records, AuditRecord{Entry: entry, Receipt: receipt})
	return receipt, nil
}

type ConfigService struct {
	mu       sync.RWMutex
	Failures *FailurePlan
	values   map[string]any
}

func (s *ConfigService) Get(context.Context) (map[string]any, error) {
	if err := s.Failures.take(OperationConfigGet); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneAnyMap(s.values), nil
}

func (s *ConfigService) Replace(_ context.Context, values map[string]any) (map[string]any, error) {
	if err := s.Failures.take(OperationConfigReplace); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.values = cloneAnyMap(values)
	result := cloneAnyMap(s.values)
	s.mu.Unlock()
	return result, nil
}

type SecretService struct {
	mu       sync.RWMutex
	Failures *FailurePlan
	values   map[string]string
}

func (s *SecretService) Get(_ context.Context, key string) (string, error) {
	if err := s.Failures.take(OperationSecretGet); err != nil {
		return "", err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.values[key]
	if !ok {
		return "", fmt.Errorf("fixture secret %q not found", key)
	}
	return value, nil
}

func (s *SecretService) Set(_ context.Context, key, value string) error {
	if err := s.Failures.take(OperationSecretSet); err != nil {
		return err
	}
	s.mu.Lock()
	s.values[key] = value
	s.mu.Unlock()
	return nil
}

type DocumentService struct{}

func (*DocumentService) Submit(context.Context, pluginsdk.DocumentWorkflowSubmitInput) (pluginsdk.DocumentWorkflowResult, error) {
	return pluginsdk.DocumentWorkflowResult{}, ErrDocumentServiceNotConfigured
}
func (*DocumentService) Act(context.Context, pluginsdk.DocumentWorkflowActionInput) (pluginsdk.DocumentWorkflowResult, error) {
	return pluginsdk.DocumentWorkflowResult{}, ErrDocumentServiceNotConfigured
}
func (*DocumentService) Get(context.Context, pluginsdk.DocumentWorkflowGetInput) (pluginsdk.DocumentWorkflowResult, error) {
	return pluginsdk.DocumentWorkflowResult{}, ErrDocumentServiceNotConfigured
}
func (*DocumentService) Search(context.Context, pluginsdk.DocumentSearchInput) (pluginsdk.DocumentSearchPage, error) {
	return pluginsdk.DocumentSearchPage{}, ErrDocumentServiceNotConfigured
}
func (*DocumentService) Print(context.Context, pluginsdk.DocumentPrintInput) (pluginsdk.DocumentPrintPayload, error) {
	return pluginsdk.DocumentPrintPayload{}, ErrDocumentServiceNotConfigured
}
func (*DocumentService) Export(context.Context, pluginsdk.DocumentExportInput) (pluginsdk.Job, error) {
	return pluginsdk.Job{}, ErrDocumentServiceNotConfigured
}
func (*DocumentService) AddAttachment(context.Context, pluginsdk.DocumentAttachmentAddInput) (pluginsdk.DocumentAttachmentResult, error) {
	return pluginsdk.DocumentAttachmentResult{}, ErrDocumentServiceNotConfigured
}
func (*DocumentService) RemoveAttachment(context.Context, pluginsdk.DocumentAttachmentRemoveInput) (pluginsdk.DocumentAttachmentResult, error) {
	return pluginsdk.DocumentAttachmentResult{}, ErrDocumentServiceNotConfigured
}
func (*DocumentService) ListAttachments(context.Context, pluginsdk.DocumentCollaborationQueryInput) ([]pluginsdk.DocumentAttachment, error) {
	return nil, ErrDocumentServiceNotConfigured
}
func (*DocumentService) AddComment(context.Context, pluginsdk.DocumentCommentAddInput) (pluginsdk.DocumentCommentResult, error) {
	return pluginsdk.DocumentCommentResult{}, ErrDocumentServiceNotConfigured
}
func (*DocumentService) ListComments(context.Context, pluginsdk.DocumentCollaborationQueryInput) ([]pluginsdk.DocumentComment, error) {
	return nil, ErrDocumentServiceNotConfigured
}
func (*DocumentService) Timeline(context.Context, pluginsdk.DocumentTimelineQueryInput) (pluginsdk.DocumentTimelinePage, error) {
	return pluginsdk.DocumentTimelinePage{}, ErrDocumentServiceNotConfigured
}

func cloneStringMap(values map[string]string) map[string]string {
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func cloneAnyMap(values map[string]any) map[string]any {
	result := make(map[string]any, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

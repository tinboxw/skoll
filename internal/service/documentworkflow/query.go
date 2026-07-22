package documentworkflow

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type SearchQuery struct {
	Key              Key
	Types            []string
	States           []string
	Text             string
	CreatedBy        []string
	CreatedFrom      *time.Time
	CreatedTo        *time.Time
	SortField        pluginsdk.DocumentSearchSortField
	Direction        pluginsdk.DocumentSearchDirection
	CursorValue      string
	CursorDocumentID string
	Limit            int
}

type documentSearchCursor struct {
	Version    int    `json:"version"`
	QueryHash  string `json:"queryHash"`
	SortValue  string `json:"sortValue"`
	DocumentID string `json:"documentId"`
}

func (s *Service) Search(ctx context.Context, pluginID string, input pluginsdk.DocumentSearchInput) (pluginsdk.DocumentSearchPage, error) {
	if err := input.Validate(); err != nil {
		return pluginsdk.DocumentSearchPage{}, err
	}
	key, err := bindingKey(pluginID, input.TenantID, "_")
	if err != nil {
		return pluginsdk.DocumentSearchPage{}, err
	}
	normalized := normalizeDocumentSearch(input)
	hash, err := documentSearchHash(key.PluginID, normalized)
	if err != nil {
		return pluginsdk.DocumentSearchPage{}, err
	}
	query := SearchQuery{
		Key: Key{PluginID: key.PluginID, TenantID: input.TenantID}, Types: normalized.Types, States: normalized.States,
		Text: normalized.Text, CreatedBy: normalized.CreatedBy, CreatedFrom: cloneSearchTime(normalized.CreatedFrom), CreatedTo: cloneSearchTime(normalized.CreatedTo),
		SortField: normalized.SortField, Direction: normalized.Direction, Limit: normalized.Limit + 1,
	}
	if normalized.Cursor != "" {
		cursor, decodeErr := decodeDocumentSearchCursor(normalized.Cursor)
		if decodeErr != nil || cursor.QueryHash != hash {
			return pluginsdk.DocumentSearchPage{}, pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorInvalidRequest, "cursor", "document search cursor does not match the query", false)
		}
		if err = validateDocumentSearchCursor(cursor, normalized.SortField); err != nil {
			return pluginsdk.DocumentSearchPage{}, err
		}
		query.CursorValue, query.CursorDocumentID = cursor.SortValue, cursor.DocumentID
	}
	bindings, err := s.repository.Search(ctx, query)
	if err != nil {
		return pluginsdk.DocumentSearchPage{}, err
	}
	hasMore := len(bindings) > normalized.Limit
	if hasMore {
		bindings = bindings[:normalized.Limit]
	}
	items := make([]pluginsdk.DocumentSummary, 0, len(bindings))
	for _, binding := range bindings {
		items = append(items, documentSummary(binding.Document))
	}
	page := pluginsdk.DocumentSearchPage{Items: items, HasMore: hasMore}
	if hasMore && len(bindings) > 0 {
		last := bindings[len(bindings)-1].Document
		page.NextCursor, err = encodeDocumentSearchCursor(documentSearchCursor{
			Version: 1, QueryHash: hash, SortValue: documentSearchSortValue(last, normalized.SortField), DocumentID: last.ID,
		})
		if err != nil {
			return pluginsdk.DocumentSearchPage{}, err
		}
	}
	return page, nil
}

func (s *Service) Print(ctx context.Context, pluginID string, input pluginsdk.DocumentPrintInput) (pluginsdk.DocumentPrintPayload, error) {
	if err := input.Validate(); err != nil {
		return pluginsdk.DocumentPrintPayload{}, err
	}
	key, err := bindingKey(pluginID, input.TenantID, input.DocumentID)
	if err != nil {
		return pluginsdk.DocumentPrintPayload{}, err
	}
	binding, err := s.repository.Get(ctx, key, false)
	if err != nil {
		return pluginsdk.DocumentPrintPayload{}, err
	}
	workflow, err := s.workflow.GetInstance(ctx, binding.WorkflowInstanceID)
	if err != nil {
		return pluginsdk.DocumentPrintPayload{}, err
	}
	document, err := cloneDocumentRecord(binding.Document)
	if err != nil {
		return pluginsdk.DocumentPrintPayload{}, err
	}
	redacted := []string(nil)
	if !input.IncludeSensitive {
		redacted = redactSensitiveDocumentFields(binding.Schema, &document)
	}
	return pluginsdk.DocumentPrintPayload{
		Schema: binding.Schema, Document: document, Workflow: workflow, RedactedFields: redacted, GeneratedAt: s.now().UTC(),
	}, nil
}

func normalizeDocumentSearch(input pluginsdk.DocumentSearchInput) pluginsdk.DocumentSearchInput {
	input.Types = append([]string(nil), input.Types...)
	input.States = append([]string(nil), input.States...)
	input.CreatedBy = append([]string(nil), input.CreatedBy...)
	if input.SortField == "" {
		input.SortField = pluginsdk.DocumentSearchSortUpdatedAt
	}
	if input.Direction == "" {
		input.Direction = pluginsdk.DocumentSearchDescending
	}
	if input.Limit == 0 {
		input.Limit = 50
	}
	return input
}

func documentSearchHash(pluginID string, input pluginsdk.DocumentSearchInput) (string, error) {
	types, states, creators := append([]string(nil), input.Types...), append([]string(nil), input.States...), append([]string(nil), input.CreatedBy...)
	sort.Strings(types)
	sort.Strings(states)
	sort.Strings(creators)
	payload := struct {
		PluginID, TenantID, Text string
		Types, States, CreatedBy []string
		CreatedFrom, CreatedTo   *time.Time
		SortField                pluginsdk.DocumentSearchSortField
		Direction                pluginsdk.DocumentSearchDirection
	}{pluginID, input.TenantID, input.Text, types, states, creators, input.CreatedFrom, input.CreatedTo, input.SortField, input.Direction}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func encodeDocumentSearchCursor(cursor documentSearchCursor) (string, error) {
	raw, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(raw)
	if len(encoded) > pluginsdk.MaxDocumentSearchCursor {
		return "", fmt.Errorf("document search cursor exceeds size limit")
	}
	return encoded, nil
}

func decodeDocumentSearchCursor(value string) (documentSearchCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return documentSearchCursor{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var cursor documentSearchCursor
	if err = decoder.Decode(&cursor); err != nil {
		return documentSearchCursor{}, err
	}
	if err = decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return documentSearchCursor{}, fmt.Errorf("document search cursor has trailing data")
		}
		return documentSearchCursor{}, err
	}
	return cursor, nil
}

func validateDocumentSearchCursor(cursor documentSearchCursor, field pluginsdk.DocumentSearchSortField) error {
	if cursor.Version != 1 || len(cursor.QueryHash) != sha256.Size*2 || strings.TrimSpace(cursor.SortValue) != cursor.SortValue || cursor.SortValue == "" || strings.TrimSpace(cursor.DocumentID) == "" {
		return pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorInvalidRequest, "cursor", "document search cursor is invalid", false)
	}
	if field == pluginsdk.DocumentSearchSortUpdatedAt || field == pluginsdk.DocumentSearchSortCreatedAt {
		value, err := time.Parse(time.RFC3339Nano, cursor.SortValue)
		if err != nil || value.Location() != time.UTC {
			return pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorInvalidRequest, "cursor", "document search time cursor is invalid", false)
		}
	}
	return nil
}

func documentSearchSortValue(document pluginsdk.DocumentRecord, field pluginsdk.DocumentSearchSortField) string {
	switch field {
	case pluginsdk.DocumentSearchSortCreatedAt:
		return document.Metadata.CreatedAt.UTC().Format(time.RFC3339Nano)
	case pluginsdk.DocumentSearchSortNumber:
		return document.Number
	default:
		return document.Metadata.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
}

func documentSummary(document pluginsdk.DocumentRecord) pluginsdk.DocumentSummary {
	return pluginsdk.DocumentSummary{
		ID: document.ID, Type: document.Type, Number: document.Number, Title: document.Title, State: document.State, Version: document.Version,
		CreatedAt: document.Metadata.CreatedAt, UpdatedAt: document.Metadata.UpdatedAt, CreatedBy: document.Metadata.CreatedBy,
		UpdatedBy: document.Metadata.UpdatedBy, Tags: append([]string(nil), document.Metadata.Tags...),
	}
}

func cloneDocumentRecord(document pluginsdk.DocumentRecord) (pluginsdk.DocumentRecord, error) {
	raw, err := json.Marshal(document)
	if err != nil {
		return pluginsdk.DocumentRecord{}, err
	}
	var out pluginsdk.DocumentRecord
	if err = json.Unmarshal(raw, &out); err != nil {
		return pluginsdk.DocumentRecord{}, err
	}
	return out, nil
}

func redactSensitiveDocumentFields(schema pluginsdk.DocumentSchema, document *pluginsdk.DocumentRecord) []string {
	redacted := make([]string, 0)
	for _, field := range schema.Header {
		if !field.Sensitive {
			continue
		}
		delete(document.Header, field.Key)
		redacted = append(redacted, "header."+field.Key)
	}
	for _, lineSchema := range schema.Lines {
		for _, field := range lineSchema.Fields {
			if !field.Sensitive {
				continue
			}
			for index := range document.Lines[lineSchema.Key] {
				delete(document.Lines[lineSchema.Key][index].Values, field.Key)
			}
			redacted = append(redacted, "lines."+lineSchema.Key+"."+field.Key)
		}
	}
	return redacted
}

func cloneSearchTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

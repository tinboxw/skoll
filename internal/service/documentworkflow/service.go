package documentworkflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

var (
	ErrNotFound            = errors.New("document workflow binding not found")
	ErrConflict            = errors.New("document workflow conflict")
	ErrTransactionRequired = errors.New("document workflow transaction is required")
)

type Key struct {
	PluginID   string
	TenantID   string
	DocumentID string
}

type Binding struct {
	Key                Key
	Schema             pluginsdk.DocumentSchema
	Document           pluginsdk.DocumentRecord
	DefinitionID       string
	WorkflowInstanceID string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type ActionRecord struct {
	Key            Key
	IdempotencyKey string
	Action         string
	RequestHash    string
	ResultJSON     []byte
	CreatedAt      time.Time
}

type Repository interface {
	Get(ctx context.Context, key Key, forUpdate bool) (Binding, error)
	FindAction(ctx context.Context, key Key, idempotencyKey string) (ActionRecord, bool, error)
	Create(ctx context.Context, binding Binding, action ActionRecord, event pluginsdk.DocumentTimelineEvent) error
	Update(ctx context.Context, binding Binding, previousVersion int64, action ActionRecord, event pluginsdk.DocumentTimelineEvent) error
	AddAttachment(ctx context.Context, key Key, attachment pluginsdk.DocumentAttachment, event pluginsdk.DocumentTimelineEvent) (pluginsdk.DocumentAttachmentResult, error)
	RemoveAttachment(ctx context.Context, key Key, attachmentID string, actor pluginsdk.WorkflowActor, now time.Time, event pluginsdk.DocumentTimelineEvent) (pluginsdk.DocumentAttachmentResult, error)
	ListAttachments(ctx context.Context, key Key, offset, limit int, includeRemoved bool) ([]pluginsdk.DocumentAttachment, error)
	AddComment(ctx context.Context, key Key, comment pluginsdk.DocumentComment, event pluginsdk.DocumentTimelineEvent) (pluginsdk.DocumentCommentResult, error)
	ListComments(ctx context.Context, key Key, offset, limit int) ([]pluginsdk.DocumentComment, error)
	Timeline(ctx context.Context, key Key, afterSequence int64, limit int) (pluginsdk.DocumentTimelinePage, error)
	Search(ctx context.Context, query SearchQuery) ([]Binding, error)
}

type Service struct {
	repository Repository
	workflow   pluginsdk.WorkflowService
	now        func() time.Time
}

func NewService(repository Repository, workflow pluginsdk.WorkflowService) *Service {
	if repository == nil || workflow == nil {
		panic("document workflow repository and workflow service are required")
	}
	return &Service{repository: repository, workflow: workflow, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) Submit(ctx context.Context, pluginID string, actor pluginsdk.WorkflowActor, input pluginsdk.DocumentWorkflowSubmitInput) (pluginsdk.DocumentWorkflowResult, error) {
	if err := input.Validate(); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	if err := validateActor(actor); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	key, err := bindingKey(pluginID, input.TenantID, input.Draft.ID)
	if err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	hash, err := requestHash(actor, input)
	if err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	if prior, exists, findErr := s.repository.FindAction(ctx, key, input.IdempotencyKey); findErr != nil {
		return pluginsdk.DocumentWorkflowResult{}, findErr
	} else if exists {
		return replay(prior, hash)
	}
	nextState, err := input.Schema.NextState(input.Schema.InitialState, "submit", input.Comment)
	if err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	now := s.now().UTC()
	document := pluginsdk.DocumentRecord{
		ID: input.Draft.ID, Type: input.Draft.Type, SchemaVersion: input.Draft.SchemaVersion,
		Number: input.Draft.Number, Title: input.Draft.Title, State: nextState, Version: 1,
		Header: input.Draft.Header, Lines: input.Draft.Lines,
		Metadata: pluginsdk.DocumentMetadata{CreatedAt: now, UpdatedAt: now, CreatedBy: actor.ID, UpdatedBy: actor.ID, Tags: input.Draft.Tags},
	}
	if err := input.Schema.ValidateRecord(document); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	workflow, err := s.workflow.Start(ctx, pluginsdk.WorkflowStartInput{
		ID: input.InstanceID, DefinitionID: input.DefinitionID, BusinessType: document.Type,
		BusinessID: document.ID, Title: document.Title,
	})
	if err != nil {
		return pluginsdk.DocumentWorkflowResult{}, fmt.Errorf("%w: start workflow: %v", ErrConflict, err)
	}
	result := pluginsdk.DocumentWorkflowResult{Document: document, Workflow: workflow}
	action, err := newActionRecord(key, input.IdempotencyKey, "submit", hash, result, now)
	if err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	binding := Binding{
		Key: key, Schema: input.Schema, Document: document, DefinitionID: input.DefinitionID,
		WorkflowInstanceID: input.InstanceID, CreatedAt: now, UpdatedAt: now,
	}
	event := documentActionEvent(input.Draft.ID, input.IdempotencyKey, "submit", actor, now)
	if err := s.repository.Create(ctx, binding, action, event); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	return result, nil
}

func (s *Service) Act(ctx context.Context, pluginID string, actor pluginsdk.WorkflowActor, input pluginsdk.DocumentWorkflowActionInput) (pluginsdk.DocumentWorkflowResult, error) {
	if err := input.Validate(); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	if err := validateActor(actor); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	key, err := bindingKey(pluginID, input.TenantID, input.DocumentID)
	if err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	hash, err := requestHash(actor, input)
	if err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	if prior, exists, findErr := s.repository.FindAction(ctx, key, input.IdempotencyKey); findErr != nil {
		return pluginsdk.DocumentWorkflowResult{}, findErr
	} else if exists {
		return replay(prior, hash)
	}
	binding, err := s.repository.Get(ctx, key, true)
	if err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	// A competing transaction may have committed while this request waited for
	// the document row lock. Recheck inside the serialized update boundary.
	if prior, exists, findErr := s.repository.FindAction(ctx, key, input.IdempotencyKey); findErr != nil {
		return pluginsdk.DocumentWorkflowResult{}, findErr
	} else if exists {
		return replay(prior, hash)
	}
	if binding.Document.Version != input.ExpectedVersion {
		return pluginsdk.DocumentWorkflowResult{}, ErrConflict
	}
	previousVersion := binding.Document.Version
	nextState := binding.Document.State
	if input.Action != pluginsdk.DocumentWorkflowDelegate {
		nextState, err = binding.Schema.NextState(binding.Document.State, string(input.Action), input.Comment)
		if err != nil {
			return pluginsdk.DocumentWorkflowResult{}, err
		}
	}
	workflow, err := s.applyWorkflowAction(ctx, binding.WorkflowInstanceID, input)
	if err != nil {
		return pluginsdk.DocumentWorkflowResult{}, fmt.Errorf("%w: apply workflow action: %v", ErrConflict, err)
	}
	now := s.now().UTC()
	if input.Action != pluginsdk.DocumentWorkflowDelegate {
		binding.Document.State = nextState
		binding.Document.Version++
		binding.Document.Metadata.UpdatedAt = now
		binding.Document.Metadata.UpdatedBy = actor.ID
	}
	if err := binding.Schema.ValidateRecord(binding.Document); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	binding.UpdatedAt = now
	result := pluginsdk.DocumentWorkflowResult{Document: binding.Document, Workflow: workflow}
	action, err := newActionRecord(key, input.IdempotencyKey, string(input.Action), hash, result, binding.UpdatedAt)
	if err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	event := documentActionEvent(input.DocumentID, input.IdempotencyKey, string(input.Action), actor, now)
	if err := s.repository.Update(ctx, binding, previousVersion, action, event); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	return result, nil
}

func (s *Service) AddAttachment(ctx context.Context, pluginID string, actor pluginsdk.WorkflowActor, input pluginsdk.DocumentAttachmentAddInput, file pluginsdk.FileObject) (pluginsdk.DocumentAttachmentResult, error) {
	if err := input.Validate(); err != nil {
		return pluginsdk.DocumentAttachmentResult{}, err
	}
	if err := validateActor(actor); err != nil {
		return pluginsdk.DocumentAttachmentResult{}, err
	}
	if strings.TrimSpace(file.ID) != input.FileID || strings.TrimSpace(file.Status) != "available" {
		return pluginsdk.DocumentAttachmentResult{}, fmt.Errorf("document attachment file is unavailable")
	}
	key, err := bindingKey(pluginID, input.TenantID, input.DocumentID)
	if err != nil {
		return pluginsdk.DocumentAttachmentResult{}, err
	}
	if _, err = s.repository.Get(ctx, key, true); err != nil {
		return pluginsdk.DocumentAttachmentResult{}, err
	}
	now := s.now().UTC()
	attachment := pluginsdk.DocumentAttachment{
		ID: input.AttachmentID, DocumentID: input.DocumentID, File: file,
		AddedBy: actor, AddedAt: now,
	}
	event := pluginsdk.DocumentTimelineEvent{
		ID: "attachment-added:" + input.AttachmentID, DocumentID: input.DocumentID,
		Kind: pluginsdk.DocumentTimelineAttachmentAdded, AttachmentID: input.AttachmentID, FileID: input.FileID,
		Actor: actor, OccurredAt: now,
	}
	return s.repository.AddAttachment(ctx, key, attachment, event)
}

func (s *Service) RemoveAttachment(ctx context.Context, pluginID string, actor pluginsdk.WorkflowActor, input pluginsdk.DocumentAttachmentRemoveInput) (pluginsdk.DocumentAttachmentResult, error) {
	if err := input.Validate(); err != nil {
		return pluginsdk.DocumentAttachmentResult{}, err
	}
	if err := validateActor(actor); err != nil {
		return pluginsdk.DocumentAttachmentResult{}, err
	}
	key, err := bindingKey(pluginID, input.TenantID, input.DocumentID)
	if err != nil {
		return pluginsdk.DocumentAttachmentResult{}, err
	}
	if _, err = s.repository.Get(ctx, key, true); err != nil {
		return pluginsdk.DocumentAttachmentResult{}, err
	}
	now := s.now().UTC()
	event := pluginsdk.DocumentTimelineEvent{
		ID: "attachment-removed:" + input.AttachmentID, DocumentID: input.DocumentID,
		Kind: pluginsdk.DocumentTimelineAttachmentRemoved, AttachmentID: input.AttachmentID,
		Actor: actor, OccurredAt: now,
	}
	return s.repository.RemoveAttachment(ctx, key, input.AttachmentID, actor, now, event)
}

func (s *Service) ListAttachments(ctx context.Context, pluginID string, input pluginsdk.DocumentCollaborationQueryInput) ([]pluginsdk.DocumentAttachment, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	key, err := bindingKey(pluginID, input.TenantID, input.DocumentID)
	if err != nil {
		return nil, err
	}
	if _, err = s.repository.Get(ctx, key, false); err != nil {
		return nil, err
	}
	return s.repository.ListAttachments(ctx, key, input.Offset, activityLimit(input.Limit), input.IncludeRemoved)
}

func (s *Service) AddComment(ctx context.Context, pluginID string, actor pluginsdk.WorkflowActor, input pluginsdk.DocumentCommentAddInput) (pluginsdk.DocumentCommentResult, error) {
	if err := input.Validate(); err != nil {
		return pluginsdk.DocumentCommentResult{}, err
	}
	if err := validateActor(actor); err != nil {
		return pluginsdk.DocumentCommentResult{}, err
	}
	key, err := bindingKey(pluginID, input.TenantID, input.DocumentID)
	if err != nil {
		return pluginsdk.DocumentCommentResult{}, err
	}
	if _, err = s.repository.Get(ctx, key, true); err != nil {
		return pluginsdk.DocumentCommentResult{}, err
	}
	now := s.now().UTC()
	comment := pluginsdk.DocumentComment{ID: input.CommentID, DocumentID: input.DocumentID, Body: input.Body, Author: actor, CreatedAt: now}
	event := pluginsdk.DocumentTimelineEvent{
		ID: "comment-added:" + input.CommentID, DocumentID: input.DocumentID,
		Kind: pluginsdk.DocumentTimelineCommentAdded, CommentID: input.CommentID, Actor: actor, OccurredAt: now,
	}
	return s.repository.AddComment(ctx, key, comment, event)
}

func (s *Service) ListComments(ctx context.Context, pluginID string, input pluginsdk.DocumentCollaborationQueryInput) ([]pluginsdk.DocumentComment, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	key, err := bindingKey(pluginID, input.TenantID, input.DocumentID)
	if err != nil {
		return nil, err
	}
	if _, err = s.repository.Get(ctx, key, false); err != nil {
		return nil, err
	}
	return s.repository.ListComments(ctx, key, input.Offset, activityLimit(input.Limit))
}

func (s *Service) Timeline(ctx context.Context, pluginID string, input pluginsdk.DocumentTimelineQueryInput) (pluginsdk.DocumentTimelinePage, error) {
	if err := input.Validate(); err != nil {
		return pluginsdk.DocumentTimelinePage{}, err
	}
	key, err := bindingKey(pluginID, input.TenantID, input.DocumentID)
	if err != nil {
		return pluginsdk.DocumentTimelinePage{}, err
	}
	if _, err = s.repository.Get(ctx, key, false); err != nil {
		return pluginsdk.DocumentTimelinePage{}, err
	}
	return s.repository.Timeline(ctx, key, input.AfterSequence, activityLimit(input.Limit))
}

func activityLimit(value int) int {
	if value == 0 {
		return 50
	}
	return value
}

func (s *Service) Get(ctx context.Context, pluginID string, input pluginsdk.DocumentWorkflowGetInput) (pluginsdk.DocumentWorkflowResult, error) {
	if err := input.Validate(); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	key, err := bindingKey(pluginID, input.TenantID, input.DocumentID)
	if err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	binding, err := s.repository.Get(ctx, key, false)
	if err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	workflow, err := s.workflow.GetInstance(ctx, binding.WorkflowInstanceID)
	if err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	return pluginsdk.DocumentWorkflowResult{Document: binding.Document, Workflow: workflow}, nil
}

func (s *Service) applyWorkflowAction(ctx context.Context, instanceID string, input pluginsdk.DocumentWorkflowActionInput) (pluginsdk.WorkflowInstance, error) {
	switch input.Action {
	case pluginsdk.DocumentWorkflowApprove:
		return s.workflow.Approve(ctx, pluginsdk.WorkflowTaskActionInput{InstanceID: instanceID, TaskID: input.TaskID, Comment: input.Comment})
	case pluginsdk.DocumentWorkflowReject:
		return s.workflow.Reject(ctx, pluginsdk.WorkflowTaskActionInput{InstanceID: instanceID, TaskID: input.TaskID, Comment: input.Comment})
	case pluginsdk.DocumentWorkflowWithdraw:
		return s.workflow.Withdraw(ctx, pluginsdk.WorkflowInstanceActionInput{InstanceID: instanceID, Comment: input.Comment})
	case pluginsdk.DocumentWorkflowDelegate:
		return s.workflow.Delegate(ctx, pluginsdk.WorkflowTargetActionInput{InstanceID: instanceID, TaskID: input.TaskID, Target: input.Target, Comment: input.Comment})
	case pluginsdk.DocumentWorkflowCancel:
		return s.workflow.Cancel(ctx, pluginsdk.WorkflowInstanceActionInput{InstanceID: instanceID, Comment: input.Comment})
	default:
		return pluginsdk.WorkflowInstance{}, fmt.Errorf("document workflow action is unsupported")
	}
}

func bindingKey(pluginID, tenantID, documentID string) (Key, error) {
	pluginID = strings.ToLower(strings.TrimSpace(pluginID))
	if pluginID == "" || len(pluginID) > 64 {
		return Key{}, fmt.Errorf("document workflow plugin identity is invalid")
	}
	return Key{PluginID: pluginID, TenantID: tenantID, DocumentID: documentID}, nil
}

func validateActor(actor pluginsdk.WorkflowActor) error {
	if strings.TrimSpace(actor.ID) == "" || len(strings.TrimSpace(actor.ID)) > 512 {
		return fmt.Errorf("document workflow actor is invalid")
	}
	return nil
}

func requestHash(actor pluginsdk.WorkflowActor, input any) (string, error) {
	payload := struct {
		Actor pluginsdk.WorkflowActor `json:"actor"`
		Input any                     `json:"input"`
	}{Actor: actor, Input: input}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

func newActionRecord(key Key, idempotencyKey, action, hash string, result pluginsdk.DocumentWorkflowResult, now time.Time) (ActionRecord, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return ActionRecord{}, err
	}
	return ActionRecord{Key: key, IdempotencyKey: idempotencyKey, Action: action, RequestHash: hash, ResultJSON: raw, CreatedAt: now}, nil
}

func documentActionEvent(documentID, idempotencyKey, action string, actor pluginsdk.WorkflowActor, now time.Time) pluginsdk.DocumentTimelineEvent {
	return pluginsdk.DocumentTimelineEvent{
		ID: "action:" + idempotencyKey, DocumentID: documentID, Kind: pluginsdk.DocumentTimelineAction,
		Action: action, Actor: actor, OccurredAt: now,
	}
}

func replay(prior ActionRecord, hash string) (pluginsdk.DocumentWorkflowResult, error) {
	if prior.RequestHash != hash {
		return pluginsdk.DocumentWorkflowResult{}, ErrConflict
	}
	var result pluginsdk.DocumentWorkflowResult
	if err := json.Unmarshal(prior.ResultJSON, &result); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	result.Duplicate = true
	return result, nil
}

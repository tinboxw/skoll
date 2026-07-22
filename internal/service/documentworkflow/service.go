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
	Create(ctx context.Context, binding Binding, action ActionRecord) error
	Update(ctx context.Context, binding Binding, previousVersion int64, action ActionRecord) error
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
	if err := s.repository.Create(ctx, binding, action); err != nil {
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
	if err := s.repository.Update(ctx, binding, previousVersion, action); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	return result, nil
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
		return s.workflow.Transfer(ctx, pluginsdk.WorkflowTargetActionInput{InstanceID: instanceID, TaskID: input.TaskID, Target: input.Target, Comment: input.Comment})
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

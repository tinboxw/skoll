package workflow

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	permissionsvc "github.com/tinboxw/skoll/internal/service/permission"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"github.com/tinboxw/skoll/pkg/security"
)

const (
	PermissionWorkflowDefinitionManage = "workflow.definition.manage"
	PermissionWorkflowInstanceStart    = "workflow.instance.start"
	PermissionWorkflowTaskAct          = "workflow.task.act"
	PermissionWorkflowInstanceRead     = "workflow.instance.read"
)

type Handler struct {
	service workflowsvc.Service
}

func RegisterWorkflowRoutes(mux *http.ServeMux, service workflowsvc.Service) {
	if mux == nil || service == nil {
		return
	}
	h := &Handler{service: service}
	mux.HandleFunc("POST /v1/workflows/definitions", h.createDefinition)
	mux.HandleFunc("GET /v1/workflows/definitions/{id}", h.getDefinition)
	mux.HandleFunc("POST /v1/workflows/definitions/{id}/publish", h.publishDefinition)
	mux.HandleFunc("POST /v1/workflows/instances", h.startInstance)
	mux.HandleFunc("GET /v1/workflows/instances/{id}", h.getInstance)
	mux.HandleFunc("POST /v1/workflows/instances/{id}/tasks/{taskId}/approve", h.approveTask)
	mux.HandleFunc("POST /v1/workflows/instances/{id}/tasks/{taskId}/reject", h.rejectTask)
	mux.HandleFunc("POST /v1/workflows/instances/{id}/withdraw", h.withdrawInstance)
	mux.HandleFunc("POST /v1/workflows/instances/{id}/tasks/{taskId}/delegate", h.delegateTask)
	mux.HandleFunc("POST /v1/workflows/instances/{id}/tasks/{taskId}/copy", h.copyTask)
	mux.HandleFunc("POST /v1/workflows/substitutions", h.createSubstitution)
	mux.HandleFunc("POST /v1/workflows/substitutions/{id}/revoke", h.revokeSubstitution)
}

func RegisterWorkflowPermissions(service permissionsvc.Service) error {
	if service == nil {
		return nil
	}
	for _, item := range WorkflowPermissionResources() {
		if _, err := service.RegisterResource(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}

func WorkflowPermissionResources() []permissionsvc.RegisterResourceInput {
	return []permissionsvc.RegisterResourceInput{
		{
			Key:    PermissionWorkflowDefinitionManage,
			Type:   domainpermission.ResourceTypeAPI,
			Module: "workflow",
			Source: "system",
			Name:   "Manage workflow definitions",
			Risk:   domainpermission.RiskLevelMedium,
			Metadata: map[string]string{
				"routes": "POST /v1/workflows/definitions;GET /v1/workflows/definitions/{id};POST /v1/workflows/definitions/{id}/publish",
			},
		},
		{
			Key:    PermissionWorkflowInstanceStart,
			Type:   domainpermission.ResourceTypeAPI,
			Module: "workflow",
			Source: "system",
			Name:   "Start workflow instances",
			Risk:   domainpermission.RiskLevelLow,
			Metadata: map[string]string{
				"routes": "POST /v1/workflows/instances",
			},
		},
		{
			Key:    PermissionWorkflowInstanceRead,
			Type:   domainpermission.ResourceTypeAPI,
			Module: "workflow",
			Source: "system",
			Name:   "Read workflow instances",
			Risk:   domainpermission.RiskLevelLow,
			Metadata: map[string]string{
				"routes": "GET /v1/workflows/instances/{id}",
			},
		},
		{
			Key:    PermissionWorkflowTaskAct,
			Type:   domainpermission.ResourceTypeAPI,
			Module: "workflow",
			Source: "system",
			Name:   "Act on workflow tasks",
			Risk:   domainpermission.RiskLevelMedium,
			Metadata: map[string]string{
				"routes": "POST /v1/workflows/instances/{id}/tasks/{taskId}/approve;POST /v1/workflows/instances/{id}/tasks/{taskId}/reject;POST /v1/workflows/instances/{id}/withdraw;POST /v1/workflows/instances/{id}/tasks/{taskId}/delegate;POST /v1/workflows/instances/{id}/tasks/{taskId}/copy;POST /v1/workflows/substitutions;POST /v1/workflows/substitutions/{id}/revoke",
			},
		},
	}
}

type definitionInput struct {
	ID          string            `json:"id"`
	Key         string            `json:"key"`
	Name        string            `json:"name"`
	Version     int               `json:"version"`
	Nodes       []nodeInput       `json:"nodes"`
	Transitions []transitionInput `json:"transitions"`
}

type nodeInput struct {
	ID         string           `json:"id"`
	Key        string           `json:"key"`
	Name       string           `json:"name"`
	Type       string           `json:"type"`
	Assignees  []string         `json:"assignees"`
	Decision   *decisionRule    `json:"decision,omitempty"`
	Escalation *escalationRule  `json:"escalation,omitempty"`
	Signature  *signaturePolicy `json:"signature,omitempty"`
}

type transitionInput struct {
	From      string          `json:"from"`
	To        string          `json:"to"`
	Condition *conditionInput `json:"condition,omitempty"`
}

type decisionRule struct {
	Strategy string `json:"strategy"`
	Quorum   int    `json:"quorum"`
}

type escalationRule struct {
	AfterSeconds int64      `json:"afterSeconds"`
	Target       actorInput `json:"target"`
}

type signaturePolicy struct {
	Meaning         string `json:"meaning"`
	RequireEvidence bool   `json:"requireEvidence"`
}

type workflowValue struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type predicateInput struct {
	Field    string         `json:"field"`
	Operator string         `json:"operator"`
	Value    *workflowValue `json:"value,omitempty"`
}

type conditionInput struct {
	Match      string           `json:"match"`
	Predicates []predicateInput `json:"predicates"`
}

type startInput struct {
	ID           string                   `json:"id"`
	DefinitionID string                   `json:"definitionId"`
	BusinessType string                   `json:"businessType"`
	BusinessID   string                   `json:"businessId"`
	Title        string                   `json:"title"`
	Starter      actorInput               `json:"starter"`
	Variables    map[string]workflowValue `json:"variables"`
}

type actorInput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type taskActionInput struct {
	Actor     actorInput              `json:"actor"`
	Comment   string                  `json:"comment"`
	Signature *decisionSignatureInput `json:"signature,omitempty"`
}

type decisionSignatureInput struct {
	Proof       string   `json:"proof"`
	Meaning     string   `json:"meaning"`
	EvidenceIDs []string `json:"evidenceIds"`
}

type targetActionInput struct {
	Actor   actorInput `json:"actor"`
	Target  actorInput `json:"target"`
	Comment string     `json:"comment"`
}

type substitutionInput struct {
	ID         string     `json:"id"`
	Principal  actorInput `json:"principal"`
	Substitute actorInput `json:"substitute"`
	StartsAt   time.Time  `json:"startsAt"`
	EndsAt     time.Time  `json:"endsAt"`
	Reason     string     `json:"reason"`
}

func (h *Handler) createDefinition(w http.ResponseWriter, r *http.Request) {
	var req definitionInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	definition, err := h.service.CreateDefinition(r.Context(), workflowsvc.CreateDefinitionInput{
		ID:          shared.ID(strings.TrimSpace(req.ID)),
		Key:         req.Key,
		Name:        req.Name,
		Version:     req.Version,
		Nodes:       req.nodes(),
		Transitions: req.transitions(),
		Now:         time.Now().UTC(),
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": definitionRecordFromDomain(*definition)})
}

func (h *Handler) getDefinition(w http.ResponseWriter, r *http.Request) {
	definition, err := h.service.GetDefinition(r.Context(), shared.ID(strings.TrimSpace(r.PathValue("id"))))
	if err != nil {
		apiv1.WriteError(w, http.StatusNotFound, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": definitionRecordFromDomain(*definition)})
}

func (h *Handler) publishDefinition(w http.ResponseWriter, r *http.Request) {
	definition, err := h.service.PublishDefinition(r.Context(), shared.ID(strings.TrimSpace(r.PathValue("id"))), time.Now().UTC())
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": definitionRecordFromDomain(*definition)})
}

func (h *Handler) startInstance(w http.ResponseWriter, r *http.Request) {
	var req startInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	instance, err := h.service.Start(r.Context(), workflowsvc.StartInput{
		ID:           shared.ID(strings.TrimSpace(req.ID)),
		DefinitionID: shared.ID(strings.TrimSpace(req.DefinitionID)),
		BusinessType: req.BusinessType,
		BusinessID:   req.BusinessID,
		Title:        req.Title,
		Starter:      req.Starter.domain(),
		Variables:    workflowVariables(req.Variables),
		Now:          time.Now().UTC(),
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": instanceRecordFromDomain(*instance)})
}

func (h *Handler) getInstance(w http.ResponseWriter, r *http.Request) {
	instance, err := h.service.GetInstance(r.Context(), shared.ID(strings.TrimSpace(r.PathValue("id"))))
	if err != nil {
		apiv1.WriteError(w, http.StatusNotFound, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": instanceRecordFromDomain(*instance)})
}

func (h *Handler) approveTask(w http.ResponseWriter, r *http.Request) {
	h.taskAction(w, r, h.service.Approve)
}

func (h *Handler) rejectTask(w http.ResponseWriter, r *http.Request) {
	h.taskAction(w, r, h.service.Reject)
}

func (h *Handler) withdrawInstance(w http.ResponseWriter, r *http.Request) {
	var req taskActionInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	instance, err := h.service.Withdraw(r.Context(), workflowsvc.InstanceActionInput{
		InstanceID: shared.ID(strings.TrimSpace(r.PathValue("id"))),
		Actor:      req.Actor.domain(),
		Comment:    req.Comment,
		Now:        time.Now().UTC(),
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": instanceRecordFromDomain(*instance)})
}

func (h *Handler) delegateTask(w http.ResponseWriter, r *http.Request) {
	h.targetAction(w, r, h.service.Delegate)
}

func (h *Handler) copyTask(w http.ResponseWriter, r *http.Request) {
	h.targetAction(w, r, h.service.Copy)
}

func (h *Handler) createSubstitution(w http.ResponseWriter, r *http.Request) {
	var req substitutionInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.CreateSubstitution(r.Context(), workflowsvc.CreateSubstitutionInput{
		ID: shared.ID(strings.TrimSpace(req.ID)), Principal: req.Principal.domain(), Substitute: req.Substitute.domain(),
		StartsAt: req.StartsAt, EndsAt: req.EndsAt, CreatedBy: req.Principal.domain(), Reason: req.Reason, Now: time.Now().UTC(),
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": substitutionRecordFromDomain(*item)})
}

func (h *Handler) revokeSubstitution(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Principal actorInput `json:"principal"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.RevokeSubstitution(r.Context(), workflowsvc.RevokeSubstitutionInput{
		ID: shared.ID(strings.TrimSpace(r.PathValue("id"))), Principal: req.Principal.domain(), Now: time.Now().UTC(),
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": substitutionRecordFromDomain(*item)})
}

func (h *Handler) taskAction(w http.ResponseWriter, r *http.Request, action func(context.Context, workflowsvc.TaskActionInput) (*domainworkflow.Instance, error)) {
	var req taskActionInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	actor := req.Actor.domain()
	if claims, ok := security.JWTClaimsFromContext(r.Context()); ok && strings.TrimSpace(claims.Subject) != "" {
		actor.ID = shared.ID(strings.TrimSpace(claims.Subject))
	}
	input := workflowsvc.TaskActionInput{
		InstanceID: shared.ID(strings.TrimSpace(r.PathValue("id"))),
		TaskID:     shared.ID(strings.TrimSpace(r.PathValue("taskId"))),
		Actor:      actor,
		Comment:    req.Comment,
		Now:        time.Now().UTC(),
	}
	if req.Signature != nil {
		input.Signature = &workflowsvc.DecisionSignatureInput{
			Proof: req.Signature.Proof, Meaning: req.Signature.Meaning, Audience: "core",
			EvidenceIDs: stringIDs(req.Signature.EvidenceIDs),
		}
	}
	instance, err := action(r.Context(), input)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": instanceRecordFromDomain(*instance)})
}

func (h *Handler) targetAction(w http.ResponseWriter, r *http.Request, action func(context.Context, workflowsvc.TaskTargetActionInput) (*domainworkflow.Instance, error)) {
	var req targetActionInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	instance, err := action(r.Context(), workflowsvc.TaskTargetActionInput{
		InstanceID: shared.ID(strings.TrimSpace(r.PathValue("id"))),
		TaskID:     shared.ID(strings.TrimSpace(r.PathValue("taskId"))),
		Actor:      req.Actor.domain(),
		Target:     req.Target.domain(),
		Comment:    req.Comment,
		Now:        time.Now().UTC(),
	})
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": instanceRecordFromDomain(*instance)})
}

func (in definitionInput) nodes() []domainworkflow.Node {
	nodes := make([]domainworkflow.Node, 0, len(in.Nodes))
	for _, node := range in.Nodes {
		assignees := make([]shared.ID, 0, len(node.Assignees))
		for _, assignee := range node.Assignees {
			assignees = append(assignees, shared.ID(strings.TrimSpace(assignee)))
		}
		nodes = append(nodes, domainworkflow.Node{
			ID:         shared.ID(strings.TrimSpace(node.ID)),
			Key:        node.Key,
			Name:       node.Name,
			Type:       domainworkflow.NodeType(strings.TrimSpace(node.Type)),
			Assignees:  assignees,
			Decision:   node.Decision.domain(),
			Escalation: node.Escalation.domain(),
			Signature:  node.Signature.domain(),
		})
	}
	return nodes
}

func (in definitionInput) transitions() []domainworkflow.Transition {
	transitions := make([]domainworkflow.Transition, 0, len(in.Transitions))
	for _, transition := range in.Transitions {
		transitions = append(transitions, domainworkflow.Transition{
			From:      shared.ID(strings.TrimSpace(transition.From)),
			To:        shared.ID(strings.TrimSpace(transition.To)),
			Condition: workflowCondition(transition.Condition),
		})
	}
	return transitions
}

func workflowCondition(input *conditionInput) *domainworkflow.Condition {
	if input == nil {
		return nil
	}
	condition := &domainworkflow.Condition{
		Match:      domainworkflow.ConditionMatch(strings.TrimSpace(input.Match)),
		Predicates: make([]domainworkflow.Predicate, 0, len(input.Predicates)),
	}
	for _, predicate := range input.Predicates {
		var value *domainworkflow.Value
		if predicate.Value != nil {
			value = &domainworkflow.Value{Type: domainworkflow.ValueType(strings.TrimSpace(predicate.Value.Type)), Value: predicate.Value.Value}
		}
		condition.Predicates = append(condition.Predicates, domainworkflow.Predicate{
			Field: strings.TrimSpace(predicate.Field), Operator: domainworkflow.PredicateOperator(strings.TrimSpace(predicate.Operator)), Value: value,
		})
	}
	return condition
}

func workflowVariables(input map[string]workflowValue) map[string]domainworkflow.Value {
	out := make(map[string]domainworkflow.Value, len(input))
	for key, value := range input {
		out[strings.TrimSpace(key)] = domainworkflow.Value{Type: domainworkflow.ValueType(strings.TrimSpace(value.Type)), Value: value.Value}
	}
	return out
}

func (in actorInput) domain() domainworkflow.Actor {
	return domainworkflow.Actor{ID: shared.ID(strings.TrimSpace(in.ID)), Name: strings.TrimSpace(in.Name)}
}

type definitionRecord struct {
	ID          string             `json:"id"`
	Key         string             `json:"key"`
	Name        string             `json:"name"`
	Version     int                `json:"version"`
	Status      string             `json:"status"`
	Nodes       []nodeRecord       `json:"nodes"`
	Transitions []transitionRecord `json:"transitions"`
	CreatedAt   time.Time          `json:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt"`
}

type nodeRecord struct {
	ID         string           `json:"id"`
	Key        string           `json:"key"`
	Name       string           `json:"name"`
	Type       string           `json:"type"`
	Assignees  []string         `json:"assignees"`
	Decision   *decisionRule    `json:"decision,omitempty"`
	Escalation *escalationRule  `json:"escalation,omitempty"`
	Signature  *signaturePolicy `json:"signature,omitempty"`
}

type transitionRecord struct {
	From      string          `json:"from"`
	To        string          `json:"to"`
	Condition *conditionInput `json:"condition,omitempty"`
}

type instanceRecord struct {
	ID            string                   `json:"id"`
	DefinitionID  string                   `json:"definitionId"`
	DefinitionKey string                   `json:"definitionKey"`
	BusinessType  string                   `json:"businessType"`
	BusinessID    string                   `json:"businessId"`
	Title         string                   `json:"title"`
	Status        string                   `json:"status"`
	Starter       actorRecord              `json:"starter"`
	CurrentNode   string                   `json:"currentNode"`
	ActiveNodes   []string                 `json:"activeNodes"`
	Variables     map[string]workflowValue `json:"variables"`
	Tasks         []taskRecord             `json:"tasks"`
	Timeline      []actionRecord           `json:"timeline"`
	Receipts      []signatureReceiptRecord `json:"receipts"`
	CreatedAt     time.Time                `json:"createdAt"`
	UpdatedAt     time.Time                `json:"updatedAt"`
}

type actorRecord struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type taskRecord struct {
	ID               string      `json:"id"`
	InstanceID       string      `json:"instanceId"`
	NodeID           string      `json:"nodeId"`
	Assignee         actorRecord `json:"assignee"`
	OriginalAssignee actorRecord `json:"originalAssignee"`
	Assignment       string      `json:"assignment"`
	AuthorizedBy     actorRecord `json:"authorizedBy"`
	AuthorizationID  string      `json:"authorizationId,omitempty"`
	Status           string      `json:"status"`
	CreatedAt        time.Time   `json:"createdAt"`
	CompletedAt      *time.Time  `json:"completedAt,omitempty"`
}

type actionRecord struct {
	ID         string      `json:"id"`
	Type       string      `json:"type"`
	InstanceID string      `json:"instanceId"`
	TaskID     string      `json:"taskId,omitempty"`
	NodeID     string      `json:"nodeId,omitempty"`
	Actor      actorRecord `json:"actor"`
	Target     actorRecord `json:"target"`
	Comment    string      `json:"comment,omitempty"`
	ReceiptID  string      `json:"receiptId,omitempty"`
	CreatedAt  time.Time   `json:"createdAt"`
}

type signatureReceiptRecord struct {
	ID                 string                    `json:"id"`
	ActionID           string                    `json:"actionId"`
	InstanceID         string                    `json:"instanceId"`
	DefinitionID       string                    `json:"definitionId"`
	DefinitionKey      string                    `json:"definitionKey"`
	BusinessType       string                    `json:"businessType"`
	BusinessID         string                    `json:"businessId"`
	TaskID             string                    `json:"taskId"`
	NodeID             string                    `json:"nodeId"`
	Action             string                    `json:"action"`
	Actor              actorRecord               `json:"actor"`
	Meaning            string                    `json:"meaning"`
	VerificationID     string                    `json:"verificationId"`
	VerificationMethod string                    `json:"verificationMethod"`
	VerificationAt     time.Time                 `json:"verificationAt"`
	Audience           string                    `json:"audience"`
	Evidence           []evidenceReferenceRecord `json:"evidence"`
	CommentDigest      string                    `json:"commentDigest"`
	EvidenceDigest     string                    `json:"evidenceDigest"`
	AuditCorrelationID string                    `json:"auditCorrelationId"`
	SignedAt           time.Time                 `json:"signedAt"`
}

type evidenceReferenceRecord struct {
	FileID string `json:"fileId"`
	Name   string `json:"name"`
	Hash   string `json:"hash"`
	Size   int64  `json:"size"`
	MIME   string `json:"mime"`
}

func definitionRecordFromDomain(definition domainworkflow.Definition) definitionRecord {
	nodes := make([]nodeRecord, 0, len(definition.Nodes))
	for _, node := range definition.Nodes {
		assignees := make([]string, 0, len(node.Assignees))
		for _, assignee := range node.Assignees {
			assignees = append(assignees, assignee.String())
		}
		var decision *decisionRule
		if node.Type == domainworkflow.NodeApproval {
			decision = &decisionRule{Strategy: string(node.Decision.Strategy), Quorum: node.Decision.Quorum}
		}
		nodes = append(nodes, nodeRecord{
			ID:         node.ID.String(),
			Key:        node.Key,
			Name:       node.Name,
			Type:       string(node.Type),
			Assignees:  assignees,
			Decision:   decision,
			Escalation: escalationRecord(node.Escalation),
			Signature:  signaturePolicyRecord(node.Signature),
		})
	}
	transitions := make([]transitionRecord, 0, len(definition.Transitions))
	for _, transition := range definition.Transitions {
		transitions = append(transitions, transitionRecord{
			From: transition.From.String(), To: transition.To.String(), Condition: conditionRecord(transition.Condition),
		})
	}
	return definitionRecord{
		ID:          definition.ID.String(),
		Key:         definition.Key,
		Name:        definition.Name,
		Version:     definition.Version,
		Status:      string(definition.Status),
		Nodes:       nodes,
		Transitions: transitions,
		CreatedAt:   definition.Meta.CreatedAt,
		UpdatedAt:   definition.Meta.UpdatedAt,
	}
}

func conditionRecord(condition *domainworkflow.Condition) *conditionInput {
	if condition == nil {
		return nil
	}
	out := &conditionInput{Match: string(condition.Match), Predicates: make([]predicateInput, 0, len(condition.Predicates))}
	for _, predicate := range condition.Predicates {
		var value *workflowValue
		if predicate.Value != nil {
			value = &workflowValue{Type: string(predicate.Value.Type), Value: predicate.Value.Value}
		}
		out.Predicates = append(out.Predicates, predicateInput{Field: predicate.Field, Operator: string(predicate.Operator), Value: value})
	}
	return out
}

func instanceRecordFromDomain(instance domainworkflow.Instance) instanceRecord {
	tasks := make([]taskRecord, 0, len(instance.Tasks))
	for _, task := range instance.Tasks {
		tasks = append(tasks, taskRecord{
			ID:               task.ID.String(),
			InstanceID:       task.InstanceID.String(),
			NodeID:           task.NodeID.String(),
			Assignee:         actorRecordFromDomain(task.Assignee),
			OriginalAssignee: actorRecordFromDomain(task.OriginalAssignee),
			Assignment:       string(task.Assignment), AuthorizedBy: actorRecordFromDomain(task.AuthorizedBy),
			AuthorizationID: task.AuthorizationID.String(),
			Status:          string(task.Status),
			CreatedAt:       task.CreatedAt,
			CompletedAt:     task.CompletedAt,
		})
	}
	timeline := make([]actionRecord, 0, len(instance.Timeline))
	for _, action := range instance.Timeline {
		timeline = append(timeline, actionRecord{
			ID:         action.ID.String(),
			Type:       string(action.Type),
			InstanceID: action.InstanceID.String(),
			TaskID:     action.TaskID.String(),
			NodeID:     action.NodeID.String(),
			Actor:      actorRecordFromDomain(action.Actor),
			Target:     actorRecordFromDomain(action.Target),
			Comment:    action.Comment,
			ReceiptID:  action.ReceiptID.String(),
			CreatedAt:  action.CreatedAt,
		})
	}
	receipts := make([]signatureReceiptRecord, 0, len(instance.Receipts))
	for _, receipt := range instance.Receipts {
		evidence := make([]evidenceReferenceRecord, 0, len(receipt.Evidence))
		for _, reference := range receipt.Evidence {
			evidence = append(evidence, evidenceReferenceRecord{
				FileID: reference.FileID.String(), Name: reference.Name, Hash: reference.Hash, Size: reference.Size, MIME: reference.MIME,
			})
		}
		receipts = append(receipts, signatureReceiptRecord{
			ID: receipt.ID.String(), ActionID: receipt.ActionID.String(), InstanceID: receipt.InstanceID.String(),
			DefinitionID: receipt.DefinitionID.String(), DefinitionKey: receipt.DefinitionKey,
			BusinessType: receipt.BusinessType, BusinessID: receipt.BusinessID, TaskID: receipt.TaskID.String(),
			NodeID: receipt.NodeID.String(), Action: string(receipt.Action), Actor: actorRecordFromDomain(receipt.Actor),
			Meaning: receipt.Meaning, VerificationID: receipt.VerificationID.String(), VerificationMethod: receipt.VerificationMethod,
			VerificationAt: receipt.VerificationAt, Audience: receipt.Audience, Evidence: evidence,
			CommentDigest: receipt.CommentDigest, EvidenceDigest: receipt.EvidenceDigest,
			AuditCorrelationID: receipt.AuditCorrelationID.String(), SignedAt: receipt.SignedAt,
		})
	}
	return instanceRecord{
		ID:            instance.ID.String(),
		DefinitionID:  instance.DefinitionID.String(),
		DefinitionKey: instance.DefinitionKey,
		BusinessType:  instance.BusinessType,
		BusinessID:    instance.BusinessID,
		Title:         instance.Title,
		Status:        string(instance.Status),
		Starter:       actorRecordFromDomain(instance.Starter),
		CurrentNode:   instance.CurrentNode.String(),
		ActiveNodes:   sharedIDs(instance.ActiveNodes),
		Variables:     workflowValueRecords(instance.Variables),
		Tasks:         tasks,
		Timeline:      timeline,
		Receipts:      receipts,
		CreatedAt:     instance.Meta.CreatedAt,
		UpdatedAt:     instance.Meta.UpdatedAt,
	}
}

func sharedIDs(ids []shared.ID) []string {
	out := make([]string, len(ids))
	for index, id := range ids {
		out[index] = id.String()
	}
	return out
}

func workflowValueRecords(values map[string]domainworkflow.Value) map[string]workflowValue {
	out := make(map[string]workflowValue, len(values))
	for key, value := range values {
		out[key] = workflowValue{Type: string(value.Type), Value: value.Value}
	}
	return out
}

func actorRecordFromDomain(actor domainworkflow.Actor) actorRecord {
	return actorRecord{ID: actor.ID.String(), Name: actor.Name}
}

func (input *decisionRule) domain() domainworkflow.DecisionRule {
	if input == nil {
		return domainworkflow.DecisionRule{}
	}
	return domainworkflow.DecisionRule{
		Strategy: domainworkflow.DecisionStrategy(strings.TrimSpace(input.Strategy)),
		Quorum:   input.Quorum,
	}
}

func (input *escalationRule) domain() *domainworkflow.EscalationRule {
	if input == nil {
		return nil
	}
	return &domainworkflow.EscalationRule{
		After:  time.Duration(input.AfterSeconds) * time.Second,
		Target: input.Target.domain(),
	}
}

func escalationRecord(input *domainworkflow.EscalationRule) *escalationRule {
	if input == nil {
		return nil
	}
	return &escalationRule{AfterSeconds: int64(input.After / time.Second), Target: actorInput{ID: input.Target.ID.String(), Name: input.Target.Name}}
}

func (input *signaturePolicy) domain() *domainworkflow.SignaturePolicy {
	if input == nil {
		return nil
	}
	return &domainworkflow.SignaturePolicy{Meaning: strings.TrimSpace(input.Meaning), RequireEvidence: input.RequireEvidence}
}

func signaturePolicyRecord(input *domainworkflow.SignaturePolicy) *signaturePolicy {
	if input == nil {
		return nil
	}
	return &signaturePolicy{Meaning: input.Meaning, RequireEvidence: input.RequireEvidence}
}

func stringIDs(input []string) []shared.ID {
	out := make([]shared.ID, 0, len(input))
	for _, value := range input {
		out = append(out, shared.ID(strings.TrimSpace(value)))
	}
	return out
}

func substitutionRecordFromDomain(item domainworkflow.SubstitutionWindow) map[string]any {
	return map[string]any{
		"id": item.ID.String(), "principal": actorRecordFromDomain(item.Principal), "substitute": actorRecordFromDomain(item.Substitute),
		"startsAt": item.StartsAt, "endsAt": item.EndsAt, "createdBy": actorRecordFromDomain(item.CreatedBy),
		"reason": item.Reason, "createdAt": item.CreatedAt, "revokedAt": item.RevokedAt,
	}
}

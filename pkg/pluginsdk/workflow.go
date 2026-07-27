package pluginsdk

import (
	"context"
	"time"
)

type WorkflowNodeType string

const (
	WorkflowNodeStart    WorkflowNodeType = "start"
	WorkflowNodeApproval WorkflowNodeType = "approval"
	WorkflowNodeCC       WorkflowNodeType = "cc"
	WorkflowNodeEnd      WorkflowNodeType = "end"
)

type WorkflowDefinitionStatus string
type WorkflowInstanceStatus string
type WorkflowTaskStatus string
type WorkflowActionType string
type WorkflowDecisionStrategy string
type WorkflowValueType string
type WorkflowPredicateOperator string
type WorkflowConditionMatch string
type WorkflowAssignmentKind string

const (
	WorkflowDefinitionDraft     WorkflowDefinitionStatus = "draft"
	WorkflowDefinitionPublished WorkflowDefinitionStatus = "published"
	WorkflowDefinitionDisabled  WorkflowDefinitionStatus = "disabled"

	WorkflowInstanceRunning   WorkflowInstanceStatus = "running"
	WorkflowInstanceApproved  WorkflowInstanceStatus = "approved"
	WorkflowInstanceRejected  WorkflowInstanceStatus = "rejected"
	WorkflowInstanceWithdrawn WorkflowInstanceStatus = "withdrawn"
	WorkflowInstanceCanceled  WorkflowInstanceStatus = "canceled"

	WorkflowTaskPending   WorkflowTaskStatus = "pending"
	WorkflowTaskApproved  WorkflowTaskStatus = "approved"
	WorkflowTaskRejected  WorkflowTaskStatus = "rejected"
	WorkflowTaskDelegated WorkflowTaskStatus = "delegated"
	WorkflowTaskCopied    WorkflowTaskStatus = "copied"
	WorkflowTaskCanceled  WorkflowTaskStatus = "canceled"

	WorkflowActionStart      WorkflowActionType = "start"
	WorkflowActionApprove    WorkflowActionType = "approve"
	WorkflowActionReject     WorkflowActionType = "reject"
	WorkflowActionWithdraw   WorkflowActionType = "withdraw"
	WorkflowActionDelegate   WorkflowActionType = "delegate"
	WorkflowActionSubstitute WorkflowActionType = "substitute"
	WorkflowActionEscalate   WorkflowActionType = "escalate"
	WorkflowActionCopy       WorkflowActionType = "copy"
	WorkflowActionCancel     WorkflowActionType = "cancel"

	WorkflowDecisionAny    WorkflowDecisionStrategy = "any"
	WorkflowDecisionAll    WorkflowDecisionStrategy = "all"
	WorkflowDecisionQuorum WorkflowDecisionStrategy = "quorum"

	WorkflowValueString  WorkflowValueType = "string"
	WorkflowValueNumber  WorkflowValueType = "number"
	WorkflowValueBoolean WorkflowValueType = "boolean"

	WorkflowPredicateEqual        WorkflowPredicateOperator = "eq"
	WorkflowPredicateNotEqual     WorkflowPredicateOperator = "ne"
	WorkflowPredicateGreaterThan  WorkflowPredicateOperator = "gt"
	WorkflowPredicateGreaterEqual WorkflowPredicateOperator = "gte"
	WorkflowPredicateLessThan     WorkflowPredicateOperator = "lt"
	WorkflowPredicateLessEqual    WorkflowPredicateOperator = "lte"
	WorkflowPredicateExists       WorkflowPredicateOperator = "exists"
	WorkflowPredicateNotExists    WorkflowPredicateOperator = "not_exists"

	WorkflowConditionAll WorkflowConditionMatch = "all"
	WorkflowConditionAny WorkflowConditionMatch = "any"

	WorkflowAssignmentDirect      WorkflowAssignmentKind = "direct"
	WorkflowAssignmentDelegated   WorkflowAssignmentKind = "delegated"
	WorkflowAssignmentSubstituted WorkflowAssignmentKind = "substituted"
	WorkflowAssignmentEscalated   WorkflowAssignmentKind = "escalated"
)

type WorkflowActor struct {
	ID   string
	Name string
}

type WorkflowNode struct {
	ID          string
	Key         string
	Name        string
	Type        WorkflowNodeType
	AssigneeIDs []string
	Decision    *WorkflowDecisionRule
	Escalation  *WorkflowEscalationRule
	Signature   *WorkflowSignaturePolicy
}

type WorkflowTransition struct {
	From      string
	To        string
	Condition *WorkflowCondition
}

type WorkflowDecisionRule struct {
	Strategy WorkflowDecisionStrategy
	Quorum   int
}

type WorkflowEscalationRule struct {
	AfterSeconds int64
	Target       WorkflowActor
}

type WorkflowSignaturePolicy struct {
	Meaning         string
	RequireEvidence bool
}

type WorkflowValue struct {
	Type  WorkflowValueType `json:"type"`
	Value string            `json:"value"`
}

type WorkflowPredicate struct {
	Field    string                    `json:"field"`
	Operator WorkflowPredicateOperator `json:"operator"`
	Value    *WorkflowValue            `json:"value,omitempty"`
}

type WorkflowCondition struct {
	Match      WorkflowConditionMatch `json:"match"`
	Predicates []WorkflowPredicate    `json:"predicates"`
}

type WorkflowDefinitionInput struct {
	ID          string
	Key         string
	Name        string
	Version     int
	Nodes       []WorkflowNode
	Transitions []WorkflowTransition
}

type WorkflowDefinition struct {
	ID          string
	Key         string
	Name        string
	Version     int
	Status      WorkflowDefinitionStatus
	Nodes       []WorkflowNode
	Transitions []WorkflowTransition
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type WorkflowStartInput struct {
	ID           string
	DefinitionID string
	BusinessType string
	BusinessID   string
	Title        string
	Variables    map[string]WorkflowValue
}

type WorkflowTaskActionInput struct {
	InstanceID string
	TaskID     string
	Comment    string
	Signature  *WorkflowDecisionSignature
}

type WorkflowDecisionSignature struct {
	Proof       string
	Meaning     string
	EvidenceIDs []string
}

type WorkflowInstanceActionInput struct {
	InstanceID string
	Comment    string
}

type WorkflowTargetActionInput struct {
	InstanceID string
	TaskID     string
	Target     WorkflowActor
	Comment    string
}

type WorkflowSubstitutionInput struct {
	ID         string
	Substitute WorkflowActor
	StartsAt   time.Time
	EndsAt     time.Time
	Reason     string
}

type WorkflowSubstitution struct {
	ID         string
	Principal  WorkflowActor
	Substitute WorkflowActor
	StartsAt   time.Time
	EndsAt     time.Time
	CreatedBy  WorkflowActor
	Reason     string
	CreatedAt  time.Time
	RevokedAt  *time.Time
}

type WorkflowTask struct {
	ID               string
	InstanceID       string
	NodeID           string
	Assignee         WorkflowActor
	OriginalAssignee WorkflowActor
	Assignment       WorkflowAssignmentKind
	AuthorizedBy     WorkflowActor
	AuthorizationID  string
	Status           WorkflowTaskStatus
	CreatedAt        time.Time
	CompletedAt      *time.Time
}

type WorkflowAction struct {
	ID         string
	Type       WorkflowActionType
	InstanceID string
	TaskID     string
	NodeID     string
	Actor      WorkflowActor
	Target     WorkflowActor
	Comment    string
	ReceiptID  string
	CreatedAt  time.Time
}

type WorkflowEvidenceReference struct {
	FileID string
	Name   string
	Hash   string
	Size   int64
	MIME   string
}

type WorkflowSignatureReceipt struct {
	ID                 string
	ActionID           string
	InstanceID         string
	DefinitionID       string
	DefinitionKey      string
	BusinessType       string
	BusinessID         string
	TaskID             string
	NodeID             string
	Action             WorkflowActionType
	Actor              WorkflowActor
	Meaning            string
	VerificationID     string
	VerificationMethod string
	VerificationAt     time.Time
	Audience           string
	Evidence           []WorkflowEvidenceReference
	CommentDigest      string
	EvidenceDigest     string
	AuditCorrelationID string
	SignedAt           time.Time
}

type WorkflowInstance struct {
	ID            string
	DefinitionID  string
	DefinitionKey string
	BusinessType  string
	BusinessID    string
	Title         string
	Status        WorkflowInstanceStatus
	Starter       WorkflowActor
	CurrentNode   string
	ActiveNodes   []string
	Variables     map[string]WorkflowValue
	Tasks         []WorkflowTask
	Timeline      []WorkflowAction
	Receipts      []WorkflowSignatureReceipt
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type WorkflowService interface {
	CreateDefinition(ctx context.Context, input WorkflowDefinitionInput) (WorkflowDefinition, error)
	GetDefinition(ctx context.Context, id string) (WorkflowDefinition, error)
	PublishDefinition(ctx context.Context, id string) (WorkflowDefinition, error)
	Start(ctx context.Context, input WorkflowStartInput) (WorkflowInstance, error)
	GetInstance(ctx context.Context, id string) (WorkflowInstance, error)
	Approve(ctx context.Context, input WorkflowTaskActionInput) (WorkflowInstance, error)
	Reject(ctx context.Context, input WorkflowTaskActionInput) (WorkflowInstance, error)
	Withdraw(ctx context.Context, input WorkflowInstanceActionInput) (WorkflowInstance, error)
	Cancel(ctx context.Context, input WorkflowInstanceActionInput) (WorkflowInstance, error)
	Delegate(ctx context.Context, input WorkflowTargetActionInput) (WorkflowInstance, error)
	Copy(ctx context.Context, input WorkflowTargetActionInput) (WorkflowInstance, error)
	CreateSubstitution(ctx context.Context, input WorkflowSubstitutionInput) (WorkflowSubstitution, error)
	RevokeSubstitution(ctx context.Context, id string) (WorkflowSubstitution, error)
}

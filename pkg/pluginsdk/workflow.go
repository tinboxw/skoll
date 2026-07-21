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

const (
	WorkflowDefinitionDraft     WorkflowDefinitionStatus = "draft"
	WorkflowDefinitionPublished WorkflowDefinitionStatus = "published"
	WorkflowDefinitionDisabled  WorkflowDefinitionStatus = "disabled"

	WorkflowInstanceRunning   WorkflowInstanceStatus = "running"
	WorkflowInstanceApproved  WorkflowInstanceStatus = "approved"
	WorkflowInstanceRejected  WorkflowInstanceStatus = "rejected"
	WorkflowInstanceWithdrawn WorkflowInstanceStatus = "withdrawn"

	WorkflowTaskPending     WorkflowTaskStatus = "pending"
	WorkflowTaskApproved    WorkflowTaskStatus = "approved"
	WorkflowTaskRejected    WorkflowTaskStatus = "rejected"
	WorkflowTaskTransferred WorkflowTaskStatus = "transferred"
	WorkflowTaskCopied      WorkflowTaskStatus = "copied"
	WorkflowTaskCanceled    WorkflowTaskStatus = "canceled"

	WorkflowActionStart    WorkflowActionType = "start"
	WorkflowActionApprove  WorkflowActionType = "approve"
	WorkflowActionReject   WorkflowActionType = "reject"
	WorkflowActionWithdraw WorkflowActionType = "withdraw"
	WorkflowActionTransfer WorkflowActionType = "transfer"
	WorkflowActionCopy     WorkflowActionType = "copy"
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
}

type WorkflowTransition struct {
	From string
	To   string
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
}

type WorkflowTaskActionInput struct {
	InstanceID string
	TaskID     string
	Comment    string
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

type WorkflowTask struct {
	ID          string
	InstanceID  string
	NodeID      string
	Assignee    WorkflowActor
	Status      WorkflowTaskStatus
	CreatedAt   time.Time
	CompletedAt *time.Time
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
	CreatedAt  time.Time
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
	Tasks         []WorkflowTask
	Timeline      []WorkflowAction
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
	Transfer(ctx context.Context, input WorkflowTargetActionInput) (WorkflowInstance, error)
	Copy(ctx context.Context, input WorkflowTargetActionInput) (WorkflowInstance, error)
}

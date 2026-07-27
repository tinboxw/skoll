package gormrepo

import "time"

type WorkflowDefinitionModel struct {
	ID            string `gorm:"size:64;primaryKey"`
	DefinitionKey string `gorm:"size:128;uniqueIndex:idx_workflow_definition_key_version,priority:1"`
	Name          string `gorm:"size:255"`
	Version       int    `gorm:"uniqueIndex:idx_workflow_definition_key_version,priority:2"`
	Status        string `gorm:"size:32;index:idx_workflow_definition_status_updated,priority:1"`
	CreatedAt     time.Time
	UpdatedAt     time.Time `gorm:"index:idx_workflow_definition_status_updated,priority:2"`
}

func (WorkflowDefinitionModel) TableName() string { return "sk_workflow_definitions" }

type WorkflowNodeModel struct {
	DefinitionID     string `gorm:"size:64;primaryKey;index:idx_workflow_node_definition_position,priority:1"`
	NodeID           string `gorm:"size:64;primaryKey"`
	NodeKey          string `gorm:"size:128"`
	Name             string `gorm:"size:255"`
	NodeType         string `gorm:"size:32"`
	DecisionStrategy string `gorm:"size:32"`
	DecisionQuorum   int
	Position         int `gorm:"index:idx_workflow_node_definition_position,priority:2"`
}

func (WorkflowNodeModel) TableName() string { return "sk_workflow_nodes" }

type WorkflowNodeAssigneeModel struct {
	DefinitionID string `gorm:"size:64;primaryKey"`
	NodeID       string `gorm:"size:64;primaryKey"`
	Position     int    `gorm:"primaryKey"`
	AssigneeID   string `gorm:"size:64;index:idx_workflow_node_assignee"`
}

func (WorkflowNodeAssigneeModel) TableName() string { return "sk_workflow_node_assignees" }

type WorkflowTransitionModel struct {
	DefinitionID  string `gorm:"size:64;primaryKey"`
	Position      int    `gorm:"primaryKey"`
	FromNodeID    string `gorm:"size:64"`
	ToNodeID      string `gorm:"size:64"`
	ConditionJSON string `gorm:"type:text"`
}

func (WorkflowTransitionModel) TableName() string { return "sk_workflow_transitions" }

type WorkflowInstanceModel struct {
	ID                string `gorm:"size:64;primaryKey"`
	DefinitionID      string `gorm:"size:64;index:idx_workflow_instance_definition_status,priority:1"`
	DefinitionKey     string `gorm:"size:128"`
	BusinessType      string `gorm:"size:128;index:idx_workflow_instance_business,priority:1"`
	BusinessID        string `gorm:"size:128;index:idx_workflow_instance_business,priority:2"`
	Title             string `gorm:"size:255"`
	Status            string `gorm:"size:32;index:idx_workflow_instance_definition_status,priority:2;index:idx_workflow_instance_starter_status,priority:2;index:idx_workflow_instance_status_updated,priority:1"`
	StarterID         string `gorm:"size:64;index:idx_workflow_instance_starter_status,priority:1"`
	StarterName       string `gorm:"size:255"`
	CurrentNodeID     string `gorm:"size:64"`
	ActiveNodeIDsJSON string `gorm:"type:text"`
	VariablesJSON     string `gorm:"type:text"`
	CreatedAt         time.Time
	UpdatedAt         time.Time `gorm:"index:idx_workflow_instance_status_updated,priority:2"`
}

func (WorkflowInstanceModel) TableName() string { return "sk_workflow_instances" }

type WorkflowTaskModel struct {
	ID           string `gorm:"size:128;primaryKey"`
	InstanceID   string `gorm:"size:64;index:idx_workflow_task_instance_position,priority:1;index:idx_workflow_task_instance_status,priority:1"`
	Position     int    `gorm:"index:idx_workflow_task_instance_position,priority:2"`
	NodeID       string `gorm:"size:64"`
	AssigneeID   string `gorm:"size:64;index:idx_workflow_task_assignee_status,priority:1"`
	AssigneeName string `gorm:"size:255"`
	Status       string `gorm:"size:32;index:idx_workflow_task_instance_status,priority:2;index:idx_workflow_task_assignee_status,priority:2"`
	CreatedAt    time.Time
	CompletedAt  *time.Time
}

func (WorkflowTaskModel) TableName() string { return "sk_workflow_tasks" }

type WorkflowActionModel struct {
	ID         string    `gorm:"size:128;primaryKey"`
	InstanceID string    `gorm:"size:64;index:idx_workflow_action_instance_position,priority:1;index:idx_workflow_action_instance_created,priority:1"`
	Position   int       `gorm:"index:idx_workflow_action_instance_position,priority:2"`
	ActionType string    `gorm:"size:32"`
	TaskID     string    `gorm:"size:128;index"`
	NodeID     string    `gorm:"size:64"`
	ActorID    string    `gorm:"size:64;index:idx_workflow_action_actor_created,priority:1"`
	ActorName  string    `gorm:"size:255"`
	TargetID   string    `gorm:"size:64"`
	TargetName string    `gorm:"size:255"`
	Comment    string    `gorm:"type:text"`
	CreatedAt  time.Time `gorm:"index:idx_workflow_action_instance_created,priority:2;index:idx_workflow_action_actor_created,priority:2"`
}

func (WorkflowActionModel) TableName() string { return "sk_workflow_actions" }

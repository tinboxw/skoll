package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WorkflowStore struct {
	db *gorm.DB
}

func NewWorkflowStore(db *gorm.DB) *WorkflowStore {
	return &WorkflowStore{db: db}
}

func (s *WorkflowStore) SaveDefinition(ctx context.Context, definition domainworkflow.Definition) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("workflow repository is required")
	}
	if err := definition.Validate(); err != nil {
		return err
	}
	row := workflowDefinitionRow(definition)
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"definition_key", "name", "version", "status", "updated_at",
				}),
			}).Create(&row).Error; err != nil {
				return err
			}
			if err := tx.Where("definition_id = ?", row.ID).Delete(&WorkflowNodeAssigneeModel{}).Error; err != nil {
				return err
			}
			if err := tx.Where("definition_id = ?", row.ID).Delete(&WorkflowTransitionModel{}).Error; err != nil {
				return err
			}
			if err := tx.Where("definition_id = ?", row.ID).Delete(&WorkflowNodeModel{}).Error; err != nil {
				return err
			}
			nodes, assignees := workflowNodeRows(definition)
			if len(nodes) > 0 {
				if err := tx.Create(&nodes).Error; err != nil {
					return err
				}
			}
			if len(assignees) > 0 {
				if err := tx.Create(&assignees).Error; err != nil {
					return err
				}
			}
			transitions := workflowTransitionRows(definition)
			if len(transitions) > 0 {
				return tx.Create(&transitions).Error
			}
			return nil
		})
	})
}

func (s *WorkflowStore) GetDefinition(ctx context.Context, id shared.ID) (*domainworkflow.Definition, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("workflow repository is required")
	}
	var row WorkflowDefinitionModel
	var nodes []WorkflowNodeModel
	var assignees []WorkflowNodeAssigneeModel
	var transitions []WorkflowTransitionModel
	if err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("id = ?", strings.TrimSpace(id.String())).First(&row).Error; err != nil {
				return err
			}
			if err := tx.Where("definition_id = ?", row.ID).Order("position ASC").Find(&nodes).Error; err != nil {
				return err
			}
			if err := tx.Where("definition_id = ?", row.ID).Order("node_id ASC").Order("position ASC").Find(&assignees).Error; err != nil {
				return err
			}
			return tx.Where("definition_id = ?", row.ID).Order("position ASC").Find(&transitions).Error
		})
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("workflow definition not found")
		}
		return nil, err
	}

	definition := workflowDefinitionFromRows(row, nodes, assignees, transitions)
	if err := definition.Validate(); err != nil {
		return nil, fmt.Errorf("restore workflow definition: %w", err)
	}
	return &definition, nil
}

func (s *WorkflowStore) SaveInstance(ctx context.Context, instance domainworkflow.Instance) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("workflow repository is required")
	}
	if err := validateWorkflowInstanceForStorage(instance); err != nil {
		return err
	}
	row := workflowInstanceRow(instance)
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"definition_id", "definition_key", "business_type", "business_id", "title", "status",
					"starter_id", "starter_name", "current_node_id", "updated_at",
				}),
			}).Create(&row).Error; err != nil {
				return err
			}
			if err := tx.Where("instance_id = ?", row.ID).Delete(&WorkflowActionModel{}).Error; err != nil {
				return err
			}
			if err := tx.Where("instance_id = ?", row.ID).Delete(&WorkflowTaskModel{}).Error; err != nil {
				return err
			}
			tasks := workflowTaskRows(instance)
			if len(tasks) > 0 {
				if err := tx.Create(&tasks).Error; err != nil {
					return err
				}
			}
			actions := workflowActionRows(instance)
			if len(actions) > 0 {
				return tx.Create(&actions).Error
			}
			return nil
		})
	})
}

func (s *WorkflowStore) GetInstance(ctx context.Context, id shared.ID) (*domainworkflow.Instance, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("workflow repository is required")
	}
	var row WorkflowInstanceModel
	var tasks []WorkflowTaskModel
	var actions []WorkflowActionModel
	if err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("id = ?", strings.TrimSpace(id.String())).First(&row).Error; err != nil {
				return err
			}
			if err := tx.Where("instance_id = ?", row.ID).Order("position ASC").Find(&tasks).Error; err != nil {
				return err
			}
			return tx.Where("instance_id = ?", row.ID).Order("position ASC").Find(&actions).Error
		})
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("workflow instance not found")
		}
		return nil, err
	}
	instance := workflowInstanceFromRows(row, tasks, actions)
	if err := validateWorkflowInstanceForStorage(instance); err != nil {
		return nil, fmt.Errorf("restore workflow instance: %w", err)
	}
	return &instance, nil
}

func workflowDefinitionRow(definition domainworkflow.Definition) WorkflowDefinitionModel {
	return WorkflowDefinitionModel{
		ID: definition.ID.String(), DefinitionKey: strings.TrimSpace(definition.Key), Name: strings.TrimSpace(definition.Name),
		Version: definition.Version, Status: string(definition.Status), CreatedAt: definition.Meta.CreatedAt, UpdatedAt: definition.Meta.UpdatedAt,
	}
}

func workflowNodeRows(definition domainworkflow.Definition) ([]WorkflowNodeModel, []WorkflowNodeAssigneeModel) {
	nodes := make([]WorkflowNodeModel, 0, len(definition.Nodes))
	assignees := make([]WorkflowNodeAssigneeModel, 0)
	for position, node := range definition.Nodes {
		nodes = append(nodes, WorkflowNodeModel{
			DefinitionID: definition.ID.String(), NodeID: node.ID.String(), NodeKey: strings.TrimSpace(node.Key),
			Name: strings.TrimSpace(node.Name), NodeType: string(node.Type), Position: position,
		})
		for assigneePosition, assigneeID := range node.Assignees {
			assignees = append(assignees, WorkflowNodeAssigneeModel{
				DefinitionID: definition.ID.String(), NodeID: node.ID.String(), Position: assigneePosition, AssigneeID: assigneeID.String(),
			})
		}
	}
	return nodes, assignees
}

func workflowTransitionRows(definition domainworkflow.Definition) []WorkflowTransitionModel {
	rows := make([]WorkflowTransitionModel, 0, len(definition.Transitions))
	for position, transition := range definition.Transitions {
		rows = append(rows, WorkflowTransitionModel{
			DefinitionID: definition.ID.String(), Position: position, FromNodeID: transition.From.String(), ToNodeID: transition.To.String(),
		})
	}
	return rows
}

func workflowDefinitionFromRows(row WorkflowDefinitionModel, nodes []WorkflowNodeModel, assignees []WorkflowNodeAssigneeModel, transitions []WorkflowTransitionModel) domainworkflow.Definition {
	assigneesByNode := make(map[string][]shared.ID, len(nodes))
	for _, item := range assignees {
		assigneesByNode[item.NodeID] = append(assigneesByNode[item.NodeID], shared.ID(item.AssigneeID))
	}
	definition := domainworkflow.Definition{
		ID: shared.ID(row.ID), Key: row.DefinitionKey, Name: row.Name, Version: row.Version, Status: domainworkflow.DefinitionStatus(row.Status),
		Nodes: make([]domainworkflow.Node, 0, len(nodes)), Transitions: make([]domainworkflow.Transition, 0, len(transitions)),
		Meta: shared.AuditMeta{CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt},
	}
	for _, node := range nodes {
		definition.Nodes = append(definition.Nodes, domainworkflow.Node{
			ID: shared.ID(node.NodeID), Key: node.NodeKey, Name: node.Name, Type: domainworkflow.NodeType(node.NodeType),
			Assignees: assigneesByNode[node.NodeID],
		})
	}
	for _, transition := range transitions {
		definition.Transitions = append(definition.Transitions, domainworkflow.Transition{From: shared.ID(transition.FromNodeID), To: shared.ID(transition.ToNodeID)})
	}
	return definition
}

func workflowInstanceRow(instance domainworkflow.Instance) WorkflowInstanceModel {
	return WorkflowInstanceModel{
		ID: instance.ID.String(), DefinitionID: instance.DefinitionID.String(), DefinitionKey: strings.TrimSpace(instance.DefinitionKey),
		BusinessType: strings.TrimSpace(instance.BusinessType), BusinessID: strings.TrimSpace(instance.BusinessID), Title: strings.TrimSpace(instance.Title),
		Status: string(instance.Status), StarterID: instance.Starter.ID.String(), StarterName: strings.TrimSpace(instance.Starter.Name),
		CurrentNodeID: instance.CurrentNode.String(), CreatedAt: instance.Meta.CreatedAt, UpdatedAt: instance.Meta.UpdatedAt,
	}
}

func workflowTaskRows(instance domainworkflow.Instance) []WorkflowTaskModel {
	rows := make([]WorkflowTaskModel, 0, len(instance.Tasks))
	for position, task := range instance.Tasks {
		rows = append(rows, WorkflowTaskModel{
			ID: task.ID.String(), InstanceID: instance.ID.String(), Position: position, NodeID: task.NodeID.String(),
			AssigneeID: task.Assignee.ID.String(), AssigneeName: strings.TrimSpace(task.Assignee.Name), Status: string(task.Status),
			CreatedAt: task.CreatedAt, CompletedAt: task.CompletedAt,
		})
	}
	return rows
}

func workflowActionRows(instance domainworkflow.Instance) []WorkflowActionModel {
	rows := make([]WorkflowActionModel, 0, len(instance.Timeline))
	for position, action := range instance.Timeline {
		rows = append(rows, WorkflowActionModel{
			ID: action.ID.String(), InstanceID: instance.ID.String(), Position: position, ActionType: string(action.Type),
			TaskID: action.TaskID.String(), NodeID: action.NodeID.String(), ActorID: action.Actor.ID.String(), ActorName: strings.TrimSpace(action.Actor.Name),
			TargetID: action.Target.ID.String(), TargetName: strings.TrimSpace(action.Target.Name), Comment: strings.TrimSpace(action.Comment), CreatedAt: action.CreatedAt,
		})
	}
	return rows
}

func workflowInstanceFromRows(row WorkflowInstanceModel, tasks []WorkflowTaskModel, actions []WorkflowActionModel) domainworkflow.Instance {
	instance := domainworkflow.Instance{
		ID: shared.ID(row.ID), DefinitionID: shared.ID(row.DefinitionID), DefinitionKey: row.DefinitionKey,
		BusinessType: row.BusinessType, BusinessID: row.BusinessID, Title: row.Title, Status: domainworkflow.InstanceStatus(row.Status),
		Starter: domainworkflow.Actor{ID: shared.ID(row.StarterID), Name: row.StarterName}, CurrentNode: shared.ID(row.CurrentNodeID),
		Tasks: make([]domainworkflow.Task, 0, len(tasks)), Timeline: make([]domainworkflow.Action, 0, len(actions)),
		Meta: shared.AuditMeta{CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt},
	}
	for _, task := range tasks {
		instance.Tasks = append(instance.Tasks, domainworkflow.Task{
			ID: shared.ID(task.ID), InstanceID: shared.ID(task.InstanceID), NodeID: shared.ID(task.NodeID),
			Assignee: domainworkflow.Actor{ID: shared.ID(task.AssigneeID), Name: task.AssigneeName}, Status: domainworkflow.TaskStatus(task.Status),
			CreatedAt: task.CreatedAt, CompletedAt: task.CompletedAt,
		})
	}
	for _, action := range actions {
		instance.Timeline = append(instance.Timeline, domainworkflow.Action{
			ID: shared.ID(action.ID), Type: domainworkflow.ActionType(action.ActionType), InstanceID: shared.ID(action.InstanceID),
			TaskID: shared.ID(action.TaskID), NodeID: shared.ID(action.NodeID), Actor: domainworkflow.Actor{ID: shared.ID(action.ActorID), Name: action.ActorName},
			Target: domainworkflow.Actor{ID: shared.ID(action.TargetID), Name: action.TargetName}, Comment: action.Comment, CreatedAt: action.CreatedAt,
		})
	}
	return instance
}

func validateWorkflowInstanceForStorage(instance domainworkflow.Instance) error {
	if instance.ID.IsZero() || instance.DefinitionID.IsZero() || strings.TrimSpace(instance.DefinitionKey) == "" {
		return fmt.Errorf("workflow instance identity is incomplete")
	}
	if strings.TrimSpace(instance.BusinessType) == "" || strings.TrimSpace(instance.BusinessID) == "" || strings.TrimSpace(instance.Title) == "" {
		return fmt.Errorf("workflow instance business identity is incomplete")
	}
	if instance.Starter.ID.IsZero() || instance.CurrentNode.IsZero() {
		return fmt.Errorf("workflow instance actor or current node is incomplete")
	}
	for _, task := range instance.Tasks {
		if task.ID.IsZero() || task.InstanceID != instance.ID || task.NodeID.IsZero() || task.Assignee.ID.IsZero() || task.CreatedAt.IsZero() {
			return fmt.Errorf("workflow task is incomplete")
		}
	}
	for _, action := range instance.Timeline {
		if action.ID.IsZero() || action.InstanceID != instance.ID || action.NodeID.IsZero() || action.Actor.ID.IsZero() || action.CreatedAt.IsZero() {
			return fmt.Errorf("workflow action is incomplete")
		}
	}
	return nil
}

var _ workflowsvc.Repository = (*WorkflowStore)(nil)

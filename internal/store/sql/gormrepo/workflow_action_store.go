package gormrepo

import (
	"fmt"

	"gorm.io/gorm"
)

func appendWorkflowActions(tx *gorm.DB, rows []WorkflowActionModel) error {
	if len(rows) == 0 {
		return nil
	}
	instanceID := rows[0].InstanceID
	incoming := make(map[string]WorkflowActionModel, len(rows))
	for _, row := range rows {
		if row.ID == "" || row.InstanceID != instanceID {
			return fmt.Errorf("workflow action history is invalid")
		}
		if _, exists := incoming[row.ID]; exists {
			return fmt.Errorf("workflow action history contains duplicate action")
		}
		incoming[row.ID] = row
	}

	var stored []WorkflowActionModel
	if err := tx.Where("instance_id = ?", instanceID).Order("position ASC").Find(&stored).Error; err != nil {
		return err
	}
	storedByID := make(map[string]WorkflowActionModel, len(stored))
	for _, existing := range stored {
		candidate, exists := incoming[existing.ID]
		if !exists {
			return fmt.Errorf("workflow action history is append-only")
		}
		if !workflowActionRowsEqual(existing, candidate) {
			return fmt.Errorf("workflow action history is immutable")
		}
		storedByID[existing.ID] = existing
	}
	for _, row := range rows {
		if _, exists := storedByID[row.ID]; exists {
			continue
		}
		if err := tx.Create(&row).Error; err != nil {
			return fmt.Errorf("append workflow action: %w", err)
		}
	}
	return nil
}

func workflowActionRowsEqual(left, right WorkflowActionModel) bool {
	return left.ID == right.ID &&
		left.InstanceID == right.InstanceID &&
		left.Position == right.Position &&
		left.ActionType == right.ActionType &&
		left.TaskID == right.TaskID &&
		left.NodeID == right.NodeID &&
		left.ActorID == right.ActorID &&
		left.ActorName == right.ActorName &&
		left.TargetID == right.TargetID &&
		left.TargetName == right.TargetName &&
		left.Comment == right.Comment &&
		left.ReceiptID == right.ReceiptID &&
		left.CreatedAt.Equal(right.CreatedAt)
}

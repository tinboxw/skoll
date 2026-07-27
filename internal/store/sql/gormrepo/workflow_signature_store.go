package gormrepo

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
	"gorm.io/gorm"
)

func appendWorkflowSignatureReceipts(tx *gorm.DB, receipts []domainworkflow.SignatureReceipt) error {
	for _, receipt := range receipts {
		if err := receipt.VerifyDigest(); err != nil {
			return err
		}
		row, err := workflowSignatureReceiptRow(receipt)
		if err != nil {
			return err
		}
		var existing WorkflowSignatureReceiptModel
		err = tx.Where("id = ?", row.ID).First(&existing).Error
		if err == nil {
			stored, restoreErr := workflowSignatureReceiptFromRow(existing)
			if restoreErr != nil {
				return restoreErr
			}
			if stored.EvidenceDigest != receipt.EvidenceDigest {
				return fmt.Errorf("workflow signature receipt is immutable")
			}
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Create(&row).Error; err != nil {
			return fmt.Errorf("append workflow signature receipt: %w", err)
		}
	}
	return nil
}

func workflowSignatureReceiptRow(receipt domainworkflow.SignatureReceipt) (WorkflowSignatureReceiptModel, error) {
	evidence, err := json.Marshal(receipt.Evidence)
	if err != nil {
		return WorkflowSignatureReceiptModel{}, fmt.Errorf("encode workflow signature evidence: %w", err)
	}
	return WorkflowSignatureReceiptModel{
		ID: receipt.ID.String(), ActionID: receipt.ActionID.String(), InstanceID: receipt.InstanceID.String(),
		DefinitionID: receipt.DefinitionID.String(), DefinitionKey: strings.TrimSpace(receipt.DefinitionKey),
		BusinessType: strings.TrimSpace(receipt.BusinessType), BusinessID: strings.TrimSpace(receipt.BusinessID),
		TaskID: receipt.TaskID.String(), NodeID: receipt.NodeID.String(), ActionType: string(receipt.Action),
		ActorID: receipt.Actor.ID.String(), ActorName: strings.TrimSpace(receipt.Actor.Name), Meaning: strings.TrimSpace(receipt.Meaning),
		VerificationID: receipt.VerificationID.String(), VerificationMethod: strings.TrimSpace(receipt.VerificationMethod),
		VerificationAt: receipt.VerificationAt, Audience: strings.TrimSpace(receipt.Audience), EvidenceJSON: string(evidence),
		CommentDigest: strings.TrimSpace(receipt.CommentDigest), EvidenceDigest: strings.TrimSpace(receipt.EvidenceDigest),
		AuditCorrelationID: receipt.AuditCorrelationID.String(), SignedAt: receipt.SignedAt,
	}, nil
}

func workflowSignatureReceiptFromRow(row WorkflowSignatureReceiptModel) (domainworkflow.SignatureReceipt, error) {
	evidence := make([]domainworkflow.EvidenceReference, 0)
	if err := json.Unmarshal([]byte(row.EvidenceJSON), &evidence); err != nil {
		return domainworkflow.SignatureReceipt{}, fmt.Errorf("decode workflow signature evidence: %w", err)
	}
	receipt := domainworkflow.SignatureReceipt{
		ID: shared.ID(row.ID), ActionID: shared.ID(row.ActionID), InstanceID: shared.ID(row.InstanceID),
		DefinitionID: shared.ID(row.DefinitionID), DefinitionKey: row.DefinitionKey,
		BusinessType: row.BusinessType, BusinessID: row.BusinessID, TaskID: shared.ID(row.TaskID),
		NodeID: shared.ID(row.NodeID), Action: domainworkflow.ActionType(row.ActionType),
		Actor: domainworkflow.Actor{ID: shared.ID(row.ActorID), Name: row.ActorName}, Meaning: row.Meaning,
		VerificationID: shared.ID(row.VerificationID), VerificationMethod: row.VerificationMethod,
		VerificationAt: row.VerificationAt, Audience: row.Audience, Evidence: evidence,
		CommentDigest: row.CommentDigest, EvidenceDigest: row.EvidenceDigest,
		AuditCorrelationID: shared.ID(row.AuditCorrelationID), SignedAt: row.SignedAt,
	}
	if err := receipt.VerifyDigest(); err != nil {
		return domainworkflow.SignatureReceipt{}, err
	}
	return receipt, nil
}

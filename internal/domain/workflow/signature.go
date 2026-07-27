package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

const (
	maxSignatureMeaningBytes = 512
	maxSignatureEvidence     = 16
)

type SignaturePolicy struct {
	Meaning         string
	RequireEvidence bool
}

func (p SignaturePolicy) Validate() error {
	meaning := strings.TrimSpace(p.Meaning)
	if meaning == "" || len(meaning) > maxSignatureMeaningBytes {
		return fmt.Errorf("workflow signature meaning is invalid")
	}
	return nil
}

type EvidenceReference struct {
	FileID shared.ID
	Name   string
	Hash   string
	Size   int64
	MIME   string
}

type DecisionSignature struct {
	VerificationID shared.ID
	Audience       string
	Method         string
	Meaning        string
	Evidence       []EvidenceReference
	VerifiedAt     time.Time
}

type SignatureReceipt struct {
	ID                 shared.ID
	ActionID           shared.ID
	InstanceID         shared.ID
	DefinitionID       shared.ID
	DefinitionKey      string
	BusinessType       string
	BusinessID         string
	TaskID             shared.ID
	NodeID             shared.ID
	Action             ActionType
	Actor              Actor
	Meaning            string
	VerificationID     shared.ID
	VerificationMethod string
	VerificationAt     time.Time
	Audience           string
	Evidence           []EvidenceReference
	CommentDigest      string
	EvidenceDigest     string
	AuditCorrelationID shared.ID
	SignedAt           time.Time
}

func (r SignatureReceipt) VerifyDigest() error {
	expected, err := signatureReceiptDigest(r)
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(r.EvidenceDigest), expected) {
		return fmt.Errorf("workflow signature receipt digest mismatch")
	}
	return nil
}

func (r SignatureReceipt) VerifyAction(action Action) error {
	actor := normalizeActor(action.Actor)
	if r.ActionID != action.ID ||
		r.InstanceID != action.InstanceID ||
		r.TaskID != action.TaskID ||
		r.NodeID != action.NodeID ||
		r.Action != action.Type ||
		r.Actor.ID != actor.ID ||
		r.Actor.Name != actor.Name ||
		r.CommentDigest != digestText(action.Comment) ||
		!r.SignedAt.Equal(action.CreatedAt.UTC()) {
		return fmt.Errorf("workflow signature receipt action correlation is invalid")
	}
	return nil
}

func (i Instance) ValidateSignatureEvidence() error {
	receipts := make(map[shared.ID]SignatureReceipt, len(i.Receipts))
	verificationIDs := make(map[shared.ID]struct{}, len(i.Receipts))
	for _, receipt := range i.Receipts {
		if receipt.ID.IsZero() ||
			receipt.ActionID.IsZero() ||
			receipt.InstanceID != i.ID ||
			receipt.DefinitionID != i.DefinitionID ||
			receipt.DefinitionKey != strings.TrimSpace(i.DefinitionKey) ||
			receipt.BusinessType != strings.TrimSpace(i.BusinessType) ||
			receipt.BusinessID != strings.TrimSpace(i.BusinessID) ||
			receipt.VerificationID.IsZero() ||
			receipt.AuditCorrelationID != receipt.ID {
			return fmt.Errorf("workflow signature receipt instance correlation is invalid")
		}
		if _, exists := receipts[receipt.ID]; exists {
			return fmt.Errorf("workflow signature receipt is duplicated")
		}
		if _, exists := verificationIDs[receipt.VerificationID]; exists {
			return fmt.Errorf("workflow signature verification is duplicated")
		}
		if err := receipt.VerifyDigest(); err != nil {
			return err
		}
		receipts[receipt.ID] = receipt
		verificationIDs[receipt.VerificationID] = struct{}{}
	}
	linked := make(map[shared.ID]struct{}, len(receipts))
	for _, action := range i.Timeline {
		if action.ReceiptID.IsZero() {
			continue
		}
		receipt, exists := receipts[action.ReceiptID]
		if !exists {
			return fmt.Errorf("workflow signature receipt is not linked to an action")
		}
		if err := receipt.VerifyAction(action); err != nil {
			return err
		}
		linked[receipt.ID] = struct{}{}
	}
	if len(linked) != len(receipts) {
		return fmt.Errorf("workflow signature receipt is not linked to an action")
	}
	return nil
}

func (i *Instance) signatureReceipt(definition Definition, node Node, action Action, signature *DecisionSignature) (*SignatureReceipt, error) {
	if node.Signature == nil && signature == nil {
		return nil, nil
	}
	if node.Signature == nil {
		return nil, fmt.Errorf("workflow node does not accept signature evidence")
	}
	if signature == nil {
		return nil, fmt.Errorf("workflow decision requires electronic signature")
	}
	if err := node.Signature.Validate(); err != nil {
		return nil, err
	}
	meaning := strings.TrimSpace(signature.Meaning)
	if meaning != strings.TrimSpace(node.Signature.Meaning) {
		return nil, fmt.Errorf("workflow signature meaning does not match decision policy")
	}
	if signature.VerificationID.IsZero() || strings.TrimSpace(signature.Audience) == "" ||
		strings.TrimSpace(signature.Method) == "" || signature.VerifiedAt.IsZero() {
		return nil, fmt.Errorf("workflow signature verification is incomplete")
	}
	if signature.VerifiedAt.After(action.CreatedAt.Add(30*time.Second)) ||
		action.CreatedAt.Sub(signature.VerifiedAt) > 5*time.Minute {
		return nil, fmt.Errorf("workflow signature verification is outside the allowed window")
	}
	evidence, err := normalizeEvidence(signature.Evidence)
	if err != nil {
		return nil, err
	}
	if node.Signature.RequireEvidence && len(evidence) == 0 {
		return nil, fmt.Errorf("workflow decision requires attachment evidence")
	}
	for _, existing := range i.Receipts {
		if existing.VerificationID == signature.VerificationID {
			return nil, fmt.Errorf("workflow reverification proof was already consumed")
		}
	}
	receiptID := actionID(i.ID, ActionType("signature"), action.ID.String())
	receipt := SignatureReceipt{
		ID: receiptID, ActionID: action.ID, InstanceID: i.ID, DefinitionID: definition.ID,
		DefinitionKey: strings.TrimSpace(definition.Key), BusinessType: strings.TrimSpace(i.BusinessType), BusinessID: strings.TrimSpace(i.BusinessID),
		TaskID: action.TaskID, NodeID: action.NodeID, Action: action.Type, Actor: normalizeActor(action.Actor),
		Meaning: meaning, VerificationID: signature.VerificationID, VerificationMethod: strings.TrimSpace(signature.Method),
		VerificationAt: signature.VerifiedAt.UTC(), Audience: strings.TrimSpace(signature.Audience), Evidence: evidence,
		CommentDigest: digestText(action.Comment), AuditCorrelationID: receiptID, SignedAt: action.CreatedAt.UTC(),
	}
	receipt.EvidenceDigest, err = signatureReceiptDigest(receipt)
	if err != nil {
		return nil, err
	}
	return &receipt, nil
}

func normalizeEvidence(input []EvidenceReference) ([]EvidenceReference, error) {
	if len(input) > maxSignatureEvidence {
		return nil, fmt.Errorf("workflow signature evidence exceeds limit")
	}
	out := append([]EvidenceReference(nil), input...)
	sort.Slice(out, func(left, right int) bool { return out[left].FileID.String() < out[right].FileID.String() })
	seen := make(map[shared.ID]struct{}, len(out))
	for index := range out {
		item := &out[index]
		item.Name = strings.TrimSpace(item.Name)
		item.Hash = strings.ToLower(strings.TrimSpace(item.Hash))
		item.MIME = strings.TrimSpace(item.MIME)
		if item.FileID.IsZero() || item.Hash == "" || strings.ContainsAny(item.Hash, " \t\r\n") || item.Size < 0 {
			return nil, fmt.Errorf("workflow signature evidence is invalid")
		}
		if _, exists := seen[item.FileID]; exists {
			return nil, fmt.Errorf("workflow signature evidence is duplicated")
		}
		seen[item.FileID] = struct{}{}
	}
	return out, nil
}

func signatureReceiptDigest(receipt SignatureReceipt) (string, error) {
	payload := struct {
		ID, ActionID, InstanceID, DefinitionID, DefinitionKey, BusinessType, BusinessID         string
		TaskID, NodeID, Action, ActorID, ActorName, Meaning, VerificationID, VerificationMethod string
		VerificationAt, Audience, CommentDigest, AuditCorrelationID, SignedAt                   string
		Evidence                                                                                []EvidenceReference
	}{
		ID: receipt.ID.String(), ActionID: receipt.ActionID.String(), InstanceID: receipt.InstanceID.String(),
		DefinitionID: receipt.DefinitionID.String(), DefinitionKey: receipt.DefinitionKey,
		BusinessType: receipt.BusinessType, BusinessID: receipt.BusinessID, TaskID: receipt.TaskID.String(),
		NodeID: receipt.NodeID.String(), Action: string(receipt.Action), ActorID: receipt.Actor.ID.String(),
		ActorName: receipt.Actor.Name, Meaning: receipt.Meaning, VerificationID: receipt.VerificationID.String(), VerificationMethod: receipt.VerificationMethod,
		VerificationAt: receipt.VerificationAt.UTC().Format(time.RFC3339Nano), Audience: receipt.Audience,
		CommentDigest: receipt.CommentDigest, AuditCorrelationID: receipt.AuditCorrelationID.String(),
		SignedAt: receipt.SignedAt.UTC().Format(time.RFC3339Nano), Evidence: receipt.Evidence,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode workflow signature receipt: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func digestText(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])
}

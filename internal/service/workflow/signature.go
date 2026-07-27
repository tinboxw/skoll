package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
	"github.com/tinboxw/skoll/pkg/security"
)

func (s *serviceImpl) prepareDecisionSignature(ctx context.Context, input TaskActionInput, action domainworkflow.ActionType) (*domainworkflow.DecisionSignature, error) {
	definition, err := s.definitionForInstance(ctx, input.InstanceID)
	if err != nil {
		return nil, err
	}
	instance, err := s.repo.GetInstance(ctx, input.InstanceID)
	if err != nil {
		return nil, err
	}
	var task *domainworkflow.Task
	for index := range instance.Tasks {
		if instance.Tasks[index].ID == input.TaskID {
			task = &instance.Tasks[index]
			break
		}
	}
	if task == nil {
		return nil, fmt.Errorf("workflow task not found")
	}
	var node *domainworkflow.Node
	for index := range definition.Nodes {
		if definition.Nodes[index].ID == task.NodeID {
			node = &definition.Nodes[index]
			break
		}
	}
	if node == nil || node.Type != domainworkflow.NodeApproval {
		return nil, fmt.Errorf("workflow approval node is invalid")
	}
	if node.Signature == nil && input.Signature == nil {
		return nil, nil
	}
	if node.Signature == nil {
		return nil, fmt.Errorf("workflow node does not accept signature evidence")
	}
	if input.Signature == nil {
		return nil, fmt.Errorf("workflow decision requires electronic signature")
	}
	if s.opts.Proofs == nil || s.opts.Evidence == nil {
		return nil, fmt.Errorf("workflow signature dependencies are not configured")
	}
	now := input.Now
	if now.IsZero() {
		now = s.opts.Now()
	}
	claims, err := s.opts.Proofs.VerifyReverificationProof(input.Signature.Proof, now.UTC())
	if err != nil {
		return nil, err
	}
	subject := strings.TrimSpace(input.Actor.ID.String())
	audience := strings.TrimSpace(input.Signature.Audience)
	if claims.Subject != subject || claims.Audience != audience ||
		claims.Purpose != security.ReverificationPurposeWorkflowSignature {
		return nil, fmt.Errorf("workflow signature proof binding is invalid")
	}
	evidenceIDs, err := normalizeEvidenceIDs(input.Signature.EvidenceIDs)
	if err != nil {
		return nil, err
	}
	evidence, err := s.opts.Evidence.ResolveWorkflowEvidence(ctx, input.Actor, evidenceIDs)
	if err != nil {
		return nil, err
	}
	return &domainworkflow.DecisionSignature{
		VerificationID: shared.ID(claims.ID), Audience: audience, Method: claims.Method,
		Meaning: strings.TrimSpace(input.Signature.Meaning), Evidence: evidence, VerifiedAt: claims.VerifiedAt,
	}, nil
}

func normalizeEvidenceIDs(input []shared.ID) ([]shared.ID, error) {
	if len(input) > 16 {
		return nil, fmt.Errorf("workflow signature evidence exceeds limit")
	}
	seen := make(map[shared.ID]struct{}, len(input))
	out := make([]shared.ID, 0, len(input))
	for _, id := range input {
		id = shared.ID(strings.TrimSpace(id.String()))
		if id.IsZero() {
			return nil, fmt.Errorf("workflow signature evidence id is required")
		}
		if _, exists := seen[id]; exists {
			return nil, fmt.Errorf("workflow signature evidence id is duplicated")
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

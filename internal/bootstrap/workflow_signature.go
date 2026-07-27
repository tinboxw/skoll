package bootstrap

import (
	"context"
	"fmt"
	"strings"

	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
	filesvc "github.com/tinboxw/skoll/internal/service/file"
	"github.com/tinboxw/skoll/pkg/security"
)

type workflowEvidenceResolver struct {
	files filesvc.Service
}

func (r workflowEvidenceResolver) ResolveWorkflowEvidence(ctx context.Context, actor domainworkflow.Actor, fileIDs []shared.ID) ([]domainworkflow.EvidenceReference, error) {
	if r.files == nil {
		return nil, fmt.Errorf("workflow evidence file service is required")
	}
	if actor.ID.IsZero() {
		return nil, fmt.Errorf("workflow evidence actor is required")
	}
	out := make([]domainworkflow.EvidenceReference, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		object, decision, err := r.files.Get(ctx, filesvc.GetInput{
			FileID: fileID, SubjectType: domainrbac.SubjectUser, SubjectID: actor.ID, ActorName: strings.TrimSpace(actor.Name),
			Metadata: map[string]any{"purpose": "workflow-signature"},
		})
		if err != nil {
			return nil, err
		}
		if object == nil || !decision.Allowed {
			return nil, fmt.Errorf("workflow signature evidence access denied")
		}
		out = append(out, domainworkflow.EvidenceReference{
			FileID: object.ID, Name: object.Name, Hash: object.Hash, Size: object.Size, MIME: object.MIME,
		})
	}
	return out, nil
}

func newWorkflowProofService(jwtSecret string) (*security.ReverificationProofService, error) {
	return security.NewReverificationProofService("skoll-reverification:" + strings.TrimSpace(jwtSecret))
}

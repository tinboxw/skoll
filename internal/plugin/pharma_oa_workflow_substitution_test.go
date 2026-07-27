package plugin

import (
	"context"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

func (w *pharmaLifecycleWorkflows) CreateSubstitution(context.Context, pluginsdk.WorkflowSubstitutionInput) (pluginsdk.WorkflowSubstitution, error) {
	return pluginsdk.WorkflowSubstitution{}, nil
}

func (w *pharmaLifecycleWorkflows) RevokeSubstitution(context.Context, string) (pluginsdk.WorkflowSubstitution, error) {
	return pluginsdk.WorkflowSubstitution{}, nil
}

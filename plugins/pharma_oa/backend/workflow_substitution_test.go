package main

import (
	"context"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

func (w *testWorkflows) CreateSubstitution(context.Context, pluginsdk.WorkflowSubstitutionInput) (pluginsdk.WorkflowSubstitution, error) {
	return pluginsdk.WorkflowSubstitution{}, nil
}

func (w *testWorkflows) RevokeSubstitution(context.Context, string) (pluginsdk.WorkflowSubstitution, error) {
	return pluginsdk.WorkflowSubstitution{}, nil
}

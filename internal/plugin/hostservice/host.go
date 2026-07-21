package hostservice

import (
	"fmt"
	"strings"

	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	jobsvc "github.com/tinboxw/skoll/internal/service/job"
	systemsvc "github.com/tinboxw/skoll/internal/service/system"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type HostServicesDependencies struct {
	PluginID     string
	Transactions pluginsdk.TransactionService
	DataScopes   pluginsdk.DataScopeService
	Files        fileBackend
	Audit        auditsvc.Service
	ConfigStore  PluginConfigStore
	System       systemsvc.Service
	MasterSecret string
	Workflow     workflowsvc.Service
	Jobs         *jobsvc.Service
}

func NewHostServices(deps HostServicesDependencies) (pluginsdk.HostServices, error) {
	pluginID := strings.ToLower(strings.TrimSpace(deps.PluginID))
	files, err := NewFileService(pluginID, deps.Files)
	if err != nil {
		return pluginsdk.HostServices{}, err
	}
	audit, err := NewAuditService(pluginID, deps.Audit)
	if err != nil {
		return pluginsdk.HostServices{}, err
	}
	config, err := NewConfigService(pluginID, deps.ConfigStore, deps.System, audit)
	if err != nil {
		return pluginsdk.HostServices{}, err
	}
	secrets, err := NewSecretService(pluginID, deps.System, audit, deps.MasterSecret)
	if err != nil {
		return pluginsdk.HostServices{}, err
	}
	workflows, err := NewWorkflowService(pluginID, deps.Workflow, audit)
	if err != nil {
		return pluginsdk.HostServices{}, err
	}
	jobs, err := NewJobService(pluginID, deps.Jobs, audit)
	if err != nil {
		return pluginsdk.HostServices{}, err
	}
	host := pluginsdk.HostServices{
		PluginID: pluginID, Transactions: deps.Transactions, DataScopes: deps.DataScopes,
		Files: files, Audit: audit, Config: config, Secrets: secrets, Workflows: workflows, Jobs: jobs,
	}
	if err := host.Validate(); err != nil {
		return pluginsdk.HostServices{}, fmt.Errorf("build plugin host services: %w", err)
	}
	return host, nil
}

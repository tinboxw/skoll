package pluginsdk

import (
	"context"
	"fmt"
	"strings"
)

type Transaction interface {
	Context() context.Context
}

type TransactionService interface {
	Within(ctx context.Context, fn func(Transaction) error) error
}

type Permission struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

type DataScopeService interface {
	Resolve(ctx context.Context, permission Permission) (ScopePredicate, error)
}

type HostServices struct {
	PluginID        string
	Capabilities    []HostCapability
	Transactions    TransactionService
	DataScopes      DataScopeService
	DataStore       DataStoreService
	Events          EventService
	DocumentNumbers DocumentNumberService
	Documents       DocumentService
	Files           FileService
	Audit           AuditService
	Config          ConfigService
	Secrets         SecretService
	Workflows       WorkflowService
	Jobs            JobService
}

func (s HostServices) Validate() error {
	if strings.TrimSpace(s.PluginID) == "" {
		return fmt.Errorf("plugin host identity is required")
	}
	seenCapabilities := make(map[HostCapability]struct{}, len(s.Capabilities))
	for _, capability := range s.Capabilities {
		if err := capability.Validate(); err != nil {
			return err
		}
		if _, exists := seenCapabilities[capability]; exists {
			return fmt.Errorf("plugin host capability %q is duplicated", capability)
		}
		seenCapabilities[capability] = struct{}{}
	}
	if s.Transactions == nil {
		return fmt.Errorf("plugin host transaction service is required")
	}
	if s.DataScopes == nil {
		return fmt.Errorf("plugin host data-scope service is required")
	}
	if s.DataStore == nil {
		return fmt.Errorf("plugin host datastore service is required")
	}
	if s.Events == nil {
		return fmt.Errorf("plugin host event service is required")
	}
	if s.DocumentNumbers == nil {
		return fmt.Errorf("plugin host document number service is required")
	}
	if s.Documents == nil {
		return fmt.Errorf("plugin host document workflow service is required")
	}
	if s.Files == nil {
		return fmt.Errorf("plugin host file service is required")
	}
	if s.Audit == nil {
		return fmt.Errorf("plugin host audit service is required")
	}
	if s.Config == nil {
		return fmt.Errorf("plugin host config service is required")
	}
	if s.Secrets == nil {
		return fmt.Errorf("plugin host secret service is required")
	}
	if s.Workflows == nil {
		return fmt.Errorf("plugin host workflow service is required")
	}
	if s.Jobs == nil {
		return fmt.Errorf("plugin host job service is required")
	}
	return nil
}

func (s HostServices) HasCapability(capability HostCapability) bool {
	for _, granted := range s.Capabilities {
		if granted == capability {
			return true
		}
	}
	return false
}

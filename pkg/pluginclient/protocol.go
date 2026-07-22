package pluginclient

const (
	EnvironmentPluginID      = "SKOLL_PLUGIN_ID"
	EnvironmentPluginAddress = "SKOLL_PLUGIN_ADDRESS"
	EnvironmentHostURL       = "SKOLL_PLUGIN_HOST_URL"
	EnvironmentHostToken     = "SKOLL_PLUGIN_HOST_TOKEN"

	AuthorizationHeader = "Authorization"
	UserTokenHeader     = "X-Skoll-User-Token"
	TransactionHeader   = "X-Skoll-Transaction-ID"

	HostAPIVersion = "v1"
)

type ScopeSnapshot struct {
	SubjectID        string   `json:"subjectId"`
	TenantIDs        []string `json:"tenantIds"`
	OwnerIDs         []string `json:"ownerIds"`
	OrganizationIDs  []string `json:"organizationIds"`
	AllTenants       bool     `json:"allTenants"`
	AllOwners        bool     `json:"allOwners"`
	AllOrganizations bool     `json:"allOrganizations"`
	Denied           bool     `json:"denied"`
}

type TransactionStartResponse struct {
	ID string `json:"id"`
}

type TransactionFinishRequest struct {
	Commit bool `json:"commit"`
}

type ErrorResponse struct {
	Code      string `json:"code"`
	Field     string `json:"field,omitempty"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable,omitempty"`
}

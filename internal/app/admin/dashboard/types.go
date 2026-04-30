package dashboard

type AggregateResponse struct {
	Contract             ContractDescriptor     `json:"contract"`
	GeneratedAtUnixSec   int64                  `json:"generated_at_unix_sec"`
	AuthSession          AuthSessionContext     `json:"auth_session"`
	AuthObservability    AuthObservability      `json:"auth_observability"`
	AuthActionability    AuthActionability      `json:"auth_actionability"`
	JWTSessionBootstrap  JWTSessionBootstrap    `json:"jwt_session_bootstrap"`
	Status               SystemStatusResponse   `json:"status"`
	RuntimeMetrics       RuntimeMetricsResponse `json:"runtime_metrics"`
	NodeHealth           NodeHealthResponse     `json:"node_health"`
	SchedulerReliability any                    `json:"scheduler_reliability"`
	HardeningPosture     any                    `json:"hardening_posture"`
}

type ContractDescriptor struct {
	Name             string   `json:"name"`
	Version          string   `json:"version"`
	Stability        string   `json:"stability"`
	RequiredSections []string `json:"required_sections"`
}

type AuthSessionContext struct {
	Authenticated          bool   `json:"authenticated"`
	AuthModeHint           string `json:"auth_mode_hint"`
	RoleID                 string `json:"role_id,omitempty"`
	HasRoleBinding         bool   `json:"has_role_binding"`
	TokenHeaderPresent     bool   `json:"token_header_present"`
	SignatureHeaderPresent bool   `json:"signature_header_present"`
}

type AuthModeCounters struct {
	Success uint64            `json:"success"`
	Failure uint64            `json:"failure"`
	Reasons map[string]uint64 `json:"reasons"`
}

type AuthObservability struct {
	StaticToken  AuthModeCounters `json:"static_token"`
	HMACSHA256   AuthModeCounters `json:"hmac_sha256"`
	TotalFailure uint64           `json:"total_failure"`
}

type AuthActionability struct {
	Severity            string   `json:"severity"`
	RecommendedAuthMode string   `json:"recommended_auth_mode"`
	FailureRate         float64  `json:"failure_rate"`
	TopFailureReasons   []string `json:"top_failure_reasons"`
	NextActions         []string `json:"next_actions"`
	Docs                []string `json:"docs"`
}

type JWTSessionBootstrap struct {
	TokenPresent          bool                     `json:"token_present"`
	TokenFormat           string                   `json:"token_format"`
	ClaimsTrusted         bool                     `json:"claims_trusted"`
	MiddlewareBridge      JWTMiddlewareBridge      `json:"middleware_bridge"`
	ProvenanceAuditExport JWTProvenanceAuditExport `json:"provenance_audit_export"`
	SessionState          string                   `json:"session_state"`
	VerificationState     string                   `json:"verification_state"`
	VerificationHint      string                   `json:"verification_hint,omitempty"`
	TrustLevel            string                   `json:"trust_level"`
	TrustMessage          string                   `json:"trust_message"`
	Subject               string                   `json:"subject,omitempty"`
	Issuer                string                   `json:"issuer,omitempty"`
	Audience              []string                 `json:"audience,omitempty"`
	IssuedAtUnixSec       int64                    `json:"issued_at_unix_sec,omitempty"`
	ExpiresAtUnixSec      int64                    `json:"expires_at_unix_sec,omitempty"`
	ExpiresInSec          int64                    `json:"expires_in_sec,omitempty"`
	RefreshAfterUnixSec   int64                    `json:"refresh_after_unix_sec,omitempty"`
	RefreshRecommended    bool                     `json:"refresh_recommended"`
	RefreshReason         string                   `json:"refresh_reason,omitempty"`
	Expired               bool                     `json:"expired"`
	ParseError            string                   `json:"parse_error,omitempty"`
}

type JWTMiddlewareBridge struct {
	Present             bool     `json:"present"`
	Verified            bool     `json:"verified"`
	Source              string   `json:"source"`
	SourceProvenance    []string `json:"source_provenance"`
	Subject             string   `json:"subject,omitempty"`
	SubjectSource       string   `json:"subject_source,omitempty"`
	RoleID              string   `json:"role_id,omitempty"`
	RoleSource          string   `json:"role_source,omitempty"`
	ClaimsVersion       string   `json:"claims_version,omitempty"`
	ClaimsVersionSource string   `json:"claims_version_source,omitempty"`
	VerifiedSource      string   `json:"verified_source,omitempty"`
}

type JWTProvenanceAuditExport struct {
	Enabled             bool                            `json:"enabled"`
	SourceProvenance    []string                        `json:"source_provenance"`
	SourcePath          string                          `json:"source_path,omitempty"`
	OperationalMetrics  JWTProvenanceOperationalMetrics `json:"operational_metrics"`
	SLODashboard        JWTProvenanceSLODashboard       `json:"slo_dashboard"`
	ErrorBudgetPolicy   JWTProvenanceErrorBudgetPolicy  `json:"error_budget_policy"`
	AlertingHints       []string                        `json:"alerting_hints,omitempty"`
	Source              string                          `json:"source,omitempty"`
	Verified            bool                            `json:"verified"`
	ClaimsTrusted       bool                            `json:"claims_trusted"`
	VerificationState   string                          `json:"verification_state"`
	Subject             string                          `json:"subject,omitempty"`
	RoleID              string                          `json:"role_id,omitempty"`
	ClaimsVersion       string                          `json:"claims_version,omitempty"`
	RoleSource          string                          `json:"role_source,omitempty"`
	SubjectSource       string                          `json:"subject_source,omitempty"`
	ClaimsVersionSource string                          `json:"claims_version_source,omitempty"`
	VerifiedSource      string                          `json:"verified_source,omitempty"`
}

type JWTProvenanceOperationalMetrics struct {
	ExportsTotal    uint64 `json:"exports_total"`
	EnabledTotal    uint64 `json:"enabled_total"`
	DisabledTotal   uint64 `json:"disabled_total"`
	VerifiedTotal   uint64 `json:"verified_total"`
	UnverifiedTotal uint64 `json:"unverified_total"`
	InvalidTotal    uint64 `json:"invalid_total"`
	AlertHintsTotal uint64 `json:"alert_hints_total"`
}

type JWTProvenanceSLODashboard struct {
	Window              string  `json:"window"`
	TargetReliability   float64 `json:"target_reliability"`
	ObservedReliability float64 `json:"observed_reliability"`
	ErrorRate           float64 `json:"error_rate"`
	BurnRate            float64 `json:"burn_rate"`
	Status              string  `json:"status"`
}

type JWTProvenanceErrorBudgetPolicy struct {
	Window              string  `json:"window"`
	BudgetRatio         float64 `json:"budget_ratio"`
	ConsumedRatio       float64 `json:"consumed_ratio"`
	RemainingRatio      float64 `json:"remaining_ratio"`
	Action              string  `json:"action"`
	FreezeRecommended   bool    `json:"freeze_recommended"`
	EscalateRecommended bool    `json:"escalate_recommended"`
}

type SystemStatusResponse struct {
	Version       string `json:"version"`
	UserCount     int    `json:"user_count"`
	RoleCount     int    `json:"role_count"`
	MenuCount     int    `json:"menu_count"`
	ConfigCount   int    `json:"config_count"`
	DictCount     int    `json:"dict_count"`
	FileCount     int    `json:"file_count"`
	JobCount      int    `json:"job_count"`
	PluginCount   int    `json:"plugin_count"`
	APIEntryCount int    `json:"api_entry_count"`
}

type RuntimeMetricsResponse struct {
	UptimeSeconds   int64  `json:"uptime_seconds"`
	Goroutines      int    `json:"goroutines"`
	MemoryAlloc     uint64 `json:"memory_alloc_bytes"`
	MemorySys       uint64 `json:"memory_sys_bytes"`
	HeapAlloc       uint64 `json:"heap_alloc_bytes"`
	HeapSys         uint64 `json:"heap_sys_bytes"`
	TotalAlloc      uint64 `json:"total_alloc_bytes"`
	GCCount         uint32 `json:"gc_count"`
	LastGCPauseNs   uint64 `json:"last_gc_pause_ns"`
	SnapshotUnixSec int64  `json:"snapshot_unix_sec"`
}

type HealthCheckItem struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Detail    string `json:"detail"`
	CheckedAt int64  `json:"checked_at_unix_sec"`
}

type NodeHealthResponse struct {
	NodeStatus   string            `json:"node_status"`
	CheckedAt    int64             `json:"checked_at_unix_sec"`
	Dependencies []HealthCheckItem `json:"dependencies"`
}

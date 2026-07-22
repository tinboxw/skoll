package plugin

import "time"

type DataControlSnapshot struct {
	PluginID   string               `json:"pluginId"`
	CapturedAt time.Time            `json:"capturedAt"`
	State      string               `json:"state"`
	Schema     DataControlSchema    `json:"schema"`
	Migration  DataControlMigration `json:"migration"`
	Policy     DataControlPolicy    `json:"policy"`
	Actions    DataControlActions   `json:"actions"`
}

type DataControlSchema struct {
	Available      bool               `json:"available"`
	Registered     bool               `json:"registered"`
	Namespace      string             `json:"namespace,omitempty"`
	Tables         []DataControlTable `json:"tables"`
	TotalSizeBytes int64              `json:"totalSizeBytes"`
	SizeKnown      bool               `json:"sizeKnown"`
}

type DataControlTable struct {
	LogicalName  string   `json:"logicalName"`
	PhysicalName string   `json:"physicalName"`
	Fields       []string `json:"fields"`
	PrimaryKey   []string `json:"primaryKey"`
	IndexCount   int      `json:"indexCount"`
	Exists       bool     `json:"exists"`
	SizeBytes    int64    `json:"sizeBytes"`
	SizeKnown    bool     `json:"sizeKnown"`
}

type DataControlMigration struct {
	DeclaredVersion string                     `json:"declaredVersion,omitempty"`
	CurrentVersion  int                        `json:"currentVersion"`
	Applied         []DataControlMigrationStep `json:"applied"`
	Pending         []DataControlMigrationStep `json:"pending"`
	Error           string                     `json:"error,omitempty"`
}

type DataControlMigrationStep struct {
	Version   int        `json:"version"`
	Name      string     `json:"name"`
	Checksum  string     `json:"checksum"`
	AppliedAt *time.Time `json:"appliedAt,omitempty"`
}

type DataControlPolicy struct {
	Uninstall string `json:"uninstall,omitempty"`
	Rollback  string `json:"rollback,omitempty"`
	Effect    string `json:"effect"`
}

type DataControlActions struct {
	CanRollback      bool   `json:"canRollback"`
	RollbackMaxSteps int    `json:"rollbackMaxSteps"`
	BlockedReason    string `json:"blockedReason,omitempty"`
}

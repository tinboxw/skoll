package system

type ConfigScope string

const (
	ConfigScopeSystem ConfigScope = "system"
	ConfigScopePlugin ConfigScope = "plugin"
)

type ConfigFieldType string

const (
	ConfigFieldString   ConfigFieldType = "string"
	ConfigFieldTextarea ConfigFieldType = "textarea"
	ConfigFieldNumber   ConfigFieldType = "number"
	ConfigFieldBoolean  ConfigFieldType = "boolean"
	ConfigFieldSelect   ConfigFieldType = "select"
)

type UpsertInput struct {
	Key       string
	Value     string
	Encrypted bool
}

type ListInput struct {
	Offset int
	Limit  int
}

type DictionaryTypeInput struct {
	ID          string
	Code        string
	Name        string
	Description string
	Status      string
	Sort        int
	Builtin     bool
}

type DictionaryTypeListInput struct {
	Offset int
	Limit  int
}

type DictionaryItemInput struct {
	ID       string
	TypeCode string
	Label    string
	Value    string
	Status   string
	Sort     int
	Builtin  bool
}

type DictionaryItemListInput struct {
	TypeCode string
	Offset   int
	Limit    int
}

type ConfigSchemaInput struct {
	Scope       ConfigScope
	Owner       string
	Title       string
	TitleZhCN   string
	TitleEnUS   string
	Description string
	Fields      []ConfigField
}

type ConfigSchemaListInput struct {
	Scope ConfigScope
}

type ConfigSchema struct {
	Scope       ConfigScope   `json:"scope,omitempty"`
	Owner       string        `json:"owner,omitempty"`
	Title       string        `json:"title,omitempty"`
	TitleZhCN   string        `json:"titleZhCN,omitempty"`
	TitleEnUS   string        `json:"titleEnUS,omitempty"`
	Description string        `json:"description,omitempty"`
	Fields      []ConfigField `json:"fields"`
}

type ConfigField struct {
	Key         string          `json:"key"`
	Label       string          `json:"label,omitempty"`
	LabelZhCN   string          `json:"labelZhCN,omitempty"`
	LabelEnUS   string          `json:"labelEnUS,omitempty"`
	Type        ConfigFieldType `json:"type,omitempty"`
	Required    bool            `json:"required,omitempty"`
	Default     string          `json:"default,omitempty"`
	Placeholder string          `json:"placeholder,omitempty"`
	Help        string          `json:"help,omitempty"`
	Min         *float64        `json:"min,omitempty"`
	Max         *float64        `json:"max,omitempty"`
	MinLength   *int            `json:"minLength,omitempty"`
	MaxLength   *int            `json:"maxLength,omitempty"`
	Pattern     string          `json:"pattern,omitempty"`
	Options     []ConfigOption  `json:"options,omitempty"`
}

type ConfigOption struct {
	Label     string `json:"label,omitempty"`
	LabelZhCN string `json:"labelZhCN,omitempty"`
	LabelEnUS string `json:"labelEnUS,omitempty"`
	Value     string `json:"value"`
}

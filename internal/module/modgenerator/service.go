package modgenerator

import (
	"fmt"
	"regexp"
	"strings"
)

var moduleNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

type Artifact struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type FormField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

type FormSchema struct {
	Version string      `json:"version"`
	Fields  []FormField `json:"fields"`
}

type TemplateCompatibility struct {
	TemplateVersion    string `json:"template_version"`
	CompatibilityLevel string `json:"compatibility_level"`
	Policy             string `json:"policy"`
}

type Result struct {
	Module        string                `json:"module"`
	Artifacts     []Artifact            `json:"artifacts"`
	FormSchema    *FormSchema           `json:"form_schema,omitempty"`
	Compatibility TemplateCompatibility `json:"compatibility"`
}

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Generate(module string) (Result, error) {
	return s.GenerateWithSchema(module, nil, "")
}

func (s *Service) GenerateWithSchema(module string, schema *FormSchema, templateVersion string) (Result, error) {
	module = strings.TrimSpace(module)
	if !moduleNamePattern.MatchString(module) {
		return Result{}, fmt.Errorf("invalid module name: %s", module)
	}

	servicePath := fmt.Sprintf("internal/module/%s/service.go", module)
	testPath := fmt.Sprintf("internal/module/%s/service_test.go", module)
	serviceContent := fmt.Sprintf("package %s\n\n// Service is generated baseline scaffold for module %s.\ntype Service struct{}\n\nfunc NewService() *Service {\n\treturn &Service{}\n}\n", module, module)
	testContent := fmt.Sprintf("package %s\n\nimport \"testing\"\n\nfunc TestNewService(t *testing.T) {\n\tif NewService() == nil {\n\t\tt.Fatalf(\"expected service instance\")\n\t}\n}\n", module)

	compatibility, err := normalizeCompatibility(templateVersion)
	if err != nil {
		return Result{}, err
	}

	normalizedSchema, err := normalizeSchema(schema)
	if err != nil {
		return Result{}, err
	}

	return Result{
		Module:        module,
		Compatibility: compatibility,
		FormSchema:    normalizedSchema,
		Artifacts: []Artifact{
			{Path: servicePath, Content: serviceContent},
			{Path: testPath, Content: testContent},
		},
	}, nil
}

func normalizeCompatibility(templateVersion string) (TemplateCompatibility, error) {
	templateVersion = strings.TrimSpace(templateVersion)
	if templateVersion == "" {
		templateVersion = "v1"
	}
	if templateVersion != "v1" && templateVersion != "v2" {
		return TemplateCompatibility{}, fmt.Errorf("unsupported template version: %s", templateVersion)
	}
	level := "stable"
	if templateVersion == "v2" {
		level = "compatible"
	}
	return TemplateCompatibility{
		TemplateVersion:    templateVersion,
		CompatibilityLevel: level,
		Policy:             "additive_fields_only",
	}, nil
}

func normalizeSchema(schema *FormSchema) (*FormSchema, error) {
	if schema == nil {
		return nil, nil
	}
	version := strings.TrimSpace(schema.Version)
	if version == "" {
		version = "v1"
	}
	if version != "v1" && version != "v2" {
		return nil, fmt.Errorf("unsupported form schema version: %s", version)
	}
	seen := make(map[string]struct{}, len(schema.Fields))
	out := make([]FormField, 0, len(schema.Fields))
	for _, field := range schema.Fields {
		name := strings.TrimSpace(field.Name)
		typ := strings.TrimSpace(field.Type)
		if name == "" || typ == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, FormField{Name: name, Type: typ, Required: field.Required})
	}
	return &FormSchema{Version: version, Fields: out}, nil
}

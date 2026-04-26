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

type Result struct {
	Module    string     `json:"module"`
	Artifacts []Artifact `json:"artifacts"`
}

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Generate(module string) (Result, error) {
	module = strings.TrimSpace(module)
	if !moduleNamePattern.MatchString(module) {
		return Result{}, fmt.Errorf("invalid module name: %s", module)
	}

	servicePath := fmt.Sprintf("internal/module/%s/service.go", module)
	testPath := fmt.Sprintf("internal/module/%s/service_test.go", module)
	serviceContent := fmt.Sprintf("package %s\n\n// Service is generated baseline scaffold for module %s.\ntype Service struct{}\n\nfunc NewService() *Service {\n\treturn &Service{}\n}\n", module, module)
	testContent := fmt.Sprintf("package %s\n\nimport \"testing\"\n\nfunc TestNewService(t *testing.T) {\n\tif NewService() == nil {\n\t\tt.Fatalf(\"expected service instance\")\n\t}\n}\n", module)

	return Result{
		Module: module,
		Artifacts: []Artifact{
			{Path: servicePath, Content: serviceContent},
			{Path: testPath, Content: testContent},
		},
	}, nil
}

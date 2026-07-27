package datastore

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"gopkg.in/yaml.v3"
)

const SchemaManifestName = "datastore.yaml"

type schemaManifest struct {
	Version int                   `yaml:"version"`
	Tables  []schemaManifestTable `yaml:"tables"`
}

type schemaManifestTable struct {
	Name           string                `yaml:"name"`
	MutationPolicy TableMutationPolicy   `yaml:"mutation_policy"`
	Fields         []schemaManifestField `yaml:"fields"`
	PrimaryKey     []string              `yaml:"primary_key"`
	Indexes        []schemaManifestIndex `yaml:"indexes"`
}

type schemaManifestField struct {
	Name       string                  `yaml:"name"`
	Type       pluginsdk.DataValueType `yaml:"type"`
	Nullable   bool                    `yaml:"nullable"`
	Mutable    bool                    `yaml:"mutable"`
	Filterable bool                    `yaml:"filterable"`
	Sortable   bool                    `yaml:"sortable"`
}

type schemaManifestIndex struct {
	Fields []string `yaml:"fields"`
	Unique bool     `yaml:"unique"`
}

// LoadSchemaManifest loads the only current datastore schema declaration.
func LoadSchemaManifest(pluginID, pluginDir string) (PluginSchema, bool, error) {
	pluginDir = strings.TrimSpace(pluginDir)
	if pluginDir == "" {
		return PluginSchema{}, false, invalidSchema("pluginDir", "plugin directory is required")
	}
	path := filepath.Join(pluginDir, SchemaManifestName)
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return PluginSchema{}, false, nil
	}
	if err != nil {
		return PluginSchema{}, false, fmt.Errorf("open plugin datastore schema: %w", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(io.LimitReader(file, 1<<20))
	decoder.KnownFields(true)
	var document schemaManifest
	if err = decoder.Decode(&document); err != nil {
		return PluginSchema{}, false, fmt.Errorf("decode plugin datastore schema: %w", err)
	}
	var trailing any
	if err = decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return PluginSchema{}, false, errors.New("plugin datastore schema must contain one YAML document")
		}
		return PluginSchema{}, false, fmt.Errorf("decode plugin datastore schema: %w", err)
	}
	if document.Version != 1 {
		return PluginSchema{}, false, invalidSchema("version", "datastore schema version must be 1")
	}

	schema := PluginSchema{PluginID: strings.TrimSpace(pluginID), Tables: make([]TableSchema, 0, len(document.Tables))}
	for _, tableDocument := range document.Tables {
		table := TableSchema{
			Name:           strings.TrimSpace(tableDocument.Name),
			MutationPolicy: tableDocument.MutationPolicy,
			Fields:         make([]FieldSchema, 0, len(tableDocument.Fields)),
			PrimaryKey:     append([]string(nil), tableDocument.PrimaryKey...),
			Indexes:        make([]IndexSchema, 0, len(tableDocument.Indexes)),
		}
		for _, field := range tableDocument.Fields {
			table.Fields = append(table.Fields, FieldSchema{
				Name: strings.TrimSpace(field.Name), Type: field.Type, Nullable: field.Nullable,
				Mutable: field.Mutable, Filterable: field.Filterable, Sortable: field.Sortable,
			})
		}
		for _, index := range tableDocument.Indexes {
			table.Indexes = append(table.Indexes, IndexSchema{Fields: append([]string(nil), index.Fields...), Unique: index.Unique})
		}
		schema.Tables = append(schema.Tables, table)
	}
	if _, err = buildRegisteredSchema(schema); err != nil {
		return PluginSchema{}, false, err
	}
	return schema, true, nil
}

package datastore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSchemaManifestUsesStrictTypedCurrentContract(t *testing.T) {
	dir := t.TempDir()
	manifest := `version: 1
tables:
  - name: records
    mutation_policy: mutable
    primary_key: [id]
    fields:
      - name: id
        type: string
        filterable: true
        sortable: true
      - name: amount
        type: decimal
        mutable: true
        filterable: true
    indexes:
      - fields: [amount]
`
	if err := os.WriteFile(filepath.Join(dir, SchemaManifestName), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	schema, present, err := LoadSchemaManifest("medical_oa", dir)
	if err != nil || !present {
		t.Fatalf("load schema: present=%v err=%v", present, err)
	}
	if schema.PluginID != "medical_oa" || len(schema.Tables) != 1 || schema.Tables[0].MutationPolicy != TableMutationMutable || schema.Tables[0].Fields[1].Type != "decimal" {
		t.Fatalf("unexpected schema: %+v", schema)
	}

	if err := os.WriteFile(filepath.Join(dir, SchemaManifestName), []byte(strings.Replace(manifest, "version: 1", "version: 1\nlegacy_columns: true", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err = LoadSchemaManifest("medical_oa", dir); err == nil || !strings.Contains(err.Error(), "legacy_columns") {
		t.Fatalf("unknown schema field was accepted: %v", err)
	}
}

func TestLoadSchemaManifestRequiresCurrentMutationPolicy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, SchemaManifestName)
	manifest := `version: 1
tables:
  - name: records
    mutation_policy: mutable
    primary_key: [id]
    fields:
      - name: id
        type: string
`
	for _, test := range []struct {
		name    string
		content string
	}{
		{name: "omitted", content: strings.Replace(manifest, "    mutation_policy: mutable\n", "", 1)},
		{name: "unknown", content: strings.Replace(manifest, "mutable", "legacy_mutable", 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte(test.content), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, _, err := LoadSchemaManifest("medical_oa", dir); err == nil || !strings.Contains(err.Error(), "mutationPolicy") {
				t.Fatalf("invalid mutation policy was accepted: %v", err)
			}
		})
	}
}

func TestLoadSchemaManifestDistinguishesUndeclaredCapability(t *testing.T) {
	schema, present, err := LoadSchemaManifest("medical_oa", t.TempDir())
	if err != nil || present || schema.PluginID != "" || len(schema.Tables) != 0 {
		t.Fatalf("unexpected absent schema result: schema=%+v present=%v err=%v", schema, present, err)
	}
}

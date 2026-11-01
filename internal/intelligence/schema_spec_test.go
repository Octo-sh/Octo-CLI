package intelligence

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSpecV1SchemaAndExampleSync(t *testing.T) {
	// 1. Locate repository root and spec/v1 files
	schemaPath := filepath.Join("..", "..", "spec", "v1", "schema.json")
	examplePath := filepath.Join("..", "..", "spec", "v1", "example.yaml")

	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", schemaPath, err)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(schemaData, &schema); err != nil {
		t.Fatalf("schema.json is not valid JSON: %v", err)
	}

	if schema["$schema"] == "" || schema["title"] == "" {
		t.Fatalf("schema.json missing required metadata fields: %v", schema)
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("schema.json missing properties object")
	}

	for _, reqKey := range []string{"spec", "project", "components", "services", "environment", "verification"} {
		if _, exists := props[reqKey]; !exists {
			t.Fatalf("schema.json missing property %q", reqKey)
		}
	}

	// 2. Validate example.yaml strictly against RepositoryOverride struct with KnownFields(true)
	exampleData, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", examplePath, err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(exampleData))
	decoder.KnownFields(true)

	var override RepositoryOverride
	if err := decoder.Decode(&override); err != nil {
		t.Fatalf("example.yaml is not in sync with Go RepositoryOverride struct: %v", err)
	}

	// Verify exact fields from example.yaml
	if override.Spec != "octo.dev/v1" {
		t.Fatalf("override.Spec = %q, want 'octo.dev/v1'", override.Spec)
	}
	if override.Project.Name != "example-app" {
		t.Fatalf("override.Project.Name = %q, want 'example-app'", override.Project.Name)
	}

	web, ok := override.Components["web"]
	if !ok {
		t.Fatalf("override.Components missing 'web'")
	}
	if web.Type != "application" {
		t.Fatalf("web.Type = %q, want 'application'", web.Type)
	}
	if web.Runtime.Language != "node" || web.Runtime.Version != "22" {
		t.Fatalf("web.Runtime = %+v, want node 22", web.Runtime)
	}
	if web.Command.Dev != "pnpm dev" {
		t.Fatalf("web.Command.Dev = %q, want 'pnpm dev'", web.Command.Dev)
	}
	if len(web.Ports) == 0 || web.Ports[0] != 3000 {
		t.Fatalf("web.Ports = %v, want [3000]", web.Ports)
	}

	pg, ok := override.Services["postgres"]
	if !ok {
		t.Fatalf("override.Services missing 'postgres'")
	}
	if pg.Type != "database" || pg.Image != "postgres:17" {
		t.Fatalf("postgres service = %+v, want database postgres:17", pg)
	}

	dbEnv, ok := override.Environment.Required["DATABASE_URL"]
	if !ok || !dbEnv.Secret {
		t.Fatalf("override.Environment.Required['DATABASE_URL'] = %+v, want secret: true", dbEnv)
	}

	// 3. Verify ApplyRepositoryOverrides correctly populates ProjectModel
	model := ProjectModel{
		Root: t.TempDir(),
	}
	ApplyRepositoryOverrides(&model, &override, examplePath)

	if model.Name != "example-app" {
		t.Fatalf("model.Name = %q, want 'example-app'", model.Name)
	}

	var foundWeb *Component
	for i := range model.Components {
		if model.Components[i].Name == "web" {
			foundWeb = &model.Components[i]
			break
		}
	}
	if foundWeb == nil {
		t.Fatalf("component 'web' not created in ProjectModel")
	}
	if foundWeb.Role != "application" {
		t.Fatalf("foundWeb.Role = %q, want 'application'", foundWeb.Role)
	}
	if foundWeb.Language != "node" {
		t.Fatalf("foundWeb.Language = %q, want 'node'", foundWeb.Language)
	}
	if foundWeb.RunCommand != "pnpm dev" {
		t.Fatalf("foundWeb.RunCommand = %q, want 'pnpm dev'", foundWeb.RunCommand)
	}
	if foundWeb.Port != 3000 || !foundWeb.PortStrict {
		t.Fatalf("foundWeb.Port = %d (strict=%v), want 3000 strict", foundWeb.Port, foundWeb.PortStrict)
	}

	var foundPg *Service
	for i := range model.Services {
		if model.Services[i].Name == "postgres" {
			foundPg = &model.Services[i]
			break
		}
	}
	if foundPg == nil || foundPg.Image != "postgres:17" {
		t.Fatalf("service 'postgres' not created in ProjectModel: %+v", foundPg)
	}

	foundEnv := false
	for _, v := range model.Environment.Variables {
		if v.Name == "DATABASE_URL" && v.Required {
			foundEnv = true
			break
		}
	}
	if !foundEnv {
		t.Fatalf("expected DATABASE_URL (required=true) in model.Environment.Variables: %+v", model.Environment.Variables)
	}
}


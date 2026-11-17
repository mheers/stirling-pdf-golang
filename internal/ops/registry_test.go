package ops

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRegistryCoversEveryDocumentedOperation(t *testing.T) {
	specPath := filepath.Join("..", "..", "api", "openapi.json")
	raw, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}

	documented := map[string]bool{}
	for path, methods := range document.Paths {
		if _, ok := methods["post"]; ok {
			documented[path] = true
		}
	}
	registered := map[string]bool{}
	for _, operation := range All {
		if registered[operation.Path] {
			t.Errorf("duplicate operation path %q", operation.Path)
		}
		registered[operation.Path] = true
		if operation.Name == "" {
			t.Errorf("%s has no command name", operation.Path)
		}
		if operation.Group == "" {
			t.Errorf("%s has no group", operation.Path)
		}
		if operation.Response != "binary" && operation.Response != "json" && operation.Response != "text" {
			t.Errorf("%s has unknown response kind %q", operation.Path, operation.Response)
		}
		flags := map[string]bool{}
		for _, field := range operation.Fields {
			if field.Flag == "" {
				t.Errorf("%s field %q has no flag", operation.Path, field.Name)
			}
			if flags[field.Flag] {
				t.Errorf("%s has duplicate flag %q", operation.Path, field.Flag)
			}
			flags[field.Flag] = true
		}
	}

	for path := range documented {
		if !registered[path] {
			t.Errorf("operation %q is in the OpenAPI document but not in the registry", path)
		}
	}
	for path := range registered {
		if !documented[path] {
			t.Errorf("operation %q is registered but no longer in the OpenAPI document", path)
		}
	}
	if len(All) != len(documented) {
		t.Errorf("registry has %d operations, document has %d", len(All), len(documented))
	}
}

func TestObjectListFields(t *testing.T) {
	for _, operation := range All {
		if operation.Path != "/api/v1/security/redact" {
			continue
		}
		for _, field := range operation.Fields {
			if field.Name == "redactions" {
				if field.Kind != ObjectListKind {
					t.Fatalf("redactions kind = %q, want %q", field.Kind, ObjectListKind)
				}
				return
			}
		}
		t.Fatal("redact operation has no redactions field")
	}
	t.Fatal("redact operation is missing from the registry")
}

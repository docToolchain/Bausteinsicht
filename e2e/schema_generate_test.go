package e2e

// TestSchemaGenerate verifies `schema generate` writes a valid JSON Schema
// derived from the Go model types (no input model needed — it introspects
// internal/model.BausteinsichtModel directly). Covers a CLI command with no
// prior E2E coverage.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSchemaGenerate(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	outPath := filepath.Join(dir, "schema.json")
	out := runCLI(t, bin, dir, "schema", "generate", "--output", "schema.json")
	if out == "" {
		t.Error("schema generate: expected a success message, got empty output")
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read generated schema: %v", err)
	}

	var schema struct {
		Type       string                 `json:"type"`
		Properties map[string]interface{} `json:"properties"`
		Required   []string               `json:"required"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("generated schema is not valid JSON: %v\ncontent: %s", err, data)
	}

	// The model's top-level sections must appear as schema properties —
	// this is what makes the schema useful for IDE autocompletion (a stated
	// top-3 quality goal in CLAUDE.md), so verify the real field names, not
	// just "valid JSON with some properties".
	for _, want := range []string{"specification", "model", "views"} {
		if _, ok := schema.Properties[want]; !ok {
			t.Errorf("generated schema missing top-level property %q; properties: %v", want, keysOf(schema.Properties))
		}
	}
}

func keysOf(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

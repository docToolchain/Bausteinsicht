package schema

import (
	"encoding/json"
	"testing"
)

// TestStructure for testing schema generation
type TestConfig struct {
	Name   string `json:"name"`
	Active bool   `json:"active,omitempty"`
	Count  int    `json:"count,omitempty"`
}

type TestModel struct {
	Title       string      `json:"title"`
	Description string      `json:"description,omitempty"`
	Config      TestConfig  `json:"config"`
	Tags        []string    `json:"tags,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

func TestGeneratorBasic(t *testing.T) {
	gen := NewGenerator()
	schema := gen.Generate(TestModel{})

	if schema.Schema != "http://json-schema.org/draft-07/schema#" {
		t.Errorf("expected schema draft, got %s", schema.Schema)
	}

	if schema.Type != "object" {
		t.Errorf("expected type object, got %s", schema.Type)
	}

	if len(schema.Properties) == 0 {
		t.Error("expected properties to be generated")
	}
}

func TestGeneratorProperties(t *testing.T) {
	gen := NewGenerator()
	schema := gen.Generate(TestModel{})

	expectedProps := []string{"title", "description", "config", "tags", "metadata"}
	for _, prop := range expectedProps {
		if _, exists := schema.Properties[prop]; !exists {
			t.Errorf("expected property %s not found", prop)
		}
	}
}

func TestGeneratorRequired(t *testing.T) {
	gen := NewGenerator()
	schema := gen.Generate(TestModel{})

	// Only fields without omitempty should be required
	if len(schema.Required) == 0 {
		t.Error("expected some required fields")
	}

	// title is required (no omitempty)
	hasTitle := false
	for _, req := range schema.Required {
		if req == "title" {
			hasTitle = true
			break
		}
	}
	if !hasTitle {
		t.Error("expected 'title' to be required")
	}
}

func TestGeneratorJSON(t *testing.T) {
	gen := NewGenerator()
	schema := gen.Generate(TestModel{})

	jsonBytes, err := schema.ToJSON()
	if err != nil {
		t.Fatalf("failed to convert to JSON: %v", err)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		t.Fatalf("generated JSON is invalid: %v", err)
	}

	if result["type"] != "object" {
		t.Error("JSON schema type should be object")
	}

	if result["title"] != "Bausteinsicht Model" {
		t.Error("JSON schema title is incorrect")
	}
}

func TestGeneratorStringType(t *testing.T) {
	gen := NewGenerator()
	schema := gen.Generate(TestModel{})

	titleProp := schema.Properties["title"]
	titleMap, ok := titleProp.(map[string]interface{})
	if !ok {
		t.Fatalf("expected title property to be map, got %T", titleProp)
	}

	if titleMap["type"] != "string" {
		t.Errorf("expected string type, got %v", titleMap["type"])
	}
}

func TestGeneratorArrayType(t *testing.T) {
	gen := NewGenerator()
	schema := gen.Generate(TestModel{})

	tagsProp := schema.Properties["tags"]
	tagsMap, ok := tagsProp.(map[string]interface{})
	if !ok {
		t.Fatalf("expected tags property to be map, got %T", tagsProp)
	}

	if tagsMap["type"] != "array" {
		t.Errorf("expected array type, got %v", tagsMap["type"])
	}
}

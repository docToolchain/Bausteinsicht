package schema

import (
	"encoding/json"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// TestStructure for testing schema generation
type TestConfig struct {
	Name   string `json:"name"`
	Active bool   `json:"active,omitempty"`
	Count  int    `json:"count,omitempty"`
}

type TestModel struct {
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	Config      TestConfig        `json:"config"`
	Tags        []string          `json:"tags,omitempty"`
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

// asMap is a small helper that fails the test if v is not a JSON object.
func asMap(t *testing.T, v interface{}, ctx string) map[string]interface{} {
	t.Helper()
	m, ok := v.(map[string]interface{})
	if !ok {
		t.Fatalf("%s: expected object, got %T", ctx, v)
	}
	return m
}

// TestMapPropertyHasValueSchema guards issue #421: map-typed fields such as
// model (map[string]Element) and views (map[string]View) must carry an
// additionalProperties schema referencing the value type, otherwise the IDE
// has nothing to complete or validate inside model/views.
func TestMapPropertyHasValueSchema(t *testing.T) {
	gen := NewGenerator()
	s := gen.Generate(model.BausteinsichtModel{})

	cases := map[string]string{
		"model": "Element",
		"views": "View",
	}
	for prop, def := range cases {
		propSchema := asMap(t, s.Properties[prop], prop)
		ap, ok := propSchema["additionalProperties"]
		if !ok {
			t.Fatalf("%s: missing additionalProperties (no element/view completion)", prop)
		}
		apMap := asMap(t, ap, prop+".additionalProperties")
		wantRef := "#/definitions/" + def
		if apMap["$ref"] != wantRef {
			t.Errorf("%s: additionalProperties.$ref = %v, want %v", prop, apMap["$ref"], wantRef)
		}
	}
}

// TestStructsRejectUnknownFields guards that struct definitions set
// additionalProperties:false so a typo (e.g. "titel" instead of "title")
// is rejected by schema validation instead of passing silently.
func TestStructsRejectUnknownFields(t *testing.T) {
	gen := NewGenerator()
	s := gen.Generate(model.BausteinsichtModel{})

	if s.AdditionalProperties {
		t.Error("top-level schema must set additionalProperties:false")
	}

	for _, name := range []string{"Element", "View", "Relationship", "DynamicView", "SequenceStep", "Specification"} {
		def, ok := s.Definitions[name]
		if !ok {
			t.Errorf("definition %q not found", name)
			continue
		}
		defMap := asMap(t, def, name)
		ap, ok := defMap["additionalProperties"]
		if !ok {
			t.Errorf("%s: missing additionalProperties (typos pass silently)", name)
			continue
		}
		if b, _ := ap.(bool); b {
			t.Errorf("%s: additionalProperties must be false, got %v", name, ap)
		}
	}
}

// TestElementDefinitionHasProperties guards that the Element definition
// actually exposes its fields (title, kind, children) for completion, and
// that children recurses back into Element.
func TestElementDefinitionHasProperties(t *testing.T) {
	gen := NewGenerator()
	s := gen.Generate(model.BausteinsichtModel{})

	elem := asMap(t, s.Definitions["Element"], "Element")
	props := asMap(t, elem["properties"], "Element.properties")

	for _, field := range []string{"kind", "title", "children"} {
		if _, ok := props[field]; !ok {
			t.Errorf("Element definition missing property %q", field)
		}
	}

	children := asMap(t, props["children"], "Element.children")
	childAP := asMap(t, children["additionalProperties"], "Element.children.additionalProperties")
	if childAP["$ref"] != "#/definitions/Element" {
		t.Errorf("Element.children should recurse into Element, got %v", childAP["$ref"])
	}
}

// TestFreeFormMapStaysOpen guards that genuinely free-form maps
// (map[string]interface{}, e.g. the top-level meta field) are NOT locked
// down with a value schema, so arbitrary project metadata remains valid.
func TestFreeFormMapStaysOpen(t *testing.T) {
	gen := NewGenerator()
	s := gen.Generate(model.BausteinsichtModel{})

	meta := asMap(t, s.Properties["meta"], "meta")
	if _, locked := meta["additionalProperties"]; locked {
		t.Error("meta (map[string]interface{}) must stay an open object")
	}
}

package schema

import (
	"encoding/json"
	"reflect"
)

// JSONSchema represents a JSON Schema Draft 7 schema
type JSONSchema struct {
	Schema               string                 `json:"$schema"`
	Title                string                 `json:"title"`
	Description          string                 `json:"description,omitempty"`
	Type                 string                 `json:"type"`
	Properties           map[string]interface{} `json:"properties,omitempty"`
	Required             []string               `json:"required,omitempty"`
	AdditionalProperties bool                   `json:"additionalProperties"`
	Definitions          map[string]interface{} `json:"definitions,omitempty"`
}

// Generator generates JSON Schema from Go types
type Generator struct {
	definitions map[string]interface{}
}

// NewGenerator creates a new schema generator
func NewGenerator() *Generator {
	return &Generator{
		definitions: make(map[string]interface{}),
	}
}

// Generate generates JSON Schema for a given type
func (g *Generator) Generate(v interface{}) *JSONSchema {
	schema := &JSONSchema{
		Schema:               "http://json-schema.org/draft-07/schema#",
		Title:                "Bausteinsicht Model",
		Description:          "Architecture model in Bausteinsicht format",
		Type:                 "object",
		Properties:           make(map[string]interface{}),
		AdditionalProperties: false,
		Definitions:          g.definitions,
	}

	// Generate properties from struct fields
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr { //nolint:govet
		t = t.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		fieldName := jsonTag
		if idx := findComma(jsonTag); idx >= 0 {
			fieldName = jsonTag[:idx]
		}

		schema.Properties[fieldName] = g.generateFieldSchema(field.Type)

		// Add to required if no omitempty
		if !hasOmitempty(jsonTag) {
			schema.Required = append(schema.Required, fieldName)
		}
	}

	return schema
}

// generateFieldSchema generates schema for a single field
func (g *Generator) generateFieldSchema(t reflect.Type) interface{} {
	if t.Kind() == reflect.Ptr { //nolint:govet
		return g.generateFieldSchema(t.Elem())
	}

	switch t.Kind() {
	case reflect.String:
		return map[string]interface{}{"type": "string"}
	case reflect.Bool:
		return map[string]interface{}{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]interface{}{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]interface{}{"type": "number"}
	case reflect.Slice, reflect.Array:
		return map[string]interface{}{
			"type":  "array",
			"items": g.generateFieldSchema(t.Elem()),
		}
	case reflect.Map:
		// A map serializes to a JSON object whose values all share the
		// element type. Emit additionalProperties so the schema validates
		// each entry (e.g. map[string]Element validates every element).
		valueType := t.Elem()
		// map[string]interface{} stays an open free-form object.
		if valueType.Kind() == reflect.Interface {
			return map[string]interface{}{
				"type": "object",
			}
		}
		return map[string]interface{}{
			"type":                 "object",
			"additionalProperties": g.generateFieldSchema(valueType),
		}
	case reflect.Struct:
		return g.generateObjectSchema(t)
	default:
		return map[string]interface{}{"type": "object"}
	}
}

// generateObjectSchema generates schema for a struct type
func (g *Generator) generateObjectSchema(t reflect.Type) interface{} {
	typeName := t.Name()
	if typeName == "" {
		return map[string]interface{}{"type": "object"}
	}

	// Check if already defined
	if _, exists := g.definitions[typeName]; exists {
		return map[string]interface{}{"$ref": "#/definitions/" + typeName}
	}

	schema := map[string]interface{}{
		"type":                 "object",
		"properties":           make(map[string]interface{}),
		"additionalProperties": false,
	}

	// Register the definition before recursing into fields so that
	// self-referential types (e.g. Element.Children map[string]Element)
	// resolve to a $ref instead of recursing forever. The schema map is a
	// reference type, so later mutations are reflected in the registry.
	g.definitions[typeName] = schema

	properties := schema["properties"].(map[string]interface{})
	required := []string{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		fieldName := jsonTag
		if idx := findComma(jsonTag); idx >= 0 {
			fieldName = jsonTag[:idx]
		}

		properties[fieldName] = g.generateFieldSchema(field.Type)

		if !hasOmitempty(jsonTag) {
			required = append(required, fieldName)
		}
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return map[string]interface{}{"$ref": "#/definitions/" + typeName}
}

// ToJSON returns the schema as formatted JSON
func (s *JSONSchema) ToJSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// helper functions

func findComma(s string) int {
	for i, c := range s {
		if c == ',' {
			return i
		}
	}
	return -1
}

func hasOmitempty(jsonTag string) bool {
	idx := findComma(jsonTag)
	if idx < 0 {
		return false
	}
	return json.Unmarshal([]byte(`"`+jsonTag[idx+1:]+`"`), new(string)) == nil &&
		contains(jsonTag[idx+1:], "omitempty")
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

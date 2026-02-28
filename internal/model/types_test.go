package model

import (
	"encoding/json"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
)


func stripComments(jsonc string) string {
	// Remove single-line comments (// ...) but not // inside strings
	// Match either a string literal (keep) or a comment (remove)
	re := regexp.MustCompile(`("(?:[^"\\]|\\.)*")|(?://[^\n]*)`)
	return re.ReplaceAllStringFunc(jsonc, func(m string) string {
		if strings.HasPrefix(m, `"`) {
			return m // keep string literals unchanged
		}
		return "" // strip comment
	})
}

func TestJSONRoundTrip(t *testing.T) {
	original := BausteinsichtModel{
		Schema: "https://example.com/schema.json",
		Specification: Specification{
			Elements: map[string]ElementKind{
				"actor": {Notation: "Actor", Description: "A person"},
				"system": {Notation: "Software System", Container: true},
			},
			Relationships: map[string]RelationshipKind{
				"uses": {Notation: "uses"},
				"async": {Notation: "async", Dashed: true},
			},
		},
		Model: map[string]Element{
			"user": {Kind: "actor", Title: "User", Description: "End user"},
			"app": {
				Kind:  "system",
				Title: "App",
				Children: map[string]Element{
					"api": {Kind: "container", Title: "API", Technology: "Go"},
				},
			},
		},
		Relationships: []Relationship{
			{From: "user", To: "app", Label: "uses", Kind: "uses", Description: "Interacts"},
		},
		Views: map[string]View{
			"context": {Title: "Context", Scope: "app", Include: []string{"user", "app"}, Description: "Overview"},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var restored BausteinsichtModel
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if !reflect.DeepEqual(original, restored) {
		t.Errorf("round-trip mismatch:\noriginal: %+v\nrestored: %+v", original, restored)
	}
}

func TestDeserializeSampleModel(t *testing.T) {
	raw, err := os.ReadFile("../../templates/sample-model.jsonc")
	if err != nil {
		t.Fatalf("failed to read sample model: %v", err)
	}
	stripped := stripComments(string(raw))

	var m BausteinsichtModel
	if err := json.Unmarshal([]byte(stripped), &m); err != nil {
		t.Fatalf("failed to parse sample model: %v", err)
	}

	if m.Schema == "" {
		t.Error("expected $schema to be set")
	}
	if len(m.Specification.Elements) == 0 {
		t.Error("expected specification.elements to be non-empty")
	}
	if len(m.Model) == 0 {
		t.Error("expected model to be non-empty")
	}
	if len(m.Relationships) == 0 {
		t.Error("expected relationships to be non-empty")
	}
	if len(m.Views) == 0 {
		t.Error("expected views to be non-empty")
	}

	// Check specific known element
	onlineshop, ok := m.Model["onlineshop"]
	if !ok {
		t.Fatal("expected 'onlineshop' in model")
	}
	if onlineshop.Title != "Online Shop" {
		t.Errorf("expected title 'Online Shop', got '%s'", onlineshop.Title)
	}
	if len(onlineshop.Children) == 0 {
		t.Error("expected onlineshop to have children")
	}
}

func TestOptionalFieldsOmitted(t *testing.T) {
	m := BausteinsichtModel{
		Specification: Specification{
			Elements: map[string]ElementKind{
				"system": {Notation: "System"},
			},
		},
		Model: map[string]Element{
			"app": {Kind: "system", Title: "App"},
		},
		Relationships: []Relationship{},
		Views:         map[string]View{},
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	s := string(data)
	if strings.Contains(s, `"$schema"`) {
		t.Error("empty $schema should be omitted")
	}
	if strings.Contains(s, `"description"`) {
		t.Error("empty description should be omitted")
	}
	if strings.Contains(s, `"technology"`) {
		t.Error("empty technology should be omitted")
	}
	if strings.Contains(s, `"tags"`) {
		t.Error("empty tags should be omitted")
	}
	if strings.Contains(s, `"children"`) {
		t.Error("empty children should be omitted")
	}
	if strings.Contains(s, `"metadata"`) {
		t.Error("empty metadata should be omitted")
	}
}

func TestNestedChildren(t *testing.T) {
	m := BausteinsichtModel{
		Specification: Specification{
			Elements: map[string]ElementKind{
				"system":    {Notation: "System", Container: true},
				"container": {Notation: "Container", Container: true},
				"component": {Notation: "Component"},
			},
		},
		Model: map[string]Element{
			"app": {
				Kind:  "system",
				Title: "App",
				Children: map[string]Element{
					"api": {
						Kind:  "container",
						Title: "API",
						Children: map[string]Element{
							"handler": {Kind: "component", Title: "Handler"},
						},
					},
				},
			},
		},
		Relationships: []Relationship{},
		Views:         map[string]View{},
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var restored BausteinsichtModel
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	app := restored.Model["app"]
	api, ok := app.Children["api"]
	if !ok {
		t.Fatal("expected 'api' child in 'app'")
	}
	handler, ok := api.Children["handler"]
	if !ok {
		t.Fatal("expected 'handler' child in 'api'")
	}
	if handler.Title != "Handler" {
		t.Errorf("expected handler title 'Handler', got '%s'", handler.Title)
	}
}

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
				"actor":  {Notation: "Actor", Description: "A person"},
				"system": {Notation: "Software System", Container: true},
			},
			Relationships: map[string]RelationshipKind{
				"uses":  {Notation: "uses"},
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

// TestElement_CodeMapping_JSONRoundTrip verifies codeMapping round-trips
// through JSON. Shape is flat (Element -> one CodeMapping), matching the
// validated Variant C prototype contract, not a per-backend map — see #562,
// ADR-013, and https://github.com/docToolchain/bausteinsicht-archunit.
func TestElement_CodeMapping_JSONRoundTrip(t *testing.T) {
	original := Element{
		Kind:        "container",
		Title:       "Order API",
		CodeMapping: &CodeMapping{Packages: []string{"com.acme.backend.orders"}},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var restored Element
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if !reflect.DeepEqual(original, restored) {
		t.Errorf("round-trip mismatch:\noriginal: %+v\nrestored: %+v", original, restored)
	}
}

// TestElement_CodeMapping_MatchesArchUnitPrototypeContract pins the exact
// JSON shape read by the running bausteinsicht-archunit Variant C prototype
// (adapter/src/test/resources/architecture.jsonc there), so a future change
// here doesn't silently break that external, already-validated adapter.
func TestElement_CodeMapping_MatchesArchUnitPrototypeContract(t *testing.T) {
	raw := `{"kind":"component","title":"API","codeMapping":{"packages":["com.acme.backend.api"]}}`

	var elem Element
	if err := json.Unmarshal([]byte(raw), &elem); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if elem.CodeMapping == nil {
		t.Fatal("expected CodeMapping to be set")
	}
	want := []string{"com.acme.backend.api"}
	if !reflect.DeepEqual(elem.CodeMapping.Packages, want) {
		t.Errorf("Packages = %v, want %v", elem.CodeMapping.Packages, want)
	}
}

// TestElement_CodeMapping_OmittedWhenAbsent ensures elements without a
// codeMapping don't grow a stray "codeMapping": null/{} in the serialized
// JSONC, keeping existing models byte-for-byte unaffected by this addition.
func TestElement_CodeMapping_OmittedWhenAbsent(t *testing.T) {
	data, err := json.Marshal(Element{Kind: "actor", Title: "Customer"})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(data), "codeMapping") {
		t.Errorf("expected no codeMapping key when unset, got: %s", data)
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

func TestSpecification_IsDashed(t *testing.T) {
	spec := &Specification{
		Relationships: map[string]RelationshipKind{
			"uses":  {Notation: "uses"},
			"async": {Notation: "async", Dashed: true},
		},
	}

	cases := []struct {
		name string
		spec *Specification
		kind string
		want bool
	}{
		{"dashed kind", spec, "async", true},
		{"non-dashed kind", spec, "uses", false},
		{"unknown kind", spec, "nonexistent", false},
		{"empty kind", spec, "", false},
		{"nil specification", nil, "async", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.spec.IsDashed(c.kind); got != c.want {
				t.Errorf("IsDashed(%q) = %v, want %v", c.kind, got, c.want)
			}
		})
	}
}

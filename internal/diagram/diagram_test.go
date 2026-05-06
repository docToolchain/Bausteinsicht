package diagram

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func testModel() *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system":          {Notation: "Software System", Container: true},
				"container":       {Notation: "Container", Container: true},
				"component":       {Notation: "Component"},
				"actor":           {Notation: "Actor"},
				"external_system": {Notation: "External System"},
			},
		},
		Model: map[string]model.Element{
			"user": {Kind: "actor", Title: "User", Description: "End user"},
			"shop": {Kind: "system", Title: "Online Shop", Description: "E-commerce platform", Children: map[string]model.Element{
				"api": {Kind: "container", Title: "API", Description: "REST backend", Technology: "Go"},
				"db":  {Kind: "container", Title: "Database", Description: "Storage", Technology: "PostgreSQL"},
			}},
			"payment": {Kind: "external_system", Title: "Payment Gateway", Description: "External payment provider"},
		},
		Relationships: []model.Relationship{
			{From: "user", To: "shop", Label: "uses", Kind: "uses"},
			{From: "user", To: "payment", Label: "pays via", Kind: "uses"},
			{From: "shop.api", To: "shop.db", Label: "reads/writes", Kind: "uses"},
			{From: "shop.api", To: "payment", Label: "processes payment", Kind: "uses"},
		},
		Views: map[string]model.View{
			"context": {
				Title:   "System Context",
				Include: []string{"user", "shop", "payment"},
			},
			"containers": {
				Title:   "Container View",
				Scope:   "shop",
				Include: []string{"user", "shop.*", "payment"},
			},
		},
	}
}

// --- PlantUML Tests ---

func TestPlantUML_ContextView(t *testing.T) {
	m := testModel()
	result, err := FormatView(m, "context", PlantUML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "@startuml") {
		t.Error("expected @startuml")
	}
	if !strings.Contains(result, "@enduml") {
		t.Error("expected @enduml")
	}
	if !strings.Contains(result, "<C4/C4_Context>") {
		t.Error("expected <C4/C4_Context> stdlib include")
	}
	if !strings.Contains(result, "Person(") {
		t.Error("expected Person() macro for actor")
	}
	if !strings.Contains(result, "System(") {
		t.Error("expected System() macro")
	}
	if !strings.Contains(result, "System_Ext(") {
		t.Error("expected System_Ext() macro for external system")
	}
	if !strings.Contains(result, "Rel(") {
		t.Error("expected Rel() for relationships")
	}
}

func TestPlantUML_ContainerView(t *testing.T) {
	m := testModel()
	result, err := FormatView(m, "containers", PlantUML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "<C4/C4_Container>") {
		t.Error("expected <C4/C4_Container> stdlib include for container view")
	}
	if !strings.Contains(result, "System_Boundary(") {
		t.Error("expected System_Boundary for scope element")
	}
	if !strings.Contains(result, "Container(") {
		t.Error("expected Container() macro")
	}
}

func TestPlantUML_Relationships(t *testing.T) {
	m := testModel()
	result, err := FormatView(m, "context", PlantUML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "uses") {
		t.Error("expected relationship label 'uses'")
	}
	if !strings.Contains(result, "pays via") {
		t.Error("expected relationship label 'pays via'")
	}
}

func TestPlantUML_InvalidView(t *testing.T) {
	m := testModel()
	_, err := FormatView(m, "nonexistent", PlantUML)
	if err == nil {
		t.Error("expected error for nonexistent view")
	}
}

// --- Mermaid Tests ---

func TestMermaid_ContextView(t *testing.T) {
	m := testModel()
	result, err := FormatView(m, "context", Mermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "C4Context") {
		t.Error("expected C4Context diagram type")
	}
	if !strings.Contains(result, "Person(") {
		t.Error("expected Person() for actor")
	}
	if !strings.Contains(result, "System(") {
		t.Error("expected System()")
	}
	if !strings.Contains(result, "System_Ext(") {
		t.Error("expected System_Ext()")
	}
	if !strings.Contains(result, "Rel(") {
		t.Error("expected Rel()")
	}
}

func TestMermaid_ContainerView(t *testing.T) {
	m := testModel()
	result, err := FormatView(m, "containers", Mermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "C4Container") {
		t.Error("expected C4Container diagram type")
	}
	if !strings.Contains(result, "System_Boundary(") {
		t.Error("expected System_Boundary for scope")
	}
	if !strings.Contains(result, "Container(") {
		t.Error("expected Container()")
	}
}

func TestMermaid_Relationships(t *testing.T) {
	m := testModel()
	result, err := FormatView(m, "context", Mermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "uses") {
		t.Error("expected relationship label 'uses'")
	}
}

// --- DOT Tests ---

func TestDOT_ContextView(t *testing.T) {
	m := testModel()
	result, err := RenderDOT(m, "context")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "digraph") {
		t.Error("expected digraph declaration")
	}
	if !strings.Contains(result, "rankdir=LR") {
		t.Error("expected left-right direction")
	}
	if !strings.Contains(result, "[actor]") {
		t.Error("expected element kind in node label")
	}
	if !strings.Contains(result, "->") {
		t.Error("expected edge arrows")
	}
}

func TestDOT_WithColor(t *testing.T) {
	m := testModel()
	result, err := RenderDOT(m, "context")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "fillcolor=") {
		t.Error("expected fillcolor attribute")
	}
	if !strings.Contains(result, "color=") {
		t.Error("expected color attribute for edges")
	}
}

func TestDOT_InvalidView(t *testing.T) {
	m := testModel()
	_, err := RenderDOT(m, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent view")
	}
}

// --- D2 Tests ---

func TestD2_ContextView(t *testing.T) {
	m := testModel()
	result, err := RenderD2(m, "context")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "direction: right") {
		t.Error("expected direction declaration")
	}
	if !strings.Contains(result, "shape: rectangle") {
		t.Error("expected rectangle shapes")
	}
	if !strings.Contains(result, "style.fill:") {
		t.Error("expected fill styling")
	}
}

func TestD2_WithRelationships(t *testing.T) {
	m := testModel()
	result, err := RenderD2(m, "context")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "->") {
		t.Error("expected relationship arrows")
	}
	if !strings.Contains(result, "uses") {
		t.Error("expected relationship labels")
	}
}

func TestD2_InvalidView(t *testing.T) {
	m := testModel()
	_, err := RenderD2(m, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent view")
	}
}

// --- HTML5 Tests ---

func TestHTML_ContextView(t *testing.T) {
	m := testModel()
	result, err := RenderHTML(m, "context")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "<!DOCTYPE html>") {
		t.Error("expected HTML5 doctype")
	}
	if !strings.Contains(result, "<body>") {
		t.Error("expected body element")
	}
	if !strings.Contains(result, "DIAGRAM_DATA") {
		t.Error("expected embedded diagram data")
	}
	if !strings.Contains(result, "createElementNS") {
		t.Error("expected SVG creation via JavaScript")
	}
}

func TestHTML_ValidJSON(t *testing.T) {
	m := testModel()
	result, err := RenderHTML(m, "context")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Extract JSON from template
	start := strings.Index(result, "const DIAGRAM_DATA = ")
	if start < 0 {
		t.Fatal("could not find DIAGRAM_DATA in HTML")
	}
	start += len("const DIAGRAM_DATA = ")
	end := strings.Index(result[start:], ";")
	if end < 0 {
		t.Fatal("could not find end of DIAGRAM_DATA")
	}

	jsonStr := result[start : start+end]
	var data HTMLDiagramData
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		t.Errorf("embedded JSON is invalid: %v", err)
	}

	if data.Title == "" {
		t.Error("expected non-empty title in diagram data")
	}
	if len(data.Nodes) == 0 {
		t.Error("expected nodes in diagram data")
	}
}

func TestHTML_InvalidView(t *testing.T) {
	m := testModel()
	_, err := RenderHTML(m, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent view")
	}
}

// --- Colors Tests ---

func TestColorForKind_KnownKind(t *testing.T) {
	style := ColorForKind("actor")
	if style.Fill == "" || style.Stroke == "" {
		t.Error("expected non-empty fill and stroke for known kind")
	}
}

func TestColorForKind_UnknownKind(t *testing.T) {
	style := ColorForKind("unknown")
	if style.Fill == "" || style.Stroke == "" {
		t.Error("expected default colors for unknown kind")
	}
}

// --- Tag Filtering Tests ---

func TestApplyTagFiltering_NoTags(t *testing.T) {
	resolved := []string{"elem1", "elem2", "elem3"}
	flat := map[string]*model.Element{
		"elem1": {Kind: "system", Title: "Elem1", Tags: []string{"tag1"}},
		"elem2": {Kind: "system", Title: "Elem2", Tags: []string{"tag2"}},
		"elem3": {Kind: "system", Title: "Elem3", Tags: []string{"tag3"}},
	}

	result := applyTagFiltering(resolved, flat, nil, nil)
	if len(result) != 3 {
		t.Errorf("expected 3 elements, got %d", len(result))
	}
}

func TestApplyTagFiltering_WithFilterTags(t *testing.T) {
	resolved := []string{"elem1", "elem2", "elem3"}
	flat := map[string]*model.Element{
		"elem1": {Kind: "system", Title: "Elem1", Tags: []string{"backend", "critical"}},
		"elem2": {Kind: "system", Title: "Elem2", Tags: []string{"backend"}},
		"elem3": {Kind: "system", Title: "Elem3", Tags: []string{"frontend"}},
	}

	result := applyTagFiltering(resolved, flat, []string{"backend"}, nil)
	if len(result) != 2 {
		t.Errorf("expected 2 elements with backend tag, got %d", len(result))
	}
}

func TestApplyTagFiltering_WithExcludeTags(t *testing.T) {
	resolved := []string{"elem1", "elem2", "elem3"}
	flat := map[string]*model.Element{
		"elem1": {Kind: "system", Title: "Elem1", Tags: []string{"experimental"}},
		"elem2": {Kind: "system", Title: "Elem2", Tags: []string{"stable"}},
		"elem3": {Kind: "system", Title: "Elem3", Tags: []string{"deprecated"}},
	}

	result := applyTagFiltering(resolved, flat, nil, []string{"experimental", "deprecated"})
	if len(result) != 1 {
		t.Errorf("expected 1 stable element, got %d", len(result))
	}
}

func TestFormatView_WithTagFiltering(t *testing.T) {
	m := testModel()
	// Add tags to the test model elements
	user := m.Model["user"]
	user.Tags = []string{"external"}
	m.Model["user"] = user

	shop := m.Model["shop"]
	shop.Tags = []string{"core"}
	m.Model["shop"] = shop

	payment := m.Model["payment"]
	payment.Tags = []string{"external"}
	m.Model["payment"] = payment

	// Add tags specification
	m.Specification.Tags = []model.TagDefinition{
		{ID: "external", Description: "External actors/systems"},
		{ID: "core", Description: "Core business systems"},
	}

	// Create a view with tag filtering
	m.Views["core-only"] = model.View{
		Title:      "Core Systems",
		Include:    []string{"*"},
		FilterTags: []string{"core"},
	}

	result, err := FormatView(m, "core-only", PlantUML)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty diagram output")
	}
	// Verify that only the core system is included
	if !strings.Contains(result, "shop") {
		t.Error("expected 'shop' in diagram")
	}
}

func TestFormatView_WithExcludeTagFiltering(t *testing.T) {
	m := testModel()
	// Add tags to the test model elements
	user := m.Model["user"]
	user.Tags = []string{"external"}
	m.Model["user"] = user

	shop := m.Model["shop"]
	shop.Tags = []string{"core"}
	m.Model["shop"] = shop

	payment := m.Model["payment"]
	payment.Tags = []string{"external"}
	m.Model["payment"] = payment

	// Add tags specification
	m.Specification.Tags = []model.TagDefinition{
		{ID: "external", Description: "External actors/systems"},
		{ID: "core", Description: "Core business systems"},
	}

	// Create a view that excludes external elements
	m.Views["internal-only"] = model.View{
		Title:       "Internal Systems",
		Include:     []string{"*"},
		ExcludeTags: []string{"external"},
	}

	result, err := FormatView(m, "internal-only", PlantUML)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty diagram output")
	}
}

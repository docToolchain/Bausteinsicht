package table

import (
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func testModel() *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system":    {Notation: "Software System", Container: true},
				"container": {Notation: "Container", Container: true},
				"component": {Notation: "Component"},
				"actor":     {Notation: "Actor"},
			},
		},
		Model: map[string]model.Element{
			"user": {Kind: "actor", Title: "User", Description: "End user of the application"},
			"shop": {Kind: "system", Title: "Online Shop", Description: "E-commerce platform", Children: map[string]model.Element{
				"api": {Kind: "container", Title: "API", Description: "REST API backend", Technology: "Go / Gin"},
				"db":  {Kind: "container", Title: "Database", Description: "Persistent storage", Technology: "PostgreSQL"},
			}},
		},
		Views: map[string]model.View{
			"context": {
				Title:   "System Context",
				Include: []string{"user", "shop"},
			},
			"containers": {
				Title:   "Container View",
				Scope:   "shop",
				Include: []string{"user", "shop.*"},
			},
		},
	}
}

func TestFormatAsciiDoc_SingleView(t *testing.T) {
	m := testModel()

	result, err := FormatView(m, "containers", AsciiDoc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain a table header
	if !strings.Contains(result, "|===") {
		t.Error("expected AsciiDoc table delimiter |===")
	}
	// Should contain column headers
	if !strings.Contains(result, "Element") || !strings.Contains(result, "Technology") {
		t.Error("expected column headers Element and Technology")
	}
	// Should contain the view title
	if !strings.Contains(result, "Container View") {
		t.Error("expected view title 'Container View'")
	}
	// Should contain element data
	if !strings.Contains(result, "API") {
		t.Error("expected element 'API' in output")
	}
	if !strings.Contains(result, "Go / Gin") {
		t.Error("expected technology 'Go / Gin' in output")
	}
	if !strings.Contains(result, "Database") {
		t.Error("expected element 'Database' in output")
	}
}

func TestFormatMarkdown_SingleView(t *testing.T) {
	m := testModel()

	result, err := FormatView(m, "containers", Markdown)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Markdown tables use | and --- separators
	if !strings.Contains(result, "| Element") {
		t.Error("expected Markdown table header with '| Element'")
	}
	if !strings.Contains(result, "---") {
		t.Error("expected Markdown separator row with ---")
	}
	if !strings.Contains(result, "API") {
		t.Error("expected element 'API' in output")
	}
}

func TestFormatAllViews(t *testing.T) {
	m := testModel()

	result, err := FormatAllViews(m, AsciiDoc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain both view titles
	if !strings.Contains(result, "System Context") {
		t.Error("expected view title 'System Context'")
	}
	if !strings.Contains(result, "Container View") {
		t.Error("expected view title 'Container View'")
	}
}

func TestFormatCombined(t *testing.T) {
	m := testModel()

	result, err := FormatCombined(m, AsciiDoc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain all elements, deduplicated
	if !strings.Contains(result, "Online Shop") {
		t.Error("expected 'Online Shop' in combined output")
	}
	if !strings.Contains(result, "API") {
		t.Error("expected 'API' in combined output")
	}
	// Should not duplicate elements
	if strings.Count(result, "| API") > 1 {
		t.Error("expected 'API' to appear only once in combined output (deduplicated)")
	}
}

func TestFormatView_InvalidView(t *testing.T) {
	m := testModel()

	_, err := FormatView(m, "nonexistent", AsciiDoc)
	if err == nil {
		t.Error("expected error for nonexistent view")
	}
}

func TestFormatView_ElementsSorted(t *testing.T) {
	m := testModel()

	result, err := FormatView(m, "containers", AsciiDoc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// API should appear before Database (sorted by ID: shop.api < shop.db)
	apiIdx := strings.Index(result, "API")
	dbIdx := strings.Index(result, "Database")
	if apiIdx < 0 || dbIdx < 0 {
		t.Fatal("expected both API and Database in output")
	}
	if apiIdx > dbIdx {
		t.Error("expected elements to be sorted (API before Database)")
	}
}

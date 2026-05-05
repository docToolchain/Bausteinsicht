package template

import (
	"strings"
	"testing"

	etree "github.com/beevik/etree"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func testSpec() model.Specification {
	return model.Specification{
		Elements: map[string]model.ElementKind{
			"person":          {Notation: "Person"},
			"system":          {Notation: "System", Container: true},
			"database":        {Notation: "Database"},
			"external_system": {Notation: "External System"},
		},
	}
}

func TestGenerateTemplateXML(t *testing.T) {
	gen := NewGenerator(testSpec(), "default")
	result := gen.Generate()

	if !strings.Contains(result, "<?xml") {
		t.Error("expected XML declaration")
	}
	if !strings.Contains(result, "<mxfile") {
		t.Error("expected mxfile root element")
	}
	if !strings.Contains(result, "<diagram") {
		t.Error("expected diagram element")
	}
	if !strings.Contains(result, "<mxGraphModel") {
		t.Error("expected mxGraphModel element")
	}
	if !strings.Contains(result, "</mxGraphModel>") {
		t.Error("expected mxGraphModel closing tag")
	}
}

func TestGenerateTemplateHasAllKinds(t *testing.T) {
	gen := NewGenerator(testSpec(), "default")
	result := gen.Generate()

	kinds := []string{"person", "system", "database", "external_system"}
	for _, kind := range kinds {
		if !strings.Contains(result, "["+kind+"]") {
			t.Errorf("expected kind %q in output", kind)
		}
	}
}

func TestGenerateTemplateCanParse(t *testing.T) {
	gen := NewGenerator(testSpec(), "default")
	result := gen.Generate()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(result); err != nil {
		t.Fatalf("generated XML is not valid: %v", err)
	}

	mxfile := doc.Root()
	if mxfile == nil || mxfile.Tag != "mxfile" {
		t.Error("expected mxfile as root element")
	}
	diagram := mxfile.SelectElement("diagram")
	if diagram == nil {
		t.Error("expected diagram element in mxfile")
	}
	graphModel := diagram.SelectElement("mxGraphModel")
	if graphModel == nil {
		t.Error("expected mxGraphModel element in diagram")
	}
}

func TestGenerateTemplateStyles(t *testing.T) {
	tests := []struct {
		style string
		name  string
	}{
		{"default", "default style"},
		{"c4", "c4 style"},
		{"minimal", "minimal style"},
		{"dark", "dark style"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewGenerator(testSpec(), tt.style)
			result := gen.Generate()

			if result == "" {
				t.Error("expected non-empty result")
			}
			if !strings.Contains(result, "fillColor=") {
				t.Error("expected fillColor in style")
			}
		})
	}
}

func TestGenerateTemplateInvalidStyle(t *testing.T) {
	gen := NewGenerator(testSpec(), "nonexistent")
	result := gen.Generate()

	// Should use default style as fallback
	if !strings.Contains(result, "fillColor=") {
		t.Error("expected fillColor in style (fallback)")
	}
}

func TestColorForKind(t *testing.T) {
	tests := []struct {
		preset string
		kind   string
	}{
		{"default", "person"},
		{"c4", "system"},
		{"minimal", "database"},
		{"dark", "external_system"},
	}

	for _, tt := range tests {
		t.Run(tt.preset+"/"+tt.kind, func(t *testing.T) {
			color := ColorForKind(tt.preset, tt.kind)
			if color.Fill == "" || color.Stroke == "" {
				t.Error("expected non-empty fill and stroke")
			}
		})
	}
}

func TestColorForKindFallback(t *testing.T) {
	color := ColorForKind("nonexistent", "nonexistent")
	if color.Fill == "" || color.Stroke == "" {
		t.Error("expected fallback colors for unknown preset/kind")
	}
}

func TestDefaultShapeConfig(t *testing.T) {
	tests := []struct {
		kind          string
		expectedShape string
	}{
		{"person", "mxgraph.archimate3.actor"},
		{"system", "rounded=1"},
		{"database", "mxgraph.flowchart.database"},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			cfg := DefaultShapeConfig(tt.kind)
			if cfg.Shape != tt.expectedShape {
				t.Errorf("expected shape %q, got %q", tt.expectedShape, cfg.Shape)
			}
			if cfg.Width <= 0 || cfg.Height <= 0 {
				t.Error("expected positive width and height")
			}
		})
	}
}

func TestDefaultShapeConfigUnknown(t *testing.T) {
	cfg := DefaultShapeConfig("unknown_kind")
	if cfg.Shape == "" || cfg.Width == 0 || cfg.Height == 0 {
		t.Error("expected default shape for unknown kind")
	}
}

func TestGridLayout(t *testing.T) {
	kinds := []string{"a", "b", "c", "d", "e"}
	layout := GridLayout(kinds, 2)

	if len(layout) != len(kinds) {
		t.Errorf("expected %d elements, got %d", len(kinds), len(layout))
	}

	// Verify all kinds are included
	kindMap := make(map[string]bool)
	for _, elem := range layout {
		kindMap[elem.Kind] = true
	}
	for _, kind := range kinds {
		if !kindMap[kind] {
			t.Errorf("expected kind %q in layout", kind)
		}
	}
}

func TestGenerateTemplateMultipleKinds(t *testing.T) {
	spec := model.Specification{
		Elements: map[string]model.ElementKind{
			"person":          {Notation: "Person"},
			"system":          {Notation: "System"},
			"database":        {Notation: "Database"},
			"cache":           {Notation: "Cache"},
			"queue":           {Notation: "Queue"},
			"component":       {Notation: "Component"},
			"container":       {Notation: "Container"},
			"external_system": {Notation: "External System"},
		},
	}

	gen := NewGenerator(spec, "default")
	result := gen.Generate()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(result); err != nil {
		t.Fatalf("generated XML is not valid: %v", err)
	}

	// Count mxCell elements
	mxfile := doc.Root()
	diagram := mxfile.SelectElement("diagram")
	graphModel := diagram.SelectElement("mxGraphModel")
	if graphModel == nil {
		t.Fatal("expected mxGraphModel element")
	}

	root := graphModel.SelectElement("root")
	if root == nil {
		t.Fatal("expected root element")
	}

	cells := root.FindElements("mxCell")
	// Should have: 2 base cells + 8 element cells
	if len(cells) < 10 {
		t.Errorf("expected at least 10 cells, got %d", len(cells))
	}
}

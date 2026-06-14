package template

import (
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// roundtripSpec covers element kinds, container kinds (which require boundary
// templates), and a non-container leaf kind.
func roundtripSpec() model.Specification {
	return model.Specification{
		Elements: map[string]model.ElementKind{
			"person":          {Notation: "Person"},
			"system":          {Notation: "System", Container: true},
			"container":       {Notation: "Container", Container: true},
			"component":       {Notation: "Component"},
			"database":        {Notation: "Database"},
			"external_system": {Notation: "External System"},
		},
	}
}

// TestGenerateTemplateRoundtrip verifies that a generated template parses back
// through drawio.LoadTemplate with a style for every specification kind — i.e. it
// is a working template, not just a visual palette. Regression test for #419.
func TestGenerateTemplateRoundtrip(t *testing.T) {
	for _, style := range []string{"default", "c4", "minimal", "dark"} {
		t.Run(style, func(t *testing.T) {
			spec := roundtripSpec()
			gen := NewGenerator(spec, style)
			xml := gen.Generate()

			ts, err := drawio.LoadTemplateFromBytes([]byte(xml))
			if err != nil {
				t.Fatalf("LoadTemplateFromBytes: %v", err)
			}

			// Every element kind must resolve to a style (no "no template style" fallback).
			for kind := range spec.Elements {
				if _, ok := ts.GetStyle(kind); !ok {
					t.Errorf("no template style for kind: %s", kind)
				}
			}

			// Container kinds must produce a usable boundary template.
			for kind, def := range spec.Elements {
				if !def.Container {
					continue
				}
				if _, ok := ts.GetBoundaryStyle(kind + "_boundary"); !ok {
					t.Errorf("no boundary template for container kind: %s", kind)
				}
			}

			// A relationship connector style must be present.
			if ts.GetConnectorStyle() == "" {
				t.Error("no relationship connector style")
			}
		})
	}
}

// TestGenerateTemplateHasSubCells verifies the generated element templates carry
// the title/tech/desc sub-cells the sync engine reads for grouped text labels.
func TestGenerateTemplateHasSubCells(t *testing.T) {
	gen := NewGenerator(roundtripSpec(), "default")
	xml := gen.Generate()

	ts, err := drawio.LoadTemplateFromBytes([]byte(xml))
	if err != nil {
		t.Fatalf("LoadTemplateFromBytes: %v", err)
	}

	style, ok := ts.GetStyle("system")
	if !ok {
		t.Fatal("no style for system")
	}
	if style.TitleStyle == nil {
		t.Error("expected title sub-cell for system")
	}
	if style.TechStyle == nil {
		t.Error("expected tech sub-cell for system")
	}
	if style.DescStyle == nil {
		t.Error("expected desc sub-cell for system")
	}
}

// TestGenerateTemplateHasVersion verifies the version attribute is emitted so
// LoadTemplate treats the output as a current-format template.
func TestGenerateTemplateHasVersion(t *testing.T) {
	gen := NewGenerator(roundtripSpec(), "default")
	xml := gen.Generate()

	if !strings.Contains(xml, `bausteinsicht_template_version="1"`) {
		t.Error("expected bausteinsicht_template_version attribute on output")
	}

	ts, err := drawio.LoadTemplateFromBytes([]byte(xml))
	if err != nil {
		t.Fatalf("LoadTemplateFromBytes: %v", err)
	}
	if ts.Version != drawio.CurrentTemplateVersion {
		t.Errorf("expected version %d, got %d", drawio.CurrentTemplateVersion, ts.Version)
	}
}

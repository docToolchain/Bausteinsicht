package drawio_test

import (
	"os"
	"testing"

	"github.com/docToolchain/Bauteinsicht/internal/drawio"
)

const defaultTemplatePath = "../../templates/default.drawio"

func TestLoadTemplate_Success(t *testing.T) {
	ts, err := drawio.LoadTemplate(defaultTemplatePath)
	if err != nil {
		t.Fatalf("LoadTemplate: %v", err)
	}
	if ts == nil {
		t.Fatal("expected non-nil TemplateSet")
	}
}

func TestGetStyle_ElementKinds(t *testing.T) {
	ts, err := drawio.LoadTemplate(defaultTemplatePath)
	if err != nil {
		t.Fatalf("LoadTemplate: %v", err)
	}

	kinds := []string{"actor", "system", "container", "component"}
	for _, kind := range kinds {
		style, ok := ts.GetStyle(kind)
		if !ok {
			t.Errorf("GetStyle(%q): expected true, got false", kind)
			continue
		}
		if style.Style == "" {
			t.Errorf("GetStyle(%q): expected non-empty style string", kind)
		}
		if style.Width <= 0 || style.Height <= 0 {
			t.Errorf("GetStyle(%q): expected positive dimensions, got w=%v h=%v", kind, style.Width, style.Height)
		}
	}
}

func TestGetBoundaryStyle_BoundaryKinds(t *testing.T) {
	ts, err := drawio.LoadTemplate(defaultTemplatePath)
	if err != nil {
		t.Fatalf("LoadTemplate: %v", err)
	}

	kinds := []string{"system_boundary", "container_boundary"}
	for _, kind := range kinds {
		style, ok := ts.GetBoundaryStyle(kind)
		if !ok {
			t.Errorf("GetBoundaryStyle(%q): expected true, got false", kind)
			continue
		}
		if style.Style == "" {
			t.Errorf("GetBoundaryStyle(%q): expected non-empty style string", kind)
		}
		if style.Width <= 0 || style.Height <= 0 {
			t.Errorf("GetBoundaryStyle(%q): expected positive dimensions, got w=%v h=%v", kind, style.Width, style.Height)
		}
	}
}

func TestGetConnectorStyle(t *testing.T) {
	ts, err := drawio.LoadTemplate(defaultTemplatePath)
	if err != nil {
		t.Fatalf("LoadTemplate: %v", err)
	}

	connector := ts.GetConnectorStyle()
	if connector == "" {
		t.Error("GetConnectorStyle: expected non-empty connector style")
	}
}

func TestGetStyle_UnknownKind(t *testing.T) {
	ts, err := drawio.LoadTemplate(defaultTemplatePath)
	if err != nil {
		t.Fatalf("LoadTemplate: %v", err)
	}

	_, ok := ts.GetStyle("nonexistent")
	if ok {
		t.Error("GetStyle(nonexistent): expected false, got true")
	}
}

func TestLoadTemplateFromBytes(t *testing.T) {
	data, err := os.ReadFile(defaultTemplatePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	ts, err := drawio.LoadTemplateFromBytes(data)
	if err != nil {
		t.Fatalf("LoadTemplateFromBytes: %v", err)
	}

	style, ok := ts.GetStyle("system")
	if !ok {
		t.Fatal("GetStyle(system): expected true, got false")
	}
	if style.Style == "" {
		t.Error("expected non-empty style string")
	}
}

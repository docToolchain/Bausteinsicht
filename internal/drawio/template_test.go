package drawio_test

import (
	"os"
	"strings"
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

	kinds := []string{"actor", "system", "container", "component", "datastore", "ui", "queue", "mobile", "filestore"}
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

func TestLoadTemplateFromBytes_InvalidXML(t *testing.T) {
	data := []byte("this is not xml")
	_, err := drawio.LoadTemplateFromBytes(data)
	if err == nil {
		t.Fatal("LoadTemplateFromBytes: expected error for non-XML input, got nil")
	}
}

func TestLoadTemplate_InvalidXMLFile(t *testing.T) {
	// Create a temp file with .drawio extension but invalid XML content.
	tmpFile, err := os.CreateTemp(t.TempDir(), "bad-*.drawio")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	if _, err := tmpFile.WriteString("this is not xml"); err != nil {
		t.Fatalf("WriteString: %v", err)
	}
	_ = tmpFile.Close()

	_, err = drawio.LoadTemplate(tmpFile.Name())
	if err == nil {
		t.Fatal("LoadTemplate: expected error for invalid XML template, got nil")
	}
}

func TestLoadTemplateFromBytes_ValidXMLButNotDrawio(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?><html><body>not drawio</body></html>`)
	_, err := drawio.LoadTemplateFromBytes(data)
	if err == nil {
		t.Fatal("LoadTemplateFromBytes: expected error for non-drawio XML, got nil")
	}
}

func TestGetStyle_SubCellStyles(t *testing.T) {
	ts, err := drawio.LoadTemplate(defaultTemplatePath)
	if err != nil {
		t.Fatalf("LoadTemplate: %v", err)
	}

	// All non-actor element kinds should have sub-cell styles.
	for _, kind := range []string{"system", "container", "component", "external_system", "datastore", "ui", "queue", "mobile", "filestore"} {
		style, ok := ts.GetStyle(kind)
		if !ok {
			t.Errorf("GetStyle(%q): expected true", kind)
			continue
		}
		if style.TitleStyle == nil {
			t.Errorf("GetStyle(%q): expected non-nil TitleStyle", kind)
		}
		if style.TechStyle == nil {
			t.Errorf("GetStyle(%q): expected non-nil TechStyle", kind)
		}
		if style.DescStyle == nil {
			t.Errorf("GetStyle(%q): expected non-nil DescStyle", kind)
		}
	}

	// Actor should have title and desc sub-cells (no tech).
	actorStyle, ok := ts.GetStyle("actor")
	if !ok {
		t.Fatal("GetStyle(actor): expected true")
	}
	if actorStyle.TitleStyle == nil {
		t.Error("actor: expected non-nil TitleStyle")
	}
	if actorStyle.TechStyle != nil {
		t.Error("actor: expected nil TechStyle")
	}
	if actorStyle.DescStyle == nil {
		t.Error("actor: expected non-nil DescStyle")
	}
}

func TestLoadTemplateFromBytes_NoVersionAttr(t *testing.T) {
	data := []byte(`<mxfile><diagram id="t" name="T"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/></root></mxGraphModel></diagram></mxfile>`)
	ts, err := drawio.LoadTemplateFromBytes(data)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if ts.Version != 1 {
		t.Errorf("expected version 1 for missing attr, got %d", ts.Version)
	}
}

func TestLoadTemplateFromBytes_Version1(t *testing.T) {
	data := []byte(`<mxfile bausteinsicht_template_version="1"><diagram id="t" name="T"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/></root></mxGraphModel></diagram></mxfile>`)
	ts, err := drawio.LoadTemplateFromBytes(data)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if ts.Version != 1 {
		t.Errorf("expected version 1, got %d", ts.Version)
	}
}

func TestLoadTemplateFromBytes_UnsupportedVersion(t *testing.T) {
	data := []byte(`<mxfile bausteinsicht_template_version="99"><diagram id="t" name="T"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/></root></mxGraphModel></diagram></mxfile>`)
	_, err := drawio.LoadTemplateFromBytes(data)
	if err == nil {
		t.Fatal("expected error for unsupported version, got nil")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("expected 'not supported' in error, got: %v", err)
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

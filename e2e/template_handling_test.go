package e2e

// TestTemplateHandling (#519) closes E2E-Test-Plan.adoc section 14 (6
// tests, previously zero e2e coverage): --template resolution and error
// paths (internal/drawio/template.go LoadTemplate/LoadTemplateFromBytes)
// plus `init`'s own template.drawio creation and collision handling.

import (
	"os"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
)

const emptyTemplateXML = `<mxfile>
  <diagram id="templates" name="Templates">
    <mxGraphModel dx="800" dy="600" grid="1" gridSize="10" page="1" pageWidth="850" pageHeight="1100" background="#ffffff">
      <root>
        <mxCell id="0"/>
        <mxCell id="1" parent="0"/>
      </root>
    </mxGraphModel>
  </diagram>
</mxfile>
`

func TestTemplateHandling(t *testing.T) {
	t.Run("14.1_CustomTemplateViaFlag", test14_1CustomTemplate)
	t.Run("14.2_TemplateWithoutRequiredStyles", test14_2EmptyTemplate)
	t.Run("14.3_TemplateNotValidDrawioXML", test14_3InvalidXMLTemplate)
	t.Run("14.4_TemplatePathDoesNotExist", test14_4NonexistentTemplate)
	t.Run("14.5_InitCreatesTemplateDrawio", test14_5InitCreatesTemplate)
	t.Run("14.6_InitBlockedByExistingTemplate", test14_6InitBlockedByExistingTemplate)
}

// test14_1CustomTemplate covers 14.1: a custom --template's styles are
// actually used for forward-synced elements, not the embedded default.
func test14_1CustomTemplate(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")
	runCLI(t, bin, dir, "generate-template", "--output", "custom.drawio", "--style", "c4")

	// Give the "system" kind's template a distinctive, unmistakable marker
	// fillColor so we can prove it (and not the embedded default) was used.
	const defaultSystemStyle = `style="fillColor=#1168bd;strokeColor=#0b4884;fontColor=#000000;fontSize=12;whiteSpace=wrap;html=1;align=center;verticalAlign=middle;container=0;rounded=1"`
	const markedSystemStyle = `style="fillColor=#abc123;strokeColor=#0b4884;fontColor=#000000;fontSize=12;whiteSpace=wrap;html=1;align=center;verticalAlign=middle;container=0;rounded=1"`
	dataBytes, err := os.ReadFile(dir + "/custom.drawio")
	if err != nil {
		t.Fatalf("read custom.drawio: %v", err)
	}
	data := string(dataBytes)
	if !strings.Contains(data, defaultSystemStyle) {
		t.Fatalf("expected generate-template's known 'system' style in custom.drawio, got:\n%s", data)
	}
	writeFile(t, dir+"/custom.drawio", strings.Replace(data, defaultSystemStyle, markedSystemStyle, 1))

	// Nest under "onlineshop" (already covered by the "containers" view's
	// "onlineshop.*" include) so the element actually renders on a page —
	// a bare top-level add wouldn't appear in any view's resolved set.
	runCLI(t, bin, dir, "add", "element", "--parent", "onlineshop", "--id", "customtemplatetest", "--kind", "system", "--title", "Custom Template Test")
	runCLI(t, bin, dir, "sync", "--template", "custom.drawio")

	doc, err := drawio.LoadDocument(dir + "/architecture.drawio")
	if err != nil {
		t.Fatalf("LoadDocument: %v", err)
	}
	found := false
	for _, page := range doc.Pages() {
		obj := page.FindElement("onlineshop.customtemplatetest")
		if obj == nil {
			continue
		}
		found = true
		cell := obj.SelectElement("mxCell")
		if cell == nil || !strings.Contains(cell.SelectAttrValue("style", ""), "#abc123") {
			t.Errorf("expected custom template's #abc123 fillColor on customtemplatetest, got style: %q", cell.SelectAttrValue("style", ""))
		}
	}
	if !found {
		t.Fatal("onlineshop.customtemplatetest element not found on any page")
	}
}

// test14_2EmptyTemplate covers 14.2: a structurally valid draw.io template
// with zero styled kinds doesn't fail sync — it degrades gracefully with a
// per-kind warning (internal/sync/forward.go:975-978 "no template style for
// kind"), not a hard error.
func test14_2EmptyTemplate(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")
	writeFile(t, dir+"/empty.drawio", emptyTemplateXML)

	// Nest under "onlineshop" so the element is actually visible on a page
	// and forward sync tries to style it (a bare top-level add wouldn't
	// reach the template-lookup code path at all).
	runCLI(t, bin, dir, "add", "element", "--parent", "onlineshop", "--id", "notemplatetest", "--kind", "system", "--title", "No Template Test")
	out, code := runCLIAllowFail(t, bin, dir, "sync", "--template", "empty.drawio")
	if code != 0 {
		t.Fatalf("expected exit 0 (graceful degradation), got %d\n%s", code, out)
	}
	if !strings.Contains(out, "no template style for kind") {
		t.Errorf("expected a 'no template style for kind' warning, got: %s", out)
	}
}

// test14_3InvalidXMLTemplate covers 14.3: a --template file that isn't
// valid XML at all is a clear parse error.
func test14_3InvalidXMLTemplate(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")
	writeFile(t, dir+"/garbage.drawio", "this is not xml at all { } < >")

	out, code := runCLIAllowFail(t, bin, dir, "sync", "--template", "garbage.drawio")
	if code != 2 {
		t.Fatalf("expected exit 2, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "loading template") {
		t.Errorf("expected a 'loading template' parse error, got: %s", out)
	}
}

// test14_4NonexistentTemplate covers 14.4: a --template path that doesn't
// exist is a clear file-not-found error.
func test14_4NonexistentTemplate(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	out, code := runCLIAllowFail(t, bin, dir, "sync", "--template", "missing.drawio")
	if code != 2 {
		t.Fatalf("expected exit 2, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "loading template") {
		t.Errorf("expected a 'loading template' not-found error, got: %s", out)
	}
}

// test14_5InitCreatesTemplate covers 14.5: `init` creates template.drawio
// with valid draw.io XML containing per-kind element styles.
func test14_5InitCreatesTemplate(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	dataBytes, err := os.ReadFile(dir + "/template.drawio")
	if err != nil {
		t.Fatalf("read template.drawio: %v", err)
	}
	data := string(dataBytes)
	if !strings.Contains(data, "bausteinsicht_template") {
		t.Error("expected template.drawio to contain bausteinsicht_template-tagged styles")
	}
	if _, err := drawio.LoadDocument(dir + "/template.drawio"); err != nil {
		t.Errorf("template.drawio is not well-formed draw.io XML: %v", err)
	}
}

// test14_6InitBlockedByExistingTemplate covers 14.6: `init` refuses to
// overwrite a pre-existing template.drawio.
func test14_6InitBlockedByExistingTemplate(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir+"/template.drawio", "pre-existing content")

	out, code := runCLIAllowFail(t, bin, dir, "init")
	if code != 2 {
		t.Fatalf("expected exit 2, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "already exists") {
		t.Errorf("expected 'already exists' error, got: %s", out)
	}
}

package e2e

// TestReverseSyncLabelHandling (#519) closes E2E-Test-Plan.adoc section 4
// tests 4.4-4.7: how reverse sync handles HTML entities, tags, line breaks,
// and empty strings in a draw.io element's title sub-cell "value" attribute.
//
// internal/drawio.Page.ReadElementFields (element.go:382) reads the "-title"
// sub-cell's "value" attribute verbatim via etree.SelectAttrValue — XML
// entities are decoded by etree's parser (correct XML round-trip), but no
// HTML-tag stripping or extra escaping happens on this code path (that only
// exists in the legacy ParseLabel fallback used for objects without
// sub-cells, internal/drawio/label.go:159 stripTags). These tests pin that
// actual, current behavior so a future change (e.g. adding tag stripping to
// the sub-cell path) is a deliberate, visible diff rather than an untested
// surprise.

import (
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// findTitleSubCellAndSetValue locates the "customer" element's "-title"
// sub-cell in the given page and sets its "value" attribute.
func findTitleSubCellAndSetValue(t *testing.T, page *drawio.Page, value string) {
	t.Helper()
	obj := page.FindElement("customer")
	if obj == nil {
		t.Fatal("'customer' not found in draw.io")
	}
	cellID := obj.SelectAttrValue("id", "")
	for _, cell := range page.Root().SelectElements("mxCell") {
		if cell.SelectAttrValue("parent", "") == cellID &&
			strings.HasSuffix(cell.SelectAttrValue("id", ""), "-title") {
			setCellAttr(cell, "value", value)
			return
		}
	}
	t.Fatal("customer element has no title sub-cell")
}

// setupCustomerTitleEdit inits a project, loads the draw.io doc, and returns
// the doc, the page containing "customer", and both file paths, ready for
// the caller to mutate the title sub-cell and save.
func setupCustomerTitleEdit(t *testing.T, bin, dir string) (doc *drawio.Document, page *drawio.Page, modelPath, drawioPath string) {
	t.Helper()
	runCLI(t, bin, dir, "init")
	modelPath = dir + "/architecture.jsonc"
	drawioPath = dir + "/architecture.drawio"

	doc, err := drawio.LoadDocument(drawioPath)
	if err != nil {
		t.Fatalf("LoadDocument: %v", err)
	}
	for _, p := range doc.Pages() {
		if p.FindElement("customer") != nil {
			page = p
			break
		}
	}
	if page == nil {
		t.Fatal("'customer' not found in draw.io after init")
	}
	return doc, page, modelPath, drawioPath
}

func TestReverseSyncLabelHandling(t *testing.T) {
	t.Run("HTMLEntitiesDecoded", testLabelHTMLEntities)
	t.Run("HTMLTagsPreservedLiterally", testLabelHTMLTags)
	t.Run("LineBreaksPreservedLiterally", testLabelLineBreaks)
	t.Run("EmptyLabelRejected", testLabelEmpty)
}

// testLabelHTMLEntities covers 4.4: characters that require XML entity
// escaping when written (&, <, >) must decode back to their literal form.
func testLabelHTMLEntities(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	doc, page, modelPath, drawioPath := setupCustomerTitleEdit(t, bin, dir)

	const want = `Sales & Support <Team>`
	findTitleSubCellAndSetValue(t, page, want)
	if err := drawio.SaveDocument(drawioPath, doc); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	out, code := runCLIAllowFail(t, bin, dir, "sync")
	if code != 0 {
		t.Fatalf("sync failed: exit %d\n%s", code, out)
	}

	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatalf("model.Load: %v", err)
	}
	if got := m.Model["customer"].Title; got != want {
		t.Errorf("customer.Title = %q, want %q (entities not properly decoded)", got, want)
	}
}

// testLabelHTMLTags covers 4.5: the sub-cell path (the default, current
// element format) does not strip HTML tags — they are stored as literal
// text, same as any other title content. Tag-stripping only exists in the
// legacy ParseLabel fallback for objects without sub-cells.
func testLabelHTMLTags(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	doc, page, modelPath, drawioPath := setupCustomerTitleEdit(t, bin, dir)

	const want = `<b>Bold Customer</b>`
	findTitleSubCellAndSetValue(t, page, want)
	if err := drawio.SaveDocument(drawioPath, doc); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	out, code := runCLIAllowFail(t, bin, dir, "sync")
	if code != 0 {
		t.Fatalf("sync failed: exit %d\n%s", code, out)
	}

	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatalf("model.Load: %v", err)
	}
	if got := m.Model["customer"].Title; got != want {
		t.Errorf("customer.Title = %q, want %q (literal, untouched by tag-stripping)", got, want)
	}
}

// testLabelLineBreaks covers 4.6: <br> in a title is handled gracefully
// (stored as literal text, no crash, no truncation).
func testLabelLineBreaks(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	doc, page, modelPath, drawioPath := setupCustomerTitleEdit(t, bin, dir)

	const want = `Line One<br>Line Two`
	findTitleSubCellAndSetValue(t, page, want)
	if err := drawio.SaveDocument(drawioPath, doc); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	out, code := runCLIAllowFail(t, bin, dir, "sync")
	if code != 0 {
		t.Fatalf("sync failed: exit %d\n%s", code, out)
	}

	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatalf("model.Load: %v", err)
	}
	if got := m.Model["customer"].Title; got != want {
		t.Errorf("customer.Title = %q, want %q", got, want)
	}
}

// testLabelEmpty covers 4.7: an empty title from draw.io is explicitly
// rejected (not silently accepted) — a warning is emitted and the model's
// existing title is preserved (internal/sync/reverse.go:122-127, #150).
func testLabelEmpty(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	doc, page, modelPath, drawioPath := setupCustomerTitleEdit(t, bin, dir)

	m0, err := model.Load(modelPath)
	if err != nil {
		t.Fatalf("model.Load before edit: %v", err)
	}
	originalTitle := m0.Model["customer"].Title

	findTitleSubCellAndSetValue(t, page, "")
	if err := drawio.SaveDocument(drawioPath, doc); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	out, code := runCLIAllowFail(t, bin, dir, "sync")
	if code != 0 {
		t.Fatalf("sync with empty title should not fail (warning only): exit %d\n%s", code, out)
	}
	if !strings.Contains(strings.ToLower(out), "ignoring empty title") {
		t.Errorf("expected 'ignoring empty title' warning in output, got: %s", out)
	}

	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatalf("model.Load after sync: %v", err)
	}
	if got := m.Model["customer"].Title; got != originalTitle {
		t.Errorf("customer.Title = %q after empty-label sync, want unchanged %q", got, originalTitle)
	}
}

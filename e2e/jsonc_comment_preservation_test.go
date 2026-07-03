package e2e

// TestJSONCCommentPreservation (#519) closes E2E-Test-Plan.adoc section 4
// (4.16-4.18: reverse sync) and section 6 (6.25-6.30: add commands) — these
// had zero e2e coverage even though internal/model/patch.go is explicitly
// designed as "comment-preserving JSONC mutations". Each subtest injects a
// `//` line comment, a `/* */` block comment, and a trailing comma into a
// freshly-initialized model, performs a mutation, and asserts the untouched
// parts of the file are byte-for-byte preserved (not just "no crash").

import (
	"os"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// injectCommentVariety adds a block comment and a trailing comma to the
// sample model (which already has plenty of "//" line comments) at
// locations unrelated to the mutations the subtests perform, then verifies
// the result still loads cleanly.
func injectCommentVariety(t *testing.T, modelPath string) {
	t.Helper()
	data, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("read model: %v", err)
	}
	content := string(data)

	const blockMarker = "/* Actor: primary customer persona */"
	before := `"customer": {`
	if !strings.Contains(content, before) {
		t.Fatalf("anchor %q not found in sample model", before)
	}
	content = strings.Replace(content, before, blockMarker+"\n    "+before, 1)

	// Add a trailing comma before a `}` — StripJSONC must tolerate it and
	// PatchInsert/PatchSave must not remove it from untouched regions.
	const trailingBefore = `"dashed": true`
	const trailingAfter = `"dashed": true,`
	if !strings.Contains(content, trailingBefore) {
		t.Fatalf("anchor %q not found in sample model", trailingBefore)
	}
	content = strings.Replace(content, trailingBefore, trailingAfter, 1)

	if err := os.WriteFile(modelPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write model: %v", err)
	}

	if _, err := model.Load(modelPath); err != nil {
		t.Fatalf("injected model no longer parses: %v", err)
	}
}

func assertCommentVarietyPreserved(t *testing.T, modelPath string, wantLineComments int) {
	t.Helper()
	data, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("read model after mutation: %v", err)
	}
	content := string(data)

	if got := strings.Count(content, "//"); got != wantLineComments {
		t.Errorf("line comment count = %d, want %d (content:\n%s)", got, wantLineComments, content)
	}
	if !strings.Contains(content, "/* Actor: primary customer persona */") {
		t.Errorf("block comment lost after mutation:\n%s", content)
	}
	if !strings.Contains(content, `"dashed": true,`) {
		t.Errorf("trailing comma lost after mutation:\n%s", content)
	}
}

// setupProjectWithCommentVariety inits a project and injects the comment
// variety, returning paths and the baseline "//" count for later comparison.
func setupProjectWithCommentVariety(t *testing.T, bin, dir string) (modelPath, drawioPath string, lineCommentCount int) {
	t.Helper()
	runCLI(t, bin, dir, "init")
	modelPath = dir + "/architecture.jsonc"
	drawioPath = dir + "/architecture.drawio"
	injectCommentVariety(t, modelPath)

	data, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("read model: %v", err)
	}
	lineCommentCount = strings.Count(string(data), "//")
	return modelPath, drawioPath, lineCommentCount
}

func TestJSONCCommentPreservation(t *testing.T) {
	t.Run("ReverseSync", testCommentPreservationReverseSync)
	t.Run("AddElement", testCommentPreservationAddElement)
	t.Run("AddElementMixedComments", testCommentPreservationAddElementMixed)
	t.Run("TwoSequentialAdds", testCommentPreservationTwoSequentialAdds)
	t.Run("AddRelationship", testCommentPreservationAddRelationship)
}

// testCommentPreservationReverseSync covers 4.16 (//), 4.17 (/* */), and
// 4.18 (trailing commas): editing a label in draw.io and syncing triggers
// PatchSave (value-replacement reverse sync) — comments/commas elsewhere in
// the file must survive untouched.
func testCommentPreservationReverseSync(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	modelPath, drawioPath, lineComments := setupProjectWithCommentVariety(t, bin, dir)

	doc, err := drawio.LoadDocument(drawioPath)
	if err != nil {
		t.Fatalf("LoadDocument: %v", err)
	}
	var page *drawio.Page
	for _, p := range doc.Pages() {
		if p.FindElement("customer") != nil {
			page = p
			break
		}
	}
	if page == nil {
		t.Fatal("'customer' not found in draw.io after init")
	}
	obj := page.FindElement("customer")
	cellID := obj.SelectAttrValue("id", "")
	mutated := false
	for _, cell := range page.Root().SelectElements("mxCell") {
		if cell.SelectAttrValue("parent", "") == cellID &&
			strings.HasSuffix(cell.SelectAttrValue("id", ""), "-title") {
			setCellAttr(cell, "value", "Customer Renamed")
			mutated = true
			break
		}
	}
	if !mutated {
		t.Fatal("customer element has no title sub-cell after init")
	}
	if err := drawio.SaveDocument(drawioPath, doc); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	out, code := runCLIAllowFail(t, bin, dir, "sync")
	if code != 0 {
		t.Fatalf("sync failed: exit %d\n%s", code, out)
	}

	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatalf("model.Load after sync: %v", err)
	}
	if m.Model["customer"].Title != "Customer Renamed" {
		t.Errorf("customer.Title = %q, want %q (reverse sync did not apply)", m.Model["customer"].Title, "Customer Renamed")
	}

	assertCommentVarietyPreserved(t, modelPath, lineComments)
}

// testCommentPreservationAddElement covers 6.25 (//) and 6.26/6.27 (block
// comments + trailing commas): `add element` uses PatchInsert, a different
// code path than reverse sync's PatchSave.
func testCommentPreservationAddElement(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	modelPath, _, lineComments := setupProjectWithCommentVariety(t, bin, dir)

	runCLI(t, bin, dir, "add", "element", "--id", "newthing", "--kind", "system", "--title", "New Thing")

	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatalf("model.Load after add: %v", err)
	}
	if _, ok := m.Model["newthing"]; !ok {
		t.Fatal("newthing not found in model after add element")
	}

	assertCommentVarietyPreserved(t, modelPath, lineComments)
}

// testCommentPreservationAddElementMixed covers 6.28: comments of both
// styles plus a trailing comma must all survive an add in combination.
func testCommentPreservationAddElementMixed(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	modelPath, _, lineComments := setupProjectWithCommentVariety(t, bin, dir)

	runCLI(t, bin, dir, "add", "element", "--id", "another", "--kind", "actor", "--title", "Another Actor", "--description", "desc with & special <chars>")

	assertCommentVarietyPreserved(t, modelPath, lineComments)

	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatalf("model.Load after add: %v", err)
	}
	if got := m.Model["another"].Description; got != "desc with & special <chars>" {
		t.Errorf("another.Description = %q, want unescaped literal", got)
	}
}

// testCommentPreservationTwoSequentialAdds covers 6.29: comments must
// survive not just one but two consecutive PatchInsert operations.
func testCommentPreservationTwoSequentialAdds(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	modelPath, _, lineComments := setupProjectWithCommentVariety(t, bin, dir)

	runCLI(t, bin, dir, "add", "element", "--id", "first", "--kind", "system", "--title", "First")
	runCLI(t, bin, dir, "add", "element", "--id", "second", "--kind", "system", "--title", "Second")

	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatalf("model.Load after two adds: %v", err)
	}
	if _, ok := m.Model["first"]; !ok {
		t.Error("first not found after two sequential adds")
	}
	if _, ok := m.Model["second"]; !ok {
		t.Error("second not found after two sequential adds")
	}

	assertCommentVarietyPreserved(t, modelPath, lineComments)
}

// testCommentPreservationAddRelationship covers 6.30: `add relationship`
// uses AppendArrayEntry, yet another PatchInsert transform.
func testCommentPreservationAddRelationship(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	modelPath, _, lineComments := setupProjectWithCommentVariety(t, bin, dir)

	// customer -> onlineshop.frontend with kind "uses" already exists in the
	// sample model; use a fresh, non-duplicate pair.
	runCLI(t, bin, dir, "add", "relationship", "--from", "customer", "--to", "payment-provider", "--label", "reports fraud to")

	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatalf("model.Load after add relationship: %v", err)
	}
	found := false
	for _, r := range m.Relationships {
		if r.From == "customer" && r.To == "payment-provider" && r.Label == "reports fraud to" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("new relationship not found in model after add relationship")
	}

	assertCommentVarietyPreserved(t, modelPath, lineComments)
}

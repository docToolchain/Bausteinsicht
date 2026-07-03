package e2e

// TestSecurityInjection (#519) closes E2E-Test-Plan.adoc section 10 (9
// tests, previously zero e2e coverage). Bausteinsicht writes user-controlled
// strings (element title/description/technology, relationship labels) into
// XML (draw.io) and JSONC (the model file) — these tests confirm that path
// is actually safe at the black-box level: etree's XML writer escapes
// correctly (internal/drawio/*.go never string-concatenates XML), the ID
// regex rejects shell-metacharacter IDs, and --parent is a model-internal
// dot-path lookup that never touches the filesystem (so path-traversal
// syntax just fails to resolve, no directory escape is possible).

import (
	"os"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// assertNoCrash fails the test if the CLI output looks like an unhandled Go
// panic rather than a controlled error/success path.
func assertNoCrash(t *testing.T, out string) {
	t.Helper()
	if strings.Contains(out, "panic:") || strings.Contains(out, "goroutine ") {
		t.Errorf("CLI output looks like an unhandled panic, not graceful handling:\n%s", out)
	}
}

// assertDrawioWellFormed reloads the draw.io file to confirm it's still
// parseable XML after a potentially-malicious sync.
func assertDrawioWellFormed(t *testing.T, drawioPath string) *drawio.Document {
	t.Helper()
	doc, err := drawio.LoadDocument(drawioPath)
	if err != nil {
		t.Fatalf("draw.io file is not well-formed XML after sync: %v", err)
	}
	return doc
}

func TestSecurityInjection(t *testing.T) {
	t.Run("10.1_XMLInjectionViaTitle", testSecXMLInjectionTitle)
	t.Run("10.2_XMLInjectionViaDescription", testSecXMLInjectionDescription)
	t.Run("10.3_XMLInjectionViaRelationshipLabel", testSecXMLInjectionRelLabel)
	t.Run("10.4_CDATAInjection", testSecCDATAInjection)
	t.Run("10.5_UnicodeNullByteInTitle", testSecNullByteTitle)
	t.Run("10.6_VeryLongString", testSecVeryLongString)
	t.Run("10.7_PathTraversalViaParent", testSecPathTraversalParent)
	t.Run("10.8_ShellInjectionViaElementID", testSecShellInjectionID)
	t.Run("10.9_JSONInjectionInTitle", testSecJSONInjectionTitle)
}

// testSecXMLInjectionTitle covers 10.1: a title containing a raw <script>
// tag must be escaped on write and must not break XML structure.
func testSecXMLInjectionTitle(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	const payload = `<script>alert('xss')</script>`
	// Nest under "onlineshop" (already covered by the "containers" view's
	// "onlineshop.*" include) so the element actually renders on a page —
	// a bare top-level add wouldn't appear in any view's resolved set.
	runCLI(t, bin, dir, "add", "element", "--parent", "onlineshop", "--id", "xsstest", "--kind", "ui", "--title", payload)
	runCLI(t, bin, dir, "sync")

	doc := assertDrawioWellFormed(t, dir+"/architecture.drawio")

	raw, err := os.ReadFile(dir + "/architecture.drawio")
	if err != nil {
		t.Fatalf("read draw.io: %v", err)
	}
	if strings.Contains(string(raw), "<script>") {
		t.Error("raw <script> tag found unescaped in draw.io XML")
	}

	found := false
	for _, p := range doc.Pages() {
		if p.FindElement("onlineshop.xsstest") != nil {
			found = true
		}
	}
	if !found {
		t.Error("onlineshop.xsstest element not found on any page after sync")
	}
}

// testSecXMLInjectionDescription covers 10.2: an attribute-breakout payload
// in description must not inject a new XML attribute.
func testSecXMLInjectionDescription(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	const payload = `"><evil attr="injected`
	runCLI(t, bin, dir, "add", "element", "--id", "attrtest", "--kind", "system", "--title", "Attr Test", "--description", payload)
	runCLI(t, bin, dir, "sync")

	assertDrawioWellFormed(t, dir+"/architecture.drawio")

	raw, err := os.ReadFile(dir + "/architecture.drawio")
	if err != nil {
		t.Fatalf("read draw.io: %v", err)
	}
	if strings.Contains(string(raw), `<evil `) {
		t.Error("injected <evil> element found unescaped in draw.io XML")
	}
}

// testSecXMLInjectionRelLabel covers 10.3: a relationship label attempting
// to close the current element and open a new mxCell must not succeed.
func testSecXMLInjectionRelLabel(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	const payload = `</mxCell><mxCell evil="true"/>`
	runCLI(t, bin, dir, "add", "relationship", "--from", "customer", "--to", "payment-provider", "--label", payload)
	runCLI(t, bin, dir, "sync")

	assertDrawioWellFormed(t, dir+"/architecture.drawio")

	raw, err := os.ReadFile(dir + "/architecture.drawio")
	if err != nil {
		t.Fatalf("read draw.io: %v", err)
	}
	if strings.Contains(string(raw), `evil="true"`) {
		t.Error("injected mxCell with evil=\"true\" found unescaped in draw.io XML")
	}
}

// testSecCDATAInjection covers 10.4: "]]>" (CDATA-section terminator) plus
// a tag in a description must not corrupt XML integrity, even though this
// codebase doesn't use CDATA sections for these fields.
func testSecCDATAInjection(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	const payload = `]]><evil>injected</evil>`
	runCLI(t, bin, dir, "add", "element", "--id", "cdatatest", "--kind", "system", "--title", "CDATA Test", "--description", payload)
	runCLI(t, bin, dir, "sync")

	assertDrawioWellFormed(t, dir+"/architecture.drawio")

	raw, err := os.ReadFile(dir + "/architecture.drawio")
	if err != nil {
		t.Fatalf("read draw.io: %v", err)
	}
	if strings.Contains(string(raw), `<evil>`) {
		t.Error("injected <evil> element found unescaped in draw.io XML")
	}
}

// testSecNullByteTitle covers 10.5: a title containing a literal NUL byte
// (only reachable by editing the model file directly — no shell/exec can
// pass a NUL byte through argv) must not crash the CLI.
func testSecNullByteTitle(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	modelPath := dir + "/architecture.jsonc"
	data, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("read model: %v", err)
	}
	// `\u0000` is the JSON escape sequence for a literal NUL byte; using
	// the escape text (not an actual NUL byte in this Go source file)
	// mirrors exactly what a hand-edited JSONC file would contain.
	after := `"title": "Bad\u0000Title"`
	patched := strings.Replace(string(data), `"title": "Customer"`, after, 1)
	if patched == string(data) {
		t.Fatal("anchor for customer title not found in sample model")
	}
	if err := os.WriteFile(modelPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write model: %v", err)
	}

	out, _ := runCLIAllowFail(t, bin, dir, "sync")
	assertNoCrash(t, out)
}

// testSecVeryLongString covers 10.6: a 100,000-character title must not
// crash the CLI or corrupt the resulting draw.io XML. The string is written
// directly into the model file (like testSecNullByteTitle) rather than
// passed as a CLI arg — Windows' CreateProcess has a ~32K command-line
// length limit, so a 100K-char --title flag fails at the OS/exec layer
// before the program even starts, on that platform only. Going through the
// model file avoids that platform-specific ceiling entirely and is a more
// accurate test of the tool's own handling of long strings.
func testSecVeryLongString(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	modelPath := dir + "/architecture.jsonc"
	data, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("read model: %v", err)
	}
	longTitle := strings.Repeat("A", 100_000)
	patched := strings.Replace(string(data), `"title": "Customer"`, `"title": "`+longTitle+`"`, 1)
	if patched == string(data) {
		t.Fatal("anchor for customer title not found in sample model")
	}
	if err := os.WriteFile(modelPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write model: %v", err)
	}

	out, code := runCLIAllowFail(t, bin, dir, "sync")
	assertNoCrash(t, out)
	if code != 0 {
		t.Fatalf("sync with 100K-char title failed: exit %d\n%s", code, out)
	}

	assertDrawioWellFormed(t, dir+"/architecture.drawio")
}

// testSecPathTraversalParent covers 10.7: --parent is a model-internal
// dot-path lookup, not a filesystem path — path-traversal syntax must
// resolve to a plain "not found" error, not touch the filesystem.
func testSecPathTraversalParent(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	out, code := runCLIAllowFail(t, bin, dir, "add", "element", "--id", "traversaltest", "--kind", "system", "--title", "T", "--parent", "../../etc/passwd")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "not found") {
		t.Errorf("expected 'not found' error, got: %s", out)
	}
	if _, err := os.Stat("/etc/passwd.bak"); err == nil {
		t.Error("unexpected filesystem side effect from path-traversal --parent")
	}
}

// testSecShellInjectionID covers 10.8: element IDs containing shell
// metacharacters are rejected by the ID regex before any further processing.
func testSecShellInjectionID(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	for _, id := range []string{"$(whoami)", "`whoami`", "; rm -rf /"} {
		out, code := runCLIAllowFail(t, bin, dir, "add", "element", "--id", id, "--kind", "system", "--title", "T")
		if code != 1 {
			t.Errorf("id %q: expected exit 1, got %d\n%s", id, code, out)
			continue
		}
		if !strings.Contains(out, "invalid element ID") {
			t.Errorf("id %q: expected 'invalid element ID' error, got: %s", id, out)
		}
	}
}

// testSecJSONInjectionTitle covers 10.9: a title containing raw JSON syntax
// must be stored as a literal string value, not interpreted as new JSON keys.
func testSecJSONInjectionTitle(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	const payload = `", "evil": "true`
	runCLI(t, bin, dir, "add", "element", "--id", "jsoninjecttest", "--kind", "system", "--title", payload)

	m, err := model.Load(dir + "/architecture.jsonc")
	if err != nil {
		t.Fatalf("model.Load after add: %v", err)
	}
	if got := m.Model["jsoninjecttest"].Title; got != payload {
		t.Errorf("Title = %q, want literal %q", got, payload)
	}

	raw, err := os.ReadFile(dir + "/architecture.jsonc")
	if err != nil {
		t.Fatalf("read model: %v", err)
	}
	// The literal, unescaped sequence `"evil": "true` (as a sibling JSON key)
	// must not appear — it must only be reachable inside the escaped title
	// string (`\", \"evil\": \"true`).
	if strings.Contains(string(raw), "\n") && strings.Contains(string(raw), `      "evil": "true`) {
		t.Error("payload broke out into a real sibling JSON key")
	}
}

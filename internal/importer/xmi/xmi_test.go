package xmi_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/importer/xmi"
)

// ─── TS-XMI-01: Basic element import ─────────────────────────────────────────

func TestImport_BasicElements(t *testing.T) {
	r, err := xmi.Import(td("basic.xmi"), nil)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	m := r.Model

	for _, kind := range []string{"component", "actor"} {
		if _, ok := m.Specification.Elements[kind]; !ok {
			t.Errorf("specification missing kind %q", kind)
		}
	}

	if e, ok := m.Model["api"]; !ok {
		t.Error("expected element 'api'")
	} else if e.Kind != "component" {
		t.Errorf("api.Kind = %q, want 'component'", e.Kind)
	}
	if e, ok := m.Model["customer"]; !ok {
		t.Error("expected element 'customer'")
	} else if e.Kind != "actor" {
		t.Errorf("customer.Kind = %q, want 'actor'", e.Kind)
	}

	if len(m.Relationships) != 1 {
		t.Fatalf("relationships count = %d, want 1", len(m.Relationships))
	}
	rel := m.Relationships[0]
	if rel.From != "customer" || rel.To != "api" {
		t.Errorf("relationship = {%s → %s}, want {customer → api}", rel.From, rel.To)
	}
	if rel.Label != "uses" {
		t.Errorf("relationship.Label = %q, want 'uses'", rel.Label)
	}
}

// ─── TS-XMI-02: Package hierarchy → dot-path children ────────────────────────

func TestImport_Hierarchy(t *testing.T) {
	r, err := xmi.Import(td("hierarchy.xmi"), nil)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	m := r.Model

	sys, ok := m.Model["system"]
	if !ok {
		t.Fatal("expected element 'system'")
	}
	if sys.Kind != "package" {
		t.Errorf("system.Kind = %q, want 'package'", sys.Kind)
	}

	backend, ok := sys.Children["backend"]
	if !ok {
		t.Fatal("expected child 'backend' in 'system'")
	}
	if backend.Kind != "package" {
		t.Errorf("backend.Kind = %q, want 'package'", backend.Kind)
	}

	if _, ok := backend.Children["api"]; !ok {
		t.Fatal("expected child 'api' in 'system.backend'")
	}

	// Package kind must be auto-detected as container because it has children
	if spec, ok := m.Specification.Elements["package"]; !ok {
		t.Error("specification missing kind 'package'")
	} else if !spec.Container {
		t.Error("package kind should have container=true (auto-set from XMI hierarchy)")
	}

	if len(m.Relationships) != 1 {
		t.Fatalf("relationships count = %d, want 1", len(m.Relationships))
	}
	rel := m.Relationships[0]
	if rel.From != "system.backend.api" {
		t.Errorf("rel.From = %q, want 'system.backend.api'", rel.From)
	}
	if rel.To != "system.backend.database" {
		t.Errorf("rel.To = %q, want 'system.backend.database'", rel.To)
	}
}

// ─── TS-XMI-04: --kind-map override ─────────────────────────────────────────

func TestImport_KindMapOverride(t *testing.T) {
	km, err := xmi.ParseKindMap("Component=service,Class=entity")
	if err != nil {
		t.Fatalf("ParseKindMap: %v", err)
	}
	r, err := xmi.Import(td("kindmap.xmi"), km)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	m := r.Model

	if e, ok := m.Model["payment-service"]; !ok {
		t.Error("expected element 'payment-service'")
	} else if e.Kind != "service" {
		t.Errorf("payment-service.Kind = %q, want 'service'", e.Kind)
	}
	if e, ok := m.Model["order-entity"]; !ok {
		t.Error("expected element 'order-entity'")
	} else if e.Kind != "entity" {
		t.Errorf("order-entity.Kind = %q, want 'entity'", e.Kind)
	}
	if _, ok := m.Specification.Elements["component"]; ok {
		t.Error("specification should not contain 'component' when mapped to 'service'")
	}
	if _, ok := m.Specification.Elements["service"]; !ok {
		t.Error("specification missing 'service'")
	}
}

// ─── TS-XMI-05: Stereotype as kind ───────────────────────────────────────────

func TestImport_StereotypeKind(t *testing.T) {
	r, err := xmi.Import(td("stereotypes.xmi"), nil)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	m := r.Model

	orders, ok := m.Model["orders"]
	if !ok {
		t.Fatal("expected element 'orders'")
	}
	if orders.Kind != "microservice" {
		t.Errorf("orders.Kind = %q, want 'microservice'", orders.Kind)
	}
	for _, w := range r.Warnings {
		if strings.Contains(strings.ToLower(w), "orders") {
			t.Errorf("unexpected warning for 'orders': %s", w)
		}
	}

	payments, ok := m.Model["payments"]
	if !ok {
		t.Fatal("expected element 'payments'")
	}
	if payments.Kind != "component" {
		t.Errorf("payments.Kind = %q, want 'component'", payments.Kind)
	}
}

// ─── TS-XMI-06: Unknown type fallback ────────────────────────────────────────

func TestImport_UnknownTypeFallback(t *testing.T) {
	r, err := xmi.Import(td("unknown_type.xmi"), nil)
	if err != nil {
		t.Fatalf("Import should succeed: %v", err)
	}
	if trigger, ok := r.Model.Model["trigger"]; !ok {
		t.Error("expected element 'trigger'")
	} else if trigger.Kind != "element" {
		t.Errorf("trigger.Kind = %q, want 'element'", trigger.Kind)
	}

	hasWarn := false
	for _, w := range r.Warnings {
		if strings.Contains(w, "uml:Signal") {
			hasWarn = true
		}
	}
	if !hasWarn {
		t.Errorf("expected warning about uml:Signal; got: %v", r.Warnings)
	}
}

// ─── TS-XMI-07: Unresolvable relationship skipped ────────────────────────────

func TestImport_UnresolvableRelationship(t *testing.T) {
	r, err := xmi.Import(td("unresolvable_rel.xmi"), nil)
	if err != nil {
		t.Fatalf("Import should succeed: %v", err)
	}
	if len(r.Model.Relationships) != 0 {
		t.Errorf("expected 0 relationships, got %d", len(r.Model.Relationships))
	}
	hasWarn := false
	for _, w := range r.Warnings {
		if strings.Contains(w, "unknown-id") {
			hasWarn = true
		}
	}
	if !hasWarn {
		t.Errorf("expected warning about 'unknown-id'; got: %v", r.Warnings)
	}
}

// ─── TS-XMI-08: Malformed XML rejected ───────────────────────────────────────

func TestImport_InvalidXML(t *testing.T) {
	_, err := xmi.Import(td("invalid.xmi"), nil)
	if err == nil {
		t.Fatal("expected error for invalid XML, got nil")
	}
	if !strings.Contains(err.Error(), "invalid XML") && !strings.Contains(err.Error(), "not a valid XMI") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestImport_NotXMI(t *testing.T) {
	_, err := xmi.Import(td("not_xmi.xml"), nil)
	if err == nil {
		t.Fatal("expected error for non-XMI XML, got nil")
	}
	if !strings.Contains(err.Error(), "not a valid XMI") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// ─── TS-XMI-09: ID sanitization and collision ────────────────────────────────

func TestImport_IDSanitization(t *testing.T) {
	r, err := xmi.Import(td("sanitize.xmi"), nil)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if _, ok := r.Model.Model["payment-service-v2"]; !ok {
		t.Error("expected element 'payment-service-v2'")
	}
}

func TestImport_IDCollision(t *testing.T) {
	r, err := xmi.Import(td("id_collision.xmi"), nil)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	m := r.Model
	if _, ok := m.Model["api"]; !ok {
		t.Error("expected element 'api'")
	}
	if _, ok := m.Model["api-2"]; !ok {
		t.Error("expected element 'api-2' for collision")
	}
	hasWarn := false
	for _, w := range r.Warnings {
		if strings.Contains(w, "collision") || strings.Contains(w, "api-2") {
			hasWarn = true
		}
	}
	if !hasWarn {
		t.Errorf("expected collision warning; got: %v", r.Warnings)
	}
}

func TestImport_IDCollision_Triple(t *testing.T) {
	// Three elements with the same name produce api, api-2, api-3.
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xmi:XMI xmi:version="2.1" xmlns:xmi="http://www.omg.org/spec/XMI/20131001" xmlns:uml="http://www.omg.org/spec/UML/20131001">
  <uml:Model xmi:type="uml:Model" name="M" xmi:id="m1">
    <packagedElement xmi:type="uml:Component" name="API" xmi:id="e1"/>
    <packagedElement xmi:type="uml:Component" name="Api" xmi:id="e2"/>
    <packagedElement xmi:type="uml:Component" name="api" xmi:id="e3"/>
  </uml:Model>
</xmi:XMI>`)

	path := filepath.Join(t.TempDir(), "triple.xmi")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	r, err := xmi.Import(path, nil)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	m := r.Model
	for _, want := range []string{"api", "api-2", "api-3"} {
		if _, ok := m.Model[want]; !ok {
			keys := make([]string, 0, len(m.Model))
			for k := range m.Model {
				keys = append(keys, k)
			}
			t.Errorf("expected element %q; model keys: %v", want, keys)
		}
	}
	collisionWarns := 0
	for _, w := range r.Warnings {
		if strings.Contains(w, "collision") {
			collisionWarns++
		}
	}
	if collisionWarns < 2 {
		t.Errorf("expected ≥2 collision warnings for 3 duplicates, got %d: %v", collisionWarns, r.Warnings)
	}
}

// ─── TS-XMI-10: Specification completeness ───────────────────────────────────

func TestImport_SpecificationCompleteness(t *testing.T) {
	r, err := xmi.Import(td("basic.xmi"), nil)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	spec := r.Model.Specification.Elements
	for _, want := range []string{"component", "actor"} {
		if _, ok := spec[want]; !ok {
			t.Errorf("specification missing kind %q", want)
		}
	}
	if _, ok := spec["package"]; ok {
		t.Error("specification should not contain 'package' for basic.xmi")
	}
}

// ─── XXE protection ───────────────────────────────────────────────────────────

func TestImport_XXEDoctype(t *testing.T) {
	xxe := []byte(`<?xml version="1.0"?>
<!DOCTYPE foo [<!ENTITY xxe SYSTEM "file:///etc/passwd">]>
<xmi:XMI xmlns:xmi="http://www.omg.org/spec/XMI/20131001">
</xmi:XMI>`)
	path := filepath.Join(t.TempDir(), "xxe.xmi")
	if err := os.WriteFile(path, xxe, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := xmi.Import(path, nil)
	if err == nil {
		t.Fatal("expected error for DOCTYPE-containing XMI, got nil")
	}
	if !strings.Contains(err.Error(), "DOCTYPE") {
		t.Errorf("expected DOCTYPE error, got: %v", err)
	}
}

// ─── Integration: large real-world EA export ──────────────────────────────────

// TestImport_BigData runs against a real Enterprise Architect XMI export
// (AUTOSAR model, windows-1252 encoding, ~114 MB, depth >20).
// The fixture is gitignored (not committed, not Git-LFS-tracked, see #553) —
// fetch it with `make fetch-testdata` (scripts/fetch-xmi-testdata.sh), which
// pulls it from the separate docToolchain/bausteinsicht-testdata repo. This
// test skips itself when the fixture is absent or too small (offline/local
// dev without having fetched it); the xmi-bigdata-integration CI job in
// go.yml fetches it explicitly and fails if the test unexpectedly skips.
func TestImport_BigData(t *testing.T) {
	const minSize = 1 * 1024 * 1024 // 1 MB — stub is only a few hundred bytes
	path := td("BigData.xmi")
	fi, err := os.Stat(path)
	if err != nil || fi.Size() < minSize {
		t.Skipf("BigData.xmi not present or too small (%d bytes); skipping integration test", func() int64 {
			if err == nil {
				return fi.Size()
			}
			return 0
		}())
	}

	r, err := xmi.Import(path, nil)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if len(r.Model.Model) == 0 {
		t.Error("expected non-empty model")
	}
	if len(r.Model.Specification.Elements) == 0 {
		t.Error("expected non-empty specification")
	}
	t.Logf("BigData: %d top-level elements, %d relationships, %d spec kinds, %d warnings",
		len(r.Model.Model), len(r.Model.Relationships),
		len(r.Model.Specification.Elements), len(r.Warnings))
}

// TestImport_SyntheticBigData exercises the importer at the same rough scale
// as TestImport_BigData's real export (windows-1252 encoding, nesting depth
// ~23, tens of thousands of elements), but via a generated fixture — so
// every CI job gets scale/depth coverage unconditionally, without depending
// on the real fixture having been fetched (#553). Complements, not replaces,
// TestImport_BigData: this one is always-on/synthetic for baseline coverage
// everywhere; that one is real-file/high-fidelity, but only in the dedicated
// xmi-bigdata-integration CI job that fetches it.
func TestImport_SyntheticBigData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "synthetic_bigdata.xmi")
	f, err := os.Create(path) // #nosec G304 -- path is our own t.TempDir() file
	if err != nil {
		t.Fatalf("create synthetic fixture: %v", err)
	}
	if err := writeSyntheticXMI(f, 20000, 23, 12); err != nil {
		t.Fatalf("generate synthetic fixture: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close synthetic fixture: %v", err)
	}
	if fi, err := os.Stat(path); err == nil {
		t.Logf("synthetic fixture size: %d bytes", fi.Size())
	}

	r, err := xmi.Import(path, nil)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if len(r.Model.Model) == 0 {
		t.Error("expected non-empty model")
	}
	if len(r.Model.Specification.Elements) == 0 {
		t.Error("expected non-empty specification")
	}
	if len(r.Model.Relationships) == 0 {
		t.Error("expected non-empty relationships")
	}
	t.Logf("synthetic BigData: %d top-level elements, %d relationships, %d spec kinds, %d warnings",
		len(r.Model.Model), len(r.Model.Relationships),
		len(r.Model.Specification.Elements), len(r.Warnings))
}

// ─── ParseKindMap ─────────────────────────────────────────────────────────────

func TestParseKindMap(t *testing.T) {
	tests := []struct {
		input   string
		want    map[string]string
		wantErr bool
	}{
		{"", nil, false},
		{"Component=service", map[string]string{"Component": "service"}, false},
		{"Component=service,Class=entity", map[string]string{"Component": "service", "Class": "entity"}, false},
		{" Component = service ", map[string]string{"Component": "service"}, false},
		{"bad", nil, true},
		{"=value", nil, true},
		{"key=", nil, true},
	}
	for _, tc := range tests {
		got, err := xmi.ParseKindMap(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseKindMap(%q): expected error", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseKindMap(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if len(got) != len(tc.want) {
			t.Errorf("ParseKindMap(%q): len=%d, want %d", tc.input, len(got), len(tc.want))
			continue
		}
		for k, v := range tc.want {
			if got[k] != v {
				t.Errorf("ParseKindMap(%q)[%q] = %q, want %q", tc.input, k, got[k], v)
			}
		}
	}
}

// ─── TS-XMI-11: Windows-1252 charset ─────────────────────────────────────────

func TestImport_Win1252Charset(t *testing.T) {
	// Build a Windows-1252 encoded XMI. The element name "ApiéTest" uses byte
	// 0xE9 (é in Latin-1/Windows-1252 = U+00E9). After sanitizeID it becomes "api-test".
	data := []byte("<?xml version=\"1.0\" encoding=\"windows-1252\"?>\n" +
		"<xmi:XMI xmlns:xmi=\"http://www.omg.org/spec/XMI/20131001\" xmi:version=\"2.1\">\n" +
		"  <uml:Model xmi:type=\"uml:Model\" name=\"Test\" xmi:id=\"m1\">\n" +
		"    <packagedElement xmi:type=\"uml:Component\" xmi:id=\"e1\" name=\"Api")
	data = append(data, 0xE9) // 'é' in Windows-1252
	data = append(data, []byte("Test\"/>\n  </uml:Model>\n</xmi:XMI>")...)

	path := filepath.Join(t.TempDir(), "win1252.xmi")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	r, err := xmi.Import(path, nil)
	if err != nil {
		t.Fatalf("Import with windows-1252 encoding: %v", err)
	}
	// "ApiéTest" → lower "apiétest" → non-ASCII stripped → "api-test"
	if _, ok := r.Model.Model["api-test"]; !ok {
		got := make([]string, 0, len(r.Model.Model))
		for k := range r.Model.Model {
			got = append(got, k)
		}
		t.Errorf("expected element 'api-test', got keys: %v", got)
	}
}

func TestImport_UnsupportedCharset(t *testing.T) {
	data := []byte("<?xml version=\"1.0\" encoding=\"utf-16\"?>\n" +
		"<xmi:XMI xmlns:xmi=\"http://www.omg.org/spec/XMI/20131001\" xmi:version=\"2.1\">\n" +
		"</xmi:XMI>")

	path := filepath.Join(t.TempDir(), "utf16.xmi")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := xmi.Import(path, nil)
	if err == nil {
		t.Fatal("expected error for unsupported charset, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "charset") &&
		!strings.Contains(strings.ToLower(err.Error()), "unsupported") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// ─── TS-XMI-12: maxDepth guard ───────────────────────────────────────────────

func TestImport_MaxDepthExceeded(t *testing.T) {
	// 55 nested packagedElement elements puts the deepest one at depth=57
	// (XMI=0, Model=1, p[0]=2 … p[54]=56), well above maxDepth=50.
	const levels = 55
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	b.WriteString("<xmi:XMI xmlns:xmi=\"http://www.omg.org/spec/XMI/20131001\" xmi:version=\"2.1\">\n")
	b.WriteString("  <uml:Model xmi:type=\"uml:Model\" name=\"Deep\" xmi:id=\"m1\">\n")
	for i := range levels {
		b.WriteString(strings.Repeat("  ", i+2))
		fmt.Fprintf(&b, "<packagedElement xmi:type=\"uml:Package\" xmi:id=\"p%d\" name=\"Level%d\">\n", i, i)
	}
	for i := levels - 1; i >= 0; i-- {
		b.WriteString(strings.Repeat("  ", i+2))
		b.WriteString("</packagedElement>\n")
	}
	b.WriteString("  </uml:Model>\n</xmi:XMI>")

	path := filepath.Join(t.TempDir(), "deep.xmi")
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := xmi.Import(path, nil)
	if err == nil {
		t.Fatal("expected error for element nesting beyond maxDepth, got nil")
	}
	if !strings.Contains(err.Error(), "depth") {
		t.Errorf("expected depth error, got: %v", err)
	}
}

// ─── TS-XMI-13: XMI version detection ────────────────────────────────────────

func TestImport_XMIVersionWarning(t *testing.T) {
	// XMI 1.1 has a completely different element encoding; only version 2.1 is supported.
	// The importer should succeed (not error) but emit a version warning.
	data := []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" +
		"<xmi:XMI xmlns:xmi=\"http://www.omg.org/spec/XMI/20110701\" xmi:version=\"1.1\">\n" +
		"  <uml:Model xmi:type=\"uml:Model\" name=\"Test\" xmi:id=\"m1\">\n" +
		"    <packagedElement xmi:type=\"uml:Component\" xmi:id=\"e1\" name=\"Api\"/>\n" +
		"  </uml:Model>\n</xmi:XMI>")

	path := filepath.Join(t.TempDir(), "xmi11.xmi")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	r, err := xmi.Import(path, nil)
	if err != nil {
		t.Fatalf("Import with XMI 1.1 should not error: %v", err)
	}

	hasVersionWarn := false
	for _, w := range r.Warnings {
		if strings.Contains(w, "1.1") || strings.Contains(strings.ToLower(w), "version") {
			hasVersionWarn = true
		}
	}
	if !hasVersionWarn {
		t.Errorf("expected version warning for XMI 1.1; got warnings: %v", r.Warnings)
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func td(name string) string {
	return filepath.Join("testdata", name)
}

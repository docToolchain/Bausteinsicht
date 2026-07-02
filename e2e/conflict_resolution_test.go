package e2e

// TestConflictResolution (#494) verifies the "model wins" conflict resolution rule:
// when both model and draw.io modify the same element field between syncs,
// the model value takes precedence and sync exits with code 1 + conflict count.
//
// Also verifies the one-sided-change cases (no conflict, clean reverse/forward).

import (
	"os"
	"strings"
	"testing"

	"github.com/beevik/etree"
	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// setCellAttr updates an existing attribute in-place or creates it if absent.
// Use instead of CreateAttr when the attribute may already exist.
func setCellAttr(el *etree.Element, key, value string) {
	for i, a := range el.Attr {
		if a.Key == key {
			el.Attr[i].Value = value
			return
		}
	}
	el.CreateAttr(key, value)
}

func TestConflictResolution(t *testing.T) {
	t.Run("ModelWins", testConflictModelWins)
	t.Run("OneSidedDrawio", testConflictOneSidedDrawio)
	t.Run("OneSidedModel", testConflictOneSidedModel)
}

// testConflictModelWins: both sides change customer.title → model value wins.
func testConflictModelWins(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	runCLI(t, bin, dir, "init")

	drawioPath := dir + "/architecture.drawio"
	modelPath := dir + "/architecture.jsonc"

	// ── Establish baseline state via first init sync ──────────────────────
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

	// ── Mutate draw.io: customer label → "API Service" ───────────────────
	obj := page.FindElement("customer")
	cellID := obj.SelectAttrValue("id", "")
	mutated := false
	for _, cell := range page.Root().SelectElements("mxCell") {
		if cell.SelectAttrValue("parent", "") == cellID &&
			strings.HasSuffix(cell.SelectAttrValue("id", ""), "-title") {
			// Update the existing "value" attribute in-place; CreateAttr would
			// add a duplicate and the original value would still be read back.
			setCellAttr(cell, "value", "API Service")
			mutated = true
			break
		}
	}
	if !mutated {
		// No title sub-cell found — this would mean the element uses an HTML
		// label. This should not happen after init (which always produces
		// sub-cell elements), so fail loudly to catch regressions.
		t.Fatal("customer element has no title sub-cell after init; cannot simulate draw.io edit")
	}
	if err := drawio.SaveDocument(drawioPath, doc); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	// ── Mutate model: customer.title → "API Gateway" ─────────────────────
	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatalf("model.Load: %v", err)
	}
	cust := m.Model["customer"]
	cust.Title = "API Gateway"
	m.Model["customer"] = cust
	if err := model.Save(modelPath, m); err != nil {
		t.Fatalf("model.Save: %v", err)
	}

	// ── Sync: both sides changed → conflict, model wins ──────────────────
	out, code := runCLIAllowFail(t, bin, dir, "sync")
	if code != 1 {
		t.Errorf("expected exit code 1 (conflict), got %d\noutput: %s", code, out)
	}
	if !strings.Contains(out, "conflict") && !strings.Contains(strings.ToLower(out), "conflict") {
		t.Errorf("expected conflict mention in output, got: %s", out)
	}

	// Model value must be preserved.
	m2, err := model.Load(modelPath)
	if err != nil {
		t.Fatalf("model.Load after conflict sync: %v", err)
	}
	if m2.Model["customer"].Title != "API Gateway" {
		t.Errorf("model title after conflict = %q, want %q", m2.Model["customer"].Title, "API Gateway")
	}

	// draw.io must now reflect the model value.
	drawioBytes, err := os.ReadFile(drawioPath)
	if err != nil {
		t.Fatalf("read draw.io after conflict sync: %v", err)
	}
	if !strings.Contains(string(drawioBytes), "API Gateway") {
		t.Errorf("draw.io does not contain winning model value 'API Gateway' after conflict sync")
	}
}

// testConflictOneSidedDrawio: only draw.io changed → clean reverse sync, no conflict.
func testConflictOneSidedDrawio(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	runCLI(t, bin, dir, "init")

	drawioPath := dir + "/architecture.drawio"
	modelPath := dir + "/architecture.jsonc"

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
			setCellAttr(cell, "value", "Reverse Only")
			mutated = true
			break
		}
	}
	if !mutated {
		t.Fatal("customer element has no title sub-cell after init; cannot simulate draw.io edit")
	}
	if err := drawio.SaveDocument(drawioPath, doc); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	// model unchanged — only draw.io changed: reverse sync, no conflict.
	_, code := runCLIAllowFail(t, bin, dir, "sync")
	if code != 0 {
		t.Errorf("expected exit code 0 (no conflict), got %d", code)
	}

	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatalf("model.Load: %v", err)
	}
	if m.Model["customer"].Title != "Reverse Only" {
		t.Errorf("reverse sync: customer.Title = %q, want %q", m.Model["customer"].Title, "Reverse Only")
	}
}

// testConflictOneSidedModel: only model changed → clean forward sync, no conflict.
func testConflictOneSidedModel(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	runCLI(t, bin, dir, "init")

	modelPath := dir + "/architecture.jsonc"
	drawioPath := dir + "/architecture.drawio"

	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatalf("model.Load: %v", err)
	}
	cust := m.Model["customer"]
	cust.Title = "Forward Only"
	m.Model["customer"] = cust
	if err := model.Save(modelPath, m); err != nil {
		t.Fatalf("model.Save: %v", err)
	}

	// draw.io unchanged — only model changed: forward sync, no conflict.
	_, code := runCLIAllowFail(t, bin, dir, "sync")
	if code != 0 {
		t.Errorf("expected exit code 0 (no conflict), got %d", code)
	}

	drawioBytes, err := os.ReadFile(drawioPath)
	if err != nil {
		t.Fatalf("read draw.io after forward sync: %v", err)
	}
	if !strings.Contains(string(drawioBytes), "Forward Only") {
		t.Errorf("draw.io does not contain forwarded title 'Forward Only'")
	}
}

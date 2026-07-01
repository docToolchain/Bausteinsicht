package e2e

// TestBidirectionalSyncRoundtrip (#484) verifies the full bidirectional sync cycle:
//
//  1. Forward:  init (creates model + draw.io) — elements appear in draw.io
//  2. Reverse:  mutate a draw.io title sub-cell → second sync → model title updated
//  3. Idempotency: third sync — sync state unchanged (stable state)

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func TestBidirectionalSyncRoundtrip(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	// ── Step 1: init (forward sync included) ──────────────────────────────────
	runCLI(t, bin, dir, "init")

	drawioPath := dir + "/architecture.drawio"
	modelPath := dir + "/architecture.jsonc"

	// Verify "customer" element exists in draw.io after forward sync.
	doc, err := drawio.LoadDocument(drawioPath)
	if err != nil {
		t.Fatalf("LoadDocument after init: %v", err)
	}
	var customerPage *drawio.Page
	for _, p := range doc.Pages() {
		if p.FindElement("customer") != nil {
			customerPage = p
			break
		}
	}
	if customerPage == nil {
		t.Fatal("element 'customer' not found in any draw.io page after init+sync")
	}
	customerObj := customerPage.FindElement("customer")

	// ── Step 2: mutate draw.io title sub-cell to simulate a draw.io label edit ─
	// Sub-cells (title/tech/desc) are siblings of the <object> in the XML root,
	// not children — they reference the parent via the "parent" attribute.
	const newTitle = "VIP Customer"
	cellID := customerObj.SelectAttrValue("id", "") // e.g. "context--customer"
	xmlRoot := customerPage.Root()

	mutated := false
	for _, cell := range xmlRoot.SelectElements("mxCell") {
		if cell.SelectAttrValue("parent", "") == cellID &&
			strings.HasSuffix(cell.SelectAttrValue("id", ""), "-title") {
			cell.CreateAttr("value", newTitle)
			mutated = true
			break
		}
	}
	if !mutated {
		// Fallback: element uses a plain HTML label on the <object> itself.
		if customerObj.SelectAttrValue("label", "") != "" {
			customerObj.CreateAttr("label", newTitle)
			mutated = true
		}
	}
	if !mutated {
		t.Fatalf("could not locate title sub-cell (parent=%q) or label on 'customer' element", cellID)
	}
	if err := drawio.SaveDocument(drawioPath, doc); err != nil {
		t.Fatalf("SaveDocument after mutation: %v", err)
	}

	// ── Step 3: second sync — reverse direction updates the model ─────────────
	out := runCLI(t, bin, dir, "sync")
	t.Logf("second sync output: %s", out)

	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatalf("model.Load after reverse sync: %v", err)
	}
	customerElem, ok := m.Model["customer"]
	if !ok {
		t.Fatal("element 'customer' not found in model after reverse sync")
	}
	if customerElem.Title != newTitle {
		t.Errorf("reverse sync: customer.Title = %q, want %q", customerElem.Title, newTitle)
	}

	// ── Step 4: third sync — idempotency check ───────────────────────────────
	// The state file includes a per-run timestamp and checksum that always
	// change. Compare only the stable fields (model_hash, drawio_hash,
	// elements) to verify no functional content changed.
	statePath := dir + "/.bausteinsicht-sync"
	state2, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read .bausteinsicht-sync after 2nd sync: %v", err)
	}

	runCLI(t, bin, dir, "sync")

	state3, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read .bausteinsicht-sync after 3rd sync: %v", err)
	}

	stableFields := func(raw []byte) []byte {
		var m map[string]interface{}
		if jsonErr := json.Unmarshal(raw, &m); jsonErr != nil {
			t.Fatalf("parse .bausteinsicht-sync: %v", jsonErr)
		}
		delete(m, "timestamp")
		delete(m, "checksum")
		out, _ := json.Marshal(m)
		return out
	}
	n2 := stableFields(state2)
	n3 := stableFields(state3)
	if !bytes.Equal(n2, n3) {
		t.Errorf("idempotency: stable sync state changed after no-op 3rd sync\nbefore: %s\nafter:  %s", n2, n3)
	}

	m2, err := model.Load(modelPath)
	if err != nil {
		t.Fatalf("model.Load after third sync: %v", err)
	}
	if m2.Model["customer"].Title != newTitle {
		t.Errorf("idempotency: customer.Title reverted to %q after third sync", m2.Model["customer"].Title)
	}

	t.Logf("bidirectional sync roundtrip OK: customer.Title=%q, state file stable", newTitle)
}

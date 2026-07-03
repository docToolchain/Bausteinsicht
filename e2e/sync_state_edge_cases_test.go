package e2e

// TestSyncStateEdgeCases (#519) closes E2E-Test-Plan.adoc section 12 (6
// tests, previously ~1/6 covered — only the idempotency check in
// sync_roundtrip_test.go touched .bausteinsicht-sync at all). Exercises
// internal/sync/state.go LoadState's error/fallback paths directly through
// the CLI.
//
// NOTE on a stale "Expected" outcome found while writing this: state.go:60-65
// explicitly treats a zero-byte sync-state file the same as a missing one
// (empty state, no error — "e.g. truncated write"), not the "Clear error"
// the original test plan row 12.5 described. Corrected in the same change
// that adds this file.

import (
	"os"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
)

const syncStatePath = "/.bausteinsicht-sync"

// stripChecksumField removes the "checksum" field from a sync-state JSON
// document. internal/sync/state.go only skips checksum verification when
// the field is absent entirely ("backward compat: old files without
// checksum skip validation") — leaving a stale checksum after hand-editing
// the file trips the checksum-mismatch error path instead of exercising
// the graceful-recovery behavior these tests target.
func stripChecksumField(t *testing.T, data string) string {
	t.Helper()
	const prefix = `"checksum": "`
	start := strings.Index(data, prefix)
	if start < 0 {
		t.Fatal(`"checksum" field not found in sync state`)
	}
	end := strings.Index(data[start+len(prefix):], `",`)
	if end < 0 {
		t.Fatal("could not find end of checksum field value")
	}
	return data[:start] + data[start+len(prefix)+end+2:]
}

// countPerPage returns, for each page, how many elements on it carry the
// given bausteinsicht_id — used to detect duplicates *within* a page, as
// opposed to the same element legitimately appearing on several different
// view pages (e.g. the sample model's "customer" is included in both the
// "context" and "containers" views by design).
func countPerPage(doc *drawio.Document, bausteinsichtID string) map[string]int {
	counts := make(map[string]int)
	for _, page := range doc.Pages() {
		for _, el := range page.FindAllElements() {
			if el.SelectAttrValue("bausteinsicht_id", "") == bausteinsichtID {
				counts[page.ID()]++
			}
		}
	}
	return counts
}

func assertNoDuplicatesPerPage(t *testing.T, doc *drawio.Document, bausteinsichtID string) {
	t.Helper()
	for pageID, count := range countPerPage(doc, bausteinsichtID) {
		if count != 1 {
			t.Errorf("page %q: %d instances of %q, want exactly 1", pageID, count, bausteinsichtID)
		}
	}
}

func TestSyncStateEdgeCases(t *testing.T) {
	t.Run("12.1_CorruptSyncState", test12_1CorruptSyncState)
	t.Run("12.2_ExtraStaleEntries", test12_2ExtraStaleEntries)
	t.Run("12.3_MissingEntries", test12_3MissingEntries)
	t.Run("12.4_EmptySyncStateObject", test12_4EmptySyncStateObject)
	t.Run("12.5_EmptyFile", test12_5EmptyFile)
	t.Run("12.6_JSONArrayInsteadOfObject", test12_6JSONArrayInsteadOfObject)
}

// test12_1CorruptSyncState covers 12.1: malformed JSON in .bausteinsicht-sync
// is a clear parse error, not a crash.
func test12_1CorruptSyncState(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	writeFile(t, dir+syncStatePath, "{{{invalid")

	out, code := runCLIAllowFail(t, bin, dir, "sync")
	if code != 2 {
		t.Fatalf("expected exit 2, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "loading sync state") {
		t.Errorf("expected a 'loading sync state' error, got: %s", out)
	}
}

// test12_2ExtraStaleEntries covers 12.2: a fake element entry in the sync
// state that doesn't correspond to any real model/drawio element is dropped
// gracefully on the next sync, not treated as an error or phantom element.
func test12_2ExtraStaleEntries(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	statePath := dir + syncStatePath
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read sync state: %v", err)
	}
	patched := strings.Replace(string(data), `"elements": {`,
		`"elements": { "totallyFakeElement": {"title": "Ghost", "kind": "system"},`, 1)
	if patched == string(data) {
		t.Fatal("anchor \"elements\": { not found in sync state")
	}
	patched = stripChecksumField(t, patched)
	if err := os.WriteFile(statePath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write sync state: %v", err)
	}

	out, code := runCLIAllowFail(t, bin, dir, "sync")
	if code != 0 {
		t.Fatalf("expected exit 0 (graceful), got %d\n%s", code, out)
	}

	after, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read sync state after sync: %v", err)
	}
	if strings.Contains(string(after), "totallyFakeElement") {
		t.Error("stale fake element entry should be dropped when sync state is rebuilt")
	}
}

// test12_3MissingEntries covers 12.3: removing a real, unchanged element's
// entry from the sync state (while it stays present and identical in both
// model and drawio) does not create a duplicate on the next sync.
func test12_3MissingEntries(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	statePath := dir + syncStatePath
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read sync state: %v", err)
	}
	// Remove the "customer" element's entry from the elements map by
	// blanking its ID key — a crude but effective way to make LoadState's
	// map lookup for "customer" miss without hand-rolling a JSON editor.
	patched := strings.Replace(string(data), `"customer":`, `"customer_removed_for_test":`, 1)
	if patched == string(data) {
		t.Fatal(`anchor "customer": not found in sync state`)
	}
	patched = stripChecksumField(t, patched)
	if err := os.WriteFile(statePath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write sync state: %v", err)
	}

	out, code := runCLIAllowFail(t, bin, dir, "sync")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d\n%s", code, out)
	}

	doc, err := drawio.LoadDocument(dir + "/architecture.drawio")
	if err != nil {
		t.Fatalf("LoadDocument: %v", err)
	}
	assertNoDuplicatesPerPage(t, doc, "customer")
}

// test12_4EmptySyncStateObject covers 12.4: "{}" is treated like missing
// state — no duplicates, and the state file is rebuilt with real content.
func test12_4EmptySyncStateObject(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	writeFile(t, dir+syncStatePath, "{}")

	out, code := runCLIAllowFail(t, bin, dir, "sync")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d\n%s", code, out)
	}

	rebuilt, err := os.ReadFile(dir + syncStatePath)
	if err != nil {
		t.Fatalf("read sync state after sync: %v", err)
	}
	if !strings.Contains(string(rebuilt), `"customer"`) {
		t.Error("expected sync state to be rebuilt with real element entries")
	}

	doc, err := drawio.LoadDocument(dir + "/architecture.drawio")
	if err != nil {
		t.Fatalf("LoadDocument: %v", err)
	}
	assertNoDuplicatesPerPage(t, doc, "customer")
}

// test12_5EmptyFile covers 12.5: a zero-byte sync-state file is treated the
// same as a missing one (internal/sync/state.go:60-65, "e.g. truncated
// write") — graceful, not a fatal error. This differs from the original
// test plan's "Clear error" expectation; see the package doc comment above.
func test12_5EmptyFile(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	writeFile(t, dir+syncStatePath, "")

	out, code := runCLIAllowFail(t, bin, dir, "sync")
	if code != 0 {
		t.Fatalf("expected exit 0 (zero-byte state treated as missing), got %d\n%s", code, out)
	}
}

// test12_6JSONArrayInsteadOfObject covers 12.6: a JSON array where an
// object is expected is a clear type error.
func test12_6JSONArrayInsteadOfObject(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	writeFile(t, dir+syncStatePath, "[]")

	out, code := runCLIAllowFail(t, bin, dir, "sync")
	if code != 2 {
		t.Fatalf("expected exit 2, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "loading sync state") {
		t.Errorf("expected a 'loading sync state' type error, got: %s", out)
	}
}

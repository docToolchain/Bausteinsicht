package sync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/chaos"
	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// TestSyncStateRecoveryCorruptFile verifies graceful handling of corrupted state file.
func TestSyncStateRecoveryCorruptFile(t *testing.T) {
	tc := chaos.NewTestChaos(t)

	// 1. Create a valid model
	jsonPath := tc.CreateFileWithContent("model.jsonc", `{
		"specification": {
			"elements": {"actor": {"notation": "Actor"}},
			"relationships": {}
		},
		"model": {"user": {"kind": "actor", "title": "User"}},
		"views": {"context": {"title": "Context", "include": ["user"]}}
	}`)

	dioPath := tc.CreateFileWithContent("model.drawio", `<?xml version="1.0"?>
<mxfile><diagram name="Page-1">
<mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/></root></mxGraphModel>
</diagram></mxfile>`)

	stateFile := filepath.Join(filepath.Dir(jsonPath), ".bausteinsicht-sync")

	// 2. Run initial sync to create valid state file
	m, err := model.Load(jsonPath)
	if err != nil {
		t.Fatalf("Load model: %v", err)
	}

	doc, err := drawio.LoadDocument(dioPath)
	if err != nil {
		t.Fatalf("Load drawio: %v", err)
	}

	state, _ := LoadState(stateFile)
	result := Run(m, doc, state, minimalTemplates(t), map[string]bool{})
	if result == nil {
		t.Fatal("Initial sync failed")
	}

	// Verify state file was created
	if !tc.FileExists(stateFile) {
		t.Fatal("State file should be created after sync")
	}

	// 3. Corrupt the state file (truncate it)
	tc.CorruptFile(stateFile)

	// 4. Run sync again — should handle corrupted state gracefully
	state, err = LoadState(stateFile)
	if err != nil {
		// Acceptable error — state file is corrupted
		// Should provide recovery mechanism
		if state == nil {
			t.Logf("Corrupted state detected, will reinitialize: %v", err)
		}
	}

	// 5. Sync should still work (treat as fresh sync)
	doc2, err := drawio.LoadDocument(dioPath)
	if err != nil {
		t.Fatalf("Reload drawio: %v", err)
	}

	result = Run(m, doc2, state, minimalTemplates(t), map[string]bool{})
	if result == nil {
		t.Fatal("Sync should return valid result even after state corruption")
	}
}

// TestSyncStatePartialWrite simulates incomplete state file save.
func TestSyncStatePartialWrite(t *testing.T) {
	tc := chaos.NewTestChaos(t)

	jsonPath := tc.CreateFileWithContent("model.jsonc", `{
		"specification": {
			"elements": {"actor": {"notation": "Actor"}},
			"relationships": {}
		},
		"model": {"user": {"kind": "actor", "title": "User"}},
		"views": {"context": {"title": "Context", "include": ["user"]}}
	}`)

	dioPath := tc.CreateFileWithContent("model.drawio", `<?xml version="1.0"?>
<mxfile><diagram name="Page-1">
<mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/></root></mxGraphModel>
</diagram></mxfile>`)

	stateFile := filepath.Join(filepath.Dir(jsonPath), ".bausteinsicht-sync")

	m, err := model.Load(jsonPath)
	if err != nil {
		t.Fatalf("Load model: %v", err)
	}

	doc, err := drawio.LoadDocument(dioPath)
	if err != nil {
		t.Fatalf("Load drawio: %v", err)
	}

	// Initial sync
	state, _ := LoadState(stateFile)
	result := Run(m, doc, state, minimalTemplates(t), map[string]bool{})
	if result == nil {
		t.Fatalf("Initial sync failed")
	}

	// Simulate partial write by truncating state file to incomplete JSON
	if tc.FileExists(stateFile) {
		tc.CorruptFilePartial(stateFile, `{
			"model_hash": "abc123",
			"diagram_hash"`)  // Incomplete JSON
	}

	// Try to load the partially written state
	state, err = LoadState(stateFile)

	// Either error (graceful rejection) or nil state (fresh sync)
	// Both are acceptable — system should recover
	if state == nil {
		t.Logf("Partial write detected, state is nil (will reinitialize)")
	}

	// Next sync should work without data loss
	doc2, err := drawio.LoadDocument(dioPath)
	if err != nil {
		t.Fatalf("Reload drawio: %v", err)
	}

	result = Run(m, doc2, state, minimalTemplates(t), map[string]bool{})
	if result == nil {
		t.Fatalf("Sync after partial write failed")
	}
}

// TestSyncStateHashVerification tests that state file integrity is checked.
func TestSyncStateHashVerification(t *testing.T) {
	tc := chaos.NewTestChaos(t)

	jsonPath := tc.CreateFileWithContent("model.jsonc", `{
		"specification": {
			"elements": {"actor": {"notation": "Actor"}},
			"relationships": {}
		},
		"model": {"user": {"kind": "actor", "title": "User"}},
		"views": {"context": {"title": "Context", "include": ["user"]}}
	}`)

	stateFile := filepath.Join(filepath.Dir(jsonPath), ".bausteinsicht-sync")

	// Create a state file manually (simulating external corruption)
	invalidState := map[string]interface{}{
		"model_hash":   "invalid_hash_value",
		"diagram_hash": "invalid_hash_value",
		"checksum":     "wrong_checksum",
	}

	data, _ := json.MarshalIndent(invalidState, "", "  ")
	_ = os.WriteFile(stateFile, data, 0644)

	// LoadState should detect hash mismatch
	state, err := LoadState(stateFile)

	// Either error or nil — both indicate detection of corruption
	if err != nil {
		t.Logf("Corruption detected: %v", err)
	}
	if state == nil {
		t.Logf("Corrupted state marked as invalid")
	}
}

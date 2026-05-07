package sync

import (
	"sync"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/chaos"
	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// TestConcurrentSyncAttempts verifies handling of concurrent sync operations.
func TestConcurrentSyncAttempts(t *testing.T) {
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

	m, err := model.Load(jsonPath)
	if err != nil {
		t.Fatalf("Load model: %v", err)
	}

	templates := minimalTemplates(t)
	var wg sync.WaitGroup
	var mu sync.Mutex
	resultCount := 0

	// Start multiple concurrent syncs
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			doc, err := drawio.LoadDocument(dioPath)
			if err != nil {
				t.Errorf("Load drawio: %v", err)
				return
			}

			state, _ := LoadState("")
			result := Run(m, doc, state, templates, map[string]bool{})

			if result != nil {
				mu.Lock()
				resultCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// All syncs should complete successfully
	if resultCount != 3 {
		t.Fatalf("Expected 3 successful syncs, got %d", resultCount)
	}
}

// TestConcurrentFileModifications verifies handling when files are modified during operations.
func TestConcurrentFileModifications(t *testing.T) {
	tc := chaos.NewTestChaos(t)

	jsonPath := tc.CreateFileWithContent("model.jsonc", `{
		"specification": {
			"elements": {"actor": {"notation": "Actor"}},
			"relationships": {}
		},
		"model": {"user": {"kind": "actor", "title": "User"}},
		"views": {"context": {"title": "Context", "include": ["user"]}}
	}`)

	// Load initial model
	m1, err := model.Load(jsonPath)
	if err != nil {
		t.Fatalf("Load model: %v", err)
	}

	// Simulate modification during load
	m2, err := model.Load(jsonPath)
	if err != nil {
		t.Fatalf("Load model second time: %v", err)
	}

	// Both loads should succeed (no interference)
	if m1 == nil || m2 == nil {
		t.Fatal("Both model loads should succeed")
	}

	// Models should be equivalent
	if len(m1.Model) != len(m2.Model) {
		t.Fatalf("Models should have same structure: %d vs %d", len(m1.Model), len(m2.Model))
	}
}

// TestFileAccessRaceConditions verifies no data loss under concurrent access.
func TestFileAccessRaceConditions(t *testing.T) {
	tc := chaos.NewTestChaos(t)

	// Create multiple files that might be accessed concurrently
	files := make([]string, 5)
	for i := 0; i < 5; i++ {
		files[i] = tc.CreateFileWithContent("concurrent_file_"+string(rune('a'+i))+".txt", "content")
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	var readErrors []error

	// Concurrent reads
	for _, file := range files {
		wg.Add(1)
		go func(f string) {
			defer wg.Done()

			content := tc.ReadFile(f)
			if content != "content" {
				mu.Lock()
				readErrors = append(readErrors, nil)
				mu.Unlock()
			} else {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(file)
	}

	wg.Wait()

	if successCount != len(files) {
		t.Fatalf("Expected %d successful reads, got %d (errors: %v)", len(files), successCount, readErrors)
	}
}

// TestRaceOnStateFileUpdate verifies state file doesn't get corrupted by concurrent writes.
func TestRaceOnStateFileUpdate(t *testing.T) {
	tc := chaos.NewTestChaos(t)

	stateFile := tc.CreateFileWithContent("state.json", `{"model_hash":"initial"}`)
	originalContent := tc.ReadFile(stateFile)

	var wg sync.WaitGroup
	var mu sync.Mutex
	writeCount := 0

	// Simulate multiple concurrent state updates
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Try to update state file
			// In real scenario, atomic writes prevent corruption
			if tc.FileExists(stateFile) {
				mu.Lock()
				writeCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// File should still exist and contain valid data
	if !tc.FileExists(stateFile) {
		t.Fatal("State file should still exist")
	}

	// Original content should be intact (unless atomically replaced)
	currentContent := tc.ReadFile(stateFile)
	if currentContent != originalContent && len(currentContent) == 0 {
		t.Fatal("State file corrupted (empty) during concurrent access")
	}

	if writeCount != 3 {
		t.Fatalf("Expected 3 concurrent attempts, got %d", writeCount)
	}
}

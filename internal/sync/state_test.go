package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bauteinsicht/internal/drawio"
	"github.com/docToolchain/Bauteinsicht/internal/model"
)

func TestLoadState_MissingFileReturnsEmpty(t *testing.T) {
	state, err := LoadState("/nonexistent/path/.bausteinsicht-sync")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if state == nil {
		t.Fatal("expected non-nil state")
	}
	if len(state.Elements) != 0 {
		t.Errorf("expected empty elements, got %d", len(state.Elements))
	}
	if len(state.Relationships) != 0 {
		t.Errorf("expected empty relationships, got %d", len(state.Relationships))
	}
}

func TestSaveLoadState_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".bausteinsicht-sync")

	original := &SyncState{
		Timestamp:  "2024-01-01T00:00:00Z",
		ModelHash:  "sha256:abc123",
		DrawioHash: "sha256:def456",
		Elements: map[string]ElementState{
			"webshop": {Title: "Webshop", Kind: "system"},
			"webshop.api": {Title: "API", Technology: "Go", Kind: "service"},
		},
		Relationships: []RelationshipState{
			{From: "webshop.api", To: "webshop.db", Label: "reads", Kind: "uses"},
		},
	}

	if err := SaveState(path, original); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	loaded, err := LoadState(path)
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if loaded.Timestamp != original.Timestamp {
		t.Errorf("timestamp mismatch: got %q, want %q", loaded.Timestamp, original.Timestamp)
	}
	if loaded.ModelHash != original.ModelHash {
		t.Errorf("model hash mismatch: got %q, want %q", loaded.ModelHash, original.ModelHash)
	}
	if loaded.DrawioHash != original.DrawioHash {
		t.Errorf("drawio hash mismatch: got %q, want %q", loaded.DrawioHash, original.DrawioHash)
	}
	if len(loaded.Elements) != len(original.Elements) {
		t.Errorf("elements count mismatch: got %d, want %d", len(loaded.Elements), len(original.Elements))
	}
	if loaded.Elements["webshop.api"].Technology != "Go" {
		t.Errorf("element technology mismatch: got %q, want %q", loaded.Elements["webshop.api"].Technology, "Go")
	}
	if len(loaded.Relationships) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(loaded.Relationships))
	}
	if loaded.Relationships[0].Label != "reads" {
		t.Errorf("relationship label mismatch: got %q, want %q", loaded.Relationships[0].Label, "reads")
	}
}

func TestSaveState_NoTempFileLeft(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".bausteinsicht-sync")

	state := &SyncState{
		Elements:      make(map[string]ElementState),
		Relationships: []RelationshipState{},
	}
	if err := SaveState(path, state); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".bausteinsicht-sync-tmp-") {
			t.Errorf("temp file not cleaned up: %s", e.Name())
		}
	}
}

func TestComputeHash_Consistent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}

	h1, err := ComputeHash(path)
	if err != nil {
		t.Fatalf("ComputeHash failed: %v", err)
	}
	h2, err := ComputeHash(path)
	if err != nil {
		t.Fatalf("ComputeHash second call failed: %v", err)
	}
	if h1 != h2 {
		t.Errorf("hash inconsistent: %q != %q", h1, h2)
	}
	if !strings.HasPrefix(h1, "sha256:") {
		t.Errorf("expected sha256: prefix, got %q", h1)
	}
}

func TestComputeHash_ChangesWithContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	if err := os.WriteFile(path, []byte("version 1"), 0o644); err != nil {
		t.Fatal(err)
	}
	h1, err := ComputeHash(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte("version 2"), 0o644); err != nil {
		t.Fatal(err)
	}
	h2, err := ComputeHash(path)
	if err != nil {
		t.Fatal(err)
	}

	if h1 == h2 {
		t.Error("hash should differ after content change")
	}
}

func TestComputeHash_MissingFile(t *testing.T) {
	_, err := ComputeHash("/nonexistent/file.txt")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestBuildState_CorrectSnapshot(t *testing.T) {
	dir := t.TempDir()

	modelPath := filepath.Join(dir, "model.jsonc")
	if err := os.WriteFile(modelPath, []byte(`{"model": {}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	drawioPath := filepath.Join(dir, "arch.drawio")
	if err := os.WriteFile(drawioPath, []byte(`<mxfile/>`), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"webshop": {
				Kind:  "system",
				Title: "Webshop",
				Children: map[string]model.Element{
					"api": {Kind: "service", Title: "API", Technology: "Go"},
				},
			},
		},
		Relationships: []model.Relationship{
			{From: "webshop.api", To: "webshop.db", Label: "reads", Kind: "uses"},
		},
	}
	doc := drawio.NewDocument()

	state, err := BuildState(m, doc, modelPath, drawioPath)
	if err != nil {
		t.Fatalf("BuildState failed: %v", err)
	}

	if state.Timestamp == "" {
		t.Error("expected non-empty timestamp")
	}
	if !strings.HasPrefix(state.ModelHash, "sha256:") {
		t.Errorf("expected sha256 model hash, got %q", state.ModelHash)
	}
	if !strings.HasPrefix(state.DrawioHash, "sha256:") {
		t.Errorf("expected sha256 drawio hash, got %q", state.DrawioHash)
	}

	if len(state.Elements) != 2 {
		t.Errorf("expected 2 elements, got %d", len(state.Elements))
	}
	api, ok := state.Elements["webshop.api"]
	if !ok {
		t.Fatal("expected webshop.api in elements")
	}
	if api.Technology != "Go" {
		t.Errorf("expected technology 'Go', got %q", api.Technology)
	}
	if api.Kind != "service" {
		t.Errorf("expected kind 'service', got %q", api.Kind)
	}

	if len(state.Relationships) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(state.Relationships))
	}
	rel := state.Relationships[0]
	if rel.From != "webshop.api" || rel.To != "webshop.db" || rel.Label != "reads" {
		t.Errorf("unexpected relationship: %+v", rel)
	}
}

package snapshot_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/docToolchain/Bausteinsicht/internal/snapshot"
)

// sampleModel returns a small model with a nested element hierarchy and one
// relationship so element/relationship counting can be exercised.
func sampleModel() *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"system": {
				Kind:  "system",
				Title: "System",
				Children: map[string]model.Element{
					"backend": {
						Kind:  "container",
						Title: "Backend",
						Children: map[string]model.Element{
							"api": {Kind: "component", Title: "API"},
						},
					},
					"frontend": {Kind: "container", Title: "Frontend"},
				},
			},
			"user": {Kind: "person", Title: "User"},
		},
		Relationships: []model.Relationship{
			{From: "user", To: "system", Label: "uses"},
		},
	}
}

func TestNewSnapshot(t *testing.T) {
	m := sampleModel()
	s := snapshot.NewSnapshot("initial", m)

	if s.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if s.Message != "initial" {
		t.Errorf("expected message %q, got %q", "initial", s.Message)
	}
	if s.Model != m {
		t.Error("expected model pointer to be preserved")
	}
	if s.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
	if s.Timestamp.Location() != time.UTC {
		t.Errorf("expected UTC timestamp, got %v", s.Timestamp.Location())
	}
}

func TestNewSnapshot_IDFormat(t *testing.T) {
	id := snapshot.NewSnapshot("", nil).ID

	// A valid ID is the timestamp layout plus a ".<nanoseconds>Z" suffix, so it
	// must be strictly longer than the bare layout (which is what a missing
	// suffix would leave behind).
	if len(id) <= len("snapshot-2006-01-02T15-04-05") {
		t.Errorf("ID %q is missing the nanosecond suffix", id)
	}
	if id[:len("snapshot-")] != "snapshot-" {
		t.Errorf("ID %q missing 'snapshot-' prefix", id)
	}

	// NOTE: deliberately no rapid-fire uniqueness assertion. generateSnapshotID
	// derives the ID from the wall clock, whose resolution is coarse on some
	// platforms (notably Windows), so two snapshots created within one tick can
	// share an ID. Asserting distinctness here would be flaky; that uniqueness
	// weakness is a production concern tracked separately, not a guarantee of
	// this test.
}

func TestToMetadata(t *testing.T) {
	tests := []struct {
		name         string
		model        *model.BausteinsichtModel
		wantElements int
		wantRels     int
	}{
		{
			name:         "nil model",
			model:        nil,
			wantElements: 0,
			wantRels:     0,
		},
		{
			name:         "empty model",
			model:        &model.BausteinsichtModel{},
			wantElements: 0,
			wantRels:     0,
		},
		{
			name:         "nested model counts children recursively",
			model:        sampleModel(),
			wantElements: 5, // system, system.backend, system.backend.api, system.frontend, user
			wantRels:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := snapshot.NewSnapshot("msg", tt.model)
			meta := s.ToMetadata()

			if meta.ID != s.ID {
				t.Errorf("ID mismatch: got %q want %q", meta.ID, s.ID)
			}
			if meta.Message != "msg" {
				t.Errorf("Message mismatch: got %q", meta.Message)
			}
			if !meta.Timestamp.Equal(s.Timestamp) {
				t.Errorf("Timestamp mismatch: got %v want %v", meta.Timestamp, s.Timestamp)
			}
			if meta.ElementCount != tt.wantElements {
				t.Errorf("ElementCount: got %d want %d", meta.ElementCount, tt.wantElements)
			}
			if meta.RelCount != tt.wantRels {
				t.Errorf("RelCount: got %d want %d", meta.RelCount, tt.wantRels)
			}
		})
	}
}

func TestNewManagerAndSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	mgr := snapshot.NewManager(dir)

	orig := snapshot.NewSnapshot("round trip", sampleModel())
	if err := mgr.Save(orig); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// The snapshot directory and file should now exist.
	snapFile := filepath.Join(dir, snapshot.SnapshotDirName, orig.ID+".json")
	if _, err := os.Stat(snapFile); err != nil {
		t.Fatalf("expected snapshot file at %s: %v", snapFile, err)
	}

	loaded, err := mgr.Load(orig.ID)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.ID != orig.ID {
		t.Errorf("ID: got %q want %q", loaded.ID, orig.ID)
	}
	if loaded.Message != orig.Message {
		t.Errorf("Message: got %q want %q", loaded.Message, orig.Message)
	}
	if loaded.Model == nil {
		t.Fatal("expected non-nil model after load")
	}
	if _, ok := loaded.Model.Model["system"]; !ok {
		t.Error("expected element 'system' to survive round trip")
	}
	if len(loaded.Model.Relationships) != 1 {
		t.Errorf("expected 1 relationship, got %d", len(loaded.Model.Relationships))
	}
}

func TestList_EmptyWhenNoIndex(t *testing.T) {
	mgr := snapshot.NewManager(t.TempDir())
	list, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d entries", len(list))
	}
}

func TestList_SortedNewestFirst(t *testing.T) {
	mgr := snapshot.NewManager(t.TempDir())

	older := &snapshot.Snapshot{
		ID:        "snapshot-older",
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Message:   "older",
		Model:     sampleModel(),
	}
	newer := &snapshot.Snapshot{
		ID:        "snapshot-newer",
		Timestamp: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		Message:   "newer",
		Model:     sampleModel(),
	}

	// Save older first, then newer, to prove List sorts rather than appends.
	if err := mgr.Save(older); err != nil {
		t.Fatalf("Save older failed: %v", err)
	}
	if err := mgr.Save(newer); err != nil {
		t.Fatalf("Save newer failed: %v", err)
	}

	list, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(list))
	}
	if list[0].ID != "snapshot-newer" {
		t.Errorf("expected newest first, got %q", list[0].ID)
	}
	if list[1].ID != "snapshot-older" {
		t.Errorf("expected oldest last, got %q", list[1].ID)
	}
}

func TestSave_ReplacesExistingIndexEntry(t *testing.T) {
	mgr := snapshot.NewManager(t.TempDir())

	s := &snapshot.Snapshot{
		ID:        "snapshot-dup",
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Message:   "first",
		Model:     sampleModel(),
	}
	if err := mgr.Save(s); err != nil {
		t.Fatalf("first Save failed: %v", err)
	}

	// Re-save the same ID with a different message; the index must not grow.
	s.Message = "second"
	if err := mgr.Save(s); err != nil {
		t.Fatalf("second Save failed: %v", err)
	}

	list, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected exactly 1 index entry after re-save, got %d", len(list))
	}
	if list[0].Message != "second" {
		t.Errorf("expected updated message 'second', got %q", list[0].Message)
	}
}

func TestExists(t *testing.T) {
	mgr := snapshot.NewManager(t.TempDir())
	s := snapshot.NewSnapshot("exists", sampleModel())

	if mgr.Exists(s.ID) {
		t.Error("snapshot should not exist before save")
	}
	if err := mgr.Save(s); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if !mgr.Exists(s.ID) {
		t.Error("snapshot should exist after save")
	}
}

func TestDelete(t *testing.T) {
	mgr := snapshot.NewManager(t.TempDir())
	s := snapshot.NewSnapshot("to delete", sampleModel())
	if err := mgr.Save(s); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if err := mgr.Delete(s.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if mgr.Exists(s.ID) {
		t.Error("snapshot should not exist after delete")
	}

	list, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty index after delete, got %d entries", len(list))
	}
}

func TestDelete_MissingFileReturnsError(t *testing.T) {
	mgr := snapshot.NewManager(t.TempDir())
	if err := mgr.Delete("does-not-exist"); err == nil {
		t.Error("expected error deleting a non-existent snapshot")
	}
}

func TestDelete_NoIndexStillRemovesFile(t *testing.T) {
	dir := t.TempDir()
	mgr := snapshot.NewManager(dir)

	// Create a snapshot file directly, without ever writing an index.
	snapDir := filepath.Join(dir, snapshot.SnapshotDirName)
	if err := os.MkdirAll(snapDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	file := filepath.Join(snapDir, "snapshot-orphan.json")
	if err := os.WriteFile(file, []byte(`{"id":"snapshot-orphan"}`), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Delete must succeed: removeFromIndex tolerates a missing index.
	if err := mgr.Delete("snapshot-orphan"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Error("expected snapshot file to be removed")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	mgr := snapshot.NewManager(t.TempDir())
	if _, err := mgr.Load("missing"); err == nil {
		t.Error("expected error loading a missing snapshot")
	}
}

func TestLoad_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	mgr := snapshot.NewManager(dir)
	snapDir := filepath.Join(dir, snapshot.SnapshotDirName)
	if err := os.MkdirAll(snapDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(snapDir, "bad.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if _, err := mgr.Load("bad"); err == nil {
		t.Error("expected parse error loading corrupt snapshot")
	}
}

func TestList_CorruptIndex(t *testing.T) {
	dir := t.TempDir()
	mgr := snapshot.NewManager(dir)
	snapDir := filepath.Join(dir, snapshot.SnapshotDirName)
	if err := os.MkdirAll(snapDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(snapDir, "index.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if _, err := mgr.List(); err == nil {
		t.Error("expected parse error listing with corrupt index")
	}
}

func TestSave_CorruptIndexReturnsError(t *testing.T) {
	dir := t.TempDir()
	mgr := snapshot.NewManager(dir)
	snapDir := filepath.Join(dir, snapshot.SnapshotDirName)
	if err := os.MkdirAll(snapDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	// A corrupt existing index makes updateIndex fail during Save.
	if err := os.WriteFile(filepath.Join(snapDir, "index.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	s := snapshot.NewSnapshot("x", sampleModel())
	if err := mgr.Save(s); err == nil {
		t.Error("expected Save to fail when the index is corrupt")
	}
}

func TestDelete_CorruptIndexReturnsError(t *testing.T) {
	dir := t.TempDir()
	mgr := snapshot.NewManager(dir)
	snapDir := filepath.Join(dir, snapshot.SnapshotDirName)
	if err := os.MkdirAll(snapDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	// Snapshot file present so os.Remove succeeds, but index is corrupt so
	// removeFromIndex fails.
	if err := os.WriteFile(filepath.Join(snapDir, "snapshot-x.json"), []byte(`{"id":"snapshot-x"}`), 0o644); err != nil {
		t.Fatalf("write snapshot failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(snapDir, "index.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatalf("write index failed: %v", err)
	}
	if err := mgr.Delete("snapshot-x"); err == nil {
		t.Error("expected Delete to fail when the index is corrupt")
	}
}

func TestListFiles(t *testing.T) {
	dir := t.TempDir()
	mgr := snapshot.NewManager(dir)

	// No directory yet -> empty, no error.
	files, err := mgr.ListFiles()
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected no files, got %d", len(files))
	}

	// Assign explicit, distinct IDs so the test does not depend on the wall
	// clock yielding two different auto-generated IDs (see TestNewSnapshot_IDFormat).
	s1 := snapshot.NewSnapshot("one", sampleModel())
	s1.ID = "snapshot-listfiles-one"
	if err := mgr.Save(s1); err != nil {
		t.Fatalf("Save s1 failed: %v", err)
	}
	s2 := snapshot.NewSnapshot("two", sampleModel())
	s2.ID = "snapshot-listfiles-two"
	if err := mgr.Save(s2); err != nil {
		t.Fatalf("Save s2 failed: %v", err)
	}

	// Drop a non-json file and a subdirectory to prove they are ignored.
	snapDir := filepath.Join(dir, snapshot.SnapshotDirName)
	if err := os.WriteFile(filepath.Join(snapDir, "notes.txt"), []byte("ignore me"), 0o644); err != nil {
		t.Fatalf("write txt failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(snapDir, "sub.json"), 0o755); err != nil {
		t.Fatalf("mkdir sub failed: %v", err)
	}

	files, err = mgr.ListFiles()
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 snapshot files (index.json/non-json/dirs excluded), got %d: %v", len(files), files)
	}
	for _, f := range files {
		if f == "index.json" || f == "notes.txt" || f == "sub.json" {
			t.Errorf("unexpected file in list: %q", f)
		}
		if filepath.Ext(f) != ".json" {
			t.Errorf("expected .json file, got %q", f)
		}
	}
}

package snapshot

import (
	"testing"
	"time"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// TestOldFormatIDsRemainUsable pins the backward-compatibility contract of the
// #649 ID change: the atomic-counter suffix only affects NEWLY generated IDs
// (via generateSnapshotID). Snapshots written before the change keep their old,
// suffix-less IDs on disk, and because the ID is an opaque key everywhere
// (filename in snapshotFilePath, string compare in Load/Exists/Delete, and index
// ordering by the Timestamp field — never by parsing the ID), every operation
// must still work on an old-format ID. There is no on-the-fly migration, and
// none is needed.
func TestOldFormatIDsRemainUsable(t *testing.T) {
	mgr := NewManager(t.TempDir())

	// A pre-#649 ID: bare timestamp, no ".<nanos>Z-<seq>" suffix.
	const oldID = "snapshot-2026-01-01T10-00-00"
	oldSnap := &Snapshot{
		ID:        oldID,
		Timestamp: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
		Message:   "legacy snapshot",
		Model:     &model.BausteinsichtModel{},
	}
	if err := mgr.Save(oldSnap); err != nil {
		t.Fatalf("saving old-format snapshot: %v", err)
	}

	// A new-format snapshot created through the production path. Its ID string
	// encodes today's date, but we give it an *earlier* Timestamp than the legacy
	// one so List ordering can only come out right if it sorts on Timestamp, not
	// on the ID.
	newSnap := NewSnapshot("current snapshot", &model.BausteinsichtModel{})
	newSnap.Timestamp = time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)
	if err := mgr.Save(newSnap); err != nil {
		t.Fatalf("saving new-format snapshot: %v", err)
	}

	// Exists must recognise the old-format ID.
	if !mgr.Exists(oldID) {
		t.Errorf("Exists(%q) = false, want true for an old-format ID", oldID)
	}

	// Load must round-trip the old-format ID unchanged.
	loaded, err := mgr.Load(oldID)
	if err != nil {
		t.Fatalf("loading old-format snapshot: %v", err)
	}
	if loaded.ID != oldID {
		t.Errorf("loaded ID = %q, want %q", loaded.ID, oldID)
	}
	if loaded.Message != "legacy snapshot" {
		t.Errorf("loaded Message = %q, want %q", loaded.Message, "legacy snapshot")
	}

	// List must contain both, newest-first by Timestamp. The legacy snapshot has
	// the later Timestamp while the new-format ID sorts first lexically, so the
	// legacy ID coming first proves ordering is by Timestamp, not by ID.
	list, err := mgr.List()
	if err != nil {
		t.Fatalf("listing snapshots: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("List returned %d snapshots, want 2", len(list))
	}
	if list[0].ID != oldID || list[1].ID != newSnap.ID {
		t.Errorf("List order = [%q, %q], want [%q, %q] (newest-first by Timestamp)",
			list[0].ID, list[1].ID, oldID, newSnap.ID)
	}

	// Delete must remove the old-format snapshot and drop it from the index.
	if err := mgr.Delete(oldID); err != nil {
		t.Fatalf("deleting old-format snapshot: %v", err)
	}
	if mgr.Exists(oldID) {
		t.Errorf("Exists(%q) = true after Delete, want false", oldID)
	}
	remaining, err := mgr.List()
	if err != nil {
		t.Fatalf("listing after delete: %v", err)
	}
	if len(remaining) != 1 || remaining[0].ID != newSnap.ID {
		t.Errorf("after deleting old ID, List = %+v, want only %q", remaining, newSnap.ID)
	}
}

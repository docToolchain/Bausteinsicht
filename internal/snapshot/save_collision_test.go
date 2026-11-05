package snapshot

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// TestSaveRejectsDuplicateID proves Save is fail-closed: once a snapshot with a
// given ID is persisted, a second Save that resolves to the same ID must fail
// instead of silently overwriting the first. The in-process counter prevents
// collisions within one process, but two CLI invocations in the same clock tick
// generate the same ID across processes; without the O_EXCL guard the second
// Save overwrites the first and the earlier snapshot is lost (#648).
func TestSaveRejectsDuplicateID(t *testing.T) {
	mgr := NewManager(t.TempDir())

	first := &Snapshot{
		ID:        "snapshot-2026-01-02T15-04-05.000000000Z-000001",
		Timestamp: time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC),
		Message:   "first",
		Model:     &model.BausteinsichtModel{},
	}
	if err := mgr.Save(first); err != nil {
		t.Fatalf("saving first snapshot: %v", err)
	}

	// Same ID, different content: simulates a same-tick collision from another
	// process, which the per-process counter cannot prevent.
	second := &Snapshot{
		ID:        first.ID,
		Timestamp: first.Timestamp,
		Message:   "second",
		Model:     &model.BausteinsichtModel{},
	}
	if err := mgr.Save(second); err == nil {
		t.Fatalf("Save overwrote existing snapshot ID %q instead of failing; the earlier snapshot would be lost", first.ID)
	}

	// The original snapshot must survive the rejected overwrite.
	got, err := mgr.Load(first.ID)
	if err != nil {
		t.Fatalf("loading snapshot after rejected overwrite: %v", err)
	}
	if got.Message != "first" {
		t.Errorf("snapshot was overwritten: message = %q, want %q", got.Message, "first")
	}
}

// TestSaveSurfacesFilesystemError verifies Save reports a filesystem failure
// (here: an unwritable snapshot directory) rather than swallowing it. This also
// exercises the non-collision branch of the O_EXCL open error, so the
// fail-closed write path is fully covered, not just the duplicate-ID case.
func TestSaveSurfacesFilesystemError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory write permissions are not enforced the same way on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permissions")
	}

	base := t.TempDir()
	mgr := NewManager(base)

	// Seed one snapshot so the snapshot directory exists.
	if err := mgr.Save(&Snapshot{
		ID:        "snapshot-a",
		Timestamp: time.Now().UTC(),
		Model:     &model.BausteinsichtModel{},
	}); err != nil {
		t.Fatalf("seed save: %v", err)
	}

	// Make the snapshot directory read-only so creating a new file fails with a
	// permission error (not os.ErrExist).
	snapDir := filepath.Join(base, snapshotDir)
	if err := os.Chmod(snapDir, 0o500); err != nil {
		t.Fatalf("chmod snapshot dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(snapDir, 0o700) }) // restore so t.TempDir can clean up

	err := mgr.Save(&Snapshot{
		ID:        "snapshot-b",
		Timestamp: time.Now().UTC(),
		Model:     &model.BausteinsichtModel{},
	})
	if err == nil {
		t.Fatal("expected Save to fail when the snapshot directory is not writable")
	}
	// Assert positively that the permission failure is surfaced: it must not be
	// mistaken for (or masked as) a collision, and the real cause must survive the
	// %w wrapping so callers can inspect it.
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("expected a permission error to be reported, got: %v", err)
	}
}

// TestSaveCleansUpAfterPostCreateFailure verifies that when Save fails after it
// has already created the snapshot file (here: the index write fails), it
// removes that file again. Otherwise the orphaned file would make O_EXCL reject
// every later retry of the same ID, so the snapshot could never be saved.
func TestSaveCleansUpAfterPostCreateFailure(t *testing.T) {
	base := t.TempDir()
	mgr := NewManager(base)

	// Make the index path a directory so updateIndex's file write fails *after*
	// the snapshot file has been created. This is cross-platform (writing a file
	// over a directory fails everywhere) and needs no permission tricks.
	indexAsDir := filepath.Join(base, snapshotDir, "index.json")
	if err := os.MkdirAll(indexAsDir, 0o755); err != nil {
		t.Fatalf("seeding index-as-directory: %v", err)
	}

	snap := &Snapshot{
		ID:        "snapshot-cleanup",
		Timestamp: time.Now().UTC(),
		Model:     &model.BausteinsichtModel{},
	}
	if err := mgr.Save(snap); err == nil {
		t.Fatal("expected Save to fail when the index cannot be written")
	}

	// The snapshot file this call created must be gone, so O_EXCL does not block
	// a retry on the same ID.
	if mgr.Exists(snap.ID) {
		t.Errorf("snapshot file %q was left behind after a failed Save", snap.ID)
	}
}

// TestSaveSurfacesSnapshotDirError verifies Save reports a failure to create the
// snapshot directory instead of proceeding. Pointing the manager at a path whose
// parent is a regular file makes MkdirAll fail cross-platform.
func TestSaveSurfacesSnapshotDirError(t *testing.T) {
	base := t.TempDir()
	notADir := filepath.Join(base, "not-a-dir")
	if err := os.WriteFile(notADir, []byte("x"), fileMode0644); err != nil {
		t.Fatalf("seeding file in place of base dir: %v", err)
	}

	mgr := NewManager(notADir) // snapshot dir would be a child of a regular file
	err := mgr.Save(&Snapshot{
		ID:        "snapshot-nodir",
		Timestamp: time.Now().UTC(),
		Model:     &model.BausteinsichtModel{},
	})
	if err == nil {
		t.Fatal("expected Save to fail when the snapshot directory cannot be created")
	}
}

// TestSaveSurfacesIndexLockError verifies the index-lock acquisition is itself
// fail-closed: if the lock file cannot be opened, Save reports the error instead
// of updating the index unsynchronized, and still cleans up the snapshot file it
// created. Making the lock path a directory forces the open to fail
// cross-platform without permission tricks.
func TestSaveSurfacesIndexLockError(t *testing.T) {
	base := t.TempDir()
	mgr := NewManager(base)

	// Pre-create the lock path as a directory so opening it as a file fails.
	lockAsDir := filepath.Join(base, snapshotDir, "index.json.lock")
	if err := os.MkdirAll(lockAsDir, 0o755); err != nil {
		t.Fatalf("seeding lock-as-directory: %v", err)
	}

	snap := &Snapshot{
		ID:        "snapshot-lockfail",
		Timestamp: time.Now().UTC(),
		Model:     &model.BausteinsichtModel{},
	}
	if err := mgr.Save(snap); err == nil {
		t.Fatal("expected Save to fail when the index lock cannot be opened")
	}

	// The snapshot file must not linger after the failed Save.
	if mgr.Exists(snap.ID) {
		t.Errorf("snapshot file %q was left behind after a failed index lock", snap.ID)
	}
}

// TestSaveSurfacesCorruptIndex verifies Save reports a corrupt existing index
// instead of silently discarding it (which would drop every earlier entry), and
// still cleans up the snapshot file it created.
func TestSaveSurfacesCorruptIndex(t *testing.T) {
	base := t.TempDir()
	mgr := NewManager(base)

	if err := mgr.Save(&Snapshot{
		ID: "snapshot-ok", Timestamp: time.Now().UTC(), Model: &model.BausteinsichtModel{},
	}); err != nil {
		t.Fatalf("seed save: %v", err)
	}

	indexPath := filepath.Join(base, snapshotDir, "index.json")
	if err := os.WriteFile(indexPath, []byte("{ not valid json"), fileMode0644); err != nil {
		t.Fatalf("corrupting index: %v", err)
	}

	err := mgr.Save(&Snapshot{
		ID: "snapshot-next", Timestamp: time.Now().UTC(), Model: &model.BausteinsichtModel{},
	})
	if err == nil {
		t.Fatal("expected Save to fail when the existing index cannot be parsed")
	}
	if mgr.Exists("snapshot-next") {
		t.Errorf("snapshot file was left behind after a failed index parse")
	}
}

// TestDeleteSurfacesCorruptIndex verifies Delete reports a corrupt index rather
// than swallowing the failure after it has already removed the snapshot file.
func TestDeleteSurfacesCorruptIndex(t *testing.T) {
	base := t.TempDir()
	mgr := NewManager(base)

	if err := mgr.Save(&Snapshot{
		ID: "snapshot-del", Timestamp: time.Now().UTC(), Model: &model.BausteinsichtModel{},
	}); err != nil {
		t.Fatalf("seed save: %v", err)
	}

	indexPath := filepath.Join(base, snapshotDir, "index.json")
	if err := os.WriteFile(indexPath, []byte("{ not valid json"), fileMode0644); err != nil {
		t.Fatalf("corrupting index: %v", err)
	}

	if err := mgr.Delete("snapshot-del"); err == nil {
		t.Fatal("expected Delete to fail when the index cannot be parsed")
	}
}

// TestDeleteWithoutIndexSucceeds verifies Delete treats a missing index as a
// no-op for the index update rather than an error: the snapshot file is gone and
// there is nothing to unlist.
func TestDeleteWithoutIndexSucceeds(t *testing.T) {
	base := t.TempDir()
	mgr := NewManager(base)

	if err := mgr.Save(&Snapshot{
		ID: "snapshot-noidx", Timestamp: time.Now().UTC(), Model: &model.BausteinsichtModel{},
	}); err != nil {
		t.Fatalf("seed save: %v", err)
	}

	// Remove the index entirely; the snapshot file remains.
	if err := os.Remove(filepath.Join(base, snapshotDir, "index.json")); err != nil {
		t.Fatalf("removing index: %v", err)
	}

	if err := mgr.Delete("snapshot-noidx"); err != nil {
		t.Errorf("Delete should succeed when there is no index to update: %v", err)
	}
}

// TestSaveKeepsIndexIntactWhenIndexWriteFails proves the index update is atomic.
// A first snapshot is saved, then a second Save is forced to fail during the
// index write. The temp-file+rename write must leave the already-published index
// untouched (not truncated, not partially overwritten) and the failed Save must
// clean up its own snapshot file — so List never references a snapshot that the
// failed write caused to be deleted.
func TestSaveKeepsIndexIntactWhenIndexWriteFails(t *testing.T) {
	base := t.TempDir()
	mgr := NewManager(base)

	first := &Snapshot{
		ID:        "snapshot-keep",
		Timestamp: time.Now().UTC(),
		Message:   "first",
		Model:     &model.BausteinsichtModel{},
	}
	if err := mgr.Save(first); err != nil {
		t.Fatalf("saving first snapshot: %v", err)
	}

	// Force writeFileAtomic to fail when it opens the temp file: make the temp
	// path a directory. index.json itself stays a valid file holding "first".
	tmpAsDir := filepath.Join(base, snapshotDir, "index.json.tmp")
	if err := os.MkdirAll(tmpAsDir, 0o755); err != nil {
		t.Fatalf("seeding temp-as-directory: %v", err)
	}

	second := &Snapshot{
		ID:        "snapshot-lost",
		Timestamp: time.Now().UTC(),
		Message:   "second",
		Model:     &model.BausteinsichtModel{},
	}
	if err := mgr.Save(second); err == nil {
		t.Fatal("expected Save to fail when the index temp file cannot be created")
	}

	// The failed Save must not leave its snapshot file behind...
	if mgr.Exists(second.ID) {
		t.Errorf("snapshot file %q was left behind after a failed index write", second.ID)
	}

	// ...and the previously published index must still list exactly the first
	// snapshot, intact and loadable.
	_ = os.Remove(tmpAsDir) // clear the obstruction so the index can be read back
	list, err := mgr.List()
	if err != nil {
		t.Fatalf("listing after failed write: %v", err)
	}
	if len(list) != 1 || list[0].ID != first.ID {
		t.Fatalf("index was corrupted by a failed write: got %+v, want only %q", list, first.ID)
	}
	if _, err := mgr.Load(first.ID); err != nil {
		t.Errorf("first snapshot no longer loadable after failed write: %v", err)
	}
}

// TestSaveConcurrentDistinctIDsAllIndexed guards against lost index updates.
// O_EXCL only protects the per-snapshot file; the index is a read-modify-write.
// When several saves with distinct IDs run concurrently they all create their
// files, but without serialization their index updates interleave and the last
// writer drops the others' entries, so List silently loses snapshots. Each
// snapshot file exists on disk, yet it is missing from the index. Run with -race
// to also catch the data race on index.json. The index lock makes every entry
// survive.
func TestSaveConcurrentDistinctIDsAllIndexed(t *testing.T) {
	mgr := NewManager(t.TempDir())

	const n = 24
	var wg sync.WaitGroup
	errs := make(chan error, n)
	start := make(chan struct{})

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start // release all goroutines together to maximise contention
			snap := &Snapshot{
				ID:        fmt.Sprintf("snapshot-concurrent-%03d", i),
				Timestamp: time.Now().UTC(),
				Message:   fmt.Sprintf("snapshot %d", i),
				Model:     &model.BausteinsichtModel{},
			}
			if err := mgr.Save(snap); err != nil {
				errs <- err
			}
		}(i)
	}

	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent Save failed: %v", err)
	}

	list, err := mgr.List()
	if err != nil {
		t.Fatalf("listing snapshots: %v", err)
	}
	if len(list) != n {
		t.Errorf("List returned %d snapshots, want %d; concurrent index updates lost entries", len(list), n)
	}
}

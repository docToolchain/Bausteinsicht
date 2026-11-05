package snapshot

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	snapshotDir     = ".bausteinsicht-snapshots"
	indexFile       = "index.json"
	indexLockFile   = "index.json.lock"
	snapshotDir0755 = 0o755
	fileMode0644    = 0o644
)

// Manager handles snapshot storage and retrieval
type Manager struct {
	baseDir string
}

// NewManager creates a new snapshot manager for a given directory
func NewManager(baseDir string) *Manager {
	return &Manager{baseDir: baseDir}
}

// snapshotPath returns the path to the snapshot directory
func (m *Manager) snapshotPath() string {
	return filepath.Join(m.baseDir, snapshotDir)
}

// indexPath returns the path to the snapshot index file
func (m *Manager) indexPath() string {
	return filepath.Join(m.snapshotPath(), indexFile)
}

// indexLockPath returns the path to the index lock file. The lock file is a
// dedicated sidecar so locking never races with reading, writing or replacing
// index.json itself.
func (m *Manager) indexLockPath() string {
	return filepath.Join(m.snapshotPath(), indexLockFile)
}

// withIndexLock runs fn while holding an exclusive, cross-process advisory lock
// on the index. Writing a snapshot file is made safe by O_EXCL, but updating
// index.json is a read-modify-write: without serialization two processes saving
// distinct IDs in the same tick can both read the same index and the later write
// drops the other's entry, so a saved snapshot vanishes from List (#648). The
// lock is an OS advisory lock on a sidecar file, released automatically when the
// process exits, so it cannot go stale.
func (m *Manager) withIndexLock(fn func() error) error {
	// Callers (Save after writing the snapshot file, Delete after removing it)
	// have already ensured the snapshot directory exists, so opening the lock
	// file here cannot be the first thing to create it.
	lock, err := os.OpenFile(m.indexLockPath(), os.O_RDWR|os.O_CREATE, fileMode0644)
	if err != nil {
		return fmt.Errorf("opening index lock: %w", err)
	}
	// Closing the file releases the advisory lock on every supported platform,
	// and the OS also drops it if the process dies, so no explicit unlock is
	// needed and the lock cannot go stale.
	defer func() { _ = lock.Close() }()
	if err := lockFileExclusive(lock); err != nil {
		return fmt.Errorf("locking index: %w", err)
	}
	return fn()
}

// snapshotFilePath returns the path to a specific snapshot file
func (m *Manager) snapshotFilePath(id string) string {
	return filepath.Join(m.snapshotPath(), id+".json")
}

// Save stores a snapshot to disk
func (m *Manager) Save(snapshot *Snapshot) (err error) {
	// Create snapshot directory if it doesn't exist
	snapPath := m.snapshotPath()
	if mkErr := os.MkdirAll(snapPath, snapshotDir0755); mkErr != nil {
		return fmt.Errorf("creating snapshot directory: %w", mkErr)
	}

	// Write snapshot file
	snapshotFile := m.snapshotFilePath(snapshot.ID)
	data, mErr := json.MarshalIndent(snapshot, "", "  ")
	if mErr != nil {
		return fmt.Errorf("marshaling snapshot: %w", mErr)
	}
	// Write the snapshot fail-closed: O_EXCL turns a colliding ID into an error
	// instead of a silent overwrite. The in-process counter keeps IDs unique
	// within one process, but two invocations in the same clock tick (e.g. two
	// CLI calls on a coarse-resolution clock) can still produce the same ID
	// across processes; refusing here protects the earlier snapshot (#648).
	f, oErr := os.OpenFile(snapshotFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fileMode0644)
	if oErr != nil {
		if errors.Is(oErr, os.ErrExist) {
			return fmt.Errorf("snapshot %q already exists: refusing to overwrite", snapshot.ID)
		}
		return fmt.Errorf("creating snapshot file: %w", oErr)
	}
	// The file now exists and belongs to this call. If anything below fails,
	// remove it so O_EXCL does not permanently block a retry on the same ID. A
	// duplicate rejected above returns before this point, so another writer's
	// file is never removed.
	defer func() {
		if err != nil {
			_ = os.Remove(snapshotFile)
		}
	}()

	_, writeErr := f.Write(data)
	if closeErr := f.Close(); writeErr == nil {
		writeErr = closeErr
	}
	if writeErr != nil {
		return fmt.Errorf("writing snapshot file: %w", writeErr)
	}

	// Update index
	if idxErr := m.updateIndex(snapshot); idxErr != nil {
		return fmt.Errorf("updating index: %w", idxErr)
	}

	return nil
}

// Load retrieves a snapshot from disk
func (m *Manager) Load(id string) (*Snapshot, error) {
	snapshotFile := m.snapshotFilePath(id)
	data, err := os.ReadFile(snapshotFile)
	if err != nil {
		return nil, fmt.Errorf("reading snapshot file: %w", err)
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("parsing snapshot: %w", err)
	}

	return &snapshot, nil
}

// List returns all saved snapshots sorted by timestamp (newest first)
func (m *Manager) List() ([]SnapshotMetadata, error) {
	indexPath := m.indexPath()
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []SnapshotMetadata{}, nil
		}
		return nil, fmt.Errorf("reading index: %w", err)
	}

	var index SnapshotIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("parsing index: %w", err)
	}

	return index.Snapshots, nil
}

// Delete removes a snapshot from disk
func (m *Manager) Delete(id string) error {
	snapshotFile := m.snapshotFilePath(id)
	if err := os.Remove(snapshotFile); err != nil {
		return fmt.Errorf("deleting snapshot file: %w", err)
	}

	// Update index to remove the snapshot
	if err := m.removeFromIndex(id); err != nil {
		return fmt.Errorf("updating index: %w", err)
	}

	return nil
}

// updateIndex adds or updates a snapshot in the index. The whole
// read-modify-write runs under the index lock so concurrent saves cannot drop
// one another's entries.
func (m *Manager) updateIndex(snapshot *Snapshot) error {
	return m.withIndexLock(func() error {
		indexPath := m.indexPath()
		var index SnapshotIndex

		// Read existing index if it exists
		if data, err := os.ReadFile(indexPath); err == nil {
			if err := json.Unmarshal(data, &index); err != nil {
				return err
			}
		}

		// Remove old entry if it exists
		index.Snapshots = removeMetadataByID(index.Snapshots, snapshot.ID)

		// Add new entry
		index.Snapshots = append(index.Snapshots, snapshot.ToMetadata())

		// Sort by timestamp (newest first)
		sort.Slice(index.Snapshots, func(i, j int) bool {
			return index.Snapshots[i].Timestamp.After(index.Snapshots[j].Timestamp)
		})

		index.Version = 1
		index.UpdatedAt = time.Now().UTC()

		// Write updated index
		data, err := json.MarshalIndent(index, "", "  ")
		if err != nil {
			return err
		}
		return writeFileAtomic(indexPath, data)
	})
}

// removeFromIndex removes a snapshot from the index. Like updateIndex, the
// read-modify-write runs under the index lock so a concurrent save or delete
// cannot clobber the result.
func (m *Manager) removeFromIndex(id string) error {
	return m.withIndexLock(func() error {
		indexPath := m.indexPath()
		data, err := os.ReadFile(indexPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		var index SnapshotIndex
		if err := json.Unmarshal(data, &index); err != nil {
			return err
		}

		index.Snapshots = removeMetadataByID(index.Snapshots, id)
		index.UpdatedAt = time.Now().UTC()

		updatedData, err := json.MarshalIndent(index, "", "  ")
		if err != nil {
			return err
		}

		return writeFileAtomic(indexPath, updatedData)
	})
}

// writeFileAtomic writes data to a temporary file in the same directory and
// renames it into place. The rename is the single commit point: a reader never
// sees a half-written index, and a failure before the rename leaves the existing
// file untouched. That matters for the index because Save removes the snapshot
// file when updateIndex fails — if a failed index write had already published a
// partial update, Save would delete a snapshot that List still references. The
// caller holds the index lock, so the fixed ".tmp" name cannot collide.
func writeFileAtomic(path string, data []byte) (err error) {
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileMode0644)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	// Remove the temp file unless it was successfully renamed away below.
	defer func() {
		if err != nil {
			_ = os.Remove(tmp)
		}
	}()
	if _, wErr := f.Write(data); wErr != nil {
		_ = f.Close()
		return fmt.Errorf("writing temp file: %w", wErr)
	}
	if cErr := f.Close(); cErr != nil {
		return fmt.Errorf("closing temp file: %w", cErr)
	}
	if rErr := os.Rename(tmp, path); rErr != nil {
		return fmt.Errorf("replacing file: %w", rErr)
	}
	return nil
}

// removeMetadataByID removes a metadata entry by ID
func removeMetadataByID(metadata []SnapshotMetadata, id string) []SnapshotMetadata {
	var result []SnapshotMetadata
	for _, m := range metadata {
		if m.ID != id {
			result = append(result, m)
		}
	}
	return result
}

// Exists checks if a snapshot exists
func (m *Manager) Exists(id string) bool {
	snapshotFile := m.snapshotFilePath(id)
	_, err := os.Stat(snapshotFile)
	return err == nil
}

// ListFiles returns all snapshot files in the directory
func (m *Manager) ListFiles() ([]string, error) {
	snapPath := m.snapshotPath()
	entries, err := os.ReadDir(snapPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" && entry.Name() != indexFile {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

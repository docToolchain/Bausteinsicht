package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	snapshotDir   = ".bausteinsicht-snapshots"
	indexFile     = "index.json"
	snapshotDir0755 = 0o755
	fileMode0644  = 0o644
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

// snapshotFilePath returns the path to a specific snapshot file
func (m *Manager) snapshotFilePath(id string) string {
	return filepath.Join(m.snapshotPath(), id+".json")
}

// Save stores a snapshot to disk
func (m *Manager) Save(snapshot *Snapshot) error {
	// Create snapshot directory if it doesn't exist
	snapPath := m.snapshotPath()
	if err := os.MkdirAll(snapPath, snapshotDir0755); err != nil {
		return fmt.Errorf("creating snapshot directory: %w", err)
	}

	// Write snapshot file
	snapshotFile := m.snapshotFilePath(snapshot.ID)
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling snapshot: %w", err)
	}
	if err := os.WriteFile(snapshotFile, data, fileMode0644); err != nil {
		return fmt.Errorf("writing snapshot file: %w", err)
	}

	// Update index
	if err := m.updateIndex(snapshot); err != nil {
		return fmt.Errorf("updating index: %w", err)
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

// updateIndex adds or updates a snapshot in the index
func (m *Manager) updateIndex(snapshot *Snapshot) error {
	snapPath := m.snapshotPath()
	if err := os.MkdirAll(snapPath, snapshotDir0755); err != nil {
		return err
	}

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
	return os.WriteFile(indexPath, data, fileMode0644)
}

// removeFromIndex removes a snapshot from the index
func (m *Manager) removeFromIndex(id string) error {
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

	return os.WriteFile(indexPath, updatedData, fileMode0644)
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

package snapshot

import (
	"fmt"
	"time"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// Snapshot represents a point-in-time capture of the architecture model
type Snapshot struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Message   string                 `json:"message,omitempty"`
	Model     *model.BausteinsichtModel `json:"model"`
}

// SnapshotMetadata is lightweight metadata for snapshots in the index
type SnapshotMetadata struct {
	ID           string    `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	Message      string    `json:"message,omitempty"`
	ElementCount int       `json:"elementCount"`
	RelCount     int       `json:"relationshipCount"`
}

// SnapshotIndex holds the list of all snapshots
type SnapshotIndex struct {
	Version   int                  `json:"version"`
	Snapshots []SnapshotMetadata   `json:"snapshots"`
	UpdatedAt time.Time            `json:"updatedAt"`
}

// NewSnapshot creates a snapshot from a model
func NewSnapshot(message string, model *model.BausteinsichtModel) *Snapshot {
	snapshot := &Snapshot{
		ID:        generateSnapshotID(),
		Timestamp: time.Now().UTC(),
		Message:   message,
		Model:     model,
	}
	return snapshot
}

// generateSnapshotID creates a timestamp-based snapshot ID with nanosecond precision
func generateSnapshotID() string {
	now := time.Now().UTC()
	return now.Format("snapshot-2006-01-02T15-04-05") + fmt.Sprintf(".%09dZ", now.Nanosecond())
}

// ToMetadata converts a snapshot to its metadata representation
func (s *Snapshot) ToMetadata() SnapshotMetadata {
	elementCount := 0
	if s.Model != nil {
		elementCount = len(flattenElements(s.Model.Model))
	}

	relationCount := 0
	if s.Model != nil {
		relationCount = len(s.Model.Relationships)
	}

	return SnapshotMetadata{
		ID:           s.ID,
		Timestamp:    s.Timestamp,
		Message:      s.Message,
		ElementCount: elementCount,
		RelCount:     relationCount,
	}
}

// flattenElements counts total elements including nested ones
func flattenElements(elems map[string]model.Element) map[string]model.Element {
	result := make(map[string]model.Element)
	for key, elem := range elems {
		result[key] = elem
		if len(elem.Children) > 0 {
			children := flattenElements(elem.Children)
			for k, v := range children {
				result[key+"."+k] = v
			}
		}
	}
	return result
}

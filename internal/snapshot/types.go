package snapshot

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// idCounter is a process-global, monotonically increasing counter appended as a
// suffix to each snapshot ID. It guarantees in-process uniqueness even when the
// wall clock does not advance between two generations (same clock tick under
// concurrency, or a coarse-resolution clock such as on Windows).
var idCounter atomic.Uint64

// Snapshot represents a point-in-time capture of the architecture model
type Snapshot struct {
	ID        string                    `json:"id"`
	Timestamp time.Time                 `json:"timestamp"`
	Message   string                    `json:"message,omitempty"`
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
	Version   int                `json:"version"`
	Snapshots []SnapshotMetadata `json:"snapshots"`
	UpdatedAt time.Time          `json:"updatedAt"`
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

// generateSnapshotID creates a snapshot ID that is guaranteed unique even when
// the wall clock does not advance between calls. The human-readable,
// time-sortable timestamp (down to nanoseconds) stays as the prefix so that
// listings and format expectations are preserved; a monotonic process-global
// counter is appended as a suffix to break ties within the same clock tick.
func generateSnapshotID() string {
	return snapshotID(time.Now().UTC(), idCounter.Add(1))
}

// snapshotID formats a snapshot ID from a timestamp and a monotonic sequence
// number. The sequence is what guarantees uniqueness within a single clock
// tick; the timestamp only makes the ID human-readable and time-sortable. It
// is split out from generateSnapshotID so a test can pin the clock and prove
// the sequence — not the clock's resolution — breaks ties.
func snapshotID(now time.Time, seq uint64) string {
	return now.Format("snapshot-2006-01-02T15-04-05") +
		fmt.Sprintf(".%09dZ-%06d", now.Nanosecond(), seq)
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

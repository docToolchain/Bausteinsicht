package diff

import "github.com/docToolchain/Bausteinsicht/internal/model"

// ChangeType describes how an element or relationship changed
type ChangeType string

const (
	ChangeAdded   ChangeType = "added"
	ChangeRemoved ChangeType = "removed"
	ChangeChanged ChangeType = "changed"
)

// ElementChange represents a single element change
type ElementChange struct {
	ID       string                 `json:"id"`
	Type     ChangeType             `json:"type"`
	AsIs     *model.Element         `json:"asIs,omitempty"`
	ToBe     *model.Element         `json:"toBe,omitempty"`
	Reason   string                 `json:"reason,omitempty"`
}

// RelationshipChange represents a single relationship change
type RelationshipChange struct {
	From     string         `json:"from"`
	To       string         `json:"to"`
	Type     ChangeType     `json:"type"`
	AsIs     *model.Relationship `json:"asIs,omitempty"`
	ToBe     *model.Relationship `json:"toBe,omitempty"`
}

// DiffResult contains all changes between two architecture snapshots
type DiffResult struct {
	Elements      []ElementChange       `json:"elements"`
	Relationships []RelationshipChange  `json:"relationships"`
	Summary       Summary              `json:"summary"`
}

// Summary counts changes by type
type Summary struct {
	AddedElements      int `json:"addedElements"`
	RemovedElements    int `json:"removedElements"`
	ChangedElements    int `json:"changedElements"`
	AddedRelationships int `json:"addedRelationships"`
	RemovedRelationships int `json:"removedRelationships"`
	TotalAddedElements   int `json:"totalAddedElements"`
	TotalRemovedElements int `json:"totalRemovedElements"`
}

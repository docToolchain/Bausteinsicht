package changelog

import (
	"time"

	"github.com/docToolchain/Bausteinsicht/internal/diff"
)

// Reference represents a git ref or snapshot identifier
type Reference struct {
	Ref  string    `json:"ref"`  // git tag/commit SHA or snapshot ID
	Date time.Time `json:"date"` // date of the reference
}

// Changelog describes changes between two architecture snapshots
type Changelog struct {
	From          Reference           `json:"from"`
	To            Reference           `json:"to"`
	Elements      ElementChanges      `json:"elements"`
	Relationships RelationshipChanges `json:"relationships"`
}

// ElementChanges groups element changes by type
type ElementChanges struct {
	Added   []diff.ElementChange `json:"added"`
	Removed []diff.ElementChange `json:"removed"`
	Changed []diff.ElementChange `json:"changed"`
}

// RelationshipChanges groups relationship changes by type
type RelationshipChanges struct {
	Added   []diff.RelationshipChange `json:"added"`
	Removed []diff.RelationshipChange `json:"removed"`
}

// CommitInfo retrieves commit metadata for a git ref
type CommitInfo struct {
	Hash      string    `json:"hash"`
	Author    string    `json:"author"`
	Date      time.Time `json:"date"`
	Message   string    `json:"message"`
	Timestamp int64     `json:"timestamp"`
}

// FilterChangesByKind returns only changes of a specific element kind
func (ec ElementChanges) FilterByKind(kind string) ElementChanges {
	return ElementChanges{
		Added:   filterElementsByKind(ec.Added, kind),
		Removed: filterElementsByKind(ec.Removed, kind),
		Changed: filterElementsByKind(ec.Changed, kind),
	}
}

func filterElementsByKind(changes []diff.ElementChange, kind string) []diff.ElementChange {
	var result []diff.ElementChange
	for _, c := range changes {
		var k string
		if c.ToBe != nil {
			k = c.ToBe.Kind
		} else if c.AsIs != nil {
			k = c.AsIs.Kind
		}
		if k == kind {
			result = append(result, c)
		}
	}
	return result
}

// CountAdded returns the number of added elements
func (ec ElementChanges) CountAdded() int {
	return len(ec.Added)
}

// CountRemoved returns the number of removed elements
func (ec ElementChanges) CountRemoved() int {
	return len(ec.Removed)
}

// CountChanged returns the number of changed elements
func (ec ElementChanges) CountChanged() int {
	return len(ec.Changed)
}

// CountAddedRelationships returns the number of added relationships
func (rc RelationshipChanges) CountAddedRelationships() int {
	return len(rc.Added)
}

// CountRemovedRelationships returns the number of removed relationships
func (rc RelationshipChanges) CountRemovedRelationships() int {
	return len(rc.Removed)
}

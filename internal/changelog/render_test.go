package changelog

import (
	"strings"
	"testing"
	"time"

	"github.com/docToolchain/Bausteinsicht/internal/diff"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func TestRenderMarkdown_NoChanges(t *testing.T) {
	cl := &Changelog{
		From: Reference{Ref: "v1.0"},
		To:   Reference{Ref: "v2.0"},
		Elements: ElementChanges{
			Added:   []diff.ElementChange{},
			Removed: []diff.ElementChange{},
			Changed: []diff.ElementChange{},
		},
		Relationships: RelationshipChanges{
			Added:   []diff.RelationshipChange{},
			Removed: []diff.RelationshipChange{},
		},
	}

	result := RenderMarkdown(cl)
	if !strings.Contains(result, "No architectural changes") {
		t.Error("Expected 'No architectural changes' message")
	}
}

func TestRenderMarkdown_WithAdded(t *testing.T) {
	cl := &Changelog{
		From: Reference{Ref: "v1.0", Date: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		To:   Reference{Ref: "v2.0", Date: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)},
		Elements: ElementChanges{
			Added: []diff.ElementChange{
				{
					ID:   "api",
					Type: diff.ChangeAdded,
					ToBe: &model.Element{
						Kind:  "container",
						Title: "API Server",
					},
				},
			},
			Removed: []diff.ElementChange{},
			Changed: []diff.ElementChange{},
		},
		Relationships: RelationshipChanges{
			Added:   []diff.RelationshipChange{},
			Removed: []diff.RelationshipChange{},
		},
	}

	result := RenderMarkdown(cl)
	if !strings.Contains(result, "Added (1 elements)") {
		t.Error("Expected 'Added (1 elements)' section")
	}
	if !strings.Contains(result, "**api**") {
		t.Error("Expected element 'api' in output")
	}
	if !strings.Contains(result, "[container]") {
		t.Error("Expected kind '[container]' in output")
	}
}

func TestRenderAsciiDoc_WithRemoved(t *testing.T) {
	cl := &Changelog{
		From: Reference{Ref: "v1.0"},
		To:   Reference{Ref: "v2.0"},
		Elements: ElementChanges{
			Added: []diff.ElementChange{},
			Removed: []diff.ElementChange{
				{
					ID:   "legacy",
					Type: diff.ChangeRemoved,
					AsIs: &model.Element{
						Kind:  "system",
						Title: "Legacy System",
					},
				},
			},
			Changed: []diff.ElementChange{},
		},
		Relationships: RelationshipChanges{
			Added:   []diff.RelationshipChange{},
			Removed: []diff.RelationshipChange{},
		},
	}

	result := RenderAsciiDoc(cl)
	if !strings.Contains(result, "Removed (1 elements)") {
		t.Error("Expected 'Removed (1 elements)' section")
	}
	if !strings.Contains(result, "line-through") {
		t.Error("Expected strikethrough formatting")
	}
}

func TestRenderJSON_Valid(t *testing.T) {
	cl := &Changelog{
		From: Reference{Ref: "v1.0"},
		To:   Reference{Ref: "v2.0"},
		Elements: ElementChanges{
			Added:   []diff.ElementChange{},
			Removed: []diff.ElementChange{},
			Changed: []diff.ElementChange{},
		},
		Relationships: RelationshipChanges{
			Added:   []diff.RelationshipChange{},
			Removed: []diff.RelationshipChange{},
		},
	}

	result, err := RenderJSON(cl)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !strings.Contains(result, "\"from\"") {
		t.Error("Expected 'from' in JSON output")
	}
	if !strings.Contains(result, "\"to\"") {
		t.Error("Expected 'to' in JSON output")
	}
}

func TestRenderMarkdown_WithChangedRelationships(t *testing.T) {
	rel := model.Relationship{
		From:  "system",
		To:    "database",
		Label: "reads/writes",
	}

	cl := &Changelog{
		From: Reference{Ref: "v1.0"},
		To:   Reference{Ref: "v2.0"},
		Elements: ElementChanges{
			Added:   []diff.ElementChange{},
			Removed: []diff.ElementChange{},
			Changed: []diff.ElementChange{},
		},
		Relationships: RelationshipChanges{
			Added: []diff.RelationshipChange{
				{
					From: "system",
					To:   "database",
					Type: diff.ChangeAdded,
					ToBe: &rel,
				},
			},
			Removed: []diff.RelationshipChange{},
		},
	}

	result := RenderMarkdown(cl)
	if !strings.Contains(result, "New Relationships (1)") {
		t.Error("Expected 'New Relationships (1)' section")
	}
	if !strings.Contains(result, "system → database") {
		t.Error("Expected relationship 'system → database' in output")
	}
}

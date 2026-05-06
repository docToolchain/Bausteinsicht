package diff

import (
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func TestGetChangeColors_Added(t *testing.T) {
	fill, stroke := GetChangeColors(ChangeAdded)
	if fill != ColorAdded || stroke != StrokeAdded {
		t.Errorf("Expected green colors for added, got fill=%s stroke=%s", fill, stroke)
	}
}

func TestGetChangeColors_Removed(t *testing.T) {
	fill, stroke := GetChangeColors(ChangeRemoved)
	if fill != ColorRemoved || stroke != StrokeRemoved {
		t.Errorf("Expected red colors for removed, got fill=%s stroke=%s", fill, stroke)
	}
}

func TestGetChangeColors_Changed(t *testing.T) {
	fill, stroke := GetChangeColors(ChangeChanged)
	if fill != ColorChanged || stroke != StrokeChanged {
		t.Errorf("Expected orange colors for changed, got fill=%s stroke=%s", fill, stroke)
	}
}

func TestGetElementStyle_AddedElement(t *testing.T) {
	change := ElementChange{
		ID:   "newservice",
		Type: ChangeAdded,
		ToBe: &model.Element{
			Title: "New Service",
			Kind:  "service",
		},
	}

	style := GetElementStyle(change)

	if style.FillColor != ColorAdded {
		t.Errorf("Expected fill color %s, got %s", ColorAdded, style.FillColor)
	}
	if style.StrokeColor != StrokeAdded {
		t.Errorf("Expected stroke color %s, got %s", StrokeAdded, style.StrokeColor)
	}
	if style.StrokeWidth != 2 {
		t.Errorf("Expected stroke width 2, got %v", style.StrokeWidth)
	}
	if style.Opacity != 1.0 {
		t.Errorf("Expected opacity 1.0, got %v", style.Opacity)
	}
}

func TestGetElementStyle_RemovedElement(t *testing.T) {
	change := ElementChange{
		ID:   "oldservice",
		Type: ChangeRemoved,
		AsIs: &model.Element{
			Title: "Old Service",
			Kind:  "service",
		},
	}

	style := GetElementStyle(change)

	if style.FillColor != ColorRemoved {
		t.Errorf("Expected fill color %s, got %s", ColorRemoved, style.FillColor)
	}
	if style.Label != "~Old Service" {
		t.Errorf("Expected label '~Old Service', got %s", style.Label)
	}
}

func TestGetElementStyle_ChangedElement(t *testing.T) {
	change := ElementChange{
		ID:   "api",
		Type: ChangeChanged,
		AsIs: &model.Element{
			Title: "API v1",
		},
		ToBe: &model.Element{
			Title: "API v2",
		},
	}

	style := GetElementStyle(change)

	if style.FillColor != ColorChanged {
		t.Errorf("Expected fill color %s, got %s", ColorChanged, style.FillColor)
	}
	if style.StrokeColor != StrokeChanged {
		t.Errorf("Expected stroke color %s, got %s", StrokeChanged, style.StrokeColor)
	}
}

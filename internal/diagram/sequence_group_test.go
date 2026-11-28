package diagram

import (
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func groupView() (model.DynamicView, map[string]*model.Element) {
	flat := map[string]*model.Element{
		"u": {Kind: "actor", Title: "U"},
		"s": {Kind: "system", Title: "S"},
	}
	return model.DynamicView{Key: "f", Title: "F", Steps: []model.SequenceStep{
		{Index: 1, Type: model.StepGroup, Label: "Checkout; phase 1"},
		{Index: 2, From: "u", To: "s", Label: "pay"},
		{Index: 3, From: "s", To: "u"},
		{Index: 4, Type: model.StepEnd},
	}}, flat
}

// Mermaid has no titled group: the grey rectangle carries the label as a note
// over the participants, so that it is not lost (#711).
func TestSequenceMermaid_GroupKeepsItsLabel(t *testing.T) {
	view, flat := groupView()
	out := RenderSequenceMermaid(view, flat, &model.Specification{})
	if !strings.Contains(out, "rect rgb(240, 240, 240)\n      Note over u,s: Checkout#59; phase 1\n") {
		t.Errorf("want the label as a note right inside the rect, got:\n%s", out)
	}
}

// A group in a diagram with one participant uses "Note over x".
func TestSequenceMermaid_GroupLabelWithOneParticipant(t *testing.T) {
	view := model.DynamicView{Key: "f", Title: "F", Steps: []model.SequenceStep{
		{Index: 1, Type: model.StepGroup, Label: "g"},
		{Index: 2, From: "u", To: "u", Label: "self"},
		{Index: 3, Type: model.StepEnd},
	}}
	out := RenderSequenceMermaid(view, map[string]*model.Element{"u": {Title: "U"}}, &model.Specification{})
	if !strings.Contains(out, "Note over u: g\n") {
		t.Errorf("want \"Note over u: g\", got:\n%s", out)
	}
}

// A group without a label has no note.
func TestSequenceMermaid_GroupWithoutLabelHasNoNote(t *testing.T) {
	view, flat := groupView()
	view.Steps[0].Label = ""
	if out := RenderSequenceMermaid(view, flat, &model.Specification{}); strings.Contains(out, "Note over") {
		t.Errorf("no label, no note:\n%s", out)
	}
}

// A message without a label has no trailing ": " (#711).
func TestSequence_MessageWithoutLabelHasNoTrailingSeparator(t *testing.T) {
	view, flat := groupView()
	mermaid := RenderSequenceMermaid(view, flat, &model.Specification{})
	if !strings.Contains(mermaid, "s->>u:\n") && !strings.Contains(mermaid, "s->>u\n") {
		t.Errorf("Mermaid: want no trailing space after the colon, got:\n%s", mermaid)
	}
	if strings.Contains(mermaid, ": \n") {
		t.Errorf("Mermaid: trailing \": \" left in:\n%s", mermaid)
	}
	puml := RenderSequencePlantUML(view, flat, &model.Specification{})
	if strings.Contains(puml, " : \n") || !strings.Contains(puml, "s -> u\n") {
		t.Errorf("PlantUML: a message without a label is \"s -> u\", got:\n%s", puml)
	}
}

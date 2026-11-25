package model

import (
	"strings"
	"testing"
)

func blockTestModel(steps ...SequenceStep) *BausteinsichtModel {
	return &BausteinsichtModel{
		Specification: Specification{Elements: map[string]ElementKind{"service": {Notation: "component"}}},
		Model: map[string]Element{
			"a": {Kind: "service", Title: "A"},
			"b": {Kind: "service", Title: "B"},
		},
		DynamicViews: []DynamicView{{Key: "v", Title: "V", Steps: steps}},
	}
}

func msg(index int) SequenceStep {
	return SequenceStep{Index: index, From: "a", To: "b", Label: "call"}
}

// Block steps (alt, else, opt, group, end) frame the steps between them and
// need no from/to (#637).
func TestValidateDynamicViews_BlockSteps(t *testing.T) {
	tests := []struct {
		name    string
		steps   []SequenceStep
		wantErr string // substring of the expected error, "" for valid
	}{
		{"alt else end", []SequenceStep{
			{Index: 1, Type: StepAlt, Label: "ok"}, msg(2),
			{Index: 3, Type: StepElse, Label: "failed"}, msg(4),
			{Index: 5, Type: StepEnd},
		}, ""},
		{"opt and group, nested", []SequenceStep{
			{Index: 1, Type: StepGroup, Label: "outer"},
			{Index: 2, Type: StepOpt, Label: "inner"}, msg(3),
			{Index: 4, Type: StepEnd}, {Index: 5, Type: StepEnd},
		}, ""},
		{"blocks are ordered by index, not by position in the list", []SequenceStep{
			{Index: 3, Type: StepEnd}, msg(2), {Index: 1, Type: StepOpt, Label: "x"},
		}, ""},
		{"block left open", []SequenceStep{
			{Index: 1, Type: StepAlt, Label: "ok"}, msg(2),
		}, `"alt" opened at step index 1 is never closed`},
		{"end without a block", []SequenceStep{
			msg(1), {Index: 2, Type: StepEnd},
		}, `"end" without an open block`},
		{"else outside an alt", []SequenceStep{
			{Index: 1, Type: StepOpt, Label: "x"}, {Index: 2, Type: StepElse, Label: "y"}, {Index: 3, Type: StepEnd},
		}, `"else" is only allowed directly inside an "alt"`},
		{"message step still needs from and to", []SequenceStep{
			{Index: 1, From: "a", Label: "call"},
		}, `to "" does not resolve`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(blockTestModel(tt.steps...))
			if tt.wantErr == "" {
				if len(errs) != 0 {
					t.Fatalf("expected a valid model, got %v", errs)
				}
				return
			}
			for _, e := range errs {
				if strings.Contains(e.Message, tt.wantErr) {
					return
				}
			}
			t.Fatalf("expected an error containing %q, got %v", tt.wantErr, errs)
		})
	}
}

func TestDynamicViewStepTypes_BlockTypesAreValid(t *testing.T) {
	for _, st := range []StepType{StepAlt, StepElse, StepOpt, StepGroup, StepEnd} {
		if !validStepTypes[st] {
			t.Errorf("step type %q is not accepted", st)
		}
	}
}

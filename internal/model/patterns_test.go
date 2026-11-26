package model

import "testing"

func TestExpandPatternTopLevelIDs(t *testing.T) {
	p := PatternDefinition{Elements: []PatternElement{
		{ID: "{base}-api", Children: []PatternElement{{ID: "cache"}, {ID: "{base}-deep"}}},
		{ID: "{BASE}-worker"},
	}}
	got := ExpandPatternTopLevelIDs(p, "pay")
	if len(got) != 2 || got[0] != "pay-api" || got[1] != "PAY-worker" {
		t.Errorf("top-level IDs = %v, want [pay-api PAY-worker] (children are not listed)", got)
	}
	// The same order as the elements ExpandPattern returns.
	els, _, _ := ExpandPattern(p, "pay", "Pay")
	if len(els) != len(got) {
		t.Errorf("%d elements but %d IDs", len(els), len(got))
	}
}

func TestCheckPatternConflicts_OnlyTopLevelIDsCount(t *testing.T) {
	p := PatternDefinition{Elements: []PatternElement{{ID: "{base}", Children: []PatternElement{{ID: "db"}}}}}
	m := &BausteinsichtModel{Model: map[string]Element{
		"db":  {Title: "unrelated top-level element with a child's ID"},
		"pay": {Title: "real conflict", Children: map[string]Element{"x": {}}},
	}}

	if got, _ := CheckPatternConflicts(m, p, "pay"); len(got) != 1 || got[0] != "pay" {
		t.Errorf("conflicts = %v, want [pay]", got)
	}
	if got, _ := CheckPatternConflicts(m, p, "shop"); len(got) != 0 {
		t.Errorf("conflicts = %v, want none: the child db lives below shop", got)
	}
}

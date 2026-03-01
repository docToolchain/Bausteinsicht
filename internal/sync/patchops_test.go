package sync

import (
	"testing"

	"github.com/docToolchain/Bauteinsicht/internal/model"
)

func TestReversePatchOps_ElementModified(t *testing.T) {
	cs := &ChangeSet{
		DrawioElementChanges: []ElementChange{
			{ID: "webshop.api", Type: Modified, Field: "technology", NewValue: "Go 1.24"},
		},
	}
	ops, ok := ReversePatchOps(cs)
	if !ok {
		t.Fatal("expected patchable")
	}
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	// Path should be: model → webshop → children → api → technology
	wantPath := []string{"model", "webshop", "children", "api", "technology"}
	if !equalPath(ops[0].Path, wantPath) {
		t.Errorf("path = %v, want %v", ops[0].Path, wantPath)
	}
	if ops[0].Value != `"Go 1.24"` {
		t.Errorf("value = %s, want %q", ops[0].Value, "Go 1.24")
	}
}

func TestReversePatchOps_TopLevelElementModified(t *testing.T) {
	cs := &ChangeSet{
		DrawioElementChanges: []ElementChange{
			{ID: "customer", Type: Modified, Field: "title", NewValue: "Customer Portal"},
		},
	}
	ops, ok := ReversePatchOps(cs)
	if !ok {
		t.Fatal("expected patchable")
	}
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	wantPath := []string{"model", "customer", "title"}
	if !equalPath(ops[0].Path, wantPath) {
		t.Errorf("path = %v, want %v", ops[0].Path, wantPath)
	}
}

func TestReversePatchOps_RelationshipModifiedNotPatchable(t *testing.T) {
	cs := &ChangeSet{
		DrawioRelationshipChanges: []RelationshipChange{
			{From: "a", To: "b", Type: Modified, Field: "label", NewValue: "connects to"},
		},
	}
	_, ok := ReversePatchOps(cs)
	if ok {
		t.Error("relationship modifications should not be patchable (array entries)")
	}
}

func TestReversePatchOps_StructuralChangeNotPatchable(t *testing.T) {
	cs := &ChangeSet{
		DrawioElementChanges: []ElementChange{
			{ID: "customer", Type: Deleted},
		},
	}
	_, ok := ReversePatchOps(cs)
	if ok {
		t.Error("structural changes should not be patchable")
	}
}

func TestReversePatchOps_AddedNotPatchable(t *testing.T) {
	cs := &ChangeSet{
		DrawioRelationshipChanges: []RelationshipChange{
			{From: "a", To: "b", Type: Added, NewValue: "uses"},
		},
	}
	_, ok := ReversePatchOps(cs)
	if ok {
		t.Error("added relationships should not be patchable")
	}
}

func equalPath(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Ensure model import is used.
var _ = model.PatchOp{}

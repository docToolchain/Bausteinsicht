package sync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// scopeExternalModel returns a model in which the actor "user" outside the
// system "shop" has a relationship with the scope element of a view that also
// shows the top-level elements (include "*"), the case of #736.
func scopeExternalModel(rel model.Relationship) *model.BausteinsichtModel {
	m := shopModel(map[string]model.Element{
		"web": {Kind: "container", Title: "Web"},
	})
	m.Model["user"] = model.Element{Kind: "actor", Title: "User"}
	m.Views["containers"] = model.View{Title: "Detail", Scope: "shop", Include: []string{"*"}}
	m.Relationships = []model.Relationship{rel}
	return m
}

// TestComputeVisibleRelationships_ScopeToExternalIsNotVisible (#736): the
// forward sync draws no connector between the scope boundary and an element
// outside it (isScopeExternalConnector), so such a relationship must not count
// as visible; otherwise reverse sync reads the missing connector as a user
// deletion and removes the relationship from the model.
func TestComputeVisibleRelationships_ScopeToExternalIsNotVisible(t *testing.T) {
	for name, rel := range map[string]model.Relationship{
		"external to scope": {From: "user", To: "shop", Label: "orders"},
		"scope to external": {From: "shop", To: "user", Label: "notifies"},
	} {
		t.Run(name, func(t *testing.T) {
			m := scopeExternalModel(rel)
			if visible := computeVisibleRelationships(m); visible[relKey(rel.From, rel.To, 0)] {
				t.Fatal("a relationship between the scope element and an element outside it has no connector on the page, so it must not count as visible")
			}
		})
	}
}

// TestRun_RelationshipToScopeElementSurvivesTheSecondSync (#736) runs two sync
// cycles (the second one with the state of the first) and expects the model to
// be unchanged.
func TestRun_RelationshipToScopeElementSurvivesTheSecondSync(t *testing.T) {
	for name, rel := range map[string]model.Relationship{
		"external to scope": {From: "user", To: "shop", Label: "orders", Kind: "uses"},
		"scope to external": {From: "shop", To: "user", Label: "notifies", Kind: "uses"},
	} {
		t.Run(name, func(t *testing.T) {
			m := scopeExternalModel(rel)
			doc := shopContainersDoc()
			ts := minimalTemplates(t)

			Run(m, doc, emptyState(), ts, nil)
			dir := t.TempDir()
			modelPath, drawioPath := filepath.Join(dir, "m.jsonc"), filepath.Join(dir, "a.drawio")
			for _, p := range []string{modelPath, drawioPath} {
				if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			state, err := BuildState(m, doc, modelPath, drawioPath)
			if err != nil {
				t.Fatal(err)
			}

			result := Run(m, doc, state, ts, nil)

			if len(m.Relationships) != 1 || m.Relationships[0].From != rel.From || m.Relationships[0].To != rel.To {
				t.Fatalf("the second sync changed the model's relationships to %+v (reverse result: %+v)", m.Relationships, result.Reverse)
			}
			if result.Reverse != nil && result.Reverse.RelationshipsDeleted != 0 {
				t.Errorf("the second sync deleted %d relationship(s) from the model", result.Reverse.RelationshipsDeleted)
			}
		})
	}
}

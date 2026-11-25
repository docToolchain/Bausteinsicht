package model

import (
	"strings"
	"testing"
)

// editTestModel is the example from #634: a nested element that is referenced
// from a relationship, a view (scope and include), a dynamic view and a
// constraint.
func editTestModel() *BausteinsichtModel {
	return &BausteinsichtModel{
		Specification: Specification{
			Elements: map[string]ElementKind{
				"system":    {Notation: "System", Container: true},
				"container": {Notation: "Container"},
			},
		},
		Model: map[string]Element{
			"shop": {Kind: "system", Title: "Shop", Children: map[string]Element{
				"web": {Kind: "container", Title: "Web App"},
				"api": {Kind: "container", Title: "API"},
			}},
			"payment": {Kind: "system", Title: "Payment"},
		},
		Relationships: []Relationship{
			{From: "shop.web", To: "payment", Label: "charges"},
			{From: "shop.web", To: "shop.api", Label: "calls"},
			{From: "payment", To: "shop", Label: "notifies"},
		},
		Views: map[string]View{
			"shop":    {Title: "Shop", Scope: "shop", Include: []string{"shop.*"}},
			"context": {Title: "Context", Include: []string{"shop", "payment"}, Exclude: []string{"shop.api"}},
		},
		DynamicViews: []DynamicView{{Key: "checkout", Title: "Checkout", Steps: []SequenceStep{
			{Index: 1, From: "shop.web", To: "payment", Label: "charge"},
			{Index: 2, Type: StepAlt, Label: "ok"},
			{Index: 3, From: "payment", To: "shop.web", Label: "receipt", Type: StepReturn},
			{Index: 4, Type: StepEnd},
		}}},
		Constraints: []Constraint{{ID: "c1", Rule: "no-relationship", From: "shop.web", To: "payment"}},
	}
}

func changePaths(changes []Change) string {
	paths := make([]string, len(changes))
	for i, c := range changes {
		paths[i] = c.Path
	}
	return strings.Join(paths, ",")
}

func TestRenameElement_RewritesEveryReference(t *testing.T) {
	m := editTestModel()
	changes, err := m.RenameElement("shop", "store")
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := m.Model["shop"]; ok {
		t.Error("old key still present")
	}
	if got := m.Model["store"].Children["web"].Title; got != "Web App" {
		t.Errorf("children did not move with the element, got %q", got)
	}
	if m.Relationships[0].From != "store.web" || m.Relationships[1].To != "store.api" || m.Relationships[2].To != "store" {
		t.Errorf("relationships not rewritten: %+v", m.Relationships)
	}
	if v := m.Views["shop"]; v.Scope != "store" || v.Include[0] != "store.*" {
		t.Errorf("view scope/include not rewritten: %+v", v)
	}
	if v := m.Views["context"]; v.Include[0] != "store" || v.Exclude[0] != "store.api" {
		t.Errorf("view include/exclude not rewritten: %+v", v)
	}
	steps := m.DynamicViews[0].Steps
	if steps[0].From != "store.web" || steps[2].To != "store.web" {
		t.Errorf("dynamic view steps not rewritten: %+v", steps)
	}
	if c := m.Constraints[0]; c.From != "store.web" || c.To != "payment" {
		t.Errorf("constraint not rewritten: %+v", c)
	}
	if errs := Validate(m); len(errs) != 0 {
		t.Errorf("model invalid after rename: %v", errs)
	}
	if len(changes) < 9 {
		t.Errorf("expected every rewritten location to be listed, got %d: %s", len(changes), changePaths(changes))
	}
}

func TestRenameElement_NestedAndErrors(t *testing.T) {
	m := editTestModel()
	if _, err := m.RenameElement("shop.web", "frontend"); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Model["shop"].Children["frontend"]; !ok {
		t.Error("nested element not renamed")
	}
	if m.Relationships[0].From != "shop.frontend" {
		t.Errorf("relationship not rewritten: %+v", m.Relationships[0])
	}

	cases := []struct {
		name, id, to, want string
	}{
		{"unknown element", "nope", "x", "not found"},
		{"target exists", "shop", "payment", "already exists"},
		{"dots in the new key", "shop", "a.b", "must not contain"},
		{"empty new key", "shop", "", "must not be empty"},
		{"same key", "shop", "shop", "same"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := editTestModel().RenameElement(tc.id, tc.to)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want it to contain %q", err, tc.want)
			}
		})
	}
}

func TestRemoveElement(t *testing.T) {
	t.Run("referenced element needs --cascade", func(t *testing.T) {
		m := editTestModel()
		m.Constraints = nil
		_, err := m.RemoveElement("shop.web", false)
		if err == nil || !strings.Contains(err.Error(), "relationships[0]") {
			t.Fatalf("expected an error listing the references, got %v", err)
		}
		if _, ok := m.Model["shop"].Children["web"]; !ok {
			t.Error("element removed although the call failed")
		}
	})

	t.Run("cascade drops dangling relationships and steps", func(t *testing.T) {
		m := editTestModel()
		m.Constraints = nil
		changes, err := m.RemoveElement("shop.web", true)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := m.Model["shop"].Children["web"]; ok {
			t.Error("element still present")
		}
		if len(m.Relationships) != 1 || m.Relationships[0].To != "shop" {
			t.Errorf("relationships after cascade: %+v", m.Relationships)
		}
		// Every message step of the only dynamic view used the element, so the
		// view would be left with nothing but block steps and goes away too.
		if len(m.DynamicViews) != 0 {
			t.Errorf("dynamic view without message steps should be removed: %+v", m.DynamicViews)
		}
		if !strings.Contains(changePaths(changes), "dynamicViews[0]") {
			t.Errorf("removed dynamic view not listed: %s", changePaths(changes))
		}
	})

	t.Run("removing a parent removes its children and their references", func(t *testing.T) {
		m := editTestModel()
		m.Views["shop"] = View{Title: "Shop", Include: []string{"shop.*"}}
		m.Constraints = nil
		if _, err := m.RemoveElement("shop", true); err != nil {
			t.Fatal(err)
		}
		if _, ok := m.Model["shop"]; ok {
			t.Error("parent still present")
		}
		if len(m.Relationships) != 0 {
			t.Errorf("relationships should all be gone: %+v", m.Relationships)
		}
		if len(m.Views["shop"].Include) != 0 || len(m.Views["context"].Include) != 1 {
			t.Errorf("view includes not cleaned: %+v", m.Views)
		}
	})

	t.Run("a view scope and a constraint always block", func(t *testing.T) {
		if _, err := editTestModel().RemoveElement("shop", true); err == nil || !strings.Contains(err.Error(), "scope") {
			t.Errorf("expected the view scope to block the removal, got %v", err)
		}
		m := editTestModel()
		m.Views["shop"] = View{Title: "Shop", Include: []string{"shop.*"}}
		if _, err := m.RemoveElement("shop", true); err == nil || !strings.Contains(err.Error(), "constraints[0]") {
			t.Errorf("expected the constraint to block the removal, got %v", err)
		}
	})

	t.Run("unreferenced element is simply removed", func(t *testing.T) {
		m := editTestModel()
		m.Model["lonely"] = Element{Kind: "system", Title: "Lonely"}
		if _, err := m.RemoveElement("lonely", false); err != nil {
			t.Fatal(err)
		}
		if _, ok := m.Model["lonely"]; ok {
			t.Error("not removed")
		}
	})

	t.Run("unknown element", func(t *testing.T) {
		if _, err := editTestModel().RemoveElement("nope", false); err == nil {
			t.Error("expected an error")
		}
	})
}

func TestRemoveRelationship(t *testing.T) {
	m := editTestModel()
	if _, err := m.RemoveRelationship("shop.web", "payment", "charges"); err != nil {
		t.Fatal(err)
	}
	if len(m.Relationships) != 2 {
		t.Errorf("got %d relationships, want 2", len(m.Relationships))
	}
	if _, err := m.RemoveRelationship("shop.web", "payment", "charges"); err == nil {
		t.Error("expected an error for a relationship that is gone")
	}
	// Without a label the first match is removed.
	if _, err := m.RemoveRelationship("payment", "shop", ""); err != nil {
		t.Errorf("label is optional: %v", err)
	}
}

func TestUpdateElement(t *testing.T) {
	m := editTestModel()
	title, tech := "Storefront", "Go"
	changes, err := m.UpdateElement("shop.web", ElementUpdate{Title: &title, Technology: &tech})
	if err != nil {
		t.Fatal(err)
	}
	web := m.Model["shop"].Children["web"]
	if web.Title != "Storefront" || web.Technology != "Go" || web.Kind != "container" {
		t.Errorf("unexpected element after update: %+v", web)
	}
	if len(changes) != 2 {
		t.Errorf("want 2 changes, got %s", changePaths(changes))
	}

	badKind := "database"
	if _, err := m.UpdateElement("shop.web", ElementUpdate{Kind: &badKind}); err == nil || !strings.Contains(err.Error(), "unknown element kind") {
		t.Errorf("expected an unknown kind error, got %v", err)
	}
	if _, err := m.UpdateElement("nope", ElementUpdate{Title: &title}); err == nil {
		t.Error("expected an error for an unknown element")
	}
	if _, err := m.UpdateElement("shop.web", ElementUpdate{}); err == nil || !strings.Contains(err.Error(), "nothing to update") {
		t.Errorf("expected an error for an empty update, got %v", err)
	}
	same := "Storefront"
	if ch, err := m.UpdateElement("shop.web", ElementUpdate{Title: &same}); err != nil || len(ch) != 0 {
		t.Errorf("setting the same value is not a change: %v, %d", err, len(ch))
	}
}

func TestRemoveElement_CascadeKeepsViewsWithOtherSteps(t *testing.T) {
	m := editTestModel()
	m.Constraints = nil
	m.DynamicViews[0].Steps = append(m.DynamicViews[0].Steps,
		SequenceStep{Index: 5, From: "payment", To: "shop.api", Label: "other"})
	if _, err := m.RemoveElement("shop.web", true); err != nil {
		t.Fatal(err)
	}
	if len(m.DynamicViews) != 1 {
		t.Fatalf("the dynamic view still has a message step and must stay: %+v", m.DynamicViews)
	}
	for _, s := range m.DynamicViews[0].Steps {
		if s.From == "shop.web" || s.To == "shop.web" {
			t.Errorf("step still references the element: %+v", s)
		}
	}
}

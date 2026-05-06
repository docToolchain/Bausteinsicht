package likec4_test

import (
	"path/filepath"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/importer/likec4"
)

func TestImport_Simple(t *testing.T) {
	result, err := likec4.Import(filepath.Join("testdata", "simple.c4"))
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	m := result.Model

	// Specification kinds
	for _, kind := range []string{"person", "system", "container", "component"} {
		if _, ok := m.Specification.Elements[kind]; !ok {
			t.Errorf("specification missing kind %q", kind)
		}
	}

	// Container kinds should have container=true (they have children)
	if !m.Specification.Elements["system"].Container {
		t.Error("expected kind 'system' to be marked as container")
	}

	// Root elements
	if _, ok := m.Model["user"]; !ok {
		t.Error("expected element 'user'")
	}
	if _, ok := m.Model["myPlatform"]; !ok {
		t.Error("expected element 'myPlatform'")
	}

	// Nested children
	platform := m.Model["myPlatform"]
	if len(platform.Children) != 3 {
		t.Errorf("expected 3 children in myPlatform, got %d", len(platform.Children))
	}
	frontend, ok := platform.Children["frontend"]
	if !ok {
		t.Fatal("expected child 'frontend' in myPlatform")
	}
	if frontend.Technology != "TypeScript" {
		t.Errorf("expected frontend.Technology=TypeScript, got %q", frontend.Technology)
	}
	if len(frontend.Children) != 1 {
		t.Errorf("expected 1 component in frontend, got %d", len(frontend.Children))
	}

	// Relationships
	if len(m.Relationships) == 0 {
		t.Error("expected relationships, got none")
	}
	found := false
	for _, r := range m.Relationships {
		if r.From == "user" && r.To == "myPlatform.frontend" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected relationship user → myPlatform.frontend, got: %+v", m.Relationships)
	}

	// Views
	if len(m.Views) != 2 {
		t.Errorf("expected 2 views, got %d", len(m.Views))
	}
	indexView, ok := m.Views["index"]
	if !ok {
		t.Error("expected view 'index'")
	}
	if indexView.Title != "System Context" {
		t.Errorf("expected view title 'System Context', got %q", indexView.Title)
	}
	if indexView.Scope != "myPlatform" {
		t.Errorf("expected view scope 'myPlatform', got %q", indexView.Scope)
	}
}

func TestImport_Source_Basic(t *testing.T) {
	const src = `
specification {
  element actor
  element service
}
model {
  a = actor "Alice"
  s = service "Auth" {
    description "Handles authentication"
    technology "Go"
  }
  a -> s "Logs in"
}
views {
  view main {
    title "Main View"
    include *
  }
}
`
	result, err := likec4.ImportSource(src)
	if err != nil {
		t.Fatalf("ImportSource failed: %v", err)
	}

	m := result.Model

	if _, ok := m.Specification.Elements["actor"]; !ok {
		t.Error("expected kind 'actor' in spec")
	}

	alice, ok := m.Model["a"]
	if !ok {
		t.Fatal("expected element 'a'")
	}
	if alice.Kind != "actor" {
		t.Errorf("expected kind 'actor', got %q", alice.Kind)
	}

	auth := m.Model["s"]
	if auth.Description != "Handles authentication" {
		t.Errorf("unexpected description: %q", auth.Description)
	}
	if auth.Technology != "Go" {
		t.Errorf("unexpected technology: %q", auth.Technology)
	}

	if len(m.Relationships) != 1 {
		t.Errorf("expected 1 relationship, got %d", len(m.Relationships))
	}
	if m.Relationships[0].Label != "Logs in" {
		t.Errorf("unexpected label: %q", m.Relationships[0].Label)
	}
}

func TestImport_ImplicitFrom(t *testing.T) {
	const src = `
specification {
  element person
  element system
}
model {
  u = person "User" {
    -> sys "Uses"
  }
  sys = system "System"
}
`
	result, err := likec4.ImportSource(src)
	if err != nil {
		t.Fatalf("ImportSource failed: %v", err)
	}
	if len(result.Model.Relationships) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(result.Model.Relationships))
	}
	r := result.Model.Relationships[0]
	if r.From != "u" || r.To != "sys" {
		t.Errorf("unexpected relationship: %+v", r)
	}
}

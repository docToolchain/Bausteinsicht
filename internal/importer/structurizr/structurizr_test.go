package structurizr_test

import (
	"path/filepath"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/importer/structurizr"
)

func TestImport_Simple(t *testing.T) {
	result, err := structurizr.Import(filepath.Join("testdata", "simple.dsl"))
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	m := result.Model

	// Specification should contain all 4 C4 kinds in order
	for _, kind := range []string{"person", "system", "container"} {
		if _, ok := m.Specification.Elements[kind]; !ok {
			t.Errorf("specification missing kind %q", kind)
		}
	}

	// Root-level elements
	if _, ok := m.Model["user"]; !ok {
		t.Error("expected element 'user'")
	}
	if _, ok := m.Model["orderSystem"]; !ok {
		t.Error("expected element 'orderSystem'")
	}
	if _, ok := m.Model["externalPayment"]; !ok {
		t.Error("expected element 'externalPayment'")
	}

	// Nested children of orderSystem
	orderSystem := m.Model["orderSystem"]
	if len(orderSystem.Children) != 3 {
		t.Errorf("expected 3 children in orderSystem, got %d", len(orderSystem.Children))
	}
	if _, ok := orderSystem.Children["webApp"]; !ok {
		t.Error("expected child 'webApp' in orderSystem")
	}

	// Relationships
	if len(m.Relationships) == 0 {
		t.Error("expected relationships, got none")
	}

	// Check path resolution: user -> webApp should resolve to orderSystem.webApp
	found := false
	for _, r := range m.Relationships {
		if r.From == "user" && r.To == "orderSystem.webApp" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected relationship user → orderSystem.webApp, got: %+v", m.Relationships)
	}

	// Views
	if len(m.Views) == 0 {
		t.Error("expected views, got none")
	}
}

func TestImport_Nested(t *testing.T) {
	result, err := structurizr.Import(filepath.Join("testdata", "nested.dsl"))
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	m := result.Model

	if _, ok := m.Model["customer"]; !ok {
		t.Error("expected element 'customer'")
	}
	mySystem := m.Model["mySystem"]
	if len(mySystem.Children) != 2 {
		t.Errorf("expected 2 children in mySystem, got %d", len(mySystem.Children))
	}

	// component nested inside frontend
	frontend, ok := mySystem.Children["frontend"]
	if !ok {
		t.Fatal("expected child 'frontend' in mySystem")
	}
	if len(frontend.Children) != 1 {
		t.Errorf("expected 1 component in frontend, got %d", len(frontend.Children))
	}

	// Implicit relationship from customer -> frontend (inline)
	found := false
	for _, r := range m.Relationships {
		if r.From == "customer" && r.To == "mySystem.frontend" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected implicit relationship customer → mySystem.frontend, got: %+v", m.Relationships)
	}
}

func TestImport_Tokenizer_Strings(t *testing.T) {
	const src = `workspace {
  model {
    s = softwareSystem "System with \"quotes\"" "Desc with\nnewline"
  }
}`
	result, err := structurizr.ImportSource(src)
	if err != nil {
		t.Fatalf("ImportSource failed: %v", err)
	}
	el, ok := result.Model.Model["s"]
	if !ok {
		t.Fatal("expected element 's'")
	}
	if el.Title != `System with "quotes"` {
		t.Errorf("unexpected title: %q", el.Title)
	}
}

func TestImport_NoViews(t *testing.T) {
	const src = `workspace {
  model {
    a = person "Alice"
    b = softwareSystem "System"
    a -> b "Uses"
  }
}`
	result, err := structurizr.ImportSource(src)
	if err != nil {
		t.Fatalf("ImportSource failed: %v", err)
	}
	if result.Model.Views == nil {
		t.Error("views map should not be nil")
	}
	if len(result.Model.Relationships) != 1 {
		t.Errorf("expected 1 relationship, got %d", len(result.Model.Relationships))
	}
}

func TestImport_UnknownViewType_Warning(t *testing.T) {
	const src = `workspace {
  model {
    a = softwareSystem "A"
  }
  views {
    dynamic a "Seq" { }
  }
}`
	result, err := structurizr.ImportSource(src)
	if err != nil {
		t.Fatalf("ImportSource failed: %v", err)
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warning for unsupported view type 'dynamic'")
	}
}

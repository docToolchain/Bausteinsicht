package structurizr_test

import (
	"os"
	"path/filepath"
	"strings"
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

func TestImport_ScopedView_IncludeExpansion(t *testing.T) {
	// Verify that "include *" on a container/component view gets expanded to
	// include scope descendants, matching Structurizr semantics.
	const src = `workspace {
  model {
    user = person "User"
    system = softwareSystem "System" {
      app = container "App" {
        ctrl = component "Controller"
      }
      db = container "DB"
    }
    external = softwareSystem "External"
  }
  views {
    systemContext system "Context" { include * }
    container system "Containers" { include * }
    component app "Components" { include * }
  }
}`
	result, err := structurizr.ImportSource(src)
	if err != nil {
		t.Fatalf("ImportSource failed: %v", err)
	}
	m := result.Model

	// systemContext with scope: keep plain "*" (top-level elements are correct)
	ctxView, ok := m.Views["system"]
	if !ok {
		t.Fatal("expected view key 'system'")
	}
	if len(ctxView.Include) != 1 || ctxView.Include[0] != "*" {
		t.Errorf("systemContext view: expected include=[\"*\"], got %v", ctxView.Include)
	}

	// container view: must add "system.*" so containers appear inside boundary
	containersView, ok := m.Views["system_1"]
	if !ok {
		t.Fatal("expected view key 'system_1'")
	}
	wantContainers := []string{"*", "system.*"}
	if len(containersView.Include) != len(wantContainers) {
		t.Errorf("container view: expected include=%v, got %v", wantContainers, containersView.Include)
	} else {
		for i, p := range wantContainers {
			if containersView.Include[i] != p {
				t.Errorf("container view include[%d]: want %q, got %q", i, p, containersView.Include[i])
			}
		}
	}

	// component view: must add parent containers + scope components
	// scope resolves to "system.app"
	componentsView, ok := m.Views["system.app"]
	if !ok {
		t.Fatal("expected view key 'system.app'")
	}
	wantComponents := []string{"*", "system.*", "system.app.*"}
	if len(componentsView.Include) != len(wantComponents) {
		t.Errorf("component view: expected include=%v, got %v", wantComponents, componentsView.Include)
	} else {
		for i, p := range wantComponents {
			if componentsView.Include[i] != p {
				t.Errorf("component view include[%d]: want %q, got %q", i, p, componentsView.Include[i])
			}
		}
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

func TestImport_AutoLayout_MapsToLayeredAndWarns(t *testing.T) {
	// autoLayout has no direction-aware Bausteinsicht equivalent; the importer
	// must map it to a valid layout value ("layered", not "auto" — which
	// model.Validate rejects) and warn that the direction argument is dropped.
	const src = `workspace {
  model {
    a = softwareSystem "A"
  }
  views {
    systemContext a "Context" {
      include *
      autoLayout lr
    }
  }
}`
	result, err := structurizr.ImportSource(src)
	if err != nil {
		t.Fatalf("ImportSource failed: %v", err)
	}
	m := result.Model

	view, ok := m.Views["a"]
	if !ok {
		t.Fatal("expected view key 'a'")
	}
	if view.Layout != "layered" {
		t.Errorf("expected layout \"layered\", got %q", view.Layout)
	}

	var found bool
	for _, w := range result.Warnings {
		if strings.Contains(w, "autoLayout direction") && strings.Contains(w, `"lr"`) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a warning about the dropped autoLayout direction, got warnings: %v", result.Warnings)
	}
}

func TestImport_AutoLayout_NoDirection_NoWarning(t *testing.T) {
	// autoLayout with no direction argument has nothing to drop, so no warning
	// should be emitted — only the layout mapping itself.
	const src = `workspace {
  model {
    a = softwareSystem "A"
  }
  views {
    systemContext a "Context" {
      include *
      autoLayout
    }
  }
}`
	result, err := structurizr.ImportSource(src)
	if err != nil {
		t.Fatalf("ImportSource failed: %v", err)
	}
	for _, w := range result.Warnings {
		if strings.Contains(w, "autoLayout direction") {
			t.Errorf("expected no autoLayout-direction warning without a direction argument, got: %v", result.Warnings)
		}
	}
}

func TestImport_PathTraversalRejected(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a sensitive file outside of tmpDir (parent directory)
	parentDir := filepath.Dir(tmpDir)
	sensitiveFile := filepath.Join(parentDir, "sensitive.txt")
	if err := os.WriteFile(sensitiveFile, []byte("SECRET"), 0600); err != nil {
		t.Fatalf("failed to create sensitive file: %v", err)
	}
	defer func() { _ = os.Remove(sensitiveFile) }()

	// Create main DSL file with path traversal attempt
	mainFile := filepath.Join(tmpDir, "main.dsl")
	mainContent := `workspace {
  model {
    a = person "Alice"
  }
}
!include ../sensitive.txt
`
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("failed to create main DSL file: %v", err)
	}

	result, err := structurizr.Import(mainFile)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	foundWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "path traversal rejected") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Errorf("expected path traversal rejection warning, got warnings: %v", result.Warnings)
	}
}

func TestImport_PathTraversalDeepEscape(t *testing.T) {
	tmpDir := t.TempDir()

	// Create main DSL file with aggressive path traversal attempt
	mainFile := filepath.Join(tmpDir, "main.dsl")
	mainContent := `workspace {
  model {
    a = person "Alice"
  }
}
!include ../../../../../../../../etc/passwd
`
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("failed to create main DSL file: %v", err)
	}

	result, err := structurizr.Import(mainFile)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	foundWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "path traversal rejected") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Errorf("expected path traversal rejection warning for deep escape, got warnings: %v", result.Warnings)
	}
}

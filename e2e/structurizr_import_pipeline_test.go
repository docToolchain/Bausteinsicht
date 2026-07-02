package e2e

// TestStructurizrImportPipeline (#512) reproduces the reported "import →
// sync/render produces empty diagrams" failure chain end-to-end, using a
// minimal Structurizr DSL that mirrors the two root causes from the real
// (islandr) workspace that triggered the report:
//
//  1. `autoLayout` on a view — the importer used to write layout: "auto",
//     which sync's own model validation rejects (only "layered"/"grid"/
//     "none"/"" are valid). A unit test on the importer alone would not have
//     caught this, because the bug only manifests when the model produced by
//     `import` is fed into `sync`.
//  2. A scoped container/component view with `include *` — the importer used
//     to leave Include as a plain ["*"], which only matches top-level
//     (dot-free) element IDs, so container/component children never resolved
//     into the view and every exported diagram was empty.
//
// A second sub-test (EmptyViewWarns) verifies the accompanying fix: a view
// that legitimately resolves to zero elements makes export-diagram warn on
// stderr instead of silently writing an empty file.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const structurizrAutoLayoutDSL = `workspace "Pipeline" "minimal repro for #512" {
    model {
        user = person "User" "A user."

        widgets = softwareSystem "Widgets" "Does widget things." {
            api = container "API" "REST API." "Go"
            db  = container "Database" "Stores widgets." "Postgres"

            handlers = container "Handlers" "Request handlers." "Go" {
                createHandler = component "CreateHandler" "Creates a widget." "Go"
            }
        }

        user -> api "Uses" "HTTPS"
        api  -> db  "Reads/writes" "SQL"
    }

    views {
        systemContext widgets "SystemContext" {
            include *
            autoLayout lr
        }

        container widgets "Containers" {
            include *
            autoLayout lr
        }

        component handlers "Components" {
            include *
            autoLayout lr
        }
    }
}
`

// TestStructurizrImportPipeline_AutoLayoutAndScopedViews runs the full
// import -> sync -> export-diagram pipeline against a DSL using autoLayout
// and scoped container/component views, and asserts every stage succeeds
// and the exported diagrams actually contain the imported elements.
func TestStructurizrImportPipeline_AutoLayoutAndScopedViews(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	dslPath := filepath.Join(dir, "workspace.dsl")
	if err := os.WriteFile(dslPath, []byte(structurizrAutoLayoutDSL), 0o600); err != nil {
		t.Fatalf("write workspace.dsl: %v", err)
	}

	// ── Step 1: import ─────────────────────────────────────────────────────────
	runCLI(t, bin, dir, "import", "workspace.dsl", "--from", "structurizr", "--output", "architecture.jsonc")

	modelPath := filepath.Join(dir, "architecture.jsonc")
	modelBytes, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("read imported model: %v", err)
	}
	modelText := string(modelBytes)
	if strings.Contains(modelText, `"layout": "auto"`) {
		t.Fatalf("imported model still writes invalid layout \"auto\":\n%s", modelText)
	}

	// ── Step 2: sync — this is where the "auto" layout used to be rejected ─────
	syncOut, code := runCLIAllowFail(t, bin, dir, "sync", "--model", "architecture.jsonc")
	if code != 0 {
		t.Fatalf("sync failed after structurizr import (exit %d):\n%s", code, syncOut)
	}
	if strings.Contains(syncOut, "invalid layout") {
		t.Fatalf("sync rejected the imported layout value:\n%s", syncOut)
	}

	// The importer keys views by scope, suffixing on collision: the
	// systemContext view (scope "widgets") gets the plain key, the container
	// view (same scope, second view) becomes "widgets_1", and the component
	// view (scope "widgets.handlers") keeps its dotted scope as the key.
	const (
		systemContextView = "widgets"
		containersView    = "widgets_1"
		componentsView    = "widgets.handlers"
	)

	// ── Step 3: export-diagram — every scoped view must resolve non-empty ──────
	for _, view := range []string{systemContextView, containersView, componentsView} {
		out := runCLI(t, bin, dir, "export-diagram", "--model", "architecture.jsonc",
			"--diagram-format", "plantuml", "--view", view)
		if strings.Contains(out, "WARNING") && strings.Contains(out, "resolves to 0 elements") {
			t.Errorf("view %q unexpectedly resolved to 0 elements:\n%s", view, out)
		}
	}

	// The container view must include its container children (widgets.api etc.),
	// not just the top-level "widgets" system — this is the scope-expansion fix.
	containersOut := runCLI(t, bin, dir, "export-diagram", "--model", "architecture.jsonc",
		"--diagram-format", "plantuml", "--view", containersView)
	for _, want := range []string{"API", "Database", "Handlers"} {
		if !strings.Contains(containersOut, want) {
			t.Errorf("container view PlantUML output missing %q — scope/include expansion regressed:\n%s", want, containersOut)
		}
	}

	// The component view must include its component children.
	componentsOut := runCLI(t, bin, dir, "export-diagram", "--model", "architecture.jsonc",
		"--diagram-format", "plantuml", "--view", componentsView)
	if !strings.Contains(componentsOut, "CreateHandler") {
		t.Errorf("component view PlantUML output missing \"CreateHandler\" — scope/include expansion regressed:\n%s", componentsOut)
	}
}

// TestStructurizrImportPipeline_EmptyViewWarns verifies that export-diagram
// warns on stderr when a view legitimately resolves to zero elements,
// instead of silently succeeding with an empty diagram (#512, observation 3).
func TestStructurizrImportPipeline_EmptyViewWarns(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	runCLI(t, bin, dir, "init")

	// "customer" is a leaf actor in the default init model (no children), so
	// scoping a view to its wildcard children resolves to zero elements — a
	// legitimate (if unusual) empty view, distinct from the #512 import bug
	// (which produced empty views for elements that DID have children).
	runCLI(t, bin, dir, "add", "view", "empty-view",
		"--title", "Empty View",
		"--scope", "customer",
		"--include", "customer.*",
	)

	out := runCLI(t, bin, dir, "export-diagram", "--model", "architecture.jsonc",
		"--diagram-format", "plantuml", "--view", "empty-view")

	if !strings.Contains(out, "WARNING") || !strings.Contains(out, `"empty-view"`) || !strings.Contains(out, "0 elements") {
		t.Errorf("expected a 0-elements warning for view %q, got:\n%s", "empty-view", out)
	}
}

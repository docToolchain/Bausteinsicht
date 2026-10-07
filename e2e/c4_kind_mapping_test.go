package e2e

// TestC4KindMapping* (#633): export-diagram picks the C4 macro from the kind key,
// then the notation, or from an explicit "c4" override, and warns for kinds that
// would silently render as System().

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const c4KindModelTemplate = `{
  "specification": {
    "elements": {
      "person": { "notation": "Person" },
      "system": { "notation": "Software System" },
      "widget": { "notation": "Widget"%s }
    },
    "relationships": { "uses": { "notation": "uses" } }
  },
  "model": {
    "customer": { "kind": "person", "title": "Customer" },
    "shop":     { "kind": "system", "title": "Shop" },
    "gizmo":    { "kind": "widget", "title": "Gizmo" }
  },
  "relationships": [ { "from": "customer", "to": "shop", "label": "places order", "kind": "uses" } ],
  "views": { "context": { "title": "Context", "include": ["*"] } }
}`

func writeC4KindModel(t *testing.T, widgetExtra string) (dir, modelPath string) {
	t.Helper()
	dir = t.TempDir()
	modelPath = filepath.Join(dir, "architecture.jsonc")
	content := strings.Replace(c4KindModelTemplate, "%s", widgetExtra, 1)
	if err := os.WriteFile(modelPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, modelPath
}

func TestC4KindMapping_PersonAndFallbackWarning(t *testing.T) {
	bin := buildBinary(t)
	dir, modelPath := writeC4KindModel(t, "")

	for _, format := range []string{"plantuml", "mermaid"} {
		stdout, stderr, code := runCLISplit(t, bin, dir,
			"export-diagram", "--model", modelPath, "--view", "context", "--diagram-format", format)
		if code != 0 {
			t.Fatalf("%s: exit %d: %s", format, code, stderr)
		}
		if !strings.Contains(stdout, "Person(customer,") {
			t.Errorf("%s: expected Person(customer, ...), got:\n%s", format, stdout)
		}
		if !strings.Contains(stderr, `element kind "widget" is not a recognised C4 kind`) {
			t.Errorf("%s: expected fallback warning for widget, got stderr:\n%s", format, stderr)
		}
	}
}

func TestC4KindMapping_OverrideRemovesWarning(t *testing.T) {
	bin := buildBinary(t)
	dir, modelPath := writeC4KindModel(t, `, "c4": "ContainerDb"`)

	stdout, stderr, code := runCLISplit(t, bin, dir,
		"export-diagram", "--model", modelPath, "--view", "context")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "ContainerDb(gizmo,") {
		t.Errorf("expected ContainerDb(gizmo, ...), got:\n%s", stdout)
	}
	if strings.Contains(stderr, "not a recognised C4 kind") {
		t.Errorf("c4 override must silence the warning, got stderr:\n%s", stderr)
	}
}

func TestC4KindMapping_InvalidOverrideRejectedByValidate(t *testing.T) {
	bin := buildBinary(t)
	dir, modelPath := writeC4KindModel(t, `, "c4": "Persn"`)

	stdout, stderr, code := runCLISplit(t, bin, dir, "validate", "--model", modelPath)
	if code == 0 {
		t.Fatalf("expected validate to fail for unknown c4 macro, got exit 0\nstdout: %s", stdout)
	}
	if !strings.Contains(stdout+stderr, "c4") {
		t.Errorf("expected error mentioning c4, got:\n%s%s", stdout, stderr)
	}
}

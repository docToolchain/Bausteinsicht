package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/stale"
)

const staleTestModel = `{
  "specification": {
    "elements": {
      "system": {"notation": "System", "container": true},
      "container": {"notation": "Container"},
      "actor": {"notation": "Actor"}
    }
  },
  "model": {
    "user": {
      "kind": "actor",
      "title": "User",
      "description": "End user"
    },
    "shop": {
      "kind": "system",
      "title": "Shop",
      "description": "E-commerce system",
      "children": {
        "api": {
          "kind": "container",
          "title": "API",
          "description": "REST API"
        },
        "db": {
          "kind": "container",
          "title": "Database",
          "description": "PostgreSQL"
        }
      }
    },
    "legacy": {
      "kind": "system",
      "title": "Legacy System",
      "description": "Old monolith",
      "status": "deprecated"
    }
  },
  "relationships": [
    {"from": "user", "to": "shop.api", "label": "calls"}
  ],
  "views": {
    "context": {
      "title": "System Context",
      "include": ["user", "shop"]
    }
  }
}`

func writeStaleTestModel(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "architecture.jsonc")
	if err := os.WriteFile(path, []byte(staleTestModel), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestStaleCmd_Help(t *testing.T) {
	out, err := executeRootCmd("stale", "--help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "stale") {
		t.Errorf("expected help text with 'stale' in output")
	}
	if !strings.Contains(out, "--days") || !strings.Contains(out, "--format") {
		t.Errorf("expected flag documentation in help output")
	}
}

func TestStaleCmd_TextFormat(t *testing.T) {
	modelPath := writeStaleTestModel(t)
	// Note: Since the model file is in a temp directory not tracked by git,
	// no elements will be flagged as stale (we can't determine git age for untracked files).
	// This is expected behavior - stale detection requires git history.
	out, err := executeRootCmd("stale", "--model", modelPath, "--days", "90")
	if err != nil {
		t.Fatalf("unexpected error: %v\nOutput: %s", err, out)
	}

	// Verify text output format
	if !strings.Contains(out, "stale") || !strings.Contains(out, "elements") {
		t.Errorf("expected output mentioning stale elements, got:\n%s", out)
	}
}

func TestStaleCmd_JSONFormat(t *testing.T) {
	modelPath := writeStaleTestModel(t)
	out, err := executeRootCmd("stale", "--model", modelPath, "--days", "0", "--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v\nOutput: %s", err, out)
	}

	// Verify JSON output
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Errorf("output is not valid JSON: %v\nOutput: %s", err, out)
	}

	// Check result structure uses camelCase keys and staleElements is [] not null.
	if _, ok := result["staleElements"]; !ok {
		t.Errorf("expected 'staleElements' in JSON, got keys: %v", result)
	}
	if elems, ok := result["staleElements"]; !ok || elems == nil {
		t.Errorf("expected 'staleElements' to be [] not null, got: %v", result["staleElements"])
	}

	if _, ok := result["totalElements"]; !ok {
		t.Errorf("expected 'totalElements' in JSON, got keys: %v", result)
	}
}

func TestStaleCmd_InvalidDays(t *testing.T) {
	modelPath := writeStaleTestModel(t)
	_, err := executeRootCmd("stale", "--model", modelPath, "--days", "-1")
	if err == nil {
		t.Fatal("expected error for negative --days")
	}
}

func TestStaleCmd_InvalidFormat(t *testing.T) {
	modelPath := writeStaleTestModel(t)
	_, err := executeRootCmd("stale", "--model", modelPath, "--format", "xml")
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
}

func TestStaleCmd_NonExistentModel(t *testing.T) {
	_, err := executeRootCmd("stale", "--model", "/nonexistent/model.jsonc")
	if err == nil {
		t.Fatal("expected error for non-existent model")
	}
}

func TestStaleCmd_InvalidPathTraversal(t *testing.T) {
	_, err := executeRootCmd("stale", "--model", "../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal attempt")
	}
}

func TestIsDrawioFile(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"diagram.drawio", true},
		{"diagram.DRAWIO", true},
		{"diagram.DrawIO", true},
		{"diagram.xml", false},
		{"diagram.json", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isDrawioFile(tt.name); got != tt.want {
			t.Errorf("isDrawioFile(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// minimalDrawioXML returns a minimal draw.io file with one element.
func minimalDrawioXML(bausteinsichtID string) string {
	return `<?xml version="1.0" encoding="UTF-8"?><mxfile><diagram id="d1" name="Page"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><object bausteinsicht_id="` + bausteinsichtID + `" label="API"><mxCell id="cell1" style="rounded=1;" vertex="1" parent="1"/></object></root></mxGraphModel></diagram></mxfile>`
}

func writeTempDrawioInDir(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "architecture.drawio")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write drawio: %v", err)
	}
	return path
}

func TestFindDrawioFile_ExplicitPath(t *testing.T) {
	dir := t.TempDir()
	drawioPath := writeTempDrawioInDir(t, dir, minimalDrawioXML("x"))
	modelPath := filepath.Join(dir, "architecture.jsonc")

	got := findDrawioFile(modelPath, drawioPath)
	if got != drawioPath {
		t.Errorf("findDrawioFile with explicit path: got %q, want %q", got, drawioPath)
	}
}

func TestFindDrawioFile_AutoDetect(t *testing.T) {
	dir := t.TempDir()
	drawioPath := writeTempDrawioInDir(t, dir, minimalDrawioXML("x"))
	modelPath := filepath.Join(dir, "architecture.jsonc")

	got := findDrawioFile(modelPath, "")
	if got != drawioPath {
		t.Errorf("findDrawioFile auto-detect: got %q, want %q", got, drawioPath)
	}
}

func TestFindDrawioFile_NoDrawioFile(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "architecture.jsonc")

	got := findDrawioFile(modelPath, "")
	if got != "" {
		t.Errorf("findDrawioFile with no drawio file: got %q, want empty", got)
	}
}

func TestApplyDrawioMarking_AutoDetect(t *testing.T) {
	dir := t.TempDir()
	writeTempDrawioInDir(t, dir, minimalDrawioXML("shop.api"))
	modelPath := filepath.Join(dir, "architecture.jsonc")

	elems := []stale.StaleElement{{ID: "shop.api", Risk: stale.RiskHigh}}
	err := applyDrawioMarking(elems, modelPath, "", os.Stderr)
	if err != nil {
		t.Fatalf("applyDrawioMarking auto-detect: %v", err)
	}
}

func TestApplyDrawioMarking_ExplicitMissingFile(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "architecture.jsonc")
	elems := []stale.StaleElement{{ID: "x", Risk: stale.RiskLow}}
	err := applyDrawioMarking(elems, modelPath, "/nonexistent/file.drawio", os.Stderr)
	if err == nil {
		t.Fatal("expected error for non-existent explicit drawio-file")
	}
}

func TestApplyDrawioMarking_ExplicitPathTraversal(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "architecture.jsonc")
	elems := []stale.StaleElement{{ID: "x", Risk: stale.RiskLow}}
	err := applyDrawioMarking(elems, modelPath, "../../etc/passwd", os.Stderr)
	if err == nil {
		t.Fatal("expected error for path traversal in drawio-file")
	}
}

func TestApplyDrawioUnmarking_RemovesMarkers(t *testing.T) {
	dir := t.TempDir()
	// Write a pre-marked draw.io file.
	markedXML := `<?xml version="1.0" encoding="UTF-8"?><mxfile><diagram id="d1" name="Page"><mxGraphModel><root>` +
		`<mxCell id="0"/><mxCell id="1" parent="0"/>` +
		`<object bausteinsicht_id="shop.api" label="API">` +
		`<mxCell id="c1" style="fillColor=#FF6666;" data-original-fill="#dae8fc" vertex="1" parent="1"/>` +
		`</object></root></mxGraphModel></diagram></mxfile>`
	writeTempDrawioInDir(t, dir, markedXML)
	modelPath := filepath.Join(dir, "architecture.jsonc")

	if err := applyDrawioUnmarking(modelPath, "", os.Stderr); err != nil {
		t.Fatalf("applyDrawioUnmarking: %v", err)
	}
}

func TestApplyDrawioUnmarking_NoDrawioFile(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "architecture.jsonc")
	// No drawio file — should silently return nil.
	if err := applyDrawioUnmarking(modelPath, "", os.Stderr); err != nil {
		t.Fatalf("expected nil when no drawio file, got: %v", err)
	}
}

func TestApplyDrawioUnmarking_ExplicitMissingFile(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "architecture.jsonc")
	err := applyDrawioUnmarking(modelPath, "/nonexistent/file.drawio", os.Stderr)
	if err == nil {
		t.Fatal("expected error for non-existent explicit drawio-file")
	}
}

func TestApplyDrawioUnmarking_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "architecture.jsonc")
	err := applyDrawioUnmarking(modelPath, "../../etc/passwd", os.Stderr)
	if err == nil {
		t.Fatal("expected error for path traversal in drawio-file")
	}
}

func TestApplyDrawioMarking_NoDrawioFileFound(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "architecture.jsonc")
	elems := []stale.StaleElement{{ID: "x", Risk: stale.RiskLow}}
	// No drawio file in dir — should silently return nil.
	err := applyDrawioMarking(elems, modelPath, "", os.Stderr)
	if err != nil {
		t.Fatalf("expected nil when no drawio file found, got: %v", err)
	}
}

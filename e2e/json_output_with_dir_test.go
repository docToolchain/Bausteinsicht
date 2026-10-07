package e2e

// TestExport*JSONWithOutputDir (#631): --output <dir> combined with --format json
// must write files to disk AND report absolute paths in the JSON response.
// Previously the commands ignored --output and printed source to stdout only.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportDiagramJSONWithOutputDir(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	outDir := filepath.Join(dir, "out")

	stdout, _, code := runCLISplit(t, bin, dir,
		"export-diagram", "--diagram-format", "plantuml",
		"--output", outDir, "--format", "json",
	)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stdout)
	}

	var entries []map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &entries); err != nil {
		t.Fatalf("invalid JSON: %v\noutput:\n%s", err, stdout)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one entry in JSON output")
	}

	for _, e := range entries {
		if _, hasSource := e["source"]; hasSource {
			t.Error("expected no 'source' field when --output is set; got it anyway")
		}
		p, ok := e["path"].(string)
		if !ok || p == "" {
			t.Errorf("expected non-empty 'path' field, got: %v", e)
			continue
		}
		if !filepath.IsAbs(p) {
			t.Errorf("expected absolute path in JSON, got relative: %q", p)
		}
		if _, err := os.ReadFile(p); err != nil {
			t.Errorf("path %q reported in JSON but file does not exist: %v", p, err)
		}
	}

	// Without --output the old "source" behaviour must still work.
	stdoutSrc, _, code2 := runCLISplit(t, bin, dir,
		"export-diagram", "--diagram-format", "plantuml", "--format", "json",
	)
	if code2 != 0 {
		t.Fatalf("source mode exit %d: %s", code2, stdoutSrc)
	}
	var srcEntries []map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdoutSrc)), &srcEntries); err != nil {
		t.Fatalf("invalid JSON (source mode): %v\noutput:\n%s", err, stdoutSrc)
	}
	if len(srcEntries) == 0 {
		t.Fatal("expected entries in source-mode JSON output")
	}
	if _, ok := srcEntries[0]["source"]; !ok {
		t.Error("expected 'source' field when --output is NOT set")
	}
}

// TestExportTableJSONWithOutputDir (#631): --format json always emits rows to
// stdout; --output additionally writes elements.json and prints "Exported:" to stderr.
func TestExportTableJSONWithOutputDir(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	outDir := filepath.Join(dir, "out-table")

	stdout, stderr, code := runCLISplit(t, bin, dir,
		"export-table", "--format", "json", "--output", outDir,
	)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stdout)
	}

	// stdout must always be raw rows JSON regardless of --output
	var rows []interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &rows); err != nil {
		t.Fatalf("expected rows JSON on stdout, got: %v\noutput:\n%s", err, stdout)
	}

	// --output writes the same rows to elements.json on disk
	elemPath := filepath.Join(outDir, "elements.json")
	fileData, err := os.ReadFile(elemPath)
	if err != nil {
		t.Fatalf("expected elements.json at %q: %v", elemPath, err)
	}
	var fileRows []interface{}
	if err := json.Unmarshal(fileData, &fileRows); err != nil {
		t.Fatalf("invalid JSON in elements.json: %v", err)
	}

	// stderr must report absolute path
	if !strings.Contains(stderr, outDir) {
		t.Errorf("expected 'Exported:' with outDir in stderr, got: %s", stderr)
	}

	// Without --output: same rows shape on stdout, no file created
	stdoutSrc, _, code2 := runCLISplit(t, bin, dir,
		"export-table", "--format", "json",
	)
	if code2 != 0 {
		t.Fatalf("source mode exit %d: %s", code2, stdoutSrc)
	}
	var rowsSrc []interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdoutSrc)), &rowsSrc); err != nil {
		t.Fatalf("invalid JSON rows (no --output): %v\noutput:\n%s", err, stdoutSrc)
	}
}

func TestExportDiagramJSONWithOutputDir_HTML(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	outDir := filepath.Join(dir, "out-html")

	stdout, _, code := runCLISplit(t, bin, dir,
		"export-diagram", "--diagram-format", "html",
		"--output", outDir, "--format", "json",
	)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stdout)
	}

	var entries []map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &entries); err != nil {
		t.Fatalf("invalid JSON: %v\noutput:\n%s", err, stdout)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one entry")
	}
	for _, e := range entries {
		if _, hasSource := e["source"]; hasSource {
			t.Error("expected no 'source' field for html when --output is set")
		}
		p, ok := e["path"].(string)
		if !ok || p == "" {
			t.Errorf("expected non-empty 'path' field, got: %v", e)
			continue
		}
		if !filepath.IsAbs(p) {
			t.Errorf("expected absolute path, got relative: %q", p)
		}
		if _, err := os.ReadFile(p); err != nil {
			t.Errorf("html file missing at %q: %v", p, err)
		}
	}
}

func assertDiagramJSONOutputDir(t *testing.T, stdout, outDir string) {
	t.Helper()
	var entries []map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &entries); err != nil {
		t.Fatalf("invalid JSON: %v\noutput:\n%s", err, stdout)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one entry")
	}
	for i, e := range entries {
		if _, hasSource := e["source"]; hasSource {
			t.Errorf("entry[%d]: expected no 'source' when --output is set", i)
		}
		p, ok := e["path"].(string)
		if !ok || p == "" {
			t.Errorf("entry[%d]: expected non-empty 'path', got: %v", i, e)
			continue
		}
		if !filepath.IsAbs(p) {
			t.Errorf("entry[%d]: expected absolute path, got: %q", i, p)
		}
		if _, err := os.ReadFile(p); err != nil {
			t.Errorf("entry[%d]: file missing at %q: %v", i, p, err)
		}
	}
}

func TestExportDiagramJSONWithOutputDir_DOT(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	outDir := filepath.Join(dir, "out-dot")
	stdout, _, code := runCLISplit(t, bin, dir,
		"export-diagram", "--diagram-format", "dot",
		"--output", outDir, "--format", "json",
	)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stdout)
	}
	assertDiagramJSONOutputDir(t, stdout, outDir)
}

func TestExportDiagramJSONWithOutputDir_D2(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	outDir := filepath.Join(dir, "out-d2")
	stdout, _, code := runCLISplit(t, bin, dir,
		"export-diagram", "--diagram-format", "d2",
		"--output", outDir, "--format", "json",
	)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stdout)
	}
	assertDiagramJSONOutputDir(t, stdout, outDir)
}

func TestExportDiagramJSONWithOutputDir_Structurizr(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	outDir := filepath.Join(dir, "out-structurizr")
	stdout, _, code := runCLISplit(t, bin, dir,
		"export-diagram", "--diagram-format", "structurizr",
		"--output", outDir, "--format", "json",
	)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stdout)
	}
	assertDiagramJSONOutputDir(t, stdout, outDir)
}

func TestExportSequenceJSONWithOutputDir(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	modelPath := filepath.Join(dir, "architecture.jsonc")
	model := `{
  "specification": { "elements": { "svc": { "notation": "Service" } } },
  "model": {
    "frontend": { "kind": "svc", "title": "Frontend" },
    "backend":  { "kind": "svc", "title": "Backend" }
  },
  "views": {},
  "dynamicViews": [
    {
      "key": "flow", "title": "Flow",
      "steps": [
        { "index": 1, "from": "frontend", "to": "backend", "label": "call", "type": "sync" }
      ]
    }
  ]
}`
	if err := os.WriteFile(modelPath, []byte(model), 0o644); err != nil {
		t.Fatal(err)
	}

	outDir := filepath.Join(dir, "out")

	stdout, _, code := runCLISplit(t, bin, dir,
		"export-sequence", "--model", modelPath,
		"--output", outDir, "--format", "json",
	)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stdout)
	}

	var entries []map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &entries); err != nil {
		t.Fatalf("invalid JSON: %v\noutput:\n%s", err, stdout)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one entry")
	}
	p, ok := entries[0]["path"].(string)
	if !ok || p == "" {
		t.Errorf("expected non-empty 'path' field, got: %v", entries[0])
	}
	if !filepath.IsAbs(p) {
		t.Errorf("expected absolute path in JSON, got relative: %q", p)
	}
	if _, err := os.ReadFile(p); err != nil {
		t.Errorf("path %q in JSON but file missing: %v", p, err)
	}
	if _, hasSource := entries[0]["source"]; hasSource {
		t.Error("expected no 'source' field when --output is set")
	}

	// Backward-compat: without --output the "source" field must still be present.
	stdoutSrc, _, code2 := runCLISplit(t, bin, dir,
		"export-sequence", "--model", modelPath, "--format", "json",
	)
	if code2 != 0 {
		t.Fatalf("source mode exit %d: %s", code2, stdoutSrc)
	}
	var srcEntries []map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdoutSrc)), &srcEntries); err != nil {
		t.Fatalf("invalid JSON (source mode): %v\noutput:\n%s", err, stdoutSrc)
	}
	if len(srcEntries) == 0 {
		t.Fatal("expected entries in source-mode JSON output")
	}
	if _, ok := srcEntries[0]["source"]; !ok {
		t.Error("expected 'source' field when --output is NOT set")
	}
}

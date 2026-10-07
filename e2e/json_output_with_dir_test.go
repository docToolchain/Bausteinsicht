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
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

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

func TestExportDiagramJSONWithOutputDir_HTML(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	outDir := filepath.Join(dir, "out-html")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

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
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

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
}

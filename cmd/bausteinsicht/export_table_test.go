package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const exportTableTestModel = `{
  "specification": {
    "elements": {
      "system": {"notation": "System", "container": true},
      "container": {"notation": "Container"},
      "actor": {"notation": "Actor"}
    }
  },
  "model": {
    "user": {"kind": "actor", "title": "User", "description": "End user"},
    "shop": {"kind": "system", "title": "Shop", "description": "E-commerce", "children": {
      "api": {"kind": "container", "title": "API", "description": "REST backend", "technology": "Go"}
    }}
  },
  "relationships": [],
  "views": {
    "context": {
      "title": "System Context",
      "include": ["user", "shop"]
    },
    "containers": {
      "title": "Container View",
      "scope": "shop",
      "include": ["user", "shop.*"]
    }
  }
}`

func writeExportTableModel(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "architecture.jsonc")
	if err := os.WriteFile(p, []byte(exportTableTestModel), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestExportTable_AsciiDocToStdout(t *testing.T) {
	modelPath := writeExportTableModel(t)

	out, err := executeRootCmd("export-table", "--model", modelPath, "--view", "containers", "--table-format", "adoc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "|===") {
		t.Error("expected AsciiDoc table delimiter")
	}
	if !strings.Contains(out, "API") {
		t.Error("expected element 'API' in output")
	}
	if !strings.Contains(out, "Container View") {
		t.Error("expected view title")
	}
}

func TestExportTable_MarkdownToStdout(t *testing.T) {
	modelPath := writeExportTableModel(t)

	out, err := executeRootCmd("export-table", "--model", modelPath, "--view", "context", "--table-format", "md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "| Element") {
		t.Error("expected Markdown table header")
	}
	if !strings.Contains(out, "Shop") {
		t.Error("expected element 'Shop' in output")
	}
}

func TestExportTable_AllViewsToStdout(t *testing.T) {
	modelPath := writeExportTableModel(t)

	out, err := executeRootCmd("export-table", "--model", modelPath, "--table-format", "adoc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "System Context") {
		t.Error("expected 'System Context' view title")
	}
	if !strings.Contains(out, "Container View") {
		t.Error("expected 'Container View' view title")
	}
}

func TestExportTable_CombinedToStdout(t *testing.T) {
	modelPath := writeExportTableModel(t)

	out, err := executeRootCmd("export-table", "--model", modelPath, "--combined", "--table-format", "adoc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "All Elements") {
		t.Error("expected 'All Elements' title")
	}
	if !strings.Contains(out, "API") {
		t.Error("expected 'API' in combined output")
	}
}

func TestExportTable_WriteToFile(t *testing.T) {
	modelPath := writeExportTableModel(t)
	outDir := t.TempDir()

	_, err := executeRootCmd("export-table", "--model", modelPath, "--view", "containers", "--table-format", "adoc", "--output", outDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outPath := filepath.Join(outDir, "containers-elements.adoc")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected output file: %v", err)
	}
	if !strings.Contains(string(data), "API") {
		t.Error("expected 'API' in output file")
	}
}

func TestExportTable_InvalidView(t *testing.T) {
	modelPath := writeExportTableModel(t)

	_, err := executeRootCmd("export-table", "--model", modelPath, "--view", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent view")
	}
}

func TestExportTable_InvalidFormat(t *testing.T) {
	modelPath := writeExportTableModel(t)

	_, err := executeRootCmd("export-table", "--model", modelPath, "--table-format", "csv")
	if err == nil {
		t.Error("expected error for invalid table format")
	}
}

func TestExportTableJSON_PathAndSourceModes(t *testing.T) {
	modelPath := writeExportTableModel(t)
	cases := []struct {
		name     string
		args     []string
		wantView string
		wantFile string
	}{
		{"all", nil, "all", "all-views-elements.json"},
		{"combined", []string{"--combined"}, "combined", "elements.json"},
		{"view", []string{"--view", "containers"}, "containers", "containers-elements.json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outDir := t.TempDir()
			var outBuf, errBuf bytes.Buffer
			root := NewRootCmd()
			root.SetOut(&outBuf)
			root.SetErr(&errBuf)
			root.SetArgs(append([]string{"export-table", "--model", modelPath, "--format", "json", "--output", outDir}, tc.args...))
			if err := root.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var entries []map[string]interface{}
			if err := json.Unmarshal(outBuf.Bytes(), &entries); err != nil {
				t.Fatalf("invalid envelope: %v\n%s", err, outBuf.String())
			}
			if len(entries) != 1 || entries[0]["view"] != tc.wantView || entries[0]["format"] != "table" {
				t.Fatalf("unexpected envelope: %v", entries)
			}
			p, _ := entries[0]["path"].(string)
			if filepath.Base(p) != tc.wantFile {
				t.Errorf("expected file %s, got %s", tc.wantFile, p)
			}
			data, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			var rows []interface{}
			if err := json.Unmarshal(data, &rows); err != nil {
				t.Errorf("file is not a JSON rows array: %v", err)
			}

			var srcBuf bytes.Buffer
			root2 := NewRootCmd()
			root2.SetOut(&srcBuf)
			root2.SetErr(&errBuf)
			root2.SetArgs(append([]string{"export-table", "--model", modelPath, "--format", "json"}, tc.args...))
			if err := root2.Execute(); err != nil {
				t.Fatalf("source mode error: %v", err)
			}
			var srcRows []interface{}
			if err := json.Unmarshal(srcBuf.Bytes(), &srcRows); err != nil {
				t.Errorf("source mode must emit rows array: %v", err)
			}
		})
	}
}

func TestExportTable_CombinedWithViewWarnsAndWriteFailure(t *testing.T) {
	modelPath := writeExportTableModel(t)
	var errBuf bytes.Buffer
	root := NewRootCmd()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&errBuf)
	root.SetArgs([]string{"export-table", "--model", modelPath, "--combined", "--view", "containers"})
	_ = root.Execute()
	if !strings.Contains(errBuf.String(), "--view is ignored") {
		t.Errorf("expected warning, got: %s", errBuf.String())
	}

	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, extra := range [][]string{{"--format", "json"}, {}} {
		args := append([]string{"export-table", "--model", modelPath, "--output", filepath.Join(blocker, "sub")}, extra...)
		if _, err := executeRootCmd(args...); err == nil {
			t.Errorf("expected write failure for %v", extra)
		}
	}
}

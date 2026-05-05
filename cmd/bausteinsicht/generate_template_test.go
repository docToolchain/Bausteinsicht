package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const generateTemplateTestModel = `{
  "specification": {
    "elements": {
      "person": {"notation": "Person"},
      "system": {"notation": "System", "container": true},
      "database": {"notation": "Database"},
      "external_system": {"notation": "External System"}
    }
  },
  "model": {
    "user": {"kind": "person", "title": "User", "description": "End user"},
    "shop": {"kind": "system", "title": "Shop", "description": "E-commerce"}
  },
  "relationships": [],
  "views": {}
}`

func writeGenerateTemplateTestModel(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "architecture.jsonc")
	if err := os.WriteFile(p, []byte(generateTemplateTestModel), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestGenerateTemplate_DefaultStyle(t *testing.T) {
	modelPath := writeGenerateTemplateTestModel(t)
	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "template.drawio")

	out, err := executeRootCmd("generate-template", "--model", modelPath, "--output", outPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Generated template") {
		t.Error("expected 'Generated template' in output")
	}

	// Verify file was created
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected output file: %v", err)
	}
	if !strings.Contains(string(data), "<?xml") {
		t.Error("expected XML in file")
	}
	if !strings.Contains(string(data), "mxGraphModel") {
		t.Error("expected mxGraphModel element")
	}
}

func TestGenerateTemplate_C4Style(t *testing.T) {
	modelPath := writeGenerateTemplateTestModel(t)
	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "template.drawio")

	_, err := executeRootCmd("generate-template", "--model", modelPath, "--style", "c4", "--output", outPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected output file: %v", err)
	}
	if !strings.Contains(string(data), "fillColor=") {
		t.Error("expected fillColor styling")
	}
}

func TestGenerateTemplate_MinimalStyle(t *testing.T) {
	modelPath := writeGenerateTemplateTestModel(t)
	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "template.drawio")

	_, err := executeRootCmd("generate-template", "--model", modelPath, "--style", "minimal", "--output", outPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected output file: %v", err)
	}
	if !strings.Contains(string(data), "mxGraphModel") {
		t.Error("expected valid XML")
	}
}

func TestGenerateTemplate_DarkStyle(t *testing.T) {
	modelPath := writeGenerateTemplateTestModel(t)
	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "template.drawio")

	_, err := executeRootCmd("generate-template", "--model", modelPath, "--style", "dark", "--output", outPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected output file: %v", err)
	}
	if !strings.Contains(string(data), "mxGraphModel") {
		t.Error("expected valid XML")
	}
}

func TestGenerateTemplate_InvalidStyle(t *testing.T) {
	modelPath := writeGenerateTemplateTestModel(t)
	_, err := executeRootCmd("generate-template", "--model", modelPath, "--style", "nonexistent")
	if err == nil {
		t.Error("expected error for invalid style")
	}
}

func TestGenerateTemplate_AutoDetectModel(t *testing.T) {
	modelPath := writeGenerateTemplateTestModel(t)
	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "template.drawio")

	// Change to directory with model for auto-detection
	oldDir, _ := os.Getwd()
	modelDir := filepath.Dir(modelPath)
	if err := os.Chdir(modelDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	_, err := executeRootCmd("generate-template", "--output", outPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.ReadFile(outPath); err != nil {
		t.Fatalf("expected output file: %v", err)
	}
}

func TestGenerateTemplate_ContainsAllKinds(t *testing.T) {
	modelPath := writeGenerateTemplateTestModel(t)
	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "template.drawio")

	_, err := executeRootCmd("generate-template", "--model", modelPath, "--output", outPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected output file: %v", err)
	}
	content := string(data)

	kinds := []string{"person", "system", "database", "external_system"}
	for _, kind := range kinds {
		if !strings.Contains(content, "["+kind+"]") {
			t.Errorf("expected kind %q in template", kind)
		}
	}
}

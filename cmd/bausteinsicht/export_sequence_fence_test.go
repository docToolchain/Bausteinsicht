package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sequenceFenceModel = `{"specification":{"elements":{"actor":{"notation":"A"},"system":{"notation":"S"}}},
 "model":{"u":{"kind":"actor","title":"U"},"s":{"kind":"system","title":"S"}},
 "dynamicViews":[{"key":"f","title":"F","steps":[{"index":1,"from":"u","to":"s","label":"go"}]}]}`

func readFenceFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// A Mermaid sequence diagram in a .md file is a fenced code block, so that
// Markdown viewers and mmdc see a diagram (#715); on stdout it stays raw.
func TestExportSequence_MermaidMarkdownFileIsFenced(t *testing.T) {
	model := createTempFile(t, sequenceFenceModel)
	defer removeTempFile(t, model)
	out := filepath.Join(t.TempDir(), "out")

	if text, err := executeImportCmd("export-sequence", "--model", model, "--diagram-format", "mermaid", "--output", out); err != nil {
		t.Fatalf("%v\n%s", err, text)
	}
	got := readFenceFile(t, filepath.Join(out, "sequence-f.md"))
	if !strings.HasPrefix(got, "```mermaid\nsequenceDiagram\n") || !strings.HasSuffix(got, "\n```\n") {
		t.Errorf("want a ```mermaid fence around the diagram, got:\n%s", got)
	}

	raw, err := executeImportCmd("export-sequence", "--model", model, "--diagram-format", "mermaid")
	if err != nil || strings.Contains(raw, "```") || !strings.HasPrefix(raw, "sequenceDiagram\n") {
		t.Errorf("stdout stays the raw diagram, got %q (%v)", raw, err)
	}

	puml := filepath.Join(t.TempDir(), "puml")
	if _, err := executeImportCmd("export-sequence", "--model", model, "--output", puml); err != nil {
		t.Fatal(err)
	}
	if got := readFenceFile(t, filepath.Join(puml, "sequence-f.puml")); strings.Contains(got, "```") {
		t.Errorf("a PlantUML file is not fenced:\n%s", got)
	}
}

// With --format json and --output the file is fenced as well; without --output the
// "source" of the entry is the raw diagram.
func TestExportSequence_MermaidJSONFileIsFenced(t *testing.T) {
	model := createTempFile(t, sequenceFenceModel)
	defer removeTempFile(t, model)
	out := filepath.Join(t.TempDir(), "json")

	if text, err := executeImportCmd("export-sequence", "--model", model, "--diagram-format", "mermaid", "--format", "json", "--output", out); err != nil {
		t.Fatalf("%v\n%s", err, text)
	}
	if got := readFenceFile(t, filepath.Join(out, "sequence-f.md")); !strings.HasPrefix(got, "```mermaid\n") {
		t.Errorf("the file must be fenced:\n%s", got)
	}
	text, err := executeImportCmd("export-sequence", "--model", model, "--diagram-format", "mermaid", "--format", "json")
	if err != nil || strings.Contains(text, "```") || !strings.Contains(text, `"source": "sequenceDiagram`) {
		t.Errorf("the source of a JSON entry stays raw, got %q (%v)", text, err)
	}
}

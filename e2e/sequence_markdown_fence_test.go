package e2e

// #715: a Mermaid sequence diagram that is written to a .md file is a fenced
// code block, so that Markdown viewers and mmdc see a diagram.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sequenceFenceModelFile = `{"specification":{"elements":{"actor":{"notation":"A"},"system":{"notation":"S"}}},
 "model":{"u":{"kind":"actor","title":"U"},"s":{"kind":"system","title":"S"}},
 "dynamicViews":[{"key":"f","title":"F","steps":[{"index":1,"from":"u","to":"s","label":"go"}]}]}`

func TestExportSequence_MermaidMarkdownFile(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "m.jsonc"), []byte(sequenceFenceModelFile), 0o600); err != nil {
		t.Fatal(err)
	}
	runCLI(t, bin, dir, "--model", "m.jsonc", "export-sequence", "--diagram-format", "mermaid", "--output", "out")
	data, err := os.ReadFile(filepath.Join(dir, "out", "sequence-f.md"))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); !strings.HasPrefix(got, "```mermaid\nsequenceDiagram\n") || !strings.HasSuffix(got, "\n```\n") {
		t.Errorf("the .md file must hold a fenced diagram:\n%s", got)
	}
}

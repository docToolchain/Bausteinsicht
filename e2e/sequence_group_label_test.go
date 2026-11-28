package e2e

// #711: the label of a group block is kept in the Mermaid sequence export.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sequenceGroupModel = `{"specification":{"elements":{"actor":{"notation":"A"},"system":{"notation":"S"}}},
 "model":{"u":{"kind":"actor","title":"U"},"s":{"kind":"system","title":"S"}},
 "dynamicViews":[{"key":"f","title":"F","steps":[{"index":1,"type":"group","label":"g"},{"index":2,"from":"u","to":"s"},{"index":3,"type":"end"}]}]}`

func TestExportSequence_MermaidGroupKeepsItsLabel(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "m.jsonc"), []byte(sequenceGroupModel), 0o600); err != nil {
		t.Fatal(err)
	}
	out := runCLI(t, bin, dir, "--model", "m.jsonc", "export-sequence", "--diagram-format", "mermaid")
	if !strings.Contains(out, "Note over u,s: g") {
		t.Errorf("the label of the group must be in the output:\n%s", out)
	}
	if strings.Contains(out, ": \n") {
		t.Errorf("no trailing \": \" for a message without a label:\n%s", out)
	}
}

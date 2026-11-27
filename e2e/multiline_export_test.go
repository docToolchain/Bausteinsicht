package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const multilineModel = `{"specification":{"elements":{"actor":{"notation":"Actor"}},"relationships":{"uses":{"notation":"uses"}}},
 "model":{"u":{"kind":"actor","title":"U","description":"line1\nline2"},"v":{"kind":"actor","title":"V"}},
 "relationships":[{"from":"u","to":"v","kind":"uses","label":"reads\nwrites"}],
 "views":{"v":{"title":"V","include":["*"]}}}`

// TestExportDiagram_MultilineTextStaysOnOneLine (#690): a description or label
// with a line break was written with a raw break inside the quoted string.
func TestExportDiagram_MultilineTextStaysOnOneLine(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	model := filepath.Join(dir, "m.jsonc")
	writeFile(t, model, multilineModel)

	want := map[string][]string{
		"plantuml": {`"line1\nline2"`, `"reads\nwrites"`},
		"mermaid":  {`"line1<br/>line2"`, `"reads<br/>writes"`},
		"dot":      {`label="reads\nwrites"`},
		"d2":       {`reads\nwrites`},
	}
	for format, needles := range want {
		t.Run(format, func(t *testing.T) {
			out := filepath.Join(dir, "out-"+format)
			runCLI(t, bin, dir, "--model", model, "export-diagram", "--diagram-format", format, "--output", out)
			files, _ := filepath.Glob(filepath.Join(out, "*"))
			if len(files) == 0 {
				t.Fatalf("no output written to %s", out)
			}
			data, err := os.ReadFile(files[0])
			if err != nil {
				t.Fatal(err)
			}
			text := string(data)
			for _, n := range needles {
				if !strings.Contains(text, n) {
					t.Errorf("%s output lacks %s:\n%s", format, n, text)
				}
			}
			if strings.Contains(text, "line1\nline2") || strings.Contains(text, "reads\nwrites") {
				t.Errorf("%s output has a raw line break inside a string:\n%s", format, text)
			}
			if format == "dot" {
				if bin, err := exec.LookPath("dot"); err == nil {
					if res, err := exec.Command(bin, "-Tsvg", files[0]).CombinedOutput(); err != nil {
						t.Errorf("dot rejects the output: %v\n%s", err, res)
					}
				}
			}
		})
	}
}

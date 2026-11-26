package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Importing a file with the wrong --from (or a text file without any model) wrote
// an empty model, printed "Next steps" and exited with 0; with --force it
// overwrote the existing architecture.jsonc (#705).
func TestImportCmd_NoElementsFound_ExitCode1(t *testing.T) {
	dir := t.TempDir()
	garbage := filepath.Join(dir, "g.txt")
	if err := os.WriteFile(garbage, []byte("garbage text\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	xmi := absDSL(t, "internal", "importer", "xmi", "testdata", "basic.xmi")

	for name, args := range map[string][]string{
		"garbage as structurizr": {"--from", "structurizr", garbage},
		"xmi file as likec4":     {"--from", "likec4", xmi},
		"garbage, dry run":       {"--from", "structurizr", "--dry-run", garbage},
	} {
		t.Run(name, func(t *testing.T) {
			out := filepath.Join(dir, strings.ReplaceAll(name, " ", "_")+".jsonc")
			existing := `{"model":{"keep":{"kind":"system","title":"Keep"}}}`
			if err := os.WriteFile(out, []byte(existing), 0o600); err != nil {
				t.Fatal(err)
			}
			text, err := executeImportCmd(append([]string{"import", "--output", out, "--force"}, args...)...)
			if err == nil || exitCodeOf(t, err) != 1 || !strings.Contains(err.Error(), "no elements found") {
				t.Fatalf("want exit code 1 with 'no elements found', got %v\n%s", err, text)
			}
			if strings.Contains(text, "Next steps") {
				t.Errorf("no next-steps hint for an empty result:\n%s", text)
			}
			if data, _ := os.ReadFile(out); string(data) != existing {
				t.Errorf("the existing model must not be overwritten, got:\n%s", data)
			}
		})
	}
}

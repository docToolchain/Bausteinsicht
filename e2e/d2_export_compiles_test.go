package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestD2ExportCompiles (#713): every file of `export-diagram --diagram-format d2`
// wrote `key: "label" [kind] {`, which the D2 compiler rejects ("unexpected text
// after double quoted string"), so no exported file compiled. The files are run
// through the real compiler (skipped when `d2` is not installed).
func TestD2ExportCompiles(t *testing.T) {
	d2, err := exec.LookPath("d2")
	if err != nil {
		t.Skip("d2 is not installed")
	}
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")
	out := filepath.Join(dir, "d2")
	runCLI(t, bin, dir, "export-diagram", "--diagram-format", "d2", "--output", out)

	files, _ := filepath.Glob(filepath.Join(out, "*.d2"))
	if len(files) < 3 {
		t.Fatalf("expected one file per view of the init model, got %v", files)
	}
	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			res, err := exec.Command(d2, f, filepath.Join(t.TempDir(), "out.svg")).CombinedOutput()
			if err != nil {
				data, _ := os.ReadFile(f)
				t.Fatalf("d2 does not compile %s: %v\n%s\n--- file ---\n%s", filepath.Base(f), err, res, strings.TrimSpace(string(data)))
			}
		})
	}
}

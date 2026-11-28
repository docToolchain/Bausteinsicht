package e2e

// #737: a model without views says so in the exports, and layout without a diagram
// says to run sync.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModelWithoutViewsIsExplained(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	const m = `{"specification":{"elements":{"actor":{"notation":"A"}}},"model":{"u":{"kind":"actor","title":"U"}}}`
	if err := os.WriteFile(filepath.Join(dir, "architecture.jsonc"), []byte(m), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, cmd := range []string{"export-diagram", "export-table"} {
		_, stderr, code := runCLISplit(t, bin, dir, cmd)
		if code != 0 || !strings.Contains(stderr, "No views defined in model") {
			t.Errorf("%s: want exit 0 and the notice on stderr, got %d: %q", cmd, code, stderr)
		}
	}
	runCLI(t, bin, dir, "sync")
	out, code := runCLIAllowFail(t, bin, dir, "layout")
	if code != 2 || !strings.Contains(out, "bausteinsicht sync") || strings.Contains(out, "LoadDocument") {
		t.Errorf("layout without a diagram: want exit 2 and the hint to run sync, got %d:\n%s", code, out)
	}
}

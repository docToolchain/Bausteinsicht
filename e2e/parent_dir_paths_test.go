package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #700: working in a sub-directory of a project (a docs folder next to the
// model) needs --model ../architecture.jsonc; the paths are typed by the user,
// so ".." is no traversal to guard against.
func TestParentDirectoryPathsForModelAndTemplate(t *testing.T) {
	bin := buildBinary(t)
	root := t.TempDir()
	runCLI(t, bin, root, "init")
	runCLI(t, bin, root, "generate-template", "--output", "architecture-template.drawio")
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	model := filepath.Join("..", "architecture.jsonc")
	// init writes the state too; remove it so that only sync can recreate it.
	if err := os.Remove(filepath.Join(root, ".bausteinsicht-sync")); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"validate"},
		{"lint"},
		{"status"},
		{"export-table"},
		{"sync"},
		{"sync", "--template", filepath.Join("..", "architecture-template.drawio")},
	} {
		full := append([]string{"--model", model}, args...)
		out, code := runCLIAllowFail(t, bin, sub, full...)
		if code != 0 || strings.Contains(out, "traversal") {
			t.Errorf("bausteinsicht %s: exit %d, want 0\n%s", strings.Join(full, " "), code, out)
		}
	}
	// The files of the sync land next to the model, not in the sub-directory.
	if _, err := os.Stat(filepath.Join(root, ".bausteinsicht-sync")); err != nil {
		t.Errorf("sync state next to the model: %v", err)
	}
	if entries, _ := os.ReadDir(sub); len(entries) != 0 {
		t.Errorf("the sub-directory should stay empty, has %d entries", len(entries))
	}
}

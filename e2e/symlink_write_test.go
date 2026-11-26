package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCommandsKeepSymbolicLinks (#723): a model, diagram or state file that is a
// symbolic link (monorepo, dotfiles) was replaced by a regular file on the first
// write; the target kept its old content and the link was gone.
func TestCommandsKeepSymbolicLinks(t *testing.T) {
	bin := buildBinary(t)
	root := t.TempDir()
	real := filepath.Join(root, "real")
	work := filepath.Join(root, "sl")
	for _, d := range []string{real, work} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	runCLI(t, bin, real, "init")
	for _, name := range []string{"architecture.jsonc", "architecture.drawio", ".bausteinsicht-sync", "template.drawio"} {
		if err := os.Symlink(filepath.Join(real, name), filepath.Join(work, name)); err != nil {
			t.Skipf("symbolic links are not available: %v", err)
		}
	}

	runCLI(t, bin, work, "add", "element", "--id", "sym1", "--kind", "actor", "--title", "S")
	runCLI(t, bin, work, "add", "view", "symv", "--title", "SV", "--include", "sym1")
	runCLI(t, bin, work, "sync")

	for _, name := range []string{"architecture.jsonc", "architecture.drawio", ".bausteinsicht-sync"} {
		info, err := os.Lstat(filepath.Join(work, name))
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("%s: the symbolic link was replaced by a regular file (err %v)", name, err)
		}
	}
	data, _ := os.ReadFile(filepath.Join(real, "architecture.jsonc"))
	if !strings.Contains(string(data), `"sym1"`) || !strings.Contains(string(data), `"symv"`) {
		t.Errorf("the model behind the link was not updated:\n%.300s", data)
	}
	draw, _ := os.ReadFile(filepath.Join(real, "architecture.drawio"))
	if !strings.Contains(string(draw), "sym1") {
		t.Error("the diagram behind the link was not updated")
	}
}

package e2e

// #724: a read-only model or diagram is not replaced, and the file that replaces a
// writable one keeps its mode.

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReadOnlyFilesAreNotOverwritten(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")
	runCLI(t, bin, dir, "sync")
	model, diagram := filepath.Join(dir, "architecture.jsonc"), filepath.Join(dir, "architecture.drawio")
	beforeModel, _ := os.ReadFile(model)

	if err := os.Chmod(model, 0o444); err != nil {
		t.Fatal(err)
	}
	out, code := runCLIAllowFail(t, bin, dir, "add", "element", "--id", "ro1", "--kind", "actor", "--title", "X")
	if code != 2 || !strings.Contains(out, "read-only") {
		t.Errorf("add element on a read-only model: want exit 2 and \"read-only\", got %d:\n%s", code, out)
	}
	if after, _ := os.ReadFile(model); string(after) != string(beforeModel) {
		t.Error("the read-only model was changed")
	}

	if err := os.Chmod(model, 0o644); err != nil {
		t.Fatal(err)
	}
	runCLI(t, bin, dir, "add", "element", "--id", "ro2", "--kind", "actor", "--title", "Y")
	if err := os.Chmod(diagram, 0o444); err != nil {
		t.Fatal(err)
	}
	beforeDiagram, _ := os.ReadFile(diagram)
	out, code = runCLIAllowFail(t, bin, dir, "sync")
	if code != 2 || !strings.Contains(out, "read-only") {
		t.Errorf("sync with a read-only diagram: want exit 2 and \"read-only\", got %d:\n%s", code, out)
	}
	if after, _ := os.ReadFile(diagram); string(after) != string(beforeDiagram) {
		t.Error("the read-only diagram was changed")
	}
}

func TestWrittenFilesKeepTheirMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no permission bits beyond read-only")
	}
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")
	runCLI(t, bin, dir, "sync")
	model, diagram := filepath.Join(dir, "architecture.jsonc"), filepath.Join(dir, "architecture.drawio")
	for _, f := range []string{model, diagram} {
		if err := os.Chmod(f, 0o664); err != nil {
			t.Fatal(err)
		}
	}
	runCLI(t, bin, dir, "add", "element", "--id", "kept", "--kind", "actor", "--title", "K")
	runCLI(t, bin, dir, "sync")
	for _, f := range []string{model, diagram} {
		if info, _ := os.Stat(f); info.Mode().Perm() != 0o664 {
			t.Errorf("%s: want mode 0664 to be kept, got %v", filepath.Base(f), info.Mode().Perm())
		}
	}
}

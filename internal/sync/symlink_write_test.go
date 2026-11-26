package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The state file can be a symbolic link too; it must be updated in place (#723).
func TestSaveState_KeepsASymbolicLink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "real", ".bausteinsicht-sync")
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := SaveState(target, emptyState()); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, ".bausteinsicht-sync")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symbolic links are not available: %v", err)
	}

	state := emptyState()
	state.Elements["marker"] = ElementState{Title: "Marker", Kind: "k"}
	if err := SaveState(link, state); err != nil {
		t.Fatal(err)
	}

	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the symbolic link was replaced by a regular file (err %v)", err)
	}
	if data, _ := os.ReadFile(target); !strings.Contains(string(data), "marker") {
		t.Errorf("the target of the link was not updated:\n%s", data)
	}
}

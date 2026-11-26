package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The lock file of a workspace can be a symbolic link; it must be updated in
// place (#723).
func TestSaveLock_KeepsASymbolicLink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "real", LockFileName)
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := SaveLock(target, &Lock{Sources: map[string]LockEntry{}}); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, LockFileName)
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symbolic links are not available: %v", err)
	}

	if err := SaveLock(link, &Lock{Sources: map[string]LockEntry{"m": {URL: "https://example.com/m", SHA256: "abc"}}}); err != nil {
		t.Fatal(err)
	}

	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the symbolic link was replaced by a regular file (err %v)", err)
	}
	if data, _ := os.ReadFile(target); !strings.Contains(string(data), "example.com/m") {
		t.Errorf("the target of the link was not updated:\n%s", data)
	}
}

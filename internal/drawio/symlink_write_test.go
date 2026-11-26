package drawio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A diagram that is a symbolic link must be updated in place (#723).
func TestSaveDocument_KeepsASymbolicLink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "real", "architecture.drawio")
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	doc := NewDocument()
	doc.AddPage("view-a", "A")
	if err := SaveDocument(target, doc); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "architecture.drawio")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symbolic links are not available: %v", err)
	}

	doc.AddPage("view-b", "B")
	if err := SaveDocument(link, doc); err != nil {
		t.Fatal(err)
	}

	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the symbolic link was replaced by a regular file (err %v)", err)
	}
	if data, _ := os.ReadFile(target); !strings.Contains(string(data), `id="view-b"`) {
		t.Errorf("the target of the link was not updated:\n%s", data)
	}
}

// A link that cannot be resolved for another reason than "does not exist" must
// make the write fail; replacing the link would report success and leave the
// target unchanged.
func TestSaveDocument_FailsWhenTheLinkCannotBeResolved(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permissions do not restrict root")
	}
	root := t.TempDir()
	dir := filepath.Join(root, "real")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "architecture.drawio")
	doc := NewDocument()
	doc.AddPage("view-a", "A")
	if err := SaveDocument(target, doc); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "architecture.drawio")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symbolic links are not available: %v", err)
	}
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Skipf("cannot restrict the directory: %v", err)
	}
	defer func() { _ = os.Chmod(dir, 0o700) }()

	if err := SaveDocument(link, doc); err == nil {
		t.Error("SaveDocument must fail when the link cannot be resolved")
	}
	_ = os.Chmod(dir, 0o700)
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the symbolic link was replaced by a regular file (err %v)", err)
	}
}

package drawio

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func savedDiagram(t *testing.T, mode os.FileMode) (string, *Document) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "architecture.drawio")
	doc := NewDocument()
	doc.AddPage("view-a", "A")
	if err := SaveDocument(path, doc); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	return path, doc
}

// A diagram file that is read-only is not replaced (#724).
func TestSaveDocument_RefusesAReadOnlyFile(t *testing.T) {
	path, doc := savedDiagram(t, 0o444)
	before, _ := os.ReadFile(path)
	doc.AddPage("view-b", "B")
	err := SaveDocument(path, doc)
	if err == nil || !strings.Contains(err.Error(), "read-only") || !strings.Contains(err.Error(), path) {
		t.Errorf("want an error \"file is read-only\" naming the file, got %v", err)
	}
	if after, _ := os.ReadFile(path); string(after) != string(before) {
		t.Error("the read-only diagram was changed")
	}
}

// The file that replaces the diagram keeps its mode (#724).
func TestSaveDocument_KeepsTheMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no permission bits beyond read-only")
	}
	path, doc := savedDiagram(t, 0o640)
	doc.AddPage("view-b", "B")
	if err := SaveDocument(path, doc); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o640 {
		t.Errorf("want mode 0640 to be kept, got %v", info.Mode().Perm())
	}
}

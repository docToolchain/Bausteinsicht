package e2e

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// moveCell sets x of the geometry of the cell object with the given
// bausteinsicht_id on every page and returns how many it changed.
func moveCell(t *testing.T, drawio, id, x string) int {
	t.Helper()
	data, err := os.ReadFile(drawio)
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`(<object [^>]*bausteinsicht_id="` + regexp.QuoteMeta(id) + `"[^>]*>\s*<mxCell [^>]*>\s*<mxGeometry )x="[^"]*"`)
	n := len(re.FindAllString(string(data), -1))
	out := re.ReplaceAllString(string(data), `${1}x="`+x+`"`)
	if err := os.WriteFile(drawio, []byte(out), 0o600); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestRenameKeepsTheDrawioCells (#686): after `rename` the next sync deleted and
// re-created every cell of the element, so its position was lost, and warned
// about connectors of the old ID.
func TestRenameKeepsTheDrawioCells(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")
	runCLI(t, bin, dir, "sync")

	drawio := filepath.Join(dir, "architecture.drawio")
	moved := moveCell(t, drawio, "customer", "777")
	if moved == 0 {
		t.Fatal("found no cell of the element customer to move")
	}
	runCLI(t, bin, dir, "sync")

	out := runCLI(t, bin, dir, "rename", "--id", "customer", "--to", "client")
	if !strings.Contains(out, "customer") {
		t.Fatalf("rename output: %s", out)
	}
	syncOut := runCLI(t, bin, dir, "sync")
	if strings.Contains(syncOut, "WARNING: Relationship") || strings.Contains(syncOut, "rejected") {
		t.Errorf("sync warns about the old ID:\n%s", syncOut)
	}
	if strings.Contains(syncOut, "deleted") && !strings.Contains(syncOut, "0 deleted") {
		t.Errorf("sync deleted cells after the rename:\n%s", syncOut)
	}
	if strings.Contains(syncOut, "added") && !strings.Contains(syncOut, "0 added") {
		t.Errorf("sync re-created cells after the rename:\n%s", syncOut)
	}
	data, err := os.ReadFile(drawio)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(data), `x="777"`); got != moved {
		t.Errorf("the moved position survives in %d place(s), want %d", got, moved)
	}
	if !strings.Contains(string(data), `bausteinsicht_id="client"`) || strings.Contains(string(data), `bausteinsicht_id="customer"`) {
		t.Errorf("the cells must carry the new ID only")
	}
}

// Renaming a parent renames the cells of its children, their sub-cells and the
// connectors that end at them.
func TestRenameKeepsTheCellsOfChildrenAndConnectors(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")
	runCLI(t, bin, dir, "sync")

	drawio := filepath.Join(dir, "architecture.drawio")
	moved := moveCell(t, drawio, "onlineshop.api", "555")
	if moved == 0 {
		t.Fatal("found no cell of onlineshop.api to move")
	}
	runCLI(t, bin, dir, "sync")
	runCLI(t, bin, dir, "rename", "--id", "onlineshop", "--to", "shop")
	syncOut := runCLI(t, bin, dir, "sync")
	if strings.Contains(syncOut, "WARNING: Relationship") || (strings.Contains(syncOut, "deleted") && !strings.Contains(syncOut, "0 deleted")) {
		t.Errorf("sync after the rename of a parent:\n%s", syncOut)
	}
	data, _ := os.ReadFile(drawio)
	text := string(data)
	if got := strings.Count(text, `x="555"`); got != moved {
		t.Errorf("the moved child keeps its position in %d place(s), want %d", got, moved)
	}
	if strings.Contains(text, `--onlineshop`) || strings.Contains(text, `bausteinsicht_id="onlineshop`) {
		t.Errorf("cells with the old ID are left in the file")
	}
	runCLI(t, bin, dir, "validate")
}

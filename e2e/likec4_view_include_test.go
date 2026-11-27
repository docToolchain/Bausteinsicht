package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLikeC4ImportViewsAreNotEmpty (#706): import -> sync -> the view pages show
// the elements. The imported views had no include list, so sync warned that
// each view resolved to 0 elements.
func TestLikeC4ImportViewsAreNotEmpty(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	src, err := os.ReadFile(filepath.Join("..", "internal", "importer", "likec4", "testdata", "simple.c4"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "simple.c4"), string(src))
	runCLI(t, bin, dir, "import", "simple.c4", "--from", "likec4")
	runCLI(t, bin, dir, "validate")

	out := runCLI(t, bin, dir, "sync")
	if strings.Contains(out, "resolves to 0 elements") || strings.Contains(out, "not visible in any view") {
		t.Errorf("sync reports empty views or hidden elements:\n%s", out)
	}
	drawio, err := os.ReadFile(filepath.Join(dir, "architecture.drawio"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(drawio), `bausteinsicht_id="myPlatform.api"`) {
		t.Errorf("the container view page has no api element:\n%s", drawio)
	}
	diagram := runCLI(t, bin, dir, "export-diagram", "--diagram-format", "plantuml", "--view", "containers")
	if !strings.Contains(diagram, "myPlatform_api") {
		t.Errorf("the exported container view is empty:\n%s", diagram)
	}
}

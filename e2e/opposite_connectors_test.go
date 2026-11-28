package e2e

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// #717: the model of the issue: u->s and s->u. After sync and layout the two
// connectors must not run on top of each other.
func TestOppositeRelationshipsAreRoutedApart(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "m.jsonc")
	if err := os.WriteFile(path, []byte(`{"specification":{"elements":{"actor":{"notation":"Actor"},"system":{"notation":"System"}},"relationships":{"uses":{"notation":"uses"}}},
 "model":{"u":{"kind":"actor","title":"User"},"s":{"kind":"system","title":"Core"}},
 "relationships":[{"from":"u","to":"s","kind":"uses","label":"submits requests"},{"from":"s","to":"u","kind":"uses","label":"sends notifications"}],
 "views":{"v":{"title":"V","include":["*"]}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	check := func(step string) {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(dir, "m.drawio"))
		if err != nil {
			data, err = os.ReadFile(filepath.Join(dir, "architecture.drawio"))
		}
		if err != nil {
			t.Fatal(err)
		}
		points := regexp.MustCompile(`<mxPoint x="[^"]+" y="[^"]+"/>`).FindAllString(string(data), -1)
		if len(points) != 2 || points[0] == points[1] {
			t.Errorf("%s: want two different waypoints, got %v", step, points)
		}
	}
	runCLI(t, bin, dir, "--model", path, "sync")
	check("after sync")
	runCLI(t, bin, dir, "--model", path, "layout")
	check("after layout")
	runCLI(t, bin, dir, "--model", path, "sync")
	check("after a second sync")
}

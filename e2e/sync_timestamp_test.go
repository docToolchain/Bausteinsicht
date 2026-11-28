package e2e

// #731, #761: a sync that changes nothing leaves the draw.io file byte-identical,
// however much later it runs; the "Last synced" time of the metadata box is only
// updated by a sync that changes something on the page.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestSync_UnchangedModelLeavesTheDiagramAlone(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")
	runCLI(t, bin, dir, "sync")

	// Make the stored time an old one, as if the last sync was long ago.
	path := filepath.Join(dir, "architecture.drawio")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stamp := regexp.MustCompile(`Last synced: \d{4}-\d{2}-\d{2} \d{2}:\d{2}`)
	if !stamp.Match(data) {
		t.Fatalf("setup: no \"Last synced\" in the diagram")
	}
	old := stamp.ReplaceAll(data, []byte("Last synced: 2001-01-01 00:00"))
	if err := os.WriteFile(path, old, 0o600); err != nil {
		t.Fatal(err)
	}

	out := runCLI(t, bin, dir, "sync")
	if !strings.Contains(out, "Already in sync") {
		t.Errorf("a sync without changes says so:\n%s", out)
	}
	if after, _ := os.ReadFile(path); string(after) != string(old) {
		t.Error("a sync without changes must leave the diagram byte-identical")
	}

	// A change refreshes the time of the page that it changed, and only of that.
	runCLI(t, bin, dir, "add", "element", "--id", "fresh", "--kind", "actor", "--title", "Fresh")
	runCLI(t, bin, dir, "add", "view", "context", "--include", "fresh")
	runCLI(t, bin, dir, "sync")
	after, _ := os.ReadFile(path)
	stamps := stamp.FindAllString(string(after), -1)
	oldOnes := 0
	for _, s := range stamps {
		if s == "Last synced: 2001-01-01 00:00" {
			oldOnes++
		}
	}
	if oldOnes == 0 || oldOnes == len(stamps) {
		t.Errorf("the page that changed gets the new time, the others keep theirs: %d of %d still have the old one", oldOnes, len(stamps))
	}
}

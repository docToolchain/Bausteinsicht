package e2e

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/beevik/etree"
)

// removeConnectors deletes the connectors with the given label (and the label
// cells that hang on them) from the draw.io file, on every page or only on the
// page with the given diagram id (""= every page); it returns how many
// connectors were removed.
func removeConnectors(t *testing.T, path, label, onlyPage string) int {
	t.Helper()
	doc := etree.NewDocument()
	if err := doc.ReadFromFile(path); err != nil {
		t.Fatal(err)
	}
	removed := 0
	for _, diagram := range doc.FindElements("//diagram") {
		if onlyPage != "" && diagram.SelectAttrValue("id", "") != onlyPage {
			continue
		}
		for _, cell := range diagram.FindElements(".//mxCell[@edge='1']") {
			if cell.SelectAttrValue("value", "") != label {
				continue
			}
			id := cell.SelectAttrValue("id", "")
			for _, child := range diagram.FindElements(".//mxCell[@parent='" + id + "']") {
				child.Parent().RemoveChild(child)
			}
			cell.Parent().RemoveChild(cell)
			removed++
		}
	}
	if err := doc.WriteToFile(path); err != nil {
		t.Fatal(err)
	}
	return removed
}

func relationshipCount(t *testing.T, modelPath string) int {
	t.Helper()
	data, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	stripped := regexp.MustCompile(`(?m)^\s*//.*$`).ReplaceAllString(string(data), "")
	var m struct {
		Relationships []json.RawMessage `json:"relationships"`
	}
	if err := json.Unmarshal([]byte(stripped), &m); err != nil {
		t.Fatalf("model is not valid JSON: %v", err)
	}
	return len(m.Relationships)
}

// TestSyncAfterConnectorDeletedInDrawio (#719): deleting a connector in draw.io
// removes the relationship from the model; the syncs after that must be no-ops.
// Before the fix the forward sync drew the connector again in the same run and
// the model drifted (lifted connectors imported as new relationships, up to a
// model that no longer validates).
func TestSyncAfterConnectorDeletedInDrawio(t *testing.T) {
	bin := buildBinary(t)

	t.Run("deleted on every page: the relationship goes and the model stays stable", func(t *testing.T) {
		dir := t.TempDir()
		runCLI(t, bin, dir, "init")
		runCLI(t, bin, dir, "sync")
		before := relationshipCount(t, dir+"/architecture.jsonc")

		if n := removeConnectors(t, dir+"/architecture.drawio", "reads/writes", ""); n == 0 {
			t.Fatal("setup: the connector was not found")
		}
		runCLI(t, bin, dir, "sync")
		if got := relationshipCount(t, dir+"/architecture.jsonc"); got != before-1 {
			t.Fatalf("the deleted connector must remove exactly one relationship: %d -> %d", before, got)
		}
		for i := 1; i <= 4; i++ {
			out := runCLI(t, bin, dir, "sync")
			if !strings.Contains(out, "Already in sync") {
				t.Fatalf("sync %d after the deletion is not a no-op:\n%s", i, out)
			}
			if got := relationshipCount(t, dir+"/architecture.jsonc"); got != before-1 {
				t.Fatalf("sync %d changed the relationships to %d, want %d", i, got, before-1)
			}
		}
		runCLI(t, bin, dir, "validate")
	})

	t.Run("deleted on one page only: the model wins and nothing drifts", func(t *testing.T) {
		dir := t.TempDir()
		runCLI(t, bin, dir, "init")
		runCLI(t, bin, dir, "sync")
		before := relationshipCount(t, dir+"/architecture.jsonc")

		// "manages inventory" is drawn on two pages (directly on the container
		// view, lifted on the context view); it is gone from one of them only.
		if n := removeConnectors(t, dir+"/architecture.drawio", "manages inventory", "view-containers"); n == 0 {
			t.Fatal("setup: the connector was not found on the containers page")
		}
		for i := 1; i <= 3; i++ {
			runCLI(t, bin, dir, "sync")
			if got := relationshipCount(t, dir+"/architecture.jsonc"); got != before {
				t.Fatalf("sync %d changed the relationships to %d, want %d", i, got, before)
			}
		}
		if out := runCLI(t, bin, dir, "sync"); !strings.Contains(out, "Already in sync") {
			t.Errorf("the last sync is not a no-op:\n%s", out)
		}
	})
}

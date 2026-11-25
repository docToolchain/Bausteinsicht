package e2e

// TestRelationshipTechnologyInDrawio covers the draw.io part of #642: a
// relationship technology is shown on the connector as a second label, follows
// changes of the model, and does not turn into a change of the relationship
// label.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/beevik/etree"
)

// edgeTechLabels returns the value of every edge label cell ("...-tech") in
// the diagram, keyed by cell ID, and the values of the connectors themselves.
func edgeTechLabels(t *testing.T, drawioPath string) (techLabels, connectorValues map[string]string) {
	t.Helper()
	data, err := os.ReadFile(drawioPath)
	if err != nil {
		t.Fatal(err)
	}
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(data); err != nil {
		t.Fatal(err)
	}
	techLabels, connectorValues = map[string]string{}, map[string]string{}
	for _, cell := range doc.FindElements("//mxCell") {
		id := cell.SelectAttrValue("id", "")
		switch {
		case strings.HasPrefix(id, "rel-") && strings.HasSuffix(id, "-tech"): // not the element sub-cells ("<element>-tech")
			techLabels[id] = cell.SelectAttrValue("value", "")
		case cell.SelectAttrValue("edge", "") == "1":
			connectorValues[id] = cell.SelectAttrValue("value", "")
		}
	}
	return techLabels, connectorValues
}

func TestRelationshipTechnologyInDrawio(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")
	runCLI(t, bin, dir, "add", "relationship", "--from", "customer", "--to", "onlineshop",
		"--label", "orders via", "--technology", "HTTPS/JSON")
	runCLI(t, bin, dir, "sync")

	drawioPath := filepath.Join(dir, "architecture.drawio")
	tech, connectors := edgeTechLabels(t, drawioPath)
	if len(tech) == 0 {
		t.Fatal("no technology label in the diagram after sync")
	}
	for id, value := range tech {
		if value != "[HTTPS/JSON]" {
			t.Errorf("label %s = %q, want [HTTPS/JSON]", id, value)
		}
		if _, ok := connectors[strings.TrimSuffix(id, "-tech")]; !ok {
			t.Errorf("label %s is not attached to a connector of the diagram", id)
		}
	}
	for id, value := range connectors {
		if strings.Contains(value, "HTTPS") {
			t.Errorf("connector %s carries the technology in its own label: %q", id, value)
		}
	}

	// The technology changes in the model: the label follows.
	modelPath := filepath.Join(dir, "architecture.jsonc")
	data, _ := os.ReadFile(modelPath)
	if err := os.WriteFile(modelPath, []byte(strings.ReplaceAll(string(data), "HTTPS/JSON", "gRPC")), 0o644); err != nil {
		t.Fatal(err)
	}
	runCLI(t, bin, dir, "sync")
	tech, _ = edgeTechLabels(t, drawioPath)
	for id, value := range tech {
		if value != "[gRPC]" {
			t.Errorf("after the change, label %s = %q, want [gRPC]", id, value)
		}
	}

	// Syncing again changes nothing about the labels or the model.
	before, _ := os.ReadFile(modelPath)
	runCLI(t, bin, dir, "sync")
	after, _ := os.ReadFile(modelPath)
	if string(before) != string(after) {
		t.Error("a second sync changed the model")
	}
	again, _ := edgeTechLabels(t, drawioPath)
	if len(again) != len(tech) {
		t.Errorf("label count changed from %d to %d on a second sync", len(tech), len(again))
	}

	// Without a technology there is no label.
	if err := os.WriteFile(modelPath, []byte(strings.ReplaceAll(string(after), `"technology": "gRPC"`, `"technology": ""`)), 0o644); err != nil {
		t.Fatal(err)
	}
	runCLI(t, bin, dir, "sync")
	if gone, _ := edgeTechLabels(t, drawioPath); len(gone) != 0 {
		t.Errorf("labels left after the technology was removed: %v", gone)
	}
}

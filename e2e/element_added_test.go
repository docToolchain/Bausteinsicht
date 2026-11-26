package e2e

import (
	"strings"
	"testing"

	"github.com/beevik/etree"
)

// TestSyncImportsAnElementAddedInDrawio (#721): a shape with an unknown
// bausteinsicht_id used to be imported as an element of the first kind of the
// specification with an empty title, so the model no longer validated and the
// following sync refused to run until the file was fixed by hand.
func TestSyncImportsAnElementAddedInDrawio(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")
	runCLI(t, bin, dir, "sync")

	doc := etree.NewDocument()
	if err := doc.ReadFromFile(dir + "/architecture.drawio"); err != nil {
		t.Fatal(err)
	}
	root := doc.FindElement("//diagram/mxGraphModel/root")
	if root == nil {
		t.Fatal("setup: no diagram root")
	}
	obj := root.CreateElement("object")
	obj.CreateAttr("id", "x1")
	obj.CreateAttr("label", "Brand New")
	obj.CreateAttr("bausteinsicht_id", "brandnew")
	obj.CreateAttr("bausteinsicht_kind", "container")
	cell := obj.CreateElement("mxCell")
	cell.CreateAttr("style", "rounded=1;")
	cell.CreateAttr("vertex", "1")
	cell.CreateAttr("parent", "1")
	geo := cell.CreateElement("mxGeometry")
	for k, v := range map[string]string{"x": "900", "y": "20", "width": "120", "height": "60", "as": "geometry"} {
		geo.CreateAttr(k, v)
	}
	if err := doc.WriteToFile(dir + "/architecture.drawio"); err != nil {
		t.Fatal(err)
	}

	out := runCLI(t, bin, dir, "sync")
	if !strings.Contains(out, `New element "brandnew" added from draw.io (kind="container")`) {
		t.Errorf("the warning must name the element and the kind it got:\n%s", out)
	}
	if out := runCLI(t, bin, dir, "validate"); !strings.Contains(out, "valid") {
		t.Errorf("validate after the import:\n%s", out)
	}
	if out := runCLI(t, bin, dir, "show", "brandnew"); !strings.Contains(out, "Kind:          container") || !strings.Contains(out, "Title:         Brand New") {
		t.Errorf("the element must keep its kind and title:\n%s", out)
	}
	if out := runCLI(t, bin, dir, "sync"); !strings.Contains(out, "Already in sync") && !strings.Contains(out, "not visible in any view") {
		t.Errorf("the second sync must not fail or change anything:\n%s", out)
	}
}

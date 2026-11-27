package diagram

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// The generated D2 is run through the real compiler (skipped when d2 is not
// installed): a node label with the kind, a description, quote characters, a
// dashed relationship and an element ID with dots and hyphens (#713).
func TestD2_OutputCompiles(t *testing.T) {
	d2, err := exec.LookPath("d2")
	if err != nil {
		t.Skip("d2 is not installed")
	}
	quoted := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements:      map[string]model.ElementKind{"container": {Notation: "Container"}},
			Relationships: map[string]model.RelationshipKind{"async": {Notation: "publishes", Dashed: true}},
		},
		Model: map[string]model.Element{
			"back-office": {Kind: "container", Title: `Back "Office" \ User a\"b`, Description: `says "hi"`,
				Children: map[string]model.Element{"api": {Kind: "container", Title: "API"}}},
		},
		Relationships: []model.Relationship{{From: "back-office", To: "back-office.api", Label: `reads "x"`, Kind: "async"}},
		Views:         map[string]model.View{"v": {Title: "V", Include: []string{"back-office", "back-office.api"}}},
	}
	for name, m := range map[string]*model.BausteinsichtModel{
		"sample": testModel(), "dashed": dashedTestModel(), "quotes and dots": quoted,
	} {
		t.Run(name, func(t *testing.T) {
			for view := range m.Views {
				out, err := RenderD2(m, view)
				if err != nil {
					t.Fatal(err)
				}
				dir := t.TempDir()
				src := filepath.Join(dir, view+".d2")
				if err := os.WriteFile(src, []byte(out), 0o600); err != nil {
					t.Fatal(err)
				}
				if res, err := exec.Command(d2, src, filepath.Join(dir, "o.svg")).CombinedOutput(); err != nil {
					t.Errorf("view %s does not compile: %v\n%s\n%s", view, err, res, out)
				}
			}
		})
	}
}

func TestD2_KindIsPartOfTheLabel(t *testing.T) {
	out, err := RenderD2(testModel(), "context")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasSuffix(line, "{") && strings.Contains(line, `"`) && !strings.HasSuffix(strings.TrimSuffix(line, " {"), `"`) {
			t.Errorf("text after the quoted label: %q", line)
		}
	}
	if !strings.Contains(out, `\n[`) {
		t.Errorf("the kind must be part of the label:\n%s", out)
	}
}

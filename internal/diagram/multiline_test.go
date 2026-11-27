package diagram

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"os"
)

func multilineModel() *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements:      map[string]model.ElementKind{"actor": {Notation: "Actor"}},
			Relationships: map[string]model.RelationshipKind{"uses": {Notation: "uses"}},
		},
		Model: map[string]model.Element{
			"u": {Kind: "actor", Title: "U\nTwo", Description: "line1\nline2 \"q\"\r\nline3", Technology: "Go\nGin"},
			"v": {Kind: "actor", Title: "V"},
		},
		Relationships: []model.Relationship{{From: "u", To: "v", Kind: "uses", Label: "reads\nwrites"}},
		Views:         map[string]model.View{"v": {Title: "V\nview", Include: []string{"*"}}},
	}
}

// A line break inside a quoted macro argument is a syntax error in PlantUML and
// Mermaid, and DOT output must not carry raw breaks either (#690).
func TestExports_NoRawLineBreakInsideQuotedStrings(t *testing.T) {
	for _, f := range []Format{PlantUML, Mermaid} {
		out, err := FormatView(multilineModel(), "v", f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(out, "\n") {
			if strings.Count(line, `"`)%2 != 0 {
				t.Errorf("%v line %d has an unterminated string: %q\n%s", f, i+1, line, out)
			}
		}
	}
	dot, err := RenderDOT(multilineModel(), "v")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(dot, "reads\nwrites") || strings.Contains(dot, "U\nTwo") {
		t.Errorf("DOT output has raw line breaks inside label=:\n%s", dot)
	}
}

func TestExports_LineBreakSpelling(t *testing.T) {
	plant, _ := FormatView(multilineModel(), "v", PlantUML)
	for _, want := range []string{`"line1\nline2 'q'\nline3"`, `"reads\nwrites"`, `"U\nTwo"`, `"Go\nGin"`, "title V\\nview"} {
		if !strings.Contains(plant, want) {
			t.Errorf("PlantUML lacks %s:\n%s", want, plant)
		}
	}
	mer, _ := FormatView(multilineModel(), "v", Mermaid)
	for _, want := range []string{`"line1<br/>line2 'q'<br/>line3"`, `"reads<br/>writes"`, `"U<br/>Two"`, "title V<br/>view"} {
		if !strings.Contains(mer, want) {
			t.Errorf("Mermaid lacks %s:\n%s", want, mer)
		}
	}
	dot, _ := RenderDOT(multilineModel(), "v")
	for _, want := range []string{`label="reads\nwrites"`, `digraph "V\nview"`} {
		if !strings.Contains(dot, want) {
			t.Errorf("DOT lacks %s:\n%s", want, dot)
		}
	}
}

// The real graphviz binary must accept the output.
func TestExports_DOTCompiles(t *testing.T) {
	bin, err := exec.LookPath("dot")
	if err != nil {
		t.Skip("graphviz dot not installed")
	}
	out, err := RenderDOT(multilineModel(), "v")
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "v.dot")
	if err := os.WriteFile(p, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
	if res, err := exec.Command(bin, "-Tsvg", p).CombinedOutput(); err != nil {
		t.Fatalf("dot rejected the output: %v\n%s\n%s", err, res, out)
	}
}

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

// PlantUML sequence diagrams: a line break ends the statement, so titles,
// participants and step labels write it as \n (#690).
func TestSequencePlantUML_LineBreaks(t *testing.T) {
	flat := map[string]*model.Element{
		"u": {Kind: "actor", Title: "User\nTwo"},
		"s": {Kind: "system", Title: "Core"},
	}
	view := model.DynamicView{Key: "f", Title: "Flow\none", Steps: []model.SequenceStep{
		{Index: 1, From: "u", To: "s", Label: "read\r\nwrite"},
		{Index: 4, From: "s", To: "u", Label: "dir\\\nnext"},
		{Index: 2, Type: model.StepAlt, Label: "ok\nor not"},
		{Index: 3, Type: model.StepEnd},
	}}
	out := RenderSequencePlantUML(view, flat, &model.Specification{})
	for _, want := range []string{"title Flow\\none", `"User\nTwo"`, ": read\\nwrite", "alt ok\\nor not", `: dir\\\nnext`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "\r") {
		t.Errorf("carriage return left in:\n%s", out)
	}
	for i, line := range strings.Split(out, "\n") {
		if line != "" && !strings.HasPrefix(line, "@") && !strings.Contains(line, " ") && line != "autonumber" && line != "end" {
			t.Errorf("line %d looks like the tail of a broken statement: %q", i+1, line)
		}
	}
}

// A literal backslash in front of a line break must not swallow the break: in
// DOT and PlantUML "\\" is a backslash, so the backslash is doubled before the
// \n marker is added (CodeRabbit finding on #754).
func TestExports_BackslashBeforeLineBreak(t *testing.T) {
	m := multilineModel()
	e := m.Model["u"]
	e.Description = "A\\\nB"
	m.Model["u"] = e
	m.Relationships[0].Label = "C:\\dir\\\nnext"

	plant, _ := FormatView(m, "v", PlantUML)
	if !strings.Contains(plant, `"A\\\nB"`) || !strings.Contains(plant, `"C:\\dir\\\nnext"`) {
		t.Errorf("PlantUML does not double the backslash before the break:\n%s", plant)
	}
	dot, _ := RenderDOT(m, "v")
	if !strings.Contains(dot, `label="C:\\dir\\\nnext"`) {
		t.Errorf("DOT does not double the backslash before the break:\n%s", dot)
	}
	mer, _ := FormatView(m, "v", Mermaid)
	if !strings.Contains(mer, `"A\<br/>B"`) {
		t.Errorf("Mermaid should keep the backslash as it is:\n%s", mer)
	}
}

// Rendered by graphviz the text keeps the backslash and still breaks the line.
func TestExports_DOTBackslashBeforeLineBreakRenders(t *testing.T) {
	bin, err := exec.LookPath("dot")
	if err != nil {
		t.Skip("graphviz dot not installed")
	}
	m := multilineModel()
	m.Relationships[0].Label = "A\\\nB"
	out, _ := RenderDOT(m, "v")
	p := filepath.Join(t.TempDir(), "v.dot")
	if err := os.WriteFile(p, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
	svg, err := exec.Command(bin, "-Tsvg", p).CombinedOutput()
	if err != nil {
		t.Fatalf("dot failed: %v\n%s", err, svg)
	}
	if strings.Contains(string(svg), `A\\nB`) || !strings.Contains(string(svg), `>A\</text>`) {
		t.Errorf("the break after the backslash was lost:\n%s", svg)
	}
}

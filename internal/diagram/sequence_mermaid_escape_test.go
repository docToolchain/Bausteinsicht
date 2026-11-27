package diagram

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func semicolonView() (model.DynamicView, map[string]*model.Element) {
	flat := map[string]*model.Element{
		"u": {Kind: "actor", Title: "Orders; Payments"},
		"s": {Kind: "system", Title: "Core"},
	}
	return model.DynamicView{Key: "f", Title: "Flow; one", Steps: []model.SequenceStep{
		{Index: 1, From: "u", To: "s", Label: "read; write"},
		{Index: 2, Type: model.StepAlt, Label: "ok; or not"},
		{Index: 3, From: "s", To: "u", Label: "done"},
		{Index: 4, Type: model.StepEnd},
	}}, flat
}

// In Mermaid a ";" ends a statement: titles, participant aliases and labels with
// one are a parse error in mmdc and on GitHub (#714). It is written as the entity
// "#59;".
func TestSequenceMermaid_EscapesSemicolons(t *testing.T) {
	view, flat := semicolonView()
	out := RenderSequenceMermaid(view, flat, &model.Specification{})
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, ";") && !strings.Contains(line, "#59;") {
			t.Errorf("unescaped semicolon: %q", line)
		}
		if strings.Count(line, ";") != strings.Count(line, "#59;") {
			t.Errorf("a raw semicolon remains in %q", line)
		}
	}
	for _, want := range []string{"title Flow#59; one", "as Orders#59; Payments", ": read#59; write", "alt ok#59; or not"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

// The plain output must not change for text without a semicolon.
func TestSequenceMermaid_PlainTextIsUnchanged(t *testing.T) {
	view := model.DynamicView{Key: "f", Title: "Flow", Steps: []model.SequenceStep{{Index: 1, From: "u", To: "s", Label: "reads, writes"}}}
	out := RenderSequenceMermaid(view, map[string]*model.Element{"u": {Title: "U"}, "s": {Title: "S"}}, &model.Specification{})
	if !strings.Contains(out, "u->>s: reads, writes") && !strings.Contains(out, ": reads, writes") {
		t.Errorf("plain label changed:\n%s", out)
	}
}

// The escaped output is rendered with mmdc when it can run (skipped when mmdc is
// missing or cannot start its browser here).
func TestSequenceMermaid_SemicolonOutputRendersWithMmdc(t *testing.T) {
	mmdc, err := exec.LookPath("mmdc")
	if err != nil {
		t.Skip("mmdc is not installed")
	}
	dir := t.TempDir()
	render := func(name, text string) error {
		src := filepath.Join(dir, name+".mmd")
		if err := os.WriteFile(src, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
		return exec.Command(mmdc, "-i", src, "-o", filepath.Join(dir, name+".svg")).Run()
	}
	if render("baseline", "sequenceDiagram\n    participant a as A\n    a->>a: hi\n") != nil {
		t.Skip("mmdc cannot render here (no browser)")
	}
	view, flat := semicolonView()
	if err := render("semicolon", RenderSequenceMermaid(view, flat, &model.Specification{})); err != nil {
		t.Errorf("mmdc rejects the output: %v", err)
	}
}

// A line break in a title or label ends the statement as well; Mermaid writes it
// as <br/>.
func TestSequenceMermaid_LineBreaksStayInsideTheStatement(t *testing.T) {
	view := model.DynamicView{Key: "f", Title: "Flow", Steps: []model.SequenceStep{{Index: 1, From: "u", To: "s", Label: "reads\nwrites"}}}
	flat := map[string]*model.Element{"u": {Title: "Two\nLines"}, "s": {Title: "S"}}
	out := RenderSequenceMermaid(view, flat, &model.Specification{})
	for _, want := range []string{": reads<br/>writes", "as Two<br/>Lines"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	for _, line := range strings.Split(out, "\n") {
		if line != "" && !strings.HasPrefix(line, " ") && line != "sequenceDiagram" {
			t.Errorf("a line break escaped into a new statement: %q", line)
		}
	}
}

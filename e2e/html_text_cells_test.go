package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const htmlTextModel = `{"specification":{"elements":{"actor":{"notation":"Actor"}},"relationships":{"uses":{"notation":"uses"}}},
 "model":{"u":{"kind":"actor","title":"R&D <Team>","description":"a < b & c"},"v":{"kind":"actor","title":"V"}},
 "relationships":[{"from":"u","to":"v","kind":"uses","label":"x < y & z"}],
 "views":{"v":{"title":"V","include":["*"]}}}`

// TestHTMLCharactersInTextCells (#699): the text cells have html=1, so "<Team>"
// was read as a tag by draw.io. The text is written escaped and a text edited in
// draw.io comes back unescaped into the model.
func TestHTMLCharactersInTextCells(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "m.jsonc"), htmlTextModel)
	runCLI(t, bin, dir, "--model", "m.jsonc", "sync")

	drawio := filepath.Join(dir, "architecture.drawio")
	data, err := os.ReadFile(drawio)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	// The XML attribute holds the HTML text, itself XML-escaped: "&amp;amp;" is "&amp;" is "&".
	for _, want := range []string{`R&amp;amp;D &amp;lt;Team&amp;gt;`, `x &amp;lt; y &amp;amp; z`} {
		if !strings.Contains(text, want) {
			t.Errorf("draw.io file lacks %q", want)
		}
	}

	// A second sync changes neither side.
	out := runCLI(t, bin, dir, "--model", "m.jsonc", "sync")
	if strings.Contains(out, "Reverse") {
		t.Errorf("the second sync imported something from draw.io:\n%s", out)
	}
	model, _ := os.ReadFile(filepath.Join(dir, "m.jsonc"))
	if !strings.Contains(string(model), `"title":"R&D <Team>"`) && !strings.Contains(string(model), `"title": "R&D <Team>"`) {
		t.Errorf("the model title changed:\n%s", model)
	}

	// Edit the title in draw.io: the value is HTML text.
	edited := strings.Replace(text, `R&amp;amp;D &amp;lt;Team&amp;gt;`, `Q&amp;amp;A &amp;lt;Crew&amp;gt;`, 1)
	if edited == text {
		t.Fatal("found no title cell to edit")
	}
	if err := os.WriteFile(drawio, []byte(edited), 0o600); err != nil {
		t.Fatal(err)
	}
	runCLI(t, bin, dir, "--model", "m.jsonc", "sync")
	model, _ = os.ReadFile(filepath.Join(dir, "m.jsonc"))
	if !strings.Contains(string(model), "Q&A <Crew>") {
		t.Errorf("the title edited in draw.io did not reach the model as plain text:\n%s", model)
	}
}

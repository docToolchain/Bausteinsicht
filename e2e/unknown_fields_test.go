package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #725: validate reports misspelt keys instead of dropping them silently.
func TestValidateWarnsAboutUnknownFields(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "m.jsonc")
	if err := os.WriteFile(path, []byte(`{"specification":{"elements":{"a":{"notation":"A"}}},"modle":{},
 "model":{"x":{"kind":"a","title":"X","titel":"typo","tecnology":"Go"}},
 "views":{"v":{"title":"V","include":["*"]}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	out, code := runCLIAllowFail(t, bin, dir, "--model", path, "validate")
	if code != 0 {
		t.Errorf("unknown fields are warnings, exit %d:\n%s", code, out)
	}
	for _, want := range []string{
		`unknown top-level key "modle" (did you mean "model"?)`,
		`WARNING: [model.x] unknown field "titel" (did you mean "title"?)`,
		`unknown field "tecnology" (did you mean "technology"?)`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("validate output lacks %q:\n%s", want, out)
		}
	}

	out, _ = runCLIAllowFail(t, bin, dir, "--model", path, "--format", "json", "validate")
	if !strings.Contains(out, `"warnings"`) || !strings.Contains(out, "titel") {
		t.Errorf("the JSON output carries no warnings:\n%s", out)
	}
}

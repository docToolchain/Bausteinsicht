package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #726: tags, lastModified, staleDetection and docLinks types were not checked.
func TestValidateChecksTagsDatesAndDocLinks(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "m.jsonc")
	base := `{"specification":{"elements":{"a":{"notation":"A"}},"tags":[{"id":"pii"},{"id":"public"}]},
 "meta":{"staleDetection":{"thresholdDays":30}},
 "model":{"x":{"kind":"a","title":"X","tags":["pii","public"],"lastModified":"2020-01-01T00:00:00Z","docLinks":[{"type":"adr","href":"a.html"}]}},
 "views":{"v":{"title":"V","include":["*"]}}}`
	write := func(text string) {
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(base)
	if out, code := runCLIAllowFail(t, bin, dir, "--model", path, "validate"); code != 0 || strings.Contains(out, "WARNING") {
		t.Fatalf("the base model must be clean, exit %d:\n%s", code, out)
	}

	for _, tc := range []struct {
		name, from, to, want string
		code                 int
	}{
		{"element tag", `"tags":["pii","public"]`, `"tags":["undefined-tag"]`, `tag "undefined-tag" is not defined`, 1},
		{"lastModified", `"2020-01-01T00:00:00Z"`, `"yesterday"`, `not an RFC3339 timestamp`, 1},
		{"thresholdDays", `"thresholdDays":30`, `"thresholdDays":-5`, `non-negative integer`, 1},
		{"docLinks type", `"type":"adr"`, `"type":"weird"`, `unknown docLinks type "weird"`, 0},
	} {
		write(strings.Replace(base, tc.from, tc.to, 1))
		out, code := runCLIAllowFail(t, bin, dir, "--model", path, "validate")
		if code != tc.code || !strings.Contains(out, tc.want) {
			t.Errorf("%s: exit %d (want %d), output lacks %q:\n%s", tc.name, code, tc.code, tc.want, out)
		}
	}
}

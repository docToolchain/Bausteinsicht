package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #689: validate accepted unknown constraint rules and duplicate constraint
// IDs, and lint printed ": :" for a constraint without a description.
func TestValidateChecksConstraints(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "m.jsonc")
	if err := os.WriteFile(path, []byte(`{"specification":{"elements":{"service":{"notation":"Service"}}},
 "model":{"a":{"kind":"service","title":"A"}},
 "constraints":[{"id":"c1","rule":"bogus-rule"},{"id":"c1","rule":"required-field","element-kind":"service","field":"description"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	out, code := runCLIAllowFail(t, bin, dir, "--model", path, "validate")
	if code == 0 {
		t.Errorf("validate accepted the model:\n%s", out)
	}
	for _, want := range []string{`unknown constraint rule "bogus-rule"`, `duplicate constraint id "c1"`} {
		if !strings.Contains(out, want) {
			t.Errorf("validate output lacks %q:\n%s", want, out)
		}
	}

	// With the two mistakes fixed lint reports the missing description without
	// an empty prefix.
	fixed := strings.Replace(strings.Replace(readFile(t, path), "bogus-rule", "max-depth", 1), `"id":"c1","rule":"required-field"`, `"id":"c2","rule":"required-field"`, 1)
	if err := os.WriteFile(path, []byte(fixed), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _ = runCLIAllowFail(t, bin, dir, "--model", path, "lint")
	if !strings.Contains(out, "VIOLATION [c2]") || strings.Contains(out, ": :") {
		t.Errorf("lint output:\n%s", out)
	}
}

// #727: the model of the issue: partial selectors must be reported by validate
// and lint, the tag selector of technology-allowed must work, and violations
// name the full element ID.
func TestLintReportsIncompleteConstraints(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "m.jsonc")
	write := func(constraints string) {
		text := `{"specification":{"elements":{"system":{"notation":"S","container":true},"db":{"notation":"DB"}},"tags":[{"id":"pii"}]},
 "model":{"s":{"kind":"system","title":"S","children":{"db":{"kind":"db","title":"DB","technology":"MongoDB","tags":["pii"]}}}},
 "constraints":[` + constraints + `]}`
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write(`{"id":"no-from","rule":"no-relationship","to-kind":"db"},{"id":"depth0","rule":"max-depth"}`)
	out, code := runCLIAllowFail(t, bin, dir, "--model", path, "validate")
	if code == 0 || !strings.Contains(out, "from selector") || !strings.Contains(out, "at least 1") {
		t.Errorf("validate (exit %d):\n%s", code, out)
	}
	out, _ = runCLIAllowFail(t, bin, dir, "--model", path, "lint")
	if !strings.Contains(out, "VIOLATION [no-from]") || !strings.Contains(out, "VIOLATION [depth0]") || strings.Contains(out, "s.db (depth") {
		t.Errorf("lint should report the incomplete constraints, not every element:\n%s", out)
	}

	write(`{"id":"tech-tag","rule":"technology-allowed","tag":"pii","technologies":["Postgres"]},{"id":"depth","rule":"max-depth","max":1}`)
	out, _ = runCLIAllowFail(t, bin, dir, "--model", path, "lint")
	if !strings.Contains(out, "VIOLATION [tech-tag]") || !strings.Contains(out, "s.db: technology") || !strings.Contains(out, "s.db (depth 2)") {
		t.Errorf("lint output:\n%s", out)
	}
}

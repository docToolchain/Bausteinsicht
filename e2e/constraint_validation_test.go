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

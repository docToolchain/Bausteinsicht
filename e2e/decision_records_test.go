package e2e

// #707: validate reports decision records that are wrong (status, date, a file
// that does not exist, a file outside the project) and adr show says that a file
// does not exist.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const decisionModelFile = `{"specification":{"elements":{"system":{"notation":"S","container":true}},
  "decisions":[{"id":"ADR-001","title":"Use PG","status":"accepted","date":"2026-01-01","file":"docs/adr1.adoc"},
               {"id":"ADR-002","title":"Missing file","status":"proposed","date":"bad-date","file":"docs/nope.adoc"},
               {"id":"ADR-003","title":"Weird","status":"foo"}]},
 "model":{"s":{"kind":"system","title":"S","decisions":["ADR-001","ADR-002","ADR-003"]}},"views":{"v":{"title":"V","include":["*"]}}}`

func TestDecisionRecords(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o750); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"docs/adr1.adoc": "= ADR 1", "m.jsonc": decisionModelFile} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	out := runCLI(t, bin, dir, "--model", "m.jsonc", "validate")
	for _, want := range []string{`"foo"`, `"bad-date"`, "docs/nope.adoc"} {
		if !strings.Contains(out, want) {
			t.Errorf("validate must report %s:\n%s", want, out)
		}
	}
	if strings.Contains(out, "ADR-001") && strings.Contains(out, `"accepted"`) {
		t.Errorf("the accepted ADR-001 is fine:\n%s", out)
	}

	list := runCLI(t, bin, dir, "--model", "m.jsonc", "adr", "list")
	if line := lineWith(list, "ADR-001"); strings.Contains(line, "?") {
		t.Errorf("an accepted decision has an icon, not ?: %q", line)
	}
	show := runCLI(t, bin, dir, "--model", "m.jsonc", "adr", "show", "ADR-002")
	if !strings.Contains(show, "not found") {
		t.Errorf("adr show must say that the file does not exist:\n%s", show)
	}

	bad := strings.Replace(decisionModelFile, `{"id":"ADR-003","title":"Weird","status":"foo"}`, `{"id":"ADR-003","title":"Out","file":"../etc/passwd"}`, 1)
	if err := os.WriteFile(filepath.Join(dir, "bad.jsonc"), []byte(bad), 0o600); err != nil {
		t.Fatal(err)
	}
	o, code := runCLIAllowFail(t, bin, dir, "--model", "bad.jsonc", "validate")
	if code != 1 || !strings.Contains(o, "outside the project") {
		t.Errorf("a file outside the project: want exit 1 and \"outside the project\", got %d:\n%s", code, o)
	}
}

func lineWith(text, sub string) string {
	for _, l := range strings.Split(text, "\n") {
		if strings.Contains(l, sub) {
			return l
		}
	}
	return ""
}

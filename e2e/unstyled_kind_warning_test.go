package e2e

// #694: kinds that the template has no style for give one warning per kind,
// with the number of elements and the hint what to do.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const unstyledKindModel = `{"specification":{"elements":{"service":{"notation":"Service"}}},
 "model":{"a":{"kind":"service","title":"A"},"b":{"kind":"service","title":"B"},"c":{"kind":"service","title":"C"}},
 "views":{"v1":{"title":"1","include":["*"]},"v2":{"title":"2","include":["*"]}}}`

func TestSync_UnstyledKindWarnsOncePerKind(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "m.jsonc"), []byte(unstyledKindModel), 0o600); err != nil {
		t.Fatal(err)
	}

	out := runCLI(t, bin, dir, "--model", "m.jsonc", "sync")
	if got := strings.Count(out, "no template style for kind: service"); got != 1 {
		t.Errorf("want one warning for the kind, got %d:\n%s", got, out)
	}
	if !strings.Contains(out, "3 elements") || !strings.Contains(out, "generate-template") {
		t.Errorf("the warning must give the number of elements and the hint:\n%s", out)
	}
}

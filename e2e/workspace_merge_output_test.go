package e2e

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestWorkspaceMergeProtectsItsOutput (#708): workspace merge replaced an existing
// output silently and accepted the workspace config as the output, which
// destroyed the workspace definition.
func TestWorkspaceMergeProtectsItsOutput(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	for _, sub := range []string{"a", "b"} {
		runCLI(t, bin, filepath.Join(mkdir(t, dir, sub)), "init")
		if sub == "b" {
			renameSampleConstraintIDs(t, filepath.Join(dir, sub, "architecture.jsonc"))
		}
	}
	cfg := `{"workspace":{"name":"ws"},"models":[
  {"id":"a","path":"a/architecture.jsonc","prefix":"a"},
  {"id":"b","path":"b/architecture.jsonc","prefix":"b"}]}`
	writeFile(t, dir+"/ws.jsonc", cfg)

	runCLI(t, bin, dir, "workspace", "merge", "ws.jsonc", "out.jsonc")
	if out, code := runCLIAllowFail(t, bin, dir, "workspace", "merge", "ws.jsonc", "out.jsonc"); code != 2 || !strings.Contains(out, "--force") {
		t.Errorf("a second merge into the same output: exit %d\n%s", code, out)
	}
	runCLI(t, bin, dir, "workspace", "merge", "ws.jsonc", "out.jsonc", "--force")

	if out, code := runCLIAllowFail(t, bin, dir, "workspace", "merge", "ws.jsonc", "ws.jsonc", "--force"); code != 2 || !strings.Contains(out, "would overwrite") {
		t.Errorf("the config as the output: exit %d\n%s", code, out)
	}
	if data, _ := os.ReadFile(dir + "/ws.jsonc"); string(data) != cfg {
		t.Errorf("the workspace config was changed:\n%s", data)
	}
	runCLI(t, bin, dir, "workspace", "merge", "ws.jsonc", "out2.jsonc") // the workspace still works
}

func mkdir(t *testing.T, dir, sub string) string {
	t.Helper()
	p := filepath.Join(dir, sub)
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// renameSampleConstraintIDs gives the constraints of an init model other IDs,
// so that two of them can be merged: constraint IDs must be unique (#689).
func renameSampleConstraintIDs(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	renamed := regexp.MustCompile(`("id":\s*)"C0(\d)"`).ReplaceAll(data, []byte(`${1}"B0${2}"`))
	if string(renamed) == string(data) {
		t.Fatal("no constraint IDs found to rename")
	}
	if err := os.WriteFile(path, renamed, 0o644); err != nil {
		t.Fatal(err)
	}
}

package e2e

import (
	"os"
	"path/filepath"
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

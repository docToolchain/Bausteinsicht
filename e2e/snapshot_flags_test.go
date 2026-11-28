package e2e

// #693: snapshot list follows the global --format, snapshot save has no flag of
// its own that shadows the global --model, and index.json is not a snapshot.

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSnapshotFlagsAndIndex(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")
	runCLI(t, bin, dir, "snapshot", "save", "--message", "a")

	out := runCLI(t, bin, dir, "snapshot", "list", "--format", "json")
	var list []map[string]any
	if err := json.Unmarshal([]byte(out), &list); err != nil || len(list) != 1 {
		t.Errorf("snapshot list --format json: want a JSON list with one snapshot, got %q (%v)", out, err)
	}

	help := runCLI(t, bin, dir, "snapshot", "save", "--help")
	own, global, found := strings.Cut(help, "Global Flags:")
	if !found || strings.Contains(own, "--model") || !strings.Contains(global, "--model") {
		t.Errorf("--model must be listed under the global flags only:\n%s", help)
	}

	for _, args := range [][]string{{"diff", "index"}, {"delete", "index"}, {"restore", "index", "out.jsonc"}} {
		o, code := runCLIAllowFail(t, bin, dir, append([]string{"snapshot"}, args...)...)
		if code != 2 || !strings.Contains(o, "snapshot not found: index") {
			t.Errorf("snapshot %v: want exit 2 and \"snapshot not found: index\", got %d:\n%s", args, code, o)
		}
	}
	out = runCLI(t, bin, dir, "snapshot", "list", "--format", "json")
	if err := json.Unmarshal([]byte(out), &list); err != nil || len(list) != 1 {
		t.Errorf("the index must survive: %q (%v)", out, err)
	}
}

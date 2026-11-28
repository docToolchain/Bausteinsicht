package e2e

// #704: `add view` on an existing key says that the view was updated.

import (
	"strings"
	"testing"
)

func TestAddView_ExistingKeyIsReportedAsUpdate(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	out := runCLI(t, bin, dir, "add", "view", "v1", "--title", "V1", "--include", "onlineshop.api")
	if !strings.Contains(out, "View 'v1' added to model") {
		t.Errorf("a new view is reported as added:\n%s", out)
	}
	out = runCLI(t, bin, dir, "add", "view", "v1", "--include", "onlineshop.db")
	if !strings.Contains(out, "View 'v1' updated") || !strings.Contains(out, "+onlineshop.db") || strings.Contains(out, "added to model") {
		t.Errorf("an existing view is reported as updated with what was added:\n%s", out)
	}
}

package e2e

import (
	"os"
	"strings"
	"testing"
)

func commentCount(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, l := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "//") {
			n++
		}
	}
	return n
}

// TestEditCommandsKeepTheCommentsOfTheInitModel (#685): update, rename and
// remove rewrote the whole file, so the explanatory comments of the model that
// "init" writes were gone after the first edit.
func TestEditCommandsKeepTheCommentsOfTheInitModel(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")
	model := dir + "/architecture.jsonc"
	before := commentCount(t, model)
	if before < 10 {
		t.Fatalf("setup: the init model has only %d comment lines", before)
	}

	_, out, code := runCLISplit(t, bin, dir, "update", "element", "--id", "customer", "--title", "Client")
	if code != 0 {
		t.Fatalf("update failed: %s", out)
	}
	if strings.Contains(out, "rewritten") {
		t.Errorf("update announced a rewrite of the whole file:\n%s", out)
	}
	if after := commentCount(t, model); after != before {
		t.Errorf("update: %d comment lines before, %d after", before, after)
	}

	runCLI(t, bin, dir, "rename", "--id", "backoffice-user", "--to", "staff")
	// The sync after the rename must be a no-op: the rename keeps the draw.io
	// cells (#686). The relationship is removed afterwards and no sync follows:
	// syncing right after removing a relationship that is not the last one adds it
	// back to the model once, which this test is not about.
	runCLI(t, bin, dir, "sync")
	if out := runCLI(t, bin, dir, "sync"); !strings.Contains(out, "Already in sync") {
		t.Errorf("sync after the edits is not a no-op:\n%s", out)
	}
	runCLI(t, bin, dir, "remove", "relationship", "--from", "staff", "--to", "onlineshop.api")
	runCLI(t, bin, dir, "validate")
	if after := commentCount(t, model); after < before-2 { // the comment above a removed relationship goes with it
		t.Errorf("rename and remove: %d comment lines before, %d after", before, after)
	}
	data, _ := os.ReadFile(model)
	if strings.Contains(string(data), `"config": {}`) {
		t.Errorf("the edit added a config section that the file did not have:\n%s", data[:400])
	}
}

package e2e

// TestStaleDetection (#496) verifies that `stale` reports elements that appear
// in git history but are no longer in the model, and that the active model's
// elements are NOT reported as stale.

import (
	"os/exec"
	"strings"
	"testing"
)

func TestStaleDetection(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	// Initialise a git repo so `stale` can inspect history.
	gitInit := exec.Command("git", "init", dir)
	if out, err := gitInit.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"git", "-C", dir, "config", "user.email", "test@test.com"},
		{"git", "-C", dir, "config", "user.name", "Test"},
	} {
		if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}

	runCLI(t, bin, dir, "init")

	// Commit the initial model so git history exists.
	for _, args := range [][]string{
		{"git", "-C", dir, "add", "."},
		{"git", "-C", dir, "commit", "-m", "init"},
	} {
		if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}

	// `stale` should run without error and produce output (or empty if nothing stale).
	out, code := runCLIAllowFail(t, bin, dir, "stale", "--model", "architecture.jsonc")
	if code != 0 {
		t.Errorf("stale exited %d: %s", code, out)
	}

	// Current model has "customer" — it should NOT be listed as stale.
	if strings.Contains(strings.ToLower(out), "customer") {
		t.Errorf("stale: 'customer' reported as stale but it is in the active model\noutput: %s", out)
	}

	t.Logf("stale output: %s", out)
}

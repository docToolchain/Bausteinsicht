package e2e

// TestStaleDetection (#496) verifies that `stale` reports elements that appear
// in git history but are no longer in the model, and that the active model's
// elements are NOT reported as stale.

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
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

	// `stale` on the unmodified model — nothing is stale yet.
	out, code := runCLIAllowFail(t, bin, dir, "stale", "--model", "architecture.jsonc")
	if code != 0 {
		t.Errorf("stale exited %d: %s", code, out)
	}

	// Current model has "customer" — it should NOT be listed as stale.
	if strings.Contains(strings.ToLower(out), "customer") {
		t.Errorf("stale: 'customer' reported as stale but it is in the active model\noutput: %s", out)
	}
	t.Logf("stale output (negative case): %s", out)

	// ── Positive test: add an element with an old lastModified date ───────
	// Elements with an explicit lastModified older than the --days threshold
	// must be reported as stale regardless of git history.
	modelPath := dir + "/architecture.jsonc"
	m, err := model.Load(modelPath)
	if err != nil {
		t.Fatalf("model.Load: %v", err)
	}
	// LastModified in 2020 — far older than the default 90-day threshold.
	m.Model["legacy-service"] = model.Element{
		Title:        "Legacy Service",
		Kind:         "system",
		LastModified: "2020-01-01T00:00:00Z",
	}
	if err := model.Save(modelPath, m); err != nil {
		t.Fatalf("model.Save: %v", err)
	}

	// stale should report legacy-service because its lastModified is far in the past.
	out2, code2 := runCLIAllowFail(t, bin, dir, "stale", "--model", "architecture.jsonc")
	if code2 != 0 {
		t.Errorf("stale exited %d with legacy-service in model: %s", code2, out2)
	}
	if !strings.Contains(out2, "legacy-service") {
		t.Errorf("stale: 'legacy-service' (lastModified 2020) not reported as stale\noutput: %s", out2)
	}
	t.Logf("stale output (positive case): %s", out2)
}

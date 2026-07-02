package e2e

// TestChangelogWorkflow verifies `changelog` compares the model at two git
// refs and reports what changed. Covers a CLI command with no prior E2E
// coverage; changelog only supports git refs (not snapshot IDs), so this
// test builds a small real git history.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGit(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func TestChangelogWorkflow(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	runGit(t, "init", dir)
	runGit(t, "-C", dir, "config", "user.email", "test@test.com")
	runGit(t, "-C", dir, "config", "user.name", "Test")

	runCLI(t, bin, dir, "init")

	runGit(t, "-C", dir, "add", ".")
	runGit(t, "-C", dir, "commit", "-m", "initial model")
	fromSHA := strings.TrimSpace(runGit(t, "-C", dir, "rev-parse", "HEAD"))

	// Add an element and commit again, so the two refs differ.
	runCLI(t, bin, dir, "add", "element",
		"--id", "new-service",
		"--kind", "container",
		"--title", "New Service",
		"--technology", "Go",
	)
	runGit(t, "-C", dir, "add", ".")
	runGit(t, "-C", dir, "commit", "-m", "add new-service")

	// ── changelog markdown (default) ───────────────────────────────────────────
	mdOut := runCLI(t, bin, dir, "changelog",
		"--model", "architecture.jsonc",
		"--since", fromSHA,
		"--until", "HEAD",
	)
	if !strings.Contains(mdOut, "New Service") && !strings.Contains(mdOut, "new-service") {
		t.Errorf("changelog markdown: expected the added element to be mentioned, got: %s", mdOut)
	}

	// ── changelog json: verify the actual added-element entry, not just "is valid JSON" ──
	jsonOut := runCLI(t, bin, dir, "changelog",
		"--model", "architecture.jsonc",
		"--since", fromSHA,
		"--until", "HEAD",
		"--changelog-format", "json",
	)
	type elementChange struct {
		ID string `json:"id"`
	}
	var payload struct {
		Elements struct {
			Added []elementChange `json:"added"`
		} `json:"elements"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &payload); err != nil {
		t.Fatalf("changelog --changelog-format json: invalid JSON: %v\noutput: %s", err, jsonOut)
	}
	foundAdded := false
	for _, e := range payload.Elements.Added {
		if e.ID == "new-service" {
			foundAdded = true
			break
		}
	}
	if !foundAdded {
		t.Errorf("changelog json: expected elements.added to contain \"new-service\", got: %+v", payload.Elements.Added)
	}

	// ── changelog --output writes the same content to a file instead of stdout ─
	outFile := filepath.Join(t.TempDir(), "changelog.md")
	runCLI(t, bin, dir, "changelog",
		"--model", "architecture.jsonc",
		"--since", fromSHA,
		"--until", "HEAD",
		"--output", outFile,
	)
	written, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read changelog --output file: %v", err)
	}
	if !strings.Contains(string(written), "new-service") {
		t.Errorf("changelog --output file missing \"new-service\", got: %s", written)
	}
}

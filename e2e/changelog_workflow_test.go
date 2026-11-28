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

// #698: without a git repository or a tag the error says what is missing and what
// to do, and with a tag the default of --since is the previous tag.
func TestChangelogErrorsAndDefaultSince(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runCLI(t, bin, dir, "init")

	out, code := runCLIAllowFail(t, bin, dir, "changelog")
	if code != 2 || !strings.Contains(out, "no git repository found") || strings.Contains(out, "exit status") {
		t.Errorf("outside a repository: want exit 2 and \"no git repository found\", got %d:\n%s", code, out)
	}

	runGit(t, "init", dir)
	runGit(t, "-C", dir, "config", "user.email", "test@test.com")
	runGit(t, "-C", dir, "config", "user.name", "Test")
	runGit(t, "-C", dir, "add", ".")
	runGit(t, "-C", dir, "commit", "-m", "initial model")
	out, code = runCLIAllowFail(t, bin, dir, "changelog")
	if code != 2 || !strings.Contains(out, "no tag found") || !strings.Contains(out, "--since") {
		t.Errorf("without a tag: want exit 2, \"no tag found\" and the hint, got %d:\n%s", code, out)
	}

	runGit(t, "-C", dir, "tag", "v1")
	runCLI(t, bin, dir, "add", "element", "--id", "late", "--kind", "container", "--title", "Late Service")
	runGit(t, "-C", dir, "commit", "-am", "add late")
	md := runCLI(t, bin, dir, "changelog")
	if !strings.Contains(md, "Late Service") {
		t.Errorf("the default --since is the previous tag v1, so the added element must show:\n%s", md)
	}
}

// #728: a snapshot ID is accepted as --since (and --until), next to git refs.
func TestChangelogResolvesSnapshotIDs(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	runGit(t, "init", dir)
	runGit(t, "-C", dir, "config", "user.email", "test@test.com")
	runGit(t, "-C", dir, "config", "user.name", "Test")
	runCLI(t, bin, dir, "init")
	runGit(t, "-C", dir, "add", ".")
	runGit(t, "-C", dir, "commit", "-m", "initial model")

	saved := runCLI(t, bin, dir, "snapshot", "save", "--message", "a")
	var id string
	for _, line := range strings.Split(saved, "\n") {
		if v, ok := strings.CutPrefix(line, "Snapshot saved: "); ok {
			id = strings.TrimSpace(v)
		}
	}
	if id == "" {
		t.Fatalf("no snapshot ID in:\n%s", saved)
	}
	runCLI(t, bin, dir, "add", "element", "--id", "newone", "--kind", "actor", "--title", "Brand New")
	runGit(t, "-C", dir, "commit", "-am", "add newone")

	out := runCLI(t, bin, dir, "changelog", "--since", id)
	if !strings.Contains(out, "Brand New") && !strings.Contains(out, "newone") {
		t.Errorf("a snapshot as --since and HEAD as --until must show the added element:\n%s", out)
	}
}

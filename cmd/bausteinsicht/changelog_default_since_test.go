package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func gitHere(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// Without --since the changelog starts at the previous tag; without a tag the
// error says so (#698).
func TestChangelogCmd_DefaultSinceIsThePreviousTag(t *testing.T) {
	t.Chdir(t.TempDir())
	model := func(title string) {
		content := `{"specification":{"elements":{"system":{"notation":"System"}}},"model":{"a":{"kind":"system","title":"` + title + `"}}}`
		if err := os.WriteFile("architecture.jsonc", []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	gitHere(t, "init", "-q")
	model("Before")
	gitHere(t, "add", ".")
	gitHere(t, "commit", "-q", "-m", "one")

	_, err := executeImportCmd("changelog")
	if err == nil || !strings.Contains(err.Error(), "no tag found") {
		t.Fatalf("without a tag: want \"no tag found\", got %v", err)
	}

	gitHere(t, "tag", "v1")
	model("After")
	gitHere(t, "commit", "-q", "-am", "two")
	out, err := executeImportCmd("changelog")
	if err != nil || !strings.Contains(out, "After") {
		t.Errorf("want the change since v1 in the changelog, got %q (%v)", out, err)
	}
}

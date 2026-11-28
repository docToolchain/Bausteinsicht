package changelog

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func commitChange(t *testing.T, dir, title string) {
	t.Helper()
	content := `{"specification":{"elements":{"system":{"notation":"System"}}},"model":{"a":{"kind":"system","title":"` + title + `"}}}`
	if err := os.WriteFile(filepath.Join(dir, "architecture.jsonc"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "commit", "-q", "-am", title)
}

// The default of --since is the previous tag: the latest tag before until; if until
// itself is tagged, the one before that (#698).
func TestPreviousTag(t *testing.T) {
	dir := setupGitRepo(t)
	chdir(t, dir)
	gitIn(t, dir, "tag", "v1")
	commitChange(t, dir, "B")

	if got, err := PreviousTag("HEAD"); err != nil || got != "v1" {
		t.Errorf("an untagged HEAD: want v1, got %q (%v)", got, err)
	}
	gitIn(t, dir, "tag", "v2")
	if got, err := PreviousTag("HEAD"); err != nil || got != "v1" {
		t.Errorf("a tagged HEAD: want the tag before it, v1, got %q (%v)", got, err)
	}
}

func TestPreviousTag_NoTag(t *testing.T) {
	chdir(t, setupGitRepo(t))
	_, err := PreviousTag("HEAD")
	if err == nil || !strings.Contains(err.Error(), "no tag found") || !strings.Contains(err.Error(), "--since") {
		t.Errorf("want \"no tag found\" with the hint to use --since, got %v", err)
	}
}

func TestPreviousTag_OnlyTagIsHEAD(t *testing.T) {
	dir := setupGitRepo(t)
	chdir(t, dir)
	gitIn(t, dir, "tag", "v1")
	if _, err := PreviousTag("HEAD"); err == nil || !strings.Contains(err.Error(), "no tag found") {
		t.Errorf("a tag at HEAD has no tag before it, got %v", err)
	}
}

// Outside a repository the error says so instead of a bare "exit status 128" (#698).
func TestGitErrors_OutsideARepository(t *testing.T) {
	chdir(t, t.TempDir())
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(t.TempDir()))
	if _, err := PreviousTag("HEAD"); err == nil || !strings.Contains(err.Error(), "no git repository found") {
		t.Errorf("PreviousTag: want \"no git repository found\", got %v", err)
	}
	if _, err := LoadModelAtGitRef("architecture.jsonc", "HEAD"); err == nil || !strings.Contains(err.Error(), "no git repository found") {
		t.Errorf("LoadModelAtGitRef: want \"no git repository found\", got %v", err)
	}
}

// A ref that does not exist shows what git said (#698).
func TestResolveGitRef_ShowsGitsMessage(t *testing.T) {
	chdir(t, setupGitRepo(t))
	_, err := LoadModelAtGitRef("architecture.jsonc", "no-such-ref")
	if err == nil || !strings.Contains(err.Error(), "no-such-ref") || strings.Contains(err.Error(), "exit status") {
		t.Errorf("want git's message about the ref instead of \"exit status\", got %v", err)
	}
}

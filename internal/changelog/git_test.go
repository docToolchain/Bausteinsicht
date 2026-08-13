package changelog

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupGitRepo creates a temporary git repository with a single commit
// containing a minimal valid model file, returning the repo dir.
func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) string {
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

	run("init", "-q")
	content := `{
  "specification": {"elements": {"system": {"notation": "System"}}},
  "model": {"a": {"kind": "system", "title": "A"}}
}`
	if err := os.WriteFile(filepath.Join(dir, "architecture.jsonc"), []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	run("add", "architecture.jsonc")
	run("commit", "-q", "-m", "initial commit")
	return dir
}

// chdir changes the working directory to dir for the duration of the test —
// resolveGitRef/loadModelFromGit shell out to git without setting cmd.Dir,
// so they operate on the process's current directory.
func chdir(t *testing.T, dir string) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
}

func TestLoadModelAtGitRef(t *testing.T) {
	chdir(t, setupGitRepo(t))

	m, err := LoadModelAtGitRef("architecture.jsonc", "HEAD")
	if err != nil {
		t.Fatalf("LoadModelAtGitRef: %v", err)
	}
	elem, ok := m.Model["a"]
	if !ok {
		t.Fatalf("expected element 'a' in loaded model, got %+v", m.Model)
	}
	if elem.Title != "A" {
		t.Errorf("Title = %q, want \"A\"", elem.Title)
	}
}

func TestLoadModelAtGitRef_UnresolvableRef(t *testing.T) {
	chdir(t, setupGitRepo(t))

	if _, err := LoadModelAtGitRef("architecture.jsonc", "not-a-real-ref"); err == nil {
		t.Error("expected error for unresolvable git ref")
	}
}

func TestLoadModelAtGitRef_MissingFile(t *testing.T) {
	chdir(t, setupGitRepo(t))

	if _, err := LoadModelAtGitRef("does-not-exist.jsonc", "HEAD"); err == nil {
		t.Error("expected error for a model path that doesn't exist at HEAD")
	}
}

func TestGetCommitInfo(t *testing.T) {
	chdir(t, setupGitRepo(t))

	info, err := GetCommitInfo("HEAD")
	if err != nil {
		t.Fatalf("GetCommitInfo: %v", err)
	}
	if info.Author != "Test" {
		t.Errorf("Author = %q, want \"Test\"", info.Author)
	}
	if info.Message != "initial commit" {
		t.Errorf("Message = %q, want \"initial commit\"", info.Message)
	}
	if info.Hash == "" {
		t.Error("expected a non-empty commit hash")
	}
	if info.Timestamp == 0 {
		t.Error("expected a non-zero timestamp")
	}
}

// TestGitPath_NotOnPATH covers the "git not available" branch shared by
// resolveGitRef/loadModelFromGit/GetCommitInfo (SEC-021's exec.LookPath
// resolution).
func TestGitPath_NotOnPATH(t *testing.T) {
	t.Setenv("PATH", "")

	if _, err := LoadModelAtGitRef("architecture.jsonc", "HEAD"); err == nil {
		t.Error("expected error when git is not on PATH")
	}
	if _, err := GetCommitInfo("HEAD"); err == nil {
		t.Error("expected error when git is not on PATH")
	}
}

package stale

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// initGitRepo initialises a bare git repo in dir and configures local user identity.
// Returns the dir so callers can use it directly.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
}

// commitFile writes content to path inside dir, stages it, and creates a commit.
func commitFile(t *testing.T, dir, name, content, message string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
	cmd := exec.Command("git", "add", name)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
}

func TestGetLastModifiedDate_GitTracked(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	commitFile(t, dir, "model.jsonc", `{"model":{}}`, "initial")

	before := time.Now().Add(-2 * time.Second)
	got, err := GetLastModifiedDate(filepath.Join(dir, "model.jsonc"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.IsZero() {
		t.Fatal("expected non-zero time for tracked file, got zero")
	}
	if got.Before(before.Add(-24 * time.Hour)) {
		t.Errorf("commit time %v is unexpectedly old", got)
	}
}

func TestGetLastModifiedDate_Untracked(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "model.jsonc")
	if err := os.WriteFile(path, []byte(`{}`), 0600); err != nil {
		t.Fatalf("writing file: %v", err)
	}

	got, err := GetLastModifiedDate(path)
	if err != nil {
		t.Fatalf("unexpected error for untracked file: %v", err)
	}
	if !got.IsZero() {
		t.Errorf("expected zero time for untracked file, got %v", got)
	}
}

func TestDetect_GitBasedStaleness(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	commitFile(t, dir, "arch.jsonc", `{}`, "initial")

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"legacy": {Kind: "system", Title: "Legacy"},
		},
		Relationships: []model.Relationship{},
		Views:         map[string]model.View{},
	}

	// ThresholdDays=0: any element with a git-tracked date is considered stale.
	config := StaleConfig{ThresholdDays: 0}
	result, err := Detect(m, filepath.Join(dir, "arch.jsonc"), config)
	if err != nil {
		t.Fatalf("detection failed: %v", err)
	}

	if result.TotalElements != 1 {
		t.Fatalf("expected 1 element, got %d", result.TotalElements)
	}
	if len(result.StaleElements) != 1 {
		t.Errorf("expected 1 stale element via git-based detection, got %d", len(result.StaleElements))
	}
}

func TestDetect_ExplicitLastModifiedTakesPriority(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	commitFile(t, dir, "arch.jsonc", `{}`, "initial")

	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"fresh": {
				Kind:         "system",
				Title:        "Fresh",
				LastModified: time.Now().Format(time.RFC3339), // today → not stale
			},
		},
		Relationships: []model.Relationship{},
		Views:         map[string]model.View{},
	}

	// ThresholdDays=1: only elements older than 1 day are stale.
	config := StaleConfig{ThresholdDays: 1}
	result, err := Detect(m, filepath.Join(dir, "arch.jsonc"), config)
	if err != nil {
		t.Fatalf("detection failed: %v", err)
	}

	// Explicit lastModified=today overrides file-level date → not stale.
	if len(result.StaleElements) != 0 {
		t.Errorf("element with explicit today-lastModified should not be stale, got %d stale", len(result.StaleElements))
	}
}

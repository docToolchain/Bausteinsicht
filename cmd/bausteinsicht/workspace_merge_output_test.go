package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const mergeTestModel = `{"specification":{"elements":{"system":{"notation":"System"}}},"model":{"s":{"kind":"system","title":"S"}},"relationships":[],"views":{}}`

// mergeWorkspace writes a workspace of two local models (prefixes a and b) and
// returns the directory and the path of the config.
func mergeWorkspace(t *testing.T) (dir, config string) {
	t.Helper()
	dir = t.TempDir()
	for _, sub := range []string{"a", "b"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, sub, "architecture.jsonc"), []byte(mergeTestModel), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	config = filepath.Join(dir, "ws.jsonc")
	cfg := `{"workspace":{"name":"ws"},"models":[
  {"id":"a","path":"a/architecture.jsonc","prefix":"a"},
  {"id":"b","path":"b/architecture.jsonc","prefix":"b"}]}`
	if err := os.WriteFile(config, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir, config
}

// workspace merge overwrote an existing output without --force (import needs it)
// and accepted the config itself or an input model as the output, which destroyed
// the workspace definition (#708).
func TestWorkspaceMerge_RefusesToOverwriteWithoutForce(t *testing.T) {
	dir, config := mergeWorkspace(t)
	out := filepath.Join(dir, "out.jsonc")
	if _, err := executeImportCmd("workspace", "merge", config, out); err != nil {
		t.Fatalf("the first merge must work: %v", err)
	}
	before, _ := os.ReadFile(out)
	if err := os.WriteFile(out, []byte(`{"precious":true}`), 0o600); err != nil {
		t.Fatal(err)
	}

	text, err := executeImportCmd("workspace", "merge", config, out)
	if err == nil || exitCodeOf(t, err) != 2 || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("want exit code 2 mentioning --force, got %v\n%s", err, text)
	}
	if data, _ := os.ReadFile(out); string(data) != `{"precious":true}` {
		t.Errorf("the existing output must not be overwritten: %s", data)
	}

	if _, err := executeImportCmd("workspace", "merge", config, out, "--force"); err != nil {
		t.Fatalf("--force must overwrite: %v", err)
	}
	if data, _ := os.ReadFile(out); string(data) != string(before) {
		t.Errorf("--force must write the merged model again:\n%s", data)
	}
}

func TestWorkspaceMerge_NeverOverwritesItsOwnInputs(t *testing.T) {
	dir, config := mergeWorkspace(t)
	configBefore, _ := os.ReadFile(config)
	link := filepath.Join(dir, "link.jsonc")
	if err := os.Symlink(config, link); err != nil {
		link = ""
	}
	for name, out := range map[string]string{
		"the config":            config,
		"the config via a path": filepath.Join(dir, "a", "..", "ws.jsonc"),
		"an input model":        filepath.Join(dir, "a", "architecture.jsonc"),
		"the lock file":         filepath.Join(dir, "bausteinsicht.lock"),
		"a link to the config":  link,
	} {
		if out == "" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			if name == "the lock file" {
				if err := os.WriteFile(out, []byte("{}"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			text, err := executeImportCmd("workspace", "merge", config, out, "--force")
			if err == nil || exitCodeOf(t, err) != 2 || !strings.Contains(err.Error(), "would overwrite") {
				t.Fatalf("even --force must refuse, got %v\n%s", err, text)
			}
		})
	}
	if data, _ := os.ReadFile(config); string(data) != string(configBefore) {
		t.Errorf("the workspace config was changed:\n%s", data)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "a", "architecture.jsonc")); string(data) != mergeTestModel {
		t.Errorf("an input model was changed:\n%s", data)
	}
}

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fetchTestModel = `{"specification":{"elements":{"system":{"notation":"S"}}},"model":{"billing":{"kind":"system","title":"Billing"}},"views":{}}`

func runWorkspaceCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(append([]string{"workspace"}, args...))
	err := cmd.Execute()
	return out.String(), err
}

func writeFetchWorkspace(t *testing.T, config string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "workspace.json")
	if err := os.WriteFile(p, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// placeLockedCache writes what a successful "workspace fetch" leaves behind, so
// the command can run without a network.
func placeLockedCache(t *testing.T, cfgPath, id, url, content string) {
	t.Helper()
	dir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(filepath.Join(dir, ".bausteinsicht-cache"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".bausteinsicht-cache", id+".jsonc"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(content))
	lock, _ := json.Marshal(map[string]interface{}{
		"version": 1,
		"sources": map[string]interface{}{id: map[string]string{"url": url, "sha256": hex.EncodeToString(sum[:])}},
	})
	if err := os.WriteFile(filepath.Join(dir, "bausteinsicht.lock"), lock, 0o600); err != nil {
		t.Fatal(err)
	}
}

const fetchTestURL = "https://models.example.com/billing/v1/architecture.jsonc"

func fetchTestConfig(hosts string) string {
	return `{"workspace":{"name":"w"` + hosts + `},"models":[{"id":"billing","source":"` + fetchTestURL + `"}]}`
}

func TestWorkspaceFetch_UpToDateFromLockedCache(t *testing.T) {
	cfg := writeFetchWorkspace(t, fetchTestConfig(`,"allowedHosts":["models.example.com"]`))
	placeLockedCache(t, cfg, "billing", fetchTestURL, fetchTestModel)

	out, err := runWorkspaceCmd(t, "fetch", cfg)
	if err != nil || !strings.Contains(out, "billing: up-to-date (sha256 ") {
		t.Fatalf("text output = %q, err = %v", out, err)
	}

	out, err = runWorkspaceCmd(t, "fetch", cfg, "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Results []struct{ ID, Status string } `json:"results"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil || len(parsed.Results) != 1 || parsed.Results[0].Status != "up-to-date" {
		t.Fatalf("json output = %q (%v)", out, err)
	}
}

func TestWorkspaceFetch_NoRemoteModels(t *testing.T) {
	cfg := writeFetchWorkspace(t, `{"workspace":{"name":"w"},"models":[{"id":"a","path":"a.jsonc"}]}`)
	out, err := runWorkspaceCmd(t, "fetch", cfg)
	if err != nil || !strings.Contains(out, "No remote models") {
		t.Fatalf("output = %q, err = %v", out, err)
	}
}

func TestWorkspaceFetch_Errors(t *testing.T) {
	if _, err := runWorkspaceCmd(t, "fetch", filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Error("a missing configuration must fail")
	}
	cfg := writeFetchWorkspace(t, fetchTestConfig(""))
	if _, err := runWorkspaceCmd(t, "fetch", cfg); err == nil || !strings.Contains(err.Error(), "allowedHosts") {
		t.Errorf("without an allowlist: err = %v", err)
	}
	// A listed host that resolves to the loopback address is refused by the
	// real fetcher before any request is sent.
	cfg = writeFetchWorkspace(t, `{"workspace":{"name":"w","allowedHosts":["localhost"]},"models":[{"id":"b","source":"https://localhost/x.jsonc"}]}`)
	if _, err := runWorkspaceCmd(t, "fetch", cfg); err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("loopback source: err = %v", err)
	}
}

func TestWorkspaceList_ShowsSource(t *testing.T) {
	cfg := writeFetchWorkspace(t, fetchTestConfig(`,"allowedHosts":["models.example.com"]`))
	out, err := runWorkspaceCmd(t, "list", cfg)
	if err != nil || !strings.Contains(out, "Source: "+fetchTestURL) {
		t.Fatalf("output = %q, err = %v", out, err)
	}
}

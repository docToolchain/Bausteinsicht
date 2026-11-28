package e2e

// TestWorkspaceRemoteSources covers Phase 2 of #227 (ADR-012) through the real
// CLI without any network access: the configuration rules, the refusal to
// connect to local and metadata addresses, and the offline path (cache plus
// bausteinsicht.lock), including tamper detection.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeRemoteWorkspace(t *testing.T, config string) (dir, cfgPath string) {
	t.Helper()
	dir = t.TempDir()
	for name, content := range map[string]string{"a.jsonc": teamAModel, "workspace.json": config} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir, filepath.Join(dir, "workspace.json")
}

func sha(data string) string {
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}

// place writes the cache file and lock entry that "workspace fetch" would have
// produced for a model, so the offline path can be tested without a server.
func placeCache(t *testing.T, dir, id, url, content string) {
	t.Helper()
	cache := filepath.Join(dir, ".bausteinsicht-cache")
	if err := os.MkdirAll(cache, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cache, id+".jsonc"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	lock, _ := json.Marshal(map[string]interface{}{
		"version": 1,
		"sources": map[string]interface{}{id: map[string]string{"url": url, "sha256": sha(content)}},
	})
	if err := os.WriteFile(filepath.Join(dir, "bausteinsicht.lock"), lock, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestWorkspaceRemoteSources(t *testing.T) {
	bin := buildBinary(t)
	const url = "https://models.example.com/team-b/v1/architecture.jsonc"
	remote := func(hosts string) string {
		return `{ "workspace": { "name": "landscape"` + hosts + ` },
  "models": [ { "id": "team_a", "path": "a.jsonc" }, { "id": "team_b", "source": "` + url + `", "prefix": "b" } ] }`
	}

	t.Run("remote sources are off without an allowlist", func(t *testing.T) {
		dir, cfg := writeRemoteWorkspace(t, remote(""))
		for _, args := range [][]string{{"workspace", "fetch", cfg}, {"workspace", "validate", cfg}, {"workspace", "merge", cfg, filepath.Join(dir, "out.json")}} {
			out, code := runCLIAllowFail(t, bin, dir, args...)
			if code == 0 || !strings.Contains(out, "allowedHosts") {
				t.Errorf("%v: exit %d, want a refusal naming allowedHosts\n%s", args, code, out)
			}
		}
		if _, err := os.Stat(filepath.Join(dir, "bausteinsicht.lock")); err == nil {
			t.Error("a lock file was written although nothing was fetched")
		}
	})

	t.Run("only https and plain hosts are accepted", func(t *testing.T) {
		dir, cfg := writeRemoteWorkspace(t, `{ "workspace": { "name": "w", "allowedHosts": ["models.example.com"] },
  "models": [ { "id": "b", "source": "http://models.example.com/x.jsonc" } ] }`)
		out, code := runCLIAllowFail(t, bin, dir, "workspace", "fetch", cfg)
		if code == 0 || !strings.Contains(out, "only https") {
			t.Errorf("http source: exit %d\n%s", code, out)
		}
		dir, cfg = writeRemoteWorkspace(t, `{ "workspace": { "name": "w", "allowedHosts": ["*.example.com"] },
  "models": [ { "id": "b", "source": "`+url+`" } ] }`)
		out, code = runCLIAllowFail(t, bin, dir, "workspace", "fetch", cfg)
		if code == 0 || !strings.Contains(out, "wildcard") {
			t.Errorf("wildcard allowlist: exit %d\n%s", code, out)
		}
	})

	t.Run("a listed host that resolves to a local or metadata address is refused", func(t *testing.T) {
		for _, host := range []string{"localhost", "127.0.0.1", "169.254.169.254"} {
			dir, cfg := writeRemoteWorkspace(t, `{ "workspace": { "name": "w", "allowedHosts": ["`+host+`"] },
  "models": [ { "id": "b", "source": "https://`+host+`/architecture.jsonc" } ] }`)
			out, code := runCLIAllowFail(t, bin, dir, "workspace", "fetch", cfg)
			if code == 0 || !strings.Contains(out, "not allowed") {
				t.Errorf("%s: exit %d, want the address to be refused\n%s", host, code, out)
			}
			if _, err := os.Stat(filepath.Join(dir, ".bausteinsicht-cache", "b.jsonc")); err == nil {
				t.Errorf("%s: a cache file was written", host)
			}
		}
	})

	t.Run("without fetch the remote model is reported as missing", func(t *testing.T) {
		dir, cfg := writeRemoteWorkspace(t, remote(`, "allowedHosts": ["models.example.com"]`))
		out, code := runCLIAllowFail(t, bin, dir, "workspace", "merge", cfg, filepath.Join(dir, "out.json"))
		if code == 0 || !strings.Contains(out, "workspace fetch") {
			t.Errorf("merge: exit %d, want a hint to run workspace fetch\n%s", code, out)
		}
	})

	t.Run("cache plus lock work offline", func(t *testing.T) {
		dir, cfg := writeRemoteWorkspace(t, remote(`, "allowedHosts": ["models.example.com"]`))
		placeCache(t, dir, "team_b", url, teamBModel)

		if out := runCLI(t, bin, dir, "workspace", "list", cfg); !strings.Contains(out, "Source: "+url) {
			t.Errorf("list does not show the source:\n%s", out)
		}
		if out := runCLI(t, bin, dir, "workspace", "validate", cfg); !strings.Contains(out, "valid") {
			t.Errorf("validate:\n%s", out)
		}
		// Already cached and locked: fetch does not need the network.
		if out := runCLI(t, bin, dir, "workspace", "fetch", cfg); !strings.Contains(out, "team_b: up-to-date") {
			t.Errorf("fetch of a locked cache:\n%s", out)
		}
		outFile := filepath.Join(dir, "merged.jsonc")
		runCLI(t, bin, dir, "workspace", "merge", cfg, outFile)
		merged := readMerged(t, outFile)
		if _, ok := merged.Model["b_billing"]; !ok {
			t.Errorf("merged model lacks b_billing: %v", keysOfRaw(merged.Model))
		}
		if out := runCLI(t, bin, dir, "workspace", "fetch", cfg, "--format", "json"); !strings.Contains(out, `"status":"up-to-date"`) {
			t.Errorf("json fetch output: %s", out)
		}
	})

	t.Run("a changed cache file is detected", func(t *testing.T) {
		dir, cfg := writeRemoteWorkspace(t, remote(`, "allowedHosts": ["models.example.com"]`))
		placeCache(t, dir, "team_b", url, teamBModel)
		cache := filepath.Join(dir, ".bausteinsicht-cache", "team_b.jsonc")
		if err := os.WriteFile(cache, []byte(strings.ReplaceAll(teamBModel, "Billing", "Evil")), 0o600); err != nil {
			t.Fatal(err)
		}
		for _, args := range [][]string{{"workspace", "validate", cfg}, {"workspace", "merge", cfg, filepath.Join(dir, "out.json")}} {
			out, code := runCLIAllowFail(t, bin, dir, args...)
			if code == 0 || !strings.Contains(out, "does not match bausteinsicht.lock") {
				t.Errorf("%v: exit %d, want a hash mismatch\n%s", args, code, out)
			}
		}
	})

	t.Run("a remote model must not set a local path as well", func(t *testing.T) {
		dir, cfg := writeRemoteWorkspace(t, `{ "workspace": { "name": "w", "allowedHosts": ["models.example.com"] },
  "models": [ { "id": "b", "path": "a.jsonc", "source": "`+url+`" } ] }`)
		out, code := runCLIAllowFail(t, bin, dir, "workspace", "validate", cfg)
		if code == 0 || !strings.Contains(out, "either path or source") {
			t.Errorf("exit %d\n%s", code, out)
		}
	})
}

// #735: a lock hash in upper case is accepted, and the cache file of a model that is
// no longer remote goes with its lock entry (no network involved).
func TestWorkspaceLockHygiene(t *testing.T) {
	bin := buildBinary(t)
	const url = "https://models.example.com/team-b/v1/architecture.jsonc"
	config := `{ "workspace": { "name": "landscape", "allowedHosts": ["models.example.com"] },
  "models": [ { "id": "team_a", "path": "a.jsonc" }, { "id": "team_b", "source": "` + url + `", "prefix": "b" } ] }`

	t.Run("a hash in upper case matches", func(t *testing.T) {
		dir, cfg := writeRemoteWorkspace(t, config)
		placeCache(t, dir, "team_b", url, teamBModel)
		lockFile := filepath.Join(dir, "bausteinsicht.lock")
		data, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(lockFile, []byte(strings.Replace(string(data), sha(teamBModel), strings.ToUpper(sha(teamBModel)), 1)), 0o600); err != nil {
			t.Fatal(err)
		}
		if out, code := runCLIAllowFail(t, bin, dir, "workspace", "validate", cfg); code != 0 {
			t.Errorf("validate with an upper-case lock hash: exit %d\n%s", code, out)
		}
	})

	t.Run("the cache of a model that is no longer remote is removed", func(t *testing.T) {
		dir, cfg := writeRemoteWorkspace(t, `{ "workspace": { "name": "landscape" }, "models": [ { "id": "team_a", "path": "a.jsonc" } ] }`)
		placeCache(t, dir, "old_model", url, teamBModel)
		stale := filepath.Join(dir, ".bausteinsicht-cache", "old_model.jsonc")
		runCLI(t, bin, dir, "workspace", "fetch", cfg)
		if _, err := os.Stat(stale); err == nil {
			t.Error("the cache file of the model that is gone is still there")
		}
		if data, _ := os.ReadFile(filepath.Join(dir, "bausteinsicht.lock")); strings.Contains(string(data), "old_model") {
			t.Errorf("the lock entry is still there:\n%s", data)
		}
	})
}

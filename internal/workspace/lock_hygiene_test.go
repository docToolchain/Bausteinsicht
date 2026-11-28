package workspace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The hash in the lock file is compared as the hex number it is: a hash written in
// upper case (by hand or by another tool) is the same hash (#735).
func TestLock_HashCaseDoesNotMatter(t *testing.T) {
	dir := t.TempDir()
	writeLocal(t, dir)
	cfgPath := filepath.Join(dir, "workspace.jsonc")
	cfg := remoteConfig("models.example.com")
	calls := 0
	if _, err := Sync(context.Background(), cfg, cfgPath, stubFetcher(remoteModel, &calls), false); err != nil {
		t.Fatal(err)
	}

	lockData, err := os.ReadFile(LockPath(cfgPath))
	if err != nil {
		t.Fatal(err)
	}
	upper := hashHex([]byte(remoteModel))
	lockData = []byte(strings.Replace(string(lockData), upper, strings.ToUpper(upper), 1))
	if err := os.WriteFile(LockPath(cfgPath), lockData, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadModels(cfg, dir); err != nil {
		t.Errorf("a cache whose lock hash is in upper case must load, got %v", err)
	}
	res, err := Sync(context.Background(), cfg, cfgPath, stubFetcher("never used", &calls), false)
	if err != nil || len(res) != 1 || res[0].Status != "up-to-date" || calls != 1 {
		t.Errorf("fetch with an upper-case lock hash: %+v, %v, calls %d", res, err, calls)
	}
}

// When a model is no longer remote (removed, renamed, or given a local path) its
// cache file goes with its lock entry; other files in the cache directory and
// lock entries with an ID that cannot be a cache file name are left alone (#735).
func TestSync_RemovesTheCacheOfModelsThatAreNoLongerRemote(t *testing.T) {
	dir := t.TempDir()
	writeLocal(t, dir)
	cfgPath := filepath.Join(dir, "workspace.jsonc")
	cfg := remoteConfig("models.example.com")
	calls := 0
	if _, err := Sync(context.Background(), cfg, cfgPath, stubFetcher(remoteModel, &calls), false); err != nil {
		t.Fatal(err)
	}
	old := CachePath(dir, "team-b")
	if _, err := os.Stat(old); err != nil {
		t.Fatalf("setup: %v", err)
	}
	notes := filepath.Join(dir, CacheDirName, "notes.txt")
	outside := filepath.Join(dir, "evil.jsonc")
	for _, f := range []string{notes, outside} {
		if err := os.WriteFile(f, []byte("keep"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	lock, err := LoadLock(LockPath(cfgPath))
	if err != nil {
		t.Fatal(err)
	}
	lock.Sources["../evil"] = LockEntry{URL: "https://models.example.com/evil", SHA256: "00"}
	if err := SaveLock(LockPath(cfgPath), lock); err != nil {
		t.Fatal(err)
	}

	// The model is renamed: "billing" is remote now, "team-b" is gone.
	cfg.Models[1].ID = "billing"
	if _, err := Sync(context.Background(), cfg, cfgPath, stubFetcher(remoteModel, &calls), false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(old); err == nil {
		t.Error("the cache file of the renamed model is still there")
	}
	if _, err := os.Stat(CachePath(dir, "billing")); err != nil {
		t.Errorf("the cache of the new ID is missing: %v", err)
	}
	for _, f := range []string{notes, outside} {
		if data, _ := os.ReadFile(f); string(data) != "keep" {
			t.Errorf("%s must be left alone", f)
		}
	}
	if after, _ := LoadLock(LockPath(cfgPath)); after.Sources["team-b"].URL != "" {
		t.Errorf("the lock entry of the old ID is still there: %+v", after.Sources)
	}
}

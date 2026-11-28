package workspace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A download that is not a model (a README, an error page behind a 200, a JSON
// error object) is refused before the cache and the lock are written; nothing is
// pinned (#734).
func TestSync_RefusesContentThatIsNotAModel(t *testing.T) {
	for name, body := range map[string]string{
		"a README":            "= README\n\nThis is not a model.\n",
		"an HTML error page":  "<!DOCTYPE html><html><body>Sign in</body></html>",
		"a JSON error object": `{"message": "Not Found"}`,
		"an empty file":       "",
		"JSON null":           "null",
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeLocal(t, dir)
			cfgPath := filepath.Join(dir, "workspace.jsonc")
			cfg := remoteConfig("models.example.com")

			calls := 0
			_, err := Sync(context.Background(), cfg, cfgPath, stubFetcher(body, &calls), false)
			if err == nil || !strings.Contains(err.Error(), "is not a Bausteinsicht model") ||
				!strings.Contains(err.Error(), "https://models.example.com/b/v1/architecture.jsonc") {
				t.Fatalf("want an error \"<url> is not a Bausteinsicht model\", got %v", err)
			}
			if _, statErr := os.Stat(LockPath(cfgPath)); statErr == nil {
				t.Error("the lock file was written")
			}
			if _, statErr := os.Stat(CachePath(dir, "team-b")); statErr == nil {
				t.Error("the content was cached")
			}
		})
	}
}

// With --update a download that is not a model does not replace the pin that is
// there: the lock and the cache keep the model that was good (#734).
func TestSync_UpdateKeepsThePinWhenTheNewContentIsNotAModel(t *testing.T) {
	dir := t.TempDir()
	writeLocal(t, dir)
	cfgPath := filepath.Join(dir, "workspace.jsonc")
	cfg := remoteConfig("models.example.com")
	calls := 0
	if _, err := Sync(context.Background(), cfg, cfgPath, stubFetcher(remoteModel, &calls), false); err != nil {
		t.Fatal(err)
	}
	lockBefore, _ := os.ReadFile(LockPath(cfgPath))

	_, err := Sync(context.Background(), cfg, cfgPath, stubFetcher("<html>maintenance</html>", &calls), true)
	if err == nil || !strings.Contains(err.Error(), "is not a Bausteinsicht model") {
		t.Fatalf("want a refusal, got %v", err)
	}
	if lockAfter, _ := os.ReadFile(LockPath(cfgPath)); string(lockAfter) != string(lockBefore) {
		t.Error("the lock changed")
	}
	if cached, _ := os.ReadFile(CachePath(dir, "team-b")); string(cached) != remoteModel {
		t.Errorf("the cache changed: %q", cached)
	}
}

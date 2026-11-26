package workspace

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const remoteModel = `{
  "specification": {"elements": {"system": {"notation": "S"}}},
  "model": {"shared": {"kind": "system", "title": "Shared"}},
  "views": {"all": {"title": "All", "include": ["*"]}}
}`

func remoteConfig(hosts ...string) *Config {
	return &Config{
		Workspace: WorkspaceMetadata{Name: "W", AllowedHosts: hosts},
		Models: []ModelRef{
			{ID: "local", Path: "local.jsonc"},
			{ID: "team-b", Source: "https://models.example.com/b/v1/architecture.jsonc", Prefix: "b"},
		},
	}
}

func stubFetcher(body string, calls *int) *Fetcher {
	return &Fetcher{stub: func(_ context.Context, _ string) ([]byte, error) {
		*calls++
		return []byte(body), nil
	}}
}

func TestBlockedIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "::1", "10.1.2.3", "172.16.0.1", "192.168.1.1", "fd00::1",
		"169.254.169.254", "169.254.0.1", "fe80::1", "0.0.0.0", "::", "224.0.0.1", "ff02::1",
		"100.64.0.1", "198.18.0.1", "240.0.0.1", "::ffff:10.0.0.1", "::ffff:169.254.169.254",
	}
	for _, s := range blocked {
		if !blockedIP(net.ParseIP(s)) {
			t.Errorf("%s must be blocked", s)
		}
	}
	for _, s := range []string{"93.184.216.34", "8.8.8.8", "2606:4700:4700::1111"} {
		if blockedIP(net.ParseIP(s)) {
			t.Errorf("%s must be allowed", s)
		}
	}
}

type fakeResolver struct {
	addrs []string
	err   error
}

func (r fakeResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	if r.err != nil {
		return nil, r.err
	}
	out := make([]net.IPAddr, 0, len(r.addrs))
	for _, a := range r.addrs {
		out = append(out, net.IPAddr{IP: net.ParseIP(a)})
	}
	return out, nil
}

func TestSafeDialer(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name    string
		res     fakeResolver
		addr    string
		wantErr string
	}{
		{"loopback name", fakeResolver{addrs: []string{"127.0.0.1"}}, "localhost:443", "not allowed"},
		{"metadata address", fakeResolver{addrs: []string{"169.254.169.254"}}, "metadata.example.com:443", "not allowed"},
		{"private", fakeResolver{addrs: []string{"10.0.0.5"}}, "intranet.example.com:443", "not allowed"},
		{"one blocked of several", fakeResolver{addrs: []string{"93.184.216.34", "192.168.0.1"}}, "mixed.example.com:443", "not allowed"},
		{"ipv6 loopback", fakeResolver{addrs: []string{"::1"}}, "v6.example.com:443", "not allowed"},
		{"no address", fakeResolver{}, "empty.example.com:443", "no address"},
		{"lookup error", fakeResolver{err: errors.New("nxdomain")}, "gone.example.com:443", "nxdomain"},
		{"bad address", fakeResolver{}, "no-port", "missing port"},
		// A public address passes the check; the canceled context stops the dial
		// before any packet is sent.
		{"public address", fakeResolver{addrs: []string{"93.184.216.34"}}, "ok.example.com:443", "canceled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &safeDialer{resolver: tt.res}
			_, err := d.DialContext(canceled, "tcp", tt.addr)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestSourceHost(t *testing.T) {
	good := map[string]string{
		"https://Models.Example.com/a/b.jsonc":    "models.example.com",
		"https://models.example.com:443/a.jsonc":  "models.example.com",
		"https://raw.githubusercontent.com/o/r/x": "raw.githubusercontent.com",
	}
	for in, want := range good {
		got, err := sourceHost(in)
		if err != nil || got != want {
			t.Errorf("sourceHost(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	bad := []string{
		"http://models.example.com/a", "ftp://models.example.com/a", "file:///etc/passwd",
		"https://user:pw@models.example.com/a", "https:///a", "https://models.example.com:8443/a",
		"https://models.example.com/a#frag", "://bad", "models.example.com/a",
	}
	for _, in := range bad {
		if _, err := sourceHost(in); err == nil {
			t.Errorf("sourceHost(%q) must fail", in)
		}
	}
}

func TestTokenEnvName(t *testing.T) {
	got := TokenEnvName("raw.githubusercontent.com")
	if want := "BAUSTEINSICHT_FETCH_TOKEN_RAW_GITHUBUSERCONTENT_COM"; got != want {
		t.Errorf("TokenEnvName = %q, want %q", got, want)
	}
	if got := TokenEnvName("git-a.example.org"); got != "BAUSTEINSICHT_FETCH_TOKEN_GIT_A_EXAMPLE_ORG" {
		t.Errorf("TokenEnvName = %q", got)
	}
}

func TestValidateSources(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{"valid", func(*Config) {}, ""},
		{"no allowlist", func(c *Config) { c.Workspace.AllowedHosts = nil }, "not listed in workspace.allowedHosts"},
		{"other host", func(c *Config) { c.Workspace.AllowedHosts = []string{"other.example.com"} }, "not listed"},
		{"wildcard allowlist", func(c *Config) { c.Workspace.AllowedHosts = []string{"*.example.com"} }, "without scheme, port or wildcard"},
		{"port in allowlist", func(c *Config) { c.Workspace.AllowedHosts = []string{"models.example.com:443"} }, "without scheme, port or wildcard"},
		{"upper-case allowlist", func(c *Config) { c.Workspace.AllowedHosts = []string{"Models.example.com"} }, "lower-case"},
		{"empty allowlist entry", func(c *Config) { c.Workspace.AllowedHosts = []string{""} }, "without scheme"},
		{"path and source", func(c *Config) { c.Models[1].Path = "x.jsonc" }, "either path or source"},
		{"neither", func(c *Config) { c.Models[1].Source = "" }, "path or source is required"},
		{"http", func(c *Config) { c.Models[1].Source = "http://models.example.com/x" }, "only https"},
		{"unsafe id", func(c *Config) { c.Models[1].ID = "../evil" }, "may contain only"},
		{"dot id", func(c *Config) { c.Models[1].ID = "a.b" }, "may contain only"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := remoteConfig("models.example.com")
			tt.mutate(cfg)
			err := errors.Join(ValidateSources(cfg)...)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func writeLocal(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "local.jsonc"), []byte(strings.ReplaceAll(remoteModel, "shared", "own")), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestSync_FetchLockAndLoad(t *testing.T) {
	dir := t.TempDir()
	writeLocal(t, dir)
	cfgPath := filepath.Join(dir, "workspace.jsonc")
	cfg := remoteConfig("models.example.com")

	calls := 0
	res, err := Sync(context.Background(), cfg, cfgPath, stubFetcher(remoteModel, &calls), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || res[0].Status != "fetched" || res[0].ID != "team-b" || calls != 1 {
		t.Fatalf("results = %+v, calls = %d", res, calls)
	}
	lock, err := LoadLock(LockPath(cfgPath))
	if err != nil || lock.Sources["team-b"].SHA256 != hashHex([]byte(remoteModel)) {
		t.Fatalf("lock = %+v, %v", lock, err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, CacheDirName, ".gitignore")); string(b) != "*\n" {
		t.Errorf("cache .gitignore = %q", b)
	}
	if info, _ := os.Stat(CachePath(dir, "team-b")); info == nil || info.Mode().Perm() != 0o600 {
		t.Errorf("cache file mode = %v", info)
	}

	// Everything reads the cache; no network involved.
	loaded, err := LoadModels(cfg, dir)
	if err != nil || len(loaded) != 2 {
		t.Fatalf("LoadModels = %d models, %v", len(loaded), err)
	}
	merged, err := MergeModels(loaded)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := merged.Model["b_shared"]; !ok {
		t.Errorf("merged model lacks b_shared: %v", merged.Model)
	}

	// A second fetch is a no-op.
	res, err = Sync(context.Background(), cfg, cfgPath, stubFetcher("never used", &calls), false)
	if err != nil || res[0].Status != "up-to-date" || calls != 1 {
		t.Fatalf("second fetch = %+v, %v, calls %d", res, err, calls)
	}
}

func TestSync_RestoresMissingCacheFromLock(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "workspace.jsonc")
	cfg := remoteConfig("models.example.com")
	calls := 0
	if _, err := Sync(context.Background(), cfg, cfgPath, stubFetcher(remoteModel, &calls), false); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(CachePath(dir, "team-b")); err != nil {
		t.Fatal(err)
	}
	res, err := Sync(context.Background(), cfg, cfgPath, stubFetcher(remoteModel, &calls), false)
	if err != nil || res[0].Status != "restored" || calls != 2 {
		t.Fatalf("restore = %+v, %v, calls %d", res, err, calls)
	}
}

func TestSync_ChangedContentNeedsUpdate(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "workspace.jsonc")
	cfg := remoteConfig("models.example.com")
	calls := 0
	if _, err := Sync(context.Background(), cfg, cfgPath, stubFetcher(remoteModel, &calls), false); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(CachePath(dir, "team-b")); err != nil {
		t.Fatal(err)
	}
	changed := strings.ReplaceAll(remoteModel, "Shared", "Tampered")

	_, err := Sync(context.Background(), cfg, cfgPath, stubFetcher(changed, &calls), false)
	if err == nil || !strings.Contains(err.Error(), "changed") || !strings.Contains(err.Error(), "--update") {
		t.Fatalf("err = %v, want a changed-content error naming --update", err)
	}
	if _, statErr := os.Stat(CachePath(dir, "team-b")); statErr == nil {
		t.Error("the cache must not be written when the hash differs from the lock")
	}
	lock, _ := LoadLock(LockPath(cfgPath))
	if lock.Sources["team-b"].SHA256 != hashHex([]byte(remoteModel)) {
		t.Error("the lock must not change without --update")
	}

	res, err := Sync(context.Background(), cfg, cfgPath, stubFetcher(changed, &calls), true)
	if err != nil || res[0].Status != "updated" {
		t.Fatalf("update = %+v, %v", res, err)
	}
	lock, _ = LoadLock(LockPath(cfgPath))
	if lock.Sources["team-b"].SHA256 != hashHex([]byte(changed)) {
		t.Error("--update must rewrite the lock")
	}
}

func TestSync_UpdateWithSameContentIsUpToDate(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "workspace.jsonc")
	cfg := remoteConfig("models.example.com")
	calls := 0
	if _, err := Sync(context.Background(), cfg, cfgPath, stubFetcher(remoteModel, &calls), false); err != nil {
		t.Fatal(err)
	}
	res, err := Sync(context.Background(), cfg, cfgPath, stubFetcher(remoteModel, &calls), true)
	if err != nil || res[0].Status != "up-to-date" || calls != 2 {
		t.Fatalf("update = %+v, %v, calls %d", res, err, calls)
	}
}

func TestSync_ChangedURLIsANewSourceAndStaleEntriesArePruned(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "workspace.jsonc")
	cfg := remoteConfig("models.example.com")
	calls := 0
	if _, err := Sync(context.Background(), cfg, cfgPath, stubFetcher(remoteModel, &calls), false); err != nil {
		t.Fatal(err)
	}
	cfg.Models[1].Source = "https://models.example.com/b/v2/architecture.jsonc"
	res, err := Sync(context.Background(), cfg, cfgPath, stubFetcher(remoteModel, &calls), false)
	if err != nil || res[0].Status != "fetched" {
		t.Fatalf("new URL = %+v, %v", res, err)
	}
	lock, _ := LoadLock(LockPath(cfgPath))
	if lock.Sources["team-b"].URL != cfg.Models[1].Source {
		t.Errorf("lock URL = %q", lock.Sources["team-b"].URL)
	}

	cfg.Models = cfg.Models[:1] // the remote model is removed from the configuration
	if _, err := Sync(context.Background(), cfg, cfgPath, stubFetcher("", &calls), false); err != nil {
		t.Fatal(err)
	}
	lock, _ = LoadLock(LockPath(cfgPath))
	if len(lock.Sources) != 0 {
		t.Errorf("stale lock entries were not pruned: %+v", lock.Sources)
	}
}

func TestSync_NoRemoteModelsWritesNothing(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{Models: []ModelRef{{ID: "a", Path: "a.jsonc"}}}
	res, err := Sync(context.Background(), cfg, filepath.Join(dir, "ws.jsonc"), stubFetcher("", new(int)), false)
	if err != nil || len(res) != 0 {
		t.Fatalf("Sync = %+v, %v", res, err)
	}
	if _, err := os.Stat(filepath.Join(dir, LockFileName)); err == nil {
		t.Error("no lock file may be written without remote models")
	}
}

func TestSync_RejectsInvalidConfigurationBeforeAnyRequest(t *testing.T) {
	cfg := remoteConfig() // no allowlist
	calls := 0
	_, err := Sync(context.Background(), cfg, filepath.Join(t.TempDir(), "ws.jsonc"), stubFetcher(remoteModel, &calls), false)
	if err == nil || !strings.Contains(err.Error(), "allowedHosts") || calls != 0 {
		t.Fatalf("err = %v, calls = %d", err, calls)
	}
}

func TestSync_FetchErrorIsReported(t *testing.T) {
	cfg := remoteConfig("models.example.com")
	f := &Fetcher{stub: func(context.Context, string) ([]byte, error) { return nil, errors.New("boom") }}
	if _, err := Sync(context.Background(), cfg, filepath.Join(t.TempDir(), "ws.jsonc"), f, false); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadModels_RemoteProblems(t *testing.T) {
	dir := t.TempDir()
	writeLocal(t, dir)
	cfgPath := filepath.Join(dir, "workspace.jsonc")
	cfg := remoteConfig("models.example.com")

	if _, err := LoadModels(cfg, dir); err == nil || !strings.Contains(err.Error(), "workspace fetch") {
		t.Fatalf("unlocked: err = %v, want a hint to run workspace fetch", err)
	}

	calls := 0
	if _, err := Sync(context.Background(), cfg, cfgPath, stubFetcher(remoteModel, &calls), false); err != nil {
		t.Fatal(err)
	}
	cache := CachePath(dir, "team-b")

	if err := os.WriteFile(cache, []byte(strings.ReplaceAll(remoteModel, "Shared", "Tampered")), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadModels(cfg, dir); err == nil || !strings.Contains(err.Error(), "does not match "+LockFileName) {
		t.Fatalf("tampered cache: err = %v", err)
	}

	if err := os.Remove(cache); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadModels(cfg, dir); err == nil || !strings.Contains(err.Error(), "not in the cache") {
		t.Fatalf("missing cache: err = %v", err)
	}

	cfg.Models[1].Source = "https://models.example.com/other.jsonc"
	if _, err := LoadModels(cfg, dir); err == nil || !strings.Contains(err.Error(), "not locked") {
		t.Fatalf("changed URL: err = %v", err)
	}

	cfg.Workspace.AllowedHosts = nil
	if _, err := LoadModels(cfg, dir); err == nil || !strings.Contains(err.Error(), "allowedHosts") {
		t.Fatalf("no allowlist: err = %v", err)
	}
}

func TestLock_Errors(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, LockFileName)
	if l, err := LoadLock(p); err != nil || len(l.Sources) != 0 {
		t.Fatalf("missing lock: %v, %v", l, err)
	}
	for name, content := range map[string]string{
		"invalid JSON":  "{",
		"wrong version": `{"version": 7, "sources": {}}`,
	} {
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadLock(p); err == nil {
			t.Errorf("%s: LoadLock must fail", name)
		}
	}
	if err := os.WriteFile(p, []byte(`{"version": 1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if l, err := LoadLock(p); err != nil || l.Sources == nil {
		t.Fatalf("lock without sources: %v, %v", l, err)
	}
	if _, err := LoadLock(dir); err == nil {
		t.Error("reading a directory as lock file must fail")
	}
	if err := SaveLock(filepath.Join(p, "sub", LockFileName), &Lock{}); err == nil {
		t.Error("SaveLock below a file must fail")
	}
}

// --- real HTTP behaviour against a local TLS server ---

func tlsFetcher(srv *httptest.Server, env map[string]string) *Fetcher {
	return &Fetcher{
		client:   srv.Client(),
		maxBytes: 64,
		getenv:   func(k string) string { return env[k] },
	}
}

func TestFetcher_Get(t *testing.T) {
	var gotAuth string
	mux := http.NewServeMux()
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte("hello"))
	})
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://169.254.169.254/latest", http.StatusFound)
	})
	mux.HandleFunc("/missing", http.NotFound)
	mux.HandleFunc("/big", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 100)))
	})
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()
	host := "127.0.0.1"
	ctx := context.Background()

	f := tlsFetcher(srv, map[string]string{TokenEnvName(host): "s3cret"})
	data, err := f.get(ctx, srv.URL+"/ok", host)
	if err != nil || string(data) != "hello" {
		t.Fatalf("get = %q, %v", data, err)
	}
	if gotAuth != "Bearer s3cret" {
		t.Errorf("Authorization = %q", gotAuth)
	}

	gotAuth = "unset"
	if _, err := tlsFetcher(srv, nil).get(ctx, srv.URL+"/ok", host); err != nil || gotAuth != "" {
		t.Errorf("without token: err = %v, Authorization = %q", err, gotAuth)
	}

	if _, err := f.get(ctx, srv.URL+"/redirect", host); err == nil || !strings.Contains(err.Error(), "redirect") {
		t.Errorf("redirect: err = %v", err)
	}
	if _, err := f.get(ctx, srv.URL+"/missing", host); err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("404: err = %v", err)
	}
	if _, err := f.get(ctx, srv.URL+"/big", host); err == nil || !strings.Contains(err.Error(), "larger than") {
		t.Errorf("size limit: err = %v", err)
	}
	if _, err := f.get(ctx, "https://127.0.0.1:1/never", host); err == nil {
		t.Error("connection error must be reported")
	}
}

func TestFetch_RejectsBadURLBeforeAnyRequest(t *testing.T) {
	f := NewFetcher()
	for _, u := range []string{"http://models.example.com/x", "https://u:p@models.example.com/x", "https://models.example.com:8080/x"} {
		if _, err := f.Fetch(context.Background(), u); err == nil {
			t.Errorf("Fetch(%q) must fail", u)
		}
	}
}

// The real fetcher must refuse a loopback server even though TLS would succeed.
func TestNewFetcher_BlocksLoopbackServer(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("x")) }))
	defer srv.Close()
	f := NewFetcher()
	_, err := f.get(context.Background(), srv.URL+"/", "127.0.0.1")
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("err = %v, want the loopback address to be refused", err)
	}
}

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "sub", "f.json")
	if err := writeFileAtomic(target, []byte("one")); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomic(target, []byte("two")); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(target); string(b) != "two" {
		t.Errorf("content = %q", b)
	}
	if info, _ := os.Stat(target); info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v", info.Mode().Perm())
	}
	if entries, _ := os.ReadDir(filepath.Dir(target)); len(entries) != 1 {
		t.Errorf("a temp file was left behind: %v", entries)
	}

	// The parent directory cannot be created below a regular file.
	if err := writeFileAtomic(filepath.Join(target, "x"), []byte("x")); err == nil {
		t.Error("writing below a file must fail")
	}
	// A directory in the way makes the final rename fail; the temp file is removed.
	blocked := filepath.Join(dir, "blocked")
	if err := os.MkdirAll(filepath.Join(blocked, "inner"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomic(blocked, []byte("x")); err == nil {
		t.Error("renaming over a non-empty directory must fail")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 2 { // sub, blocked
		t.Errorf("a temp file was left behind: %v", entries)
	}
}

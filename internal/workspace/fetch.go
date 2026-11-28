package workspace

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

const (
	fetchTimeout = 30 * time.Second
	// tokenEnvPrefix is the prefix of the per-host token environment variable
	// (see TokenEnvName).
	tokenEnvPrefix = "BAUSTEINSICHT_FETCH_TOKEN_"
)

// blockedNets are address ranges a remote source must never resolve to, on top
// of the loopback/private/link-local checks of the net package. They cover
// carrier-grade NAT, benchmarking, reserved and documentation ranges.
var blockedNets = mustParseCIDRs(
	"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24",
	"198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4",
	"64:ff9b::/96", "2001:db8::/32",
)

func mustParseCIDRs(cidrs ...string) []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			panic(err)
		}
		nets = append(nets, n)
	}
	return nets
}

// blockedIP reports whether a fetch must not connect to ip: loopback, private
// (RFC 1918 / ULA), link-local (including the cloud metadata address
// 169.254.169.254), multicast, unspecified and reserved ranges.
func blockedIP(ip net.IP) bool {
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	for _, n := range blockedNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// resolver is the part of net.Resolver the safe dialer needs; tests replace it.
type resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

// safeDialer resolves the host itself, rejects the connection if any resolved
// address is blocked, and then dials the address it validated. Dialing the
// validated IP (not the name) closes the DNS-rebinding window between check and
// connect.
type safeDialer struct {
	resolver resolver
	dialer   net.Dialer
}

func (d *safeDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	addrs, err := d.resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("host %q has no address", host)
	}
	for _, a := range addrs {
		if blockedIP(a.IP) {
			return nil, fmt.Errorf("host %q resolves to %s, which is not allowed for remote sources", host, a.IP)
		}
	}
	return d.dialer.DialContext(ctx, network, net.JoinHostPort(addrs[0].IP.String(), port))
}

// Fetcher downloads remote model files. Create it with NewFetcher; tests build
// it with their own client.
type Fetcher struct {
	client   *http.Client
	maxBytes int64
	getenv   func(string) string
	// stub replaces the download after the URL has been validated (tests).
	stub func(ctx context.Context, rawURL string) ([]byte, error)
}

// NewFetcher returns a Fetcher with the safe dialer: HTTPS only (enforced by the
// caller), no redirects, a 30 second timeout and a size limit equal to the
// model file limit.
func NewFetcher() *Fetcher {
	sd := &safeDialer{resolver: net.DefaultResolver, dialer: net.Dialer{Timeout: 10 * time.Second}}
	return &Fetcher{
		client: &http.Client{
			Timeout: fetchTimeout,
			Transport: &http.Transport{
				DialContext:           sd.DialContext,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 15 * time.Second,
				// A proxy would defeat the address check, so none is used.
				Proxy: nil,
			},
		},
		maxBytes: model.MaxModelFileSize,
		getenv:   os.Getenv,
	}
}

// TokenEnvName returns the environment variable that holds the token for host:
// BAUSTEINSICHT_FETCH_TOKEN_ plus the upper-cased host with every character
// outside A-Z and 0-9 replaced by '_' (raw.githubusercontent.com becomes
// BAUSTEINSICHT_FETCH_TOKEN_RAW_GITHUBUSERCONTENT_COM).
func TokenEnvName(host string) string {
	var b strings.Builder
	b.WriteString(tokenEnvPrefix)
	for _, r := range strings.ToUpper(host) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

// sourceHost validates a source URL (https, no credentials, no fragment, port
// 443 only) and returns its lower-case host.
func sourceHost(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid source URL: %w", err)
	}
	if u.Scheme != "https" {
		return "", fmt.Errorf("source %q: only https URLs are allowed", raw)
	}
	if u.User != nil {
		return "", fmt.Errorf("source %q: credentials in the URL are not allowed; use %s<HOST>", raw, tokenEnvPrefix)
	}
	if u.Hostname() == "" {
		return "", fmt.Errorf("source %q: host is missing", raw)
	}
	if p := u.Port(); p != "" && p != "443" {
		return "", fmt.Errorf("source %q: only port 443 is allowed", raw)
	}
	if u.Fragment != "" {
		return "", fmt.Errorf("source %q: a fragment is not allowed", raw)
	}
	return strings.ToLower(u.Hostname()), nil
}

// Fetch downloads rawURL. The caller has already checked the host against the
// allowlist. Redirects are not followed: a redirect target would bypass both
// the allowlist and the address check, so it is reported as an error.
func (f *Fetcher) Fetch(ctx context.Context, rawURL string) ([]byte, error) {
	host, err := sourceHost(rawURL)
	if err != nil {
		return nil, err
	}
	if f.stub != nil {
		return f.stub(ctx, rawURL)
	}
	return f.get(ctx, rawURL, host)
}

// get performs the request for an already validated URL.
func (f *Fetcher) get(ctx context.Context, rawURL, host string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	if tok := f.getenv(TokenEnvName(host)); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	client := *f.client
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", rawURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return nil, fmt.Errorf("fetching %s: redirect to %q is not followed; use the final URL as source", rawURL, resp.Header.Get("Location"))
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching %s: HTTP %d", rawURL, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, f.maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", rawURL, err)
	}
	if int64(len(data)) > f.maxBytes {
		return nil, fmt.Errorf("fetching %s: larger than %d bytes", rawURL, f.maxBytes)
	}
	return data, nil
}

// FetchResult describes what Sync did for one remote model.
type FetchResult struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	// Status is "fetched" (downloaded and locked), "restored" (cache rebuilt
	// from the lock), "updated" (lock changed by --update) or "up-to-date".
	Status string `json:"status"`
}

// Sync downloads the remote models of cfg into the cache next to the workspace
// configuration and maintains the lock file. Models that are already cached and
// match the lock are not downloaded again. A download whose hash differs from
// the lock is an error unless update is set.
func Sync(ctx context.Context, cfg *Config, configPath string, f *Fetcher, update bool) ([]FetchResult, error) {
	if problems := ValidateSources(cfg); len(problems) > 0 {
		return nil, errors.Join(problems...)
	}
	baseDir := filepath.Dir(configPath)
	lock, err := LoadLock(LockPath(configPath))
	if err != nil {
		return nil, err
	}

	var results []FetchResult
	remote := make(map[string]bool)
	for _, ref := range cfg.Models {
		if ref.Source == "" {
			continue
		}
		remote[ref.ID] = true
		res, err := syncOne(ctx, ref, baseDir, lock, f, update)
		if err != nil {
			return results, err
		}
		results = append(results, res)
	}
	pruned := false
	for _, id := range lock.sortedIDs() {
		if !remote[id] {
			delete(lock.Sources, id)
			removeCacheFile(baseDir, id)
			pruned = true
		}
	}
	if len(results) == 0 && !pruned {
		return results, nil
	}
	if err := SaveLock(LockPath(configPath), lock); err != nil {
		return results, fmt.Errorf("writing lock file: %w", err)
	}
	return results, nil
}

func syncOne(ctx context.Context, ref ModelRef, baseDir string, lock *Lock, f *Fetcher, update bool) (FetchResult, error) {
	res := FetchResult{ID: ref.ID, URL: ref.Source}
	entry, locked := lock.Sources[ref.ID]
	locked = locked && entry.URL == ref.Source
	cache := CachePath(baseDir, ref.ID)

	if locked && !update {
		if data, err := os.ReadFile(cache); err == nil && hashHex(data) == entry.SHA256 {
			res.SHA256, res.Status = entry.SHA256, "up-to-date"
			return res, nil
		}
	}

	data, err := f.Fetch(ctx, ref.Source)
	if err != nil {
		return res, err
	}
	// Status 200 does not make a model: a README, an error page or a login page
	// must not be cached and pinned (#734).
	if err := checkIsModel(ref.Source, data); err != nil {
		return res, err
	}
	got := hashHex(data)
	switch {
	case !locked:
		res.Status = "fetched"
	case got == entry.SHA256:
		res.Status = "restored"
		if update {
			res.Status = "up-to-date"
		}
	case update:
		res.Status = "updated"
	default:
		return res, fmt.Errorf("remote model %q: the content of %s changed (locked sha256 %s, got %s); review it and run 'bausteinsicht workspace fetch --update' to accept", ref.ID, ref.Source, entry.SHA256, got)
	}
	if err := writeFileAtomic(cache, data); err != nil {
		return res, fmt.Errorf("writing cache of remote model %q: %w", ref.ID, err)
	}
	// Keep the cache out of git without relying on the user's .gitignore.
	if err := os.WriteFile(filepath.Join(filepath.Dir(cache), ".gitignore"), []byte("*\n"), 0o600); err != nil {
		return res, fmt.Errorf("writing cache .gitignore: %w", err)
	}
	lock.Sources[ref.ID] = LockEntry{URL: ref.Source, SHA256: got}
	res.SHA256 = got
	return res, nil
}

// checkIsModel parses the downloaded bytes as a model, with the loader that local
// files go through, without writing anything. A file that parses but holds neither
// a specification nor a model (a JSON error object, say) is not one either.
func checkIsModel(source string, data []byte) error {
	m, err := model.ParseModel(data)
	if err != nil {
		return fmt.Errorf("%s is not a Bausteinsicht model: %w", source, err)
	}
	if len(m.Specification.Elements) == 0 && len(m.Model) == 0 {
		return fmt.Errorf("%s is not a Bausteinsicht model: it has neither a specification nor a model", source)
	}
	return nil
}

// removeCacheFile deletes the cache file of a model that is no longer remote. Only
// a file named <id>.jsonc in the cache directory is touched: an ID of the lock file
// that could not be one (a path, say) is ignored (#735).
func removeCacheFile(baseDir, id string) {
	if !sourceIDPattern.MatchString(id) {
		return
	}
	_ = os.Remove(CachePath(baseDir, id))
}

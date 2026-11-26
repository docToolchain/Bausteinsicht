package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	// LockFileName is the lock file written next to the workspace configuration.
	LockFileName = "bausteinsicht.lock"
	// CacheDirName is the project-local directory that holds fetched models.
	CacheDirName = ".bausteinsicht-cache"

	lockVersion = 1
)

// sourceIDPattern restricts the IDs of remote models: the ID becomes a file name
// in the cache directory, so it must not contain path separators or dots.
var sourceIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

// Lock records the SHA-256 of every fetched model file.
type Lock struct {
	Version int                  `json:"version"`
	Sources map[string]LockEntry `json:"sources"`
}

// LockEntry pins one remote model.
type LockEntry struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

// LockPath returns the lock file path for a workspace configuration file.
func LockPath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), LockFileName)
}

// CachePath returns the cache file of a remote model.
func CachePath(baseDir, id string) string {
	return filepath.Join(baseDir, CacheDirName, id+".jsonc")
}

// LoadLock reads the lock file. A missing file yields an empty lock.
func LoadLock(path string) (*Lock, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Lock{Version: lockVersion, Sources: map[string]LockEntry{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading lock file: %w", err)
	}
	var l Lock
	if err := json.Unmarshal(data, &l); err != nil {
		return nil, fmt.Errorf("parsing lock file %s: %w", path, err)
	}
	if l.Version != lockVersion {
		return nil, fmt.Errorf("lock file %s has version %d, expected %d", path, l.Version, lockVersion)
	}
	if l.Sources == nil {
		l.Sources = map[string]LockEntry{}
	}
	return &l, nil
}

// SaveLock writes the lock file atomically with owner-only permissions. The
// JSON encoder sorts map keys, so the file is stable and diff-friendly.
func SaveLock(path string, l *Lock) error {
	l.Version = lockVersion
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling lock file: %w", err)
	}
	return writeFileAtomic(path, append(data, '\n'))
}

// hashHex returns the SHA-256 of data as lower-case hex.
func hashHex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// writeFileAtomic writes data to path through a temp file in the same
// directory, with 0600 permissions.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Chmod(name, 0o600); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		_ = os.Remove(name)
		return err
	}
	return nil
}

// ValidateSources checks the remote model references of a configuration without
// touching the network or the cache: exactly one of path/source, a safe ID, an
// https URL on an allowed host.
func ValidateSources(cfg *Config) []error {
	var problems []error
	allowed := make(map[string]bool, len(cfg.Workspace.AllowedHosts))
	for _, h := range cfg.Workspace.AllowedHosts {
		if h == "" || strings.ContainsAny(h, "*:/@ ") || strings.ToLower(h) != h {
			problems = append(problems, fmt.Errorf("allowedHosts entry %q must be a lower-case host name without scheme, port or wildcard", h))
			continue
		}
		allowed[h] = true
	}
	for _, ref := range cfg.Models {
		switch {
		case ref.Path != "" && ref.Source != "":
			problems = append(problems, fmt.Errorf("model %q: set either path or source, not both", ref.ID))
		case ref.Path == "" && ref.Source == "":
			problems = append(problems, fmt.Errorf("model %q: path or source is required", ref.ID))
		case ref.Source != "":
			if err := validateSource(ref, allowed); err != nil {
				problems = append(problems, err)
			}
		}
	}
	return problems
}

func validateSource(ref ModelRef, allowed map[string]bool) error {
	if !sourceIDPattern.MatchString(ref.ID) {
		return fmt.Errorf("model %q: the ID of a remote model may contain only letters, digits, '-' and '_'", ref.ID)
	}
	host, err := sourceHost(ref.Source)
	if err != nil {
		return fmt.Errorf("model %q: %w", ref.ID, err)
	}
	if !allowed[host] {
		return fmt.Errorf("model %q: host %q is not listed in workspace.allowedHosts", ref.ID, host)
	}
	return nil
}

// resolveModelFile resolves a model reference to the file to load. A remote model
// is read from the cache and only after its SHA-256 matches the lock file.
func resolveModelFile(ref ModelRef, baseDir string, lock *Lock) (string, error) {
	if ref.Source == "" {
		if filepath.IsAbs(ref.Path) {
			return ref.Path, nil
		}
		return filepath.Join(baseDir, ref.Path), nil
	}
	entry, ok := lock.Sources[ref.ID]
	if !ok || entry.URL != ref.Source {
		return "", fmt.Errorf("remote model %q is not locked; run 'bausteinsicht workspace fetch' first", ref.ID)
	}
	file := CachePath(baseDir, ref.ID)
	data, err := os.ReadFile(file)
	if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("remote model %q is not in the cache; run 'bausteinsicht workspace fetch' first", ref.ID)
	}
	if err != nil {
		return "", fmt.Errorf("reading cache of remote model %q: %w", ref.ID, err)
	}
	if got := hashHex(data); got != entry.SHA256 {
		return "", fmt.Errorf("cache of remote model %q does not match %s (expected sha256 %s, got %s); run 'bausteinsicht workspace fetch' to restore it", ref.ID, LockFileName, entry.SHA256, got)
	}
	return file, nil
}

// sortedIDs returns the keys of a lock in a stable order.
func (l *Lock) sortedIDs() []string {
	ids := make([]string, 0, len(l.Sources))
	for id := range l.Sources {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

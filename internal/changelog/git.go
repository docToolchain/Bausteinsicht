package changelog

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// LoadModelAtGitRef loads a model from a specific git reference
func LoadModelAtGitRef(modelPath, gitRef string) (*model.BausteinsichtModel, error) {
	// Resolve the git ref to its full commit hash
	hash, _, err := resolveGitRef(gitRef)
	if err != nil {
		return nil, fmt.Errorf("resolving git ref %q: %w", gitRef, err)
	}

	// Load model from git
	m, err := loadModelFromGit(modelPath, hash)
	if err != nil {
		return nil, fmt.Errorf("loading model from git ref %q: %w", gitRef, err)
	}

	// Store metadata for changelog generation
	if m == nil {
		m = &model.BausteinsichtModel{}
	}

	return m, nil
}

// gitPath resolves the absolute path of the git binary via PATH lookup.
// This satisfies go:S4036 — the resolved path is fixed and unwriteable at call time.
func gitPath() (string, error) {
	return exec.LookPath("git")
}

// resolveGitRef converts a git ref (tag, branch, commit) to its hash and timestamp
func resolveGitRef(gitRef string) (hash string, date time.Time, err error) {
	git, err := gitPath()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("git not available: %w", err)
	}

	// Get commit hash
	hash, err = runGit("rev-parse", "--verify", gitRef+"^{commit}")
	if err != nil {
		return "", time.Time{}, fmt.Errorf("cannot resolve git ref %q: %w", gitRef, err)
	}

	// Get commit date
	dateCmd := exec.Command(git, "log", "-1", "--format=%ct", hash)
	dateOut, err := dateCmd.Output()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to get commit date: %w", err)
	}

	timestamp, err := strconv.ParseInt(strings.TrimSpace(string(dateOut)), 10, 64)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to parse commit timestamp: %w", err)
	}

	date = time.Unix(timestamp, 0).UTC()
	return hash, date, nil
}

// loadModelFromGit retrieves the model file from git at a specific commit
func loadModelFromGit(modelPath, commitHash string) (*model.BausteinsichtModel, error) {
	git, err := gitPath()
	if err != nil {
		return nil, fmt.Errorf("git not available: %w", err)
	}
	cmd := exec.Command(git, "show", commitHash+":"+modelPath)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git show failed: %w", err)
	}

	// Strip JSONC comments and parse
	clean := model.StripJSONC(out)
	trimmed := strings.TrimSpace(string(clean))
	if trimmed == "null" || trimmed == "" {
		return nil, fmt.Errorf("model file is empty or null at %s in %s", modelPath, commitHash)
	}

	var m model.BausteinsichtModel
	if err := json.Unmarshal(clean, &m); err != nil {
		return nil, fmt.Errorf("parsing model: %w", err)
	}

	m.ElementOrder = extractElementOrder(clean)
	return &m, nil
}

// extractElementOrder extracts the definition order of element kinds from specification
func extractElementOrder(data []byte) []string {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	specRaw, ok := raw["specification"]
	if !ok {
		return nil
	}
	var spec map[string]json.RawMessage
	if err := json.Unmarshal(specRaw, &spec); err != nil {
		return nil
	}
	elemsRaw, ok := spec["elements"]
	if !ok {
		return nil
	}
	var elems map[string]interface{}
	d := json.NewDecoder(strings.NewReader(string(elemsRaw)))
	d.UseNumber()
	if err := d.Decode(&elems); err != nil {
		return nil
	}
	var order []string
	for k := range elems {
		order = append(order, k)
	}
	return order
}

// GetCommitInfo retrieves metadata about a git commit
func GetCommitInfo(gitRef string) (*CommitInfo, error) {
	hash, date, err := resolveGitRef(gitRef)
	if err != nil {
		return nil, err
	}

	git, err := gitPath()
	if err != nil {
		return nil, fmt.Errorf("git not available: %w", err)
	}

	// Get author
	authorCmd := exec.Command(git, "log", "-1", "--format=%an", hash)
	authorOut, err := authorCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commit author: %w", err)
	}

	// Get message
	msgCmd := exec.Command(git, "log", "-1", "--format=%s", hash)
	msgOut, err := msgCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commit message: %w", err)
	}

	return &CommitInfo{
		Hash:      hash,
		Author:    strings.TrimSpace(string(authorOut)),
		Date:      date,
		Message:   strings.TrimSpace(string(msgOut)),
		Timestamp: date.Unix(),
	}, nil
}

// errNoRepository is the error for a command that runs outside a git repository.
var errNoRepository = errors.New("no git repository found (changelog compares git refs; run it inside the repository of the model)")

// runGit runs git and returns its output. A failure carries git's own message
// (its first line), not just the exit status; outside a repository the error says so.
func runGit(args ...string) (string, error) {
	git, err := gitPath()
	if err != nil {
		return "", fmt.Errorf("git not available: %w", err)
	}
	cmd := exec.Command(git, args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if strings.Contains(msg, "not a git repository") {
			return "", errNoRepository
		}
		if first, _, _ := strings.Cut(msg, "\n"); first != "" {
			return "", fmt.Errorf("%s", first)
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// PreviousTag returns the most recent tag before until, the default of --since:
// the latest tag reachable from until, or, if until itself is tagged, the one
// before it.
func PreviousTag(until string) (string, error) {
	tag, err := runGit("describe", "--tags", "--abbrev=0", until)
	if err != nil {
		if errors.Is(err, errNoRepository) {
			return "", err
		}
		return "", noTagError(until)
	}
	if tagHash, err := runGit("rev-list", "-1", tag); err == nil {
		if untilHash, err := runGit("rev-parse", until+"^{commit}"); err == nil && tagHash == untilHash {
			if tag, err = runGit("describe", "--tags", "--abbrev=0", tag+"^"); err != nil {
				return "", noTagError(until)
			}
		}
	}
	return tag, nil
}

func noTagError(until string) error {
	return fmt.Errorf("no tag found before %s: tag a release or use --since <git ref>", until)
}

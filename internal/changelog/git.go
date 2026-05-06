package changelog

import (
	"encoding/json"
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

// resolveGitRef converts a git ref (tag, branch, commit) to its hash and timestamp
func resolveGitRef(gitRef string) (hash string, date time.Time, err error) {
	// Get commit hash
	hashCmd := exec.Command("git", "rev-parse", gitRef)
	hashOut, err := hashCmd.Output()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to resolve git ref: %w", err)
	}
	hash = strings.TrimSpace(string(hashOut))

	// Get commit date
	dateCmd := exec.Command("git", "log", "-1", "--format=%ct", hash)
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
	cmd := exec.Command("git", "show", commitHash+":"+modelPath)
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

	// Get author
	authorCmd := exec.Command("git", "log", "-1", "--format=%an", hash)
	authorOut, err := authorCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commit author: %w", err)
	}

	// Get message
	msgCmd := exec.Command("git", "log", "-1", "--format=%s", hash)
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

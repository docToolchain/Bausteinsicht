package stale

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// gitPath resolves the absolute path of the git binary via PATH lookup.
// This satisfies go:S4036 — the resolved path is fixed and unwriteable at call time.
func gitPath() (string, error) {
	return exec.LookPath("git")
}

// GetLastModifiedDate returns the date of the last git commit that touched the given file.
// If the file is not in a git repository or has never been committed, returns zero time.
func GetLastModifiedDate(filePath string) (time.Time, error) {
	// Get the absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return time.Time{}, fmt.Errorf("resolving path: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(absPath); err != nil {
		return time.Time{}, fmt.Errorf("file not found: %w", err)
	}

	git, err := gitPath()
	if err != nil {
		return time.Time{}, nil // git not available
	}

	// Run git log in the file's directory so the correct repo is used.
	// --follow: follow file renames
	// -1: get only the latest commit
	// --format=%aI: ISO 8601 strict format
	cmd := exec.Command(git, "log", "--follow", "-1", "--format=%aI", "--", absPath)
	cmd.Dir = filepath.Dir(absPath)
	output, err := cmd.Output()
	if err != nil {
		// File might not be tracked in git
		return time.Time{}, nil
	}

	dateStr := strings.TrimSpace(string(output))
	if dateStr == "" {
		// File not tracked
		return time.Time{}, nil
	}

	// Parse ISO 8601 format
	parsedTime, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing git date %q: %w", dateStr, err)
	}

	return parsedTime, nil
}

// DaysSince calculates the number of days between a past time and now.
// Returns 0 for future timestamps.
func DaysSince(t time.Time) int {
	if t.IsZero() {
		return 0
	}
	d := int(time.Since(t).Hours() / 24)
	if d < 0 {
		return 0
	}
	return d
}

// IsStale checks if a date is older than the threshold days.
func IsStale(lastModified time.Time, thresholdDays int) bool {
	if lastModified.IsZero() {
		return false // Untracked files are not considered stale
	}
	daysSince := DaysSince(lastModified)
	return daysSince >= thresholdDays
}

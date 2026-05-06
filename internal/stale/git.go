package stale

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

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

	// Get git log for the file
	// --follow: follow file renames
	// -1: get only the latest commit
	// --format=%aI: ISO 8601 strict format
	cmd := exec.Command("git", "log", "--follow", "-1", "--format=%aI", "--", absPath)
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
func DaysSince(t time.Time) int {
	if t.IsZero() {
		return 0
	}
	return int(time.Since(t).Hours() / 24)
}

// IsStale checks if a date is older than the threshold days.
func IsStale(lastModified time.Time, thresholdDays int) bool {
	if lastModified.IsZero() {
		return false // Untracked files are not considered stale
	}
	daysSince := DaysSince(lastModified)
	return daysSince >= thresholdDays
}

// GetLastModifiedDateForElement returns the date when an element was last modified in git.
// It searches git history for the most recent change that touched the element key.
// If per-element tracking is not possible, falls back to file-level modification date.
func GetLastModifiedDateForElement(filePath string, elementKey string) (time.Time, error) {
	// First, get the file's last modified date as fallback
	fileMod, err := GetLastModifiedDate(filePath)
	if err != nil {
		return time.Time{}, err
	}

	// Try to find element-specific changes via git log with pattern search
	// Search for commits that modified this specific element key in the JSON
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fileMod, nil // Fallback to file-level
	}

	// Build a regex pattern to search for this element key in JSON
	// e.g. for "api.backend" search for "\"backend\"" or "'backend'"
	keyParts := strings.Split(elementKey, ".")
	if len(keyParts) == 0 {
		return fileMod, nil
	}
	searchKey := keyParts[len(keyParts)-1] // Use the leaf key for less ambiguity

	// git log -S: search for given string in diffs
	// --follow: follow file renames
	// -1: get most recent
	// --format=%aI: ISO 8601 format
	cmd := exec.Command("git", "log", "-S", "\""+searchKey+"\"", "--follow", "-1", "--format=%aI", "--", absPath)
	output, err := cmd.Output()
	if err != nil {
		// Element key not found in git history, use file-level
		return fileMod, nil
	}

	dateStr := strings.TrimSpace(string(output))
	if dateStr == "" {
		return fileMod, nil // Fallback to file-level
	}

	parsedTime, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return fileMod, nil // Fallback to file-level on parse error
	}

	return parsedTime, nil
}

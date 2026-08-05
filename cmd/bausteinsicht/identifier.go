package main

import "regexp"

// validKeyPattern is the shared identifier form for element IDs, specification
// keys, and view keys: a letter followed by letters, digits, hyphens, or
// underscores. Dots are NOT allowed - they serve as element-hierarchy
// separators. Defined once so the callers cannot drift apart again: they did,
// which is how specification and view keys ended up lowercase-only while
// element IDs allowed camelCase (#582).
var validKeyPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

// isValidKey reports whether s is a valid element ID, specification key, or view key.
func isValidKey(s string) bool {
	return validKeyPattern.MatchString(s)
}

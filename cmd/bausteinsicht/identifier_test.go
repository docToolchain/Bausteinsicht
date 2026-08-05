package main

import "testing"

// TestIsValidKey covers the shared identifier rule used for element IDs,
// specification keys, and view keys (consolidated in #582).
func TestIsValidKey(t *testing.T) {
	tests := []struct {
		key   string
		valid bool
	}{
		{"system", true},
		{"my_component", true},
		{"my-component", true},
		{"custom123", true},
		{"orderRecord", true},        // camelCase element ID
		{"boundaryObject", true},     // camelCase specification kind (#582)
		{"applicationService", true}, // camelCase specification kind (#582)
		{"systemContext", true},      // camelCase view key (#582)
		{"Component", true},          // leading uppercase allowed
		{"_invalid", false},          // starts with underscore
		{"123invalid", false},        // starts with digit
		{"has.dot", false},           // dots reserved for hierarchy separators
		{"my component", false},      // space
		{"", false},                  // empty
	}

	for _, tt := range tests {
		if got := isValidKey(tt.key); got != tt.valid {
			t.Errorf("isValidKey(%q) = %v, want %v", tt.key, got, tt.valid)
		}
	}
}

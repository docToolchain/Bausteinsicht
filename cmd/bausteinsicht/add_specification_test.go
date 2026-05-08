package main

import (
	"testing"
)

func TestIsValidSpecKey(t *testing.T) {
	tests := []struct {
		key   string
		valid bool
	}{
		{"system", true},
		{"my_component", true},
		{"my-component", true},
		{"custom123", true},
		{"_invalid", false},    // starts with underscore
		{"123invalid", false},  // starts with digit
		{"Component", false},   // uppercase
		{"my component", false}, // space
		{"", false},            // empty
	}

	for _, tt := range tests {
		got := isValidSpecKey(tt.key)
		if got != tt.valid {
			t.Errorf("isValidSpecKey(%q) = %v, want %v", tt.key, got, tt.valid)
		}
	}
}

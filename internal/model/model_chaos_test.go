package model

import (
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/chaos"
)

// TestLoadCorruptJSONC tests model loading with corrupted input.
func TestLoadCorruptJSONC(t *testing.T) {
	tc := chaos.NewTestChaos(t)

	testCases := []string{
		"{invalid json}",
		"{\"model\": }",  // incomplete
		"[not an object]",
		"",               // empty
		"{\"spec\": ",    // truncated
	}

	for _, invalid := range testCases {
		path := tc.CreateFileWithContent("corrupt.jsonc", invalid)
		_, err := Load(path)
		if err == nil {
			t.Fatalf("Should reject invalid JSON: %s", invalid)
		}
	}
}

// TestLoadMissingFile tests model loading when file doesn't exist.
func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/model.jsonc")
	if err == nil {
		t.Fatal("Should error on missing file")
	}
}

// TestLoadEmptyFile tests model loading with empty file.
func TestLoadEmptyFile(t *testing.T) {
	tc := chaos.NewTestChaos(t)
	path := tc.CreateEmptyFile("empty.jsonc")
	_, err := Load(path)
	if err == nil {
		t.Fatal("Should reject empty file")
	}
}

// TestLoadReadOnlyFile tests model loading from read-only file.
func TestLoadReadOnlyFile(t *testing.T) {
	tc := chaos.NewTestChaos(t)
	path := tc.CreateFileWithContent("readonly.jsonc", `{
		"specification": {
			"elements": {"actor": {"notation": "Actor"}},
			"relationships": {}
		},
		"model": {"user": {"kind": "actor"}},
		"views": {"context": {"title": "Context", "include": ["user"]}}
	}`)

	tc.MakeReadOnly(path)
	defer tc.MakeWritable(path)

	// Should still be readable despite read-only flag
	m, err := Load(path)
	if err != nil {
		t.Fatalf("Should read read-only file: %v", err)
	}
	if m == nil {
		t.Fatal("Model should not be nil")
	}
}

// TestValidModelStructure tests that loaded model has expected structure.
func TestValidModelStructure(t *testing.T) {
	tc := chaos.NewTestChaos(t)
	path := tc.CreateFileWithContent("valid.jsonc", `{
		"specification": {
			"elements": {"actor": {"notation": "Actor"}},
			"relationships": {}
		},
		"model": {"user": {"kind": "actor", "title": "User"}},
		"views": {"context": {"title": "Context", "include": ["user"]}}
	}`)

	m, err := Load(path)
	if err != nil {
		t.Fatalf("Load valid model: %v", err)
	}

	if m == nil {
		t.Fatal("Model should not be nil")
	}

	if len(m.Model) == 0 {
		t.Fatal("Model should have elements")
	}

	if len(m.Views) == 0 {
		t.Fatal("Model should have views")
	}
}

package main

import (
	"os"
	"testing"
)

func createTempFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "test-*.jsonc")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	return tmpFile.Name()
}

func removeTempFile(t *testing.T, path string) {
	if err := os.Remove(path); err != nil {
		t.Logf("Failed to remove temp file: %v", err)
	}
}

func TestDiff_NoAsIsToBe(t *testing.T) {
	// Create temp model without asIs/toBe
	modelContent := `{
    "specification": {
      "elements": {
        "system": { "notation": "Software System" }
      }
    },
    "model": {
      "mysystem": { "kind": "system", "title": "My System" }
    },
    "views": {}
  }`

	tmpFile := createTempFile(t, modelContent)
	defer removeTempFile(t, tmpFile)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"diff", "--model", tmpFile, "--format", "text"})
	err := cmd.Execute()

	if err == nil {
		t.Error("Expected error for model without asIs/toBe")
	}
}

func TestDiff_TextFormat(t *testing.T) {
	// Create model with asIs/toBe
	modelContent := `{
    "specification": {
      "elements": {
        "system": { "notation": "Software System" }
      }
    },
    "model": {
      "mysystem": { "kind": "system", "title": "My System" }
    },
    "asIs": {
      "elements": {
        "legacy": { "kind": "system", "title": "Legacy System" }
      },
      "relationships": []
    },
    "toBe": {
      "elements": {
        "modern": { "kind": "system", "title": "Modern System" }
      },
      "relationships": []
    },
    "views": {}
  }`

	tmpFile := createTempFile(t, modelContent)
	defer removeTempFile(t, tmpFile)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"diff", "--model", tmpFile, "--format", "text"})
	err := cmd.Execute()

	if err != nil {
		t.Errorf("diff command failed: %v", err)
	}
}

func TestDiff_JSONFormat(t *testing.T) {
	modelContent := `{
    "specification": {
      "elements": {
        "system": { "notation": "Software System" }
      }
    },
    "model": {
      "mysystem": { "kind": "system", "title": "My System" }
    },
    "asIs": {
      "elements": {
        "api": { "kind": "container", "title": "API v1" }
      },
      "relationships": []
    },
    "toBe": {
      "elements": {
        "api": { "kind": "container", "title": "API v2" }
      },
      "relationships": []
    },
    "views": {}
  }`

	tmpFile := createTempFile(t, modelContent)
	defer removeTempFile(t, tmpFile)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"diff", "--model", tmpFile, "--format", "json"})
	err := cmd.Execute()

	if err != nil {
		t.Errorf("diff command failed: %v", err)
	}
}

func TestDiff_InvalidFormat(t *testing.T) {
	modelContent := `{
    "specification": {
      "elements": {
        "system": { "notation": "Software System" }
      }
    },
    "asIs": {
      "elements": {},
      "relationships": []
    },
    "toBe": {
      "elements": {},
      "relationships": []
    },
    "views": {}
  }`

	tmpFile := createTempFile(t, modelContent)
	defer removeTempFile(t, tmpFile)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"diff", "--model", tmpFile, "--format", "invalid"})
	err := cmd.Execute()

	if err == nil {
		t.Error("Expected error for invalid format")
	}
}

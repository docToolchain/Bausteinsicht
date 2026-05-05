package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const staleTestModel = `{
  "specification": {
    "elements": {
      "system": {"notation": "System", "container": true},
      "container": {"notation": "Container"},
      "actor": {"notation": "Actor"}
    }
  },
  "model": {
    "user": {
      "kind": "actor",
      "title": "User",
      "description": "End user"
    },
    "shop": {
      "kind": "system",
      "title": "Shop",
      "description": "E-commerce system",
      "children": {
        "api": {
          "kind": "container",
          "title": "API",
          "description": "REST API"
        },
        "db": {
          "kind": "container",
          "title": "Database",
          "description": "PostgreSQL"
        }
      }
    },
    "legacy": {
      "kind": "system",
      "title": "Legacy System",
      "description": "Old monolith",
      "status": "deprecated"
    }
  },
  "relationships": [
    {"from": "user", "to": "shop.api", "label": "calls"}
  ],
  "views": {
    "context": {
      "title": "System Context",
      "include": ["user", "shop"]
    }
  }
}`

func writeStaleTestModel(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "architecture.jsonc")
	if err := os.WriteFile(path, []byte(staleTestModel), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestStaleCmd_Help(t *testing.T) {
	out, err := executeRootCmd("stale", "--help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "stale") {
		t.Errorf("expected help text with 'stale' in output")
	}
	if !strings.Contains(out, "--days") || !strings.Contains(out, "--format") {
		t.Errorf("expected flag documentation in help output")
	}
}

func TestStaleCmd_TextFormat(t *testing.T) {
	modelPath := writeStaleTestModel(t)
	// Note: Since the model file is in a temp directory not tracked by git,
	// no elements will be flagged as stale (we can't determine git age for untracked files).
	// This is expected behavior - stale detection requires git history.
	out, err := executeRootCmd("stale", "--model", modelPath, "--days", "90")
	if err != nil {
		t.Fatalf("unexpected error: %v\nOutput: %s", err, out)
	}

	// Verify text output format
	if !strings.Contains(out, "stale") || !strings.Contains(out, "elements") {
		t.Errorf("expected output mentioning stale elements, got:\n%s", out)
	}
}

func TestStaleCmd_JSONFormat(t *testing.T) {
	modelPath := writeStaleTestModel(t)
	out, err := executeRootCmd("stale", "--model", modelPath, "--days", "0", "--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v\nOutput: %s", err, out)
	}

	// Verify JSON output
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Errorf("output is not valid JSON: %v\nOutput: %s", err, out)
	}

	// Check result structure
	if _, ok := result["StaleElements"]; !ok {
		t.Errorf("expected 'StaleElements' in JSON, got keys: %v", result)
	}

	if _, ok := result["TotalElements"]; !ok {
		t.Errorf("expected 'TotalElements' in JSON, got keys: %v", result)
	}
}

func TestStaleCmd_InvalidDays(t *testing.T) {
	modelPath := writeStaleTestModel(t)
	_, err := executeRootCmd("stale", "--model", modelPath, "--days", "-1")
	if err == nil {
		t.Fatal("expected error for negative --days")
	}
}

func TestStaleCmd_InvalidFormat(t *testing.T) {
	modelPath := writeStaleTestModel(t)
	_, err := executeRootCmd("stale", "--model", modelPath, "--format", "xml")
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
}

func TestStaleCmd_NonExistentModel(t *testing.T) {
	_, err := executeRootCmd("stale", "--model", "/nonexistent/model.jsonc")
	if err == nil {
		t.Fatal("expected error for non-existent model")
	}
}

func TestStaleCmd_InvalidPathTraversal(t *testing.T) {
	_, err := executeRootCmd("stale", "--model", "../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal attempt")
	}
}

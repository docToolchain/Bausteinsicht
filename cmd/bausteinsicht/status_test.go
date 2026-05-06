package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusCmd_Help(t *testing.T) {
	out, err := executeRootCmd("status", "--help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "status") {
		t.Error("expected 'status' in help output")
	}
	if !strings.Contains(out, "--filter") {
		t.Error("expected --filter flag in help output")
	}
}

func TestStatusCmd_TextFormat(t *testing.T) {
	modelPath := writeStatusTestModel(t)
	out, err := executeRootCmd("status", "--model", modelPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Element Lifecycle Status") {
		t.Error("expected title in output")
	}
	if !strings.Contains(out, "proposed") {
		t.Error("expected status categories in output")
	}
}

func TestStatusCmd_JSONFormat(t *testing.T) {
	modelPath := writeStatusTestModel(t)
	out, err := executeRootCmd("status", "--model", modelPath, "--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result statusResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Errorf("invalid JSON output: %v\nOutput: %s", err, out)
	}

	if len(result.Summary) == 0 {
		t.Error("expected summary in JSON output")
	}
}

func TestStatusCmd_FilterByStatus(t *testing.T) {
	modelPath := writeStatusTestModel(t)
	out, err := executeRootCmd("status", "--model", modelPath, "--filter", "deployed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "deployed") {
		t.Error("expected 'deployed' status in filtered output")
	}
}

func TestStatusCmd_InvalidFilter(t *testing.T) {
	modelPath := writeStatusTestModel(t)
	_, err := executeRootCmd("status", "--model", modelPath, "--filter", "invalid-status")
	if err == nil {
		t.Fatal("expected error for invalid filter")
	}
}

func TestStatusCmd_NonExistentModel(t *testing.T) {
	_, err := executeRootCmd("status", "--model", "/nonexistent/model.jsonc")
	if err == nil {
		t.Fatal("expected error for non-existent model")
	}
}

func TestStatusCmd_InvalidPathTraversal(t *testing.T) {
	_, err := executeRootCmd("status", "--model", "../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal attempt")
	}
}

func writeStatusTestModel(t *testing.T) string {
	t.Helper()
	const model = `{
  "specification": {
    "elements": {
      "service": { "notation": "Service", "container": true },
      "database": { "notation": "Database" },
      "actor": { "notation": "Actor" }
    }
  },
  "model": {
    "user": {
      "kind": "actor",
      "title": "User",
      "status": "deployed"
    },
    "payment-v1": {
      "kind": "service",
      "title": "Payment Service v1",
      "status": "deprecated"
    },
    "payment-v2": {
      "kind": "service",
      "title": "Payment Service v2",
      "status": "implementation"
    },
    "notification": {
      "kind": "service",
      "title": "Notification Service",
      "status": "proposed"
    },
    "db": {
      "kind": "database",
      "title": "Main Database"
    }
  },
  "relationships": [],
  "views": {}
}`
	dir := t.TempDir()
	path := filepath.Join(dir, "model.jsonc")
	if err := os.WriteFile(path, []byte(model), 0644); err != nil {
		t.Fatalf("failed to write test model: %v", err)
	}
	return path
}

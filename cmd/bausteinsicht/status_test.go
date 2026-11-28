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

func TestStatusCmd_MissingModelInParentDirectory(t *testing.T) {
	_, err := executeRootCmd("status", "--model", "../../no-such-dir/model.jsonc")
	if err == nil {
		t.Fatal("expected error for a missing model")
	}
	if strings.Contains(err.Error(), "traversal") {
		t.Fatalf("a parent directory must not be rejected as traversal: %v", err)
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

const statusUnknownModel = `{"specification":{"elements":{"system":{"notation":"S","container":true},"db":{"notation":"D"}}},
 "model":{"s":{"kind":"system","title":"S","status":"deployed","children":{"e":{"kind":"db","title":"E","status":"bogus"},"f":{"kind":"db","title":"F","status":"odd"}}}},
 "views":{"v":{"title":"V","include":["*"]}}}`

// An element with a status that is not a lifecycle status is listed too, in a
// section of its own, so that the sections add up (#710).
func TestStatusCmd_ListsElementsWithAnUnknownStatus(t *testing.T) {
	path := createTempFile(t, statusUnknownModel)
	defer removeTempFile(t, path)

	out, err := executeRootCmd("status", "--model", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"unknown status (2)", "s.e", `status "bogus"`, "s.f", `status "odd"`} {
		if !strings.Contains(out, want) {
			t.Errorf("want %q in:\n%s", want, out)
		}
	}
	if strings.Index(out, "unknown status") < strings.Index(out, "unset (") {
		t.Errorf("the unknown section comes after the known ones and unset:\n%s", out)
	}
}

// Without an unknown status there is no such section.
func TestStatusCmd_NoUnknownSectionWithoutUnknownStatus(t *testing.T) {
	out, err := executeRootCmd("status", "--model", writeStatusTestModel(t))
	if err != nil || strings.Contains(out, "unknown status") {
		t.Errorf("no unknown section expected, got %q (%v)", out, err)
	}
}

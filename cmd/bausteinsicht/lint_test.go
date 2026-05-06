package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func executeLintCmd(args ...string) (string, error) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

const modelWithConstraints = `{
	"specification": {
		"elements": {
			"system":    {"notation": "System"},
			"container": {"notation": "Container"},
			"person":    {"notation": "Person"}
		}
	},
	"model": {
		"frontend": {"kind": "system",    "title": "Frontend", "description": "desc"},
		"backend":  {"kind": "system",    "title": "Backend",  "description": "desc"},
		"db":       {"kind": "container", "title": "Database", "technology": "PostgreSQL"}
	},
	"relationships": [
		{"from": "frontend", "to": "backend"}
	],
	"views": {"main": {"title": "Main"}},
	"constraints": [
		{
			"id":          "C01",
			"description": "Systems must have descriptions",
			"rule":        "required-field",
			"element-kind": "system",
			"field":       "description"
		}
	]
}`

const modelNoConstraints = `{
	"specification": {
		"elements": {"system": {"notation": "System"}}
	},
	"model": {"a": {"kind": "system", "title": "A"}},
	"relationships": [],
	"views": {"main": {"title": "Main"}}
}`

const modelWithViolation = `{
	"specification": {
		"elements": {
			"system":    {"notation": "System"},
			"container": {"notation": "Container"}
		}
	},
	"model": {
		"a": {"kind": "system",    "title": "A"},
		"b": {"kind": "container", "title": "B"}
	},
	"relationships": [{"from": "a", "to": "b"}],
	"views": {"main": {"title": "Main"}},
	"constraints": [
		{
			"id":          "C01",
			"description": "Systems must not use containers",
			"rule":        "required-field",
			"element-kind": "system",
			"field":       "description"
		}
	]
}`

func writeModel(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "model.jsonc")
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLint_NoViolations(t *testing.T) {
	p := writeModel(t, modelWithConstraints)
	out, err := executeLintCmd("lint", "--model", p)
	if err != nil {
		t.Fatalf("expected no error, got %v\nOutput: %s", err, out)
	}
	if !strings.Contains(out, "All constraints passed.") {
		t.Errorf("expected 'All constraints passed.', got %q", out)
	}
}

func TestLint_NoConstraintsDefined(t *testing.T) {
	p := writeModel(t, modelNoConstraints)
	out, err := executeLintCmd("lint", "--model", p)
	if err != nil {
		t.Fatalf("expected no error, got %v\nOutput: %s", err, out)
	}
	if !strings.Contains(out, "No constraints defined.") {
		t.Errorf("expected 'No constraints defined.', got %q", out)
	}
}

func TestLint_Violation_ExitCode1(t *testing.T) {
	p := writeModel(t, modelWithViolation)
	out, err := executeLintCmd("lint", "--model", p)
	if err == nil {
		t.Fatal("expected error (exit code 1), got nil")
	}
	ee, ok := err.(*exitError)
	if !ok {
		t.Fatalf("expected exitError, got %T: %v", err, err)
	}
	if ee.code != 1 {
		t.Errorf("expected exit code 1, got %d", ee.code)
	}
	if !strings.Contains(out, "VIOLATION") {
		t.Errorf("expected VIOLATION in output, got %q", out)
	}
}

func TestLint_NonExistentFile_ExitCode2(t *testing.T) {
	_, err := executeLintCmd("lint", "--model", "/nonexistent/model.jsonc")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
	ee, ok := err.(*exitError)
	if !ok {
		t.Fatalf("expected exitError, got %T: %v", err, err)
	}
	if ee.code != 2 {
		t.Errorf("expected exit code 2, got %d", ee.code)
	}
}

func TestLint_JSONOutput_Passed(t *testing.T) {
	p := writeModel(t, modelWithConstraints)
	out, err := executeLintCmd("lint", "--model", p, "--format", "json")
	if err != nil {
		t.Fatalf("expected no error, got %v\nOutput: %s", err, out)
	}

	var result struct {
		Passed     bool        `json:"passed"`
		Total      int         `json:"total"`
		Violations interface{} `json:"violations"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\nOutput: %s", err, out)
	}
	if !result.Passed {
		t.Error("expected passed=true")
	}
	if result.Total != 0 {
		t.Errorf("expected total=0, got %d", result.Total)
	}
}

func TestLint_JSONOutput_Violation(t *testing.T) {
	p := writeModel(t, modelWithViolation)
	out, err := executeLintCmd("lint", "--model", p, "--format", "json")
	if err == nil {
		t.Fatal("expected error")
	}

	var result struct {
		Passed     bool `json:"passed"`
		Total      int  `json:"total"`
		Violations []struct {
			ConstraintID string   `json:"constraint_id"`
			Message      string   `json:"message"`
			Elements     []string `json:"elements"`
		} `json:"violations"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\nOutput: %s", err, out)
	}
	if result.Passed {
		t.Error("expected passed=false")
	}
	if result.Total == 0 {
		t.Error("expected total > 0")
	}
	if len(result.Violations) == 0 {
		t.Error("expected violations array to be non-empty")
	}
}

func TestLint_ViolationLists_Elements(t *testing.T) {
	p := writeModel(t, modelWithViolation)
	out, err := executeLintCmd("lint", "--model", p)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(out, "  - ") {
		t.Errorf("expected indented element list in output, got %q", out)
	}
}

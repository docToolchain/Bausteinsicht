package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bauteinsicht/templates"
)

func executeValidateCmd(args ...string) (string, error) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

func TestValidate_ValidModel(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.jsonc")
	if err := os.WriteFile(modelPath, templates.SampleModel, 0644); err != nil {
		t.Fatal(err)
	}

	out, err := executeValidateCmd("validate", "--model", modelPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(out, "Model is valid.") {
		t.Errorf("expected 'Model is valid.' in output, got %q", out)
	}
}

func TestValidate_InvalidModel_MissingTitle(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.jsonc")
	invalidModel := `{
		"specification": {
			"elements": {
				"system": {"notation": "System"}
			}
		},
		"model": {
			"mySystem": {
				"kind": "system",
				"title": ""
			}
		},
		"relationships": [],
		"views": {
			"main": {"title": "Main View"}
		}
	}`
	if err := os.WriteFile(modelPath, []byte(invalidModel), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := executeValidateCmd("validate", "--model", modelPath)
	if err == nil {
		t.Fatal("expected an error for invalid model")
	}
	ee, ok := err.(*exitError)
	if !ok {
		t.Fatalf("expected exitError, got %T: %v", err, err)
	}
	if ee.code != 1 {
		t.Errorf("expected exit code 1, got %d", ee.code)
	}
	if !strings.Contains(out, "ERROR:") {
		t.Errorf("expected ERROR in output, got %q", out)
	}
}

func TestValidate_NonExistentFile(t *testing.T) {
	_, err := executeValidateCmd("validate", "--model", "/nonexistent/model.jsonc")
	if err == nil {
		t.Fatal("expected an error for non-existent file")
	}
	ee, ok := err.(*exitError)
	if !ok {
		t.Fatalf("expected exitError, got %T: %v", err, err)
	}
	if ee.code != 2 {
		t.Errorf("expected exit code 2, got %d", ee.code)
	}
}

func TestValidate_JSONOutput_Valid(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.jsonc")
	if err := os.WriteFile(modelPath, templates.SampleModel, 0644); err != nil {
		t.Fatal(err)
	}

	out, err := executeValidateCmd("validate", "--model", modelPath, "--format", "json")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var result validateResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, out)
	}
	if !result.Valid {
		t.Errorf("expected valid=true, got false")
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(result.Errors))
	}
}

func TestValidate_JSONOutput_Invalid(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.jsonc")
	invalidModel := `{
		"specification": {
			"elements": {
				"system": {"notation": "System"}
			}
		},
		"model": {
			"mySystem": {
				"kind": "system",
				"title": ""
			}
		},
		"relationships": [],
		"views": {
			"main": {"title": "Main View"}
		}
	}`
	if err := os.WriteFile(modelPath, []byte(invalidModel), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := executeValidateCmd("validate", "--model", modelPath, "--format", "json")
	if err == nil {
		t.Fatal("expected an error for invalid model")
	}

	var result validateResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, out)
	}
	if result.Valid {
		t.Errorf("expected valid=false, got true")
	}
	if len(result.Errors) == 0 {
		t.Errorf("expected errors, got none")
	}
}

func TestValidate_AutoDetect(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "architecture.jsonc")
	if err := os.WriteFile(modelPath, templates.SampleModel, 0644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	out, cmdErr := executeValidateCmd("validate")
	if cmdErr != nil {
		t.Fatalf("expected no error, got %v", cmdErr)
	}
	if !strings.Contains(out, "Model is valid.") {
		t.Errorf("expected 'Model is valid.' in output, got %q", out)
	}
}

func TestValidate_AutoDetect_NoFile(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	_, cmdErr := executeValidateCmd("validate")
	if cmdErr == nil {
		t.Fatal("expected an error when no .jsonc file exists")
	}
	ee, ok := cmdErr.(*exitError)
	if !ok {
		t.Fatalf("expected exitError, got %T: %v", cmdErr, cmdErr)
	}
	if ee.code != 2 {
		t.Errorf("expected exit code 2, got %d", ee.code)
	}
}

func TestValidate_EmptyJSON_WarnsButPasses(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.jsonc")
	if err := os.WriteFile(modelPath, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := executeValidateCmd("validate", "--model", modelPath)
	if err != nil {
		t.Fatalf("expected no error (exit 0) for empty model, got %v", err)
	}
	if !strings.Contains(out, "WARNING:") {
		t.Errorf("expected WARNING in output for empty model, got %q", out)
	}
	if !strings.Contains(out, "Model is valid.") {
		t.Errorf("expected 'Model is valid.' in output, got %q", out)
	}
}

func TestValidate_SpecOnly_WarnsNoElements(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.jsonc")
	specOnly := `{
		"specification": {
			"elements": {
				"system": {"notation": "System"}
			}
		}
	}`
	if err := os.WriteFile(modelPath, []byte(specOnly), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := executeValidateCmd("validate", "--model", modelPath)
	if err != nil {
		t.Fatalf("expected no error (exit 0) for spec-only model, got %v", err)
	}
	if !strings.Contains(out, "WARNING:") {
		t.Errorf("expected WARNING in output for spec-only model, got %q", out)
	}
	if !strings.Contains(out, "no elements defined") {
		t.Errorf("expected warning about no elements, got %q", out)
	}
	if !strings.Contains(out, "Model is valid.") {
		t.Errorf("expected 'Model is valid.' in output, got %q", out)
	}
}

func TestValidate_EmptyJSON_JSONFormat_IncludesWarnings(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.jsonc")
	if err := os.WriteFile(modelPath, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := executeValidateCmd("validate", "--model", modelPath, "--format", "json")
	if err != nil {
		t.Fatalf("expected no error for empty model in JSON format, got %v", err)
	}

	var result validateResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, out)
	}
	if !result.Valid {
		t.Errorf("expected valid=true for empty model, got false")
	}
	if len(result.Warnings) == 0 {
		t.Errorf("expected warnings in JSON output for empty model, got none")
	}
}

// executeValidateCmdSplit runs the validate command and captures stdout and
// stderr into separate buffers so verbose output (written to stderr) can be
// verified independently.
func executeValidateCmdSplit(args ...string) (stdout, stderr string, err error) {
	cmd := NewRootCmd()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)
	cmd.SetErr(errBuf)
	cmd.SetArgs(args)
	err = cmd.Execute()
	return outBuf.String(), errBuf.String(), err
}

func TestValidate_Verbose_PrintsModelPathAndElementCount(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.jsonc")
	if err := os.WriteFile(modelPath, templates.SampleModel, 0644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := executeValidateCmdSplit("validate", "--model", modelPath, "--verbose")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(stderr, "Validating model:") {
		t.Errorf("verbose output should contain 'Validating model:', got stderr=%q", stderr)
	}
	if !strings.Contains(stderr, modelPath) {
		t.Errorf("verbose output should contain model path %q, got stderr=%q", modelPath, stderr)
	}
	if !strings.Contains(stderr, "elements") {
		t.Errorf("verbose output should mention element count, got stderr=%q", stderr)
	}
	if !strings.Contains(stderr, "relationships") {
		t.Errorf("verbose output should mention relationship count, got stderr=%q", stderr)
	}
	if !strings.Contains(stderr, "views") {
		t.Errorf("verbose output should mention view count, got stderr=%q", stderr)
	}
}

func TestValidate_Verbose_NotShownWithoutFlag(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.jsonc")
	if err := os.WriteFile(modelPath, templates.SampleModel, 0644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := executeValidateCmdSplit("validate", "--model", modelPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if strings.Contains(stderr, "Validating model:") {
		t.Errorf("verbose output should NOT appear without --verbose flag, got stderr=%q", stderr)
	}
}

func TestValidate_Verbose_SuppressedWithJSONFormat(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.jsonc")
	if err := os.WriteFile(modelPath, templates.SampleModel, 0644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := executeValidateCmdSplit("validate", "--model", modelPath, "--verbose", "--format", "json")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if strings.Contains(stderr, "Validating model:") {
		t.Errorf("verbose output should be suppressed with --format json, got stderr=%q", stderr)
	}

	// JSON output should still be valid.
	var result validateResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, stdout)
	}
}

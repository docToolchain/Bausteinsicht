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
	t.Cleanup(func() { os.Chdir(origDir) })

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
	t.Cleanup(func() { os.Chdir(origDir) })

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

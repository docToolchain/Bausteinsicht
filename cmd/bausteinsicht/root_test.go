package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/templates"
)

func executeRootCmd(args ...string) (string, error) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

func TestRootCmd_RejectsInvalidFormat(t *testing.T) {
	for _, format := range []string{"xml", "yaml", "csv", "HTML"} {
		t.Run(format, func(t *testing.T) {
			_, err := executeRootCmd("validate", "--format", format)
			if err == nil {
				t.Fatalf("expected error for invalid format %q, but got none", format)
			}
			if !strings.Contains(err.Error(), "unknown format") {
				t.Fatalf("expected 'unknown format' in error, got: %v", err)
			}
		})
	}
}

func TestRootCmd_AcceptsJsonCaseInsensitive(t *testing.T) {
	// "JSON" (uppercase) should be normalized to "json" and work correctly.
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.jsonc")
	if err := os.WriteFile(modelPath, templates.SampleModel, 0644); err != nil {
		t.Fatal(err)
	}

	out, err := executeRootCmd("validate", "--model", modelPath, "--format", "JSON")
	if err != nil {
		t.Fatalf("expected no error for format 'JSON', got: %v", err)
	}
	// The command should succeed — this also proves normalization works
	// since the validate command checks format == "json".
	if !strings.Contains(out, "valid") {
		t.Errorf("expected validation output, got: %q", out)
	}
}

func TestRootCmd_AcceptsTextFormat(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.jsonc")
	if err := os.WriteFile(modelPath, templates.SampleModel, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := executeRootCmd("validate", "--model", modelPath, "--format", "text")
	if err != nil {
		t.Fatalf("expected no error for format 'text', got: %v", err)
	}
}

func TestRootCmd_AcceptsJsonLowercase(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.jsonc")
	if err := os.WriteFile(modelPath, templates.SampleModel, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := executeRootCmd("validate", "--model", modelPath, "--format", "json")
	if err != nil {
		t.Fatalf("expected no error for format 'json', got: %v", err)
	}
}

func TestRootCmd_AcceptsTextCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.jsonc")
	if err := os.WriteFile(modelPath, templates.SampleModel, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := executeRootCmd("validate", "--model", modelPath, "--format", "TEXT")
	if err != nil {
		t.Fatalf("expected no error for format 'TEXT', got: %v", err)
	}
}

func TestRootCmd_RejectsNonDrawioTemplate(t *testing.T) {
	for _, name := range []string{"model.jsonc", "styles.xml", "template.png", "notes.txt"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			// Create the file so it exists on disk.
			fpath := filepath.Join(dir, name)
			if err := os.WriteFile(fpath, []byte("dummy"), 0644); err != nil {
				t.Fatal(err)
			}
			// Also create a valid model so the command doesn't fail for
			// unrelated auto-detect reasons.
			modelPath := filepath.Join(dir, "arch.jsonc")
			if err := os.WriteFile(modelPath, templates.SampleModel, 0644); err != nil {
				t.Fatal(err)
			}

			_, err := executeRootCmd("validate", "--model", modelPath, "--template", fpath)
			if err == nil {
				t.Fatalf("expected error for non-.drawio template %q, but got none", name)
			}
			if !strings.Contains(err.Error(), ".drawio") {
				t.Fatalf("expected error to mention .drawio, got: %v", err)
			}
		})
	}
}

func TestRootCmd_AcceptsDrawioTemplate(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.jsonc")
	if err := os.WriteFile(modelPath, templates.SampleModel, 0644); err != nil {
		t.Fatal(err)
	}
	// Create a minimal .drawio template file.
	tmplPath := filepath.Join(dir, "custom.drawio")
	if err := os.WriteFile(tmplPath, []byte("<mxfile/>"), 0644); err != nil {
		t.Fatal(err)
	}

	// The command may fail for other reasons (e.g. missing drawio file for sync),
	// but it should NOT fail due to template extension validation.
	_, err := executeRootCmd("validate", "--model", modelPath, "--template", tmplPath)
	if err != nil && strings.Contains(err.Error(), ".drawio") && strings.Contains(err.Error(), "must have") {
		t.Fatalf("should accept .drawio template, got: %v", err)
	}
}

func TestRootCmd_AcceptsEmptyTemplate(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.jsonc")
	if err := os.WriteFile(modelPath, templates.SampleModel, 0644); err != nil {
		t.Fatal(err)
	}

	// No --template flag should work fine (uses default).
	_, err := executeRootCmd("validate", "--model", modelPath)
	if err != nil {
		t.Fatalf("expected no error without --template flag, got: %v", err)
	}
}

func TestRootCmd_RejectsModelPathTraversal(t *testing.T) {
	_, err := executeRootCmd("validate", "--model", "../../etc/model.jsonc")
	if err == nil {
		t.Fatal("expected error for model path with traversal")
	}
	if !strings.Contains(err.Error(), "traversal") {
		t.Fatalf("expected 'traversal' in error, got: %v", err)
	}
}

func TestRootCmd_RejectsTemplatePathTraversal(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.jsonc")
	if err := os.WriteFile(modelPath, templates.SampleModel, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := executeRootCmd("validate", "--model", modelPath, "--template", "../../../tmp/evil.drawio")
	if err == nil {
		t.Fatal("expected error for template path with traversal")
	}
	if !strings.Contains(err.Error(), "traversal") {
		t.Fatalf("expected 'traversal' in error, got: %v", err)
	}
}

func TestValidatePathContainment(t *testing.T) {
	if err := validatePathContainment("subdir/file.jsonc"); err != nil {
		t.Errorf("relative sub-path should be allowed: %v", err)
	}
	if err := validatePathContainment("./file.jsonc"); err != nil {
		t.Errorf("current-dir path should be allowed: %v", err)
	}
	if err := validatePathContainment("/tmp/absolute/file.jsonc"); err != nil {
		t.Errorf("absolute path should be allowed: %v", err)
	}
	if err := validatePathContainment("../../etc/passwd"); err == nil {
		t.Error("path traversal should be rejected")
	}
	if err := validatePathContainment("foo/../../bar"); err == nil {
		t.Error("hidden path traversal should be rejected")
	}
}

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCreatesFiles(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"init"})
	buf := &strings.Builder{}
	cmd.SetOut(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Check all 4 files exist.
	for _, name := range []string{defaultModelFile, defaultTemplFile, defaultDrawioFile, defaultSyncState} {
		if _, err := os.Stat(filepath.Join(dir, name)); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", name)
		}
	}
}

func TestInitFailsIfFilesExist(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Create model file to trigger conflict.
	if err := os.WriteFile(defaultModelFile, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"init"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when files already exist")
	}
	if e, ok := err.(*exitError); ok {
		if e.code != 2 {
			t.Errorf("expected exit code 2, got %d", e.code)
		}
	}
}

func TestInitDrawioFilesExist(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Create drawio file to trigger conflict.
	if err := os.WriteFile(defaultDrawioFile, []byte("<mxfile/>"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"init"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when drawio file already exists")
	}
}

func TestInitTemplateFileExists(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Create template file to trigger conflict.
	if err := os.WriteFile(defaultTemplFile, []byte("custom"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"init"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when template file already exists")
	}

	// Verify custom template was NOT overwritten.
	data, _ := os.ReadFile(defaultTemplFile)
	if string(data) != "custom" {
		t.Error("template.drawio was overwritten despite already existing")
	}
}

func TestInitGeneratesValidModel(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"init"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// The model file should be loadable.
	_, err := os.ReadFile(defaultModelFile)
	if err != nil {
		t.Fatalf("reading model: %v", err)
	}
}

func TestInitGeneratesValidDrawio(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"init"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// The drawio file should contain mxfile and diagram elements.
	data, err := os.ReadFile(defaultDrawioFile)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "<mxfile") {
		t.Error("drawio file missing <mxfile> element")
	}
	if !strings.Contains(content, "<diagram") {
		t.Error("drawio file missing <diagram> element")
	}
}

func TestInitJSONOutput(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Capture stdout.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"init", "--format", "json"})
	err := cmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\noutput: %s", err, output)
	}
	if result["success"] != true {
		t.Error("expected success=true")
	}
	files, ok := result["files"].([]interface{})
	if !ok || len(files) != 4 {
		t.Errorf("expected 4 files, got %v", result["files"])
	}
}

func TestInitSyncStateValid(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"init"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Sync state should be valid JSON with expected fields.
	data, err := os.ReadFile(defaultSyncState)
	if err != nil {
		t.Fatal(err)
	}
	var state map[string]interface{}
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("invalid sync state JSON: %v", err)
	}
	if _, ok := state["timestamp"]; !ok {
		t.Error("sync state missing timestamp")
	}
	if _, ok := state["elements"]; !ok {
		t.Error("sync state missing elements")
	}
}

func TestInitGenerateTemplate(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"init", "--generate-template"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init with --generate-template failed: %v", err)
	}

	// Template should be generated from spec and contain mxGraphModel
	data, err := os.ReadFile(defaultTemplFile)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "<mxGraphModel") {
		t.Error("generated template missing mxGraphModel element")
	}
	if !strings.Contains(content, "fillColor=") {
		t.Error("generated template missing color styling")
	}
}

func TestInitGenerateTemplateContainsAllKinds(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"init", "--generate-template"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init with --generate-template failed: %v", err)
	}

	// Load the generated template and verify it contains kind elements
	data, err := os.ReadFile(defaultTemplFile)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	// The default model should have certain kinds
	expectedKinds := []string{"actor", "system", "container"}
	for _, kind := range expectedKinds {
		if !strings.Contains(content, "["+kind+"]") {
			t.Errorf("generated template missing kind %q", kind)
		}
	}
}

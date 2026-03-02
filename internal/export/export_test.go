package export

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDetectDrawioBinary_FindsDrawioExport(t *testing.T) {
	// Create a fake drawio-export in a temp dir.
	dir := t.TempDir()
	fakeBin := filepath.Join(dir, "drawio-export")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", dir+":"+origPath)

	bin, err := DetectDrawioBinary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bin != fakeBin {
		t.Errorf("expected %q, got %q", fakeBin, bin)
	}
}

func TestDetectDrawioBinary_FallsBackToDrawio(t *testing.T) {
	dir := t.TempDir()
	fakeBin := filepath.Join(dir, "drawio")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	// Set PATH to only the temp dir so drawio-export is NOT found.
	t.Setenv("PATH", dir)

	bin, err := DetectDrawioBinary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bin != fakeBin {
		t.Errorf("expected %q, got %q", fakeBin, bin)
	}
}

func TestDetectDrawioBinary_ErrorWhenNotFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := DetectDrawioBinary()
	if err == nil {
		t.Error("expected error when no draw.io binary found")
	}
}

func TestBuildExportArgs(t *testing.T) {
	args := BuildExportArgs(ExportOptions{
		Format:       "png",
		PageIndex:    2,
		OutputPath:   "/tmp/out.png",
		EmbedDiagram: true,
		InputFile:    "arch.drawio",
	})

	expected := []string{
		"--export",
		"--format", "png",
		"--page-index", "2",
		"--output", "/tmp/out.png",
		"--embed-diagram",
		"arch.drawio",
	}
	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(args), args)
	}
	for i, want := range expected {
		if args[i] != want {
			t.Errorf("arg[%d] = %q, want %q", i, args[i], want)
		}
	}
}

func TestBuildExportArgs_WithoutEmbed(t *testing.T) {
	args := BuildExportArgs(ExportOptions{
		Format:       "svg",
		PageIndex:    1,
		OutputPath:   "/tmp/out.svg",
		EmbedDiagram: false,
		InputFile:    "arch.drawio",
	})

	for _, arg := range args {
		if arg == "--embed-diagram" {
			t.Error("--embed-diagram should not be present when EmbedDiagram is false")
		}
	}
}

func TestOutputFileName(t *testing.T) {
	tests := []struct {
		viewKey string
		format  string
		want    string
	}{
		{"context", "png", "architecture-context.png"},
		{"containers", "svg", "architecture-containers.svg"},
		{"my-view", "png", "architecture-my-view.png"},
	}
	for _, tt := range tests {
		got := OutputFileName(tt.viewKey, tt.format)
		if got != tt.want {
			t.Errorf("OutputFileName(%q, %q) = %q, want %q", tt.viewKey, tt.format, got, tt.want)
		}
	}
}

func TestExportPage_Integration(t *testing.T) {
	// Skip if draw.io CLI is not available.
	if _, err := exec.LookPath("drawio-export"); err != nil {
		if _, err := exec.LookPath("drawio"); err != nil {
			t.Skip("draw.io CLI not available, skipping integration test")
		}
	}

	// We need a real drawio file to test with.
	// Use the project's test data or init a fresh one.
	drawioFile := filepath.Join("..", "..", "internal", "drawio", "testdata", "simple-diagram.drawio")
	if _, err := os.Stat(drawioFile); err != nil {
		t.Skipf("test drawio file not found: %v", err)
	}

	dir := t.TempDir()
	outFile := filepath.Join(dir, "test-export.png")

	bin, err := DetectDrawioBinary()
	if err != nil {
		t.Fatalf("DetectDrawioBinary: %v", err)
	}

	err = ExportPage(bin, ExportOptions{
		Format:     "png",
		PageIndex:  1,
		OutputPath: outFile,
		InputFile:  drawioFile,
	})
	if err != nil {
		t.Fatalf("ExportPage: %v", err)
	}

	info, err := os.Stat(outFile)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Error("output file is empty")
	}
}

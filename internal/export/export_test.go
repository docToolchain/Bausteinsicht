package export

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// fakeBinary creates a minimal executable in dir with the given base name,
// adding .exe on Windows so exec.LookPath can find it.
func fakeBinary(t *testing.T, dir, name string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDetectDrawioBinary_FindsDrawioExport(t *testing.T) {
	dir := t.TempDir()
	fakeBin := fakeBinary(t, dir, "drawio-export")
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+origPath)

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
	fakeBin := fakeBinary(t, dir, "drawio")
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
	// Override platform paths so filesystem installs don't interfere.
	old := platformPaths
	platformPaths = func() []string { return nil }
	t.Cleanup(func() { platformPaths = old })
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
		"--", "arch.drawio",
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

// TestBuildExportArgs_InputFileIsLastArg is a regression test for the bug where
// unrecognized Electron flags (e.g. --disable-gpu) passed before the input file
// would land as program.args[0] in draw.io's CLI parser, causing
// "Error: input file/directory not found" with exit code 0.
// The input file must always be the last argument so it is unambiguously paths[0].
func TestBuildExportArgs_InputFileIsLastArg(t *testing.T) {
	for _, scale := range []float64{0, 1.0, 2.0} {
		args := BuildExportArgs(ExportOptions{
			Format:     "png",
			PageIndex:  1,
			OutputPath: "/tmp/out.png",
			InputFile:  "arch.drawio",
			Scale:      scale,
		})
		if args[len(args)-1] != "arch.drawio" {
			t.Errorf("scale=%v: input file must be the last argument, got %q (full args: %v)", scale, args[len(args)-1], args)
		}
		if args[len(args)-2] != "--" {
			t.Errorf("scale=%v: '--' separator must precede input file, got %q (full args: %v)", scale, args[len(args)-2], args)
		}
	}
}

// TestBuildExportArgs_ScaleOneNotIncluded verifies that Scale=1.0 (the headless-safe
// default) does not add a --scale flag, avoiding the GPU process crash that occurs
// when draw.io tries to render at scale > 1 without hardware GPU acceleration.
func TestBuildExportArgs_ScaleOneNotIncluded(t *testing.T) {
	args := BuildExportArgs(ExportOptions{
		Format:     "png",
		PageIndex:  1,
		OutputPath: "/tmp/out.png",
		InputFile:  "arch.drawio",
		Scale:      1.0,
	})
	for i, arg := range args {
		if arg == "--scale" {
			t.Errorf("--scale should not be present for Scale=1.0 (GPU not required), but found at index %d", i)
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

func TestOutputFileName_StripsPathTraversal(t *testing.T) {
	tests := []struct {
		viewKey string
		format  string
		want    string
	}{
		{"../../../tmp/pwned", "png", "architecture-pwned.png"},
		{"/etc/passwd", "svg", "architecture-passwd.svg"},
		{"foo/../../bar", "png", "architecture-bar.png"},
		{"normal-key", "png", "architecture-normal-key.png"},
	}
	for _, tt := range tests {
		got := OutputFileName(tt.viewKey, tt.format)
		if got != tt.want {
			t.Errorf("OutputFileName(%q, %q) = %q, want %q", tt.viewKey, tt.format, got, tt.want)
		}
	}
}

func TestSafeViewKey(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"context", "context"},
		{"my-view", "my-view"},
		{"../../../tmp/pwned", "pwned"},
		{"/etc/passwd", "passwd"},
		{"foo/bar", "bar"},
		{"foo\\bar", "bar"},
	}
	for _, tt := range tests {
		got := SafeViewKey(tt.key)
		if got != tt.want {
			t.Errorf("SafeViewKey(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

// TestExportPage_ErrorWhenOutputMissing verifies that ExportPage returns an
// error when the draw.io CLI exits successfully but the output file does not
// exist (e.g., permission denied on output directory). (#195)
func TestExportPage_ErrorWhenOutputMissing(t *testing.T) {
	// Create a fake "draw.io" binary that exits 0 but writes nothing.
	dir := t.TempDir()
	fakeBin := filepath.Join(dir, "drawio-fake")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}

	outFile := filepath.Join(dir, "should-not-exist.png")

	err := ExportPage(fakeBin, ExportOptions{
		Format:     "png",
		PageIndex:  1,
		OutputPath: outFile,
		InputFile:  "/dev/null",
	})
	if err == nil {
		t.Error("expected error when output file not created, got nil")
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

// Regression Tests for #388: Windows/macOS Package Manager Path Detection

// TestDetectDrawioBinary_WindowsScoopPath verifies Scoop package manager detection on Windows.
// Regression test for #388: Users with Scoop-installed draw.io should be detected.
func TestDetectDrawioBinary_WindowsScoopPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test, skipping on non-Windows")
	}

	dir := t.TempDir()
	// Simulate Scoop installation directory
	scoopApp := filepath.Join(dir, "apps", "drawio", "current")
	if err := os.MkdirAll(scoopApp, 0755); err != nil {
		t.Fatal(err)
	}
	fakeBin := fakeBinary(t, scoopApp, "draw.io")

	// Set SCOOP env var to temp dir to override user's actual Scoop installation
	t.Setenv("SCOOP", dir)
	t.Setenv("PATH", "")

	// Override platform paths to only return Scoop paths
	old := platformPaths
	platformPaths = func() []string {
		return []string{
			filepath.Join(dir, "apps", "drawio", "current", "draw.io.exe"),
			filepath.Join(dir, "shims", "draw.io.exe"),
		}
	}
	t.Cleanup(func() { platformPaths = old })

	bin, err := DetectDrawioBinary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bin != fakeBin {
		t.Errorf("expected %q, got %q", fakeBin, bin)
	}
}

// TestDetectDrawioBinary_WindowsChocolateyPath verifies Chocolatey package manager detection.
// Regression test for #388: Users with Chocolatey-installed draw.io should be detected.
func TestDetectDrawioBinary_WindowsChocolateyPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test, skipping on non-Windows")
	}

	dir := t.TempDir()
	chocoDir := filepath.Join(dir, "chocolatey", "bin")
	if err := os.MkdirAll(chocoDir, 0755); err != nil {
		t.Fatal(err)
	}
	fakeBin := fakeBinary(t, chocoDir, "draw.io")

	// Clear environment and override platform paths
	t.Setenv("PATH", "")
	t.Setenv("SCOOP", "")

	old := platformPaths
	platformPaths = func() []string {
		return []string{
			filepath.Join(dir, "chocolatey", "bin", "draw.io.exe"),
		}
	}
	t.Cleanup(func() { platformPaths = old })

	bin, err := DetectDrawioBinary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bin != fakeBin {
		t.Errorf("expected %q, got %q", fakeBin, bin)
	}
}

// TestDetectDrawioBinary_macOSHomebrewAppleSilicon verifies Homebrew detection on Apple Silicon Macs.
// Regression test for #388: macOS users with Homebrew (M1/M2/M3) should be detected.
func TestDetectDrawioBinary_macOSHomebrewAppleSilicon(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-specific test, skipping on non-macOS")
	}

	dir := t.TempDir()
	brewDir := filepath.Join(dir, "opt", "homebrew", "bin")
	if err := os.MkdirAll(brewDir, 0755); err != nil {
		t.Fatal(err)
	}
	fakeBin := fakeBinary(t, brewDir, "draw.io")

	// Clear PATH and override platform paths
	t.Setenv("PATH", "")

	old := platformPaths
	platformPaths = func() []string {
		return []string{
			filepath.Join(dir, "opt", "homebrew", "bin", "draw.io"),
		}
	}
	t.Cleanup(func() { platformPaths = old })

	bin, err := DetectDrawioBinary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bin != fakeBin {
		t.Errorf("expected %q, got %q", fakeBin, bin)
	}
}

// TestDetectDrawioBinary_macOSHomebrewIntel verifies Homebrew detection on Intel Macs.
// Regression test for #388: Intel Mac users with Homebrew should be detected.
func TestDetectDrawioBinary_macOSHomebrewIntel(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-specific test, skipping on non-macOS")
	}

	dir := t.TempDir()
	brewDir := filepath.Join(dir, "usr", "local", "bin")
	if err := os.MkdirAll(brewDir, 0755); err != nil {
		t.Fatal(err)
	}
	fakeBin := fakeBinary(t, brewDir, "draw.io")

	// Clear PATH and override platform paths
	t.Setenv("PATH", "")

	old := platformPaths
	platformPaths = func() []string {
		return []string{
			filepath.Join(dir, "usr", "local", "bin", "draw.io"),
		}
	}
	t.Cleanup(func() { platformPaths = old })

	bin, err := DetectDrawioBinary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bin != fakeBin {
		t.Errorf("expected %q, got %q", fakeBin, bin)
	}
}

// TestDetectDrawioBinary_SearchOrderPriority verifies that PATH takes priority over platform paths.
// Regression test for #388: ensure search order is correct (PATH first, then package managers).
func TestDetectDrawioBinary_SearchOrderPriority(t *testing.T) {
	pathDir := t.TempDir()
	platformDir := t.TempDir()

	// Create draw.io in both PATH and platform-specific locations
	pathBin := fakeBinary(t, pathDir, "drawio")
	platformBin := fakeBinary(t, platformDir, "draw.io")

	t.Setenv("PATH", pathDir)

	old := platformPaths
	platformPaths = func() []string {
		return []string{platformBin}
	}
	t.Cleanup(func() { platformPaths = old })

	bin, err := DetectDrawioBinary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should find drawio in PATH first, not the platform-specific one
	if bin != pathBin {
		t.Errorf("expected PATH binary %q, got %q (should prioritize PATH)", pathBin, bin)
	}
}

// Package export handles exporting draw.io diagrams to PNG/SVG using the
// draw.io CLI.
package export

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// ExportOptions configures a single page export operation.
type ExportOptions struct {
	Format       string  // "png" or "svg"
	PageIndex    int     // 1-based page index
	OutputPath   string  // full path to output file
	EmbedDiagram bool    // embed draw.io XML source in output
	InputFile    string  // path to the .drawio file
	Scale        float64 // export scale factor (0 = default, e.g. 2.0 for retina)
}

// platformPaths is a function variable so tests can override it.
var platformPaths = platformDrawioPaths

// DetectDrawioBinary finds the draw.io CLI binary.
// Search order:
//  1. "drawio-export" — devcontainer wrapper (Linux, adds xvfb + --no-sandbox)
//  2. "drawio" — on PATH (Linux package install)
//  3. Platform-native install paths (Windows, macOS) via platformPaths()
func DetectDrawioBinary() (string, error) {
	for _, name := range []string{"drawio-export", "drawio"} {
		path, err := exec.LookPath(name)
		if err == nil {
			return path, nil
		}
	}
	for _, candidate := range platformPaths() {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("draw.io CLI not found; install from https://www.drawio.com/")
}

// BuildExportArgs constructs the command-line arguments for a draw.io export.
func BuildExportArgs(opts ExportOptions) []string {
	args := []string{
		"--export",
		"--format", opts.Format,
		"--page-index", strconv.Itoa(opts.PageIndex),
		"--output", opts.OutputPath,
	}
	if opts.EmbedDiagram {
		args = append(args, "--embed-diagram")
	}
	// Only pass --scale for values > 1. Scale=1 is draw.io's native resolution
	// and does not need an explicit flag. Scale > 1 (e.g. 2.0 for retina) uses
	// the GPU rendering pipeline and requires hardware GPU acceleration.
	// Passing --scale 2 in headless containers (where the GPU process is
	// disabled via ELECTRON_DISABLE_GPU) causes the GPU process to crash with
	// exit code 9, resulting in a silent export failure (exit 0, no output file).
	if opts.Scale > 1 {
		args = append(args, "--scale", fmt.Sprintf("%g", opts.Scale))
	}
	args = append(args, "--", opts.InputFile)
	return args
}

// SafeViewKey strips directory components from a view key to prevent
// path traversal when used in filenames (SEC-015).
func SafeViewKey(key string) string {
	key = filepath.Base(strings.ReplaceAll(key, "\\", "/"))
	return key
}

// OutputFileName returns the canonical output file name for a view export.
func OutputFileName(viewKey, format string) string {
	return fmt.Sprintf("architecture-%s.%s", SafeViewKey(viewKey), format)
}

// ExportPage runs the draw.io CLI to export a single page.
func ExportPage(binary string, opts ExportOptions) error {
	args := BuildExportArgs(opts)
	cmd := exec.Command(binary, args...) // #nosec G204 -- binary is auto-detected draw.io CLI path
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("draw.io export failed: %w\nOutput: %s", err, string(output))
	}
	// Verify the output file was actually created (#195).
	if _, err := os.Stat(opts.OutputPath); err != nil {
		return fmt.Errorf("draw.io CLI exited successfully but output file not created: %s", opts.OutputPath)
	}
	return nil
}

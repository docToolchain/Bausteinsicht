// Package export handles exporting draw.io diagrams to PNG/SVG using the
// draw.io CLI.
package export

import (
	"errors"
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

// DrawioPathEnvVar is the environment variable that overrides draw.io binary
// auto-detection. The --drawio-path flag takes precedence over it.
const DrawioPathEnvVar = "BAUSTEINSICHT_DRAWIO_PATH"

// ResolveDrawioBinary determines which draw.io CLI binary to use.
// Precedence (highest first):
//  1. flagPath — the explicit --drawio-path flag value (if non-empty)
//  2. BAUSTEINSICHT_DRAWIO_PATH environment variable (if set)
//  3. auto-detection via DetectDrawioBinary
//
// When an explicit path (flag or env) is given, it must point at an existing
// file; otherwise an error is returned rather than silently falling back to
// auto-detection, so misconfiguration is surfaced loudly.
func ResolveDrawioBinary(flagPath string) (string, error) {
	if flagPath != "" {
		return verifyExplicitBinary(flagPath, "--drawio-path")
	}
	if envPath := os.Getenv(DrawioPathEnvVar); envPath != "" {
		return verifyExplicitBinary(envPath, DrawioPathEnvVar)
	}
	return DetectDrawioBinary()
}

// verifyExplicitBinary checks that an explicitly configured draw.io path exists
// and is not a directory, returning a clear error referencing its source.
func verifyExplicitBinary(path, source string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("draw.io binary from %s not found: %s", source, path)
	}
	if info.IsDir() {
		return "", fmt.Errorf("draw.io binary from %s is a directory, not an executable: %s", source, path)
	}
	return path, nil
}

// DetectDrawioBinary finds the draw.io CLI binary.
// Search order:
//  1. "drawio-export" — devcontainer wrapper (Linux, adds xvfb + --no-sandbox)
//  2. "drawio" — on PATH (Linux package install)
//  3. Platform-native install paths (Windows, macOS) via platformPaths()
func DetectDrawioBinary() (string, error) {
	var searched strings.Builder

	// Try PATH first. "draw.io" is included because Scoop on Windows installs
	// the binary as draw.io.exe and exec.LookPath resolves the .exe extension.
	for _, name := range []string{"drawio-export", "drawio", "draw.io"} {
		path, err := exec.LookPath(name)
		if err == nil {
			return path, nil
		}
		fmt.Fprintf(&searched, "  PATH: %s not found\n", name)
	}

	// Try platform-specific paths
	for _, candidate := range platformPaths() {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		fmt.Fprintf(&searched, "  %s not found\n", candidate)
	}
	return "", buildDrawioNotFoundError(searched.String())
}

// buildDrawioNotFoundError returns a detailed error message with troubleshooting steps and searched paths.
func buildDrawioNotFoundError(searchedPaths string) error {
	msg := strings.Builder{}
	msg.WriteString("draw.io CLI not found\n")
	if searchedPaths != "" {
		msg.WriteString("\nSearched locations:\n")
		msg.WriteString(searchedPaths)
	}
	msg.WriteString("\nInstallation options:\n")
	msg.WriteString("  Windows (Scoop):    scoop install drawio\n")
	msg.WriteString("  Windows (Choco):    choco install drawio\n")
	msg.WriteString("  macOS (Homebrew):   brew install draw.io\n")
	msg.WriteString("  Linux:              See https://www.drawio.com\n\n")
	msg.WriteString("If already installed, try these troubleshooting steps:\n")
	msg.WriteString("  1. Add draw.io to PATH (Scoop): scoop reset drawio\n")
	msg.WriteString("  2. Set env var: export BAUSTEINSICHT_DRAWIO_PATH=/path/to/draw.io\n")
	msg.WriteString("  3. Use CLI flag: bausteinsicht export --drawio-path /path/to/draw.io\n\n")
	msg.WriteString("More info: https://github.com/docToolchain/Bausteinsicht/issues/385\n")
	return errors.New(msg.String())
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
	// The input file is always the last positional argument. No "--" separator is
	// used: draw.io v29+ does not treat "--" as an end-of-options marker and
	// silently fails with exit 0 when it encounters it. The original workaround
	// ("--disable-gpu" landing as paths[0]) is now avoided via ELECTRON_DISABLE_GPU=1
	// in the drawio-export wrapper instead.
	args = append(args, opts.InputFile)
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

// Package export handles exporting draw.io diagrams to PNG/SVG using the
// draw.io CLI.
package export

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

// ExportOptions configures a single page export operation.
type ExportOptions struct {
	Format       string // "png" or "svg"
	PageIndex    int    // 1-based page index
	OutputPath   string // full path to output file
	EmbedDiagram bool   // embed draw.io XML source in output
	InputFile    string // path to the .drawio file
}

// DetectDrawioBinary finds the draw.io CLI binary. It checks for
// "drawio-export" first (devcontainer wrapper with xvfb), then "drawio".
func DetectDrawioBinary() (string, error) {
	for _, name := range []string{"drawio-export", "drawio"} {
		path, err := exec.LookPath(name)
		if err == nil {
			return path, nil
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
	args = append(args, opts.InputFile)
	return args
}

// OutputFileName returns the canonical output file name for a view export.
func OutputFileName(viewKey, format string) string {
	return fmt.Sprintf("architecture-%s.%s", viewKey, format)
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

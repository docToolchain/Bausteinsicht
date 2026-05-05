package lsp

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strings"
)

type Diagnostic struct {
	Range   Range  `json:"range"`
	Message string `json:"message"`
	Severity int `json:"severity"`
	Source  string `json:"source"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

const (
	DiagnosticError   = 1
	DiagnosticWarning = 2
	DiagnosticInfo    = 3
	DiagnosticHint    = 4
)

type ValidateOutput struct {
	Valid   bool `json:"valid"`
	Errors  []ValidationError `json:"errors,omitempty"`
	Warnings []ValidationWarning `json:"warnings,omitempty"`
}

type ValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
}

type ValidationWarning struct {
	Path    string `json:"path"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
}

func ValidateDocument(doc *Document, workDir string) []Diagnostic {
	if !strings.HasSuffix(doc.Filename, "architecture.jsonc") {
		return nil
	}

	// Call bausteinsicht validate --format json
	cmd := exec.Command("bausteinsicht", "validate", "--format", "json", "--model", doc.Filename)
	cmd.Dir = workDir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Command failed, but we might still get useful output
	}

	// Parse JSON output
	var output ValidateOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		// If parsing fails, try to extract errors from stderr
		return parseValidationErrors(stderr.String())
	}

	return convertValidateOutput(&output, doc)
}

func convertValidateOutput(output *ValidateOutput, doc *Document) []Diagnostic {
	var diags []Diagnostic

	// Process errors
	for _, err := range output.Errors {
		line, char := findLineInDocument(doc, err.Path, err.Line)
		diags = append(diags, Diagnostic{
			Range: Range{
				Start: Position{Line: line, Character: char},
				End:   Position{Line: line, Character: char + 10},
			},
			Message:  err.Message,
			Severity: DiagnosticError,
			Source:   "bausteinsicht",
		})
	}

	// Process warnings
	for _, warn := range output.Warnings {
		line, char := findLineInDocument(doc, warn.Path, warn.Line)
		diags = append(diags, Diagnostic{
			Range: Range{
				Start: Position{Line: line, Character: char},
				End:   Position{Line: line, Character: char + 10},
			},
			Message:  warn.Message,
			Severity: DiagnosticWarning,
			Source:   "bausteinsicht",
		})
	}

	return diags
}

func findLineInDocument(doc *Document, path string, preferredLine int) (int, int) {
	lines := strings.Split(doc.Content, "\n")

	// If a preferred line is given, use it (adjusted for 0-indexing)
	if preferredLine > 0 && preferredLine-1 < len(lines) {
		return preferredLine - 1, 0
	}

	// Otherwise, search for the path in the document
	// This is a simple search - real implementation would parse JSON
	for i, line := range lines {
		if strings.Contains(line, path) || strings.Contains(line, strings.TrimPrefix(path, "\"")) {
			return i, 0
		}
	}

	return 0, 0
}

func parseValidationErrors(stderr string) []Diagnostic {
	var diags []Diagnostic

	lines := strings.Split(stderr, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Error:") || strings.Contains(line, "error:") {
			// Extract error message
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				msg := strings.TrimSpace(strings.Join(parts[1:], ":"))
				diags = append(diags, Diagnostic{
					Range: Range{
						Start: Position{Line: 0, Character: 0},
						End:   Position{Line: 0, Character: 10},
					},
					Message:  msg,
					Severity: DiagnosticError,
					Source:   "bausteinsicht",
				})
			}
		}
	}

	return diags
}

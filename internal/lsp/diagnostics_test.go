package lsp

import (
	"testing"
)

func TestConvertValidateOutput(t *testing.T) {
	doc := &Document{
		URI:      "file:///tmp/arch.jsonc",
		Content:  "{\n  \"model\": {\n    \"svc\": {}\n  }\n}",
		Filename: "arch.jsonc",
		Version:  1,
	}

	output := &ValidateOutput{
		Valid: false,
		Errors: []ValidationError{
			{
				Path:    "model.svc",
				Message: "unknown kind",
				Line:    3,
			},
		},
		Warnings: []ValidationWarning{
			{
				Path:    "model.svc",
				Message: "missing description",
				Line:    3,
			},
		},
	}

	diags := convertValidateOutput(output, doc)

	if len(diags) != 2 {
		t.Errorf("expected 2 diagnostics, got %d", len(diags))
	}

	// Check error diagnostic
	if diags[0].Severity != DiagnosticError {
		t.Errorf("expected DiagnosticError, got %d", diags[0].Severity)
	}
	if diags[0].Message != "unknown kind" {
		t.Errorf("expected message 'unknown kind', got %q", diags[0].Message)
	}
	if diags[0].Source != "bausteinsicht" {
		t.Errorf("expected source 'bausteinsicht', got %q", diags[0].Source)
	}
	if diags[0].Range.Start.Line != 2 { // Line 3 → 0-indexed = 2
		t.Errorf("expected line 2, got %d", diags[0].Range.Start.Line)
	}

	// Check warning diagnostic
	if diags[1].Severity != DiagnosticWarning {
		t.Errorf("expected DiagnosticWarning, got %d", diags[1].Severity)
	}
	if diags[1].Message != "missing description" {
		t.Errorf("expected message 'missing description', got %q", diags[1].Message)
	}
}

func TestFindLineInDocument(t *testing.T) {
	doc := &Document{
		Content: `{
  "model": {
    "svc": {
      "kind": "service"
    }
  }
}`,
		Filename: "arch.jsonc",
	}

	tests := []struct {
		path         string
		preferredLine int
		expectedLine int
	}{
		{"model", 2, 1},                    // Line 2 → 0-indexed = 1
		{"model.svc", 3, 2},                // Line 3 → 0-indexed = 2
		{"model.svc.kind", 4, 3},           // Line 4 → 0-indexed = 3
		{"nonexistent", 0, 0},              // Not found → fallback line 0
		{"svc", 0, 2},                      // Search for "svc" in document
	}

	for _, tt := range tests {
		line, _ := findLineInDocument(doc, tt.path, tt.preferredLine)
		if line != tt.expectedLine {
			t.Errorf("findLineInDocument(%q, %d) = %d, want %d",
				tt.path, tt.preferredLine, line, tt.expectedLine)
		}
	}
}

func TestValidateDiagnosticRange(t *testing.T) {
	doc := &Document{
		Content: `{
  "model": {
    "duplicate": {},
    "duplicate": {}
  }
}`,
		Filename: "arch.jsonc",
	}

	output := &ValidateOutput{
		Valid: false,
		Errors: []ValidationError{
			{
				Path:    "model.duplicate",
				Message: "duplicate ID",
				Line:    4,
			},
		},
	}

	diags := convertValidateOutput(output, doc)

	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}

	diag := diags[0]

	// Verify range is set (start and end positions)
	if diag.Range.Start.Line < 0 {
		t.Errorf("range start line should be >= 0, got %d", diag.Range.Start.Line)
	}

	if diag.Range.End.Line < diag.Range.Start.Line {
		t.Errorf("range end line should be >= start line, got %d < %d",
			diag.Range.End.Line, diag.Range.Start.Line)
	}
}

func TestDiagnosticSeverityMapping(t *testing.T) {
	doc := &Document{
		Content:  "{}",
		Filename: "arch.jsonc",
	}

	tests := []struct {
		name         string
		output       ValidateOutput
		expectedSev  int
		expectedMsg  string
	}{
		{
			"error severity",
			ValidateOutput{
				Errors: []ValidationError{
					{Path: "x", Message: "invalid", Line: 1},
				},
			},
			DiagnosticError,
			"invalid",
		},
		{
			"warning severity",
			ValidateOutput{
				Warnings: []ValidationWarning{
					{Path: "x", Message: "deprecated", Line: 1},
				},
			},
			DiagnosticWarning,
			"deprecated",
		},
	}

	for _, tt := range tests {
		diags := convertValidateOutput(&tt.output, doc)
		if len(diags) == 0 {
			t.Errorf("%s: expected at least 1 diagnostic", tt.name)
			continue
		}

		if diags[0].Severity != tt.expectedSev {
			t.Errorf("%s: expected severity %d, got %d", tt.name, tt.expectedSev, diags[0].Severity)
		}

		if diags[0].Message != tt.expectedMsg {
			t.Errorf("%s: expected message %q, got %q", tt.name, tt.expectedMsg, diags[0].Message)
		}
	}
}

func TestEmptyValidateOutput(t *testing.T) {
	doc := &Document{
		Content:  "{}",
		Filename: "arch.jsonc",
	}

	output := &ValidateOutput{
		Valid: true,
	}

	diags := convertValidateOutput(output, doc)

	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics for valid output, got %d", len(diags))
	}
}

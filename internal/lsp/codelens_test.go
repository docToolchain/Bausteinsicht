package lsp

import (
	"strings"
	"testing"
)

func TestGenerateCodeLens(t *testing.T) {
	doc := &Document{
		URI: "file:///tmp/arch.jsonc",
		Content: `{
  "model": {
    "svc": {
      "kind": "service",
      "status": "active"
    }
  }
}`,
		Filename: "architecture.jsonc",
	}

	lenses := GenerateCodeLens(doc)

	if len(lenses) < 1 {
		t.Fatalf("expected at least 1 CodeLens, got %d", len(lenses))
	}

	// Find the "svc" lens
	svcLens := lenses[0]

	// Verify command is set
	if svcLens.Command == nil {
		t.Error("expected command to be set")
	}

	if svcLens.Command.Command != "bausteinsicht.openInDrawio" {
		t.Errorf("expected command 'bausteinsicht.openInDrawio', got %q", svcLens.Command.Command)
	}

	// Verify title includes metadata
	title := svcLens.Command.Title
	if !strings.Contains(title, "service") {
		t.Errorf("expected title to contain 'service', got %q", title)
	}
}

func TestExtractKind(t *testing.T) {
	tests := []struct {
		lines       []string
		startLine   int
		expectedKind string
	}{
		{
			[]string{
				`  "svc": {`,
				`    "kind": "service",`,
				`    "status": "active"`,
				`  }`,
			},
			0,
			"service",
		},
		{
			[]string{
				`  "db": {`,
				`    "kind": "database",`,
			},
			0,
			"database",
		},
		{
			[]string{`  "element": {`},
			0,
			"unknown",
		},
	}

	for _, tt := range tests {
		kind := extractKind(tt.lines, tt.startLine)
		if kind != tt.expectedKind {
			t.Errorf("extractKind() = %q, want %q", kind, tt.expectedKind)
		}
	}
}

func TestExtractStatus(t *testing.T) {
	tests := []struct {
		lines         []string
		startLine     int
		expectedStatus string
	}{
		{
			[]string{
				`  "svc": {`,
				`    "kind": "service",`,
				`    "status": "deprecated"`,
				`  }`,
			},
			0,
			"deprecated",
		},
		{
			[]string{
				`  "api": {`,
				`    "status": "active",`,
			},
			0,
			"active",
		},
		{
			[]string{`  "element": {`},
			0,
			"active", // Default
		},
	}

	for _, tt := range tests {
		status := extractStatus(tt.lines, tt.startLine)
		if status != tt.expectedStatus {
			t.Errorf("extractStatus() = %q, want %q", status, tt.expectedStatus)
		}
	}
}

func TestEstimateViewCount(t *testing.T) {
	tests := []struct {
		content    string
		elementID  string
		minViews   int
		maxViews   int
	}{
		{
			`{"model": {"svc": {}, "api": {"depends_on": "svc"}}}`,
			"svc",
			0,
			5, // Rough estimate, allow some variation
		},
		{
			`{"views": [{"include": "svc"}, {"include": "svc"}]}`,
			"svc",
			0,
			5,
		},
		{
			`{"model": {"unused": {}}}`,
			"unused",
			0,
			1,
		},
	}

	for _, tt := range tests {
		count := estimateViewCount(tt.content, tt.elementID)
		if count < tt.minViews || count > tt.maxViews {
			t.Errorf("estimateViewCount(%q) = %d, want between %d and %d",
				tt.elementID, count, tt.minViews, tt.maxViews)
		}
	}
}

func TestCodeLensRange(t *testing.T) {
	doc := &Document{
		Content: `{
  "model": {
    "api": {
      "kind": "service"
    }
  }
}`,
		Filename: "architecture.jsonc",
	}

	lenses := GenerateCodeLens(doc)

	if len(lenses) == 0 {
		t.Fatal("expected at least 1 CodeLens")
	}

	lens := lenses[0]

	// Verify range is set correctly
	if lens.Range.Start.Line < 0 || lens.Range.Start.Line >= 6 {
		t.Errorf("expected start line between 0-6, got %d", lens.Range.Start.Line)
	}

	if lens.Range.End.Line < lens.Range.Start.Line {
		t.Error("expected end line >= start line")
	}
}

func TestNonArchitectureFile(t *testing.T) {
	doc := &Document{
		URI:      "file:///tmp/other.json",
		Content:  `{"key": "value"}`,
		Filename: "other.json",
	}

	lenses := GenerateCodeLens(doc)

	if lenses != nil && len(lenses) > 0 {
		t.Errorf("expected no CodeLens for non-architecture file, got %d", len(lenses))
	}
}

func TestCodeLensCommand(t *testing.T) {
	doc := &Document{
		URI: "file:///tmp/arch.jsonc",
		Content: `{
  "model": {
    "backend": {
      "kind": "service",
      "status": "stable"
    }
  }
}`,
		Filename: "architecture.jsonc",
	}

	lenses := GenerateCodeLens(doc)

	if len(lenses) == 0 {
		t.Fatal("expected at least 1 CodeLens")
	}

	lens := lenses[0]
	cmd := lens.Command

	// Verify command structure
	if cmd.Title == "" {
		t.Error("expected non-empty title")
	}

	if cmd.Command == "" {
		t.Error("expected non-empty command")
	}

	if len(cmd.Arguments) == 0 {
		t.Error("expected arguments to be set")
	}

	// First argument should be element ID
	if elementID, ok := cmd.Arguments[0].(string); !ok || elementID == "" {
		t.Error("expected first argument to be non-empty element ID")
	}
}

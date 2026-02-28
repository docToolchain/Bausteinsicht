package templates_test

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/docToolchain/Bauteinsicht/templates"
)

func TestDefaultTemplateNotEmpty(t *testing.T) {
	if len(templates.DefaultTemplate) == 0 {
		t.Fatal("DefaultTemplate is empty")
	}
}

func TestSampleModelNotEmpty(t *testing.T) {
	if len(templates.SampleModel) == 0 {
		t.Fatal("SampleModel is empty")
	}
}

func TestSampleModelValidJSON(t *testing.T) {
	// Strip single-line comments before parsing
	content := string(templates.SampleModel)
	var stripped strings.Builder
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		// Remove inline comments (after content)
		if idx := strings.Index(line, " //"); idx >= 0 {
			line = line[:idx]
		}
		stripped.WriteString(line)
		stripped.WriteString("\n")
	}

	// Validate it's valid JSON by checking basic structure
	s := strings.TrimSpace(stripped.String())
	if !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
		t.Fatalf("SampleModel does not look like a JSON object: starts=%q ends=%q",
			s[:min(20, len(s))], s[max(0, len(s)-20):])
	}
}

func TestDefaultTemplateValidXML(t *testing.T) {
	if err := xml.Unmarshal(templates.DefaultTemplate, new(interface{})); err != nil {
		t.Fatalf("DefaultTemplate is not valid XML: %v", err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

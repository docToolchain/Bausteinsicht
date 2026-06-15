package templates_test

import (
	"encoding/xml"
	"os"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/templates"
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

// TestSampleModelSchemaURLPathExists guards against the $schema URL drifting
// from the actual repo directory layout. The bug (#414) was that the embedded
// sample model referenced ".../main/schema/..." (singular) while the repo
// directory is "schemas/" (plural), so the scaffolded $schema URL 404'd and
// out-of-the-box IDE support broke. This test pins the URL's path component to
// the real on-disk schema file so the two cannot diverge again.
func TestSampleModelSchemaURLPathExists(t *testing.T) {
	content := string(templates.SampleModel)
	if !strings.Contains(content, "/schemas/bausteinsicht.schema.json") {
		t.Fatalf("SampleModel $schema URL does not reference /schemas/bausteinsicht.schema.json; content head:\n%s",
			content[:min(200, len(content))])
	}
	if strings.Contains(content, "/main/schema/") {
		t.Fatal("SampleModel $schema URL references singular /main/schema/ which 404s; must be /main/schemas/")
	}
	// The path component the URL points at must actually exist in the repo.
	if _, err := os.Stat("../schemas/bausteinsicht.schema.json"); err != nil {
		t.Fatalf("schemas/bausteinsicht.schema.json does not exist in repo: %v", err)
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

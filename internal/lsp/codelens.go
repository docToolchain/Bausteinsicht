package lsp

import (
	"fmt"
	"regexp"
	"strings"
)

type CodeLens struct {
	Range   Range       `json:"range"`
	Command *Command    `json:"command,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type Command struct {
	Title     string        `json:"title"`
	Command   string        `json:"command"`
	Arguments []interface{} `json:"arguments,omitempty"`
}

type CodeLensData struct {
	ElementID string
	Kind      string
	Status    string
	ViewCount int
}

// GenerateCodeLens extracts element definitions from the document and generates CodeLens objects.
func GenerateCodeLens(doc *Document) []CodeLens {
	if !strings.HasSuffix(doc.Filename, "architecture.jsonc") {
		return nil
	}

	var lenses []CodeLens
	lines := strings.Split(doc.Content, "\n")

	// Pattern: "elementName": { or "elementName": {, matching JSON object keys
	elementPattern := regexp.MustCompile(`^\s*"([a-zA-Z_][a-zA-Z0-9_]*)"\s*:\s*{`)

	for i, line := range lines {
		matches := elementPattern.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue
		}

		elementID := matches[1]

		// Skip parent keys like "model", "views", etc.
		if elementID == "model" || elementID == "views" || elementID == "relationships" {
			continue
		}

		// Extract metadata (kind and status from nearby lines)
		kind := extractKind(lines, i)
		status := extractStatus(lines, i)
		viewCount := estimateViewCount(doc.Content, elementID)

		// Create CodeLens entry
		lens := CodeLens{
			Range: Range{
				Start: Position{Line: i, Character: 0},
				End:   Position{Line: i, Character: len(line)},
			},
			Command: &Command{
				Title:   fmt.Sprintf("%s | status: %s | views: %d", kind, status, viewCount),
				Command: "bausteinsicht.openInDrawio",
				Arguments: []interface{}{
					elementID,
					map[string]interface{}{
						"kind":  kind,
						"status": status,
						"views":  viewCount,
					},
				},
			},
		}

		lenses = append(lenses, lens)
	}

	return lenses
}

// extractKind finds the "kind" field value in the element definition.
func extractKind(lines []string, startLine int) string {
	// Search within the next 10 lines for a "kind" field
	for i := startLine; i < startLine+10 && i < len(lines); i++ {
		if strings.Contains(lines[i], `"kind"`) {
			// Extract value: "kind": "service" → service
			kindPattern := regexp.MustCompile(`"kind"\s*:\s*"([^"]*)"`)
			matches := kindPattern.FindStringSubmatch(lines[i])
			if len(matches) > 1 {
				return matches[1]
			}
		}
	}
	return "unknown"
}

// extractStatus finds the "status" field value in the element definition.
func extractStatus(lines []string, startLine int) string {
	// Search within the next 10 lines for a "status" field
	for i := startLine; i < startLine+10 && i < len(lines); i++ {
		if strings.Contains(lines[i], `"status"`) {
			// Extract value: "status": "active" → active
			statusPattern := regexp.MustCompile(`"status"\s*:\s*"([^"]*)"`)
			matches := statusPattern.FindStringSubmatch(lines[i])
			if len(matches) > 1 {
				return matches[1]
			}
		}
	}
	return "active"
}

// estimateViewCount counts how many views reference this element.
func estimateViewCount(content string, elementID string) int {
	// Count occurrences of the element ID in the document (rough estimate)
	// A more precise implementation would parse the model and count actual view references
	count := strings.Count(content, elementID) - 1 // Subtract 1 for the element definition itself
	if count < 0 {
		count = 0
	}
	return count
}

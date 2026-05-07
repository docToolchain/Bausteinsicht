package benchmarks

import (
	"os"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/diagram"
	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/docToolchain/Bausteinsicht/internal/table"
)

// BenchmarkModelLoad measures time to load JSONC model
func BenchmarkModelLoad(b *testing.B) {
	tmpFile := createTempModel(b, generateLargeModel(100, 200))
	defer os.Remove(tmpFile) //nolint:errcheck

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = model.Load(tmpFile)
	}
}

// BenchmarkModelValidation measures time to validate model
func BenchmarkModelValidation(b *testing.B) {
	m := generateLargeModel(100, 200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = model.Validate(m)
	}
}

// BenchmarkFlattenElements measures time to flatten nested hierarchy
func BenchmarkFlattenElements(b *testing.B) {
	m := generateLargeModel(50, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = model.FlattenElements(m)
	}
}

// BenchmarkDiagramFormatView measures time to format diagram view
func BenchmarkDiagramFormatView(b *testing.B) {
	m := generateLargeModel(50, 100)
	m.Views = map[string]model.View{
		"test": {Title: "Test", Include: []string{"*"}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = diagram.FormatView(m, "test", diagram.PlantUML)
	}
}

// BenchmarkTableExportMarkdown measures time to export as markdown table
func BenchmarkTableExportMarkdown(b *testing.B) {
	m := generateLargeModel(30, 50)
	m.Views = map[string]model.View{
		"default": {Title: "Default", Include: []string{"*"}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = table.FormatCombined(m, table.Markdown)
	}
}

// BenchmarkTableExportAsciidoc measures time to export as AsciiDoc table
func BenchmarkTableExportAsciidoc(b *testing.B) {
	m := generateLargeModel(30, 50)
	m.Views = map[string]model.View{
		"default": {Title: "Default", Include: []string{"*"}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = table.FormatCombined(m, table.AsciiDoc)
	}
}

// Helper functions

func createTempModel(b *testing.B, m *model.BausteinsichtModel) string {
	f, err := os.CreateTemp("", "model-*.jsonc")
	if err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}
	defer f.Close() //nolint:errcheck

	if err := model.Save(f.Name(), m); err != nil {
		b.Fatalf("failed to save model: %v", err)
	}

	return f.Name()
}

// generateLargeModel creates a synthetic model for benchmarking
// elementCount: number of top-level elements
// relationshipCount: number of relationships
func generateLargeModel(elementCount, relationshipCount int) *model.BausteinsichtModel {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system": {Notation: "box", Container: true},
				"service": {Notation: "component", Container: true},
				"module": {Notation: "box"},
			},
			Relationships: map[string]model.RelationshipKind{
				"calls": {Notation: "arrow", Dashed: false},
				"async": {Notation: "arrow", Dashed: true},
			},
		},
		Model: make(map[string]model.Element),
		Relationships: []model.Relationship{},
		Views: map[string]model.View{
			"overview": {
				Title: "Overview",
				Include: []string{"*"},
			},
		},
	}

	// Create elements
	for i := 0; i < elementCount; i++ {
		id := "elem" + string(rune(i))
		m.Model[id] = model.Element{
			Kind:        "system",
			Title:       "Element " + id,
			Description: "Test element for benchmarking",
			Technology:  "Go",
		}
	}

	// Create relationships
	for i := 0; i < relationshipCount && i < elementCount*elementCount; i++ {
		from := "elem" + string(rune(i%elementCount))
		to := "elem" + string(rune((i+1)%elementCount))
		if from != to {
			m.Relationships = append(m.Relationships, model.Relationship{
				From:  from,
				To:    to,
				Label: "calls",
				Kind:  "calls",
			})
		}
	}

	return m
}

// BenchmarkLargeModelLoad loads a model with 500+ elements
func BenchmarkLargeModelLoad(b *testing.B) {
	tmpFile := createTempModel(b, generateLargeModel(500, 1000))
	defer os.Remove(tmpFile) //nolint:errcheck

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = model.Load(tmpFile)
	}
}

// BenchmarkLargeModelValidation validates a large model
func BenchmarkLargeModelValidation(b *testing.B) {
	m := generateLargeModel(500, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = model.Validate(m)
	}
}

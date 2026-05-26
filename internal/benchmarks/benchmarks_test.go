package benchmarks

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/docToolchain/Bausteinsicht/internal/diagram"
	"github.com/docToolchain/Bausteinsicht/internal/table"
)

// generateSyntheticModel creates a test model with specified element and relationship counts.
func generateSyntheticModel(numElements, numRelationships int) *model.BausteinsichtModel {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system":    {Notation: "box", Container: true},
				"container": {Notation: "box", Container: true},
				"component": {Notation: "box"},
			},
			Relationships: map[string]model.RelationshipKind{
				"uses": {Notation: "arrow"},
			},
		},
		Model: make(map[string]model.Element),
	}

	// Generate elements
	for i := 0; i < numElements; i++ {
		id := fmt.Sprintf("elem_%d", i)
		kind := "component"
		if i%5 == 0 {
			kind = "system"
		} else if i%3 == 0 {
			kind = "container"
		}
		m.Model[id] = model.Element{
			Kind:  kind,
			Title: fmt.Sprintf("Element %d", i),
		}
	}

	// Generate relationships
	for i := 0; i < numRelationships && i < numElements*2; i++ {
		from := fmt.Sprintf("elem_%d", i%numElements)
		to := fmt.Sprintf("elem_%d", (i+1)%numElements)
		m.Relationships = append(m.Relationships, model.Relationship{
			From:  from,
			To:    to,
			Label: fmt.Sprintf("rel_%d", i),
			Kind:  "uses",
		})
	}

	return m
}

// BenchmarkModelLoad benchmarks JSONC model parsing and unmarshaling.
func BenchmarkModelLoad(b *testing.B) {
	m := generateSyntheticModel(100, 200)
	data, _ := json.Marshal(m)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result model.BausteinsichtModel
		_ = json.Unmarshal(data, &result)
	}
}

// BenchmarkModelLoadLarge benchmarks parsing of larger models.
func BenchmarkModelLoadLarge(b *testing.B) {
	m := generateSyntheticModel(500, 1000)
	data, _ := json.Marshal(m)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result model.BausteinsichtModel
		_ = json.Unmarshal(data, &result)
	}
}

// BenchmarkModelValidation benchmarks validation of a typical model.
func BenchmarkModelValidation(b *testing.B) {
	m := generateSyntheticModel(100, 200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.Validate(m)
	}
}

// BenchmarkModelValidationLarge benchmarks validation at scale.
func BenchmarkModelValidationLarge(b *testing.B) {
	m := generateSyntheticModel(500, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.Validate(m)
	}
}

// BenchmarkFlattenElements benchmarks hierarchy flattening.
func BenchmarkFlattenElements(b *testing.B) {
	m := generateSyntheticModel(100, 200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = model.FlattenElements(m)
	}
}

// BenchmarkDiagramFormatView benchmarks diagram generation for a single view.
func BenchmarkDiagramFormatView(b *testing.B) {
	m := generateSyntheticModel(50, 100)

	// Create a view
	m.Views = map[string]model.View{
		"test": {
			Title:   "Test View",
			Include: []string{"elem_0", "elem_1", "elem_2", "elem_3", "elem_4"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = diagram.FormatView(m, "test", diagram.PlantUML)
	}
}

// BenchmarkTableExportMarkdown benchmarks markdown table export.
func BenchmarkTableExportMarkdown(b *testing.B) {
	m := generateSyntheticModel(100, 200)
	m.Views = map[string]model.View{
		"test": {Title: "Test View", Include: []string{"elem_*"}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = table.FormatView(m, "test", table.Markdown)
	}
}

// BenchmarkTableExportAsciidoc benchmarks AsciiDoc table export.
func BenchmarkTableExportAsciidoc(b *testing.B) {
	m := generateSyntheticModel(100, 200)
	m.Views = map[string]model.View{
		"test": {Title: "Test View", Include: []string{"elem_*"}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = table.FormatView(m, "test", table.AsciiDoc)
	}
}

// BenchmarkRoundTripJSON benchmarks JSON marshal/unmarshal cycle.
func BenchmarkRoundTripJSON(b *testing.B) {
	m := generateSyntheticModel(100, 200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _ := json.Marshal(m)
		var result model.BausteinsichtModel
		_ = json.Unmarshal(data, &result)
	}
}

// BenchmarkFileWrite benchmarks saving a model to disk.
func BenchmarkFileWrite(b *testing.B) {
	m := generateSyntheticModel(100, 200)
	tmpdir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := filepath.Join(tmpdir, fmt.Sprintf("model_%d.json", i))
		_ = model.Save(path, m)
	}
}

// BenchmarkFileRead benchmarks loading a model from disk.
func BenchmarkFileRead(b *testing.B) {
	m := generateSyntheticModel(100, 200)
	tmpdir := b.TempDir()
	path := filepath.Join(tmpdir, "model.json")
	if err := model.Save(path, m); err != nil {
		b.Fatalf("Save failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = model.Load(path)
	}
}

// BenchmarkValidateAndFlatten combines two common operations.
func BenchmarkValidateAndFlatten(b *testing.B) {
	m := generateSyntheticModel(100, 200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.Validate(m)
		_, _ = model.FlattenElements(m)
	}
}

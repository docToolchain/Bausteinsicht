// Package importer provides parsers for importing architecture models from
// external formats (Structurizr DSL, LikeC4) into Bausteinsicht.
package importer

import "github.com/docToolchain/Bausteinsicht/internal/model"

// ImportResult holds the imported model and non-fatal warnings produced during import.
type ImportResult struct {
	Model    *model.BausteinsichtModel
	Warnings []string
}

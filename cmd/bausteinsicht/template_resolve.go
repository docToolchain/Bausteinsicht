package main

import (
	"os"
	"path/filepath"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/templates"
)

// resolveTemplate loads the draw.io TemplateSet used for forward sync, applying
// the precedence: explicit --template flag > local "template.drawio" next to the
// model > embedded default. This ensures the template scaffolded by `init` (which
// the tutorial tells users to edit) is actually honored by `sync` and `watch`
// even when no --template flag is given (#418).
//
// modelDir is the directory containing the model file; templatePath is the value
// of the --template flag (empty when not set).
func resolveTemplate(templatePath, modelDir string) (*drawio.TemplateSet, error) {
	if templatePath != "" {
		return drawio.LoadTemplate(templatePath)
	}

	localTemplate := filepath.Join(modelDir, defaultTemplFile)
	if info, err := os.Stat(localTemplate); err == nil && !info.IsDir() {
		return drawio.LoadTemplate(localTemplate)
	}

	return drawio.LoadTemplateFromBytes(templates.DefaultTemplate)
}

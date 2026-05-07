package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// LoadConfig reads and parses a workspace configuration file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

// LoadModels loads all models referenced in the configuration.
// The basePath is used to resolve relative model paths.
func LoadModels(cfg *Config, basePath string) ([]LoadedModel, error) {
	var loaded []LoadedModel

	for _, ref := range cfg.Models {
		modelPath := ref.Path
		// If path is relative, resolve it relative to basePath
		if !filepath.IsAbs(modelPath) {
			modelPath = filepath.Join(basePath, modelPath)
		}

		m, err := model.Load(modelPath)
		if err != nil {
			return nil, fmt.Errorf("loading model %s (%s): %w", ref.ID, ref.Path, err)
		}

		loaded = append(loaded, LoadedModel{
			Ref:   ref,
			Model: m,
		})
	}

	return loaded, nil
}

// SaveConfig writes a workspace configuration to a file.
func SaveConfig(cfg *Config, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

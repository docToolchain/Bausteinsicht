package workspace

import (
	"encoding/json"
	"errors"
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
//
// Local models are loaded from their path. Remote models (Source) are read
// from the cache next to the workspace configuration, after the cached file has
// been verified against bausteinsicht.lock; nothing is fetched here.
func LoadModels(cfg *Config, basePath string) ([]LoadedModel, error) {
	var loaded []LoadedModel

	if problems := ValidateSources(cfg); len(problems) > 0 {
		return nil, errors.Join(problems...)
	}
	lock := &Lock{Sources: map[string]LockEntry{}}
	for _, ref := range cfg.Models {
		if ref.Source != "" {
			l, err := LoadLock(filepath.Join(basePath, LockFileName))
			if err != nil {
				return nil, err
			}
			lock = l
			break
		}
	}

	for _, ref := range cfg.Models {
		modelPath, err := resolveModelFile(ref, basePath, lock)
		if err != nil {
			return nil, err
		}
		where := ref.Path
		if ref.Source != "" {
			where = ref.Source
		}

		m, err := model.Load(modelPath)
		if err != nil {
			return nil, fmt.Errorf("loading model %s (%s): %w", ref.ID, where, err)
		}

		m, err = ApplySelection(m, ref)
		if err != nil {
			return nil, fmt.Errorf("model %s (%s): %w", ref.ID, where, err)
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

package model

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Load reads a JSONC file, strips comments and trailing commas, and parses it.
func Load(path string) (*BausteinsichtModel, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	clean := StripJSONC(data)
	var m BausteinsichtModel
	if err := json.Unmarshal(clean, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &m, nil
}

// Save marshals the model and atomically writes it to path.
func Save(path string, model *BausteinsichtModel) error {
	data, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling model: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("renaming temp file: %w", err)
	}
	return nil
}

// AutoDetect finds the first *.jsonc file in dir.
func AutoDetect(dir string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.jsonc"))
	if err != nil {
		return "", fmt.Errorf("scanning %s: %w", dir, err)
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no .jsonc file found in %s", dir)
	}
	return matches[0], nil
}

// StripJSONC removes single-line comments and trailing commas from JSONC data.
// Comments inside strings are preserved.
func StripJSONC(data []byte) []byte {
	var sb strings.Builder
	src := string(data)
	i := 0
	for i < len(src) {
		// Handle string literals — skip their content intact
		if src[i] == '"' {
			sb.WriteByte(src[i])
			i++
			for i < len(src) {
				if src[i] == '\\' && i+1 < len(src) {
					sb.WriteByte(src[i])
					sb.WriteByte(src[i+1])
					i += 2
					continue
				}
				sb.WriteByte(src[i])
				if src[i] == '"' {
					i++
					break
				}
				i++
			}
			continue
		}
		// Handle single-line comments
		if i+1 < len(src) && src[i] == '/' && src[i+1] == '/' {
			// Trim trailing whitespace written before the comment
			s := sb.String()
			trimmed := strings.TrimRight(s, " \t")
			sb.Reset()
			sb.WriteString(trimmed)
			for i < len(src) && src[i] != '\n' {
				i++
			}
			continue
		}
		sb.WriteByte(src[i])
		i++
	}

	// Remove trailing commas before } or ]
	result := trailingCommaRe.ReplaceAllString(sb.String(), "$1")
	return []byte(result)
}

// trailingCommaRe matches a comma optionally followed by whitespace before } or ]
var trailingCommaRe = regexp.MustCompile(`,(\s*[}\]])`)

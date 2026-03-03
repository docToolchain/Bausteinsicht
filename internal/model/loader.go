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
	data, err := os.ReadFile(path) // #nosec G304 -- path from CLI flag
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	clean := StripJSONC(data)

	// Reject null JSON root — json.Unmarshal silently accepts "null"
	// and produces a zero-value struct, which passes validation vacuously.
	trimmed := strings.TrimSpace(string(clean))
	if trimmed == "null" || trimmed == "" {
		return nil, fmt.Errorf("parsing %s: model file is empty or contains a null JSON root", path)
	}

	var m BausteinsichtModel
	if err := json.Unmarshal(clean, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &m, nil
}

// Save marshals the model and atomically writes it to path.
// Uses os.CreateTemp for a randomized temp file name to prevent TOCTOU attacks.
func Save(path string, model *BausteinsichtModel) error {
	data, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling model: %w", err)
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".model-tmp-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("closing temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
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
	// Strip UTF-8 BOM if present.
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}

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
		// Handle block comments
		if i+1 < len(src) && src[i] == '/' && src[i+1] == '*' {
			// Trim trailing whitespace before comment if it's only
			// whitespace since the last newline (i.e., comment on its own line).
			s := sb.String()
			lastNL := strings.LastIndex(s, "\n")
			linePrefix := s[lastNL+1:]
			if strings.TrimRight(linePrefix, " \t") == "" {
				sb.Reset()
				sb.WriteString(s[:lastNL+1])
			}
			i += 2
			for i+1 < len(src) {
				if src[i] == '*' && src[i+1] == '/' {
					i += 2
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

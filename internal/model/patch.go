package model

import (
	"fmt"
	"os"
	"path/filepath"
)

// PatchOp describes a single value replacement in a JSONC file.
type PatchOp struct {
	Path  []string // JSON path segments, e.g., ["model", "api", "technology"]
	Value string   // New JSON-encoded value, e.g., `"Go 1.24"`
}

// PatchSave reads the JSONC file at path, applies each PatchOp, and writes
// the result back atomically. Comments, formatting, and key ordering are
// preserved because only the target values are replaced in the raw text.
func PatchSave(path string, ops []PatchOp) error {
	data, err := os.ReadFile(path) // #nosec G304 -- path from CLI flag
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	for _, op := range ops {
		data, err = PatchValue(data, op.Path, op.Value)
		if err != nil {
			return fmt.Errorf("patching %v: %w", op.Path, err)
		}
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

// PatchInsert reads the JSONC file at path, applies a raw data transformation,
// and writes the result back atomically. Used by InsertObjectEntry and
// AppendArrayEntry for comment-preserving insertions.
func PatchInsert(path string, transform func([]byte) ([]byte, error)) error {
	data, err := os.ReadFile(path) // #nosec G304 -- path from CLI flag
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	data, err = transform(data)
	if err != nil {
		return err
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

// PatchValue finds the JSON value at path in JSONC data and replaces it with
// newValue. The rest of the document (comments, whitespace, key ordering)
// is preserved. Returns the patched data or an error if the path is not found.
func PatchValue(data []byte, path []string, newValue string) ([]byte, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("empty path")
	}

	start, end, err := findValueRange(data, path)
	if err != nil {
		return nil, err
	}

	result := make([]byte, 0, len(data)+len(newValue))
	result = append(result, data[:start]...)
	result = append(result, []byte(newValue)...)
	result = append(result, data[end:]...)
	return result, nil
}

// findValueRange locates the byte range [start, end) of the JSON value at
// the given path within JSONC data. It handles single-line comments and
// string escaping.
func findValueRange(data []byte, path []string) (int, int, error) {
	i := 0
	n := len(data)

	for depth := 0; depth < len(path); depth++ {
		key := path[depth]
		// Find the opening { of the current object.
		i = skipToChar(data, i, n, '{')
		if i >= n {
			return 0, 0, fmt.Errorf("path %v: expected object, not found", path[:depth+1])
		}
		i++ // skip '{'

		// Find the matching key within this object.
		found := false
		for i < n {
			i = skipWhitespaceAndComments(data, i, n)
			if i >= n {
				break
			}
			if data[i] == '}' {
				break
			}

			// Read the key.
			if data[i] != '"' {
				return 0, 0, fmt.Errorf("expected '\"' at offset %d", i)
			}
			keyStart := i
			keyEnd := skipString(data, i, n)
			currentKey := string(data[keyStart+1 : keyEnd-1]) // strip quotes
			i = keyEnd

			// Skip colon.
			i = skipWhitespaceAndComments(data, i, n)
			if i >= n || data[i] != ':' {
				return 0, 0, fmt.Errorf("expected ':' after key %q at offset %d", currentKey, i)
			}
			i++ // skip ':'
			i = skipWhitespaceAndComments(data, i, n)

			if currentKey == key {
				if depth == len(path)-1 {
					// This is the target value — find its extent.
					valStart := i
					valEnd := skipValue(data, i, n)
					return valStart, valEnd, nil
				}
				// Need to descend into this value (next iteration).
				found = true
				break
			}

			// Skip the value to move to the next key.
			i = skipValue(data, i, n)

			// Skip optional comma.
			i = skipWhitespaceAndComments(data, i, n)
			if i < n && data[i] == ',' {
				i++
			}
		}
		if !found && depth < len(path)-1 {
			return 0, 0, fmt.Errorf("key %q not found in path %v", key, path[:depth+1])
		}
	}

	return 0, 0, fmt.Errorf("path %v not found", path)
}

// skipWhitespaceAndComments advances past whitespace and // comments.
func skipWhitespaceAndComments(data []byte, i, n int) int {
	for i < n {
		if data[i] == ' ' || data[i] == '\t' || data[i] == '\n' || data[i] == '\r' {
			i++
			continue
		}
		if i+1 < n && data[i] == '/' && data[i+1] == '/' {
			// Skip to end of line.
			for i < n && data[i] != '\n' {
				i++
			}
			continue
		}
		if i+1 < n && data[i] == '/' && data[i+1] == '*' {
			i += 2
			for i+1 < n {
				if data[i] == '*' && data[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
			continue
		}
		break
	}
	return i
}

// skipString skips a JSON string starting at data[i] (which must be '"')
// and returns the index after the closing quote.
func skipString(data []byte, i, n int) int {
	i++ // skip opening '"'
	for i < n {
		if data[i] == '\\' && i+1 < n {
			i += 2
			continue
		}
		if data[i] == '"' {
			return i + 1
		}
		i++
	}
	return i
}

// skipValue skips a complete JSON value (string, number, object, array, bool, null).
func skipValue(data []byte, i, n int) int {
	if i >= n {
		return i
	}
	switch data[i] {
	case '"':
		return skipString(data, i, n)
	case '{':
		return skipBraced(data, i, n, '{', '}')
	case '[':
		return skipBraced(data, i, n, '[', ']')
	default:
		// Number, bool, null — skip until delimiter.
		for i < n {
			c := data[i]
			if c == ',' || c == '}' || c == ']' || c == ' ' || c == '\t' || c == '\n' || c == '\r' {
				break
			}
			// Also stop at // or /* comment.
			if c == '/' && i+1 < n && (data[i+1] == '/' || data[i+1] == '*') {
				break
			}
			i++
		}
		return i
	}
}

// skipBraced skips a matched pair of braces/brackets, handling strings and
// comments within.
func skipBraced(data []byte, i, n int, open, close byte) int {
	depth := 0
	for i < n {
		c := data[i]
		if c == '"' {
			i = skipString(data, i, n)
			continue
		}
		if c == '/' && i+1 < n && data[i+1] == '/' {
			for i < n && data[i] != '\n' {
				i++
			}
			continue
		}
		if c == '/' && i+1 < n && data[i+1] == '*' {
			i += 2
			for i+1 < n {
				if data[i] == '*' && data[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
			continue
		}
		switch c {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return i + 1
			}
		}
		i++
	}
	return i
}

// InsertObjectEntry inserts a new key-value pair into the object at the given
// path. The value is inserted before the closing '}' of the target object.
// Comments and formatting are preserved.
func InsertObjectEntry(data []byte, objectPath []string, key, valueJSON string) ([]byte, error) {
	// Find the object's value range.
	start, end, err := findValueRange(data, objectPath)
	if err != nil {
		return nil, err
	}
	if data[start] != '{' {
		return nil, fmt.Errorf("value at path %v is not an object", objectPath)
	}

	// Find the closing '}' of this object (end-1 since findValueRange returns after it).
	closeBrace := end - 1
	for closeBrace > start && data[closeBrace] != '}' {
		closeBrace--
	}

	// Detect indentation from the closing brace line.
	indent := detectIndent(data, closeBrace)

	// Check if the object has existing entries by scanning for non-whitespace
	// between '{' and '}'.
	hasEntries := false
	scan := start + 1
	scan = skipWhitespaceAndComments(data, scan, len(data))
	if scan < closeBrace {
		hasEntries = true
	}

	// Build the insertion text.
	var insertion string
	if hasEntries {
		// Find the last non-whitespace/comment byte before closeBrace to
		// append a comma after the previous entry (not before the new one).
		lastContent := closeBrace - 1
		for lastContent > start && (data[lastContent] == ' ' || data[lastContent] == '\t' || data[lastContent] == '\n' || data[lastContent] == '\r') {
			lastContent--
		}
		// Insert comma after last entry, then newline + new entry.
		comma := ""
		if lastContent > start && data[lastContent] != ',' {
			comma = ","
		}
		insertion = fmt.Sprintf("%s\n%s  %q: %s\n%s", comma, indent, key, valueJSON, indent)
		// Replace from after last content to closing brace (inclusive) with:
		// comma + \n + indent + new entry + \n + indent + }
		result := make([]byte, 0, len(data)+len(insertion))
		result = append(result, data[:lastContent+1]...)
		result = append(result, []byte(insertion)...)
		result = append(result, data[closeBrace:]...)
		return result, nil
	}

	insertion = fmt.Sprintf("\n%s  %q: %s\n%s", indent, key, valueJSON, indent)
	result := make([]byte, 0, len(data)+len(insertion))
	result = append(result, data[:closeBrace]...)
	result = append(result, []byte(insertion)...)
	result = append(result, data[closeBrace:]...)
	return result, nil
}

// AppendArrayEntry appends a new value to the array at the given path.
// The value is inserted before the closing ']'. Comments and formatting
// are preserved.
func AppendArrayEntry(data []byte, arrayPath []string, valueJSON string) ([]byte, error) {
	start, end, err := findValueRange(data, arrayPath)
	if err != nil {
		return nil, err
	}
	if data[start] != '[' {
		return nil, fmt.Errorf("value at path %v is not an array", arrayPath)
	}

	// Find the closing ']'.
	closeBracket := end - 1
	for closeBracket > start && data[closeBracket] != ']' {
		closeBracket--
	}

	indent := detectIndent(data, closeBracket)

	// Check if the array has existing entries.
	hasEntries := false
	scan := start + 1
	scan = skipWhitespaceAndComments(data, scan, len(data))
	if scan < closeBracket {
		hasEntries = true
	}

	var insertion string
	if hasEntries {
		// Find the last non-whitespace byte before closeBracket to
		// append comma after the previous entry.
		lastContent := closeBracket - 1
		for lastContent > start && (data[lastContent] == ' ' || data[lastContent] == '\t' || data[lastContent] == '\n' || data[lastContent] == '\r') {
			lastContent--
		}
		comma := ""
		if lastContent > start && data[lastContent] != ',' {
			comma = ","
		}
		insertion = fmt.Sprintf("%s\n%s  %s\n%s", comma, indent, valueJSON, indent)
		result := make([]byte, 0, len(data)+len(insertion))
		result = append(result, data[:lastContent+1]...)
		result = append(result, []byte(insertion)...)
		result = append(result, data[closeBracket:]...)
		return result, nil
	}

	insertion = fmt.Sprintf("\n%s  %s\n%s", indent, valueJSON, indent)
	result := make([]byte, 0, len(data)+len(insertion))
	result = append(result, data[:closeBracket]...)
	result = append(result, []byte(insertion)...)
	result = append(result, data[closeBracket:]...)
	return result, nil
}

// detectIndent returns the whitespace prefix of the line containing position pos.
func detectIndent(data []byte, pos int) string {
	lineStart := pos
	for lineStart > 0 && data[lineStart-1] != '\n' {
		lineStart--
	}
	indent := ""
	for i := lineStart; i < pos; i++ {
		if data[i] == ' ' || data[i] == '\t' {
			indent += string(data[i])
		} else {
			break
		}
	}
	return indent
}

// skipToChar advances to the first occurrence of ch, skipping strings and comments.
func skipToChar(data []byte, i, n int, ch byte) int {
	for i < n {
		if data[i] == '"' {
			i = skipString(data, i, n)
			continue
		}
		if data[i] == '/' && i+1 < n && data[i+1] == '/' {
			for i < n && data[i] != '\n' {
				i++
			}
			continue
		}
		if data[i] == '/' && i+1 < n && data[i+1] == '*' {
			i += 2
			for i+1 < n {
				if data[i] == '*' && data[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
			continue
		}
		if data[i] == ch {
			return i
		}
		i++
	}
	return i
}

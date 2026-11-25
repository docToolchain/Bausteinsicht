package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// savePatched writes a change to the model file with a comment-preserving
// patch (see ADR-011). If the patch cannot be applied, it falls back to a full
// model.Save and says so on w, because that rewrites the whole file: comments
// are lost, keys are sorted and one-line objects are expanded (#635).
func savePatched(w io.Writer, modelPath string, m *model.BausteinsichtModel, patch func([]byte) ([]byte, error)) error {
	if err := model.PatchInsert(modelPath, patch); err == nil {
		return nil
	}
	_, _ = fmt.Fprintln(w, "note: model rewritten, comments and formatting were not preserved")
	return model.Save(modelPath, m)
}

// marshalEntryJSON renders v as JSON for use as the value of an object entry
// whose key sits at the given depth (each level is two spaces). The first line
// carries no indent, the following lines are indented to match the entry.
func marshalEntryJSON(v any, depth int) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent(strings.Repeat("  ", depth), "  ")
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

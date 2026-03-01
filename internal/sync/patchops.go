package sync

import (
	"strings"

	"github.com/docToolchain/Bauteinsicht/internal/model"
)

// ReversePatchOps converts the reverse (draw.io → model) changes in cs into
// PatchOps that can be applied to the JSONC file directly. Returns the ops
// and true if all changes are patchable (only element field modifications).
// Returns nil, false if any structural change (add/delete) or relationship
// change is present — the caller should fall back to full Save.
func ReversePatchOps(cs *ChangeSet) ([]model.PatchOp, bool) {
	// Any relationship change or structural element change means we can't patch.
	if len(cs.DrawioRelationshipChanges) > 0 {
		return nil, false
	}

	var ops []model.PatchOp
	for _, ch := range cs.DrawioElementChanges {
		if ch.Type != Modified || ch.Field == "" {
			return nil, false
		}
		path := elementFieldPath(ch.ID, ch.Field)
		ops = append(ops, model.PatchOp{
			Path:  path,
			Value: `"` + jsonEscapeString(ch.NewValue) + `"`,
		})
	}
	return ops, true
}

// elementFieldPath converts a dot-separated element ID and field name into
// a JSON path. E.g., ("webshop.api", "technology") →
// ["model", "webshop", "children", "api", "technology"]
func elementFieldPath(id, field string) []string {
	parts := strings.Split(id, ".")
	path := []string{"model"}
	for i, part := range parts {
		path = append(path, part)
		if i < len(parts)-1 {
			path = append(path, "children")
		}
	}
	path = append(path, field)
	return path
}

// jsonEscapeString escapes special characters for JSON string values.
func jsonEscapeString(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

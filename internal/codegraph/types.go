// Package codegraph implements the reverse contract of the code-governance
// bridge (#562, ADR-013): a minimal, language-neutral package-dependency
// graph that any code-side extractor can emit (jdeps, an ArchUnit dumper,
// go list, Roslyn, ...), and that Bausteinsicht projects onto model elements
// via their codeMapping to produce an asIs snapshot.
package codegraph

import (
	"encoding/json"
	"fmt"
	"os"
)

// schemaKind is the required CodeGraph.Kind value — a cheap guard against
// pointing import-graph at an unrelated JSON file.
const schemaKind = "bausteinsicht.codegraph"

// SchemaTitle and SchemaDescription are the JSON Schema title/description
// used by `bausteinsicht schema generate --type codegraph`, exported so the
// CLI command and its drift-guard test share one source of truth instead of
// duplicating the strings.
const (
	SchemaTitle       = "Bausteinsicht CodeGraph"
	SchemaDescription = "Reverse code-governance-bridge contract (#562, ADR-013): a language-neutral package-dependency graph emitted by a code-side extractor"
)

// CodeGraph is the reverse contract: a real package-dependency graph
// extracted from code, independent of any Bausteinsicht model.
type CodeGraph struct {
	SchemaVersion string `json:"schemaVersion"`
	Kind          string `json:"kind"`
	Language      string `json:"language"`
	Nodes         []Node `json:"nodes"`
	Edges         []Edge `json:"edges"`
}

// Node is one package in the extracted graph.
type Node struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

// Edge is one observed package-level dependency. Weight and Examples are
// optional context for a human resolving drift (the "why" behind an edge).
type Edge struct {
	From     string   `json:"from"`
	To       string   `json:"to"`
	Weight   int      `json:"weight,omitempty"`
	Examples []string `json:"examples,omitempty"`
}

// Load reads and validates a codegraph JSON document from path.
func Load(path string) (*CodeGraph, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is CLI-supplied and containment-checked by the caller
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var cg CodeGraph
	if err := json.Unmarshal(data, &cg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	if cg.Kind != schemaKind {
		return nil, fmt.Errorf("%s: kind %q, expected %q", path, cg.Kind, schemaKind)
	}
	if cg.SchemaVersion == "" {
		return nil, fmt.Errorf("%s: missing required field \"schemaVersion\"", path)
	}

	return &cg, nil
}

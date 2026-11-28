package e2e

// #730: graph on a model with some thousand elements finishes in seconds, not minutes.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGraphPerformanceBudget(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	spec := map[string]any{"elements": map[string]any{
		"system": map[string]any{"notation": "S", "container": true},
		"svc":    map[string]any{"notation": "V", "container": true},
		"db":     map[string]any{"notation": "D"},
	}, "relationships": map[string]any{"uses": map[string]any{"notation": "uses"}}}
	model := map[string]any{}
	var rels []map[string]any
	const rings, services, dbs = 30, 10, 3 // 1230 elements
	for i := 0; i < rings; i++ {
		children := map[string]any{}
		for j := 0; j < services; j++ {
			dbc := map[string]any{}
			for k := 0; k < dbs; k++ {
				dbc[fmt.Sprintf("db%d", k)] = map[string]any{"kind": "db", "title": "db"}
			}
			children[fmt.Sprintf("svc%d", j)] = map[string]any{"kind": "svc", "title": "svc", "children": dbc}
		}
		model[fmt.Sprintf("sys%d", i)] = map[string]any{"kind": "system", "title": "sys", "children": children}
	}
	for i := 0; i < rings; i++ {
		for j := 0; j < services; j++ {
			from := fmt.Sprintf("sys%d.svc%d", i, j)
			rels = append(rels,
				map[string]any{"from": from, "to": fmt.Sprintf("sys%d.svc%d", (i+1)%rings, j), "kind": "uses", "label": "calls"},
				map[string]any{"from": from, "to": from + ".db0", "kind": "uses"})
		}
	}
	data, err := json.Marshal(map[string]any{"specification": spec, "model": model, "relationships": rels,
		"views": map[string]any{"all": map[string]any{"title": "All", "include": []string{"**"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "m.jsonc"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	runCLI(t, bin, dir, "--model", "m.jsonc", "graph")
	elapsed := time.Since(start)
	const budget = 10 * time.Second
	if elapsed > budget {
		t.Errorf("graph of 1230 elements took %v, budget %v", elapsed, budget)
	}
	t.Logf("graph of 1230 elements: %v", elapsed)
}

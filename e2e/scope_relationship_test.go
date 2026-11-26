package e2e

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestSyncKeepsRelationshipsOfTheScopeElement (#736): a relationship that ends
// at the scope element of a view, with its other end shown on the same page, is
// not drawn on that page. Syncing again used to read the missing connector as a
// user deletion and removed the relationship from the model.
func TestSyncKeepsRelationshipsOfTheScopeElement(t *testing.T) {
	bin := buildBinary(t)
	const spec = `"specification": {
    "elements": { "actor": { "notation": "Actor" }, "system": { "notation": "System", "container": true } },
    "relationships": { "uses": { "notation": "uses" } } }`

	tests := []struct {
		name, model, rel, view string
	}{
		{"external to scope", `"user": { "kind": "actor", "title": "User" }, "shop": { "kind": "system", "title": "Shop" }`,
			`{ "from": "user", "to": "shop", "kind": "uses", "label": "orders" }`, `{ "title": "Detail", "scope": "shop", "include": ["*"] }`},
		{"scope to external", `"user": { "kind": "actor", "title": "User" }, "shop": { "kind": "system", "title": "Shop" }`,
			`{ "from": "shop", "to": "user", "kind": "uses", "label": "notifies" }`, `{ "title": "Detail", "scope": "shop", "include": ["**"] }`},
		{"nested scope", `"user": { "kind": "actor", "title": "User" }, "shop": { "kind": "system", "title": "Shop", "children": { "api": { "kind": "system", "title": "API" } } }`,
			`{ "from": "user", "to": "shop.api", "kind": "uses", "label": "calls" }`, `{ "title": "Detail", "scope": "shop.api", "include": ["*"] }`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir+"/architecture.jsonc", `{ `+spec+`,
  "model": { `+tt.model+` },
  "relationships": [ `+tt.rel+` ],
  "views": { "detail": `+tt.view+` } }`)

			for i := 1; i <= 3; i++ {
				out := runCLI(t, bin, dir, "sync")
				if strings.Contains(out, "deleted") && strings.Contains(out, "Reverse") && !strings.Contains(out, "0 deleted") {
					t.Errorf("sync %d reported a deletion from draw.io:\n%s", i, out)
				}
				data, err := os.ReadFile(dir + "/architecture.jsonc")
				if err != nil {
					t.Fatal(err)
				}
				var m struct {
					Relationships []json.RawMessage `json:"relationships"`
				}
				if err := json.Unmarshal(data, &m); err != nil {
					t.Fatal(err)
				}
				if len(m.Relationships) != 1 {
					t.Fatalf("after sync %d the model has %d relationships, want 1:\n%s", i, len(m.Relationships), data)
				}
			}
			if out := runCLI(t, bin, dir, "sync"); !strings.Contains(out, "Already in sync") {
				t.Errorf("the fourth sync is not idempotent:\n%s", out)
			}
		})
	}
}

package e2e

// #709: diff --view filters by the elements of that view, an unknown view fails,
// and a relationship without a label is printed without "()".

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const diffViewModelFile = `{"specification":{"elements":{"actor":{"notation":"A"},"db":{"notation":"D"}},"relationships":{"uses":{"notation":"uses"}}},
 "model":{"u":{"kind":"actor","title":"U"}},
 "asIs":{"elements":{"u":{"kind":"actor","title":"U"},"old":{"kind":"db","title":"Old"}},"relationships":[{"from":"u","to":"old","kind":"uses"}]},
 "toBe":{"elements":{"u":{"kind":"actor","title":"U"},"new":{"kind":"db","title":"New"}},"relationships":[{"from":"u","to":"new","kind":"uses"}]},
 "views":{"v":{"title":"V","include":["u"]}}}`

func TestDiff_ViewFilterAndUnknownView(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "m.jsonc"), []byte(diffViewModelFile), 0o600); err != nil {
		t.Fatal(err)
	}

	full := runCLI(t, bin, dir, "--model", "m.jsonc", "diff")
	if !strings.Contains(full, "u → new") || strings.Contains(full, "()") {
		t.Errorf("the full diff names the added relationship without ():\n%s", full)
	}
	view := runCLI(t, bin, dir, "--model", "m.jsonc", "diff", "--view", "v")
	if strings.Contains(view, "new") || strings.Contains(view, "old") {
		t.Errorf("view v does not show new and old:\n%s", view)
	}
	out, code := runCLIAllowFail(t, bin, dir, "--model", "m.jsonc", "diff", "--view", "nope")
	if code != 1 || !strings.Contains(out, `"nope"`) {
		t.Errorf("an unknown view: want exit 1 naming it, got %d:\n%s", code, out)
	}
}

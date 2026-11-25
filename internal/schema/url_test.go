package schema_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// schemaURLPattern matches links into this repository's main branch that point
// at a *.schema.json file, e.g.
// https://raw.githubusercontent.com/docToolchain/Bausteinsicht/main/schemas/bausteinsicht.schema.json
// and captures the repo-relative path.
var schemaURLPattern = regexp.MustCompile(
	`https://(?:raw\.githubusercontent\.com/docToolchain/Bausteinsicht/main|github\.com/docToolchain/Bausteinsicht/raw/main)/([\w./-]+\.schema\.json)`)

// TestSchemaURLsResolveToExistingFile guards against the issue #638 regression
// (and #414 before it): a "$schema" URL such as ".../main/schema/..." (singular)
// 404s because the file lives under "schemas/". Every such link in docs, skills,
// examples and models must point at a file that exists in the repository.
func TestSchemaURLsResolveToExistingFile(t *testing.T) {
	root := repoRoot(t)
	checked := 0

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules":
				return filepath.SkipDir
			}
			// Local git worktrees of other branches are not part of this checkout.
			if rel, _ := filepath.Rel(root, path); filepath.ToSlash(rel) == ".claude/worktrees" {
				return filepath.SkipDir
			}
			return nil
		}
		switch filepath.Ext(path) {
		case ".md", ".adoc", ".jsonc", ".txt":
		default:
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		for _, m := range schemaURLPattern.FindAllStringSubmatch(string(data), -1) {
			checked++
			if _, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(m[1]))); statErr != nil {
				t.Errorf("%s: schema URL %q points at %q, which does not exist in the repository (the schema lives under schemas/)",
					filepath.ToSlash(rel), strings.TrimSpace(m[0]), m[1])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repository: %v", err)
	}
	if checked == 0 {
		t.Fatal("no schema URLs found; the pattern or the file filter is broken")
	}
}

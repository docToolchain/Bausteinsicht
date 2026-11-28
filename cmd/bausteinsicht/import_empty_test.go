package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Importing a file with the wrong --from (or a text file without any model) wrote
// an empty model, printed "Next steps" and exited with 0; with --force it
// overwrote the existing architecture.jsonc (#705).
func TestImportCmd_NoElementsFound_ExitCode1(t *testing.T) {
	dir := t.TempDir()
	garbage := filepath.Join(dir, "g.txt")
	if err := os.WriteFile(garbage, []byte("garbage text\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	xmi := absDSL(t, "internal", "importer", "xmi", "testdata", "basic.xmi")

	for name, args := range map[string][]string{
		"garbage as structurizr": {"--from", "structurizr", garbage},
		"xmi file as likec4":     {"--from", "likec4", xmi},
		"garbage, dry run":       {"--from", "structurizr", "--dry-run", garbage},
	} {
		t.Run(name, func(t *testing.T) {
			out := filepath.Join(dir, strings.ReplaceAll(name, " ", "_")+".jsonc")
			existing := `{"model":{"keep":{"kind":"system","title":"Keep"}}}`
			if err := os.WriteFile(out, []byte(existing), 0o600); err != nil {
				t.Fatal(err)
			}
			text, err := executeImportCmd(append([]string{"import", "--output", out, "--force"}, args...)...)
			if err == nil || exitCodeOf(t, err) != 1 || !strings.Contains(err.Error(), "no elements found") {
				t.Fatalf("want exit code 1 with 'no elements found', got %v\n%s", err, text)
			}
			if strings.Contains(text, "Next steps") {
				t.Errorf("no next-steps hint for an empty result:\n%s", text)
			}
			if data, _ := os.ReadFile(out); string(data) != existing {
				t.Errorf("the existing model must not be overwritten, got:\n%s", data)
			}
		})
	}
}

// --generate-template writes template.drawio next to the model, refuses to
// replace one that exists and writes nothing with --dry-run (#688).
func TestImportCmd_GenerateTemplate(t *testing.T) {
	dsl := absDSL(t, "internal", "importer", "structurizr", "testdata", "simple.dsl")

	t.Run("writes the template next to the model", func(t *testing.T) {
		dir := t.TempDir()
		out := filepath.Join(dir, "m.jsonc")
		if text, err := executeImportCmd("import", "--from", "structurizr", "--generate-template", "--output", out, dsl); err != nil {
			t.Fatalf("import failed: %v\n%s", err, text)
		}
		data, err := os.ReadFile(filepath.Join(dir, "template.drawio"))
		if err != nil || !strings.Contains(string(data), "<mxfile") {
			t.Fatalf("want a draw.io template next to the model, got %v / %q", err, data)
		}
	})

	t.Run("keeps an existing template and writes nothing", func(t *testing.T) {
		dir := t.TempDir()
		mine := filepath.Join(dir, "template.drawio")
		if err := os.WriteFile(mine, []byte("mine"), 0o600); err != nil {
			t.Fatal(err)
		}
		out := filepath.Join(dir, "m.jsonc")
		_, err := executeImportCmd("import", "--from", "structurizr", "--generate-template", "--output", out, dsl)
		if err == nil || exitCodeOf(t, err) != 2 {
			t.Fatalf("want exit code 2, got %v", err)
		}
		if got, _ := os.ReadFile(mine); string(got) != "mine" {
			t.Errorf("the template was overwritten: %q", got)
		}
		if _, err := os.Stat(out); err == nil {
			t.Error("the model must not be written when the import is refused")
		}
	})

	t.Run("dry run writes no template", func(t *testing.T) {
		dir := t.TempDir()
		out := filepath.Join(dir, "m.jsonc")
		if _, err := executeImportCmd("import", "--from", "structurizr", "--generate-template", "--dry-run", "--output", out, dsl); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(dir, "template.drawio")); err == nil {
			t.Error("--dry-run must not write the template")
		}
	})
}

// A template.drawio that is a dangling symbolic link looks absent to a plain
// existence check, but creating the file would write through the link. The
// template is created exclusively, so the import is refused and neither the
// link target nor the model is written (CodeRabbit finding on #775).
func TestImportCmd_GenerateTemplateRefusesADanglingLink(t *testing.T) {
	dsl := absDSL(t, "internal", "importer", "structurizr", "testdata", "simple.dsl")
	dir := t.TempDir()
	target := filepath.Join(dir, "elsewhere.drawio")
	if err := os.Symlink(target, filepath.Join(dir, "template.drawio")); err != nil {
		t.Skipf("symbolic links are not available: %v", err)
	}
	out := filepath.Join(dir, "m.jsonc")
	_, err := executeImportCmd("import", "--from", "structurizr", "--generate-template", "--output", out, dsl)
	if err == nil || exitCodeOf(t, err) != 2 {
		t.Fatalf("want exit code 2, got %v", err)
	}
	if _, err := os.Stat(target); err == nil {
		t.Error("the import wrote through the dangling link")
	}
	if _, err := os.Stat(out); err == nil {
		t.Error("the model must not be written when the template is refused")
	}
}

package e2e

// #688: an imported Structurizr workspace keeps its view keys and the tags
// that decide the kind, and `import --generate-template` writes a template
// with a style for every imported kind, so the first sync has nothing to
// complain about.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const structurizrKeysAndTagsDSL = `workspace "Big Bank" "desc" {
  model {
    u = person "User" "A user"
    ss = softwareSystem "Banking" "Core" {
      web = container "Web" "SPA" "React"
      db = container "DB" "store" "Postgres" "Database"
    }
    ext = softwareSystem "Mail" "ext" "External"
    u -> web "Uses" "HTTPS"
    web -> db "Reads"
    web -> ext "Sends"
  }
  views {
    systemContext ss "ctx" { include * }
    container ss "cont" { include * }
  }
}
`

func TestStructurizrImport_KeepsViewKeysAndTagKinds(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "w.dsl"), []byte(structurizrKeysAndTagsDSL), 0o600); err != nil {
		t.Fatal(err)
	}

	runCLI(t, bin, dir, "import", "w.dsl", "--from", "structurizr", "--generate-template")

	data, err := os.ReadFile(filepath.Join(dir, "architecture.jsonc"))
	if err != nil {
		t.Fatal(err)
	}
	model := string(data)
	for _, want := range []string{`"ctx": {`, `"cont": {`, `"external_system"`, `"datastore"`} {
		if !strings.Contains(model, want) {
			t.Errorf("imported model lacks %s:\n%s", want, model)
		}
	}
	if out, code := runCLIAllowFail(t, bin, dir, "validate"); code != 0 {
		t.Fatalf("the imported model does not validate (exit %d):\n%s", code, out)
	}

	if _, err := os.Stat(filepath.Join(dir, "template.drawio")); err != nil {
		t.Fatalf("--generate-template must write template.drawio next to the model: %v", err)
	}
	out := runCLI(t, bin, dir, "sync")
	if strings.Contains(out, "no template style") {
		t.Errorf("sync still reports kinds without a style after --generate-template:\n%s", out)
	}
}

func TestStructurizrImport_GenerateTemplateKeepsAnExistingTemplate(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "w.dsl"), []byte(structurizrKeysAndTagsDSL), 0o600); err != nil {
		t.Fatal(err)
	}
	mine := filepath.Join(dir, "template.drawio")
	if err := os.WriteFile(mine, []byte("my template"), 0o600); err != nil {
		t.Fatal(err)
	}

	out, code := runCLIAllowFail(t, bin, dir, "import", "w.dsl", "--from", "structurizr", "--generate-template")
	if code != 2 {
		t.Fatalf("want exit 2 for an existing template.drawio, got %d:\n%s", code, out)
	}
	if got, _ := os.ReadFile(mine); string(got) != "my template" {
		t.Errorf("the existing template was overwritten: %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "architecture.jsonc")); err == nil {
		t.Error("nothing may be written when the import is refused")
	}
}

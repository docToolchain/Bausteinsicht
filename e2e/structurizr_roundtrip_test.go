package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const roundtripDSL = `workspace "Big Bank" "desc" {
  model {
    u = person "User" "A user"
    ss = softwareSystem "Banking" "Core" {
      web = container "Web" "SPA" "React"
      api = container "API" "REST" "Java" {
        ctrl = component "Controller" "ctl" "Spring"
      }
      db = container "DB" "store" "Postgres" "Database"
    }
    ext = softwareSystem "Mail" "ext" "External"
    u -> web "Uses" "HTTPS"
    web -> api "Calls" "JSON"
    api -> db "Reads"
    api -> ext "Sends \"mail\""
    ctrl -> db "x"
  }
  views {
    systemContext ss "ctx" { include * autolayout lr }
    container ss "cont" { include * }
    component api "comp" { include * }
    styles { element "Database" { shape cylinder } }
  }
}`

// TestStructurizrRoundtrip_ArgumentOrderAndViewTypes (#687): import -> export ->
// import keeps description and technology, and the three view types survive.
func TestStructurizrRoundtrip_ArgumentOrderAndViewTypes(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "w.dsl"), roundtripDSL)
	runCLI(t, bin, dir, "import", "w.dsl", "--from", "structurizr", "--output", "m.jsonc")

	out := filepath.Join(dir, "out")
	runCLI(t, bin, dir, "--model", "m.jsonc", "export-diagram", "--diagram-format", "structurizr", "--output", out)
	data, err := os.ReadFile(filepath.Join(out, "workspace.dsl"))
	if err != nil {
		t.Fatal(err)
	}
	dsl := string(data)
	for _, want := range []string{
		`api = container "API" "REST" "Java"`,
		`ctrl = component "Controller" "ctl" "Spring"`,
		`systemContext ss "ctx" "ctx"`,
		`container ss "cont" "cont"`,
		`component api "comp" "comp"`,
	} {
		if !strings.Contains(dsl, want) {
			t.Errorf("exported DSL lacks %q:\n%s", want, dsl)
		}
	}

	runCLI(t, bin, dir, "import", filepath.Join("out", "workspace.dsl"), "--from", "structurizr", "--output", "m2.jsonc")
	for _, f := range []string{"m.jsonc", "m2.jsonc"} {
		b, _ := os.ReadFile(filepath.Join(dir, f))
		for _, want := range []string{`"description": "REST"`, `"technology": "Java"`, `"title": "cont"`, `"kind": "datastore"`, `"kind": "external_system"`} {
			if !strings.Contains(string(b), want) {
				t.Errorf("%s lacks %s", f, want)
			}
		}
	}
}

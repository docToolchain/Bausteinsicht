package structurizr_test

import (
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/exporter/structurizr"
	importer "github.com/docToolchain/Bausteinsicht/internal/importer/structurizr"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

const bankDSL = `workspace "Big Bank" "desc" {
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

func exportBank(t *testing.T) (string, *model.BausteinsichtModel) {
	t.Helper()
	r, err := importer.ImportSource(bankDSL)
	if err != nil {
		t.Fatal(err)
	}
	return structurizr.Export(r.Model), r.Model
}

// Structurizr takes `container <name> <description> <technology>` and
// `component <name> <description> <technology>`; the exporter wrote the
// technology before the description (#687).
func TestExport_DescriptionComesBeforeTechnology(t *testing.T) {
	out, _ := exportBank(t)
	for _, want := range []string{
		`api = container "API" "REST" "Java"`,
		`ctrl = component "Controller" "ctl" "Spring"`,
		`web = container "Web" "SPA" "React"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

// Without a description the technology still has to be the third argument.
func TestExport_TechnologyWithoutDescriptionKeepsItsSlot(t *testing.T) {
	m := &model.BausteinsichtModel{Model: map[string]model.Element{
		"s": {Kind: "system", Title: "S", Children: map[string]model.Element{
			"c": {Kind: "container", Title: "C", Technology: "Go"},
		}},
	}}
	out := structurizr.Export(m)
	if !strings.Contains(out, `c = container "C" "" "Go"`) {
		t.Errorf("technology is not in the third slot:\n%s", out)
	}
}

// person and softwareSystem have no technology argument; a third string would
// be read as a tag.
func TestExport_SystemAndPersonTakeNoTechnology(t *testing.T) {
	m := &model.BausteinsichtModel{Model: map[string]model.Element{
		"u": {Kind: "actor", Title: "U", Description: "d", Technology: "Browser"},
		"s": {Kind: "system", Title: "S", Description: "d", Technology: "Java"},
	}}
	out := structurizr.Export(m)
	for _, want := range []string{`u = person "U" "d"`, `s = softwareSystem "S" "d"`} {
		if !strings.Contains(out, want+"\n") {
			t.Errorf("want %q followed by a line end in:\n%s", want, out)
		}
	}
}

// A system-scoped view that shows no container is a context view, one that
// shows the system's containers is a container view (#687).
func TestExport_ContextViewIsNotExportedAsContainerView(t *testing.T) {
	out, m := exportBank(t)
	for key, v := range m.Views {
		t.Logf("view %s: scope=%q include=%v", key, v.Scope, v.Include)
	}
	for _, want := range []string{`systemContext ss "ctx" "ctx"`, `container ss "cont" "cont"`, `component api "comp" "comp"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

// The second argument of a view is its key, the third its description, and the
// export writes them in that order.
func TestExport_ViewKeyThenDescription(t *testing.T) {
	m := buildTestModel()
	m.Views = map[string]model.View{"landscape": {Title: "All systems", Include: []string{"*"}}}
	out := structurizr.Export(m)
	if !strings.Contains(out, `systemLandscape "landscape" "All systems"`) {
		t.Errorf("landscape view not written as key then description:\n%s", out)
	}
}

// import -> export -> import keeps descriptions, technologies, view keys and
// titles.
func TestRoundtrip_KeepsDescriptionTechnologyAndViews(t *testing.T) {
	out, m1 := exportBank(t)
	r2, err := importer.ImportSource(out)
	if err != nil {
		t.Fatalf("re-import failed: %v\n%s", err, out)
	}
	flat1, _ := model.FlattenElements(m1)
	flat2, _ := model.FlattenElements(r2.Model)
	for id, e1 := range flat1 {
		e2 := flat2[id]
		if e2 == nil {
			t.Errorf("element %s lost\n%s", id, out)
			continue
		}
		if e1.Description != e2.Description || e1.Technology != e2.Technology || e1.Title != e2.Title {
			t.Errorf("element %s changed: %+v -> %+v\n%s", id, *e1, *e2, out)
		}
	}
	for key, v1 := range m1.Views {
		v2, ok := r2.Model.Views[key]
		if !ok {
			t.Errorf("view %s lost; views: %v\n%s", key, r2.Model.Views, out)
			continue
		}
		if v1.Title != v2.Title || v1.Scope != v2.Scope {
			t.Errorf("view %s changed: %+v -> %+v\n%s", key, v1, v2, out)
		}
	}
}

// The kinds external_system and datastore and the tags of an element survive
// export and import again: the exporter writes the tags argument, with the tag
// that selects the kind (CodeRabbit finding on #775).
func TestExport_KindsAndTagsSurviveReimport(t *testing.T) {
	m := &model.BausteinsichtModel{Model: map[string]model.Element{
		"ext": {Kind: "external_system", Title: "Mail", Description: "d", Tags: []string{"Vendor"}},
		"s": {Kind: "system", Title: "S", Children: map[string]model.Element{
			"db": {Kind: "datastore", Title: "DB", Technology: "Postgres", Tags: []string{"Critical", "Internal"}},
			"c":  {Kind: "container", Title: "C", Tags: []string{"Edge"}},
		}},
	}}
	out := structurizr.Export(m)
	for _, want := range []string{
		`ext = softwareSystem "Mail" "d" "External,Vendor"`,
		`db = container "DB" "" "Postgres" "Database,Critical,Internal"`,
		`c = container "C" "" "" "Edge"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("want %q in:\n%s", want, out)
		}
	}
	back, err := importer.ImportSource(out)
	if err != nil {
		t.Fatal(err)
	}
	if got := back.Model.Model["ext"]; got.Kind != "external_system" || len(got.Tags) != 1 || got.Tags[0] != "Vendor" {
		t.Errorf("ext after re-import: kind %q tags %v", got.Kind, got.Tags)
	}
	db := back.Model.Model["s"].Children["db"]
	if db.Kind != "datastore" || strings.Join(db.Tags, ",") != "Critical,Internal" {
		t.Errorf("db after re-import: kind %q tags %v", db.Kind, db.Tags)
	}
}

package structurizr_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/exporter/structurizr"
	importer "github.com/docToolchain/Bausteinsicht/internal/importer/structurizr"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func buildTestModel() *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"actor":     {Notation: "Actor"},
				"system":    {Notation: "System", Container: true},
				"container": {Notation: "Container"},
			},
		},
		Model: map[string]model.Element{
			"user": {Kind: "actor", Title: "User", Description: "A human user"},
			"orderSystem": {
				Kind:        "system",
				Title:       "Order System",
				Description: "Handles orders",
				Children: map[string]model.Element{
					"webApp": {Kind: "container", Title: "Web App", Technology: "TypeScript", Description: "Frontend"},
					"api":    {Kind: "container", Title: "API", Technology: "Go", Description: "Backend"},
				},
			},
			"paymentGateway": {Kind: "system", Title: "Payment Gateway", Description: "External payment"},
		},
		Relationships: []model.Relationship{
			{From: "user", To: "orderSystem.webApp", Label: "Uses"},
			{From: "orderSystem.webApp", To: "orderSystem.api", Label: "Calls"},
			{From: "orderSystem.api", To: "paymentGateway", Label: "Charges via"},
		},
		Views: map[string]model.View{
			"context": {Title: "System Context", Scope: "orderSystem", Include: []string{"*"}},
			"containers": {Title: "Containers", Scope: "orderSystem", Include: []string{"*"}},
		},
	}
}

func TestExport_ContainsWorkspaceBlock(t *testing.T) {
	m := buildTestModel()
	out := structurizr.Export(m)
	if !strings.HasPrefix(out, "workspace {") {
		t.Errorf("expected output to start with 'workspace {', got:\n%s", out)
	}
	if !strings.HasSuffix(strings.TrimRight(out, "\n"), "}") {
		t.Errorf("expected output to end with '}', got:\n%s", out)
	}
}

func TestExport_ElementKinds(t *testing.T) {
	m := buildTestModel()
	out := structurizr.Export(m)

	cases := []struct {
		want string
	}{
		{"user = person"},
		{"orderSystem = softwareSystem"},
		{"paymentGateway = softwareSystem"},
	}
	for _, c := range cases {
		if !strings.Contains(out, c.want) {
			t.Errorf("expected %q in output:\n%s", c.want, out)
		}
	}
}

func TestExport_NestedChildren(t *testing.T) {
	m := buildTestModel()
	out := structurizr.Export(m)

	// Children use leaf keys when globally unique.
	if !strings.Contains(out, "webApp = container") {
		t.Errorf("expected nested container variable 'webApp', got:\n%s", out)
	}
	if !strings.Contains(out, "api = container") {
		t.Errorf("expected nested container variable 'api', got:\n%s", out)
	}
}

func TestExport_Relationships(t *testing.T) {
	m := buildTestModel()
	out := structurizr.Export(m)

	// Leaf keys are used when globally unique.
	if !strings.Contains(out, `user -> webApp "Uses"`) {
		t.Errorf("expected relationship 'user -> webApp', got:\n%s", out)
	}
	if !strings.Contains(out, `api -> paymentGateway "Charges via"`) {
		t.Errorf("expected relationship 'api -> paymentGateway', got:\n%s", out)
	}
}

func TestExport_AmbiguousLeafKey_UsesDotPath(t *testing.T) {
	// Two elements with the same leaf key "app" in different parents.
	m := &model.BausteinsichtModel{
		Specification: model.Specification{Elements: map[string]model.ElementKind{}},
		Model: map[string]model.Element{
			"systemA": {Kind: "system", Title: "System A", Children: map[string]model.Element{
				"app": {Kind: "container", Title: "App A"},
			}},
			"systemB": {Kind: "system", Title: "System B", Children: map[string]model.Element{
				"app": {Kind: "container", Title: "App B"},
			}},
		},
		Relationships: []model.Relationship{
			{From: "systemA.app", To: "systemB.app", Label: "Calls"},
		},
		Views: map[string]model.View{},
	}
	out := structurizr.Export(m)
	// Both "app" leaves are ambiguous → fall back to dot-path variable names.
	if !strings.Contains(out, "systemA_app = container") {
		t.Errorf("expected dot-path variable 'systemA_app' for ambiguous leaf, got:\n%s", out)
	}
	if !strings.Contains(out, "systemB_app = container") {
		t.Errorf("expected dot-path variable 'systemB_app' for ambiguous leaf, got:\n%s", out)
	}
	if !strings.Contains(out, `systemA_app -> systemB_app "Calls"`) {
		t.Errorf("expected relationship with dot-path vars, got:\n%s", out)
	}
}

func TestExport_Views(t *testing.T) {
	m := buildTestModel()
	out := structurizr.Export(m)

	if !strings.Contains(out, "views {") {
		t.Errorf("expected 'views {' block")
	}
	// containers view has container-kind children → container view type
	if !strings.Contains(out, "container orderSystem") {
		t.Errorf("expected 'container orderSystem' view, got:\n%s", out)
	}
}

func TestExport_NoViews(t *testing.T) {
	m := buildTestModel()
	m.Views = nil
	out := structurizr.Export(m)
	// views block should be empty
	if !strings.Contains(out, "views {\n    }") {
		t.Errorf("expected empty views block, got:\n%s", out)
	}
}

func TestExport_NoRelationships(t *testing.T) {
	m := buildTestModel()
	m.Relationships = nil
	out := structurizr.Export(m)
	// should not contain -> outside comments
	modelSection := extractSection(out, "model {", "    }")
	if strings.Contains(modelSection, "->") {
		t.Errorf("unexpected relationship in model with no relationships:\n%s", modelSection)
	}
}

func TestExport_KindMapping(t *testing.T) {
	cases := []struct {
		kind     string
		wantType string
	}{
		{"actor", "person"},
		{"person", "person"},
		{"system", "softwareSystem"},
		{"external_system", "softwareSystem"},
		{"container", "container"},
		{"ui", "container"},
		{"mobile", "container"},
		{"datastore", "container"},
		{"queue", "container"},
		{"component", "component"},
		{"unknown_kind", "softwareSystem"},
	}

	for _, c := range cases {
		m := &model.BausteinsichtModel{
			Specification: model.Specification{Elements: map[string]model.ElementKind{}},
			Model: map[string]model.Element{
				"elem": {Kind: c.kind, Title: "Test"},
			},
			Relationships: []model.Relationship{},
			Views:         map[string]model.View{},
		}
		out := structurizr.Export(m)
		want := "elem = " + c.wantType
		if !strings.Contains(out, want) {
			t.Errorf("kind %q: expected %q, got:\n%s", c.kind, want, out)
		}
	}
}

func TestExport_LandscapeView(t *testing.T) {
	m := buildTestModel()
	m.Views = map[string]model.View{
		"overview": {Title: "Overview", Include: []string{"*"}},
	}
	out := structurizr.Export(m)
	if !strings.Contains(out, "landscape") {
		t.Errorf("expected 'landscape' view type for scope-less view, got:\n%s", out)
	}
}

func TestExport_TechnologyAndDescription(t *testing.T) {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{Elements: map[string]model.ElementKind{}},
		Model: map[string]model.Element{
			"svc": {Kind: "container", Title: "Service", Technology: "Go", Description: "Does stuff"},
		},
		Relationships: []model.Relationship{},
		Views:         map[string]model.View{},
	}
	out := structurizr.Export(m)
	if !strings.Contains(out, `svc = container "Service" "Go" "Does stuff"`) {
		t.Errorf("expected technology and description in output, got:\n%s", out)
	}
}

// extractSection extracts text between start and end markers (inclusive first occurrence).
func extractSection(s, start, end string) string {
	si := strings.Index(s, start)
	if si < 0 {
		return ""
	}
	ei := strings.Index(s[si:], end)
	if ei < 0 {
		return s[si:]
	}
	return s[si : si+ei+len(end)]
}

// TestRoundtrip_ImportExportReImport imports simple.dsl, exports to DSL,
// re-imports, and verifies structural equivalence.
func TestRoundtrip_ImportExportReImport(t *testing.T) {
	// Phase 1: import original DSL.
	r1, err := importer.Import(filepath.Join("..", "..", "importer", "structurizr", "testdata", "simple.dsl"))
	if err != nil {
		t.Fatalf("initial import failed: %v", err)
	}
	m1 := r1.Model

	// Phase 2: export to Structurizr DSL.
	dsl := structurizr.Export(m1)
	if !strings.Contains(dsl, "workspace {") {
		t.Fatalf("exported DSL missing 'workspace {': %s", dsl)
	}

	// Phase 3: re-import the exported DSL.
	r2, err := importer.ImportSource(dsl)
	if err != nil {
		t.Fatalf("re-import of exported DSL failed: %v\nDSL:\n%s", err, dsl)
	}
	m2 := r2.Model

	// Compare element count (flat).
	flat1, _ := model.FlattenElements(m1)
	flat2, _ := model.FlattenElements(m2)
	if len(flat1) != len(flat2) {
		t.Errorf("element count mismatch: original=%d re-imported=%d\nDSL:\n%s", len(flat1), len(flat2), dsl)
	}

	// Every element in m1 must exist in m2 (by path).
	for id := range flat1 {
		if flat2[id] == nil {
			t.Errorf("element %q present in original but missing after roundtrip\nDSL:\n%s", id, dsl)
		}
	}

	// Relationship count must be preserved.
	if len(m1.Relationships) != len(m2.Relationships) {
		t.Errorf("relationship count mismatch: original=%d re-imported=%d\nDSL:\n%s",
			len(m1.Relationships), len(m2.Relationships), dsl)
	}

	// Every from→to pair in m1 must exist in m2.
	type relKey struct{ from, to string }
	rels2 := make(map[relKey]bool, len(m2.Relationships))
	for _, r := range m2.Relationships {
		rels2[relKey{r.From, r.To}] = true
	}
	for _, r := range m1.Relationships {
		if !rels2[relKey{r.From, r.To}] {
			t.Errorf("relationship %q → %q missing after roundtrip\nDSL:\n%s", r.From, r.To, dsl)
		}
	}

	// View count must be preserved.
	if len(m1.Views) != len(m2.Views) {
		t.Errorf("view count mismatch: original=%d re-imported=%d\nDSL:\n%s",
			len(m1.Views), len(m2.Views), dsl)
	}
}

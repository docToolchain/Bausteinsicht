package structurizr_test

import (
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/exporter/structurizr"
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

	// Children should use dot-path variable names.
	if !strings.Contains(out, "orderSystem_webApp = container") {
		t.Errorf("expected nested container variable 'orderSystem_webApp', got:\n%s", out)
	}
	if !strings.Contains(out, "orderSystem_api = container") {
		t.Errorf("expected nested container variable 'orderSystem_api', got:\n%s", out)
	}
}

func TestExport_Relationships(t *testing.T) {
	m := buildTestModel()
	out := structurizr.Export(m)

	if !strings.Contains(out, `user -> orderSystem_webApp "Uses"`) {
		t.Errorf("expected relationship 'user -> orderSystem_webApp', got:\n%s", out)
	}
	if !strings.Contains(out, `orderSystem_api -> paymentGateway "Charges via"`) {
		t.Errorf("expected relationship 'orderSystem_api -> paymentGateway', got:\n%s", out)
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

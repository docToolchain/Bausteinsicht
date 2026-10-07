package diagram

import (
	"reflect"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func c4KindModel() *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"person": {Notation: "Person"},
				"system": {Notation: "Software System"},
				"widget": {Notation: "Widget"},
			},
		},
		Model: map[string]model.Element{
			"customer": {Kind: "person", Title: "Customer"},
			"shop":     {Kind: "system", Title: "Shop"},
			"gizmo":    {Kind: "widget", Title: "Gizmo"},
		},
		Relationships: []model.Relationship{
			{From: "customer", To: "shop", Label: "places order"},
		},
		Views: map[string]model.View{
			"context": {Title: "Context", Include: []string{"*"}},
		},
	}
}

// TestC4Macro_Resolution (#633): override > kind key > notation > System fallback.
func TestC4Macro_Resolution(t *testing.T) {
	spec := &model.Specification{Elements: map[string]model.ElementKind{
		"person":       {Notation: "Person"},
		"stakeholder":  {Notation: "Actor"},
		"vendor":       {Notation: "External System"},
		"store":        {Notation: "Database"},
		"jobs":         {Notation: "Queue"},
		"human":        {Notation: "Widget", C4: "Person"},
		"actor":        {Notation: "Widget", C4: "System_Ext"},
		"widget":       {Notation: "Widget"},
		"datastore":    {Notation: "Whatever"},
		"mixed-case":   {Notation: "  software SYSTEM "},
		"external_sys": {Notation: "external system"},
		"Actor":        {Notation: "Whatever"},
	}}
	cases := []struct {
		kind      string
		wantMacro string
		wantOK    bool
	}{
		{"person", "Person", true},
		{"stakeholder", "Person", true},
		{"vendor", "System_Ext", true},
		{"store", "ContainerDb", true},
		{"jobs", "ContainerQueue", true},
		{"human", "Person", true},
		{"actor", "System_Ext", true},
		{"datastore", "ContainerDb", true},
		{"mixed-case", "System", true},
		{"external_sys", "System_Ext", true},
		{"Actor", "Person", true},
		{"widget", "System", false},
		{"undefined-kind", "System", false},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			macro, ok := C4Macro(spec, tc.kind)
			if macro != tc.wantMacro || ok != tc.wantOK {
				t.Errorf("C4Macro(%q) = (%q, %v), want (%q, %v)", tc.kind, macro, ok, tc.wantMacro, tc.wantOK)
			}
		})
	}
}

func TestPlantUML_PersonKindRendersAsPerson(t *testing.T) {
	out, err := FormatView(c4KindModel(), "context", PlantUML)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Person(customer,") {
		t.Errorf("expected Person(customer, ...), got:\n%s", out)
	}
	if strings.Contains(out, "System(customer,") {
		t.Errorf("person kind must not render as System, got:\n%s", out)
	}
}

func TestMermaid_PersonKindRendersAsPerson(t *testing.T) {
	out, err := FormatView(c4KindModel(), "context", Mermaid)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Person(customer,") {
		t.Errorf("expected Person(customer, ...), got:\n%s", out)
	}
}

func TestPlantUML_C4OverrideWins(t *testing.T) {
	m := c4KindModel()
	m.Specification.Elements["widget"] = model.ElementKind{Notation: "Widget", C4: "ContainerDb"}
	out, err := FormatView(m, "context", PlantUML)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "ContainerDb(gizmo,") {
		t.Errorf("expected ContainerDb(gizmo, ...), got:\n%s", out)
	}
}

func TestUnmappedKinds(t *testing.T) {
	got, err := UnmappedKinds(c4KindModel(), "context")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"widget"}; !reflect.DeepEqual(got, want) {
		t.Errorf("UnmappedKinds = %v, want %v", got, want)
	}

	m := c4KindModel()
	m.Specification.Elements["widget"] = model.ElementKind{Notation: "Widget", C4: "Component"}
	got, err = UnmappedKinds(m, "context")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected no unmapped kinds with c4 override, got %v", got)
	}

	if _, err := UnmappedKinds(m, "nope"); err == nil {
		t.Error("expected error for unknown view")
	}
}

func scopedC4Model(scopeKind model.ElementKind) *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"svc":   scopeKind,
				"actor": {Notation: "Actor"},
			},
		},
		Model: map[string]model.Element{
			"user": {Kind: "actor", Title: "User"},
			"app": {Kind: "svc", Title: "App", Children: map[string]model.Element{
				"api": {Kind: "svc", Title: "API"},
			}},
		},
		Views: map[string]model.View{
			"v": {Title: "V", Scope: "app", Include: []string{"user", "app.*"}},
		},
	}
}

// TestBoundaryMacro_FollowsC4Mapping (#633): the scope boundary uses the same
// kind resolution as elements (override / key / notation), not only the key "container".
func TestBoundaryMacro_FollowsC4Mapping(t *testing.T) {
	cases := []struct {
		name string
		kind model.ElementKind
		want string
	}{
		{"override", model.ElementKind{Notation: "Service", C4: "Container"}, "Container_Boundary(app,"},
		{"notation", model.ElementKind{Notation: "Container"}, "Container_Boundary(app,"},
		{"fallback", model.ElementKind{Notation: "Service"}, "System_Boundary(app,"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, f := range []Format{PlantUML, Mermaid} {
				out, err := FormatView(scopedC4Model(tc.kind), "v", f)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(out, tc.want) {
					t.Errorf("format %v: expected %s, got:\n%s", f, tc.want, out)
				}
			}
		})
	}
}

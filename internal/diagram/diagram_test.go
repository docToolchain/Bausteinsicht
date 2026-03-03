package diagram

import (
	"strings"
	"testing"

	"github.com/docToolchain/Bauteinsicht/internal/model"
)

func testModel() *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system":          {Notation: "Software System", Container: true},
				"container":       {Notation: "Container", Container: true},
				"component":       {Notation: "Component"},
				"actor":           {Notation: "Actor"},
				"external_system": {Notation: "External System"},
			},
		},
		Model: map[string]model.Element{
			"user": {Kind: "actor", Title: "User", Description: "End user"},
			"shop": {Kind: "system", Title: "Online Shop", Description: "E-commerce platform", Children: map[string]model.Element{
				"api": {Kind: "container", Title: "API", Description: "REST backend", Technology: "Go"},
				"db":  {Kind: "container", Title: "Database", Description: "Storage", Technology: "PostgreSQL"},
			}},
			"payment": {Kind: "external_system", Title: "Payment Gateway", Description: "External payment provider"},
		},
		Relationships: []model.Relationship{
			{From: "user", To: "shop", Label: "uses", Kind: "uses"},
			{From: "user", To: "payment", Label: "pays via", Kind: "uses"},
			{From: "shop.api", To: "shop.db", Label: "reads/writes", Kind: "uses"},
			{From: "shop.api", To: "payment", Label: "processes payment", Kind: "uses"},
		},
		Views: map[string]model.View{
			"context": {
				Title:   "System Context",
				Include: []string{"user", "shop", "payment"},
			},
			"containers": {
				Title:   "Container View",
				Scope:   "shop",
				Include: []string{"user", "shop.*", "payment"},
			},
		},
	}
}

// --- PlantUML Tests ---

func TestPlantUML_ContextView(t *testing.T) {
	m := testModel()
	result, err := FormatView(m, "context", PlantUML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "@startuml") {
		t.Error("expected @startuml")
	}
	if !strings.Contains(result, "@enduml") {
		t.Error("expected @enduml")
	}
	if !strings.Contains(result, "<C4/C4_Context>") {
		t.Error("expected <C4/C4_Context> stdlib include")
	}
	if !strings.Contains(result, "Person(") {
		t.Error("expected Person() macro for actor")
	}
	if !strings.Contains(result, "System(") {
		t.Error("expected System() macro")
	}
	if !strings.Contains(result, "System_Ext(") {
		t.Error("expected System_Ext() macro for external system")
	}
	if !strings.Contains(result, "Rel(") {
		t.Error("expected Rel() for relationships")
	}
}

func TestPlantUML_ContainerView(t *testing.T) {
	m := testModel()
	result, err := FormatView(m, "containers", PlantUML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "<C4/C4_Container>") {
		t.Error("expected <C4/C4_Container> stdlib include for container view")
	}
	if !strings.Contains(result, "System_Boundary(") {
		t.Error("expected System_Boundary for scope element")
	}
	if !strings.Contains(result, "Container(") {
		t.Error("expected Container() macro")
	}
}

func TestPlantUML_Relationships(t *testing.T) {
	m := testModel()
	result, err := FormatView(m, "context", PlantUML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "uses") {
		t.Error("expected relationship label 'uses'")
	}
	if !strings.Contains(result, "pays via") {
		t.Error("expected relationship label 'pays via'")
	}
}

func TestPlantUML_InvalidView(t *testing.T) {
	m := testModel()
	_, err := FormatView(m, "nonexistent", PlantUML)
	if err == nil {
		t.Error("expected error for nonexistent view")
	}
}

// --- Mermaid Tests ---

func TestMermaid_ContextView(t *testing.T) {
	m := testModel()
	result, err := FormatView(m, "context", Mermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "C4Context") {
		t.Error("expected C4Context diagram type")
	}
	if !strings.Contains(result, "Person(") {
		t.Error("expected Person() for actor")
	}
	if !strings.Contains(result, "System(") {
		t.Error("expected System()")
	}
	if !strings.Contains(result, "System_Ext(") {
		t.Error("expected System_Ext()")
	}
	if !strings.Contains(result, "Rel(") {
		t.Error("expected Rel()")
	}
}

func TestMermaid_ContainerView(t *testing.T) {
	m := testModel()
	result, err := FormatView(m, "containers", Mermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "C4Container") {
		t.Error("expected C4Container diagram type")
	}
	if !strings.Contains(result, "System_Boundary(") {
		t.Error("expected System_Boundary for scope")
	}
	if !strings.Contains(result, "Container(") {
		t.Error("expected Container()")
	}
}

func TestMermaid_Relationships(t *testing.T) {
	m := testModel()
	result, err := FormatView(m, "context", Mermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "uses") {
		t.Error("expected relationship label 'uses'")
	}
}

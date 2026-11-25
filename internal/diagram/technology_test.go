package diagram

import (
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// techModel is the example from #642: a relationship with a technology, one
// without, and a dashed one.
func techModel() *model.BausteinsichtModel {
	return &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{"system": {Notation: "Software System"}},
			Relationships: map[string]model.RelationshipKind{
				"uses":  {Notation: "uses"},
				"async": {Notation: "async", Dashed: true},
			},
		},
		Model: map[string]model.Element{
			"shop":    {Kind: "system", Title: "Shop"},
			"payment": {Kind: "system", Title: "Payment"},
			"mail":    {Kind: "system", Title: "Mail"},
		},
		Relationships: []model.Relationship{
			{From: "shop", To: "payment", Label: "charges", Technology: "HTTPS/JSON", Kind: "uses"},
			{From: "shop", To: "mail", Label: "notifies", Kind: "uses"},
			{From: "payment", To: "mail", Label: "receipts", Technology: "SMTP", Kind: "async"},
		},
		Views: map[string]model.View{
			"context":  {Title: "System Context", Include: []string{"*"}},
			"nolegend": {Title: "Without legend", Include: []string{"*"}, Legend: boolPtr(false)},
		},
	}
}

func boolPtr(b bool) *bool { return &b }

func TestPlantUML_RelationshipTechnology(t *testing.T) {
	out, err := FormatView(techModel(), "context", PlantUML)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`Rel(shop, payment, "charges", "HTTPS/JSON")`,
		`Rel(shop, mail, "notifies")`, // no technology: unchanged
		`payment ..> mail : "receipts [SMTP]"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

func TestMermaid_RelationshipTechnology(t *testing.T) {
	out, err := FormatView(techModel(), "context", Mermaid)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `Rel(shop, payment, "charges", "HTTPS/JSON")`) {
		t.Errorf("technology missing:\n%s", out)
	}
	if !strings.Contains(out, `Rel(shop, mail, "notifies")`) {
		t.Errorf("relationship without technology changed:\n%s", out)
	}
}

// The view title and a legend are part of every C4 PlantUML diagram (#642).
func TestPlantUML_TitleAndLegend(t *testing.T) {
	out, err := FormatView(techModel(), "context", PlantUML)
	if err != nil {
		t.Fatal(err)
	}
	want := "!include <C4/C4_Context>\ntitle System Context\nLAYOUT_WITH_LEGEND()\n"
	if !strings.Contains(out, want) {
		t.Errorf("expected the title and the legend right after the include:\n%s", out)
	}

	out, err = FormatView(techModel(), "nolegend", PlantUML)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "LAYOUT_WITH_LEGEND") {
		t.Errorf("legend must be left out with legend: false:\n%s", out)
	}
	if !strings.Contains(out, "title Without legend\n") {
		t.Errorf("title must stay:\n%s", out)
	}
}

func TestDotAndD2_RelationshipTechnology(t *testing.T) {
	m := techModel()
	dot, err := RenderDOT(m, "context")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dot, `label="charges [HTTPS/JSON]"`) {
		t.Errorf("DOT label lacks the technology:\n%s", dot)
	}
	d2, err := RenderD2(m, "context")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(d2, "charges [HTTPS/JSON]") {
		t.Errorf("D2 label lacks the technology:\n%s", d2)
	}
}

func TestHTML_RelationshipTechnology(t *testing.T) {
	out, err := RenderHTML(techModel(), "context")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "charges [HTTPS/JSON]") {
		t.Errorf("HTML edge label lacks the technology")
	}
}

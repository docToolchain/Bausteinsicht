package sync

import (
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/drawio"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// A kind that the template has no style for is reported once, with the number of
// elements and what to do, not once per element and page (#694).
func TestRun_UnstyledKindIsReportedOncePerKind(t *testing.T) {
	ts := minimalTemplates(t)
	m := &model.BausteinsichtModel{
		Model: map[string]model.Element{
			"a": {Kind: "service", Title: "A"},
			"b": {Kind: "service", Title: "B"},
			"c": {Kind: "service", Title: "C"},
			"d": {Kind: "database", Title: "D"},
		},
		Relationships: []model.Relationship{},
		Views: map[string]model.View{
			"v1": {Title: "1", Include: []string{"*"}},
			"v2": {Title: "2", Include: []string{"*"}},
		},
	}

	doc := drawio.NewDocument()
	doc.AddPage("view-v1", "1")
	doc.AddPage("view-v2", "2")
	result := Run(m, doc, emptyState(), ts, nil)

	var service, database []string
	for _, w := range result.Warnings {
		switch {
		case strings.Contains(w, "no template style for kind: service"):
			service = append(service, w)
		case strings.Contains(w, "no template style for kind: database"):
			database = append(database, w)
		}
	}
	if len(service) != 1 || len(database) != 1 {
		t.Fatalf("want one warning per kind, got service=%q database=%q", service, database)
	}
	for _, want := range []string{"3 elements", "generate-template", "--template"} {
		if !strings.Contains(service[0], want) {
			t.Errorf("the warning must contain %q: %s", want, service[0])
		}
	}
	if !strings.Contains(database[0], "1 element)") {
		t.Errorf("a single element is counted as such: %s", database[0])
	}
}

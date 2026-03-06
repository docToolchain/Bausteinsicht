package sync

import (
	"fmt"
	"testing"

	"github.com/docToolchain/Bauteinsicht/internal/drawio"
	"github.com/docToolchain/Bauteinsicht/internal/model"
)

func generateBenchModel(n int) *model.BausteinsichtModel {
	m := &model.BausteinsichtModel{
		Specification: model.Specification{
			Elements: map[string]model.ElementKind{
				"system":    {Notation: "System", Container: true},
				"container": {Notation: "Container", Container: true},
				"component": {Notation: "Component"},
			},
			Relationships: map[string]model.RelationshipKind{
				"uses": {Notation: "uses"},
			},
		},
		Model: make(map[string]model.Element),
		Views: map[string]model.View{
			"all": {Title: "All", Include: []string{"**"}},
		},
	}

	systemCount := max(1, n/10)
	containersPer := max(1, n/(systemCount*3))
	componentsPer := max(1, n/(systemCount*containersPer))

	created := 0
	for s := 0; s < systemCount && created < n; s++ {
		sysID := fmt.Sprintf("sys%d", s)
		sys := model.Element{
			Kind:     "system",
			Title:    fmt.Sprintf("System %d", s),
			Children: make(map[string]model.Element),
		}
		created++

		for c := 0; c < containersPer && created < n; c++ {
			contID := fmt.Sprintf("cont%d", c)
			cont := model.Element{
				Kind:     "container",
				Title:    fmt.Sprintf("Container %d-%d", s, c),
				Children: make(map[string]model.Element),
			}
			created++

			for p := 0; p < componentsPer && created < n; p++ {
				compID := fmt.Sprintf("comp%d", p)
				cont.Children[compID] = model.Element{
					Kind:        "component",
					Title:       fmt.Sprintf("Component %d-%d-%d", s, c, p),
					Technology:  "Go",
					Description: "A component",
				}
				created++
			}
			sys.Children[contID] = cont
		}
		m.Model[sysID] = sys
	}

	return m
}

func BenchmarkSyncRun(b *testing.B) {
	templateData := []byte(`<mxfile bausteinsicht_template_version="1"><diagram id="t" name="T"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/></root></mxGraphModel></diagram></mxfile>`)
	ts, err := drawio.LoadTemplateFromBytes(templateData)
	if err != nil {
		b.Fatal(err)
	}

	for _, size := range []int{10, 100, 500} {
		m := generateBenchModel(size)
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			for b.Loop() {
				doc := drawio.NewDocument()
				for viewID, view := range m.Views {
					doc.AddPage("view-"+viewID, view.Title)
				}
				emptyState := &SyncState{
					Elements:      make(map[string]ElementState),
					Relationships: []RelationshipState{},
				}
				_ = Run(m, doc, emptyState, ts, nil)
			}
		})
	}
}

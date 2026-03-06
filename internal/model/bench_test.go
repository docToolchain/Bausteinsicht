package model

import (
	"encoding/json"
	"fmt"
	"testing"
)

func generateModel(n int) *BausteinsichtModel {
	m := &BausteinsichtModel{
		Specification: Specification{
			Elements: map[string]ElementKind{
				"system":    {Notation: "System", Container: true},
				"container": {Notation: "Container", Container: true},
				"component": {Notation: "Component"},
			},
		},
		Model: make(map[string]Element),
	}

	// Create a realistic hierarchy: systems → containers → components.
	systemCount := max(1, n/10)
	containersPer := max(1, n/(systemCount*3))
	componentsPer := max(1, n/(systemCount*containersPer))

	created := 0
	for s := 0; s < systemCount && created < n; s++ {
		sysID := fmt.Sprintf("sys%d", s)
		sys := Element{
			Kind:     "system",
			Title:    fmt.Sprintf("System %d", s),
			Children: make(map[string]Element),
		}
		created++

		for c := 0; c < containersPer && created < n; c++ {
			contID := fmt.Sprintf("cont%d", c)
			cont := Element{
				Kind:     "container",
				Title:    fmt.Sprintf("Container %d-%d", s, c),
				Children: make(map[string]Element),
			}
			created++

			for p := 0; p < componentsPer && created < n; p++ {
				compID := fmt.Sprintf("comp%d", p)
				cont.Children[compID] = Element{
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

func BenchmarkFlattenElements(b *testing.B) {
	for _, size := range []int{10, 100, 500} {
		m := generateModel(size)
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			for b.Loop() {
				_, _ = FlattenElements(m)
			}
		})
	}
}

func BenchmarkLoadModel(b *testing.B) {
	for _, size := range []int{10, 100, 500} {
		m := generateModel(size)
		data, err := json.Marshal(m)
		if err != nil {
			b.Fatal(err)
		}
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			for b.Loop() {
				var out BausteinsichtModel
				_ = json.Unmarshal(data, &out)
			}
		})
	}
}

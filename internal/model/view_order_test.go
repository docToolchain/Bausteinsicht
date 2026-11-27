package model

import (
	"reflect"
	"testing"
)

// The order of the view pages must not depend on Go's map iteration (#683).
func TestOrderedViewKeys(t *testing.T) {
	m := &BausteinsichtModel{Views: map[string]View{
		"zeta":  {Title: "z", Scope: "a"},
		"inner": {Title: "i", Scope: "a"},
		"ctx":   {Title: "c"},
		"deep":  {Title: "d", Scope: "b"},
		"empty": {Title: "e"},
		"alpha": {Title: "a", Scope: "a"},
	}}
	want := []string{"ctx", "empty", "alpha", "deep", "inner", "zeta"}
	for i := 0; i < 50; i++ {
		if got := OrderedViewKeys(m); !reflect.DeepEqual(got, want) {
			t.Fatalf("run %d: got %v, want scope-less views first, then scoped ones, each sorted by key: %v", i, got, want)
		}
	}
	if got := OrderedViewKeys(&BausteinsichtModel{}); len(got) != 0 {
		t.Errorf("no views: got %v", got)
	}
}

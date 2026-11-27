package likec4_test

import (
	"reflect"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/importer/likec4"
)

const includeSrc = `
specification {
  element system
  element container
}
model {
  p = system "Platform" {
    api = container "API"
    db = container "DB"
  }
  u = system "User"
}
views {
  view all {
    title "All"
    include *
  }
  view ctx of p {
    title "Ctx"
    include *
  }
  view some of p {
    include api, db
    exclude db
  }
  view kids of p {
    include p.*
  }
  view deep of p {
    include p.**
  }
}
`

// `include *` of a LikeC4 view was dropped by the tokenizer, so every imported
// view had no include list and resolved to 0 elements (#706).
func TestImport_ViewIncludes(t *testing.T) {
	r, err := likec4.ImportSource(includeSrc)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{
		"all":  {"*"},
		"ctx":  {"*", "p.*"},
		"some": {"p.api", "p.db"},
		"kids": {"p.*"},
		"deep": {"p.**"},
	}
	for key, inc := range want {
		v, ok := r.Model.Views[key]
		if !ok {
			t.Errorf("view %s missing", key)
			continue
		}
		if !reflect.DeepEqual(v.Include, inc) {
			t.Errorf("view %s: include = %v, want %v", key, v.Include, inc)
		}
	}
	if got := r.Model.Views["some"].Exclude; !reflect.DeepEqual(got, []string{"p.db"}) {
		t.Errorf("exclude = %v, want [p.db]", got)
	}
}

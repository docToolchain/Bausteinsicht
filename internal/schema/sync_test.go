package schema_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/docToolchain/Bausteinsicht/internal/schema"
)

// repoRoot walks up from this test file to the module root (the directory
// that contains go.mod), so the test is independent of the working dir.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine caller file")
	}
	dir := filepath.Dir(thisFile)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found walking up from test file")
		}
		dir = parent
	}
}

// TestCommittedSchemaInSync guards against the issue #421 regression where the
// committed schema drifted from the Go types (model/views were left as bare
// objects). If this fails, run:
//
//	go run ./cmd/bausteinsicht schema generate
//
// and commit the regenerated schemas/bausteinsicht.schema.json.
func TestCommittedSchemaInSync(t *testing.T) {
	schemaPath := filepath.Join(repoRoot(t), "schemas", "bausteinsicht.schema.json")

	committed, err := os.ReadFile(schemaPath) //nolint:gosec // fixed, repo-relative path
	if err != nil {
		t.Fatalf("reading committed schema: %v", err)
	}

	gen := schema.NewGenerator()
	generated, err := gen.Generate(model.BausteinsichtModel{}).ToJSON()
	if err != nil {
		t.Fatalf("generating schema: %v", err)
	}

	// ToJSON does not append a trailing newline; os.WriteFile in the
	// generator writes the bytes verbatim, so compare verbatim.
	if string(committed) != string(generated) {
		t.Errorf("committed schema is out of sync with Go types; " +
			"run `go run ./cmd/bausteinsicht schema generate` and commit the result")
	}
}

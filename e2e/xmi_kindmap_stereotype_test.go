package e2e

// TestXMIKindMapAndStereotype verifies `import --from xmi --kind-map` and EA
// stereotype resolution through the full CLI pipeline (import -> sync ->
// find), reusing the existing unit-test fixtures. TestXMIPipeline (#491)
// only exercises the importer's default kind mapping; --kind-map and
// stereotype precedence had no E2E coverage of their own.

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func TestXMIKindMapAndStereotype(t *testing.T) {
	bin := buildBinary(t)
	root := findModuleRoot(t)

	t.Run("KindMapOverride", func(t *testing.T) {
		dir := t.TempDir()
		copyTestFile(t, filepath.Join(root, "internal/importer/xmi/testdata/kindmap.xmi"), filepath.Join(dir, "kindmap.xmi"))

		runCLI(t, bin, dir, "import", "kindmap.xmi",
			"--from", "xmi",
			"--kind-map", "Component=service,Class=entity",
			"--output", "architecture.jsonc",
		)

		m, err := model.Load(filepath.Join(dir, "architecture.jsonc"))
		if err != nil {
			t.Fatalf("model.Load: %v", err)
		}
		flat, _ := model.FlattenElements(m)
		var paymentKind, orderKind string
		for _, elem := range flat {
			switch elem.Title {
			case "Payment Service":
				paymentKind = elem.Kind
			case "Order Entity":
				orderKind = elem.Kind
			}
		}
		if paymentKind != "service" {
			t.Errorf("Payment Service: expected kind \"service\" (via --kind-map Component=service), got %q", paymentKind)
		}
		if orderKind != "entity" {
			t.Errorf("Order Entity: expected kind \"entity\" (via --kind-map Class=entity), got %q", orderKind)
		}

		// The imported model must actually sync — a mapped kind that isn't
		// registered in specification.elements would fail validation, the
		// same class of bug this repo has been fixing for Structurizr import.
		runCLI(t, bin, dir, "sync", "--model", "architecture.jsonc")
	})

	t.Run("StereotypePrecedence", func(t *testing.T) {
		dir := t.TempDir()
		copyTestFile(t, filepath.Join(root, "internal/importer/xmi/testdata/stereotypes.xmi"), filepath.Join(dir, "stereotypes.xmi"))

		runCLI(t, bin, dir, "import", "stereotypes.xmi",
			"--from", "xmi",
			"--output", "architecture.jsonc",
		)

		findOut := runCLI(t, bin, dir, "find", "Orders", "--model", "architecture.jsonc")
		if !strings.Contains(findOut, "microservice") {
			t.Errorf("find Orders: expected kind \"microservice\" (EA stereotype takes precedence over uml:Component), got: %s", findOut)
		}

		runCLI(t, bin, dir, "sync", "--model", "architecture.jsonc")
	})
}

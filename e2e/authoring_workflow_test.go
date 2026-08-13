package e2e

// TestAuthoringWorkflow (#487) walks the full architecture authoring loop:
//
//  1. init — creates model + draw.io (model already has a required-field constraint C02)
//  2. validate — model passes schema validation
//  3. lint — passes (all containers have technology in the default model)
//  4. add element — adds a container without a technology field
//  5. lint — C02 violation: container with missing technology is reported

import (
	"strings"
	"testing"
)

func TestAuthoringWorkflow(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	// ── Step 1: init ───────────────────────────────────────────────────────────
	runCLI(t, bin, dir, "init")

	// ── Step 2: validate → model is schema-valid ──────────────────────────────
	validateOut := runCLI(t, bin, dir, "validate")
	t.Logf("validate output: %s", validateOut)

	// ── Step 3: lint → should pass (all containers have technology) ───────────
	lintOut, code := runCLIAllowFail(t, bin, dir, "lint")
	t.Logf("initial lint output: %s", lintOut)
	if code != 0 {
		// Default model should be clean.
		t.Fatalf("initial lint failed with exit %d: %s", code, lintOut)
	}
	if !strings.Contains(lintOut, "passed") && !strings.Contains(lintOut, "No constraints") {
		t.Errorf("initial lint output does not indicate passing: %q", lintOut)
	}

	// ── Step 4: add a container without technology ────────────────────────────
	// The default model has constraint C02: required-field container.technology.
	// Adding a container with no technology should trigger that constraint.
	runCLI(t, bin, dir, "add", "element",
		"--id", "bare-service",
		"--kind", "container",
		"--title", "Bare Service",
	)

	// ── Step 5: lint → C02 violation reported ─────────────────────────────────
	lintAfterOut, lintCode := runCLIAllowFail(t, bin, dir, "lint")
	t.Logf("post-add lint output: %s", lintAfterOut)
	if lintCode == 0 {
		t.Error("lint should have failed after adding a container without technology (C02 constraint)")
	}
	// bare-service must be mentioned specifically — a generic VIOLATION elsewhere is not sufficient.
	if !strings.Contains(lintAfterOut, "bare-service") {
		t.Errorf("lint output does not mention 'bare-service': %q", lintAfterOut)
	}

	// ── Bonus: lint --format json also reports violation ──────────────────────
	lintJSONOut, _ := runCLIAllowFail(t, bin, dir, "lint", "--format", "json")
	if !strings.Contains(lintJSONOut, `"passed":false`) && !strings.Contains(lintJSONOut, `"total":`) {
		t.Errorf("lint --format json output missing expected JSON fields: %s", lintJSONOut)
	}

	t.Log("authoring workflow OK: init → validate → lint (pass) → add element → lint (fail) verified")
}

// TestAuthoringWorkflow_CamelCaseKinds (#582) verifies the CLI can author a
// model whose specification kinds and view key are camelCase. The model,
// schema, `validate`, and `add element --kind` already accept camelCase kinds
// (e.g. the hexagonal/DDD example), so `add specification` and `add view` must
// be able to create them too.
func TestAuthoringWorkflow_CamelCaseKinds(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	runCLI(t, bin, dir, "init")

	// Define camelCase specification kinds (hexagonal/DDD style).
	runCLI(t, bin, dir, "add", "specification", "element", "boundaryObject",
		"--notation", "Boundary Object", "--container")
	runCLI(t, bin, dir, "add", "specification", "element", "applicationService",
		"--notation", "Application Service")
	runCLI(t, bin, dir, "add", "specification", "relationship", "dependsOn",
		"--notation", "depends on")

	// A camelCase view key.
	runCLI(t, bin, dir, "add", "view", "systemContext", "--title", "System Context")

	// Use the camelCase kinds when adding elements.
	runCLI(t, bin, dir, "add", "element",
		"--id", "orderRecord", "--kind", "boundaryObject", "--title", "Order Record")
	runCLI(t, bin, dir, "add", "element",
		"--id", "orderService", "--kind", "applicationService", "--title", "Order Service")

	// The resulting model must still validate.
	validateOut := runCLI(t, bin, dir, "validate")
	if !strings.Contains(validateOut, "Model is valid.") {
		t.Errorf("expected model to validate after camelCase authoring, got: %q", validateOut)
	}

	t.Log("camelCase authoring OK: add specification/view/element with camelCase kinds → validate")
}

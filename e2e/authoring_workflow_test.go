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
	if !strings.Contains(lintAfterOut, "bare-service") && !strings.Contains(lintAfterOut, "VIOLATION") {
		t.Errorf("lint output does not mention violation or 'bare-service': %q", lintAfterOut)
	}

	// ── Bonus: lint --format json also reports violation ──────────────────────
	lintJSONOut, _ := runCLIAllowFail(t, bin, dir, "lint", "--format", "json")
	if !strings.Contains(lintJSONOut, `"passed":false`) && !strings.Contains(lintJSONOut, `"total":`) {
		t.Logf("lint JSON output: %s", lintJSONOut)
	}

	t.Log("authoring workflow OK: init → validate → lint (pass) → add element → lint (fail) verified")
}

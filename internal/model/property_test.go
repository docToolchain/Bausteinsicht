package model

import (
	"encoding/json"
	"testing"

	"pgregory.net/rapid"
)

// TestElementRoundTrip validates that Element can be JSON encoded/decoded without data loss
func TestElementRoundTrip(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		orig := &Element{
			Kind:        rapid.StringMatching(`[a-z]+`).Draw(t, "kind"),
			Title:       rapid.StringMatching(`[A-Z][a-z ]+`).Draw(t, "title"),
			Description: rapid.StringMatching(`[a-z0-9 .,]*`).Draw(t, "description"),
			Technology:  rapid.StringMatching(`[A-Z][a-z0-9-]*`).Draw(t, "tech"),
		}

		// Encode to JSON
		data, err := json.Marshal(orig)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}

		// Decode back
		var decoded Element
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}

		// Verify all fields match
		if decoded.Kind != orig.Kind || decoded.Title != orig.Title ||
			decoded.Description != orig.Description || decoded.Technology != orig.Technology {
			t.Fatalf("roundtrip mismatch: orig=%+v decoded=%+v", orig, decoded)
		}
	})
}

// TestRelationshipRoundTrip validates Relationship roundtrip
func TestRelationshipRoundTrip(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		orig := &Relationship{
			From:  rapid.StringMatching(`[a-z]+`).Draw(t, "from"),
			To:    rapid.StringMatching(`[a-z]+`).Draw(t, "to"),
			Label: rapid.StringMatching(`[a-z ]*`).Draw(t, "label"),
		}

		data, _ := json.Marshal(orig)
		var decoded Relationship
		_ = json.Unmarshal(data, &decoded)

		if decoded.From != orig.From || decoded.To != orig.To || decoded.Label != orig.Label {
			t.Fatalf("relationship roundtrip failed")
		}
	})
}

// TestModelValidationIdempotent validates validation is idempotent
func TestModelValidationIdempotent(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		m := &BausteinsichtModel{
			Specification: Specification{
				Elements: map[string]ElementKind{
					"system": {Notation: "box"},
				},
			},
			Model: map[string]Element{
				"test": {Kind: "system", Title: "Test"},
			},
		}

		// Validate multiple times
		errs1 := Validate(m)
		errs2 := Validate(m)

		// Should get same errors
		if len(errs1) != len(errs2) {
			t.Fatalf("validation not idempotent: %d vs %d errors", len(errs1), len(errs2))
		}
	})
}

// TestFlattenElementsCompleteness validates all elements are found
func TestFlattenElementsCompleteness(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		count := rapid.IntRange(1, 10).Draw(t, "count")

		m := &BausteinsichtModel{
			Specification: Specification{
				Elements: map[string]ElementKind{
					"system": {Notation: "box", Container: true},
					"module": {Notation: "box"},
				},
			},
			Model: make(map[string]Element),
		}

		// Create elements
		for i := 0; i < count; i++ {
			id := rapid.StringMatching(`[a-z]+`).Draw(t, "id")
			m.Model[id] = Element{Kind: "system", Title: "E" + id}
		}

		flat, _ := FlattenElements(m)

		// All original elements should be in flattened result
		if len(flat) < len(m.Model) {
			t.Fatalf("flattening lost elements: %d -> %d", len(m.Model), len(flat))
		}
	})
}

// TestElementIDValidation validates ID requirements
func TestElementIDValidation(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		// Valid IDs must not be empty or whitespace
		validID := rapid.StringMatching(`[a-z][a-z0-9]*`).Draw(t, "id")

		m := &BausteinsichtModel{
			Specification: Specification{
				Elements: map[string]ElementKind{
					"system": {Notation: "box"},
				},
			},
			Model: map[string]Element{
				validID: {Kind: "system", Title: "Test"},
			},
		}

		errs := Validate(m)
		if len(errs) > 0 {
			t.Fatalf("valid ID rejected: %v", errs)
		}

		// Invalid ID should fail
		m2 := &BausteinsichtModel{
			Specification: Specification{
				Elements: map[string]ElementKind{
					"system": {Notation: "box"},
				},
			},
			Model: map[string]Element{
				"": {Kind: "system", Title: "Test"},
			},
		}

		errs2 := Validate(m2)
		if len(errs2) == 0 {
			t.Fatal("empty ID should be rejected")
		}
	})
}

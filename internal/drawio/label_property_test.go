package drawio

import (
	"testing"

	"pgregory.net/rapid"
)

// safeText generates printable text without HTML-special chars that could
// break the label structure (no angle brackets in raw input).
func safeText() *rapid.Generator[string] {
	return rapid.StringMatching(`[A-Za-z0-9 _\-\.]{1,50}`)
}

func TestGenerateParseLabelRoundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		title := safeText().Draw(t, "title")
		tech := safeText().Draw(t, "tech")
		desc := safeText().Draw(t, "desc")

		html := GenerateLabel(title, tech, desc)
		gotTitle, gotTech, gotDesc := ParseLabel(html)

		if gotTitle != title {
			t.Fatalf("title mismatch: got %q, want %q (html: %s)", gotTitle, title, html)
		}
		if gotTech != tech {
			t.Fatalf("tech mismatch: got %q, want %q (html: %s)", gotTech, tech, html)
		}
		if gotDesc != desc {
			t.Fatalf("desc mismatch: got %q, want %q (html: %s)", gotDesc, desc, html)
		}
	})
}

func TestGenerateParseLabelRoundtrip_NoTech(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		title := safeText().Draw(t, "title")
		desc := safeText().Draw(t, "desc")

		html := GenerateLabel(title, "", desc)
		gotTitle, gotTech, gotDesc := ParseLabel(html)

		if gotTitle != title {
			t.Fatalf("title mismatch: got %q, want %q", gotTitle, title)
		}
		if gotTech != "" {
			t.Fatalf("tech should be empty, got %q", gotTech)
		}
		if gotDesc != desc {
			t.Fatalf("desc mismatch: got %q, want %q", gotDesc, desc)
		}
	})
}

func TestGenerateParseLabelRoundtrip_TitleOnly(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		title := safeText().Draw(t, "title")

		html := GenerateLabel(title, "", "")
		gotTitle, gotTech, gotDesc := ParseLabel(html)

		if gotTitle != title {
			t.Fatalf("title mismatch: got %q, want %q", gotTitle, title)
		}
		if gotTech != "" {
			t.Fatalf("tech should be empty, got %q", gotTech)
		}
		if gotDesc != "" {
			t.Fatalf("desc should be empty, got %q", gotDesc)
		}
	})
}

func TestEscapeUnescapeHTMLRoundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := rapid.String().Draw(t, "input")
		result := unescapeHTML(escapeHTML(input))
		if result != input {
			t.Fatalf("escape/unescape roundtrip failed: got %q, want %q", result, input)
		}
	})
}

func TestTrimBracketsIdempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := safeText().Draw(t, "text")
		bracketed := "[" + s + "]"

		result := trimBrackets(bracketed)
		if result != s {
			t.Fatalf("trimBrackets([%s]) = %q, want %q", s, result, s)
		}

		// Without brackets, should return unchanged
		result2 := trimBrackets(s)
		if result2 != s {
			t.Fatalf("trimBrackets(%s) = %q, want %q", s, result2, s)
		}
	})
}

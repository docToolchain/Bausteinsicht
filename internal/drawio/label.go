package drawio

import (
	"strings"
)

// GenerateLabel creates an HTML label for draw.io elements.
// Format: <b>Title</b><br><font color="#666666">Technology</font>
// If technology is empty, just: <b>Title</b>
// The returned string is unescaped HTML; etree handles XML attribute escaping.
func GenerateLabel(title, technology string) string {
	escaped := escapeHTML(title)
	if technology == "" {
		return "<b>" + escaped + "</b>"
	}
	escapedTech := escapeHTML(technology)
	return "<b>" + escaped + "</b><br><font color=\"#666666\">" + escapedTech + "</font>"
}

// GenerateActorLabel creates a label for actor elements (just the title, no technology line).
func GenerateActorLabel(title string) string {
	return "<b>" + escapeHTML(title) + "</b>"
}

// ParseLabel extracts title and technology from an HTML label.
// Expected format: <b>Title</b><br><font color="#666666">Technology</font>
// If the label doesn't match, return the full text as title with empty technology.
func ParseLabel(html string) (title, technology string) {
	// Try to match <b>...</b><br><font ...>...</font>
	if strings.HasPrefix(html, "<b>") {
		rest := html[len("<b>"):]
		closeB := strings.Index(rest, "</b>")
		if closeB >= 0 {
			titlePart := rest[:closeB]
			after := rest[closeB+len("</b>"):]

			if after == "" {
				return unescapeHTML(titlePart), ""
			}

			// Check for <br><font ...>...</font>
			if strings.HasPrefix(after, "<br>") {
				fontPart := after[len("<br>"):]
				if strings.HasPrefix(fontPart, "<font") {
					fontClose := strings.Index(fontPart, ">")
					if fontClose >= 0 {
						techRest := fontPart[fontClose+1:]
						endFont := strings.Index(techRest, "</font>")
						if endFont >= 0 {
							tech := techRest[:endFont]
							return unescapeHTML(titlePart), unescapeHTML(tech)
						}
					}
				}
			}
		}
	}

	// Fallback: strip all HTML tags and return plain text as title
	return stripTags(html), ""
}

// escapeHTML escapes special HTML characters in text content.
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// unescapeHTML reverses HTML entity escaping.
func unescapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&amp;", "&")
	return s
}

// stripTags removes all HTML tags from a string.
func stripTags(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return unescapeHTML(b.String())
}

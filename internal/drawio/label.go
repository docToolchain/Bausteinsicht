package drawio

import (
	"strings"
)

// Label color constants for technology and description lines.
// These are light enough to be readable on dark C4 backgrounds (#08427B,
// #1168BD, #438DD5) while still providing contrast on the lighter
// component background (#85BBF0).
const (
	techColor = "#CCCCCC"
	descColor = "#BBBBBB"
)

// maxLabelDescLen is the maximum rune length of the description portion of an
// HTML label. HTML labels are rendered inside a fixed-size element box
// (typically 120×60 px) that has no sub-cell clipping, so descriptions must
// be kept short. The full text is always preserved in the tooltip attribute.
const maxLabelDescLen = 60

// GenerateLabel creates an HTML label for draw.io elements.
// Format: <b>Title</b><br><font color="..."><i>[Technology]</i></font><br><font color="..." style="font-size:11px">Description</font>
// Technology is wrapped in square brackets per C4 convention and rendered in italic.
// Empty technology or description lines are omitted.
// The returned string is unescaped HTML; etree handles XML attribute escaping.
func GenerateLabel(title, technology, description string) string {
	var b strings.Builder
	b.WriteString("<b>" + escapeHTML(title) + "</b>")
	if technology != "" {
		b.WriteString("<br><font color=\"" + techColor + "\"><i>[" + escapeHTML(technology) + "]</i></font>")
	}
	if description != "" {
		b.WriteString("<br><font color=\"" + descColor + "\" style=\"font-size:11px\">" + escapeHTML(truncateText(description, maxLabelDescLen)) + "</font>")
	}
	return b.String()
}

// GenerateActorLabel creates a label for actor elements (just the title, no technology line).
func GenerateActorLabel(title string) string {
	return "<b>" + escapeHTML(title) + "</b>"
}

// ParseLabel extracts title, technology and description from an HTML label.
// Expected format: <b>Title</b><br><font color="#666666">[Technology]</font><br><font color="#999999">Description</font>
// Also handles legacy format without brackets around technology.
// If the label doesn't match, return the full text as title.
func ParseLabel(html string) (title, technology, description string) {
	if !strings.HasPrefix(html, "<b>") {
		return stripTags(html), "", ""
	}

	rest := html[len("<b>"):]
	closeB := strings.Index(rest, "</b>")
	if closeB < 0 {
		return stripTags(html), "", ""
	}

	titlePart := rest[:closeB]
	after := rest[closeB+len("</b>"):]

	cleanTitle := stripTags(titlePart)

	if after == "" {
		return cleanTitle, "", ""
	}

	// Parse remaining <br><font ...>...</font> segments
	segments := parseFontSegments(after)

	switch len(segments) {
	case 1:
		seg := segments[0]
		if seg.color == descColor || seg.color == "#999999" {
			// Description only (no technology)
			return cleanTitle, "", unescapeHTML(stripTags(seg.text))
		}
		// Technology (with or without brackets)
		return cleanTitle, unescapeHTML(trimBrackets(stripTags(seg.text))), ""
	case 2:
		tech := unescapeHTML(trimBrackets(stripTags(segments[0].text)))
		desc := unescapeHTML(stripTags(segments[1].text))
		return cleanTitle, tech, desc
	default:
		return cleanTitle, "", ""
	}
}

type fontSegment struct {
	color string
	text  string
}

// parseFontSegments extracts consecutive <br><font color="...">...</font> segments.
func parseFontSegments(s string) []fontSegment {
	var segments []fontSegment
	for strings.HasPrefix(s, "<br>") {
		s = s[len("<br>"):]
		if !strings.HasPrefix(s, "<font") {
			break
		}
		// Extract color attribute
		colorStart := strings.Index(s, `color="`)
		if colorStart < 0 {
			break
		}
		colorStart += len(`color="`)
		colorEnd := strings.Index(s[colorStart:], `"`)
		if colorEnd < 0 {
			break
		}
		color := s[colorStart : colorStart+colorEnd]

		// Extract text content
		tagClose := strings.Index(s, ">")
		if tagClose < 0 {
			break
		}
		textStart := tagClose + 1
		endFont := strings.Index(s[textStart:], "</font>")
		if endFont < 0 {
			break
		}
		text := s[textStart : textStart+endFont]
		segments = append(segments, fontSegment{color: color, text: text})
		s = s[textStart+endFont+len("</font>"):]
	}
	return segments
}

// trimBrackets removes surrounding square brackets if present.
func trimBrackets(s string) string {
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		return s[1 : len(s)-1]
	}
	return s
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

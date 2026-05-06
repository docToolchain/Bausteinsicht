package changelog

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// RenderMarkdown renders the changelog as Markdown
func RenderMarkdown(cl *Changelog) string {
	var sb strings.Builder

	sb.WriteString("# Architecture Changelog\n\n")

	// Title with date range
	dateRange := fmt.Sprintf("%s → %s", cl.From.Ref, cl.To.Ref)
	if !cl.From.Date.IsZero() && !cl.To.Date.IsZero() {
		dateRange = fmt.Sprintf("%s → %s (%s → %s)",
			cl.From.Ref, cl.To.Ref,
			cl.From.Date.Format("2006-01-02"),
			cl.To.Date.Format("2006-01-02"))
	}
	sb.WriteString(fmt.Sprintf("## %s\n\n", dateRange))

	// Added elements
	if cl.Elements.CountAdded() > 0 {
		sb.WriteString(fmt.Sprintf("### Added (%d elements)\n", cl.Elements.CountAdded()))
		for _, change := range cl.Elements.Added {
			if change.ToBe != nil {
				desc := ""
				if change.ToBe.Description != "" {
					desc = fmt.Sprintf(" _{%s}_", change.ToBe.Description)
				}
				sb.WriteString(fmt.Sprintf("- **%s** `[%s]` — %s%s\n",
					change.ID, change.ToBe.Kind, change.ToBe.Title, desc))
			}
		}
		sb.WriteString("\n")
	}

	// Removed elements
	if cl.Elements.CountRemoved() > 0 {
		sb.WriteString(fmt.Sprintf("### Removed (%d elements)\n", cl.Elements.CountRemoved()))
		for _, change := range cl.Elements.Removed {
			if change.AsIs != nil {
				desc := ""
				if change.AsIs.Description != "" {
					desc = fmt.Sprintf(" _{%s}_", change.AsIs.Description)
				}
				sb.WriteString(fmt.Sprintf("- ~~**%s**~~ `[%s]` — %s%s\n",
					change.ID, change.AsIs.Kind, change.AsIs.Title, desc))
			}
		}
		sb.WriteString("\n")
	}

	// Changed elements
	if cl.Elements.CountChanged() > 0 {
		sb.WriteString(fmt.Sprintf("### Changed (%d elements)\n", cl.Elements.CountChanged()))
		for _, change := range cl.Elements.Changed {
			if change.AsIs != nil && change.ToBe != nil {
				sb.WriteString(fmt.Sprintf("- **%s** — ", change.ID))
				changes := renderElementChanges(*change.AsIs, *change.ToBe)
				sb.WriteString(changes)
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}

	// Added relationships
	if cl.Relationships.CountAddedRelationships() > 0 {
		sb.WriteString(fmt.Sprintf("### New Relationships (%d)\n", cl.Relationships.CountAddedRelationships()))
		for _, change := range cl.Relationships.Added {
			label := ""
			if change.ToBe != nil && change.ToBe.Label != "" {
				label = fmt.Sprintf(" (%s)", change.ToBe.Label)
			}
			sb.WriteString(fmt.Sprintf("- %s → %s%s\n", change.From, change.To, label))
		}
		sb.WriteString("\n")
	}

	// Removed relationships
	if cl.Relationships.CountRemovedRelationships() > 0 {
		sb.WriteString(fmt.Sprintf("### Removed Relationships (%d)\n", cl.Relationships.CountRemovedRelationships()))
		for _, change := range cl.Relationships.Removed {
			label := ""
			if change.AsIs != nil && change.AsIs.Label != "" {
				label = fmt.Sprintf(" (%s)", change.AsIs.Label)
			}
			sb.WriteString(fmt.Sprintf("- ~~%s → %s~~%s\n", change.From, change.To, label))
		}
		sb.WriteString("\n")
	}

	if cl.Elements.CountAdded() == 0 && cl.Elements.CountRemoved() == 0 &&
		cl.Elements.CountChanged() == 0 && cl.Relationships.CountAddedRelationships() == 0 &&
		cl.Relationships.CountRemovedRelationships() == 0 {
		sb.WriteString("No architectural changes detected.\n")
	}

	return sb.String()
}

// RenderAsciiDoc renders the changelog as AsciiDoc
func RenderAsciiDoc(cl *Changelog) string {
	var sb strings.Builder

	sb.WriteString("= Architecture Changelog\n\n")

	// Title with date range
	dateRange := fmt.Sprintf("%s → %s", cl.From.Ref, cl.To.Ref)
	if !cl.From.Date.IsZero() && !cl.To.Date.IsZero() {
		dateRange = fmt.Sprintf("%s → %s (%s → %s)",
			cl.From.Ref, cl.To.Ref,
			cl.From.Date.Format("2006-01-02"),
			cl.To.Date.Format("2006-01-02"))
	}
	sb.WriteString(fmt.Sprintf("== %s\n\n", dateRange))

	// Added elements
	if cl.Elements.CountAdded() > 0 {
		sb.WriteString(fmt.Sprintf("=== Added (%d elements)\n", cl.Elements.CountAdded()))
		for _, change := range cl.Elements.Added {
			if change.ToBe != nil {
				desc := ""
				if change.ToBe.Description != "" {
					desc = fmt.Sprintf(": %s", change.ToBe.Description)
				}
				sb.WriteString(fmt.Sprintf("* *%s* `[%s]` – %s%s\n",
					change.ID, change.ToBe.Kind, change.ToBe.Title, desc))
			}
		}
		sb.WriteString("\n")
	}

	// Removed elements
	if cl.Elements.CountRemoved() > 0 {
		sb.WriteString(fmt.Sprintf("=== Removed (%d elements)\n", cl.Elements.CountRemoved()))
		for _, change := range cl.Elements.Removed {
			if change.AsIs != nil {
				desc := ""
				if change.AsIs.Description != "" {
					desc = fmt.Sprintf(": %s", change.AsIs.Description)
				}
				sb.WriteString(fmt.Sprintf("* [line-through]#*%s* `[%s]` – %s#%s\n",
					change.ID, change.AsIs.Kind, change.AsIs.Title, desc))
			}
		}
		sb.WriteString("\n")
	}

	// Changed elements
	if cl.Elements.CountChanged() > 0 {
		sb.WriteString(fmt.Sprintf("=== Changed (%d elements)\n", cl.Elements.CountChanged()))
		for _, change := range cl.Elements.Changed {
			if change.AsIs != nil && change.ToBe != nil {
				sb.WriteString(fmt.Sprintf("* *%s* – ", change.ID))
				changes := renderElementChanges(*change.AsIs, *change.ToBe)
				sb.WriteString(changes)
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}

	// Added relationships
	if cl.Relationships.CountAddedRelationships() > 0 {
		sb.WriteString(fmt.Sprintf("=== New Relationships (%d)\n", cl.Relationships.CountAddedRelationships()))
		for _, change := range cl.Relationships.Added {
			label := ""
			if change.ToBe != nil && change.ToBe.Label != "" {
				label = fmt.Sprintf(" (%s)", change.ToBe.Label)
			}
			sb.WriteString(fmt.Sprintf("* %s → %s%s\n", change.From, change.To, label))
		}
		sb.WriteString("\n")
	}

	// Removed relationships
	if cl.Relationships.CountRemovedRelationships() > 0 {
		sb.WriteString(fmt.Sprintf("=== Removed Relationships (%d)\n", cl.Relationships.CountRemovedRelationships()))
		for _, change := range cl.Relationships.Removed {
			label := ""
			if change.AsIs != nil && change.AsIs.Label != "" {
				label = fmt.Sprintf(" (%s)", change.AsIs.Label)
			}
			sb.WriteString(fmt.Sprintf("* [line-through]#%s → %s#%s\n", change.From, change.To, label))
		}
		sb.WriteString("\n")
	}

	if cl.Elements.CountAdded() == 0 && cl.Elements.CountRemoved() == 0 &&
		cl.Elements.CountChanged() == 0 && cl.Relationships.CountAddedRelationships() == 0 &&
		cl.Relationships.CountRemovedRelationships() == 0 {
		sb.WriteString("No architectural changes detected.\n")
	}

	return sb.String()
}

// RenderJSON renders the changelog as JSON
func RenderJSON(cl *Changelog) (string, error) {
	data, err := json.MarshalIndent(cl, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// renderElementChanges formats what changed in an element
func renderElementChanges(asIs, toBe model.Element) string {
	var sb strings.Builder

	if asIs.Title != toBe.Title {
		sb.WriteString(fmt.Sprintf("title: \"%s\" → \"%s\"; ", asIs.Title, toBe.Title))
	}
	if asIs.Kind != toBe.Kind {
		sb.WriteString(fmt.Sprintf("kind: \"%s\" → \"%s\"; ", asIs.Kind, toBe.Kind))
	}
	if asIs.Technology != toBe.Technology {
		sb.WriteString(fmt.Sprintf("technology: \"%s\" → \"%s\"; ", asIs.Technology, toBe.Technology))
	}
	if asIs.Description != toBe.Description {
		sb.WriteString("description: changed; ")
	}
	if asIs.Status != toBe.Status {
		sb.WriteString(fmt.Sprintf("status: \"%s\" → \"%s\"; ", asIs.Status, toBe.Status))
	}

	result := sb.String()
	return strings.TrimSuffix(result, "; ")
}

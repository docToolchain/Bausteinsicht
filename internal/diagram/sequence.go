package diagram

import (
	"fmt"
	"sort"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// RenderSequencePlantUML renders a DynamicView as a PlantUML sequence diagram.
func RenderSequencePlantUML(view model.DynamicView, flat map[string]*model.Element) string {
	steps := sortedSteps(view.Steps)
	participants := collectParticipants(steps)

	var b strings.Builder
	fmt.Fprintf(&b, "@startuml %s\n", sanitizeID(view.Key))
	fmt.Fprintf(&b, "title %s\n\n", escapeQuotes(view.Title))

	for _, id := range participants {
		title := participantTitle(id, flat)
		fmt.Fprintf(&b, "participant \"%s\" as %s\n", escapeQuotes(title), sanitizeID(id))
	}
	b.WriteString("\n")

	for _, step := range steps {
		arrow := plantUMLArrow(step.Type)
		label := fmt.Sprintf("%d. %s", step.Index, escapeQuotes(step.Label))
		fmt.Fprintf(&b, "%s %s %s : %s\n", sanitizeID(step.From), arrow, sanitizeID(step.To), label)
	}

	b.WriteString("\n@enduml\n")
	return b.String()
}

// RenderSequenceMermaid renders a DynamicView as a Mermaid sequence diagram.
func RenderSequenceMermaid(view model.DynamicView, flat map[string]*model.Element) string {
	steps := sortedSteps(view.Steps)
	participants := collectParticipants(steps)

	var b strings.Builder
	b.WriteString("sequenceDiagram\n")
	fmt.Fprintf(&b, "    title %s\n\n", escapeQuotes(view.Title))

	for _, id := range participants {
		title := participantTitle(id, flat)
		fmt.Fprintf(&b, "    participant %s as %s\n", sanitizeID(id), escapeQuotes(title))
	}
	b.WriteString("\n")

	for _, step := range steps {
		arrow := mermaidArrow(step.Type)
		label := fmt.Sprintf("%d. %s", step.Index, escapeQuotes(step.Label))
		fmt.Fprintf(&b, "    %s%s%s: %s\n", sanitizeID(step.From), arrow, sanitizeID(step.To), label)
	}

	return b.String()
}

// sortedSteps returns steps sorted by index.
func sortedSteps(steps []model.SequenceStep) []model.SequenceStep {
	sorted := make([]model.SequenceStep, len(steps))
	copy(sorted, steps)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Index < sorted[j].Index
	})
	return sorted
}

// collectParticipants returns participant IDs in first-appearance order.
func collectParticipants(steps []model.SequenceStep) []string {
	seen := make(map[string]bool)
	var order []string
	for _, s := range steps {
		if !seen[s.From] {
			seen[s.From] = true
			order = append(order, s.From)
		}
		if !seen[s.To] {
			seen[s.To] = true
			order = append(order, s.To)
		}
	}
	return order
}

func participantTitle(id string, flat map[string]*model.Element) string {
	if flat != nil {
		if e, ok := flat[id]; ok && e.Title != "" {
			return e.Title
		}
	}
	return id
}

func plantUMLArrow(t model.StepType) string {
	switch t {
	case model.StepAsync:
		return "->>"
	case model.StepReturn:
		return "-->"
	default: // sync or empty
		return "->"
	}
}

func mermaidArrow(t model.StepType) string {
	switch t {
	case model.StepAsync:
		return "-)"
	case model.StepReturn:
		return "-->>"
	default: // sync or empty
		return "->>"
	}
}

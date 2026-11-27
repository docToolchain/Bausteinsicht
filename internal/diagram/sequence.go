package diagram

import (
	"fmt"
	"sort"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// RenderSequencePlantUML renders a DynamicView as a PlantUML sequence diagram.
//
// Step numbers come from PlantUML's autonumber, not from the labels, and the
// step index only orders the steps. Elements that C4 renders as Person become
// actors, data stores databases and queues queues; spec decides that the same
// way the C4 export does and may be nil.
func RenderSequencePlantUML(view model.DynamicView, flat map[string]*model.Element, spec *model.Specification) string {
	steps := sortedSteps(view.Steps)

	var b strings.Builder
	fmt.Fprintf(&b, "@startuml %s\n", sanitizeID(view.Key))
	fmt.Fprintf(&b, "title %s\n", escapePlantUMLText(view.Title))
	b.WriteString("autonumber\n\n")

	for _, id := range collectParticipants(steps) {
		shape := plantUMLParticipantShape(participantMacro(id, flat, spec))
		fmt.Fprintf(&b, "%s \"%s\" as %s\n", shape, escapePlantUMLText(participantTitle(id, flat)), sanitizeID(id))
	}
	b.WriteString("\n")

	depth := 0
	for _, step := range steps {
		step.Label = replaceLineBreaks(strings.ReplaceAll(step.Label, "\\", "\\\\"), `\n`)
		if keyword, isBlock := blockKeyword(step.Type); isBlock {
			depth = writeBlockStep(&b, "", keyword, step, depth)
			continue
		}
		fmt.Fprintf(&b, "%s%s %s %s : %s\n", strings.Repeat("  ", depth),
			sanitizeID(step.From), plantUMLArrow(step.Type), sanitizeID(step.To), step.Label)
	}

	b.WriteString("\n@enduml\n")
	return b.String()
}

// RenderSequenceMermaid renders a DynamicView as a Mermaid sequence diagram.
// See RenderSequencePlantUML for the numbering and participant rules; Mermaid
// has actors but no database or queue participants here, and no "group", which
// is drawn as a grey rectangle.
func RenderSequenceMermaid(view model.DynamicView, flat map[string]*model.Element, spec *model.Specification) string {
	steps := sortedSteps(view.Steps)

	var b strings.Builder
	b.WriteString("sequenceDiagram\n")
	fmt.Fprintf(&b, "    title %s\n", mermaidSequenceText(escapeQuotes(view.Title)))
	b.WriteString("    autonumber\n\n")

	for _, id := range collectParticipants(steps) {
		kind := "participant"
		if strings.HasPrefix(participantMacro(id, flat, spec), "Person") {
			kind = "actor"
		}
		fmt.Fprintf(&b, "    %s %s as %s\n", kind, sanitizeID(id), mermaidSequenceText(escapeQuotes(participantTitle(id, flat))))
	}
	b.WriteString("\n")

	depth := 0
	for _, step := range steps {
		if keyword, isBlock := blockKeyword(step.Type); isBlock {
			if keyword == "group" {
				keyword = "rect rgb(240, 240, 240)"
				step.Label = ""
			}
			step.Label = mermaidSequenceText(step.Label)
			depth = writeBlockStep(&b, "    ", keyword, step, depth)
			continue
		}
		fmt.Fprintf(&b, "    %s%s%s%s: %s\n", strings.Repeat("  ", depth),
			sanitizeID(step.From), mermaidArrow(step.Type), sanitizeID(step.To), mermaidSequenceText(step.Label))
	}

	return b.String()
}

// mermaidSequenceText makes free text safe inside one statement of a Mermaid
// sequence diagram: ";" ends a statement and is written as the entity "#59;", a
// line break would start a new statement and becomes <br/> (#714).
func mermaidSequenceText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n", "<br/>")
	return strings.ReplaceAll(s, ";", "#59;")
}

// blockKeyword returns the diagram keyword of a block step (alt, else, opt,
// group, end) and whether t is a block step at all.
func blockKeyword(t model.StepType) (string, bool) {
	switch t {
	case model.StepAlt, model.StepElse, model.StepOpt, model.StepGroup, model.StepEnd:
		return string(t), true
	}
	return "", false
}

// writeBlockStep writes one block step and returns the nesting depth for the
// steps that follow: alt, opt and group open a level, end closes one, else
// stays on the level of its alt.
func writeBlockStep(b *strings.Builder, indent, keyword string, step model.SequenceStep, depth int) int {
	if step.Type == model.StepEnd && depth > 0 {
		depth--
	}
	line := keyword
	if step.Type != model.StepEnd && step.Label != "" {
		line += " " + step.Label
	}
	level := depth
	if step.Type == model.StepElse && level > 0 {
		level--
	}
	fmt.Fprintf(b, "%s%s%s\n", indent, strings.Repeat("  ", level), line)
	if step.Type == model.StepAlt || step.Type == model.StepOpt || step.Type == model.StepGroup {
		depth++
	}
	return depth
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

// collectParticipants returns participant IDs in first-appearance order. Block
// steps have no endpoints and are skipped.
func collectParticipants(steps []model.SequenceStep) []string {
	seen := make(map[string]bool)
	var order []string
	for _, s := range steps {
		if _, isBlock := blockKeyword(s.Type); isBlock {
			continue
		}
		for _, id := range []string{s.From, s.To} {
			if !seen[id] {
				seen[id] = true
				order = append(order, id)
			}
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

// participantMacro returns the C4 macro the element would get in the C4
// export, or "" if the element is unknown.
func participantMacro(id string, flat map[string]*model.Element, spec *model.Specification) string {
	e, ok := flat[id]
	if !ok || e == nil {
		return ""
	}
	macro, _ := C4Macro(spec, e.Kind)
	return macro
}

func plantUMLParticipantShape(macro string) string {
	switch {
	case strings.HasPrefix(macro, "Person"):
		return "actor"
	case strings.Contains(macro, "Db"):
		return "database"
	case strings.Contains(macro, "Queue"):
		return "queue"
	default:
		return "participant"
	}
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

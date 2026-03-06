package table

import (
	"fmt"
	"sort"
	"strings"

	"github.com/docToolchain/Bauteinsicht/internal/model"
)

// Format represents the output format for table export.
type Format int

const (
	AsciiDoc Format = iota
	Markdown
)

// Row represents a single element row in the table export.
type Row struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Kind        string `json:"kind"`
	Technology  string `json:"technology,omitempty"`
	Description string `json:"description,omitempty"`
}

// FormatView renders a single view's elements as a table.
func FormatView(m *model.BausteinsichtModel, viewKey string, f Format) (string, error) {
	view, ok := m.Views[viewKey]
	if !ok {
		return "", fmt.Errorf("view %q not found", viewKey)
	}

	rows, err := resolveRows(m, &view)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	writeTitle(&b, view.Title, f)
	writeTable(&b, rows, f)
	return b.String(), nil
}

// FormatAllViews renders all views as tables in a single document.
func FormatAllViews(m *model.BausteinsichtModel, f Format) (string, error) {
	keys := sortedViewKeys(m)
	var b strings.Builder
	for i, key := range keys {
		if i > 0 {
			b.WriteString("\n")
		}
		view, ok := m.Views[key]
		if !ok {
			continue
		}
		rows, err := resolveRows(m, &view)
		if err != nil {
			return "", err
		}
		writeTitle(&b, view.Title, f)
		writeTable(&b, rows, f)
	}
	return b.String(), nil
}

// FormatCombined renders all elements across all views (deduplicated) as a single table.
func FormatCombined(m *model.BausteinsichtModel, f Format) (string, error) {
	seen := make(map[string]bool)
	var rows []Row

	flat, _ := model.FlattenElements(m)
	keys := sortedViewKeys(m)

	for _, key := range keys {
		view, ok := m.Views[key]
		if !ok {
			continue
		}
		v := view
		resolved, err := model.ResolveView(m, &v)
		if err != nil {
			continue
		}
		for _, id := range resolved {
			if seen[id] {
				continue
			}
			seen[id] = true
			elem := flat[id]
			if elem == nil {
				continue
			}
			rows = append(rows, Row{
				ID:          id,
				Title:       elem.Title,
				Kind:        elem.Kind,
				Technology:  elem.Technology,
				Description: elem.Description,
			})
		}
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })

	var b strings.Builder
	writeTitle(&b, "All Elements", f)
	writeTable(&b, rows, f)
	return b.String(), nil
}

func resolveRows(m *model.BausteinsichtModel, view *model.View) ([]Row, error) {
	resolved, err := model.ResolveView(m, view)
	if err != nil {
		return nil, err
	}

	flat, _ := model.FlattenElements(m)
	sort.Strings(resolved)

	var rows []Row
	for _, id := range resolved {
		elem := flat[id]
		if elem == nil {
			continue
		}
		rows = append(rows, Row{
			ID:          id,
			Title:       elem.Title,
			Kind:        elem.Kind,
			Technology:  elem.Technology,
			Description: elem.Description,
		})
	}
	return rows, nil
}

func writeTitle(b *strings.Builder, title string, f Format) {
	switch f {
	case AsciiDoc:
		fmt.Fprintf(b, "=== %s\n\n", title)
	case Markdown:
		fmt.Fprintf(b, "### %s\n\n", title)
	}
}

func writeTable(b *strings.Builder, rows []Row, f Format) {
	switch f {
	case AsciiDoc:
		writeAsciiDocTable(b, rows)
	case Markdown:
		writeMarkdownTable(b, rows)
	}
}

func writeAsciiDocTable(b *strings.Builder, rows []Row) {
	b.WriteString("[cols=\"2,1,1,3\"]\n|===\n")
	b.WriteString("| Element | Kind | Technology | Description\n\n")
	for _, r := range rows {
		fmt.Fprintf(b, "| %s\n| %s\n| %s\n| %s\n\n", r.Title, r.Kind, r.Technology, r.Description)
	}
	b.WriteString("|===\n")
}

func writeMarkdownTable(b *strings.Builder, rows []Row) {
	b.WriteString("| Element | Kind | Technology | Description |\n")
	b.WriteString("|---------|------|------------|-------------|\n")
	for _, r := range rows {
		fmt.Fprintf(b, "| %s | %s | %s | %s |\n", r.Title, r.Kind, r.Technology, r.Description)
	}
}

// CollectRows returns the row data for JSON export.
// If viewKey is set, only that view's rows are returned.
// If combined is true, all views are merged (deduplicated).
// Otherwise, all views' rows are returned.
func CollectRows(m *model.BausteinsichtModel, viewKey string, combined bool) ([]Row, error) {
	switch {
	case combined:
		return collectCombinedRows(m)
	case viewKey != "":
		view, ok := m.Views[viewKey]
		if !ok {
			return nil, fmt.Errorf("view %q not found", viewKey)
		}
		return resolveRows(m, &view)
	default:
		return collectAllRows(m)
	}
}

func collectCombinedRows(m *model.BausteinsichtModel) ([]Row, error) {
	seen := make(map[string]bool)
	var rows []Row
	flat, _ := model.FlattenElements(m)
	for _, key := range sortedViewKeys(m) {
		view, ok := m.Views[key]
		if !ok {
			continue
		}
		v := view
		resolved, err := model.ResolveView(m, &v)
		if err != nil {
			continue
		}
		for _, id := range resolved {
			if seen[id] {
				continue
			}
			seen[id] = true
			elem := flat[id]
			if elem == nil {
				continue
			}
			rows = append(rows, Row{
				ID: id, Title: elem.Title, Kind: elem.Kind,
				Technology: elem.Technology, Description: elem.Description,
			})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows, nil
}

func collectAllRows(m *model.BausteinsichtModel) ([]Row, error) {
	var rows []Row
	for _, key := range sortedViewKeys(m) {
		view, ok := m.Views[key]
		if !ok {
			continue
		}
		viewRows, err := resolveRows(m, &view)
		if err != nil {
			return nil, err
		}
		rows = append(rows, viewRows...)
	}
	return rows, nil
}

func sortedViewKeys(m *model.BausteinsichtModel) []string {
	keys := make([]string, 0, len(m.Views))
	for k := range m.Views {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

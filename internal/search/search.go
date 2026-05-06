// Package search implements full-text search over Bausteinsicht model objects
// (elements, relationships, views) with field-weighted relevance scoring.
package search

import (
	"sort"
	"strings"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// Run searches the model for the given query and returns ranked results.
// The query is split into words; all words must appear (AND semantics).
// Results are sorted by descending score, then alphabetically by ID.
func Run(query string, m *model.BausteinsichtModel, opts Options) Response {
	words := tokenise(query)
	if len(words) == 0 {
		return Response{Query: query, Results: []Result{}, Total: 0}
	}

	flat, err := model.FlattenElements(m)
	if err != nil {
		return Response{Query: query, Results: []Result{}, Total: 0}
	}

	// Build a title lookup for relationships (from/to display).
	titleOf := func(id string) string {
		if e, ok := flat[id]; ok && e.Title != "" {
			return e.Title
		}
		return id
	}

	var results []Result

	if opts.Type == "" || opts.Type == ResultElement {
		for id, elem := range flat {
			score, matched := scoreElement(id, elem.Title, elem.Description, elem.Technology, elem.Kind, elem.Tags, words)
			if score == 0 {
				continue
			}
			results = append(results, Result{
				Type:          ResultElement,
				ID:            id,
				Title:         elem.Title,
				Kind:          elem.Kind,
				Technology:    elem.Technology,
				Description:   elem.Description,
				Score:         score,
				MatchedFields: matched,
			})
		}
	}

	if opts.Type == "" || opts.Type == ResultRelationship {
		for _, rel := range m.Relationships {
			id := rel.From + "->" + rel.To
			score, matched := scoreRelationship(id, rel.Label, rel.Kind, titleOf(rel.From), titleOf(rel.To), words)
			if score == 0 {
				continue
			}
			results = append(results, Result{
				Type:          ResultRelationship,
				ID:            id,
				Title:         rel.Label,
				Kind:          rel.Kind,
				From:          rel.From,
				To:            rel.To,
				Description:   rel.Description,
				Score:         score,
				MatchedFields: matched,
			})
		}
	}

	if opts.Type == "" || opts.Type == ResultView {
		for key, view := range m.Views {
			score, matched := scoreView(key, view.Title, view.Description, words)
			if score == 0 {
				continue
			}
			results = append(results, Result{
				Type:          ResultView,
				ID:            key,
				Title:         view.Title,
				Description:   view.Description,
				Score:         score,
				MatchedFields: matched,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].ID < results[j].ID
	})

	return Response{
		Query:   query,
		Results: results,
		Total:   len(results),
	}
}

// tokenise lowercases the query and splits it into words.
func tokenise(query string) []string {
	lower := strings.ToLower(strings.TrimSpace(query))
	if lower == "" {
		return nil
	}
	return strings.Fields(lower)
}

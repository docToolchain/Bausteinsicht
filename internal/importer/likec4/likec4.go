// Package likec4 parses LikeC4 DSL files and converts them to the
// Bausteinsicht model format.
package likec4

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/docToolchain/Bausteinsicht/internal/importer"
	"github.com/docToolchain/Bausteinsicht/internal/importer/dsl"
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// ─── Tokenizer (identical grammar to Structurizr) ────────────────────────────
//
// The scanner/parser primitives shared with Structurizr's near-identical DSL
// grammar live in internal/importer/dsl; only the language-specific token
// and statement dispatch stay here.

func tokenize(src string) ([]dsl.Token, error) {
	return dsl.Tokenize(src, identStart, scanIdent)
}

// identStart reports whether c can start a LikeC4 identifier. Unlike
// Structurizr, LikeC4 has no "*" wildcard or "!" prefix identifiers.
func identStart(c rune) bool {
	return unicode.IsLetter(c) || c == '_'
}

func scanIdent(s *dsl.Scanner, line int) (dsl.Token, error) {
	var sb strings.Builder
	dsl.ScanIdentBody(s, &sb)
	return dsl.Token{Kind: dsl.Ident, Val: sb.String(), Line: line}, nil
}

// ─── Parser ──────────────────────────────────────────────────────────────────
//
// LikeC4's statement grammar is identical to Structurizr's — both are
// "[var =] keyword args {body}" / "[from] -> to args {body}" — so parsing
// itself lives entirely in the dsl package (dsl.ParseAllStmts).

// ─── Mapper ──────────────────────────────────────────────────────────────────

type lc4State struct {
	kinds          map[string]bool // known element kind names from specification
	kindsContainer map[string]bool // kinds that have children in the model
	spec           map[string]model.ElementKind
	elements       map[string]model.Element
	varToPath      map[string]string
	pendingRels    []pendingRel
	views          map[string]model.View
	viewKeys       map[string]int
	warnings       []string
}

type pendingRel struct {
	from  string
	to    string
	label string
	line  int
}

func newLC4State() *lc4State {
	return &lc4State{
		kinds:          make(map[string]bool),
		kindsContainer: make(map[string]bool),
		spec:           make(map[string]model.ElementKind),
		elements:       make(map[string]model.Element),
		varToPath:      make(map[string]string),
		views:          make(map[string]model.View),
		viewKeys:       make(map[string]int),
	}
}

func (ls *lc4State) processSpecification(stmts []dsl.Stmt) {
	for _, s := range stmts {
		switch s.Keyword {
		case "element":
			if len(s.Args) == 0 {
				continue
			}
			kindName := s.Args[0]
			ls.kinds[kindName] = true
			notation := strings.ToUpper(kindName[:1]) + kindName[1:]
			ls.spec[kindName] = model.ElementKind{Notation: notation}
		case "relationship":
			// ignore — Bausteinsicht relationships don't need pre-declared types
		}
	}
}

func (ls *lc4State) resolveVar(v string) string {
	if p, ok := ls.varToPath[v]; ok {
		return p
	}
	return v
}

func (ls *lc4State) processModelStmts(stmts []dsl.Stmt, parentPath, parentVar string, dest map[string]model.Element) {
	for _, s := range stmts {
		ls.processModelStmt(s, parentPath, parentVar, dest)
	}
}

func (ls *lc4State) processModelStmt(s dsl.Stmt, parentPath, parentVar string, dest map[string]model.Element) {
	if s.IsRel {
		ls.addPendingRel(s, parentVar)
		return
	}

	if !ls.kinds[s.Keyword] {
		// Not a known kind — treat as property inside element body
		return
	}

	key := ls.resolveElementKey(s)
	path := key
	if parentPath != "" {
		path = parentPath + "." + key
	}
	ls.varToPath[key] = path

	el := model.Element{Kind: s.Keyword}
	if len(s.Args) > 0 {
		el.Title = s.Args[0]
	}

	children := make(map[string]model.Element)
	for _, child := range s.Body {
		ls.processModelChild(child, path, key, &el, children)
	}

	if len(children) > 0 {
		el.Children = children
		ls.kindsContainer[s.Keyword] = true
	}

	dest[key] = el
}

// addPendingRel records a relationship statement for later resolution once
// all elements have been discovered, defaulting its source to parentVar
// when the statement omitted an explicit source (e.g. a nested "-> b" inside
// element a's body).
func (ls *lc4State) addPendingRel(s dsl.Stmt, parentVar string) {
	from := s.RelFrom
	if from == "" {
		from = parentVar
	}
	label := ""
	if len(s.Args) > 0 {
		label = s.Args[0]
	}
	ls.pendingRels = append(ls.pendingRels, pendingRel{from: from, to: s.RelTo, label: label, line: s.Line})
}

// resolveElementKey returns s's variable name, or a slugified fallback
// derived from its title (or keyword if untitled), warning when no variable
// name was given in the DSL.
func (ls *lc4State) resolveElementKey(s dsl.Stmt) string {
	if s.VarName != "" {
		return s.VarName
	}
	key := s.Keyword
	if len(s.Args) > 0 {
		key = dsl.Slugify(s.Args[0])
	}
	ls.warnings = append(ls.warnings, fmt.Sprintf("line %d: element has no variable name, using %q", s.Line, key))
	return key
}

// processModelChild applies a single child statement of an element's body:
// a nested relationship, a nested element, or one of the description/
// technology/title/tags fields.
func (ls *lc4State) processModelChild(child dsl.Stmt, path, key string, el *model.Element, children map[string]model.Element) {
	switch {
	case child.IsRel:
		ls.addPendingRel(child, key)
	case ls.kinds[child.Keyword]:
		ls.processModelStmt(child, path, key, children)
	case child.Keyword == "description" && len(child.Args) > 0:
		el.Description = child.Args[0]
	case child.Keyword == "technology" && len(child.Args) > 0:
		el.Technology = child.Args[0]
	case child.Keyword == "title" && len(child.Args) > 0:
		el.Title = child.Args[0]
	case child.Keyword == "tags":
		el.Tags = child.Args
	}
}

func (ls *lc4State) processViews(stmts []dsl.Stmt) {
	for _, s := range stmts {
		if s.Keyword != "view" {
			continue
		}

		viewKey, scope, title := ls.parseViewHeader(s)
		if title == "" {
			title = viewKey
		}

		v := model.View{Title: title, Scope: scope, Include: []string{"*"}}
		ls.applyViewBody(&v, s.Body)

		ls.views[viewKey] = v
	}
}

// parseViewHeader parses a "view <key> [of <element>] [title]" statement's
// header arguments and computes its (deduplicated) view key.
func (ls *lc4State) parseViewHeader(s dsl.Stmt) (viewKey, scope, title string) {
	// LikeC4: view <key> [of <element>] { ... }
	// args can be: ["key"], ["key", "of", "element"], or ["key", "of", "element", "title"]
	args := s.Args
	if len(args) > 0 {
		viewKey = args[0]
		args = args[1:]
	}

	// Check for "of" keyword
	if len(args) >= 2 && args[0] == "of" {
		scope = ls.resolveVar(args[1])
		args = args[2:]
	}

	title = strings.Join(args, " ")

	if viewKey == "" {
		viewKey = ls.nextAnonymousViewKey(scope)
	}
	ls.viewKeys[viewKey]++

	return viewKey, scope, title
}

// nextAnonymousViewKey computes a deduplicated view key for a view statement
// with no explicit key, based on its scope (or "view" if unscoped).
func (ls *lc4State) nextAnonymousViewKey(scope string) string {
	baseKey := "view"
	if scope != "" {
		baseKey = scope
	}
	viewKey := baseKey
	if ls.viewKeys[baseKey] > 0 {
		viewKey = fmt.Sprintf("%s_%d", baseKey, ls.viewKeys[baseKey])
	}
	return viewKey
}

// applyViewBody applies a view's body statements (title, description,
// include, exclude) to v.
func (ls *lc4State) applyViewBody(v *model.View, body []dsl.Stmt) {
	for _, bs := range body {
		switch bs.Keyword {
		case "title":
			if len(bs.Args) > 0 {
				v.Title = bs.Args[0]
			}
		case "description":
			if len(bs.Args) > 0 {
				v.Description = bs.Args[0]
			}
		case "include":
			ls.applyViewInclude(v, bs.Args)
		case "exclude":
			for _, arg := range bs.Args {
				v.Exclude = append(v.Exclude, ls.resolveVar(arg))
			}
		}
	}
}

// applyViewInclude sets v.Include from an "include" statement's arguments.
func (ls *lc4State) applyViewInclude(v *model.View, args []string) {
	if len(args) == 1 && args[0] == "*" {
		v.Include = []string{"*"}
		return
	}
	v.Include = nil
	for _, arg := range args {
		if arg == "*" {
			v.Include = []string{"*"}
			return
		}
		v.Include = append(v.Include, ls.resolveVar(arg))
	}
}

func (ls *lc4State) buildRelationships() []model.Relationship {
	var rels []model.Relationship
	for _, pr := range ls.pendingRels {
		fromPath := ls.resolveVar(pr.from)
		toPath := ls.resolveVar(pr.to)
		if fromPath == "" || toPath == "" {
			ls.warnings = append(ls.warnings, fmt.Sprintf("line %d: relationship skipped (unresolved variable)", pr.line))
			continue
		}
		rels = append(rels, model.Relationship{From: fromPath, To: toPath, Label: pr.label})
	}
	return rels
}

func (ls *lc4State) updateSpecWithContainers() {
	for kind, ek := range ls.spec {
		if ls.kindsContainer[kind] {
			ek.Container = true
			ls.spec[kind] = ek
		}
	}
}

// ─── Public API ──────────────────────────────────────────────────────────────

const schemaURL = "https://raw.githubusercontent.com/docToolchain/Bausteinsicht/main/schemas/bausteinsicht.schema.json"

// Import reads the LikeC4 DSL file at path and returns an ImportResult.
func Import(path string) (*importer.ImportResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	_ = filepath.Dir(path) // reserved for future !include support
	return importSource(string(data))
}

// ImportSource parses a LikeC4 DSL string directly (useful for testing).
func ImportSource(src string) (*importer.ImportResult, error) {
	return importSource(src)
}

func importSource(src string) (*importer.ImportResult, error) {
	toks, err := tokenize(src)
	if err != nil {
		return nil, fmt.Errorf("tokenize: %w", err)
	}

	p := &dsl.Parser{Toks: toks}
	stmts, err := dsl.ParseAllStmts(p)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	ls := newLC4State()

	for _, s := range stmts {
		switch s.Keyword {
		case "specification":
			ls.processSpecification(s.Body)
		case "model":
			ls.processModelStmts(s.Body, "", "", ls.elements)
		case "views":
			ls.processViews(s.Body)
		}
	}

	ls.updateSpecWithContainers()

	rels := ls.buildRelationships()

	spec := model.Specification{Elements: make(map[string]model.ElementKind)}
	for k, v := range ls.spec {
		spec.Elements[k] = v
	}

	m := &model.BausteinsichtModel{
		Schema:        schemaURL,
		Specification: spec,
		Model:         ls.elements,
		Relationships: rels,
		Views:         ls.views,
	}
	if m.Relationships == nil {
		m.Relationships = []model.Relationship{}
	}
	if m.Views == nil {
		m.Views = make(map[string]model.View)
	}

	return &importer.ImportResult{Model: m, Warnings: ls.warnings}, nil
}

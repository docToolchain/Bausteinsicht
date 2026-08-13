// Package structurizr parses Structurizr DSL files and converts them to the
// Bausteinsicht model format.
package structurizr

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

// ─── Tokenizer ───────────────────────────────────────────────────────────────
//
// The scanner/parser primitives shared with LikeC4's near-identical DSL
// grammar live in internal/importer/dsl; only the language-specific token
// and statement dispatch stay here.

type (
	token     = dsl.Token
	stmt      = dsl.Stmt
	scanner   = dsl.Scanner
	dslParser = dsl.Parser
)

const (
	tokEOF     = dsl.EOF
	tokNewline = dsl.Newline
	tokString  = dsl.String
	tokIdent   = dsl.Ident
	tokLBrace  = dsl.LBrace
	tokRBrace  = dsl.RBrace
	tokAssign  = dsl.Assign
	tokArrow   = dsl.Arrow
)

func tokenize(src string) ([]token, error) {
	return dsl.Tokenize(src, identStart, scanIdent)
}

// identStart reports whether c can start a Structurizr identifier: a "*" or
// "**" wildcard (so "include *"/"include **" parse correctly), a "!"-prefixed
// reference, or an ordinary identifier.
func identStart(c rune) bool {
	return c == '*' || c == '!' || unicode.IsLetter(c) || c == '_'
}

func scanIdent(s *scanner, line int) (token, error) {
	if c, _ := s.At(0); c == '*' {
		// Consume one or two '*' and return as a single identifier token
		// so that "include *" and "include **" are parsed correctly.
		s.Consume()
		if n, _ := s.At(0); n == '*' {
			s.Consume()
			return token{Kind: tokIdent, Val: "**", Line: line}, nil
		}
		return token{Kind: tokIdent, Val: "*", Line: line}, nil
	}

	var sb strings.Builder
	if c, _ := s.At(0); c == '!' {
		sb.WriteRune(s.Consume())
	}
	dsl.ScanIdentBody(s, &sb)
	return token{Kind: tokIdent, Val: sb.String(), Line: line}, nil
}

// ─── Parser ──────────────────────────────────────────────────────────────────
//
// Structurizr's statement grammar is identical to LikeC4's — both are
// "[var =] keyword args {body}" / "[from] -> to args {body}" — so parsing
// itself lives entirely in the dsl package (dsl.ParseAllStmts).

// ─── Mapper ──────────────────────────────────────────────────────────────────

type kindDef struct {
	kind      string
	notation  string
	container bool
}

// elementKindOrder defines the canonical C4 layer order for specification.elements.
var elementKindOrder = []kindDef{
	{"person", "Person", false},
	{"system", "Software System", true},
	{"container", "Container", true},
	{"component", "Component", false},
}

var structurizrKindMap = map[string]kindDef{
	"person":         elementKindOrder[0],
	"softwareSystem": elementKindOrder[1],
	"container":      elementKindOrder[2],
	"component":      elementKindOrder[3],
}

type pendingRel struct {
	from  string
	to    string
	label string
	line  int
}

type importState struct {
	specAdded   map[string]bool
	spec        map[string]model.ElementKind
	elements    map[string]model.Element
	varToPath   map[string]string
	pendingRels []pendingRel
	views       map[string]model.View
	viewKeys    map[string]int
	warnings    []string
}

func newImportState() *importState {
	return &importState{
		specAdded: make(map[string]bool),
		spec:      make(map[string]model.ElementKind),
		elements:  make(map[string]model.Element),
		varToPath: make(map[string]string),
		views:     make(map[string]model.View),
		viewKeys:  make(map[string]int),
	}
}

func (is *importState) registerKind(kw string) {
	kd := structurizrKindMap[kw]
	if !is.specAdded[kd.kind] {
		is.spec[kd.kind] = model.ElementKind{
			Notation:  kd.notation,
			Container: kd.container,
		}
		is.specAdded[kd.kind] = true
	}
}

func (is *importState) resolveVar(v string) string {
	if p, ok := is.varToPath[v]; ok {
		return p
	}
	return v
}

func (is *importState) processModelStmts(stmts []stmt, parentPath, parentVar string, dest map[string]model.Element) {
	for _, s := range stmts {
		is.processModelStmt(s, parentPath, parentVar, dest)
	}
}

func (is *importState) processModelStmt(s stmt, parentPath, parentVar string, dest map[string]model.Element) {
	if s.IsRel {
		is.addPendingRel(s, parentVar)
		return
	}

	kd, isElement := structurizrKindMap[s.Keyword]
	if !isElement {
		switch s.Keyword {
		case "enterprise", "group":
			is.processModelStmts(s.Body, parentPath, parentVar, dest)
		}
		return
	}

	is.registerKind(s.Keyword)

	key := is.resolveElementKey(s, kd)
	path := key
	if parentPath != "" {
		path = parentPath + "." + key
	}
	is.varToPath[key] = path

	el := buildElementFromStmt(s, kd)

	children := make(map[string]model.Element)
	for _, child := range s.Body {
		is.processModelChild(child, path, key, &el, children)
	}
	if len(children) > 0 {
		el.Children = children
	}
	dest[key] = el
}

// addPendingRel records a relationship statement for later resolution once
// all elements have been discovered, defaulting its source to parentVar
// when the statement omitted an explicit source (e.g. a nested "-> b" inside
// element a's body).
func (is *importState) addPendingRel(s stmt, parentVar string) {
	from := s.RelFrom
	if from == "" {
		from = parentVar
	}
	label := ""
	if len(s.Args) > 0 {
		label = s.Args[0]
	}
	is.pendingRels = append(is.pendingRels, pendingRel{from: from, to: s.RelTo, label: label, line: s.Line})
}

// resolveElementKey returns s's variable name, or a slugified fallback
// derived from its title (or kind if untitled), warning when no variable
// name was given in the DSL.
func (is *importState) resolveElementKey(s stmt, kd kindDef) string {
	if s.VarName != "" {
		return s.VarName
	}
	key := kd.kind
	if len(s.Args) > 0 {
		key = dsl.Slugify(s.Args[0])
	}
	is.warnings = append(is.warnings, fmt.Sprintf("line %d: element has no variable name, using %q", s.Line, key))
	return key
}

// buildElementFromStmt constructs the base model.Element from s's title,
// description, and (for containers/components) technology arguments.
func buildElementFromStmt(s stmt, kd kindDef) model.Element {
	el := model.Element{Kind: kd.kind}
	if len(s.Args) > 0 {
		el.Title = s.Args[0]
	}
	if len(s.Args) > 1 {
		el.Description = s.Args[1]
	}
	if (kd.kind == "container" || kd.kind == "component") && len(s.Args) > 2 {
		el.Technology = s.Args[2]
	}
	return el
}

// processModelChild applies a single child statement of an element's body:
// a nested relationship, a nested element, or one of the description/
// technology/tags/properties fields.
func (is *importState) processModelChild(child stmt, path, key string, el *model.Element, children map[string]model.Element) {
	switch {
	case child.IsRel:
		is.addPendingRel(child, key)
	case structurizrKindMap[child.Keyword].kind != "":
		is.processModelStmt(child, path, key, children)
	case child.Keyword == "description" && len(child.Args) > 0:
		el.Description = child.Args[0]
	case child.Keyword == "technology" && len(child.Args) > 0:
		el.Technology = child.Args[0]
	case child.Keyword == "tags":
		el.Tags = child.Args
	case child.Keyword == "properties":
		el.Metadata = parseProperties(child.Body)
	}
}

func (is *importState) processViewsStmts(stmts []stmt) {
	for _, s := range stmts {
		if !is.isSupportedViewKeyword(s) {
			continue
		}

		scope := ""
		if s.Keyword != "systemLandscape" && len(s.Args) > 0 {
			scope = is.resolveVar(s.Args[0])
		}

		titleArgs := s.Args
		if scope != "" {
			titleArgs = s.Args[1:]
		}
		title := strings.Join(titleArgs, " ")

		viewKey := is.nextViewKey(s.Keyword, scope)
		if title == "" {
			title = viewKey
		}

		v := model.View{Title: title, Scope: scope, Include: []string{"*"}}
		is.applyViewBodyStmts(&v, s.Body, viewKey)
		expandScopeWildcardInclude(&v, s.Keyword, scope)

		is.views[viewKey] = v
	}
}

// isSupportedViewKeyword reports whether s is a supported view statement,
// warning and returning false for recognized-but-unsupported view types.
func (is *importState) isSupportedViewKeyword(s stmt) bool {
	switch s.Keyword {
	case "systemContext", "container", "component", "systemLandscape":
		return true
	case "filtered", "dynamic", "deployment":
		is.warnings = append(is.warnings, fmt.Sprintf("line %d: %s view not supported, skipped", s.Line, s.Keyword))
		return false
	default:
		return false
	}
}

// nextViewKey computes a view's map key from its keyword/scope, deduplicating
// repeated combinations with a numeric suffix.
func (is *importState) nextViewKey(keyword, scope string) string {
	baseKey := keyword
	if scope != "" {
		baseKey = scope
	}
	viewKey := baseKey
	if is.viewKeys[baseKey] > 0 {
		viewKey = fmt.Sprintf("%s_%d", baseKey, is.viewKeys[baseKey])
	}
	is.viewKeys[baseKey]++
	return viewKey
}

// applyViewBodyStmts applies a view's body statements (include, exclude,
// title, description, autoLayout) to v.
func (is *importState) applyViewBodyStmts(v *model.View, body []stmt, viewKey string) {
	for _, bs := range body {
		switch bs.Keyword {
		case "include":
			is.applyViewInclude(v, bs.Args)
		case "exclude":
			for _, arg := range bs.Args {
				v.Exclude = append(v.Exclude, is.resolveVar(arg))
			}
		case "title":
			if len(bs.Args) > 0 {
				v.Title = bs.Args[0]
			}
		case "description":
			if len(bs.Args) > 0 {
				v.Description = bs.Args[0]
			}
		case "autoLayout":
			is.applyAutoLayout(v, bs, viewKey)
		}
	}
}

// applyViewInclude sets v.Include from an "include" statement's arguments,
// resolving variables and treating "*" as a wildcard that overrides any
// prior explicit includes.
func (is *importState) applyViewInclude(v *model.View, args []string) {
	if len(args) == 1 && args[0] == "*" {
		v.Include = []string{"*"}
		return
	}
	v.Include = nil
	for _, arg := range args {
		if arg != "*" {
			v.Include = append(v.Include, is.resolveVar(arg))
			continue
		}
		v.Include = []string{"*"}
		return
	}
}

// applyAutoLayout maps Structurizr's autoLayout to Bausteinsicht's "layered"
// layout, since there is no direct equivalent (no direction-aware layout
// engine); "layered" is the closest match and Bausteinsicht's own default.
// "auto" is not a valid Layout value (see model.validate) and would fail
// validation on the very model this importer just wrote.
func (is *importState) applyAutoLayout(v *model.View, bs stmt, viewKey string) {
	v.Layout = "layered"
	if len(bs.Args) > 0 {
		is.warnings = append(is.warnings, fmt.Sprintf(
			"line %d: view %q: autoLayout direction %q not preserved, mapped to layout: \"layered\"",
			bs.Line, viewKey, strings.Join(bs.Args, " ")))
	}
}

// expandScopeWildcardInclude expands a scoped view's "include *" pattern to
// explicitly include scope children, since Bausteinsicht's "*" only matches
// top-level IDs (no dots) while Structurizr's "include *" on a scoped view
// means all elements visible in that scope.
func expandScopeWildcardInclude(v *model.View, keyword, scope string) {
	if scope == "" || len(v.Include) != 1 || v.Include[0] != "*" {
		return
	}
	switch keyword {
	case "container":
		v.Include = []string{"*", scope + ".*"}
	case "component":
		parts := strings.Split(scope, ".")
		if len(parts) > 1 {
			parentScope := strings.Join(parts[:len(parts)-1], ".")
			v.Include = []string{"*", parentScope + ".*", scope + ".*"}
		} else {
			v.Include = []string{"*", scope + ".*"}
		}
	}
}

func (is *importState) buildRelationships() []model.Relationship {
	var rels []model.Relationship
	for _, pr := range is.pendingRels {
		fromPath := is.resolveVar(pr.from)
		toPath := is.resolveVar(pr.to)
		if fromPath == "" || toPath == "" {
			is.warnings = append(is.warnings, fmt.Sprintf("line %d: relationship skipped (unresolved variable)", pr.line))
			continue
		}
		rels = append(rels, model.Relationship{From: fromPath, To: toPath, Label: pr.label})
	}
	return rels
}

func parseProperties(body []stmt) map[string]string {
	m := make(map[string]string)
	for _, s := range body {
		if s.Keyword != "" && len(s.Args) > 0 {
			m[s.Keyword] = s.Args[0]
		}
	}
	return m
}

// ─── Public API ──────────────────────────────────────────────────────────────

const schemaURL = "https://raw.githubusercontent.com/docToolchain/Bausteinsicht/main/schemas/bausteinsicht.schema.json"

// ImportSource parses a Structurizr DSL string directly (useful for testing).
func ImportSource(src string) (*importer.ImportResult, error) {
	return importSource(src)
}

// Import reads the Structurizr DSL file at path and returns an ImportResult.
func Import(path string) (*importer.ImportResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	baseDir := filepath.Dir(path)
	src, includeWarnings := resolveIncludes(string(data), baseDir, map[string]bool{})
	result, err := importSource(src)
	if err != nil {
		return nil, err
	}
	result.Warnings = append(includeWarnings, result.Warnings...)
	return result, nil
}

func resolveIncludes(src, baseDir string, visited map[string]bool) (string, []string) {
	var warnings []string
	var out strings.Builder
	absDirBase, _ := filepath.Abs(baseDir)
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "!include ") {
			out.WriteString(line)
			out.WriteByte('\n')
			continue
		}
		includePath := strings.TrimSpace(trimmed[len("!include "):])
		content, w := resolveOneInclude(includePath, baseDir, absDirBase, visited)
		warnings = append(warnings, w...)
		out.WriteString(content)
		out.WriteByte('\n')
	}
	return out.String(), warnings
}

// resolveOneInclude resolves a single "!include <path>" directive, returning
// its expanded content (recursively resolving nested includes) — or an
// empty string plus a single warning if the include could not be honored
// (HTTP URL, path traversal, circular include, or unreadable file).
func resolveOneInclude(includePath, baseDir, absDirBase string, visited map[string]bool) (content string, warnings []string) {
	if strings.HasPrefix(includePath, "http://") || strings.HasPrefix(includePath, "https://") {
		return "", []string{"!include: HTTP includes not supported, skipped: " + includePath}
	}

	cleanedPath := filepath.Clean(includePath)
	fullPath := filepath.Join(baseDir, cleanedPath)
	absFullPath, _ := filepath.Abs(fullPath)

	// Verify that the resolved path is within baseDir (prevent path traversal).
	// Use filepath.Rel to check if the path escapes the base directory via .. sequences.
	relPath, err := filepath.Rel(absDirBase, absFullPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return "", []string{"!include: path traversal rejected: " + includePath}
	}

	if visited[absFullPath] {
		return "", []string{"!include: circular include ignored: " + includePath}
	}

	data, err := os.ReadFile(absFullPath)
	if err != nil {
		return "", []string{fmt.Sprintf("!include: cannot read %s: %v", includePath, err)}
	}

	newVisited := make(map[string]bool, len(visited)+1)
	for k, v := range visited {
		newVisited[k] = v
	}
	newVisited[absFullPath] = true

	return resolveIncludes(string(data), filepath.Dir(absFullPath), newVisited)
}

func importSource(src string) (*importer.ImportResult, error) {
	toks, err := tokenize(src)
	if err != nil {
		return nil, fmt.Errorf("tokenize: %w", err)
	}

	p := &dslParser{Toks: toks}
	stmts, err := dsl.ParseAllStmts(p)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	is := newImportState()

	modelStmts, viewsStmts := extractModelAndViewsStmts(stmts)
	is.processModelStmts(modelStmts, "", "", is.elements)
	if len(viewsStmts) > 0 {
		is.processViewsStmts(viewsStmts)
	}

	rels := is.buildRelationships()
	m := is.buildResultModel(rels)

	return &importer.ImportResult{Model: m, Warnings: is.warnings}, nil
}

// extractModelAndViewsStmts finds the top-level "model" and "views" blocks
// among stmts, which may either be nested inside a "workspace" block or
// appear directly at the top level.
func extractModelAndViewsStmts(stmts []stmt) (modelStmts, viewsStmts []stmt) {
	for _, s := range stmts {
		switch s.Keyword {
		case "workspace":
			for _, ws := range s.Body {
				switch ws.Keyword {
				case "model":
					modelStmts = ws.Body
				case "views":
					viewsStmts = ws.Body
				}
			}
		case "model":
			modelStmts = s.Body
		case "views":
			viewsStmts = s.Body
		}
	}
	return modelStmts, viewsStmts
}

// buildResultModel assembles the final BausteinsichtModel from accumulated
// import state and resolved relationships.
func (is *importState) buildResultModel(rels []model.Relationship) *model.BausteinsichtModel {
	spec := model.Specification{Elements: make(map[string]model.ElementKind)}
	for _, kd := range elementKindOrder {
		if ek, ok := is.spec[kd.kind]; ok {
			spec.Elements[kd.kind] = ek
		}
	}

	m := &model.BausteinsichtModel{
		Schema:        schemaURL,
		Specification: spec,
		Model:         is.elements,
		Relationships: rels,
		Views:         is.views,
	}
	if m.Relationships == nil {
		m.Relationships = []model.Relationship{}
	}
	if m.Views == nil {
		m.Views = make(map[string]model.View)
	}
	return m
}

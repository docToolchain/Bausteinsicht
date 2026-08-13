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
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// ─── Tokenizer ───────────────────────────────────────────────────────────────

type tokKind int

const (
	tokEOF     tokKind = iota
	tokNewline         // statement separator — emitted for each (group of) newline(s)
	tokString
	tokIdent
	tokLBrace
	tokRBrace
	tokAssign
	tokArrow
)

type token struct {
	kind tokKind
	val  string
	line int
}

type scanner struct {
	src  []rune
	pos  int
	line int
}

func tokenize(src string) ([]token, error) {
	s := &scanner{src: []rune(src), line: 1}
	var toks []token
	for {
		tok, err := s.next()
		if err != nil {
			return nil, err
		}
		toks = append(toks, tok)
		if tok.kind == tokEOF {
			break
		}
	}
	return toks, nil
}

func (s *scanner) at(offset int) (rune, bool) {
	i := s.pos + offset
	if i >= len(s.src) {
		return 0, false
	}
	return s.src[i], true
}

func (s *scanner) consume() rune {
	r := s.src[s.pos]
	s.pos++
	if r == '\n' {
		s.line++
	}
	return r
}

func (s *scanner) next() (token, error) {
	// Skip horizontal whitespace and handle comments.
	// Newlines are NOT skipped here — they are emitted as tokNewline.
	if err := s.skipWhitespaceAndComments(); err != nil {
		return token{}, err
	}

	c, ok := s.at(0)
	if !ok {
		return token{kind: tokEOF, line: s.line}, nil
	}
	line := s.line

	// Collapse consecutive newlines into a single tokNewline.
	if c == '\n' {
		s.skipNewlines()
		return token{kind: tokNewline, line: line}, nil
	}

	return s.nextSymbolOrIdent(c, line)
}

// skipWhitespaceAndComments advances past horizontal whitespace and // and
// /* */ comments. Newlines are NOT skipped — next emits them as tokNewline.
func (s *scanner) skipWhitespaceAndComments() error {
	for {
		c, ok := s.at(0)
		if !ok {
			return nil
		}
		if c == ' ' || c == '\t' || c == '\r' {
			s.consume()
			continue
		}
		if c == '/' {
			consumed, err := s.skipComment()
			if err != nil {
				return err
			}
			if consumed {
				continue
			}
		}
		return nil
	}
}

// skipComment consumes a // or /* */ comment starting at the current
// position (c == '/'), if any, and reports whether one was consumed.
func (s *scanner) skipComment() (bool, error) {
	switch n, _ := s.at(1); n {
	case '/':
		s.skipLineComment()
		return true, nil
	case '*':
		if err := s.skipBlockComment(); err != nil {
			return false, err
		}
		return true, nil
	default:
		return false, nil
	}
}

// skipLineComment consumes to end of line (leaving \n for next call).
func (s *scanner) skipLineComment() {
	for {
		ch, ok := s.at(0)
		if !ok || ch == '\n' {
			return
		}
		s.consume()
	}
}

// skipBlockComment consumes a /* ... */ block comment starting at the
// current position (the leading "/*").
func (s *scanner) skipBlockComment() error {
	s.consume()
	s.consume()
	for {
		ch, ok := s.at(0)
		if !ok {
			return fmt.Errorf("unterminated block comment")
		}
		s.consume()
		if ch == '*' {
			if nn, _ := s.at(0); nn == '/' {
				s.consume()
				return nil
			}
		}
	}
}

// skipNewlines consumes consecutive '\n' characters.
func (s *scanner) skipNewlines() {
	for {
		ch, ok := s.at(0)
		if !ok || ch != '\n' {
			return
		}
		s.consume()
	}
}

// nextSymbolOrIdent scans the next symbol, string, or identifier token,
// given the already-peeked lookahead character c at line.
func (s *scanner) nextSymbolOrIdent(c rune, line int) (token, error) {
	switch {
	case c == '{':
		s.consume()
		return token{kind: tokLBrace, val: "{", line: line}, nil
	case c == '}':
		s.consume()
		return token{kind: tokRBrace, val: "}", line: line}, nil
	case c == '=':
		s.consume()
		return token{kind: tokAssign, val: "=", line: line}, nil
	case c == '-':
		if n, _ := s.at(1); n == '>' {
			s.consume()
			s.consume()
			return token{kind: tokArrow, val: "->", line: line}, nil
		}
		s.consume()
		return s.next()
	case c == '"':
		return s.scanString(line)
	case c == '*':
		// Consume one or two '*' and return as a single identifier token
		// so that "include *" and "include **" are parsed correctly.
		s.consume()
		if n, _ := s.at(0); n == '*' {
			s.consume()
			return token{kind: tokIdent, val: "**", line: line}, nil
		}
		return token{kind: tokIdent, val: "*", line: line}, nil
	case c == '!' || unicode.IsLetter(c) || c == '_':
		return s.scanIdent(line)
	default:
		s.consume()
		return s.next()
	}
}

func (s *scanner) scanString(line int) (token, error) {
	s.consume()
	var sb strings.Builder
	for {
		c, ok := s.at(0)
		if !ok {
			return token{}, fmt.Errorf("line %d: unterminated string", line)
		}
		if c == '"' {
			s.consume()
			break
		}
		if c == '\\' {
			s.consume()
			esc, ok := s.at(0)
			if !ok {
				return token{}, fmt.Errorf("line %d: EOF in string escape", line)
			}
			s.consume()
			switch esc {
			case '"', '\\':
				sb.WriteRune(esc)
			case 'n':
				sb.WriteRune('\n')
			default:
				sb.WriteRune('\\')
				sb.WriteRune(esc)
			}
			continue
		}
		sb.WriteRune(s.consume())
	}
	return token{kind: tokString, val: sb.String(), line: line}, nil
}

func (s *scanner) scanIdent(line int) (token, error) {
	var sb strings.Builder
	if c, _ := s.at(0); c == '!' {
		sb.WriteRune(s.consume())
	}
	for {
		c, ok := s.at(0)
		if !ok {
			break
		}
		if c == '-' {
			if n, _ := s.at(1); n == '>' {
				break
			}
			sb.WriteRune(s.consume())
			continue
		}
		if unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_' || c == '.' || c == '/' || c == ':' {
			sb.WriteRune(s.consume())
			continue
		}
		break
	}
	return token{kind: tokIdent, val: sb.String(), line: line}, nil
}

// ─── Parser ──────────────────────────────────────────────────────────────────

// stmt represents one parsed statement in the DSL.
type stmt struct {
	line    int
	varName string
	keyword string
	args    []string
	isRel   bool
	relFrom string
	relTo   string
	body    []stmt
}

type dslParser struct {
	toks []token
	pos  int
}

func (p *dslParser) peek() token {
	if p.pos >= len(p.toks) {
		return token{kind: tokEOF}
	}
	return p.toks[p.pos]
}

func (p *dslParser) advance() token {
	t := p.peek()
	if t.kind != tokEOF {
		p.pos++
	}
	return t
}

func (p *dslParser) skipNewlines() {
	for p.peek().kind == tokNewline {
		p.advance()
	}
}

func (p *dslParser) parseAll() ([]stmt, error) {
	return p.parseStmts(false)
}

func (p *dslParser) parseStmts(inBlock bool) ([]stmt, error) {
	var stmts []stmt
	for {
		p.skipNewlines()
		tok := p.peek()
		if tok.kind == tokEOF {
			break
		}
		if inBlock && tok.kind == tokRBrace {
			break
		}
		s, err := p.parseOneStmt()
		if err != nil {
			return nil, err
		}
		if s != nil {
			stmts = append(stmts, *s)
		}
	}
	return stmts, nil
}

func (p *dslParser) parseBlock() ([]stmt, error) {
	if p.peek().kind != tokLBrace {
		return nil, nil
	}
	p.advance() // {
	p.skipNewlines()
	stmts, err := p.parseStmts(true)
	if err != nil {
		return nil, err
	}
	if p.peek().kind == tokRBrace {
		p.advance()
	}
	return stmts, nil
}

// optBlock skips newlines then reads a block if the next token is {.
func (p *dslParser) optBlock(s *stmt) error {
	p.skipNewlines()
	if p.peek().kind == tokLBrace {
		body, err := p.parseBlock()
		if err != nil {
			return err
		}
		s.body = body
	}
	return nil
}

func (p *dslParser) parseOneStmt() (*stmt, error) {
	tok := p.peek()
	if tok.kind == tokEOF || tok.kind == tokRBrace {
		return nil, nil
	}

	line := tok.line

	if tok.kind == tokArrow {
		p.advance()
		to := p.advance()
		return p.finishStmt(&stmt{line: line, isRel: true, relTo: to.val, args: p.collectArgs()})
	}

	if tok.kind == tokLBrace {
		if _, err := p.parseBlock(); err != nil {
			return nil, err
		}
		return nil, nil
	}

	if tok.kind != tokIdent && tok.kind != tokString {
		p.advance()
		return nil, nil
	}

	p.advance()

	switch p.peek().kind {
	case tokAssign:
		p.advance()
		kw := p.advance()
		return p.finishStmt(&stmt{line: line, varName: tok.val, keyword: kw.val, args: p.collectArgs()})

	case tokArrow:
		p.advance()
		to := p.advance()
		return p.finishStmt(&stmt{line: line, isRel: true, relFrom: tok.val, relTo: to.val, args: p.collectArgs()})

	default:
		return p.finishStmt(&stmt{line: line, keyword: tok.val, args: p.collectArgs()})
	}
}

// finishStmt parses s's optional trailing "{ ... }" block, if present, and
// returns s (or the error from parsing the block).
func (p *dslParser) finishStmt(s *stmt) (*stmt, error) {
	if err := p.optBlock(s); err != nil {
		return nil, err
	}
	return s, nil
}

func (p *dslParser) collectArgs() []string {
	var args []string
	for {
		k := p.peek().kind
		if k == tokString || k == tokIdent {
			args = append(args, p.advance().val)
		} else {
			break
		}
	}
	return args
}

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
	if s.isRel {
		is.addPendingRel(s, parentVar)
		return
	}

	kd, isElement := structurizrKindMap[s.keyword]
	if !isElement {
		switch s.keyword {
		case "enterprise", "group":
			is.processModelStmts(s.body, parentPath, parentVar, dest)
		}
		return
	}

	is.registerKind(s.keyword)

	key := is.resolveElementKey(s, kd)
	path := key
	if parentPath != "" {
		path = parentPath + "." + key
	}
	is.varToPath[key] = path

	el := buildElementFromStmt(s, kd)

	children := make(map[string]model.Element)
	for _, child := range s.body {
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
	from := s.relFrom
	if from == "" {
		from = parentVar
	}
	label := ""
	if len(s.args) > 0 {
		label = s.args[0]
	}
	is.pendingRels = append(is.pendingRels, pendingRel{from: from, to: s.relTo, label: label, line: s.line})
}

// resolveElementKey returns s's variable name, or a slugified fallback
// derived from its title (or kind if untitled), warning when no variable
// name was given in the DSL.
func (is *importState) resolveElementKey(s stmt, kd kindDef) string {
	if s.varName != "" {
		return s.varName
	}
	key := kd.kind
	if len(s.args) > 0 {
		key = slugify(s.args[0])
	}
	is.warnings = append(is.warnings, fmt.Sprintf("line %d: element has no variable name, using %q", s.line, key))
	return key
}

// buildElementFromStmt constructs the base model.Element from s's title,
// description, and (for containers/components) technology arguments.
func buildElementFromStmt(s stmt, kd kindDef) model.Element {
	el := model.Element{Kind: kd.kind}
	if len(s.args) > 0 {
		el.Title = s.args[0]
	}
	if len(s.args) > 1 {
		el.Description = s.args[1]
	}
	if (kd.kind == "container" || kd.kind == "component") && len(s.args) > 2 {
		el.Technology = s.args[2]
	}
	return el
}

// processModelChild applies a single child statement of an element's body:
// a nested relationship, a nested element, or one of the description/
// technology/tags/properties fields.
func (is *importState) processModelChild(child stmt, path, key string, el *model.Element, children map[string]model.Element) {
	switch {
	case child.isRel:
		is.addPendingRel(child, key)
	case structurizrKindMap[child.keyword].kind != "":
		is.processModelStmt(child, path, key, children)
	case child.keyword == "description" && len(child.args) > 0:
		el.Description = child.args[0]
	case child.keyword == "technology" && len(child.args) > 0:
		el.Technology = child.args[0]
	case child.keyword == "tags":
		el.Tags = child.args
	case child.keyword == "properties":
		el.Metadata = parseProperties(child.body)
	}
}

func (is *importState) processViewsStmts(stmts []stmt) {
	for _, s := range stmts {
		if !is.isSupportedViewKeyword(s) {
			continue
		}

		scope := ""
		if s.keyword != "systemLandscape" && len(s.args) > 0 {
			scope = is.resolveVar(s.args[0])
		}

		titleArgs := s.args
		if scope != "" {
			titleArgs = s.args[1:]
		}
		title := strings.Join(titleArgs, " ")

		viewKey := is.nextViewKey(s.keyword, scope)
		if title == "" {
			title = viewKey
		}

		v := model.View{Title: title, Scope: scope, Include: []string{"*"}}
		is.applyViewBodyStmts(&v, s.body, viewKey)
		expandScopeWildcardInclude(&v, s.keyword, scope)

		is.views[viewKey] = v
	}
}

// isSupportedViewKeyword reports whether s is a supported view statement,
// warning and returning false for recognized-but-unsupported view types.
func (is *importState) isSupportedViewKeyword(s stmt) bool {
	switch s.keyword {
	case "systemContext", "container", "component", "systemLandscape":
		return true
	case "filtered", "dynamic", "deployment":
		is.warnings = append(is.warnings, fmt.Sprintf("line %d: %s view not supported, skipped", s.line, s.keyword))
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
		switch bs.keyword {
		case "include":
			is.applyViewInclude(v, bs.args)
		case "exclude":
			for _, arg := range bs.args {
				v.Exclude = append(v.Exclude, is.resolveVar(arg))
			}
		case "title":
			if len(bs.args) > 0 {
				v.Title = bs.args[0]
			}
		case "description":
			if len(bs.args) > 0 {
				v.Description = bs.args[0]
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
	if len(bs.args) > 0 {
		is.warnings = append(is.warnings, fmt.Sprintf(
			"line %d: view %q: autoLayout direction %q not preserved, mapped to layout: \"layered\"",
			bs.line, viewKey, strings.Join(bs.args, " ")))
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
		if s.keyword != "" && len(s.args) > 0 {
			m[s.keyword] = s.args[0]
		}
	}
	return m
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var sb strings.Builder
	prevUnderscore := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
			prevUnderscore = false
		} else if !prevUnderscore && sb.Len() > 0 {
			sb.WriteRune('_')
			prevUnderscore = true
		}
	}
	result := strings.TrimRight(sb.String(), "_")
	if result == "" {
		return "element"
	}
	return result
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

	p := &dslParser{toks: toks}
	stmts, err := p.parseAll()
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
		switch s.keyword {
		case "workspace":
			for _, ws := range s.body {
				switch ws.keyword {
				case "model":
					modelStmts = ws.body
				case "views":
					viewsStmts = ws.body
				}
			}
		case "model":
			modelStmts = s.body
		case "views":
			viewsStmts = s.body
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

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
	"github.com/docToolchain/Bausteinsicht/internal/model"
)

// ─── Tokenizer (identical grammar to Structurizr) ────────────────────────────

type tokKind int

const (
	tokEOF tokKind = iota
	tokNewline
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
	for {
		c, ok := s.at(0)
		if !ok {
			return token{kind: tokEOF, line: s.line}, nil
		}
		if c == ' ' || c == '\t' || c == '\r' {
			s.consume()
			continue
		}
		if c == '/' {
			n, _ := s.at(1)
			if n == '/' {
				for {
					ch, ok := s.at(0)
					if !ok || ch == '\n' {
						break
					}
					s.consume()
				}
				continue
			}
			if n == '*' {
				s.consume()
				s.consume()
				for {
					ch, ok := s.at(0)
					if !ok {
						return token{}, fmt.Errorf("unterminated block comment")
					}
					s.consume()
					if ch == '*' {
						if nn, _ := s.at(0); nn == '/' {
							s.consume()
							break
						}
					}
				}
				continue
			}
		}
		break
	}

	c, ok := s.at(0)
	if !ok {
		return token{kind: tokEOF, line: s.line}, nil
	}
	line := s.line

	if c == '\n' {
		for {
			ch, ok := s.at(0)
			if !ok || ch != '\n' {
				break
			}
			s.consume()
		}
		return token{kind: tokNewline, line: line}, nil
	}

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
	case unicode.IsLetter(c) || c == '_':
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
	p.advance()
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
		s := &stmt{line: line, isRel: true, relTo: to.val, args: p.collectArgs()}
		if err := p.optBlock(s); err != nil {
			return nil, err
		}
		return s, nil
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
		s := &stmt{line: line, varName: tok.val, keyword: kw.val, args: p.collectArgs()}
		if err := p.optBlock(s); err != nil {
			return nil, err
		}
		return s, nil

	case tokArrow:
		p.advance()
		to := p.advance()
		s := &stmt{line: line, isRel: true, relFrom: tok.val, relTo: to.val, args: p.collectArgs()}
		if err := p.optBlock(s); err != nil {
			return nil, err
		}
		return s, nil

	default:
		s := &stmt{line: line, keyword: tok.val, args: p.collectArgs()}
		if err := p.optBlock(s); err != nil {
			return nil, err
		}
		return s, nil
	}
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

type lc4State struct {
	kinds           map[string]bool // known element kind names from specification
	kindsContainer  map[string]bool // kinds that have children in the model
	spec            map[string]model.ElementKind
	elements        map[string]model.Element
	varToPath       map[string]string
	pendingRels     []pendingRel
	views           map[string]model.View
	viewKeys        map[string]int
	warnings        []string
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

func (ls *lc4State) processSpecification(stmts []stmt) {
	for _, s := range stmts {
		switch s.keyword {
		case "element":
			if len(s.args) == 0 {
				continue
			}
			kindName := s.args[0]
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

func (ls *lc4State) processModelStmts(stmts []stmt, parentPath, parentVar string, dest map[string]model.Element) {
	for _, s := range stmts {
		ls.processModelStmt(s, parentPath, parentVar, dest)
	}
}

func (ls *lc4State) processModelStmt(s stmt, parentPath, parentVar string, dest map[string]model.Element) {
	if s.isRel {
		from := s.relFrom
		if from == "" {
			from = parentVar
		}
		label := ""
		if len(s.args) > 0 {
			label = s.args[0]
		}
		ls.pendingRels = append(ls.pendingRels, pendingRel{from: from, to: s.relTo, label: label, line: s.line})
		return
	}

	if !ls.kinds[s.keyword] {
		// Not a known kind — treat as property inside element body
		return
	}

	key := s.varName
	if key == "" {
		if len(s.args) > 0 {
			key = slugify(s.args[0])
		} else {
			key = s.keyword
		}
		ls.warnings = append(ls.warnings, fmt.Sprintf("line %d: element has no variable name, using %q", s.line, key))
	}

	path := key
	if parentPath != "" {
		path = parentPath + "." + key
	}
	ls.varToPath[key] = path

	el := model.Element{Kind: s.keyword}
	if len(s.args) > 0 {
		el.Title = s.args[0]
	}

	children := make(map[string]model.Element)
	for _, child := range s.body {
		switch {
		case child.isRel:
			from := child.relFrom
			if from == "" {
				from = key
			}
			label := ""
			if len(child.args) > 0 {
				label = child.args[0]
			}
			ls.pendingRels = append(ls.pendingRels, pendingRel{from: from, to: child.relTo, label: label, line: child.line})
		case ls.kinds[child.keyword]:
			ls.processModelStmt(child, path, key, children)
		case child.keyword == "description" && len(child.args) > 0:
			el.Description = child.args[0]
		case child.keyword == "technology" && len(child.args) > 0:
			el.Technology = child.args[0]
		case child.keyword == "title" && len(child.args) > 0:
			el.Title = child.args[0]
		case child.keyword == "tags":
			el.Tags = child.args
		}
	}

	if len(children) > 0 {
		el.Children = children
		ls.kindsContainer[s.keyword] = true
	}

	dest[key] = el
}

func (ls *lc4State) processViews(stmts []stmt) {
	for _, s := range stmts {
		if s.keyword != "view" {
			continue
		}

		// LikeC4: view <key> [of <element>] { ... }
		// args can be: ["key"], ["key", "of", "element"], or ["key", "of", "element", "title"]
		viewKey := ""
		scope := ""
		title := ""

		args := s.args
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
			baseKey := "view"
			if scope != "" {
				baseKey = scope
			}
			viewKey = baseKey
			if ls.viewKeys[baseKey] > 0 {
				viewKey = fmt.Sprintf("%s_%d", baseKey, ls.viewKeys[baseKey])
			}
		}
		ls.viewKeys[viewKey]++

		if title == "" {
			title = viewKey
		}

		v := model.View{Title: title, Scope: scope, Include: []string{"*"}}

		for _, bs := range s.body {
			switch bs.keyword {
			case "title":
				if len(bs.args) > 0 {
					v.Title = bs.args[0]
				}
			case "description":
				if len(bs.args) > 0 {
					v.Description = bs.args[0]
				}
			case "include":
				if len(bs.args) == 1 && bs.args[0] == "*" {
					v.Include = []string{"*"}
				} else {
					v.Include = nil
					for _, arg := range bs.args {
						if arg == "*" {
							v.Include = []string{"*"}
							break
						}
						v.Include = append(v.Include, ls.resolveVar(arg))
					}
				}
			case "exclude":
				for _, arg := range bs.args {
					v.Exclude = append(v.Exclude, ls.resolveVar(arg))
				}
			}
		}

		ls.views[viewKey] = v
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

const schemaURL = "https://raw.githubusercontent.com/docToolchain/Bausteinsicht/main/schema/bausteinsicht.schema.json"

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

	p := &dslParser{toks: toks}
	stmts, err := p.parseAll()
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	ls := newLC4State()

	for _, s := range stmts {
		switch s.keyword {
		case "specification":
			ls.processSpecification(s.body)
		case "model":
			ls.processModelStmts(s.body, "", "", ls.elements)
		case "views":
			ls.processViews(s.body)
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

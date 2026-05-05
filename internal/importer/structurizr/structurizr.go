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
	tokEOF tokKind = iota
	tokNewline // statement separator — emitted for each (group of) newline(s)
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
				// Line comment: consume to end of line (leave \n for next call)
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

	// Collapse consecutive newlines into a single tokNewline.
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
		from := s.relFrom
		if from == "" {
			from = parentVar
		}
		label := ""
		if len(s.args) > 0 {
			label = s.args[0]
		}
		is.pendingRels = append(is.pendingRels, pendingRel{from: from, to: s.relTo, label: label, line: s.line})
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

	key := s.varName
	if key == "" {
		if len(s.args) > 0 {
			key = slugify(s.args[0])
		} else {
			key = kd.kind
		}
		is.warnings = append(is.warnings, fmt.Sprintf("line %d: element has no variable name, using %q", s.line, key))
	}

	path := key
	if parentPath != "" {
		path = parentPath + "." + key
	}
	is.varToPath[key] = path

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
			is.pendingRels = append(is.pendingRels, pendingRel{from: from, to: child.relTo, label: label, line: child.line})
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

	if len(children) > 0 {
		el.Children = children
	}
	dest[key] = el
}

func (is *importState) processViewsStmts(stmts []stmt) {
	for _, s := range stmts {
		switch s.keyword {
		case "systemContext", "container", "component", "systemLandscape":
		case "filtered", "dynamic", "deployment":
			is.warnings = append(is.warnings, fmt.Sprintf("line %d: %s view not supported, skipped", s.line, s.keyword))
			continue
		default:
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

		baseKey := s.keyword
		if scope != "" {
			baseKey = scope
		}
		viewKey := baseKey
		if is.viewKeys[baseKey] > 0 {
			viewKey = fmt.Sprintf("%s_%d", baseKey, is.viewKeys[baseKey])
		}
		is.viewKeys[baseKey]++

		if title == "" {
			title = viewKey
		}

		v := model.View{Title: title, Scope: scope, Include: []string{"*"}}

		for _, bs := range s.body {
			switch bs.keyword {
			case "include":
				if len(bs.args) == 1 && bs.args[0] == "*" {
					v.Include = []string{"*"}
				} else {
					v.Include = nil
					for _, arg := range bs.args {
						if arg != "*" {
							v.Include = append(v.Include, is.resolveVar(arg))
						} else {
							v.Include = []string{"*"}
							break
						}
					}
				}
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
				v.Layout = "auto"
			}
		}

		is.views[viewKey] = v
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

const schemaURL = "https://raw.githubusercontent.com/docToolchain/Bausteinsicht/main/schema/bausteinsicht.schema.json"

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
		if strings.HasPrefix(trimmed, "!include ") {
			includePath := strings.TrimSpace(trimmed[len("!include "):])
			if strings.HasPrefix(includePath, "http://") || strings.HasPrefix(includePath, "https://") {
				warnings = append(warnings, "!include: HTTP includes not supported, skipped: "+includePath)
				out.WriteByte('\n')
				continue
			}
			cleanedPath := filepath.Clean(includePath)
			fullPath := filepath.Join(baseDir, cleanedPath)
			absFullPath, _ := filepath.Abs(fullPath)

			// Verify that the resolved path is within baseDir (prevent path traversal).
			// Use filepath.Rel to check if the path escapes the base directory via .. sequences.
			relPath, err := filepath.Rel(absDirBase, absFullPath)
			if err != nil || strings.HasPrefix(relPath, "..") {
				warnings = append(warnings, "!include: path traversal rejected: "+includePath)
				out.WriteByte('\n')
				continue
			}

			if visited[absFullPath] {
				warnings = append(warnings, "!include: circular include ignored: "+includePath)
				out.WriteByte('\n')
				continue
			}
			data, err := os.ReadFile(absFullPath)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("!include: cannot read %s: %v", includePath, err))
				out.WriteByte('\n')
				continue
			}
			newVisited := make(map[string]bool, len(visited)+1)
			for k, v := range visited {
				newVisited[k] = v
			}
			newVisited[absFullPath] = true
			included, w := resolveIncludes(string(data), filepath.Dir(absFullPath), newVisited)
			warnings = append(warnings, w...)
			out.WriteString(included)
			out.WriteByte('\n')
			continue
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return out.String(), warnings
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

	var modelStmts, viewsStmts []stmt
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

	is.processModelStmts(modelStmts, "", "", is.elements)
	if len(viewsStmts) > 0 {
		is.processViewsStmts(viewsStmts)
	}

	rels := is.buildRelationships()

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

	return &importer.ImportResult{Model: m, Warnings: is.warnings}, nil
}

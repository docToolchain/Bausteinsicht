// Package dsl provides shared tokenizer and parser primitives for
// Bausteinsicht's hand-written DSL importers (structurizr, likec4). Both
// DSLs share the same underlying grammar shape — brace-delimited,
// newline-terminated statements, C-style comments, "quoted strings", and
// "var = keyword args {...}" / "from -> to args {...}" statement forms —
// differing only in a handful of language-specific tokenizing/parsing
// details, which callers implement themselves on top of these primitives.
package dsl

import (
	"fmt"
	"strings"
)

// TokKind classifies a Token.
type TokKind int

const (
	EOF     TokKind = iota
	Newline         // statement separator — emitted for each (group of) newline(s)
	String
	Ident
	LBrace
	RBrace
	Assign
	Arrow
)

// Token is one lexical token produced by Scanner.
type Token struct {
	Kind TokKind
	Val  string
	Line int
}

// Stmt represents one parsed statement in a brace-delimited,
// newline-terminated DSL: "[var =] keyword arg1 arg2 {body}" or
// "[from] -> to arg1 {body}".
type Stmt struct {
	Line    int
	VarName string
	Keyword string
	Args    []string
	IsRel   bool
	RelFrom string
	RelTo   string
	Body    []Stmt
}

// Scanner tokenizes a rune stream shared by Structurizr- and LikeC4-style
// DSLs: C-style // and /* */ comments, newline-as-statement-separator,
// {}/=/-> punctuation, "quoted strings", and bare identifiers. Src/Pos/Line
// are exported so callers can implement their own language-specific token
// dispatch (e.g. wildcard or prefix characters) directly against the cursor.
type Scanner struct {
	Src  []rune
	Pos  int
	Line int
}

// NewScanner creates a Scanner positioned at the start of src.
func NewScanner(src string) *Scanner {
	return &Scanner{Src: []rune(src), Line: 1}
}

// At returns the rune at the given offset from the current position, and
// whether it exists (false at end of input).
func (s *Scanner) At(offset int) (rune, bool) {
	i := s.Pos + offset
	if i >= len(s.Src) {
		return 0, false
	}
	return s.Src[i], true
}

// Consume returns the rune at the current position and advances past it,
// tracking line number on newlines.
func (s *Scanner) Consume() rune {
	r := s.Src[s.Pos]
	s.Pos++
	if r == '\n' {
		s.Line++
	}
	return r
}

// SkipWhitespaceAndComments advances past horizontal whitespace and // and
// /* */ comments. Newlines are NOT skipped — callers emit them as Newline
// tokens.
func (s *Scanner) SkipWhitespaceAndComments() error {
	for {
		c, ok := s.At(0)
		if !ok {
			return nil
		}
		if c == ' ' || c == '\t' || c == '\r' {
			s.Consume()
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
func (s *Scanner) skipComment() (bool, error) {
	switch n, _ := s.At(1); n {
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

// skipLineComment consumes to end of line (leaving \n for the caller).
func (s *Scanner) skipLineComment() {
	for {
		ch, ok := s.At(0)
		if !ok || ch == '\n' {
			return
		}
		s.Consume()
	}
}

// skipBlockComment consumes a /* ... */ block comment starting at the
// current position (the leading "/*").
func (s *Scanner) skipBlockComment() error {
	s.Consume()
	s.Consume()
	for {
		ch, ok := s.At(0)
		if !ok {
			return fmt.Errorf("unterminated block comment")
		}
		s.Consume()
		if ch == '*' {
			if nn, _ := s.At(0); nn == '/' {
				s.Consume()
				return nil
			}
		}
	}
}

// SkipNewlines consumes consecutive '\n' characters.
func (s *Scanner) SkipNewlines() {
	for {
		ch, ok := s.At(0)
		if !ok || ch != '\n' {
			return
		}
		s.Consume()
	}
}

// IdentStartFunc reports whether c can start an identifier-like token in a
// caller's language — this is where callers hook in language-specific
// dispatch (e.g. Structurizr's "*"/"**" wildcard or "!"-prefixed
// identifiers) without Next needing to know about it.
type IdentStartFunc func(c rune) bool

// ScanIdentFunc scans an identifier-like token starting at the scanner's
// current position (the character identStart matched on is not yet
// consumed).
type ScanIdentFunc func(*Scanner, int) (Token, error)

// Next scans the next token using the shared grammar common to Structurizr-
// and LikeC4-style DSLs: {}/=/-> punctuation, "quoted strings", C-style
// comments, and newline-as-statement-separator. Any character for which
// identStart reports true is delegated to scanIdent, letting callers
// implement their own identifier/wildcard/prefix handling on top of this
// shared dispatch.
func Next(s *Scanner, identStart IdentStartFunc, scanIdent ScanIdentFunc) (Token, error) {
	// Skip horizontal whitespace and handle comments.
	// Newlines are NOT skipped here — they are emitted as a Newline token.
	if err := s.SkipWhitespaceAndComments(); err != nil {
		return Token{}, err
	}

	c, ok := s.At(0)
	if !ok {
		return Token{Kind: EOF, Line: s.Line}, nil
	}
	line := s.Line

	// Collapse consecutive newlines into a single Newline token.
	if c == '\n' {
		s.SkipNewlines()
		return Token{Kind: Newline, Line: line}, nil
	}

	switch {
	case c == '{':
		s.Consume()
		return Token{Kind: LBrace, Val: "{", Line: line}, nil
	case c == '}':
		s.Consume()
		return Token{Kind: RBrace, Val: "}", Line: line}, nil
	case c == '=':
		s.Consume()
		return Token{Kind: Assign, Val: "=", Line: line}, nil
	case c == '-':
		if n, _ := s.At(1); n == '>' {
			s.Consume()
			s.Consume()
			return Token{Kind: Arrow, Val: "->", Line: line}, nil
		}
		s.Consume()
		return Next(s, identStart, scanIdent)
	case c == '"':
		return s.ScanString(line)
	case identStart(c):
		return scanIdent(s, line)
	default:
		s.Consume()
		return Next(s, identStart, scanIdent)
	}
}

// ScanString scans a double-quoted string starting at the current position
// (the opening quote), handling \", \\, and \n escapes; any other escaped
// character is passed through literally including its backslash.
func (s *Scanner) ScanString(line int) (Token, error) {
	s.Consume()
	var sb strings.Builder
	for {
		c, ok := s.At(0)
		if !ok {
			return Token{}, fmt.Errorf("line %d: unterminated string", line)
		}
		if c == '"' {
			s.Consume()
			break
		}
		if c == '\\' {
			s.Consume()
			esc, ok := s.At(0)
			if !ok {
				return Token{}, fmt.Errorf("line %d: EOF in string escape", line)
			}
			s.Consume()
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
		sb.WriteRune(s.Consume())
	}
	return Token{Kind: String, Val: sb.String(), Line: line}, nil
}

// ParseOneFunc parses a single statement from p, returning nil (not an
// error) for tokens that don't produce a Stmt (e.g. a bare block or a
// skipped token) — the same convention as Parser.ParseStmts' loop.
type ParseOneFunc func(*Parser) (*Stmt, error)

// Parser holds a token stream and cursor for parsing brace-delimited,
// newline-terminated statement lists shared by Structurizr- and
// LikeC4-style DSLs. The per-language statement grammar is supplied by
// callers as a ParseOneFunc.
type Parser struct {
	Toks []Token
	Pos  int
}

// Peek returns the current token without consuming it (EOF past the end).
func (p *Parser) Peek() Token {
	if p.Pos >= len(p.Toks) {
		return Token{Kind: EOF}
	}
	return p.Toks[p.Pos]
}

// Advance returns the current token and moves past it (a no-op at EOF).
func (p *Parser) Advance() Token {
	t := p.Peek()
	if t.Kind != EOF {
		p.Pos++
	}
	return t
}

// SkipNewlines consumes consecutive Newline tokens.
func (p *Parser) SkipNewlines() {
	for p.Peek().Kind == Newline {
		p.Advance()
	}
}

// ParseAll parses a full top-level statement list.
func (p *Parser) ParseAll(parseOne ParseOneFunc) ([]Stmt, error) {
	return p.ParseStmts(false, parseOne)
}

// ParseStmts parses statements until EOF, or until a closing '}' when
// inBlock is true.
func (p *Parser) ParseStmts(inBlock bool, parseOne ParseOneFunc) ([]Stmt, error) {
	var stmts []Stmt
	for {
		p.SkipNewlines()
		tok := p.Peek()
		if tok.Kind == EOF {
			break
		}
		if inBlock && tok.Kind == RBrace {
			break
		}
		s, err := parseOne(p)
		if err != nil {
			return nil, err
		}
		if s != nil {
			stmts = append(stmts, *s)
		}
	}
	return stmts, nil
}

// ParseBlock parses a "{ stmt* }" block, or returns (nil, nil) if the
// current token isn't '{'.
func (p *Parser) ParseBlock(parseOne ParseOneFunc) ([]Stmt, error) {
	if p.Peek().Kind != LBrace {
		return nil, nil
	}
	p.Advance() // {
	p.SkipNewlines()
	stmts, err := p.ParseStmts(true, parseOne)
	if err != nil {
		return nil, err
	}
	if p.Peek().Kind == RBrace {
		p.Advance()
	}
	return stmts, nil
}

// OptBlock skips newlines then reads a block into s.Body if the next token
// is '{'.
func (p *Parser) OptBlock(s *Stmt, parseOne ParseOneFunc) error {
	p.SkipNewlines()
	if p.Peek().Kind == LBrace {
		body, err := p.ParseBlock(parseOne)
		if err != nil {
			return err
		}
		s.Body = body
	}
	return nil
}

// FinishStmt parses s's optional trailing "{ ... }" block, if present, and
// returns s (or the error from parsing the block).
func (p *Parser) FinishStmt(s *Stmt, parseOne ParseOneFunc) (*Stmt, error) {
	if err := p.OptBlock(s, parseOne); err != nil {
		return nil, err
	}
	return s, nil
}

// CollectArgs consumes consecutive String/Ident tokens as statement
// arguments.
func (p *Parser) CollectArgs() []string {
	var args []string
	for {
		k := p.Peek().Kind
		if k == String || k == Ident {
			args = append(args, p.Advance().Val)
		} else {
			break
		}
	}
	return args
}

// ParseAllStmts parses a full top-level statement list using the shared
// "[var =] keyword args {body}" / "[from] -> to args {body}" statement
// grammar common to Structurizr- and LikeC4-style DSLs.
func ParseAllStmts(p *Parser) ([]Stmt, error) {
	return p.ParseAll(ParseOneStmt)
}

// ParseOneStmt parses a single statement using the shared
// "[var =] keyword args {body}" / "[from] -> to args {body}" grammar common
// to Structurizr- and LikeC4-style DSLs. Both languages' tokenizers only
// ever differ in how they produce Ident/String tokens (see Next), not in
// how those tokens combine into statements, so this needs no per-language
// hook.
func ParseOneStmt(p *Parser) (*Stmt, error) {
	tok := p.Peek()
	if tok.Kind == EOF || tok.Kind == RBrace {
		return nil, nil
	}

	line := tok.Line

	if tok.Kind == Arrow {
		p.Advance()
		to := p.Advance()
		return p.FinishStmt(&Stmt{Line: line, IsRel: true, RelTo: to.Val, Args: p.CollectArgs()}, ParseOneStmt)
	}

	if tok.Kind == LBrace {
		if _, err := p.ParseBlock(ParseOneStmt); err != nil {
			return nil, err
		}
		return nil, nil
	}

	if tok.Kind != Ident && tok.Kind != String {
		p.Advance()
		return nil, nil
	}

	p.Advance()

	switch p.Peek().Kind {
	case Assign:
		p.Advance()
		kw := p.Advance()
		return p.FinishStmt(&Stmt{Line: line, VarName: tok.Val, Keyword: kw.Val, Args: p.CollectArgs()}, ParseOneStmt)

	case Arrow:
		p.Advance()
		to := p.Advance()
		return p.FinishStmt(&Stmt{Line: line, IsRel: true, RelFrom: tok.Val, RelTo: to.Val, Args: p.CollectArgs()}, ParseOneStmt)

	default:
		return p.FinishStmt(&Stmt{Line: line, Keyword: tok.Val, Args: p.CollectArgs()}, ParseOneStmt)
	}
}

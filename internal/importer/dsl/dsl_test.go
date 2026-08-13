package dsl

import (
	"strings"
	"testing"
)

func TestScanner_SkipWhitespaceAndComments(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantPos int
	}{
		{"spaces and tabs", "  \t\t x", 5},
		{"line comment", "// comment\nrest", 10},
		{"block comment", "/* comment */x", 13},
		{"multiline block comment", "/* a\nb */x", 9},
		{"mixed", " // c\n", 5}, // trailing newline is intentionally not skipped
		{"bare slash not a comment", "/x", 0},
		{"no whitespace", "x", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner(tt.src)
			if err := s.SkipWhitespaceAndComments(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if s.Pos != tt.wantPos {
				t.Errorf("Pos = %d, want %d", s.Pos, tt.wantPos)
			}
		})
	}
}

func TestScanner_SkipWhitespaceAndComments_UnterminatedBlockComment(t *testing.T) {
	s := NewScanner("/* never closes")
	if err := s.SkipWhitespaceAndComments(); err == nil {
		t.Error("expected error for unterminated block comment")
	}
}

func TestScanner_SkipNewlines(t *testing.T) {
	s := NewScanner("\n\n\nx")
	s.SkipNewlines()
	if s.Pos != 3 {
		t.Errorf("Pos = %d, want 3", s.Pos)
	}
	if s.Line != 4 {
		t.Errorf("Line = %d, want 4", s.Line)
	}
}

func TestScanner_ScanString(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantVal string
	}{
		{"simple", `"hello"`, "hello"},
		{"escaped quote", `"a\"b"`, `a"b`},
		{"escaped backslash", `"a\\b"`, `a\b`},
		{"escaped newline", `"a\nb"`, "a\nb"},
		{"unknown escape passed through", `"a\qb"`, `a\qb`},
		{"empty", `""`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner(tt.src)
			tok, err := s.ScanString(1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tok.Val != tt.wantVal {
				t.Errorf("Val = %q, want %q", tok.Val, tt.wantVal)
			}
			if tok.Kind != String {
				t.Errorf("Kind = %v, want String", tok.Kind)
			}
		})
	}
}

func TestScanner_ScanString_Unterminated(t *testing.T) {
	s := NewScanner(`"no closing quote`)
	if _, err := s.ScanString(1); err == nil {
		t.Error("expected error for unterminated string")
	}
}

func TestScanner_ScanString_EOFInEscape(t *testing.T) {
	s := NewScanner(`"trailing\`)
	if _, err := s.ScanString(1); err == nil {
		t.Error("expected error for EOF in string escape")
	}
}

func TestScanner_AtAndConsume(t *testing.T) {
	s := NewScanner("ab")
	if c, ok := s.At(0); !ok || c != 'a' {
		t.Errorf("At(0) = %q, %v; want 'a', true", c, ok)
	}
	if c, ok := s.At(2); ok {
		t.Errorf("At(2) = %q, %v; want _, false", c, ok)
	}
	if r := s.Consume(); r != 'a' {
		t.Errorf("Consume() = %q, want 'a'", r)
	}
	if s.Pos != 1 {
		t.Errorf("Pos = %d, want 1", s.Pos)
	}
}

func TestScanner_Consume_TracksLine(t *testing.T) {
	s := NewScanner("a\nb")
	s.Consume() // 'a'
	s.Consume() // '\n'
	if s.Line != 2 {
		t.Errorf("Line = %d, want 2", s.Line)
	}
}

// parseSimple is a minimal ParseOneFunc for testing the Parser: it treats
// every Ident/String token as a Keyword-only Stmt with an optional block.
func parseSimple(p *Parser) (*Stmt, error) {
	tok := p.Peek()
	if tok.Kind == EOF || tok.Kind == RBrace {
		return nil, nil
	}
	if tok.Kind == LBrace {
		if _, err := p.ParseBlock(parseSimple); err != nil {
			return nil, err
		}
		return nil, nil
	}
	if tok.Kind != Ident && tok.Kind != String {
		p.Advance()
		return nil, nil
	}
	p.Advance()
	return p.FinishStmt(&Stmt{Line: tok.Line, Keyword: tok.Val}, parseSimple)
}

func TestParser_ParseAll_FlatStatements(t *testing.T) {
	toks := []Token{
		{Kind: Ident, Val: "a", Line: 1},
		{Kind: Newline, Line: 1},
		{Kind: Ident, Val: "b", Line: 2},
		{Kind: EOF},
	}
	p := &Parser{Toks: toks}
	stmts, err := p.ParseAll(parseSimple)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(stmts))
	}
	if stmts[0].Keyword != "a" || stmts[1].Keyword != "b" {
		t.Errorf("got keywords %q, %q; want \"a\", \"b\"", stmts[0].Keyword, stmts[1].Keyword)
	}
}

func TestParser_ParseAll_NestedBlock(t *testing.T) {
	// a { b }
	toks := []Token{
		{Kind: Ident, Val: "a", Line: 1},
		{Kind: LBrace, Line: 1},
		{Kind: Ident, Val: "b", Line: 1},
		{Kind: RBrace, Line: 1},
		{Kind: EOF},
	}
	p := &Parser{Toks: toks}
	stmts, err := p.ParseAll(parseSimple)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 top-level statement, got %d", len(stmts))
	}
	if len(stmts[0].Body) != 1 || stmts[0].Body[0].Keyword != "b" {
		t.Errorf("expected nested body [b], got %+v", stmts[0].Body)
	}
}

func TestParser_ParseBlock_NotABlock(t *testing.T) {
	p := &Parser{Toks: []Token{{Kind: Ident, Val: "x"}}}
	stmts, err := p.ParseBlock(parseSimple)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stmts != nil {
		t.Errorf("expected nil stmts for non-block, got %v", stmts)
	}
}

func TestParser_PeekAdvance_EOF(t *testing.T) {
	p := &Parser{Toks: []Token{{Kind: Ident, Val: "x"}}}
	p.Advance()
	if p.Peek().Kind != EOF {
		t.Errorf("Peek() past end = %v, want EOF", p.Peek().Kind)
	}
	// Advance past EOF should be a no-op, not panic or move Pos further.
	p.Advance()
	if p.Pos != 1 {
		t.Errorf("Pos after advancing past EOF = %d, want 1", p.Pos)
	}
}

// testIdentStart/testScanIdent form a minimal identifier grammar (bare
// letters/digits/underscore) used to exercise Next and ParseOneStmt/
// ParseAllStmts without depending on either real DSL importer.
func testIdentStart(c rune) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func testScanIdent(s *Scanner, line int) (Token, error) {
	start := s.Pos
	for {
		c, ok := s.At(0)
		isDigit := c >= '0' && c <= '9'
		if !ok || (!testIdentStart(c) && !isDigit) {
			break
		}
		s.Consume()
	}
	return Token{Kind: Ident, Val: string(s.Src[start:s.Pos]), Line: line}, nil
}

func testTokenize(t *testing.T, src string) []Token {
	t.Helper()
	s := NewScanner(src)
	var toks []Token
	for {
		tok, err := Next(s, testIdentStart, testScanIdent)
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		toks = append(toks, tok)
		if tok.Kind == EOF {
			return toks
		}
	}
}

func TestNext_Symbols(t *testing.T) {
	toks := testTokenize(t, `a = b { c -> d "e" }`)
	var kinds []TokKind
	for _, tok := range toks {
		kinds = append(kinds, tok.Kind)
	}
	want := []TokKind{Ident, Assign, Ident, LBrace, Ident, Arrow, Ident, String, RBrace, EOF}
	if len(kinds) != len(want) {
		t.Fatalf("got %d tokens %v, want %d %v", len(kinds), kinds, len(want), want)
	}
	for i, k := range want {
		if kinds[i] != k {
			t.Errorf("token %d: kind = %v, want %v", i, kinds[i], k)
		}
	}
}

func TestNext_CommentsAndNewlines(t *testing.T) {
	toks := testTokenize(t, "a // comment\n\n\nb")
	var vals []string
	for _, tok := range toks {
		if tok.Kind == Ident {
			vals = append(vals, tok.Val)
		}
	}
	if len(vals) != 2 || vals[0] != "a" || vals[1] != "b" {
		t.Errorf("got idents %v, want [a b]", vals)
	}
	if toks[1].Kind != Newline {
		t.Errorf("expected a single collapsed Newline token between idents, got %v", toks[1].Kind)
	}
}

func TestNext_UnterminatedBlockComment(t *testing.T) {
	s := NewScanner("/* oops")
	if _, err := Next(s, testIdentStart, testScanIdent); err == nil {
		t.Error("expected error for unterminated block comment")
	}
}

func TestNext_DashNotArrow_SkipsSingleDash(t *testing.T) {
	// A lone '-' (not followed by '>') isn't a token in this shared grammar
	// and is silently skipped, same as any other unrecognized character.
	toks := testTokenize(t, "a - b")
	var vals []string
	for _, tok := range toks {
		if tok.Kind == Ident {
			vals = append(vals, tok.Val)
		}
	}
	if len(vals) != 2 || vals[0] != "a" || vals[1] != "b" {
		t.Errorf("got idents %v, want [a b]", vals)
	}
}

func TestParseAllStmts_VarAssignAndRelationship(t *testing.T) {
	toks := testTokenize(t, `x = person "Alice"
x -> y "uses"
`)
	p := &Parser{Toks: toks}
	stmts, err := ParseAllStmts(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 2 {
		t.Fatalf("expected 2 statements, got %d: %+v", len(stmts), stmts)
	}

	elem := stmts[0]
	if elem.VarName != "x" || elem.Keyword != "person" || len(elem.Args) != 1 || elem.Args[0] != "Alice" {
		t.Errorf("element stmt = %+v, want VarName=x Keyword=person Args=[Alice]", elem)
	}

	rel := stmts[1]
	if !rel.IsRel || rel.RelFrom != "x" || rel.RelTo != "y" || len(rel.Args) != 1 || rel.Args[0] != "uses" {
		t.Errorf("relationship stmt = %+v, want IsRel RelFrom=x RelTo=y Args=[uses]", rel)
	}
}

func TestParseAllStmts_NestedBlockAndBareRelationship(t *testing.T) {
	// group { a -> b } — a relationship with no explicit source nested
	// inside a block; the grammar itself doesn't fill in the source (that's
	// each importer's own responsibility), so RelFrom is empty here.
	toks := testTokenize(t, "group {\na -> b\n}")
	p := &Parser{Toks: toks}
	stmts, err := ParseAllStmts(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 1 || stmts[0].Keyword != "group" {
		t.Fatalf("expected 1 top-level 'group' statement, got %+v", stmts)
	}
	body := stmts[0].Body
	if len(body) != 1 || !body[0].IsRel || body[0].RelFrom != "a" || body[0].RelTo != "b" {
		t.Errorf("nested body = %+v, want a single a->b relationship", body)
	}
}

func TestParseOneStmt_SkipsUnrecognizedToken(t *testing.T) {
	// A token that's neither Ident/String nor Arrow (here: a bare RBrace at
	// the top level, which ParseOneStmt doesn't otherwise special-case) is
	// consumed and produces no statement.
	p := &Parser{Toks: []Token{{Kind: Assign}, {Kind: EOF}}}
	stmts, err := ParseAllStmts(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stmts) != 0 {
		t.Errorf("expected no statements, got %+v", stmts)
	}
	if p.Pos != 1 {
		t.Errorf("expected the unrecognized token to be consumed, Pos = %d, want 1", p.Pos)
	}
}

func TestTokenize(t *testing.T) {
	toks, err := Tokenize(`a = b "c"`, testIdentStart, testScanIdent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []TokKind{Ident, Assign, Ident, String, EOF}
	if len(toks) != len(want) {
		t.Fatalf("got %d tokens, want %d: %+v", len(toks), len(want), toks)
	}
	for i, k := range want {
		if toks[i].Kind != k {
			t.Errorf("token %d: kind = %v, want %v", i, toks[i].Kind, k)
		}
	}
}

func TestTokenize_Error(t *testing.T) {
	if _, err := Tokenize("/* unterminated", testIdentStart, testScanIdent); err == nil {
		t.Error("expected error for unterminated block comment")
	}
}

func TestScanIdentBody(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{"plain", "abc123 rest", "abc123"},
		{"dotted path with dash", "a.b.c-x", "a.b.c-x"}, // '-' only stops before "->"
		{"stops before arrow", "a->b", "a"},
		{"slash and colon", "a/b:c d", "a/b:c"},
		{"empty at eof", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner(tt.src)
			var sb strings.Builder
			ScanIdentBody(s, &sb)
			if sb.String() != tt.want {
				t.Errorf("got %q, want %q", sb.String(), tt.want)
			}
		})
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "Hello World", "hello_world"},
		{"already lower", "already_ok", "already_ok"},
		{"punctuation collapsed", "A!!!B", "a_b"},
		{"trailing punctuation trimmed", "Trailing!!!", "trailing"},
		{"empty falls back", "!!!", "element"},
		{"fully empty falls back", "", "element"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Slugify(tt.in); got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParser_CollectArgs(t *testing.T) {
	p := &Parser{Toks: []Token{
		{Kind: String, Val: "a"},
		{Kind: Ident, Val: "b"},
		{Kind: Newline},
	}}
	args := p.CollectArgs()
	if len(args) != 2 || args[0] != "a" || args[1] != "b" {
		t.Errorf("CollectArgs() = %v, want [a b]", args)
	}
}

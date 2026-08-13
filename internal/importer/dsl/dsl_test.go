package dsl

import "testing"

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

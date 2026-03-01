// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.
//
// The ProbeChain is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package lexer_test

import (
	"testing"

	"github.com/probechain/go-probe/probe-lang/lang/lexer"
	"github.com/probechain/go-probe/probe-lang/lang/token"
)

// tokenCase is a single expected token in a table-driven test.
type tokenCase struct {
	typ     token.Type
	literal string
}

// runTokenize lexes input and checks that it produces exactly the expected
// sequence (plus a final EOF).
func runTokenize(t *testing.T, name, input string, want []tokenCase) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		t.Helper()
		l := lexer.New("test.probe", input)
		toks := l.Tokenize()

		// Tokenize always appends EOF; the want slice should NOT include EOF.
		if len(toks) == 0 {
			t.Fatal("Tokenize returned empty slice")
		}
		// Last token must be EOF.
		last := toks[len(toks)-1]
		if last.Type != token.EOF {
			t.Errorf("last token is %s, want EOF", last.Type)
		}
		body := toks[:len(toks)-1]

		if len(body) != len(want) {
			t.Errorf("got %d tokens (excl. EOF), want %d", len(body), len(want))
			for i, tok := range body {
				t.Logf("  [%d] %s %q", i, tok.Type, tok.Literal)
			}
			return
		}
		for i, w := range want {
			got := body[i]
			if got.Type != w.typ {
				t.Errorf("token[%d]: type = %s, want %s (literal %q)", i, got.Type, w.typ, got.Literal)
			}
			if got.Literal != w.literal {
				t.Errorf("token[%d]: literal = %q, want %q", i, got.Literal, w.literal)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// Single-character operators and delimiters
// ---------------------------------------------------------------------------

func TestSingleCharTokens(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantTyp token.Type
		wantLit string
	}{
		{"plus", "+", token.PLUS, "+"},
		{"minus", "-", token.MINUS, "-"},
		{"star", "*", token.STAR, "*"},
		{"slash", "/", token.SLASH, "/"},
		{"percent", "%", token.PERCENT, "%"},
		{"hash", "#", token.HASH, "#"},
		{"tilde", "~", token.TILDE, "~"},
		{"amp", "&", token.AMP, "&"},
		{"pipe", "|", token.PIPE, "|"},
		{"caret", "^", token.CARET, "^"},
		{"bang", "!", token.BANG, "!"},
		{"dot", ".", token.DOT, "."},
		{"lt", "<", token.LT, "<"},
		{"gt", ">", token.GT, ">"},
		{"assign", "=", token.ASSIGN, "="},
		{"colon", ":", token.COLON, ":"},
		{"at", "@", token.AT, "@"},
		{"lparen", "(", token.LPAREN, "("},
		{"rparen", ")", token.RPAREN, ")"},
		{"lbracket", "[", token.LBRACKET, "["},
		{"rbracket", "]", token.RBRACKET, "]"},
		{"lbrace", "{", token.LBRACE, "{"},
		{"rbrace", "}", token.RBRACE, "}"},
		{"comma", ",", token.COMMA, ","},
		{"semicolon", ";", token.SEMICOLON, ";"},
	}
	for _, c := range cases {
		runTokenize(t, c.name, c.input, []tokenCase{{c.wantTyp, c.wantLit}})
	}
}

// ---------------------------------------------------------------------------
// Multi-character operators
// ---------------------------------------------------------------------------

func TestMultiCharOperators(t *testing.T) {
	runTokenize(t, "EQ", "==", []tokenCase{{token.EQ, "=="}})
	runTokenize(t, "NEQ", "!=", []tokenCase{{token.NEQ, "!="}})
	runTokenize(t, "LTE", "<=", []tokenCase{{token.LTE, "<="}})
	runTokenize(t, "GTE", ">=", []tokenCase{{token.GTE, ">="}})
	runTokenize(t, "AND", "&&", []tokenCase{{token.AND, "&&"}})
	runTokenize(t, "OR", "||", []tokenCase{{token.OR, "||"}})
	runTokenize(t, "ARROW", "->", []tokenCase{{token.ARROW, "->"}})
	runTokenize(t, "FATARROW", "=>", []tokenCase{{token.FATARROW, "=>"}})
	runTokenize(t, "COLONCOLON", "::", []tokenCase{{token.COLONCOLON, "::"}})
	runTokenize(t, "DOTDOT", "..", []tokenCase{{token.DOTDOT, ".."}})
	runTokenize(t, "LSHIFT", "<<", []tokenCase{{token.LSHIFT, "<<"}})
	runTokenize(t, "RSHIFT", ">>", []tokenCase{{token.RSHIFT, ">>"}})
}

// ---------------------------------------------------------------------------
// Compound assignment operators
// ---------------------------------------------------------------------------

func TestCompoundAssignment(t *testing.T) {
	runTokenize(t, "PLUSEQ", "+=", []tokenCase{{token.PLUSEQ, "+="}})
	runTokenize(t, "MINUSEQ", "-=", []tokenCase{{token.MINUSEQ, "-="}})
	runTokenize(t, "STAREQ", "*=", []tokenCase{{token.STAREQ, "*="}})
	runTokenize(t, "SLASHEQ", "/=", []tokenCase{{token.SLASHEQ, "/="}})
	runTokenize(t, "PERCENTEQ", "%=", []tokenCase{{token.PERCENTEQ, "%="}})
	runTokenize(t, "AMPEQ", "&=", []tokenCase{{token.AMPEQ, "&="}})
	runTokenize(t, "PIPEEQ", "|=", []tokenCase{{token.PIPEEQ, "|="}})
	runTokenize(t, "CARETEQ", "^=", []tokenCase{{token.CARETEQ, "^="}})
	runTokenize(t, "LSHIFTEQ", "<<=", []tokenCase{{token.LSHIFTEQ, "<<="}})
	runTokenize(t, "RSHIFTEQ", ">>=", []tokenCase{{token.RSHIFTEQ, ">>="}})
}

// ---------------------------------------------------------------------------
// Integer literals
// ---------------------------------------------------------------------------

func TestIntLiterals(t *testing.T) {
	runTokenize(t, "zero", "0", []tokenCase{{token.INT, "0"}})
	runTokenize(t, "single", "7", []tokenCase{{token.INT, "7"}})
	runTokenize(t, "multi", "42", []tokenCase{{token.INT, "42"}})
	runTokenize(t, "large", "1000000", []tokenCase{{token.INT, "1000000"}})
}

// ---------------------------------------------------------------------------
// Float literals
// ---------------------------------------------------------------------------

func TestFloatLiterals(t *testing.T) {
	runTokenize(t, "basic", "3.14", []tokenCase{{token.FLOAT, "3.14"}})
	runTokenize(t, "leading_zero", "0.5", []tokenCase{{token.FLOAT, "0.5"}})
	runTokenize(t, "exponent", "1.5e10", []tokenCase{{token.FLOAT, "1.5e10"}})
	runTokenize(t, "exponent_upper", "2.0E3", []tokenCase{{token.FLOAT, "2.0E3"}})
	runTokenize(t, "exponent_neg", "1.0e-5", []tokenCase{{token.FLOAT, "1.0e-5"}})
	runTokenize(t, "exponent_pos", "1.0e+5", []tokenCase{{token.FLOAT, "1.0e+5"}})
}

// ---------------------------------------------------------------------------
// Hex / BYTES literals
// ---------------------------------------------------------------------------

func TestHexBytesLiterals(t *testing.T) {
	runTokenize(t, "short", "0xff", []tokenCase{{token.BYTES, "0xff"}})
	runTokenize(t, "upper_x", "0XFF", []tokenCase{{token.BYTES, "0XFF"}})
	runTokenize(t, "deadbeef", "0xdeadbeef", []tokenCase{{token.BYTES, "0xdeadbeef"}})
	runTokenize(t, "mixed_case", "0xDeAdBeEf", []tokenCase{{token.BYTES, "0xDeAdBeEf"}})
	runTokenize(t, "long_hash", "0xabcdef0123456789ABCDEF",
		[]tokenCase{{token.BYTES, "0xabcdef0123456789ABCDEF"}})
}

// ---------------------------------------------------------------------------
// Address literals
// ---------------------------------------------------------------------------

func TestAddressLiterals(t *testing.T) {
	runTokenize(t, "short_addr", "@0x1234",
		[]tokenCase{{token.ADDRESS, "@0x1234"}})
	runTokenize(t, "full_addr", "@0xd3CdA913deB6f4967b2Ef3aa68f5Ca9ac58A",
		[]tokenCase{{token.ADDRESS, "@0xd3CdA913deB6f4967b2Ef3aa68f5Ca9ac58A"}})
	// bare @ (not followed by 0x) should be AT
	runTokenize(t, "at_alone", "@", []tokenCase{{token.AT, "@"}})
	runTokenize(t, "at_nonhex", "@foo", []tokenCase{
		{token.AT, "@"},
		{token.IDENT, "foo"},
	})
}

// ---------------------------------------------------------------------------
// String literals
// ---------------------------------------------------------------------------

func TestStringLiterals(t *testing.T) {
	runTokenize(t, "empty", `""`, []tokenCase{{token.STRING, `""`}})
	runTokenize(t, "hello", `"hello"`, []tokenCase{{token.STRING, `"hello"`}})
	runTokenize(t, "escape_n", `"line\nfeed"`, []tokenCase{{token.STRING, `"line\nfeed"`}})
	runTokenize(t, "escape_t", `"tab\there"`, []tokenCase{{token.STRING, `"tab\there"`}})
	runTokenize(t, "escape_backslash", `"back\\slash"`, []tokenCase{{token.STRING, `"back\\slash"`}})
	runTokenize(t, "escape_quote", `"say\"hi\""`, []tokenCase{{token.STRING, `"say\"hi\""`}})
	runTokenize(t, "escape_r", `"cr\rhere"`, []tokenCase{{token.STRING, `"cr\rhere"`}})
	runTokenize(t, "escape_0", `"null\0byte"`, []tokenCase{{token.STRING, `"null\0byte"`}})
	runTokenize(t, "spaces", `"hello world"`, []tokenCase{{token.STRING, `"hello world"`}})
}

// ---------------------------------------------------------------------------
// Identifiers
// ---------------------------------------------------------------------------

func TestIdentifiers(t *testing.T) {
	runTokenize(t, "simple", "foo", []tokenCase{{token.IDENT, "foo"}})
	runTokenize(t, "underscore_prefix", "_bar", []tokenCase{{token.IDENT, "_bar"}})
	runTokenize(t, "underscore_only", "_", []tokenCase{{token.IDENT, "_"}})
	runTokenize(t, "mixed_case", "MyVar", []tokenCase{{token.IDENT, "MyVar"}})
	runTokenize(t, "with_digits", "x1y2z3", []tokenCase{{token.IDENT, "x1y2z3"}})
	runTokenize(t, "all_caps", "CONST_VAL", []tokenCase{{token.IDENT, "CONST_VAL"}})
}

// ---------------------------------------------------------------------------
// Keywords
// ---------------------------------------------------------------------------

func TestKeywords(t *testing.T) {
	cases := []struct {
		kw  string
		typ token.Type
	}{
		{"fn", token.FN},
		{"let", token.LET},
		{"mut", token.MUT},
		{"move", token.MOVE},
		{"copy", token.COPY},
		{"drop", token.DROP},
		{"if", token.IF},
		{"else", token.ELSE},
		{"match", token.MATCH},
		{"for", token.FOR},
		{"in", token.IN},
		{"while", token.WHILE},
		{"return", token.RETURN},
		{"break", token.BREAK},
		{"continue", token.CONTINUE},
		{"struct", token.STRUCT},
		{"enum", token.ENUM},
		{"impl", token.IMPL},
		{"trait", token.TRAIT},
		{"type", token.TYPE},
		{"pub", token.PUB},
		{"use", token.USE},
		{"mod", token.MOD},
		{"as", token.AS},
		{"self", token.SELF},
		{"true", token.TRUE},
		{"false", token.FALSE},
		{"nil", token.NIL},
		{"agent", token.AGENT},
		{"msg", token.MSG},
		{"send", token.SEND},
		{"recv", token.RECV},
		{"spawn", token.SPAWN},
		{"state", token.STATE},
		{"tx", token.TX},
		{"emit", token.EMIT},
		{"require", token.REQUIRE},
		{"assert", token.ASSERT},
		{"resource", token.RESOURCE},
	}
	for _, c := range cases {
		runTokenize(t, c.kw, c.kw, []tokenCase{{c.typ, c.kw}})
	}
}

// Prefix of a keyword should still be an IDENT.
func TestKeywordPrefixIsIdent(t *testing.T) {
	runTokenize(t, "fn_prefix", "fnn", []tokenCase{{token.IDENT, "fnn"}})
	runTokenize(t, "let_prefix", "letx", []tokenCase{{token.IDENT, "letx"}})
	runTokenize(t, "if_prefix", "iff", []tokenCase{{token.IDENT, "iff"}})
}

// ---------------------------------------------------------------------------
// Comments
// ---------------------------------------------------------------------------

func TestLineComment(t *testing.T) {
	runTokenize(t, "empty_line_comment", "//", []tokenCase{{token.COMMENT, "//"}})
	runTokenize(t, "line_comment", "// hello world", []tokenCase{{token.COMMENT, "// hello world"}})
	runTokenize(t, "line_comment_then_code", "// comment\nfoo", []tokenCase{
		{token.COMMENT, "// comment"},
		{token.IDENT, "foo"},
	})
}

func TestBlockComment(t *testing.T) {
	runTokenize(t, "empty_block", "/**/", []tokenCase{{token.COMMENT, "/**/"}})
	runTokenize(t, "block_comment", "/* hello */", []tokenCase{{token.COMMENT, "/* hello */"}})
	runTokenize(t, "block_multiline", "/* line1\nline2 */", []tokenCase{{token.COMMENT, "/* line1\nline2 */"}})
	runTokenize(t, "block_then_code", "/* c */x", []tokenCase{
		{token.COMMENT, "/* c */"},
		{token.IDENT, "x"},
	})
}

func TestUnterminatedBlockComment(t *testing.T) {
	t.Run("unterminated_block", func(t *testing.T) {
		l := lexer.New("test.probe", "/* oops")
		tok := l.NextToken()
		if tok.Type != token.ILLEGAL {
			t.Errorf("expected ILLEGAL for unterminated block comment, got %s", tok.Type)
		}
	})
}

func TestUnterminatedString(t *testing.T) {
	t.Run("unterminated_string", func(t *testing.T) {
		l := lexer.New("test.probe", `"no closing`)
		tok := l.NextToken()
		if tok.Type != token.ILLEGAL {
			t.Errorf("expected ILLEGAL for unterminated string, got %s", tok.Type)
		}
	})
}

// ---------------------------------------------------------------------------
// Whitespace handling
// ---------------------------------------------------------------------------

func TestWhitespaceSkipping(t *testing.T) {
	runTokenize(t, "spaces", "   foo   ", []tokenCase{{token.IDENT, "foo"}})
	runTokenize(t, "tabs", "\t\tfoo\t\t", []tokenCase{{token.IDENT, "foo"}})
	runTokenize(t, "newlines", "\n\nfoo\n\n", []tokenCase{{token.IDENT, "foo"}})
	runTokenize(t, "mixed_ws", " \t\n foo \n\t", []tokenCase{{token.IDENT, "foo"}})
}

// ---------------------------------------------------------------------------
// Compound expressions
// ---------------------------------------------------------------------------

func TestFunctionDeclaration(t *testing.T) {
	input := `fn add(x: i32, y: i32) -> i32 { return x + y; }`
	runTokenize(t, "fn_decl", input, []tokenCase{
		{token.FN, "fn"},
		{token.IDENT, "add"},
		{token.LPAREN, "("},
		{token.IDENT, "x"},
		{token.COLON, ":"},
		{token.IDENT, "i32"},
		{token.COMMA, ","},
		{token.IDENT, "y"},
		{token.COLON, ":"},
		{token.IDENT, "i32"},
		{token.RPAREN, ")"},
		{token.ARROW, "->"},
		{token.IDENT, "i32"},
		{token.LBRACE, "{"},
		{token.RETURN, "return"},
		{token.IDENT, "x"},
		{token.PLUS, "+"},
		{token.IDENT, "y"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
	})
}

func TestLetStatement(t *testing.T) {
	input := `let mut x = 42;`
	runTokenize(t, "let_stmt", input, []tokenCase{
		{token.LET, "let"},
		{token.MUT, "mut"},
		{token.IDENT, "x"},
		{token.ASSIGN, "="},
		{token.INT, "42"},
		{token.SEMICOLON, ";"},
	})
}

func TestMatchExpression(t *testing.T) {
	input := `match x { 1 => true, _ => false }`
	runTokenize(t, "match_expr", input, []tokenCase{
		{token.MATCH, "match"},
		{token.IDENT, "x"},
		{token.LBRACE, "{"},
		{token.INT, "1"},
		{token.FATARROW, "=>"},
		{token.TRUE, "true"},
		{token.COMMA, ","},
		{token.IDENT, "_"},
		{token.FATARROW, "=>"},
		{token.FALSE, "false"},
		{token.RBRACE, "}"},
	})
}

func TestAgentDeclaration(t *testing.T) {
	input := `agent Counter { state count: i32; }`
	runTokenize(t, "agent_decl", input, []tokenCase{
		{token.AGENT, "agent"},
		{token.IDENT, "Counter"},
		{token.LBRACE, "{"},
		{token.STATE, "state"},
		{token.IDENT, "count"},
		{token.COLON, ":"},
		{token.IDENT, "i32"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
	})
}

func TestBlockchainKeywords(t *testing.T) {
	input := `require(balance > 0); emit Transfer(from, to);`
	runTokenize(t, "blockchain_kws", input, []tokenCase{
		{token.REQUIRE, "require"},
		{token.LPAREN, "("},
		{token.IDENT, "balance"},
		{token.GT, ">"},
		{token.INT, "0"},
		{token.RPAREN, ")"},
		{token.SEMICOLON, ";"},
		{token.EMIT, "emit"},
		{token.IDENT, "Transfer"},
		{token.LPAREN, "("},
		{token.IDENT, "from"},
		{token.COMMA, ","},
		{token.IDENT, "to"},
		{token.RPAREN, ")"},
		{token.SEMICOLON, ";"},
	})
}

func TestPathExpression(t *testing.T) {
	input := `std::io::print`
	runTokenize(t, "path_expr", input, []tokenCase{
		{token.IDENT, "std"},
		{token.COLONCOLON, "::"},
		{token.IDENT, "io"},
		{token.COLONCOLON, "::"},
		{token.IDENT, "print"},
	})
}

func TestRangeExpression(t *testing.T) {
	input := `0..10`
	runTokenize(t, "range_expr", input, []tokenCase{
		{token.INT, "0"},
		{token.DOTDOT, ".."},
		{token.INT, "10"},
	})
}

func TestBitwiseOperators(t *testing.T) {
	input := `a & b | c ^ d`
	runTokenize(t, "bitwise", input, []tokenCase{
		{token.IDENT, "a"},
		{token.AMP, "&"},
		{token.IDENT, "b"},
		{token.PIPE, "|"},
		{token.IDENT, "c"},
		{token.CARET, "^"},
		{token.IDENT, "d"},
	})
}

func TestShiftOperatorsWithAssign(t *testing.T) {
	input := `x <<= 2; y >>= 3;`
	runTokenize(t, "shift_assign", input, []tokenCase{
		{token.IDENT, "x"},
		{token.LSHIFTEQ, "<<="},
		{token.INT, "2"},
		{token.SEMICOLON, ";"},
		{token.IDENT, "y"},
		{token.RSHIFTEQ, ">>="},
		{token.INT, "3"},
		{token.SEMICOLON, ";"},
	})
}

func TestAddressInExpression(t *testing.T) {
	input := `send(msg, @0xdeadbeef);`
	runTokenize(t, "addr_in_expr", input, []tokenCase{
		{token.SEND, "send"},
		{token.LPAREN, "("},
		{token.MSG, "msg"},
		{token.COMMA, ","},
		{token.ADDRESS, "@0xdeadbeef"},
		{token.RPAREN, ")"},
		{token.SEMICOLON, ";"},
	})
}

func TestHexInExpression(t *testing.T) {
	input := `let mask = 0xFF00;`
	runTokenize(t, "hex_in_expr", input, []tokenCase{
		{token.LET, "let"},
		{token.IDENT, "mask"},
		{token.ASSIGN, "="},
		{token.BYTES, "0xFF00"},
		{token.SEMICOLON, ";"},
	})
}

func TestFloatInExpression(t *testing.T) {
	input := `let pi = 3.14159;`
	runTokenize(t, "float_in_expr", input, []tokenCase{
		{token.LET, "let"},
		{token.IDENT, "pi"},
		{token.ASSIGN, "="},
		{token.FLOAT, "3.14159"},
		{token.SEMICOLON, ";"},
	})
}

func TestLogicalOperators(t *testing.T) {
	input := `if a && b || c {}`
	runTokenize(t, "logical_ops", input, []tokenCase{
		{token.IF, "if"},
		{token.IDENT, "a"},
		{token.AND, "&&"},
		{token.IDENT, "b"},
		{token.OR, "||"},
		{token.IDENT, "c"},
		{token.LBRACE, "{"},
		{token.RBRACE, "}"},
	})
}

func TestCommentAmidCode(t *testing.T) {
	input := "x // ignore this\ny"
	runTokenize(t, "comment_amid_code", input, []tokenCase{
		{token.IDENT, "x"},
		{token.COMMENT, "// ignore this"},
		{token.IDENT, "y"},
	})
}

func TestBlockCommentAmidCode(t *testing.T) {
	input := "x /* ignored */ y"
	runTokenize(t, "block_comment_amid_code", input, []tokenCase{
		{token.IDENT, "x"},
		{token.COMMENT, "/* ignored */"},
		{token.IDENT, "y"},
	})
}

func TestFieldAccess(t *testing.T) {
	input := `obj.field`
	runTokenize(t, "field_access", input, []tokenCase{
		{token.IDENT, "obj"},
		{token.DOT, "."},
		{token.IDENT, "field"},
	})
}

// ---------------------------------------------------------------------------
// Position tracking
// ---------------------------------------------------------------------------

func TestPositionTracking(t *testing.T) {
	t.Run("line_and_column", func(t *testing.T) {
		l := lexer.New("src.probe", "foo\nbar")
		toks := l.Tokenize()
		// toks: [IDENT(foo), IDENT(bar), EOF]
		if len(toks) < 2 {
			t.Fatal("expected at least 2 tokens")
		}
		foo := toks[0]
		bar := toks[1]
		if foo.Pos.Line != 1 {
			t.Errorf("foo: line = %d, want 1", foo.Pos.Line)
		}
		if foo.Pos.Column != 1 {
			t.Errorf("foo: col = %d, want 1", foo.Pos.Column)
		}
		if bar.Pos.Line != 2 {
			t.Errorf("bar: line = %d, want 2", bar.Pos.Line)
		}
		if bar.Pos.Column != 1 {
			t.Errorf("bar: col = %d, want 1", bar.Pos.Column)
		}
	})

	t.Run("filename_propagated", func(t *testing.T) {
		l := lexer.New("myfile.probe", "x")
		tok := l.NextToken()
		if tok.Pos.File != "myfile.probe" {
			t.Errorf("file = %q, want %q", tok.Pos.File, "myfile.probe")
		}
	})
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestEmptyInput(t *testing.T) {
	t.Run("empty_input", func(t *testing.T) {
		l := lexer.New("test.probe", "")
		tok := l.NextToken()
		if tok.Type != token.EOF {
			t.Errorf("expected EOF for empty input, got %s", tok.Type)
		}
	})
}

func TestWhitespaceOnlyInput(t *testing.T) {
	t.Run("whitespace_only", func(t *testing.T) {
		l := lexer.New("test.probe", "   \t\n  ")
		tok := l.NextToken()
		if tok.Type != token.EOF {
			t.Errorf("expected EOF for whitespace-only input, got %s", tok.Type)
		}
	})
}

func TestIllegalCharacter(t *testing.T) {
	t.Run("illegal_char", func(t *testing.T) {
		l := lexer.New("test.probe", "`")
		tok := l.NextToken()
		if tok.Type != token.ILLEGAL {
			t.Errorf("expected ILLEGAL for backtick, got %s", tok.Type)
		}
		if tok.Literal != "`" {
			t.Errorf("expected literal '`', got %q", tok.Literal)
		}
	})
}

func TestMultipleCallsAfterEOF(t *testing.T) {
	t.Run("eof_idempotent", func(t *testing.T) {
		l := lexer.New("test.probe", "")
		for i := 0; i < 5; i++ {
			tok := l.NextToken()
			if tok.Type != token.EOF {
				t.Errorf("call %d: expected EOF, got %s", i, tok.Type)
			}
		}
	})
}

func TestIntDotIsNotFloat(t *testing.T) {
	// "1.fn" - the dot should not start a float because 'f' is not a digit
	runTokenize(t, "int_dot_kw", "1.fn", []tokenCase{
		{token.INT, "1"},
		{token.DOT, "."},
		{token.FN, "fn"},
	})
}

func TestZeroAlone(t *testing.T) {
	runTokenize(t, "zero_alone", "0", []tokenCase{{token.INT, "0"}})
}

func TestZeroHexPrefix(t *testing.T) {
	// Just "0x" with no hex digits is still BYTES (empty hex).
	runTokenize(t, "zero_x_empty", "0x", []tokenCase{{token.BYTES, "0x"}})
}

func TestNegativeNumberIsMinusThenInt(t *testing.T) {
	// The lexer does not produce negative literals; '-' is always a MINUS token.
	runTokenize(t, "negative", "-42", []tokenCase{
		{token.MINUS, "-"},
		{token.INT, "42"},
	})
}

func TestForLoopRange(t *testing.T) {
	input := `for i in 0..n {}`
	runTokenize(t, "for_range", input, []tokenCase{
		{token.FOR, "for"},
		{token.IDENT, "i"},
		{token.IN, "in"},
		{token.INT, "0"},
		{token.DOTDOT, ".."},
		{token.IDENT, "n"},
		{token.LBRACE, "{"},
		{token.RBRACE, "}"},
	})
}

func TestSpawnAgent(t *testing.T) {
	input := `spawn MyAgent { msg: "start" }`
	runTokenize(t, "spawn_agent", input, []tokenCase{
		{token.SPAWN, "spawn"},
		{token.IDENT, "MyAgent"},
		{token.LBRACE, "{"},
		{token.MSG, "msg"},
		{token.COLON, ":"},
		{token.STRING, `"start"`},
		{token.RBRACE, "}"},
	})
}

func TestResourceKeyword(t *testing.T) {
	input := `resource Token { amount: u64 }`
	runTokenize(t, "resource_kw", input, []tokenCase{
		{token.RESOURCE, "resource"},
		{token.IDENT, "Token"},
		{token.LBRACE, "{"},
		{token.IDENT, "amount"},
		{token.COLON, ":"},
		{token.IDENT, "u64"},
		{token.RBRACE, "}"},
	})
}

func TestTxKeyword(t *testing.T) {
	input := `tx.sender`
	runTokenize(t, "tx_dot", input, []tokenCase{
		{token.TX, "tx"},
		{token.DOT, "."},
		{token.IDENT, "sender"},
	})
}

func TestComparisonChain(t *testing.T) {
	input := `a == b != c < d > e <= f >= g`
	runTokenize(t, "comparison_chain", input, []tokenCase{
		{token.IDENT, "a"},
		{token.EQ, "=="},
		{token.IDENT, "b"},
		{token.NEQ, "!="},
		{token.IDENT, "c"},
		{token.LT, "<"},
		{token.IDENT, "d"},
		{token.GT, ">"},
		{token.IDENT, "e"},
		{token.LTE, "<="},
		{token.IDENT, "f"},
		{token.GTE, ">="},
		{token.IDENT, "g"},
	})
}

func TestMonadicOperators(t *testing.T) {
	// J-style monadic uses: #x (length), ~x (bitwise not), !x (logical not)
	input := `#arr ~bits !flag`
	runTokenize(t, "monadic_ops", input, []tokenCase{
		{token.HASH, "#"},
		{token.IDENT, "arr"},
		{token.TILDE, "~"},
		{token.IDENT, "bits"},
		{token.BANG, "!"},
		{token.IDENT, "flag"},
	})
}

func TestComplexProgram(t *testing.T) {
	input := `
agent Transfer {
    state balance: u64;

    fn transfer(to: @0xdead, amount: u64) -> bool {
        require(self.balance >= amount);
        self.balance -= amount;
        emit Sent(to, amount);
        return true;
    }
}
`
	runTokenize(t, "complex_program", input, []tokenCase{
		{token.AGENT, "agent"},
		{token.IDENT, "Transfer"},
		{token.LBRACE, "{"},
		{token.STATE, "state"},
		{token.IDENT, "balance"},
		{token.COLON, ":"},
		{token.IDENT, "u64"},
		{token.SEMICOLON, ";"},
		{token.FN, "fn"},
		{token.IDENT, "transfer"},
		{token.LPAREN, "("},
		{token.IDENT, "to"},
		{token.COLON, ":"},
		{token.ADDRESS, "@0xdead"},
		{token.COMMA, ","},
		{token.IDENT, "amount"},
		{token.COLON, ":"},
		{token.IDENT, "u64"},
		{token.RPAREN, ")"},
		{token.ARROW, "->"},
		{token.IDENT, "bool"},
		{token.LBRACE, "{"},
		{token.REQUIRE, "require"},
		{token.LPAREN, "("},
		{token.SELF, "self"},
		{token.DOT, "."},
		{token.IDENT, "balance"},
		{token.GTE, ">="},
		{token.IDENT, "amount"},
		{token.RPAREN, ")"},
		{token.SEMICOLON, ";"},
		{token.SELF, "self"},
		{token.DOT, "."},
		{token.IDENT, "balance"},
		{token.MINUSEQ, "-="},
		{token.IDENT, "amount"},
		{token.SEMICOLON, ";"},
		{token.EMIT, "emit"},
		{token.IDENT, "Sent"},
		{token.LPAREN, "("},
		{token.IDENT, "to"},
		{token.COMMA, ","},
		{token.IDENT, "amount"},
		{token.RPAREN, ")"},
		{token.SEMICOLON, ";"},
		{token.RETURN, "return"},
		{token.TRUE, "true"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
		{token.RBRACE, "}"},
	})
}

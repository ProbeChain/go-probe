// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.
//
// The ProbeChain is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Package lexer implements a single-pass, no-backtracking lexer for the PROBE language.
//
// Design principles:
//   - ASCII-only input
//   - BPE-aligned tokenization (tokens map well to LLM BPE tokens)
//   - Single-pass, no backtracking
//   - Support // line comments and /* */ block comments
//   - Hex literals (0x...) produce BYTES tokens
//   - Address literals (@0x...) produce ADDRESS tokens
//   - String literals ("...") support standard escape sequences
package lexer

import (
	"github.com/probechain/go-probe/probe-lang/lang/token"
)

// Lexer holds the state for a single-pass tokenization run.
type Lexer struct {
	filename string
	input    []byte

	// pos is the index into input of the next byte to be loaded into ch.
	// After advance(), ch == input[pos-1] and pos points one past it.
	pos  int
	line int // 1-based current line number
	col  int // 1-based current column number

	ch byte // current character; 0 when past end
}

// New creates a new Lexer for the given filename and input string.
func New(filename, input string) *Lexer {
	l := &Lexer{
		filename: filename,
		input:    []byte(input),
		line:     1,
		col:      0,
	}
	l.advance() // prime l.ch with the first byte
	return l
}

// advance moves to the next byte in the input, updating line/column tracking.
// When the end of input is reached, ch is set to 0.
func (l *Lexer) advance() {
	if l.ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	if l.pos >= len(l.input) {
		l.ch = 0
		return
	}
	l.ch = l.input[l.pos]
	l.pos++
}

// peek returns the byte after the current character without consuming it.
// Returns 0 if at or past end.
func (l *Lexer) peek() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

// currentPos returns a token.Position capturing the lexer's state right now.
// Call this before consuming the first character of a token.
func (l *Lexer) currentPos() token.Position {
	// After advance(), pos is already one past ch, so the byte offset of ch is pos-1.
	return token.Position{
		File:   l.filename,
		Line:   l.line,
		Column: l.col,
		Offset: l.pos - 1,
	}
}

// makeToken constructs a token with the given type, literal, and position.
func makeToken(typ token.Type, literal string, pos token.Position) token.Token {
	return token.Token{Type: typ, Literal: literal, Pos: pos}
}

// skipWhitespace consumes space, tab, carriage return, and newline characters.
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\n' {
		l.advance()
	}
}

// NextToken scans and returns the next token from the input.
// After EOF is reached, subsequent calls continue returning EOF tokens.
func (l *Lexer) NextToken() token.Token {
	l.skipWhitespace()

	pos := l.currentPos()
	ch := l.ch

	if ch == 0 {
		return makeToken(token.EOF, "", pos)
	}

	l.advance() // consume ch; from here on, l.ch is the character AFTER ch

	switch {
	// -------------------------------------------------------------------------
	// Identifiers and keywords
	// -------------------------------------------------------------------------
	case isIdentStart(ch):
		lit := l.readIdentFromFirst(ch)
		typ := token.LookupIdent(lit)
		return makeToken(typ, lit, pos)

	// -------------------------------------------------------------------------
	// Numeric literals
	// -------------------------------------------------------------------------
	case isDigit(ch):
		typ, lit := l.readNumberFromFirst(ch)
		return makeToken(typ, lit, pos)

	// -------------------------------------------------------------------------
	// String literals
	// -------------------------------------------------------------------------
	case ch == '"':
		// The opening '"' has been consumed; read the rest.
		lit, ok := l.readStringBody()
		if !ok {
			return makeToken(token.ILLEGAL, lit, pos)
		}
		return makeToken(token.STRING, lit, pos)

	// -------------------------------------------------------------------------
	// Address literals  @0x...
	// -------------------------------------------------------------------------
	case ch == '@':
		if l.ch == '0' && (l.peek() == 'x' || l.peek() == 'X') {
			// Build "@0x<hexdigits>" directly.
			buf := []byte{'@', '0', l.peek()}
			l.advance() // consume '0'
			l.advance() // consume 'x'/'X'
			for isHexDigit(l.ch) {
				buf = append(buf, l.ch)
				l.advance()
			}
			return makeToken(token.ADDRESS, string(buf), pos)
		}
		return makeToken(token.AT, "@", pos)

	// -------------------------------------------------------------------------
	// Slash: comments or division
	// -------------------------------------------------------------------------
	case ch == '/':
		switch l.ch {
		case '/':
			l.advance() // consume second '/'
			body := l.readLineCommentBody()
			return makeToken(token.COMMENT, "//"+body, pos)
		case '*':
			lit, ok := l.readBlockCommentBody()
			if !ok {
				return makeToken(token.ILLEGAL, lit, pos)
			}
			return makeToken(token.COMMENT, lit, pos)
		case '=':
			l.advance()
			return makeToken(token.SLASHEQ, "/=", pos)
		default:
			return makeToken(token.SLASH, "/", pos)
		}

	// -------------------------------------------------------------------------
	// Arithmetic and compound-assignment operators
	// -------------------------------------------------------------------------
	case ch == '+':
		if l.ch == '=' {
			l.advance()
			return makeToken(token.PLUSEQ, "+=", pos)
		}
		return makeToken(token.PLUS, "+", pos)

	case ch == '-':
		switch l.ch {
		case '=':
			l.advance()
			return makeToken(token.MINUSEQ, "-=", pos)
		case '>':
			l.advance()
			return makeToken(token.ARROW, "->", pos)
		default:
			return makeToken(token.MINUS, "-", pos)
		}

	case ch == '*':
		if l.ch == '=' {
			l.advance()
			return makeToken(token.STAREQ, "*=", pos)
		}
		return makeToken(token.STAR, "*", pos)

	case ch == '%':
		if l.ch == '=' {
			l.advance()
			return makeToken(token.PERCENTEQ, "%=", pos)
		}
		return makeToken(token.PERCENT, "%", pos)

	// -------------------------------------------------------------------------
	// Bitwise / logical operators
	// -------------------------------------------------------------------------
	case ch == '&':
		switch l.ch {
		case '&':
			l.advance()
			return makeToken(token.AND, "&&", pos)
		case '=':
			l.advance()
			return makeToken(token.AMPEQ, "&=", pos)
		default:
			return makeToken(token.AMP, "&", pos)
		}

	case ch == '|':
		switch l.ch {
		case '|':
			l.advance()
			return makeToken(token.OR, "||", pos)
		case '=':
			l.advance()
			return makeToken(token.PIPEEQ, "|=", pos)
		default:
			return makeToken(token.PIPE, "|", pos)
		}

	case ch == '^':
		if l.ch == '=' {
			l.advance()
			return makeToken(token.CARETEQ, "^=", pos)
		}
		return makeToken(token.CARET, "^", pos)

	// -------------------------------------------------------------------------
	// Comparison and assignment operators
	// -------------------------------------------------------------------------
	case ch == '!':
		if l.ch == '=' {
			l.advance()
			return makeToken(token.NEQ, "!=", pos)
		}
		return makeToken(token.BANG, "!", pos)

	case ch == '=':
		switch l.ch {
		case '=':
			l.advance()
			return makeToken(token.EQ, "==", pos)
		case '>':
			l.advance()
			return makeToken(token.FATARROW, "=>", pos)
		default:
			return makeToken(token.ASSIGN, "=", pos)
		}

	case ch == '<':
		switch l.ch {
		case '<':
			l.advance() // consume second '<'
			if l.ch == '=' {
				l.advance()
				return makeToken(token.LSHIFTEQ, "<<=", pos)
			}
			return makeToken(token.LSHIFT, "<<", pos)
		case '=':
			l.advance()
			return makeToken(token.LTE, "<=", pos)
		default:
			return makeToken(token.LT, "<", pos)
		}

	case ch == '>':
		switch l.ch {
		case '>':
			l.advance() // consume second '>'
			if l.ch == '=' {
				l.advance()
				return makeToken(token.RSHIFTEQ, ">>=", pos)
			}
			return makeToken(token.RSHIFT, ">>", pos)
		case '=':
			l.advance()
			return makeToken(token.GTE, ">=", pos)
		default:
			return makeToken(token.GT, ">", pos)
		}

	// -------------------------------------------------------------------------
	// Dot: field access or range (..)
	// -------------------------------------------------------------------------
	case ch == '.':
		if l.ch == '.' {
			l.advance()
			return makeToken(token.DOTDOT, "..", pos)
		}
		return makeToken(token.DOT, ".", pos)

	// -------------------------------------------------------------------------
	// Colon: type annotation (:) or path separator (::)
	// -------------------------------------------------------------------------
	case ch == ':':
		if l.ch == ':' {
			l.advance()
			return makeToken(token.COLONCOLON, "::", pos)
		}
		return makeToken(token.COLON, ":", pos)

	// -------------------------------------------------------------------------
	// Single-character punctuation
	// -------------------------------------------------------------------------
	case ch == '#':
		return makeToken(token.HASH, "#", pos)
	case ch == '~':
		return makeToken(token.TILDE, "~", pos)
	case ch == '(':
		return makeToken(token.LPAREN, "(", pos)
	case ch == ')':
		return makeToken(token.RPAREN, ")", pos)
	case ch == '[':
		return makeToken(token.LBRACKET, "[", pos)
	case ch == ']':
		return makeToken(token.RBRACKET, "]", pos)
	case ch == '{':
		return makeToken(token.LBRACE, "{", pos)
	case ch == '}':
		return makeToken(token.RBRACE, "}", pos)
	case ch == ',':
		return makeToken(token.COMMA, ",", pos)
	case ch == ';':
		return makeToken(token.SEMICOLON, ";", pos)
	}

	// Anything else is ILLEGAL.
	return makeToken(token.ILLEGAL, string([]byte{ch}), pos)
}

// Tokenize returns all tokens (including the final EOF) produced by repeated
// calls to NextToken.
func (l *Lexer) Tokenize() []token.Token {
	var toks []token.Token
	for {
		tok := l.NextToken()
		toks = append(toks, tok)
		if tok.Type == token.EOF {
			break
		}
	}
	return toks
}

// ---------------------------------------------------------------------------
// Internal readers — each assumes the first character has already been
// consumed by the advance() call inside NextToken.
// ---------------------------------------------------------------------------

// readIdentFromFirst builds an identifier literal starting with the already-
// consumed byte `first`, then consuming subsequent ident-continue bytes.
func (l *Lexer) readIdentFromFirst(first byte) string {
	buf := make([]byte, 1, 16)
	buf[0] = first
	for isIdentContinue(l.ch) {
		buf = append(buf, l.ch)
		l.advance()
	}
	return string(buf)
}

// readNumberFromFirst parses an integer, float, or hex-bytes literal given
// the already-consumed first digit `first`.
//
//   - "0x..." or "0X..."  →  BYTES
//   - digits "." digits   →  FLOAT  (with optional exponent)
//   - digits              →  INT
func (l *Lexer) readNumberFromFirst(first byte) (token.Type, string) {
	buf := make([]byte, 1, 24)
	buf[0] = first

	// Hex literal: starts with "0" and next char is 'x'/'X'.
	if first == '0' && (l.ch == 'x' || l.ch == 'X') {
		buf = append(buf, l.ch)
		l.advance() // consume 'x'/'X'
		for isHexDigit(l.ch) {
			buf = append(buf, l.ch)
			l.advance()
		}
		return token.BYTES, string(buf)
	}

	// Accumulate remaining decimal digits.
	for isDigit(l.ch) {
		buf = append(buf, l.ch)
		l.advance()
	}

	// Float: a '.' followed by at least one digit.
	if l.ch == '.' && isDigit(l.peek()) {
		buf = append(buf, '.') // append the '.'
		l.advance()            // consume '.'
		for isDigit(l.ch) {
			buf = append(buf, l.ch)
			l.advance()
		}
		// Optional exponent: e/E, optional sign, one or more digits.
		if l.ch == 'e' || l.ch == 'E' {
			buf = append(buf, l.ch)
			l.advance()
			if l.ch == '+' || l.ch == '-' {
				buf = append(buf, l.ch)
				l.advance()
			}
			for isDigit(l.ch) {
				buf = append(buf, l.ch)
				l.advance()
			}
		}
		return token.FLOAT, string(buf)
	}

	return token.INT, string(buf)
}

// readStringBody reads the content of a string literal after the opening '"'
// has been consumed.  It returns the full literal — including both quote
// characters — and a bool that is false when the string was unterminated.
//
// Standard escape sequences (\n, \t, \\, \", etc.) are preserved verbatim in
// the literal; no decoding is performed at the lexing stage.
func (l *Lexer) readStringBody() (string, bool) {
	buf := make([]byte, 1, 32)
	buf[0] = '"' // re-add the already-consumed opening quote
	for {
		switch l.ch {
		case 0, '\n':
			// Unterminated string.
			return string(buf), false
		case '\\':
			buf = append(buf, '\\')
			l.advance() // consume '\'
			if l.ch == 0 {
				return string(buf), false
			}
			buf = append(buf, l.ch)
			l.advance() // consume the escaped character
		case '"':
			buf = append(buf, '"')
			l.advance() // consume closing '"'
			return string(buf), true
		default:
			buf = append(buf, l.ch)
			l.advance()
		}
	}
}

// readLineCommentBody reads from the current position to end-of-line (not
// including the newline byte).  The "//" prefix has already been consumed.
func (l *Lexer) readLineCommentBody() string {
	var buf []byte
	for l.ch != '\n' && l.ch != 0 {
		buf = append(buf, l.ch)
		l.advance()
	}
	return string(buf)
}

// readBlockCommentBody reads a /* ... */ block comment.  The opening '/' has
// already been consumed; l.ch is currently '*'.  Returns the full literal
// including "/*" and "*/", and false when the comment is unterminated.
func (l *Lexer) readBlockCommentBody() (string, bool) {
	buf := []byte{'/', '*'}
	l.advance() // consume the '*' that opened the block comment
	for {
		switch {
		case l.ch == 0:
			return string(buf), false
		case l.ch == '*' && l.peek() == '/':
			buf = append(buf, '*', '/')
			l.advance() // consume '*'
			l.advance() // consume '/'
			return string(buf), true
		default:
			buf = append(buf, l.ch)
			l.advance()
		}
	}
}

// ---------------------------------------------------------------------------
// Character classification helpers
// ---------------------------------------------------------------------------

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') ||
		(ch >= 'a' && ch <= 'f') ||
		(ch >= 'A' && ch <= 'F')
}

func isIdentStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isIdentContinue(ch byte) bool {
	return isIdentStart(ch) || isDigit(ch)
}

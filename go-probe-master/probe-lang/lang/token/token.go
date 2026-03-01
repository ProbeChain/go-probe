// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.
//
// The ProbeChain is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Package token defines the lexical token types for the PROBE language.
//
// Design principles:
//   - ASCII-only primitives aligned with BPE tokenizers (~70 tokens/task)
//   - J-style arity overloading (monadic/dyadic per symbol)
//   - Brace-based scoping (not whitespace-significant)
//   - Regular or weakly context-free grammar (zero-overhead constrained decoding)
package token

import "fmt"

// Token represents a lexical token.
type Token struct {
	Type    Type
	Literal string
	Pos     Position
}

// Position tracks source location.
type Position struct {
	File   string
	Line   int
	Column int
	Offset int
}

func (p Position) String() string {
	if p.File != "" {
		return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Column)
	}
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

// Type is the set of lexical token types.
type Type int

const (
	// Special tokens
	ILLEGAL Type = iota
	EOF
	COMMENT

	// Literals
	IDENT   // main, x, agent_id
	INT     // 42
	FLOAT   // 3.14
	STRING  // "hello"
	BYTES   // 0xdeadbeef
	ADDRESS // @0x1234...

	// Operators — J-style arity overloading:
	//   monadic (prefix): -x (negate), #x (length), ~x (bitwise not)
	//   dyadic (infix):   x+y, x-y, x*y, etc.
	PLUS     // +  (add / identity)
	MINUS    // -  (sub / negate)
	STAR     // *  (mul / dereference)
	SLASH    // /  (div)
	PERCENT  // %  (mod)
	HASH     // #  (length / tag)
	TILDE    // ~  (bitwise not / bitwise xor)
	AMP      // &  (bitwise and / address-of)
	PIPE     // |  (bitwise or / pipeline)
	CARET    // ^  (power / bitwise xor)
	BANG     // !  (logical not)
	DOT      // .  (field access)
	DOTDOT   // .. (range)
	ARROW    // -> (function return type / move)
	FATARROW // => (match arm)
	LSHIFT   // <<
	RSHIFT   // >>

	// Comparison
	EQ    // ==
	NEQ   // !=
	LT    // <
	GT    // >
	LTE   // <=
	GTE   // >=

	// Assignment
	ASSIGN     // =
	PLUSEQ     // +=
	MINUSEQ    // -=
	STAREQ     // *=
	SLASHEQ    // /=
	PERCENTEQ  // %=
	AMPEQ      // &=
	PIPEEQ     // |=
	CARETEQ    // ^=
	LSHIFTEQ   // <<=
	RSHIFTEQ   // >>=

	// Logical
	AND // &&
	OR  // ||

	// Delimiters
	LPAREN    // (
	RPAREN    // )
	LBRACKET  // [
	RBRACKET  // ]
	LBRACE    // {
	RBRACE    // }
	COMMA     // ,
	SEMICOLON // ;
	COLON     // :
	COLONCOLON // ::
	AT        // @

	// Keywords — agent-first design
	keywordStart
	FN       // fn
	LET      // let
	MUT      // mut
	MOVE     // move (linear type transfer)
	COPY     // copy (explicit clone)
	DROP     // drop (explicit destroy)
	IF       // if
	ELSE     // else
	MATCH    // match
	FOR      // for
	IN       // in
	WHILE    // while
	RETURN   // return
	BREAK    // break
	CONTINUE // continue
	STRUCT   // struct
	ENUM     // enum
	IMPL     // impl
	TRAIT    // trait
	TYPE     // type
	PUB      // pub
	USE      // use
	MOD      // mod
	AS       // as
	SELF     // self
	TRUE     // true
	FALSE    // false
	NIL      // nil

	// Agent-specific keywords
	AGENT    // agent (first-class agent declaration)
	MSG      // msg   (message passing)
	SEND     // send  (send message)
	RECV     // recv  (receive message)
	SPAWN    // spawn (create agent)
	STATE    // state (agent state)

	// Blockchain-specific keywords
	TX       // tx    (transaction context)
	EMIT     // emit  (emit event/log)
	REQUIRE  // require (assertion)
	ASSERT   // assert
	RESOURCE // resource (linear resource type)
	keywordEnd
)

var tokenNames = [...]string{
	ILLEGAL: "ILLEGAL",
	EOF:     "EOF",
	COMMENT: "COMMENT",

	IDENT:   "IDENT",
	INT:     "INT",
	FLOAT:   "FLOAT",
	STRING:  "STRING",
	BYTES:   "BYTES",
	ADDRESS: "ADDRESS",

	PLUS:     "+",
	MINUS:    "-",
	STAR:     "*",
	SLASH:    "/",
	PERCENT:  "%",
	HASH:     "#",
	TILDE:    "~",
	AMP:      "&",
	PIPE:     "|",
	CARET:    "^",
	BANG:     "!",
	DOT:      ".",
	DOTDOT:   "..",
	ARROW:    "->",
	FATARROW: "=>",
	LSHIFT:   "<<",
	RSHIFT:   ">>",

	EQ:  "==",
	NEQ: "!=",
	LT:  "<",
	GT:  ">",
	LTE: "<=",
	GTE: ">=",

	ASSIGN:    "=",
	PLUSEQ:    "+=",
	MINUSEQ:   "-=",
	STAREQ:    "*=",
	SLASHEQ:   "/=",
	PERCENTEQ: "%=",
	AMPEQ:     "&=",
	PIPEEQ:    "|=",
	CARETEQ:   "^=",
	LSHIFTEQ:  "<<=",
	RSHIFTEQ:  ">>=",

	AND: "&&",
	OR:  "||",

	LPAREN:     "(",
	RPAREN:     ")",
	LBRACKET:   "[",
	RBRACKET:   "]",
	LBRACE:     "{",
	RBRACE:     "}",
	COMMA:      ",",
	SEMICOLON:  ";",
	COLON:      ":",
	COLONCOLON: "::",
	AT:         "@",

	FN:       "fn",
	LET:      "let",
	MUT:      "mut",
	MOVE:     "move",
	COPY:     "copy",
	DROP:     "drop",
	IF:       "if",
	ELSE:     "else",
	MATCH:    "match",
	FOR:      "for",
	IN:       "in",
	WHILE:    "while",
	RETURN:   "return",
	BREAK:    "break",
	CONTINUE: "continue",
	STRUCT:   "struct",
	ENUM:     "enum",
	IMPL:     "impl",
	TRAIT:    "trait",
	TYPE:     "type",
	PUB:      "pub",
	USE:      "use",
	MOD:      "mod",
	AS:       "as",
	SELF:     "self",
	TRUE:     "true",
	FALSE:    "false",
	NIL:      "nil",

	AGENT:    "agent",
	MSG:      "msg",
	SEND:     "send",
	RECV:     "recv",
	SPAWN:    "spawn",
	STATE:    "state",

	TX:       "tx",
	EMIT:     "emit",
	REQUIRE:  "require",
	ASSERT:   "assert",
	RESOURCE: "resource",
}

// String returns the string form of a token type.
func (t Type) String() string {
	if int(t) < len(tokenNames) {
		return tokenNames[t]
	}
	return fmt.Sprintf("token(%d)", t)
}

// IsKeyword returns true if the token is a keyword.
func (t Type) IsKeyword() bool {
	return t > keywordStart && t < keywordEnd
}

// IsOperator returns true if the token is an operator.
func (t Type) IsOperator() bool {
	return t >= PLUS && t <= RSHIFT
}

// IsLiteral returns true if the token is a literal value.
func (t Type) IsLiteral() bool {
	return t >= IDENT && t <= ADDRESS
}

// keywords maps keyword strings to token types.
var keywords map[string]Type

func init() {
	keywords = make(map[string]Type)
	for i := keywordStart + 1; i < keywordEnd; i++ {
		keywords[tokenNames[i]] = i
	}
}

// LookupIdent checks if an identifier is a keyword.
func LookupIdent(ident string) Type {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

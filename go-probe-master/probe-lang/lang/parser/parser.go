// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.
//
// The ProbeChain is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Package parser implements a recursive-descent / Pratt parser for the PROBE
// language.
//
// Design overview:
//
//   - Declarations are parsed with straightforward recursive descent.
//   - Expressions are parsed with a Pratt (top-down operator precedence) table.
//   - Errors are collected rather than aborting; the parser attempts to recover
//     by skipping to the next semicolon or closing brace so that subsequent
//     declarations can still be parsed.
//   - Comments produced by the lexer are silently skipped.
package parser

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/probechain/go-probe/probe-lang/lang/ast"
	"github.com/probechain/go-probe/probe-lang/lang/lexer"
	"github.com/probechain/go-probe/probe-lang/lang/token"
)

// ---------------------------------------------------------------------------
// Precedence levels (Pratt)
// ---------------------------------------------------------------------------

type precedence int

const (
	precLowest  precedence = iota // base
	precOr                        // ||
	precAnd                       // &&
	precCmp                       // == != < > <= >=
	precBitOr                     // |
	precBitXor                    // ^
	precBitAnd                    // &
	precShift                     // << >>
	precAdd                       // + -
	precMul                       // * / %
	precPrefix                    // -x !x ~x #x &x *x
	precPostfix                   // . [] () ::
)

// infixPrecedence maps a token type to its infix binding power.
var infixPrecedence = map[token.Type]precedence{
	token.OR:         precOr,
	token.AND:        precAnd,
	token.EQ:         precCmp,
	token.NEQ:        precCmp,
	token.LT:         precCmp,
	token.GT:         precCmp,
	token.LTE:        precCmp,
	token.GTE:        precCmp,
	token.PIPE:       precBitOr,
	token.CARET:      precBitXor,
	token.AMP:        precBitAnd,
	token.LSHIFT:     precShift,
	token.RSHIFT:     precShift,
	token.PLUS:       precAdd,
	token.MINUS:      precAdd,
	token.STAR:       precMul,
	token.SLASH:      precMul,
	token.PERCENT:    precMul,
	token.DOTDOT:     precAdd, // range between additive and mul
	token.DOT:        precPostfix,
	token.LBRACKET:   precPostfix,
	token.LPAREN:     precPostfix,
	token.COLONCOLON: precPostfix,
}

// ---------------------------------------------------------------------------
// Parser
// ---------------------------------------------------------------------------

// Parser holds the mutable state for a single parse run.
type Parser struct {
	lex     *lexer.Lexer
	cur     token.Token // current token
	peek    token.Token // lookahead token
	errors  []error
}

// newParser initialises a Parser from source text.
func newParser(filename, source string) *Parser {
	p := &Parser{
		lex: lexer.New(filename, source),
	}
	// Prime cur and peek, skipping comments.
	p.advance()
	p.advance()
	return p
}

// Parse is the public entry point. It tokenises source, runs the parser, and
// returns the program AST together with any non-fatal errors that were
// collected during parsing.
func Parse(filename, source string) (*ast.Program, []error) {
	p := newParser(filename, source)
	prog := p.parseProgram()
	return prog, p.errors
}

// ---------------------------------------------------------------------------
// Token navigation helpers
// ---------------------------------------------------------------------------

// advance reads the next non-comment token from the lexer into cur/peek.
func (p *Parser) advance() {
	p.cur = p.peek
	for {
		p.peek = p.lex.NextToken()
		if p.peek.Type != token.COMMENT {
			break
		}
	}
}

// expect consumes the current token if it matches typ, otherwise records an
// error and does NOT consume the token.
func (p *Parser) expect(typ token.Type) (token.Token, bool) {
	if p.cur.Type == typ {
		tok := p.cur
		p.advance()
		return tok, true
	}
	p.errorf(p.cur.Pos, "expected %s, got %s (%q)", typ, p.cur.Type, p.cur.Literal)
	return p.cur, false
}

// expectPeek consumes the peek token if it matches typ, returning true.
// Otherwise records an error and returns false without advancing.
func (p *Parser) expectPeek(typ token.Type) bool {
	if p.peek.Type == typ {
		p.advance()
		return true
	}
	p.errorf(p.peek.Pos, "expected %s, got %s (%q)", typ, p.peek.Type, p.peek.Literal)
	return false
}

// curIs returns true if the current token has the given type.
func (p *Parser) curIs(typ token.Type) bool { return p.cur.Type == typ }

// peekIs returns true if the lookahead token has the given type.
func (p *Parser) peekIs(typ token.Type) bool { return p.peek.Type == typ }

// skipTo advances past tokens until one of the given types (or EOF) is the
// current token.  Used for error recovery.
func (p *Parser) skipTo(types ...token.Type) {
	for p.cur.Type != token.EOF {
		for _, t := range types {
			if p.cur.Type == t {
				return
			}
		}
		p.advance()
	}
}

// errorf records a parse error at the given position.
func (p *Parser) errorf(pos token.Position, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	p.errors = append(p.errors, fmt.Errorf("%s: %s", pos, msg))
}

// ---------------------------------------------------------------------------
// Program and declarations
// ---------------------------------------------------------------------------

func (p *Parser) parseProgram() *ast.Program {
	prog := &ast.Program{}
	for !p.curIs(token.EOF) {
		decl := p.parseDeclaration()
		if decl != nil {
			prog.Declarations = append(prog.Declarations, decl)
		}
	}
	return prog
}

// parseDeclaration dispatches to the appropriate declaration parser.
// Unknown tokens trigger an error and single-token skip for recovery.
func (p *Parser) parseDeclaration() ast.Declaration {
	pub := false
	pubTok := p.cur
	if p.curIs(token.PUB) {
		pub = true
		p.advance()
	}

	switch p.cur.Type {
	case token.FN:
		return p.parseFnDecl(pub, pubTok)
	case token.STRUCT:
		return p.parseStructDecl(pub)
	case token.ENUM:
		return p.parseEnumDecl(pub)
	case token.TRAIT:
		return p.parseTraitDecl(pub)
	case token.IMPL:
		if pub {
			p.errorf(pubTok.Pos, "'pub' is not valid before 'impl'")
		}
		return p.parseImplDecl()
	case token.AGENT:
		return p.parseAgentDecl(pub)
	case token.RESOURCE:
		return p.parseResourceDecl(pub)
	case token.TYPE:
		return p.parseTypeDecl(pub)
	case token.USE:
		if pub {
			p.errorf(pubTok.Pos, "'pub' is not valid before 'use'")
		}
		return p.parseUseDecl()
	case token.MOD:
		return p.parseModDecl(pub)
	default:
		p.errorf(p.cur.Pos, "unexpected token %s (%q) at top level", p.cur.Type, p.cur.Literal)
		p.advance() // skip the bad token
		return nil
	}
}

// ---------------------------------------------------------------------------
// fn_decl = [ "pub" ] "fn" IDENT "(" [ param_list ] ")" [ "->" type_expr ] block ;
// ---------------------------------------------------------------------------

func (p *Parser) parseFnDecl(pub bool, _ token.Token) *ast.FnDecl {
	tok := p.cur // 'fn'
	p.advance()

	name := p.cur.Literal
	if _, ok := p.expect(token.IDENT); !ok {
		p.skipTo(token.LBRACE, token.SEMICOLON, token.EOF)
	}

	params := p.parseParamList()

	var retType ast.TypeExpr
	if p.curIs(token.ARROW) {
		p.advance()
		retType = p.parseType()
	}

	body := p.parseBlockExpr()

	return &ast.FnDecl{
		Token:      tok,
		Public:     pub,
		Name:       name,
		Params:     params,
		ReturnType: retType,
		Body:       body,
	}
}

// parseParamList parses "(" [ param { "," param } ] ")" and returns the slice.
func (p *Parser) parseParamList() []ast.Param {
	if _, ok := p.expect(token.LPAREN); !ok {
		return nil
	}
	var params []ast.Param
	for !p.curIs(token.RPAREN) && !p.curIs(token.EOF) {
		param := p.parseParam()
		params = append(params, param)
		if p.curIs(token.COMMA) {
			p.advance()
		} else {
			break
		}
	}
	p.expect(token.RPAREN) //nolint
	return params
}

// parseParam parses a single "[ mut ] IDENT : type" parameter.
// The IDENT may be the keyword 'self' (bare receiver parameter with no type
// annotation), in which case we do not require a colon or type.
func (p *Parser) parseParam() ast.Param {
	mut := false
	if p.curIs(token.MUT) {
		mut = true
		p.advance()
	}
	tok := p.cur
	name := p.cur.Literal

	// Allow 'self' as a special bare parameter (no ": type" required).
	if p.curIs(token.SELF) {
		p.advance()
		// If followed by ": type", parse it; otherwise leave type nil.
		if p.curIs(token.COLON) {
			p.advance()
			typ := p.parseType()
			return ast.Param{Token: tok, Name: name, Mutable: mut, Type: typ}
		}
		return ast.Param{Token: tok, Name: name, Mutable: mut, Type: nil}
	}

	p.expect(token.IDENT) //nolint
	p.expect(token.COLON) //nolint
	typ := p.parseType()
	return ast.Param{Token: tok, Name: name, Mutable: mut, Type: typ}
}

// ---------------------------------------------------------------------------
// struct_decl = [ "pub" ] "struct" IDENT "{" [ field_list ] "}" ;
// ---------------------------------------------------------------------------

func (p *Parser) parseStructDecl(pub bool) *ast.StructDecl {
	tok := p.cur // 'struct'
	p.advance()

	name := p.cur.Literal
	if _, ok := p.expect(token.IDENT); !ok {
		p.skipTo(token.RBRACE, token.EOF)
	}

	if _, ok := p.expect(token.LBRACE); !ok {
		p.skipTo(token.RBRACE, token.EOF)
	}

	fields := p.parseFieldList()
	p.expect(token.RBRACE) //nolint

	return &ast.StructDecl{Token: tok, Public: pub, Name: name, Fields: fields}
}

// parseFieldList parses "field { , field } [,]" until "}" or EOF.
func (p *Parser) parseFieldList() []ast.Field {
	var fields []ast.Field
	for !p.curIs(token.RBRACE) && !p.curIs(token.EOF) {
		f := p.parseField()
		fields = append(fields, f)
		if p.curIs(token.COMMA) {
			p.advance()
		} else {
			break
		}
	}
	return fields
}

// parseField parses "[ pub ] IDENT : type_expr".
func (p *Parser) parseField() ast.Field {
	pub := false
	if p.curIs(token.PUB) {
		pub = true
		p.advance()
	}
	tok := p.cur
	name := p.cur.Literal
	p.expect(token.IDENT) //nolint
	p.expect(token.COLON) //nolint
	typ := p.parseType()
	return ast.Field{Token: tok, Name: name, Public: pub, Type: typ}
}

// ---------------------------------------------------------------------------
// enum_decl = [ "pub" ] "enum" IDENT "{" variant { "," variant } [ "," ] "}" ;
// ---------------------------------------------------------------------------

func (p *Parser) parseEnumDecl(pub bool) *ast.EnumDecl {
	tok := p.cur // 'enum'
	p.advance()

	name := p.cur.Literal
	p.expect(token.IDENT) //nolint
	p.expect(token.LBRACE) //nolint

	var variants []ast.EnumVariant
	for !p.curIs(token.RBRACE) && !p.curIs(token.EOF) {
		v := p.parseEnumVariant()
		variants = append(variants, v)
		if p.curIs(token.COMMA) {
			p.advance()
		} else {
			break
		}
	}
	p.expect(token.RBRACE) //nolint

	return &ast.EnumDecl{Token: tok, Public: pub, Name: name, Variants: variants}
}

func (p *Parser) parseEnumVariant() ast.EnumVariant {
	tok := p.cur
	name := p.cur.Literal
	p.expect(token.IDENT) //nolint

	var fields []ast.TypeExpr
	if p.curIs(token.LPAREN) {
		p.advance()
		for !p.curIs(token.RPAREN) && !p.curIs(token.EOF) {
			fields = append(fields, p.parseType())
			if p.curIs(token.COMMA) {
				p.advance()
			} else {
				break
			}
		}
		p.expect(token.RPAREN) //nolint
	}
	return ast.EnumVariant{Token: tok, Name: name, Fields: fields}
}

// ---------------------------------------------------------------------------
// trait_decl = [ "pub" ] "trait" IDENT "{" { trait_method } "}" ;
// trait_method = "fn" IDENT "(" [ param_list ] ")" [ "->" type_expr ] ";" ;
// ---------------------------------------------------------------------------

func (p *Parser) parseTraitDecl(pub bool) *ast.TraitDecl {
	tok := p.cur // 'trait'
	p.advance()

	name := p.cur.Literal
	p.expect(token.IDENT)  //nolint
	p.expect(token.LBRACE) //nolint

	var methods []ast.TraitMethod
	for !p.curIs(token.RBRACE) && !p.curIs(token.EOF) {
		m := p.parseTraitMethod()
		methods = append(methods, m)
	}
	p.expect(token.RBRACE) //nolint

	return &ast.TraitDecl{Token: tok, Public: pub, Name: name, Methods: methods}
}

func (p *Parser) parseTraitMethod() ast.TraitMethod {
	tok := p.cur // 'fn'
	p.expect(token.FN) //nolint

	name := p.cur.Literal
	p.expect(token.IDENT) //nolint

	params := p.parseParamList()

	var retType ast.TypeExpr
	if p.curIs(token.ARROW) {
		p.advance()
		retType = p.parseType()
	}
	p.expect(token.SEMICOLON) //nolint

	return ast.TraitMethod{Token: tok, Name: name, Params: params, ReturnType: retType}
}

// ---------------------------------------------------------------------------
// impl_decl = "impl" [ IDENT "for" ] type_expr "{" { fn_decl } "}" ;
// ---------------------------------------------------------------------------

func (p *Parser) parseImplDecl() *ast.ImplDecl {
	tok := p.cur // 'impl'
	p.advance()

	traitName := ""
	// Lookahead: if after the current IDENT we see "for", it's a trait impl.
	if p.curIs(token.IDENT) && p.peekIs(token.FOR) {
		traitName = p.cur.Literal
		p.advance() // consume trait name
		p.advance() // consume 'for'
	}

	typeName := p.cur.Literal
	p.expect(token.IDENT)  //nolint
	p.expect(token.LBRACE) //nolint

	var methods []ast.FnDecl
	for !p.curIs(token.RBRACE) && !p.curIs(token.EOF) {
		pub := false
		pubTok := p.cur
		if p.curIs(token.PUB) {
			pub = true
			p.advance()
		}
		if p.curIs(token.FN) {
			m := p.parseFnDecl(pub, pubTok)
			if m != nil {
				methods = append(methods, *m)
			}
		} else {
			p.errorf(p.cur.Pos, "expected 'fn' inside impl block, got %s", p.cur.Type)
			p.advance()
		}
	}
	p.expect(token.RBRACE) //nolint

	return &ast.ImplDecl{Token: tok, Trait: traitName, TypeName: typeName, Methods: methods}
}

// ---------------------------------------------------------------------------
// agent_decl = [ "pub" ] "agent" IDENT "{" [ state_block ] { msg_handler } "}" ;
// ---------------------------------------------------------------------------

func (p *Parser) parseAgentDecl(pub bool) *ast.AgentDecl {
	tok := p.cur // 'agent'
	p.advance()

	name := p.cur.Literal
	p.expect(token.IDENT)  //nolint
	p.expect(token.LBRACE) //nolint

	var stateBlock *ast.AgentStateBlock
	if p.curIs(token.STATE) {
		stateBlock = p.parseAgentStateBlock()
	}

	var handlers []ast.MsgHandler
	for p.curIs(token.MSG) {
		h := p.parseMsgHandler()
		handlers = append(handlers, h)
	}

	p.expect(token.RBRACE) //nolint

	return &ast.AgentDecl{
		Token:    tok,
		Public:   pub,
		Name:     name,
		State:    stateBlock,
		Handlers: handlers,
	}
}

func (p *Parser) parseAgentStateBlock() *ast.AgentStateBlock {
	tok := p.cur // 'state'
	p.advance()
	p.expect(token.LBRACE) //nolint
	fields := p.parseFieldList()
	p.expect(token.RBRACE) //nolint
	return &ast.AgentStateBlock{Token: tok, Fields: fields}
}

func (p *Parser) parseMsgHandler() ast.MsgHandler {
	tok := p.cur // 'msg'
	p.advance()

	name := p.cur.Literal
	p.expect(token.IDENT) //nolint

	params := p.parseParamList()

	body := p.parseBlockExpr()
	return ast.MsgHandler{Token: tok, Name: name, Params: params, Body: body}
}

// ---------------------------------------------------------------------------
// resource_decl = [ "pub" ] "resource" IDENT "{" [ field_list ] "}" ;
// ---------------------------------------------------------------------------

func (p *Parser) parseResourceDecl(pub bool) *ast.ResourceDecl {
	tok := p.cur // 'resource'
	p.advance()

	name := p.cur.Literal
	p.expect(token.IDENT)  //nolint
	p.expect(token.LBRACE) //nolint
	fields := p.parseFieldList()
	p.expect(token.RBRACE) //nolint

	return &ast.ResourceDecl{Token: tok, Public: pub, Name: name, Fields: fields}
}

// ---------------------------------------------------------------------------
// type_decl = [ "pub" ] "type" IDENT "=" type_expr ";" ;
// ---------------------------------------------------------------------------

func (p *Parser) parseTypeDecl(pub bool) *ast.TypeDecl {
	tok := p.cur // 'type'
	p.advance()

	name := p.cur.Literal
	p.expect(token.IDENT)  //nolint
	p.expect(token.ASSIGN) //nolint
	typ := p.parseType()
	p.expect(token.SEMICOLON) //nolint

	return &ast.TypeDecl{Token: tok, Public: pub, Name: name, Type: typ}
}

// ---------------------------------------------------------------------------
// use_decl = "use" path [ "::" ( IDENT | "*" ) ] ";" ;
// ---------------------------------------------------------------------------

func (p *Parser) parseUseDecl() *ast.UseDecl {
	tok := p.cur // 'use'
	p.advance()

	var path []string
	for p.curIs(token.IDENT) {
		path = append(path, p.cur.Literal)
		p.advance()
		if p.curIs(token.COLONCOLON) {
			p.advance()
		} else {
			break
		}
	}

	alias := ""
	if p.curIs(token.AS) {
		p.advance()
		alias = p.cur.Literal
		p.expect(token.IDENT) //nolint
	}

	p.expect(token.SEMICOLON) //nolint
	return &ast.UseDecl{Token: tok, Path: path, Alias: alias}
}

// ---------------------------------------------------------------------------
// mod_decl = [ "pub" ] "mod" IDENT ( "{" { declaration } "}" | ";" ) ;
// ---------------------------------------------------------------------------

func (p *Parser) parseModDecl(pub bool) *ast.ModDecl {
	tok := p.cur // 'mod'
	p.advance()

	name := p.cur.Literal
	p.expect(token.IDENT) //nolint

	var decls []ast.Declaration
	if p.curIs(token.LBRACE) {
		p.advance()
		for !p.curIs(token.RBRACE) && !p.curIs(token.EOF) {
			d := p.parseDeclaration()
			if d != nil {
				decls = append(decls, d)
			}
		}
		p.expect(token.RBRACE) //nolint
	} else {
		p.expect(token.SEMICOLON) //nolint
		decls = nil // external module
	}

	return &ast.ModDecl{Token: tok, Public: pub, Name: name, Declarations: decls}
}

// ---------------------------------------------------------------------------
// Type expressions
// ---------------------------------------------------------------------------

// parseType parses a type expression.
//
// type_expr = named_type | path_type | array_type | slice_type | ref_type | fn_type
func (p *Parser) parseType() ast.TypeExpr {
	switch p.cur.Type {
	case token.LBRACKET:
		return p.parseArrayOrSliceType()
	case token.AMP:
		return p.parseRefType()
	case token.FN:
		return p.parseFnType()
	case token.IDENT:
		return p.parseNamedOrPathType()
	default:
		p.errorf(p.cur.Pos, "expected type expression, got %s (%q)", p.cur.Type, p.cur.Literal)
		// return a placeholder to keep parsing
		tok := p.cur
		return &ast.NamedType{Token: tok, Name: tok.Literal}
	}
}

// parseNamedOrPathType handles "IDENT" or "IDENT :: IDENT { :: IDENT }".
func (p *Parser) parseNamedOrPathType() ast.TypeExpr {
	tok := p.cur
	first := p.cur.Literal
	p.advance()

	if !p.curIs(token.COLONCOLON) {
		return &ast.NamedType{Token: tok, Name: first}
	}

	// Path type.
	segments := []string{first}
	for p.curIs(token.COLONCOLON) {
		p.advance()
		if !p.curIs(token.IDENT) {
			p.errorf(p.cur.Pos, "expected identifier after '::'")
			break
		}
		segments = append(segments, p.cur.Literal)
		p.advance()
	}
	return &ast.PathType{Token: tok, Segments: segments}
}

// parseArrayOrSliceType handles "[T; N]" (array) or "[T]" (slice).
func (p *Parser) parseArrayOrSliceType() ast.TypeExpr {
	tok := p.cur // '['
	p.advance()

	elem := p.parseType()

	if p.curIs(token.SEMICOLON) {
		// Array type: [T; N]
		p.advance()
		size := p.parseExpression(precLowest)
		p.expect(token.RBRACKET) //nolint
		return &ast.ArrayType{Token: tok, Elem: elem, Size: size}
	}

	// Slice type: [T]
	p.expect(token.RBRACKET) //nolint
	return &ast.SliceType{Token: tok, Elem: elem}
}

// parseRefType handles "&T" and "&mut T".
func (p *Parser) parseRefType() ast.TypeExpr {
	tok := p.cur // '&'
	p.advance()

	if p.curIs(token.MUT) {
		p.advance()
		elem := p.parseType()
		return &ast.MutRefType{Token: tok, Elem: elem}
	}
	elem := p.parseType()
	return &ast.RefType{Token: tok, Elem: elem}
}

// parseFnType handles "fn ( [T {, T}] ) [-> R]".
func (p *Parser) parseFnType() ast.TypeExpr {
	tok := p.cur // 'fn'
	p.advance()
	p.expect(token.LPAREN) //nolint

	var params []ast.TypeExpr
	for !p.curIs(token.RPAREN) && !p.curIs(token.EOF) {
		params = append(params, p.parseType())
		if p.curIs(token.COMMA) {
			p.advance()
		} else {
			break
		}
	}
	p.expect(token.RPAREN) //nolint

	var retType ast.TypeExpr
	if p.curIs(token.ARROW) {
		p.advance()
		retType = p.parseType()
	}
	return &ast.FnType{Token: tok, ParamTypes: params, ReturnType: retType}
}

// ---------------------------------------------------------------------------
// Statements
// ---------------------------------------------------------------------------

// parseStatement parses a single statement and returns it.
// Returns nil if no statement could be parsed (for error recovery).
func (p *Parser) parseStatement() ast.Statement {
	switch p.cur.Type {
	case token.LET:
		return p.parseLetStmt()
	case token.RETURN:
		return p.parseReturnStmt()
	case token.FOR:
		return p.parseForStmt()
	case token.WHILE:
		return p.parseWhileStmt()
	case token.BREAK:
		tok := p.cur
		p.advance()
		p.expect(token.SEMICOLON) //nolint
		return &ast.BreakStmt{Token: tok}
	case token.CONTINUE:
		tok := p.cur
		p.advance()
		p.expect(token.SEMICOLON) //nolint
		return &ast.ContinueStmt{Token: tok}
	case token.DROP:
		return p.parseDropStmt()
	case token.EMIT:
		return p.parseEmitStmt()
	case token.REQUIRE:
		return p.parseRequireStmt()
	default:
		return p.parseExprOrAssignStmt()
	}
}

// parseLetStmt parses "let [mut] name [: type] = expr ;".
func (p *Parser) parseLetStmt() *ast.LetStmt {
	tok := p.cur // 'let'
	p.advance()

	mut := false
	if p.curIs(token.MUT) {
		mut = true
		p.advance()
	}

	nameTok := p.cur
	name := p.cur.Literal
	p.expect(token.IDENT) //nolint

	var typ ast.TypeExpr
	if p.curIs(token.COLON) {
		p.advance()
		typ = p.parseType()
	}

	var val ast.Expression
	if p.curIs(token.ASSIGN) {
		p.advance()
		val = p.parseExpression(precLowest)
	}

	p.expect(token.SEMICOLON) //nolint

	return &ast.LetStmt{
		Token:   tok,
		Mutable: mut,
		Name:    &ast.Ident{Token: nameTok, Value: name},
		Type:    typ,
		Value:   val,
	}
}

// parseReturnStmt parses "return [expr] ;".
func (p *Parser) parseReturnStmt() *ast.ReturnStmt {
	tok := p.cur // 'return'
	p.advance()

	var val ast.Expression
	if !p.curIs(token.SEMICOLON) && !p.curIs(token.RBRACE) && !p.curIs(token.EOF) {
		val = p.parseExpression(precLowest)
	}
	p.expect(token.SEMICOLON) //nolint
	return &ast.ReturnStmt{Token: tok, Value: val}
}

// parseForStmt parses "for IDENT in expr block".
func (p *Parser) parseForStmt() *ast.ForStmt {
	tok := p.cur // 'for'
	p.advance()

	bindTok := p.cur
	bindName := p.cur.Literal
	p.expect(token.IDENT) //nolint
	p.expect(token.IN)    //nolint

	iter := p.parseExpression(precLowest)
	body := p.parseBlockExpr()

	return &ast.ForStmt{
		Token:    tok,
		Binding:  &ast.Ident{Token: bindTok, Value: bindName},
		Iterable: iter,
		Body:     body,
	}
}

// parseWhileStmt parses "while expr block".
func (p *Parser) parseWhileStmt() *ast.WhileStmt {
	tok := p.cur // 'while'
	p.advance()

	cond := p.parseExpression(precLowest)
	body := p.parseBlockExpr()

	return &ast.WhileStmt{Token: tok, Condition: cond, Body: body}
}

// parseDropStmt parses "drop IDENT ;".
func (p *Parser) parseDropStmt() *ast.DropStmt {
	tok := p.cur // 'drop'
	p.advance()

	nameTok := p.cur
	name := p.cur.Literal
	p.expect(token.IDENT)     //nolint
	p.expect(token.SEMICOLON) //nolint

	return &ast.DropStmt{Token: tok, Value: &ast.Ident{Token: nameTok, Value: name}}
}

// parseEmitStmt parses "emit IDENT { field_init_list } ;".
func (p *Parser) parseEmitStmt() *ast.EmitStmt {
	tok := p.cur // 'emit'
	p.advance()

	event := p.cur.Literal
	p.expect(token.IDENT)  //nolint
	p.expect(token.LBRACE) //nolint

	fields := p.parseFieldInitList()

	p.expect(token.RBRACE)    //nolint
	p.expect(token.SEMICOLON) //nolint

	return &ast.EmitStmt{Token: tok, Event: event, Fields: fields}
}

// parseRequireStmt parses "require ( expr [, expr] ) ;".
func (p *Parser) parseRequireStmt() *ast.RequireStmt {
	tok := p.cur // 'require'
	p.advance()
	p.expect(token.LPAREN) //nolint

	cond := p.parseExpression(precLowest)

	var msg ast.Expression
	if p.curIs(token.COMMA) {
		p.advance()
		msg = p.parseExpression(precLowest)
	}

	p.expect(token.RPAREN)    //nolint
	p.expect(token.SEMICOLON) //nolint

	return &ast.RequireStmt{Token: tok, Condition: cond, Message: msg}
}

// parseExprOrAssignStmt parses either:
//   - an assignment: expr assign_op expr ;
//   - an expression statement: expr ;
func (p *Parser) parseExprOrAssignStmt() ast.Statement {
	tok := p.cur
	expr := p.parseExpression(precLowest)

	// Check for assignment operators.
	assignOp := ""
	switch p.cur.Type {
	case token.ASSIGN:
		assignOp = "="
	case token.PLUSEQ:
		assignOp = "+="
	case token.MINUSEQ:
		assignOp = "-="
	case token.STAREQ:
		assignOp = "*="
	case token.SLASHEQ:
		assignOp = "/="
	case token.PERCENTEQ:
		assignOp = "%="
	case token.AMPEQ:
		assignOp = "&="
	case token.PIPEEQ:
		assignOp = "|="
	case token.CARETEQ:
		assignOp = "^="
	case token.LSHIFTEQ:
		assignOp = "<<="
	case token.RSHIFTEQ:
		assignOp = ">>="
	}

	if assignOp != "" {
		opTok := p.cur
		p.advance() // consume the assignment operator
		rhs := p.parseExpression(precLowest)
		p.expect(token.SEMICOLON) //nolint
		return &ast.AssignStmt{Token: opTok, Target: expr, Operator: assignOp, Value: rhs}
	}

	p.expect(token.SEMICOLON) //nolint
	return &ast.ExprStmt{Token: tok, Expression: expr}
}

// ---------------------------------------------------------------------------
// Block expression
// ---------------------------------------------------------------------------

// parseBlockExpr parses "{ { statement } [ expr ] }".
func (p *Parser) parseBlockExpr() *ast.BlockExpr {
	tok := p.cur // '{'
	if _, ok := p.expect(token.LBRACE); !ok {
		return &ast.BlockExpr{Token: tok}
	}

	var stmts []ast.Statement
	var tail ast.Expression

	for !p.curIs(token.RBRACE) && !p.curIs(token.EOF) {
		// A trailing expression (no semicolon at end) becomes the block's tail.
		// We try to parse a statement; if the next token after an expression is
		// '}' (no semicolon consumed), we backtrack conceptually by noting that
		// parseExprOrAssignStmt will error on the missing semicolon.
		//
		// Strategy: parse an expression first; if the current token afterward is
		// '}' (or EOF) and no semicolon appeared, treat it as the tail.
		if isStatementStart(p.cur.Type) {
			stmt := p.parseStatement()
			if stmt != nil {
				stmts = append(stmts, stmt)
			}
		} else {
			// Try to parse as expression — may be tail or stmt.
			expr := p.parseExpression(precLowest)

			// Check assignment operators.
			assignOp := p.curAssignOp()
			if assignOp != "" {
				opTok := p.cur
				p.advance()
				rhs := p.parseExpression(precLowest)
				p.expect(token.SEMICOLON) //nolint
				stmts = append(stmts, &ast.AssignStmt{
					Token: opTok, Target: expr, Operator: assignOp, Value: rhs,
				})
			} else if p.curIs(token.SEMICOLON) {
				p.advance()
				stmts = append(stmts, &ast.ExprStmt{Token: p.cur, Expression: expr})
			} else {
				// No semicolon — treat as block tail.
				tail = expr
				break
			}
		}
	}

	p.expect(token.RBRACE) //nolint
	return &ast.BlockExpr{Token: tok, Statements: stmts, Tail: tail}
}

// curAssignOp returns the assignment operator string if the current token is
// an assignment operator, otherwise "".
func (p *Parser) curAssignOp() string {
	switch p.cur.Type {
	case token.ASSIGN:
		return "="
	case token.PLUSEQ:
		return "+="
	case token.MINUSEQ:
		return "-="
	case token.STAREQ:
		return "*="
	case token.SLASHEQ:
		return "/="
	case token.PERCENTEQ:
		return "%="
	case token.AMPEQ:
		return "&="
	case token.PIPEEQ:
		return "|="
	case token.CARETEQ:
		return "^="
	case token.LSHIFTEQ:
		return "<<="
	case token.RSHIFTEQ:
		return ">>="
	}
	return ""
}

// isStatementStart returns true for token types that unambiguously begin a
// statement (and cannot be the start of a trailing expression).
func isStatementStart(t token.Type) bool {
	switch t {
	case token.LET, token.RETURN, token.FOR, token.WHILE,
		token.BREAK, token.CONTINUE, token.DROP, token.EMIT, token.REQUIRE:
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// Expression parsing — Pratt / TDOP
// ---------------------------------------------------------------------------

// parseExpression is the Pratt entry point.  It parses a prefix expression
// first, then repeatedly consumes infix/postfix operators whose precedence is
// strictly greater than `prec`.
func (p *Parser) parseExpression(prec precedence) ast.Expression {
	left := p.parsePrefix()
	if left == nil {
		return nil
	}

	for {
		// Postfix operators (DOT, LBRACKET, LPAREN, COLONCOLON) are handled
		// inside parsePostfix which is called here with the current left.
		infixPrec, hasInfix := infixPrecedence[p.cur.Type]
		if !hasInfix || infixPrec <= prec {
			break
		}

		left = p.parseInfix(left, infixPrec)
		if left == nil {
			break
		}
	}

	return left
}

// parsePrefix dispatches to the handler for the current token when it appears
// at prefix (left-edge) position.
func (p *Parser) parsePrefix() ast.Expression {
	switch p.cur.Type {
	// Literals
	case token.INT:
		return p.parseIntLiteral()
	case token.FLOAT:
		return p.parseFloatLiteral()
	case token.STRING:
		return p.parseStringLiteral()
	case token.BYTES:
		return p.parseBytesLiteral()
	case token.ADDRESS:
		return p.parseAddressLiteral()
	case token.TRUE:
		tok := p.cur
		p.advance()
		return &ast.BoolLiteral{Token: tok, Value: true}
	case token.FALSE:
		tok := p.cur
		p.advance()
		return &ast.BoolLiteral{Token: tok, Value: false}
	case token.NIL:
		tok := p.cur
		p.advance()
		return &ast.NilLiteral{Token: tok}

	// Identifier / self
	case token.IDENT:
		tok := p.cur
		p.advance()
		return &ast.Ident{Token: tok, Value: tok.Literal}
	case token.SELF:
		tok := p.cur
		p.advance()
		return &ast.Ident{Token: tok, Value: "self"}

	// Prefix operators
	case token.MINUS, token.BANG, token.TILDE, token.HASH, token.AMP, token.STAR:
		return p.parsePrefixExpr()

	// Grouped expression
	case token.LPAREN:
		return p.parseGroupedExpr()

	// Block expression
	case token.LBRACE:
		return p.parseBlockExpr()

	// Control-flow expressions
	case token.IF:
		return p.parseIfExpr()
	case token.MATCH:
		return p.parseMatchExpr()

	// Array literal
	case token.LBRACKET:
		return p.parseArrayExpr()

	// Agent / linear-type expressions
	case token.MOVE:
		return p.parseMoveExpr()
	case token.COPY:
		return p.parseCopyExpr()
	case token.SPAWN:
		return p.parseSpawnExpr()
	case token.SEND:
		return p.parseSendExpr()
	case token.RECV:
		tok := p.cur
		p.advance()
		return &ast.RecvExpr{Token: tok}

	default:
		p.errorf(p.cur.Pos, "unexpected token %s (%q) in expression", p.cur.Type, p.cur.Literal)
		tok := p.cur
		p.advance()
		// Return a placeholder ident so callers always get a non-nil node.
		return &ast.Ident{Token: tok, Value: tok.Literal}
	}
}

// parseInfix handles operators that appear between two expressions.
// left is the already-parsed left-hand operand.
func (p *Parser) parseInfix(left ast.Expression, prec precedence) ast.Expression {
	switch p.cur.Type {
	// Standard binary operators.
	case token.PLUS, token.MINUS, token.STAR, token.SLASH, token.PERCENT,
		token.OR, token.AND,
		token.EQ, token.NEQ, token.LT, token.GT, token.LTE, token.GTE,
		token.PIPE, token.CARET, token.AMP,
		token.LSHIFT, token.RSHIFT:
		return p.parseBinaryExpr(left, prec)

	// Range: a..b  (parsed as infix with precAdd level)
	case token.DOTDOT:
		tok := p.cur
		p.advance()
		right := p.parseExpression(prec) // right-associative: use same prec
		return &ast.RangeExpr{Token: tok, Start: left, End: right}

	// Postfix: .field or .method(args)
	case token.DOT:
		return p.parseDotExpr(left)

	// Postfix: [index]
	case token.LBRACKET:
		return p.parseIndexExpr(left)

	// Postfix: (args) — free call
	case token.LPAREN:
		return p.parseCallExpr(left)

	// Postfix: ::segment
	case token.COLONCOLON:
		return p.parsePathExpr(left)

	default:
		// Should not be reached given the guard in parseExpression.
		return left
	}
}

// parseBinaryExpr parses a left-associative binary infix expression.
func (p *Parser) parseBinaryExpr(left ast.Expression, prec precedence) ast.Expression {
	tok := p.cur
	op := p.cur.Literal
	p.advance()
	right := p.parseExpression(prec) // left-associative: same prec cuts off
	return &ast.InfixExpr{Token: tok, Left: left, Operator: op, Right: right}
}

// parseDotExpr handles ".field" and ".method(args)".
func (p *Parser) parseDotExpr(left ast.Expression) ast.Expression {
	tok := p.cur // '.'
	p.advance()

	if !p.curIs(token.IDENT) {
		p.errorf(p.cur.Pos, "expected field name after '.', got %s", p.cur.Type)
		return left
	}
	field := p.cur.Literal
	p.advance()

	// Method call: receiver.method(args)
	if p.curIs(token.LPAREN) {
		p.advance() // consume '('
		args := p.parseArgList()
		p.expect(token.RPAREN) //nolint
		return &ast.MethodCallExpr{
			Token:     tok,
			Receiver:  left,
			Method:    field,
			Arguments: args,
		}
	}

	return &ast.FieldExpr{Token: tok, Object: left, Field: field}
}

// parseIndexExpr handles "left[index]".
func (p *Parser) parseIndexExpr(left ast.Expression) ast.Expression {
	tok := p.cur // '['
	p.advance()
	index := p.parseExpression(precLowest)
	p.expect(token.RBRACKET) //nolint
	return &ast.IndexExpr{Token: tok, Left: left, Index: index}
}

// parseCallExpr handles "left(args)".
func (p *Parser) parseCallExpr(left ast.Expression) ast.Expression {
	tok := p.cur // '('
	p.advance()
	args := p.parseArgList()
	p.expect(token.RPAREN) //nolint
	return &ast.CallExpr{Token: tok, Function: left, Arguments: args}
}

// parsePathExpr handles "left::segment", turning a simple Ident into a
// path-like expression represented as an Ident with a "::" joined name.
// (Full path-type semantics are handled at the type level; here we produce an
// Ident so the expression layer can chain further postfix operators.)
func (p *Parser) parsePathExpr(left ast.Expression) ast.Expression {
	tok := p.cur // '::'
	p.advance()
	if !p.curIs(token.IDENT) {
		p.errorf(p.cur.Pos, "expected identifier after '::'")
		return left
	}

	// Build a new Ident whose value concatenates the left side with "::name".
	segment := p.cur.Literal
	p.advance()

	leftStr := left.String()
	return &ast.Ident{Token: tok, Value: leftStr + "::" + segment}
}

// parseArgList parses a comma-separated list of expressions until ')'.
func (p *Parser) parseArgList() []ast.Expression {
	var args []ast.Expression
	for !p.curIs(token.RPAREN) && !p.curIs(token.EOF) {
		args = append(args, p.parseExpression(precLowest))
		if p.curIs(token.COMMA) {
			p.advance()
		} else {
			break
		}
	}
	return args
}

// parseFieldInitList parses "IDENT : expr { , IDENT : expr } [,]" inside
// braces (used by spawn and emit).
func (p *Parser) parseFieldInitList() map[string]ast.Expression {
	fields := make(map[string]ast.Expression)
	for !p.curIs(token.RBRACE) && !p.curIs(token.EOF) {
		name := p.cur.Literal
		p.expect(token.IDENT) //nolint
		p.expect(token.COLON) //nolint
		val := p.parseExpression(precLowest)
		fields[name] = val
		if p.curIs(token.COMMA) {
			p.advance()
		} else {
			break
		}
	}
	return fields
}

// ---------------------------------------------------------------------------
// Prefix expression parsers
// ---------------------------------------------------------------------------

func (p *Parser) parsePrefixExpr() *ast.PrefixExpr {
	tok := p.cur
	op := p.cur.Literal
	p.advance()
	right := p.parseExpression(precPrefix)
	return &ast.PrefixExpr{Token: tok, Operator: op, Right: right}
}

func (p *Parser) parseGroupedExpr() ast.Expression {
	p.advance() // consume '('
	expr := p.parseExpression(precLowest)
	p.expect(token.RPAREN) //nolint
	return expr
}

func (p *Parser) parseIfExpr() *ast.IfExpr {
	tok := p.cur // 'if'
	p.advance()

	cond := p.parseExpression(precLowest)
	consequence := p.parseBlockExpr()

	var alt ast.Expression
	if p.curIs(token.ELSE) {
		p.advance()
		if p.curIs(token.IF) {
			alt = p.parseIfExpr()
		} else {
			alt = p.parseBlockExpr()
		}
	}

	return &ast.IfExpr{Token: tok, Condition: cond, Consequence: consequence, Alternative: alt}
}

func (p *Parser) parseMatchExpr() *ast.MatchExpr {
	tok := p.cur // 'match'
	p.advance()

	subject := p.parseExpression(precLowest)
	p.expect(token.LBRACE) //nolint

	var arms []ast.MatchArm
	for !p.curIs(token.RBRACE) && !p.curIs(token.EOF) {
		arm := p.parseMatchArm()
		arms = append(arms, arm)
		if p.curIs(token.COMMA) {
			p.advance()
		} else {
			break
		}
	}
	p.expect(token.RBRACE) //nolint

	return &ast.MatchExpr{Token: tok, Subject: subject, Arms: arms}
}

func (p *Parser) parseMatchArm() ast.MatchArm {
	tok := p.cur
	pattern := p.parsePattern()

	var guard ast.Expression
	if p.curIs(token.IF) {
		p.advance()
		guard = p.parseExpression(precLowest)
	}

	p.expect(token.FATARROW) //nolint

	var body ast.Expression
	if p.curIs(token.LBRACE) {
		body = p.parseBlockExpr()
	} else {
		body = p.parseExpression(precLowest)
	}

	return ast.MatchArm{Token: tok, Pattern: pattern, Guard: guard, Body: body}
}

// parsePattern parses a match arm pattern (simplified: ident, literal, or
// ident "(" patterns ")").
func (p *Parser) parsePattern() ast.Expression {
	switch p.cur.Type {
	case token.INT:
		return p.parseIntLiteral()
	case token.STRING:
		return p.parseStringLiteral()
	case token.TRUE:
		tok := p.cur
		p.advance()
		return &ast.BoolLiteral{Token: tok, Value: true}
	case token.FALSE:
		tok := p.cur
		p.advance()
		return &ast.BoolLiteral{Token: tok, Value: false}
	case token.IDENT:
		tok := p.cur
		name := p.cur.Literal
		p.advance()
		// Enum-variant pattern: Name(sub-patterns)
		if p.curIs(token.LPAREN) {
			p.advance()
			var subPats []ast.Expression
			for !p.curIs(token.RPAREN) && !p.curIs(token.EOF) {
				subPats = append(subPats, p.parsePattern())
				if p.curIs(token.COMMA) {
					p.advance()
				} else {
					break
				}
			}
			p.expect(token.RPAREN) //nolint
			// Represent as CallExpr with Ident callee.
			return &ast.CallExpr{
				Token:     tok,
				Function:  &ast.Ident{Token: tok, Value: name},
				Arguments: subPats,
			}
		}
		return &ast.Ident{Token: tok, Value: name}
	default:
		// Wildcard or unknown — return a placeholder ident.
		tok := p.cur
		p.advance()
		return &ast.Ident{Token: tok, Value: tok.Literal}
	}
}

func (p *Parser) parseArrayExpr() *ast.ArrayExpr {
	tok := p.cur // '['
	p.advance()

	var elems []ast.Expression
	for !p.curIs(token.RBRACKET) && !p.curIs(token.EOF) {
		elems = append(elems, p.parseExpression(precLowest))
		if p.curIs(token.COMMA) {
			p.advance()
		} else {
			break
		}
	}
	p.expect(token.RBRACKET) //nolint
	return &ast.ArrayExpr{Token: tok, Elements: elems}
}

func (p *Parser) parseMoveExpr() *ast.MoveExpr {
	tok := p.cur
	p.advance()
	val := p.parseExpression(precPrefix)
	return &ast.MoveExpr{Token: tok, Value: val}
}

func (p *Parser) parseCopyExpr() *ast.CopyExpr {
	tok := p.cur
	p.advance()
	val := p.parseExpression(precPrefix)
	return &ast.CopyExpr{Token: tok, Value: val}
}

func (p *Parser) parseSpawnExpr() *ast.SpawnExpr {
	tok := p.cur // 'spawn'
	p.advance()

	agent := p.cur.Literal
	p.expect(token.IDENT)  //nolint
	p.expect(token.LBRACE) //nolint

	fields := p.parseFieldInitList()
	p.expect(token.RBRACE) //nolint

	return &ast.SpawnExpr{Token: tok, Agent: agent, Fields: fields}
}

func (p *Parser) parseSendExpr() *ast.SendExpr {
	tok := p.cur // 'send'
	p.advance()

	target := p.parseExpression(precPrefix)
	msg := p.parseExpression(precPrefix)

	return &ast.SendExpr{Token: tok, Target: target, Message: msg}
}

// ---------------------------------------------------------------------------
// Literal parsers
// ---------------------------------------------------------------------------

func (p *Parser) parseIntLiteral() *ast.IntLiteral {
	tok := p.cur
	var val int64
	var err error
	lit := tok.Literal
	if strings.HasPrefix(lit, "0x") || strings.HasPrefix(lit, "0X") {
		val, err = strconv.ParseInt(lit[2:], 16, 64)
	} else {
		val, err = strconv.ParseInt(lit, 10, 64)
	}
	if err != nil {
		// Overflow — store 0 and note the error.
		p.errorf(tok.Pos, "integer literal %q overflows int64: %v", lit, err)
	}
	p.advance()
	return &ast.IntLiteral{Token: tok, Value: val}
}

func (p *Parser) parseFloatLiteral() *ast.FloatLiteral {
	tok := p.cur
	val, err := strconv.ParseFloat(tok.Literal, 64)
	if err != nil {
		p.errorf(tok.Pos, "invalid float literal %q: %v", tok.Literal, err)
	}
	p.advance()
	return &ast.FloatLiteral{Token: tok, Value: val}
}

func (p *Parser) parseStringLiteral() *ast.StringLiteral {
	tok := p.cur
	// The lexer stores the literal including surrounding quotes.
	// Strip the quotes for the Value field.
	lit := tok.Literal
	inner := ""
	if len(lit) >= 2 && lit[0] == '"' && lit[len(lit)-1] == '"' {
		inner = lit[1 : len(lit)-1]
	} else {
		inner = lit
	}
	p.advance()
	return &ast.StringLiteral{Token: tok, Value: inner}
}

func (p *Parser) parseBytesLiteral() *ast.BytesLiteral {
	tok := p.cur
	lit := tok.Literal
	// The literal is "0x<hexdigits>".
	hexStr := ""
	if strings.HasPrefix(lit, "0x") || strings.HasPrefix(lit, "0X") {
		hexStr = lit[2:]
	} else {
		hexStr = lit
	}
	// Pad to even length for hex.DecodeString.
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		p.errorf(tok.Pos, "invalid bytes literal %q: %v", lit, err)
	}
	p.advance()
	return &ast.BytesLiteral{Token: tok, Value: b}
}

func (p *Parser) parseAddressLiteral() *ast.AddressLiteral {
	tok := p.cur
	p.advance()
	return &ast.AddressLiteral{Token: tok, Value: tok.Literal}
}

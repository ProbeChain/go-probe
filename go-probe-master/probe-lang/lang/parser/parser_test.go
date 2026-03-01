// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.
//
// The ProbeChain is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package parser

import (
	"strings"
	"testing"

	"github.com/probechain/go-probe/probe-lang/lang/ast"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// mustParse asserts that the source parses without errors and returns the
// program. If there are errors it fails the test immediately.
func mustParse(t *testing.T, src string) *ast.Program {
	t.Helper()
	prog, errs := Parse("test.probe", src)
	if len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		t.Fatalf("unexpected parse errors:\n%s", strings.Join(msgs, "\n"))
	}
	return prog
}

// parseWithErrors parses and expects at least one error to be reported.
// It returns both the (partial) program and the error slice.
func parseWithErrors(t *testing.T, src string) (*ast.Program, []error) {
	t.Helper()
	prog, errs := Parse("test.probe", src)
	if len(errs) == 0 {
		t.Fatal("expected parse errors, but none were reported")
	}
	return prog, errs
}

// firstDecl returns the first declaration in prog, failing if there is none.
func firstDecl(t *testing.T, prog *ast.Program) ast.Declaration {
	t.Helper()
	if len(prog.Declarations) == 0 {
		t.Fatal("expected at least one declaration in program, got none")
	}
	return prog.Declarations[0]
}

// ---------------------------------------------------------------------------
// Simple function
// ---------------------------------------------------------------------------

func TestParseFnDecl_Simple(t *testing.T) {
	src := `fn add(a: u64, b: u64) -> u64 { a + b }`
	prog := mustParse(t, src)

	fn, ok := firstDecl(t, prog).(*ast.FnDecl)
	if !ok {
		t.Fatalf("expected *ast.FnDecl, got %T", firstDecl(t, prog))
	}

	if fn.Name != "add" {
		t.Errorf("fn name: want %q, got %q", "add", fn.Name)
	}
	if fn.Public {
		t.Error("fn should not be public")
	}
	if len(fn.Params) != 2 {
		t.Fatalf("want 2 params, got %d", len(fn.Params))
	}
	if fn.Params[0].Name != "a" || fn.Params[1].Name != "b" {
		t.Errorf("params: want a, b got %q, %q", fn.Params[0].Name, fn.Params[1].Name)
	}
	if fn.ReturnType == nil {
		t.Fatal("expected return type, got nil")
	}
	if fn.ReturnType.String() != "u64" {
		t.Errorf("return type: want %q, got %q", "u64", fn.ReturnType.String())
	}
	if fn.Body == nil {
		t.Fatal("expected body, got nil")
	}
	// The body should have a tail expression (a + b).
	if fn.Body.Tail == nil {
		t.Fatal("expected tail expression in body, got nil")
	}
}

func TestParseFnDecl_Pub(t *testing.T) {
	src := `pub fn greet() { }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	if !fn.Public {
		t.Error("expected fn to be public")
	}
	if fn.Name != "greet" {
		t.Errorf("fn name: want %q, got %q", "greet", fn.Name)
	}
	if fn.ReturnType != nil {
		t.Error("expected nil return type for unit function")
	}
}

func TestParseFnDecl_EmptyBody(t *testing.T) {
	src := `fn noop() { }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	if fn.Body.Tail != nil {
		t.Error("expected nil tail for empty body")
	}
	if len(fn.Body.Statements) != 0 {
		t.Errorf("expected 0 statements, got %d", len(fn.Body.Statements))
	}
}

// ---------------------------------------------------------------------------
// Let statement
// ---------------------------------------------------------------------------

func TestParseLetStmt(t *testing.T) {
	src := `fn f() { let x: u64 = 42; }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)

	if len(fn.Body.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(fn.Body.Statements))
	}
	let, ok := fn.Body.Statements[0].(*ast.LetStmt)
	if !ok {
		t.Fatalf("expected *ast.LetStmt, got %T", fn.Body.Statements[0])
	}
	if let.Name.Value != "x" {
		t.Errorf("let name: want %q, got %q", "x", let.Name.Value)
	}
	if let.Mutable {
		t.Error("let should not be mutable")
	}
	if let.Type == nil || let.Type.String() != "u64" {
		t.Errorf("let type: want %q, got %v", "u64", let.Type)
	}
	lit, ok := let.Value.(*ast.IntLiteral)
	if !ok {
		t.Fatalf("expected *ast.IntLiteral, got %T", let.Value)
	}
	if lit.Value != 42 {
		t.Errorf("int value: want 42, got %d", lit.Value)
	}
}

func TestParseLetStmt_Mut(t *testing.T) {
	src := `fn f() { let mut count: u64 = 0; }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	let := fn.Body.Statements[0].(*ast.LetStmt)
	if !let.Mutable {
		t.Error("expected let to be mutable")
	}
	if let.Name.Value != "count" {
		t.Errorf("let name: want %q, got %q", "count", let.Name.Value)
	}
}

// ---------------------------------------------------------------------------
// If expression
// ---------------------------------------------------------------------------

func TestParseIfExpr(t *testing.T) {
	src := `fn abs(x: i64) -> i64 { if x > 0 { x } else { -x } }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)

	ifExpr, ok := fn.Body.Tail.(*ast.IfExpr)
	if !ok {
		t.Fatalf("expected *ast.IfExpr as block tail, got %T", fn.Body.Tail)
	}
	if ifExpr.Condition == nil {
		t.Fatal("if condition is nil")
	}
	if ifExpr.Consequence == nil {
		t.Fatal("if consequence is nil")
	}
	if ifExpr.Alternative == nil {
		t.Fatal("expected else branch, got nil")
	}
}

func TestParseIfExpr_NoElse(t *testing.T) {
	src := `fn f() { if true { }; }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	stmt := fn.Body.Statements[0].(*ast.ExprStmt)
	ifExpr := stmt.Expression.(*ast.IfExpr)
	if ifExpr.Alternative != nil {
		t.Error("expected nil alternative when no else")
	}
}

func TestParseIfExpr_ElseIf(t *testing.T) {
	src := `fn classify(x: i64) -> i64 { if x > 0 { 1 } else if x < 0 { -1 } else { 0 } }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	ifExpr, ok := fn.Body.Tail.(*ast.IfExpr)
	if !ok {
		t.Fatalf("expected *ast.IfExpr, got %T", fn.Body.Tail)
	}
	// Alternative should be another IfExpr.
	_, ok = ifExpr.Alternative.(*ast.IfExpr)
	if !ok {
		t.Fatalf("expected else-if to be *ast.IfExpr, got %T", ifExpr.Alternative)
	}
}

// ---------------------------------------------------------------------------
// Struct declaration
// ---------------------------------------------------------------------------

func TestParseStructDecl(t *testing.T) {
	src := `struct Point { x: u64, y: u64 }`
	prog := mustParse(t, src)
	s, ok := firstDecl(t, prog).(*ast.StructDecl)
	if !ok {
		t.Fatalf("expected *ast.StructDecl, got %T", firstDecl(t, prog))
	}
	if s.Name != "Point" {
		t.Errorf("struct name: want %q, got %q", "Point", s.Name)
	}
	if len(s.Fields) != 2 {
		t.Fatalf("want 2 fields, got %d", len(s.Fields))
	}
	if s.Fields[0].Name != "x" || s.Fields[1].Name != "y" {
		t.Errorf("fields: want x,y got %q,%q", s.Fields[0].Name, s.Fields[1].Name)
	}
	if s.Fields[0].Type.String() != "u64" {
		t.Errorf("field x type: want u64, got %q", s.Fields[0].Type.String())
	}
}

func TestParseStructDecl_PubFields(t *testing.T) {
	src := `pub struct Rect { pub width: u64, pub height: u64 }`
	prog := mustParse(t, src)
	s := firstDecl(t, prog).(*ast.StructDecl)
	if !s.Public {
		t.Error("struct should be public")
	}
	for _, f := range s.Fields {
		if !f.Public {
			t.Errorf("field %q should be public", f.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// Agent declaration
// ---------------------------------------------------------------------------

func TestParseAgentDecl(t *testing.T) {
	src := `agent Echo {
		state { count: u64 }
		msg handle(data: bytes) { }
	}`
	prog := mustParse(t, src)
	ag, ok := firstDecl(t, prog).(*ast.AgentDecl)
	if !ok {
		t.Fatalf("expected *ast.AgentDecl, got %T", firstDecl(t, prog))
	}
	if ag.Name != "Echo" {
		t.Errorf("agent name: want %q, got %q", "Echo", ag.Name)
	}
	if ag.State == nil {
		t.Fatal("expected state block, got nil")
	}
	if len(ag.State.Fields) != 1 {
		t.Fatalf("want 1 state field, got %d", len(ag.State.Fields))
	}
	if ag.State.Fields[0].Name != "count" {
		t.Errorf("state field: want %q, got %q", "count", ag.State.Fields[0].Name)
	}
	if len(ag.Handlers) != 1 {
		t.Fatalf("want 1 msg handler, got %d", len(ag.Handlers))
	}
	if ag.Handlers[0].Name != "handle" {
		t.Errorf("handler name: want %q, got %q", "handle", ag.Handlers[0].Name)
	}
	if len(ag.Handlers[0].Params) != 1 {
		t.Fatalf("want 1 handler param, got %d", len(ag.Handlers[0].Params))
	}
	if ag.Handlers[0].Params[0].Name != "data" {
		t.Errorf("handler param: want %q, got %q", "data", ag.Handlers[0].Params[0].Name)
	}
}

func TestParseAgentDecl_NoState(t *testing.T) {
	src := `agent Pinger { msg ping() { } }`
	prog := mustParse(t, src)
	ag := firstDecl(t, prog).(*ast.AgentDecl)
	if ag.State != nil {
		t.Error("expected nil state block")
	}
	if len(ag.Handlers) != 1 {
		t.Fatalf("want 1 handler, got %d", len(ag.Handlers))
	}
}

// ---------------------------------------------------------------------------
// Resource declaration
// ---------------------------------------------------------------------------

func TestParseResourceDecl(t *testing.T) {
	src := `resource Token { balance: u64 }`
	prog := mustParse(t, src)
	r, ok := firstDecl(t, prog).(*ast.ResourceDecl)
	if !ok {
		t.Fatalf("expected *ast.ResourceDecl, got %T", firstDecl(t, prog))
	}
	if r.Name != "Token" {
		t.Errorf("resource name: want %q, got %q", "Token", r.Name)
	}
	if len(r.Fields) != 1 {
		t.Fatalf("want 1 field, got %d", len(r.Fields))
	}
	if r.Fields[0].Name != "balance" {
		t.Errorf("field name: want %q, got %q", "balance", r.Fields[0].Name)
	}
}

// ---------------------------------------------------------------------------
// Match expression
// ---------------------------------------------------------------------------

func TestParseMatchExpr(t *testing.T) {
	src := `fn label(x: u64) -> String { match x { 1 => "one", _ => "other" } }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)

	m, ok := fn.Body.Tail.(*ast.MatchExpr)
	if !ok {
		t.Fatalf("expected *ast.MatchExpr as block tail, got %T", fn.Body.Tail)
	}
	ident, ok := m.Subject.(*ast.Ident)
	if !ok || ident.Value != "x" {
		t.Errorf("match subject: want ident %q, got %v", "x", m.Subject)
	}
	if len(m.Arms) != 2 {
		t.Fatalf("want 2 match arms, got %d", len(m.Arms))
	}
	// First arm: 1 => "one"
	lit, ok := m.Arms[0].Pattern.(*ast.IntLiteral)
	if !ok || lit.Value != 1 {
		t.Errorf("arm[0] pattern: want int literal 1, got %T %v", m.Arms[0].Pattern, m.Arms[0].Pattern)
	}
	// Second arm: _ => "other"
	wildcard, ok := m.Arms[1].Pattern.(*ast.Ident)
	if !ok || wildcard.Value != "_" {
		t.Errorf("arm[1] pattern: want wildcard '_', got %T %v", m.Arms[1].Pattern, m.Arms[1].Pattern)
	}
}

func TestParseMatchExpr_BoolPattern(t *testing.T) {
	src := `fn f(b: bool) { match b { true => 1, false => 0 }; }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	stmt := fn.Body.Statements[0].(*ast.ExprStmt)
	m := stmt.Expression.(*ast.MatchExpr)
	if len(m.Arms) != 2 {
		t.Fatalf("want 2 arms, got %d", len(m.Arms))
	}
	b0, ok := m.Arms[0].Pattern.(*ast.BoolLiteral)
	if !ok || !b0.Value {
		t.Errorf("arm[0]: want true literal")
	}
}

// ---------------------------------------------------------------------------
// Operator precedence
// ---------------------------------------------------------------------------

func TestPrecedence_AddMul(t *testing.T) {
	// a + b * c should parse as a + (b * c)
	src := `fn f() { a + b * c }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)

	add, ok := fn.Body.Tail.(*ast.InfixExpr)
	if !ok {
		t.Fatalf("expected *ast.InfixExpr, got %T", fn.Body.Tail)
	}
	if add.Operator != "+" {
		t.Errorf("top operator: want '+', got %q", add.Operator)
	}
	// Right side should be the multiplication.
	mul, ok := add.Right.(*ast.InfixExpr)
	if !ok {
		t.Fatalf("expected *ast.InfixExpr on right of '+', got %T", add.Right)
	}
	if mul.Operator != "*" {
		t.Errorf("right operator: want '*', got %q", mul.Operator)
	}
}

func TestPrecedence_Comparison(t *testing.T) {
	// a + b == c - d  should be (a+b) == (c-d)
	src := `fn f() { a + b == c - d }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	eq, ok := fn.Body.Tail.(*ast.InfixExpr)
	if !ok || eq.Operator != "==" {
		t.Fatalf("expected == at top level, got %T %q", fn.Body.Tail, eq.Operator)
	}
	_, ok = eq.Left.(*ast.InfixExpr)
	if !ok {
		t.Error("expected infix expression on left of ==")
	}
}

func TestPrecedence_Logical(t *testing.T) {
	// a || b && c  should parse as a || (b && c)
	src := `fn f() { a || b && c }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	or, ok := fn.Body.Tail.(*ast.InfixExpr)
	if !ok || or.Operator != "||" {
		t.Fatalf("expected || at top, got %T %q", fn.Body.Tail, or.Operator)
	}
	and, ok := or.Right.(*ast.InfixExpr)
	if !ok || and.Operator != "&&" {
		t.Fatalf("expected && on right of ||, got %T", or.Right)
	}
	_ = and
}

func TestPrecedence_Prefix(t *testing.T) {
	// -a * b should parse as (-a) * b
	src := `fn f() { -a * b }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	mul, ok := fn.Body.Tail.(*ast.InfixExpr)
	if !ok || mul.Operator != "*" {
		t.Fatalf("expected * at top, got %T", fn.Body.Tail)
	}
	prefix, ok := mul.Left.(*ast.PrefixExpr)
	if !ok || prefix.Operator != "-" {
		t.Fatalf("expected prefix - on left of *, got %T", mul.Left)
	}
}

func TestPrecedence_Grouped(t *testing.T) {
	// (a + b) * c — grouping overrides precedence.
	src := `fn f() { (a + b) * c }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	mul, ok := fn.Body.Tail.(*ast.InfixExpr)
	if !ok || mul.Operator != "*" {
		t.Fatalf("expected * at top, got %T", fn.Body.Tail)
	}
	add, ok := mul.Left.(*ast.InfixExpr)
	if !ok || add.Operator != "+" {
		t.Fatalf("expected + on left of *, got %T", mul.Left)
	}
	_ = add
}

// ---------------------------------------------------------------------------
// Method call
// ---------------------------------------------------------------------------

func TestParseMethodCall(t *testing.T) {
	src := `fn f() { x.foo(y) }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)

	mc, ok := fn.Body.Tail.(*ast.MethodCallExpr)
	if !ok {
		t.Fatalf("expected *ast.MethodCallExpr, got %T", fn.Body.Tail)
	}
	recv, ok := mc.Receiver.(*ast.Ident)
	if !ok || recv.Value != "x" {
		t.Errorf("receiver: want %q, got %v", "x", mc.Receiver)
	}
	if mc.Method != "foo" {
		t.Errorf("method: want %q, got %q", "foo", mc.Method)
	}
	if len(mc.Arguments) != 1 {
		t.Fatalf("want 1 arg, got %d", len(mc.Arguments))
	}
}

func TestParseMethodCall_Chained(t *testing.T) {
	src := `fn f() { a.b().c() }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	_, ok := fn.Body.Tail.(*ast.MethodCallExpr)
	if !ok {
		t.Fatalf("expected *ast.MethodCallExpr, got %T", fn.Body.Tail)
	}
}

// ---------------------------------------------------------------------------
// Field access
// ---------------------------------------------------------------------------

func TestParseFieldAccess(t *testing.T) {
	src := `fn f() { p.x }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	fe, ok := fn.Body.Tail.(*ast.FieldExpr)
	if !ok {
		t.Fatalf("expected *ast.FieldExpr, got %T", fn.Body.Tail)
	}
	if fe.Field != "x" {
		t.Errorf("field: want %q, got %q", "x", fe.Field)
	}
}

// ---------------------------------------------------------------------------
// Array expression
// ---------------------------------------------------------------------------

func TestParseArrayExpr(t *testing.T) {
	src := `fn f() { [1, 2, 3] }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	arr, ok := fn.Body.Tail.(*ast.ArrayExpr)
	if !ok {
		t.Fatalf("expected *ast.ArrayExpr, got %T", fn.Body.Tail)
	}
	if len(arr.Elements) != 3 {
		t.Fatalf("want 3 elements, got %d", len(arr.Elements))
	}
	for i, e := range arr.Elements {
		lit, ok := e.(*ast.IntLiteral)
		if !ok {
			t.Errorf("element %d: expected *ast.IntLiteral, got %T", i, e)
			continue
		}
		if lit.Value != int64(i+1) {
			t.Errorf("element %d value: want %d, got %d", i, i+1, lit.Value)
		}
	}
}

func TestParseArrayExpr_Empty(t *testing.T) {
	src := `fn f() { [] }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	arr, ok := fn.Body.Tail.(*ast.ArrayExpr)
	if !ok {
		t.Fatalf("expected *ast.ArrayExpr, got %T", fn.Body.Tail)
	}
	if len(arr.Elements) != 0 {
		t.Errorf("want 0 elements, got %d", len(arr.Elements))
	}
}

// ---------------------------------------------------------------------------
// Enum declaration
// ---------------------------------------------------------------------------

func TestParseEnumDecl(t *testing.T) {
	src := `enum Color { Red, Green, Blue }`
	prog := mustParse(t, src)
	e, ok := firstDecl(t, prog).(*ast.EnumDecl)
	if !ok {
		t.Fatalf("expected *ast.EnumDecl, got %T", firstDecl(t, prog))
	}
	if e.Name != "Color" {
		t.Errorf("enum name: want %q, got %q", "Color", e.Name)
	}
	if len(e.Variants) != 3 {
		t.Fatalf("want 3 variants, got %d", len(e.Variants))
	}
	names := []string{"Red", "Green", "Blue"}
	for i, v := range e.Variants {
		if v.Name != names[i] {
			t.Errorf("variant %d: want %q, got %q", i, names[i], v.Name)
		}
	}
}

func TestParseEnumDecl_WithData(t *testing.T) {
	src := `enum Msg { Quit, Value(u64), Point(u64, u64) }`
	prog := mustParse(t, src)
	e := firstDecl(t, prog).(*ast.EnumDecl)
	if len(e.Variants) != 3 {
		t.Fatalf("want 3 variants, got %d", len(e.Variants))
	}
	if len(e.Variants[0].Fields) != 0 {
		t.Error("Quit should have no fields")
	}
	if len(e.Variants[1].Fields) != 1 {
		t.Errorf("Value should have 1 field, got %d", len(e.Variants[1].Fields))
	}
	if len(e.Variants[2].Fields) != 2 {
		t.Errorf("Point should have 2 fields, got %d", len(e.Variants[2].Fields))
	}
}

// ---------------------------------------------------------------------------
// Trait declaration
// ---------------------------------------------------------------------------

func TestParseTraitDecl(t *testing.T) {
	src := `trait Greet { fn hello(name: String) -> String; }`
	prog := mustParse(t, src)
	tr, ok := firstDecl(t, prog).(*ast.TraitDecl)
	if !ok {
		t.Fatalf("expected *ast.TraitDecl, got %T", firstDecl(t, prog))
	}
	if tr.Name != "Greet" {
		t.Errorf("trait name: want %q, got %q", "Greet", tr.Name)
	}
	if len(tr.Methods) != 1 {
		t.Fatalf("want 1 method, got %d", len(tr.Methods))
	}
	if tr.Methods[0].Name != "hello" {
		t.Errorf("method name: want %q, got %q", "hello", tr.Methods[0].Name)
	}
}

// ---------------------------------------------------------------------------
// Impl declaration
// ---------------------------------------------------------------------------

func TestParseImplDecl(t *testing.T) {
	src := `impl Point { fn new(x: u64, y: u64) -> Point { Point } }`
	prog := mustParse(t, src)
	im, ok := firstDecl(t, prog).(*ast.ImplDecl)
	if !ok {
		t.Fatalf("expected *ast.ImplDecl, got %T", firstDecl(t, prog))
	}
	if im.TypeName != "Point" {
		t.Errorf("impl type: want %q, got %q", "Point", im.TypeName)
	}
	if im.Trait != "" {
		t.Errorf("impl should not have trait, got %q", im.Trait)
	}
	if len(im.Methods) != 1 {
		t.Fatalf("want 1 method, got %d", len(im.Methods))
	}
}

func TestParseImplDecl_TraitFor(t *testing.T) {
	src := `impl Greet for Point { fn hello(name: String) -> String { name } }`
	prog := mustParse(t, src)
	im := firstDecl(t, prog).(*ast.ImplDecl)
	if im.Trait != "Greet" {
		t.Errorf("impl trait: want %q, got %q", "Greet", im.Trait)
	}
	if im.TypeName != "Point" {
		t.Errorf("impl type: want %q, got %q", "Point", im.TypeName)
	}
}

// ---------------------------------------------------------------------------
// Use declaration
// ---------------------------------------------------------------------------

func TestParseUseDecl(t *testing.T) {
	src := `use std::collections::HashMap;`
	prog := mustParse(t, src)
	u, ok := firstDecl(t, prog).(*ast.UseDecl)
	if !ok {
		t.Fatalf("expected *ast.UseDecl, got %T", firstDecl(t, prog))
	}
	want := []string{"std", "collections", "HashMap"}
	if len(u.Path) != len(want) {
		t.Fatalf("path len: want %d, got %d", len(want), len(u.Path))
	}
	for i, seg := range want {
		if u.Path[i] != seg {
			t.Errorf("path[%d]: want %q, got %q", i, seg, u.Path[i])
		}
	}
}

func TestParseUseDecl_WithAlias(t *testing.T) {
	src := `use std::io as io;`
	prog := mustParse(t, src)
	u := firstDecl(t, prog).(*ast.UseDecl)
	if u.Alias != "io" {
		t.Errorf("alias: want %q, got %q", "io", u.Alias)
	}
}

// ---------------------------------------------------------------------------
// Type alias declaration
// ---------------------------------------------------------------------------

func TestParseTypeDecl(t *testing.T) {
	src := `type Bytes32 = [u8; 32];`
	prog := mustParse(t, src)
	td, ok := firstDecl(t, prog).(*ast.TypeDecl)
	if !ok {
		t.Fatalf("expected *ast.TypeDecl, got %T", firstDecl(t, prog))
	}
	if td.Name != "Bytes32" {
		t.Errorf("type name: want %q, got %q", "Bytes32", td.Name)
	}
	arr, ok := td.Type.(*ast.ArrayType)
	if !ok {
		t.Fatalf("type should be array, got %T", td.Type)
	}
	if arr.Elem.String() != "u8" {
		t.Errorf("array elem: want %q, got %q", "u8", arr.Elem.String())
	}
}

// ---------------------------------------------------------------------------
// Mod declaration
// ---------------------------------------------------------------------------

func TestParseModDecl_External(t *testing.T) {
	src := `mod utils;`
	prog := mustParse(t, src)
	m, ok := firstDecl(t, prog).(*ast.ModDecl)
	if !ok {
		t.Fatalf("expected *ast.ModDecl, got %T", firstDecl(t, prog))
	}
	if m.Name != "utils" {
		t.Errorf("mod name: want %q, got %q", "utils", m.Name)
	}
	if m.Declarations != nil {
		t.Error("external mod should have nil Declarations")
	}
}

func TestParseModDecl_Inline(t *testing.T) {
	src := `mod helpers { fn noop() { } }`
	prog := mustParse(t, src)
	m := firstDecl(t, prog).(*ast.ModDecl)
	if m.Declarations == nil {
		t.Fatal("inline mod should have non-nil Declarations")
	}
	if len(m.Declarations) != 1 {
		t.Fatalf("want 1 inner decl, got %d", len(m.Declarations))
	}
}

// ---------------------------------------------------------------------------
// Type expressions
// ---------------------------------------------------------------------------

func TestParseType_PathType(t *testing.T) {
	src := `fn f() -> std::String { }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	pt, ok := fn.ReturnType.(*ast.PathType)
	if !ok {
		t.Fatalf("expected *ast.PathType, got %T", fn.ReturnType)
	}
	if len(pt.Segments) != 2 {
		t.Fatalf("want 2 segments, got %d", len(pt.Segments))
	}
	if pt.Segments[0] != "std" || pt.Segments[1] != "String" {
		t.Errorf("segments: want [std String], got %v", pt.Segments)
	}
}

func TestParseType_SliceType(t *testing.T) {
	src := `fn f() -> [u8] { }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	st, ok := fn.ReturnType.(*ast.SliceType)
	if !ok {
		t.Fatalf("expected *ast.SliceType, got %T", fn.ReturnType)
	}
	if st.Elem.String() != "u8" {
		t.Errorf("elem: want u8, got %q", st.Elem.String())
	}
}

func TestParseType_RefType(t *testing.T) {
	src := `fn f(x: &u64) { }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	rt, ok := fn.Params[0].Type.(*ast.RefType)
	if !ok {
		t.Fatalf("expected *ast.RefType, got %T", fn.Params[0].Type)
	}
	if rt.Elem.String() != "u64" {
		t.Errorf("elem: want u64, got %q", rt.Elem.String())
	}
}

func TestParseType_MutRefType(t *testing.T) {
	src := `fn f(x: &mut u64) { }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	_, ok := fn.Params[0].Type.(*ast.MutRefType)
	if !ok {
		t.Fatalf("expected *ast.MutRefType, got %T", fn.Params[0].Type)
	}
}

func TestParseType_FnType(t *testing.T) {
	src := `fn apply(f: fn(u64, u64) -> u64, a: u64, b: u64) -> u64 { f(a, b) }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	ft, ok := fn.Params[0].Type.(*ast.FnType)
	if !ok {
		t.Fatalf("expected *ast.FnType, got %T", fn.Params[0].Type)
	}
	if len(ft.ParamTypes) != 2 {
		t.Fatalf("want 2 param types, got %d", len(ft.ParamTypes))
	}
	if ft.ReturnType == nil || ft.ReturnType.String() != "u64" {
		t.Errorf("fn type return: want u64, got %v", ft.ReturnType)
	}
}

// ---------------------------------------------------------------------------
// Additional expression forms
// ---------------------------------------------------------------------------

func TestParseLiterals(t *testing.T) {
	tests := []struct {
		src  string
		desc string
	}{
		{`fn f() { 42 }`, "int literal"},
		{`fn f() { 3.14 }`, "float literal"},
		{`fn f() { "hello" }`, "string literal"},
		{`fn f() { true }`, "bool true"},
		{`fn f() { false }`, "bool false"},
		{`fn f() { nil }`, "nil literal"},
		{`fn f() { 0xdeadbeef }`, "bytes literal"},
		{`fn f() { @0x1234abcd }`, "address literal"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			mustParse(t, tt.src)
		})
	}
}

func TestParsePrefixExpressions(t *testing.T) {
	ops := []string{"-", "!", "~", "#", "&", "*"}
	for _, op := range ops {
		t.Run(op, func(t *testing.T) {
			src := `fn f() { ` + op + `x }`
			prog := mustParse(t, src)
			fn := firstDecl(t, prog).(*ast.FnDecl)
			pe, ok := fn.Body.Tail.(*ast.PrefixExpr)
			if !ok {
				t.Fatalf("expected *ast.PrefixExpr, got %T", fn.Body.Tail)
			}
			if pe.Operator != op {
				t.Errorf("operator: want %q, got %q", op, pe.Operator)
			}
		})
	}
}

func TestParseIndexExpr(t *testing.T) {
	src := `fn f() { arr[0] }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	ie, ok := fn.Body.Tail.(*ast.IndexExpr)
	if !ok {
		t.Fatalf("expected *ast.IndexExpr, got %T", fn.Body.Tail)
	}
	idx, ok := ie.Index.(*ast.IntLiteral)
	if !ok || idx.Value != 0 {
		t.Errorf("index: want 0, got %v", ie.Index)
	}
}

func TestParseCallExpr(t *testing.T) {
	src := `fn f() { foo(1, 2) }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	ce, ok := fn.Body.Tail.(*ast.CallExpr)
	if !ok {
		t.Fatalf("expected *ast.CallExpr, got %T", fn.Body.Tail)
	}
	ident, ok := ce.Function.(*ast.Ident)
	if !ok || ident.Value != "foo" {
		t.Errorf("callee: want foo, got %v", ce.Function)
	}
	if len(ce.Arguments) != 2 {
		t.Fatalf("want 2 args, got %d", len(ce.Arguments))
	}
}

func TestParseRangeExpr(t *testing.T) {
	src := `fn f() { 0..10 }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	re, ok := fn.Body.Tail.(*ast.RangeExpr)
	if !ok {
		t.Fatalf("expected *ast.RangeExpr, got %T", fn.Body.Tail)
	}
	start, ok := re.Start.(*ast.IntLiteral)
	if !ok || start.Value != 0 {
		t.Errorf("range start: want 0, got %v", re.Start)
	}
	end, ok := re.End.(*ast.IntLiteral)
	if !ok || end.Value != 10 {
		t.Errorf("range end: want 10, got %v", re.End)
	}
}

func TestParseMoveExpr(t *testing.T) {
	src := `fn f() { move x }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	me, ok := fn.Body.Tail.(*ast.MoveExpr)
	if !ok {
		t.Fatalf("expected *ast.MoveExpr, got %T", fn.Body.Tail)
	}
	ident, ok := me.Value.(*ast.Ident)
	if !ok || ident.Value != "x" {
		t.Errorf("move value: want x, got %v", me.Value)
	}
}

func TestParseCopyExpr(t *testing.T) {
	src := `fn f() { copy x }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	ce, ok := fn.Body.Tail.(*ast.CopyExpr)
	if !ok {
		t.Fatalf("expected *ast.CopyExpr, got %T", fn.Body.Tail)
	}
	ident, ok := ce.Value.(*ast.Ident)
	if !ok || ident.Value != "x" {
		t.Errorf("copy value: want x, got %v", ce.Value)
	}
}

func TestParseSpawnExpr(t *testing.T) {
	src := `fn f() { spawn Counter { count: 0 } }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	se, ok := fn.Body.Tail.(*ast.SpawnExpr)
	if !ok {
		t.Fatalf("expected *ast.SpawnExpr, got %T", fn.Body.Tail)
	}
	if se.Agent != "Counter" {
		t.Errorf("spawn agent: want %q, got %q", "Counter", se.Agent)
	}
	if _, ok := se.Fields["count"]; !ok {
		t.Error("expected 'count' field in spawn")
	}
}

func TestParseSendExpr(t *testing.T) {
	src := `fn f() { send target msg_val }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	se, ok := fn.Body.Tail.(*ast.SendExpr)
	if !ok {
		t.Fatalf("expected *ast.SendExpr, got %T", fn.Body.Tail)
	}
	tgt, ok := se.Target.(*ast.Ident)
	if !ok || tgt.Value != "target" {
		t.Errorf("send target: want %q, got %v", "target", se.Target)
	}
}

func TestParseRecvExpr(t *testing.T) {
	src := `fn f() { recv }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	_, ok := fn.Body.Tail.(*ast.RecvExpr)
	if !ok {
		t.Fatalf("expected *ast.RecvExpr, got %T", fn.Body.Tail)
	}
}

// ---------------------------------------------------------------------------
// Statements
// ---------------------------------------------------------------------------

func TestParseReturnStmt(t *testing.T) {
	src := `fn f() -> u64 { return 42; }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	ret, ok := fn.Body.Statements[0].(*ast.ReturnStmt)
	if !ok {
		t.Fatalf("expected *ast.ReturnStmt, got %T", fn.Body.Statements[0])
	}
	lit, ok := ret.Value.(*ast.IntLiteral)
	if !ok || lit.Value != 42 {
		t.Errorf("return value: want 42, got %v", ret.Value)
	}
}

func TestParseReturnStmt_Bare(t *testing.T) {
	src := `fn f() { return; }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	ret := fn.Body.Statements[0].(*ast.ReturnStmt)
	if ret.Value != nil {
		t.Errorf("bare return should have nil value, got %v", ret.Value)
	}
}

func TestParseForStmt(t *testing.T) {
	src := `fn f() { for i in items { } }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	fs, ok := fn.Body.Statements[0].(*ast.ForStmt)
	if !ok {
		t.Fatalf("expected *ast.ForStmt, got %T", fn.Body.Statements[0])
	}
	if fs.Binding.Value != "i" {
		t.Errorf("binding: want %q, got %q", "i", fs.Binding.Value)
	}
}

func TestParseWhileStmt(t *testing.T) {
	src := `fn f() { while true { } }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	ws, ok := fn.Body.Statements[0].(*ast.WhileStmt)
	if !ok {
		t.Fatalf("expected *ast.WhileStmt, got %T", fn.Body.Statements[0])
	}
	b, ok := ws.Condition.(*ast.BoolLiteral)
	if !ok || !b.Value {
		t.Error("while condition: want true")
	}
}

func TestParseBreakContinue(t *testing.T) {
	src := `fn f() { break; continue; }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	if len(fn.Body.Statements) != 2 {
		t.Fatalf("want 2 stmts, got %d", len(fn.Body.Statements))
	}
	if _, ok := fn.Body.Statements[0].(*ast.BreakStmt); !ok {
		t.Errorf("stmt[0]: want *ast.BreakStmt, got %T", fn.Body.Statements[0])
	}
	if _, ok := fn.Body.Statements[1].(*ast.ContinueStmt); !ok {
		t.Errorf("stmt[1]: want *ast.ContinueStmt, got %T", fn.Body.Statements[1])
	}
}

func TestParseDropStmt(t *testing.T) {
	src := `fn f() { drop tok; }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	ds, ok := fn.Body.Statements[0].(*ast.DropStmt)
	if !ok {
		t.Fatalf("expected *ast.DropStmt, got %T", fn.Body.Statements[0])
	}
	if ds.Value.Value != "tok" {
		t.Errorf("drop value: want %q, got %q", "tok", ds.Value.Value)
	}
}

func TestParseEmitStmt(t *testing.T) {
	src := `fn f() { emit Transfer { from: a, to: b }; }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	es, ok := fn.Body.Statements[0].(*ast.EmitStmt)
	if !ok {
		t.Fatalf("expected *ast.EmitStmt, got %T", fn.Body.Statements[0])
	}
	if es.Event != "Transfer" {
		t.Errorf("event: want %q, got %q", "Transfer", es.Event)
	}
	if len(es.Fields) != 2 {
		t.Errorf("want 2 fields, got %d", len(es.Fields))
	}
}

func TestParseRequireStmt(t *testing.T) {
	src := `fn f() { require(x > 0, "must be positive"); }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	rs, ok := fn.Body.Statements[0].(*ast.RequireStmt)
	if !ok {
		t.Fatalf("expected *ast.RequireStmt, got %T", fn.Body.Statements[0])
	}
	if rs.Condition == nil {
		t.Error("require condition is nil")
	}
	if rs.Message == nil {
		t.Error("require message is nil")
	}
}

func TestParseAssignStmt(t *testing.T) {
	src := `fn f() { x = 1; y += 2; }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	if len(fn.Body.Statements) != 2 {
		t.Fatalf("want 2 stmts, got %d", len(fn.Body.Statements))
	}
	a0, ok := fn.Body.Statements[0].(*ast.AssignStmt)
	if !ok {
		t.Fatalf("stmt[0]: expected *ast.AssignStmt, got %T", fn.Body.Statements[0])
	}
	if a0.Operator != "=" {
		t.Errorf("op[0]: want '=', got %q", a0.Operator)
	}
	a1, ok := fn.Body.Statements[1].(*ast.AssignStmt)
	if !ok {
		t.Fatalf("stmt[1]: expected *ast.AssignStmt, got %T", fn.Body.Statements[1])
	}
	if a1.Operator != "+=" {
		t.Errorf("op[1]: want '+=', got %q", a1.Operator)
	}
}

// ---------------------------------------------------------------------------
// Error recovery
// ---------------------------------------------------------------------------

func TestErrorRecovery_InvalidTopLevel(t *testing.T) {
	// The "42" is invalid at top level; the parser should skip it and still
	// find the subsequent fn declaration.
	src := `42 fn valid() { }`
	prog, errs := parseWithErrors(t, src)
	if len(errs) == 0 {
		t.Fatal("expected errors for invalid top-level token")
	}
	// The valid function should still be in the AST.
	found := false
	for _, d := range prog.Declarations {
		if fn, ok := d.(*ast.FnDecl); ok && fn.Name == "valid" {
			found = true
			break
		}
	}
	if !found {
		t.Error("error recovery failed: 'valid' fn not found after bad token")
	}
}

func TestErrorRecovery_MissingParamType(t *testing.T) {
	// Missing type after colon — parser should handle gracefully.
	src := `fn bad(x:) { }`
	prog, errs := Parse("test.probe", src)
	// We expect errors but should still get something back.
	_ = errs // errors expected
	_ = prog  // partial AST returned
}

func TestErrorRecovery_UnclosedBrace(t *testing.T) {
	src := `fn incomplete() {`
	prog, errs := Parse("test.probe", src)
	_ = prog
	if len(errs) == 0 {
		t.Error("expected errors for unclosed brace")
	}
}

func TestErrorRecovery_MultipleDecls(t *testing.T) {
	// Valid decl, then bad token, then another valid decl.
	src := `
fn first() { }
@ invalid_here @
fn second() { }
`
	prog, errs := Parse("test.probe", src)
	if len(errs) == 0 {
		t.Fatal("expected errors")
	}
	count := 0
	for _, d := range prog.Declarations {
		if _, ok := d.(*ast.FnDecl); ok {
			count++
		}
	}
	if count < 1 {
		t.Errorf("expected at least 1 fn decl after recovery, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// String representation (round-trip smoke tests)
// ---------------------------------------------------------------------------

func TestString_FnDecl(t *testing.T) {
	src := `fn add(a: u64, b: u64) -> u64 { a + b }`
	prog := mustParse(t, src)
	s := prog.String()
	if !strings.Contains(s, "fn add") {
		t.Errorf("String() missing fn add: %q", s)
	}
}

func TestString_StructDecl(t *testing.T) {
	src := `struct Point { x: u64, y: u64 }`
	prog := mustParse(t, src)
	s := prog.String()
	if !strings.Contains(s, "struct Point") {
		t.Errorf("String() missing struct Point: %q", s)
	}
}

// ---------------------------------------------------------------------------
// Multiple declarations in one program
// ---------------------------------------------------------------------------

func TestMultipleDeclarations(t *testing.T) {
	src := `
struct Wallet { balance: u64 }
resource Token { amount: u64 }
fn transfer(from: Wallet, to: Wallet, amount: u64) { }
`
	prog := mustParse(t, src)
	if len(prog.Declarations) != 3 {
		t.Fatalf("want 3 declarations, got %d", len(prog.Declarations))
	}
	if _, ok := prog.Declarations[0].(*ast.StructDecl); !ok {
		t.Errorf("decl[0]: expected StructDecl, got %T", prog.Declarations[0])
	}
	if _, ok := prog.Declarations[1].(*ast.ResourceDecl); !ok {
		t.Errorf("decl[1]: expected ResourceDecl, got %T", prog.Declarations[1])
	}
	if _, ok := prog.Declarations[2].(*ast.FnDecl); !ok {
		t.Errorf("decl[2]: expected FnDecl, got %T", prog.Declarations[2])
	}
}

// ---------------------------------------------------------------------------
// Path expressions (::)
// ---------------------------------------------------------------------------

func TestParsePathExpr(t *testing.T) {
	src := `fn f() { std::mem::size }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	ident, ok := fn.Body.Tail.(*ast.Ident)
	if !ok {
		t.Fatalf("expected *ast.Ident for path expr, got %T", fn.Body.Tail)
	}
	if !strings.Contains(ident.Value, "::") {
		t.Errorf("path ident value should contain '::', got %q", ident.Value)
	}
}

// ---------------------------------------------------------------------------
// Block as expression
// ---------------------------------------------------------------------------

func TestParseBlockExpr_Standalone(t *testing.T) {
	src := `fn f() -> u64 { { 42 } }`
	prog := mustParse(t, src)
	fn := firstDecl(t, prog).(*ast.FnDecl)
	// Outer block tail should be an inner BlockExpr.
	inner, ok := fn.Body.Tail.(*ast.BlockExpr)
	if !ok {
		t.Fatalf("expected *ast.BlockExpr as tail, got %T", fn.Body.Tail)
	}
	lit, ok := inner.Tail.(*ast.IntLiteral)
	if !ok || lit.Value != 42 {
		t.Errorf("inner block tail: want 42, got %v", inner.Tail)
	}
}

// ---------------------------------------------------------------------------
// Self parameter
// ---------------------------------------------------------------------------

func TestParseSelf(t *testing.T) {
	src := `impl Counter { fn get(self) -> u64 { self } }`
	prog := mustParse(t, src)
	im := firstDecl(t, prog).(*ast.ImplDecl)
	if len(im.Methods) == 0 {
		t.Fatal("expected method")
	}
	m := im.Methods[0]
	if len(m.Params) != 1 || m.Params[0].Name != "self" {
		t.Errorf("expected self param, got %v", m.Params)
	}
}

// ---------------------------------------------------------------------------
// Smoke test: realistic contract-like program
// ---------------------------------------------------------------------------

func TestContract_Smoke(t *testing.T) {
	src := `
pub resource Token {
    balance: u64
}

pub agent Vault {
    state { total: u64 }

    msg deposit(amount: u64) {
        require(amount > 0, "amount must be positive");
        self.total = self.total + amount;
    }

    msg withdraw(amount: u64) {
        require(self.total >= amount, "insufficient funds");
        self.total = self.total - amount;
    }
}

pub fn new_vault() -> Vault {
    spawn Vault { total: 0 }
}
`
	prog := mustParse(t, src)
	if len(prog.Declarations) != 3 {
		t.Fatalf("want 3 declarations, got %d", len(prog.Declarations))
	}
	if _, ok := prog.Declarations[0].(*ast.ResourceDecl); !ok {
		t.Errorf("decl[0]: want ResourceDecl, got %T", prog.Declarations[0])
	}
	if _, ok := prog.Declarations[1].(*ast.AgentDecl); !ok {
		t.Errorf("decl[1]: want AgentDecl, got %T", prog.Declarations[1])
	}
	if _, ok := prog.Declarations[2].(*ast.FnDecl); !ok {
		t.Errorf("decl[2]: want FnDecl, got %T", prog.Declarations[2])
	}
}

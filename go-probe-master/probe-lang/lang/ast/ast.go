// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.
//
// The ProbeChain is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Package ast defines the Abstract Syntax Tree for the PROBE language.
//
// Design overview:
//
//   - All AST nodes implement the Node interface via TokenLiteral and String.
//   - Expressions, Statements, and Declarations each have a marker interface
//     that embeds Node to enable type-safe dispatch.
//   - The tree is position-annotated via token.Token so error messages can
//     reference source locations.
//   - Agent-first primitives (SpawnExpr, SendExpr, RecvExpr, AgentDecl) and
//     linear-type primitives (MoveExpr, CopyExpr, DropStmt, ResourceDecl) are
//     first-class AST nodes, not syntactic sugar.
package ast

import (
	"bytes"
	"strings"

	"github.com/probechain/go-probe/probe-lang/lang/token"
)

// ---------------------------------------------------------------------------
// Core interfaces
// ---------------------------------------------------------------------------

// Node is the base interface that every AST node must implement.
type Node interface {
	// TokenLiteral returns the literal value of the token that originated this
	// node. Used primarily for debugging and testing.
	TokenLiteral() string

	// String returns a human-readable, parenthesised representation of the node
	// suitable for unit tests and debug output.
	String() string
}

// Expression is a marker interface for all expression nodes.
// Every Expression is also a Node.
type Expression interface {
	Node
	expressionNode()
}

// Statement is a marker interface for all statement nodes.
// Every Statement is also a Node.
type Statement interface {
	Node
	statementNode()
}

// Declaration is a marker interface for all top-level declaration nodes.
// Every Declaration is also a Node.
type Declaration interface {
	Node
	declarationNode()
}

// ---------------------------------------------------------------------------
// Program — root of every parse tree
// ---------------------------------------------------------------------------

// Program is the top-level AST node. It holds all declarations found in a
// source file or module.
type Program struct {
	Declarations []Declaration
}

func (p *Program) TokenLiteral() string {
	if len(p.Declarations) > 0 {
		return p.Declarations[0].TokenLiteral()
	}
	return ""
}

func (p *Program) String() string {
	var out bytes.Buffer
	for _, d := range p.Declarations {
		out.WriteString(d.String())
		out.WriteByte('\n')
	}
	return out.String()
}

// ---------------------------------------------------------------------------
// Type nodes
// ---------------------------------------------------------------------------

// TypeExpr is an expression that represents a PROBE type annotation.
// Type nodes are separate from value expressions so that the parser and type
// checker can handle them distinctly.
type TypeExpr interface {
	Node
	typeNode()
}

// NamedType is a simple, built-in or user-defined named type: u64, bool, String.
type NamedType struct {
	Token token.Token // the IDENT token
	Name  string
}

func (t *NamedType) typeNode()            {}
func (t *NamedType) TokenLiteral() string { return t.Token.Literal }
func (t *NamedType) String() string       { return t.Name }

// PathType is a module-qualified type: module::Type or a::b::C.
type PathType struct {
	Token    token.Token // first IDENT token in the path
	Segments []string    // ["module", "Type"] for module::Type
}

func (t *PathType) typeNode()            {}
func (t *PathType) TokenLiteral() string { return t.Token.Literal }
func (t *PathType) String() string       { return strings.Join(t.Segments, "::") }

// ArrayType is a fixed-length array type: [T; N].
type ArrayType struct {
	Token token.Token // '['
	Elem  TypeExpr
	Size  Expression // must evaluate to a compile-time integer constant
}

func (t *ArrayType) typeNode()            {}
func (t *ArrayType) TokenLiteral() string { return t.Token.Literal }
func (t *ArrayType) String() string {
	return "[" + t.Elem.String() + "; " + t.Size.String() + "]"
}

// SliceType is a dynamically-sized slice type: [T].
type SliceType struct {
	Token token.Token // '['
	Elem  TypeExpr
}

func (t *SliceType) typeNode()            {}
func (t *SliceType) TokenLiteral() string { return t.Token.Literal }
func (t *SliceType) String() string       { return "[" + t.Elem.String() + "]" }

// RefType is an immutable reference type: &T.
type RefType struct {
	Token token.Token // '&'
	Elem  TypeExpr
}

func (t *RefType) typeNode()            {}
func (t *RefType) TokenLiteral() string { return t.Token.Literal }
func (t *RefType) String() string       { return "&" + t.Elem.String() }

// MutRefType is a mutable reference type: &mut T.
type MutRefType struct {
	Token token.Token // '&'
	Elem  TypeExpr
}

func (t *MutRefType) typeNode()            {}
func (t *MutRefType) TokenLiteral() string { return t.Token.Literal }
func (t *MutRefType) String() string       { return "&mut " + t.Elem.String() }

// FnType is a function type: fn(T1, T2) -> R.
type FnType struct {
	Token      token.Token // 'fn'
	ParamTypes []TypeExpr
	ReturnType TypeExpr // nil means unit ()
}

func (t *FnType) typeNode()            {}
func (t *FnType) TokenLiteral() string { return t.Token.Literal }
func (t *FnType) String() string {
	var out bytes.Buffer
	out.WriteString("fn(")
	parts := make([]string, len(t.ParamTypes))
	for i, p := range t.ParamTypes {
		parts[i] = p.String()
	}
	out.WriteString(strings.Join(parts, ", "))
	out.WriteString(")")
	if t.ReturnType != nil {
		out.WriteString(" -> ")
		out.WriteString(t.ReturnType.String())
	}
	return out.String()
}

// ---------------------------------------------------------------------------
// Helper types shared by multiple nodes
// ---------------------------------------------------------------------------

// Param represents a single parameter in a function or method signature.
type Param struct {
	Token   token.Token // the IDENT token of the name
	Name    string
	Mutable bool
	Type    TypeExpr
}

func (p *Param) String() string {
	prefix := ""
	if p.Mutable {
		prefix = "mut "
	}
	if p.Type != nil {
		return prefix + p.Name + ": " + p.Type.String()
	}
	return prefix + p.Name
}

// Field represents a named field in a struct or resource declaration.
type Field struct {
	Token  token.Token // IDENT token
	Name   string
	Public bool
	Type   TypeExpr
}

func (f *Field) String() string {
	pub := ""
	if f.Public {
		pub = "pub "
	}
	return pub + f.Name + ": " + f.Type.String()
}

// MatchArm is a single arm inside a MatchExpr: pattern => body.
type MatchArm struct {
	Token   token.Token // token of the pattern start
	Pattern Expression  // pattern expression (Ident, literal, etc.)
	Guard   Expression  // optional: if guard expression; nil if absent
	Body    Expression  // right-hand side expression
}

func (m *MatchArm) String() string {
	var out bytes.Buffer
	out.WriteString(m.Pattern.String())
	if m.Guard != nil {
		out.WriteString(" if ")
		out.WriteString(m.Guard.String())
	}
	out.WriteString(" => ")
	out.WriteString(m.Body.String())
	return out.String()
}

// AgentStateBlock is the `state { fields }` section inside an AgentDecl.
type AgentStateBlock struct {
	Token  token.Token // 'state'
	Fields []Field
}

func (a *AgentStateBlock) String() string {
	var out bytes.Buffer
	out.WriteString("state { ")
	parts := make([]string, len(a.Fields))
	for i, f := range a.Fields {
		parts[i] = f.String()
	}
	out.WriteString(strings.Join(parts, "; "))
	out.WriteString(" }")
	return out.String()
}

// MsgHandler is a message handler inside an AgentDecl: msg MsgName(params) { body }.
type MsgHandler struct {
	Token  token.Token // 'msg'
	Name   string
	Params []Param
	Body   *BlockExpr
}

func (m *MsgHandler) String() string {
	var out bytes.Buffer
	out.WriteString("msg ")
	out.WriteString(m.Name)
	out.WriteString("(")
	parts := make([]string, len(m.Params))
	for i, p := range m.Params {
		parts[i] = p.String()
	}
	out.WriteString(strings.Join(parts, ", "))
	out.WriteString(") ")
	out.WriteString(m.Body.String())
	return out.String()
}

// EnumVariant is a single variant in an EnumDecl.
// If Fields is non-nil the variant carries data; otherwise it is a unit variant.
type EnumVariant struct {
	Token  token.Token // IDENT token of the variant name
	Name   string
	Fields []TypeExpr // nil for unit variants; one element for newtype; many for tuple
}

func (e *EnumVariant) String() string {
	if len(e.Fields) == 0 {
		return e.Name
	}
	parts := make([]string, len(e.Fields))
	for i, f := range e.Fields {
		parts[i] = f.String()
	}
	return e.Name + "(" + strings.Join(parts, ", ") + ")"
}

// TraitMethod is a method signature (without body) inside a TraitDecl.
type TraitMethod struct {
	Token      token.Token // 'fn'
	Name       string
	Params     []Param
	ReturnType TypeExpr // nil means unit
}

func (t *TraitMethod) String() string {
	var out bytes.Buffer
	out.WriteString("fn ")
	out.WriteString(t.Name)
	out.WriteString("(")
	parts := make([]string, len(t.Params))
	for i, p := range t.Params {
		parts[i] = p.String()
	}
	out.WriteString(strings.Join(parts, ", "))
	out.WriteString(")")
	if t.ReturnType != nil {
		out.WriteString(" -> ")
		out.WriteString(t.ReturnType.String())
	}
	return out.String()
}

// ---------------------------------------------------------------------------
// Expression nodes
// ---------------------------------------------------------------------------

// Ident is an identifier reference: x, my_var, AgentName.
type Ident struct {
	Token token.Token // the IDENT token
	Value string
}

func (e *Ident) expressionNode()      {}
func (e *Ident) TokenLiteral() string { return e.Token.Literal }
func (e *Ident) String() string       { return e.Value }

// IntLiteral is an integer literal: 42, 0, 1_000_000.
type IntLiteral struct {
	Token token.Token // the INT token
	Value int64
}

func (e *IntLiteral) expressionNode()      {}
func (e *IntLiteral) TokenLiteral() string { return e.Token.Literal }
func (e *IntLiteral) String() string       { return e.Token.Literal }

// FloatLiteral is a floating-point literal: 3.14, 2.0e10.
type FloatLiteral struct {
	Token token.Token // the FLOAT token
	Value float64
}

func (e *FloatLiteral) expressionNode()      {}
func (e *FloatLiteral) TokenLiteral() string { return e.Token.Literal }
func (e *FloatLiteral) String() string       { return e.Token.Literal }

// StringLiteral is a double-quoted string: "hello, world".
type StringLiteral struct {
	Token token.Token // the STRING token
	Value string
}

func (e *StringLiteral) expressionNode()      {}
func (e *StringLiteral) TokenLiteral() string { return e.Token.Literal }
func (e *StringLiteral) String() string       { return `"` + e.Value + `"` }

// BoolLiteral is a boolean literal: true or false.
type BoolLiteral struct {
	Token token.Token // TRUE or FALSE
	Value bool
}

func (e *BoolLiteral) expressionNode()      {}
func (e *BoolLiteral) TokenLiteral() string { return e.Token.Literal }
func (e *BoolLiteral) String() string       { return e.Token.Literal }

// BytesLiteral is a hex byte-string literal: 0xdeadbeef.
type BytesLiteral struct {
	Token token.Token // the BYTES token
	Value []byte
}

func (e *BytesLiteral) expressionNode()      {}
func (e *BytesLiteral) TokenLiteral() string { return e.Token.Literal }
func (e *BytesLiteral) String() string       { return e.Token.Literal }

// NilLiteral represents the nil keyword.
type NilLiteral struct {
	Token token.Token // NIL
}

func (e *NilLiteral) expressionNode()      {}
func (e *NilLiteral) TokenLiteral() string { return e.Token.Literal }
func (e *NilLiteral) String() string       { return "nil" }

// AddressLiteral is a blockchain address literal: @0x1234abcd....
type AddressLiteral struct {
	Token token.Token // the ADDRESS token
	Value string      // the raw literal text including the leading @
}

func (e *AddressLiteral) expressionNode()      {}
func (e *AddressLiteral) TokenLiteral() string { return e.Token.Literal }
func (e *AddressLiteral) String() string       { return e.Value }

// PrefixExpr is a monadic (prefix/unary) expression: -x, !x, ~x, #x, *x, &x.
//
// Operators follow J-style arity overloading:
//
//	-  negate
//	!  logical not
//	~  bitwise not
//	#  length / cardinality
//	*  dereference
//	&  address-of
type PrefixExpr struct {
	Token    token.Token // the operator token
	Operator string      // "-", "!", "~", "#", "*", "&"
	Right    Expression
}

func (e *PrefixExpr) expressionNode()      {}
func (e *PrefixExpr) TokenLiteral() string { return e.Token.Literal }
func (e *PrefixExpr) String() string       { return "(" + e.Operator + e.Right.String() + ")" }

// InfixExpr is a dyadic (binary infix) expression: x + y, x == y, x && y, etc.
type InfixExpr struct {
	Token    token.Token // the operator token
	Left     Expression
	Operator string // "+", "-", "*", "/", "%", "==", "!=", "<", ">", "<=", ">=", "&&", "||", etc.
	Right    Expression
}

func (e *InfixExpr) expressionNode()      {}
func (e *InfixExpr) TokenLiteral() string { return e.Token.Literal }
func (e *InfixExpr) String() string {
	return "(" + e.Left.String() + " " + e.Operator + " " + e.Right.String() + ")"
}

// IndexExpr is a subscript expression: a[i].
type IndexExpr struct {
	Token token.Token // '['
	Left  Expression
	Index Expression
}

func (e *IndexExpr) expressionNode()      {}
func (e *IndexExpr) TokenLiteral() string { return e.Token.Literal }
func (e *IndexExpr) String() string {
	return "(" + e.Left.String() + "[" + e.Index.String() + "])"
}

// FieldExpr is a field-access expression: a.b.
type FieldExpr struct {
	Token  token.Token // '.'
	Object Expression
	Field  string
}

func (e *FieldExpr) expressionNode()      {}
func (e *FieldExpr) TokenLiteral() string { return e.Token.Literal }
func (e *FieldExpr) String() string       { return "(" + e.Object.String() + "." + e.Field + ")" }

// CallExpr is a free function call: f(x, y).
type CallExpr struct {
	Token     token.Token  // '('
	Function  Expression   // the callee — usually an Ident or FieldExpr
	Arguments []Expression
}

func (e *CallExpr) expressionNode()      {}
func (e *CallExpr) TokenLiteral() string { return e.Token.Literal }
func (e *CallExpr) String() string {
	var out bytes.Buffer
	out.WriteString(e.Function.String())
	out.WriteString("(")
	args := make([]string, len(e.Arguments))
	for i, a := range e.Arguments {
		args[i] = a.String()
	}
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")
	return out.String()
}

// MethodCallExpr is a method call on a receiver: a.f(x, y).
//
// This is kept distinct from CallExpr so that the type checker can perform
// receiver-type dispatch without deconstructing a FieldExpr + CallExpr pair.
type MethodCallExpr struct {
	Token     token.Token  // '.'
	Receiver  Expression
	Method    string
	Arguments []Expression
}

func (e *MethodCallExpr) expressionNode()      {}
func (e *MethodCallExpr) TokenLiteral() string { return e.Token.Literal }
func (e *MethodCallExpr) String() string {
	var out bytes.Buffer
	out.WriteString(e.Receiver.String())
	out.WriteString(".")
	out.WriteString(e.Method)
	out.WriteString("(")
	args := make([]string, len(e.Arguments))
	for i, a := range e.Arguments {
		args[i] = a.String()
	}
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")
	return out.String()
}

// BlockExpr is a brace-delimited sequence of statements with an optional
// trailing expression: { stmt; stmt; expr }.
//
// When Tail is non-nil the block evaluates to the value of that expression;
// otherwise the block has unit type.
type BlockExpr struct {
	Token      token.Token // '{'
	Statements []Statement
	Tail       Expression // optional trailing expression (no semicolon)
}

func (e *BlockExpr) expressionNode()      {}
func (e *BlockExpr) TokenLiteral() string { return e.Token.Literal }
func (e *BlockExpr) String() string {
	var out bytes.Buffer
	out.WriteString("{ ")
	for _, s := range e.Statements {
		out.WriteString(s.String())
		out.WriteString("; ")
	}
	if e.Tail != nil {
		out.WriteString(e.Tail.String())
		out.WriteString(" ")
	}
	out.WriteString("}")
	return out.String()
}

// IfExpr is an if/else expression: if cond { consequence } else { alternative }.
//
// Alternative is optional. When present it may be another IfExpr (else if) or
// a BlockExpr (plain else).
type IfExpr struct {
	Token       token.Token // 'if'
	Condition   Expression
	Consequence *BlockExpr
	Alternative Expression // *BlockExpr or *IfExpr; nil when there is no else branch
}

func (e *IfExpr) expressionNode()      {}
func (e *IfExpr) TokenLiteral() string { return e.Token.Literal }
func (e *IfExpr) String() string {
	var out bytes.Buffer
	out.WriteString("if ")
	out.WriteString(e.Condition.String())
	out.WriteString(" ")
	out.WriteString(e.Consequence.String())
	if e.Alternative != nil {
		out.WriteString(" else ")
		out.WriteString(e.Alternative.String())
	}
	return out.String()
}

// MatchExpr is a pattern-matching expression: match subject { arms }.
type MatchExpr struct {
	Token   token.Token // 'match'
	Subject Expression
	Arms    []MatchArm
}

func (e *MatchExpr) expressionNode()      {}
func (e *MatchExpr) TokenLiteral() string { return e.Token.Literal }
func (e *MatchExpr) String() string {
	var out bytes.Buffer
	out.WriteString("match ")
	out.WriteString(e.Subject.String())
	out.WriteString(" { ")
	arms := make([]string, len(e.Arms))
	for i, a := range e.Arms {
		arms[i] = a.String()
	}
	out.WriteString(strings.Join(arms, ", "))
	out.WriteString(" }")
	return out.String()
}

// RangeExpr is an inclusive or exclusive range: a..b.
//
// PROBE uses J-inspired syntax where .. is a dyadic range constructor.
// Half-open ranges (a..) and full-open ranges (..) may be represented by
// setting Start or End to nil.
type RangeExpr struct {
	Token token.Token // '..'
	Start Expression  // nil for open left end
	End   Expression  // nil for open right end
}

func (e *RangeExpr) expressionNode()      {}
func (e *RangeExpr) TokenLiteral() string { return e.Token.Literal }
func (e *RangeExpr) String() string {
	start, end := "_", "_"
	if e.Start != nil {
		start = e.Start.String()
	}
	if e.End != nil {
		end = e.End.String()
	}
	return "(" + start + ".." + end + ")"
}

// ArrayExpr is an array literal: [1, 2, 3].
type ArrayExpr struct {
	Token    token.Token  // '['
	Elements []Expression
}

func (e *ArrayExpr) expressionNode()      {}
func (e *ArrayExpr) TokenLiteral() string { return e.Token.Literal }
func (e *ArrayExpr) String() string {
	var out bytes.Buffer
	out.WriteString("[")
	elems := make([]string, len(e.Elements))
	for i, el := range e.Elements {
		elems[i] = el.String()
	}
	out.WriteString(strings.Join(elems, ", "))
	out.WriteString("]")
	return out.String()
}

// MoveExpr transfers ownership of a linear resource: move x.
//
// After a MoveExpr the source binding is consumed and may not be used again.
// This is the primary mechanism for linear-type semantics in PROBE.
type MoveExpr struct {
	Token token.Token // 'move'
	Value Expression
}

func (e *MoveExpr) expressionNode()      {}
func (e *MoveExpr) TokenLiteral() string { return e.Token.Literal }
func (e *MoveExpr) String() string       { return "(move " + e.Value.String() + ")" }

// CopyExpr is an explicit clone of a copyable value: copy x.
//
// Values of types that implement Copy may be explicitly duplicated; the
// compiler will reject copy on linear (non-Copy) types.
type CopyExpr struct {
	Token token.Token // 'copy'
	Value Expression
}

func (e *CopyExpr) expressionNode()      {}
func (e *CopyExpr) TokenLiteral() string { return e.Token.Literal }
func (e *CopyExpr) String() string       { return "(copy " + e.Value.String() + ")" }

// SpawnExpr creates a new agent instance: spawn AgentName { field: val, ... }.
//
// The Fields map provides initial state for the agent. The result of a
// SpawnExpr is an agent handle (reference) typed as the named agent type.
type SpawnExpr struct {
	Token  token.Token       // 'spawn'
	Agent  string            // name of the agent type to instantiate
	Fields map[string]Expression
}

func (e *SpawnExpr) expressionNode()      {}
func (e *SpawnExpr) TokenLiteral() string { return e.Token.Literal }
func (e *SpawnExpr) String() string {
	var out bytes.Buffer
	out.WriteString("spawn ")
	out.WriteString(e.Agent)
	out.WriteString(" { ")
	parts := make([]string, 0, len(e.Fields))
	for k, v := range e.Fields {
		parts = append(parts, k+": "+v.String())
	}
	out.WriteString(strings.Join(parts, ", "))
	out.WriteString(" }")
	return out.String()
}

// SendExpr delivers a message to an agent: send target msg.
//
// In PROBE's actor model, send is non-blocking; the message is enqueued on
// the target agent's mailbox. The expression has unit type.
type SendExpr struct {
	Token   token.Token // 'send'
	Target  Expression  // agent handle
	Message Expression  // message value
}

func (e *SendExpr) expressionNode()      {}
func (e *SendExpr) TokenLiteral() string { return e.Token.Literal }
func (e *SendExpr) String() string {
	return "(send " + e.Target.String() + " " + e.Message.String() + ")"
}

// RecvExpr blocks the current agent until a message arrives: recv.
//
// The type of the expression is inferred from the enclosing msg handler or
// from the message type the agent can accept.
type RecvExpr struct {
	Token token.Token // 'recv'
}

func (e *RecvExpr) expressionNode()      {}
func (e *RecvExpr) TokenLiteral() string { return e.Token.Literal }
func (e *RecvExpr) String() string       { return "recv" }

// ---------------------------------------------------------------------------
// Statement nodes
// ---------------------------------------------------------------------------

// LetStmt introduces a new binding: let [mut] name [: Type] = expr.
//
// When Mutable is true the binding may be reassigned. The Type annotation is
// optional; when absent the compiler infers it from the initialiser.
type LetStmt struct {
	Token   token.Token // 'let'
	Mutable bool
	Name    *Ident
	Type    TypeExpr   // optional; nil when omitted
	Value   Expression // optional initialiser; nil for declaration-only
}

func (s *LetStmt) statementNode()       {}
func (s *LetStmt) TokenLiteral() string { return s.Token.Literal }
func (s *LetStmt) String() string {
	var out bytes.Buffer
	out.WriteString("let ")
	if s.Mutable {
		out.WriteString("mut ")
	}
	out.WriteString(s.Name.String())
	if s.Type != nil {
		out.WriteString(": ")
		out.WriteString(s.Type.String())
	}
	if s.Value != nil {
		out.WriteString(" = ")
		out.WriteString(s.Value.String())
	}
	return out.String()
}

// AssignStmt assigns (or compound-assigns) a value to a place:
// x = expr, x += expr, x >>= expr, etc.
type AssignStmt struct {
	Token    token.Token // the assignment operator token
	Target   Expression  // must be a valid place expression
	Operator string      // "=", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>="
	Value    Expression
}

func (s *AssignStmt) statementNode()       {}
func (s *AssignStmt) TokenLiteral() string { return s.Token.Literal }
func (s *AssignStmt) String() string {
	return s.Target.String() + " " + s.Operator + " " + s.Value.String()
}

// ReturnStmt exits the enclosing function, optionally with a value.
type ReturnStmt struct {
	Token token.Token // 'return'
	Value Expression  // nil for bare return
}

func (s *ReturnStmt) statementNode()       {}
func (s *ReturnStmt) TokenLiteral() string { return s.Token.Literal }
func (s *ReturnStmt) String() string {
	if s.Value != nil {
		return "return " + s.Value.String()
	}
	return "return"
}

// ExprStmt wraps an expression used in a statement position.
// The value of the expression is discarded.
type ExprStmt struct {
	Token      token.Token // first token of the expression
	Expression Expression
}

func (s *ExprStmt) statementNode()       {}
func (s *ExprStmt) TokenLiteral() string { return s.Token.Literal }
func (s *ExprStmt) String() string       { return s.Expression.String() }

// ForStmt is a for-in loop: for binding in iterable { body }.
type ForStmt struct {
	Token    token.Token // 'for'
	Binding  *Ident      // loop variable
	Iterable Expression
	Body     *BlockExpr
}

func (s *ForStmt) statementNode()       {}
func (s *ForStmt) TokenLiteral() string { return s.Token.Literal }
func (s *ForStmt) String() string {
	return "for " + s.Binding.String() + " in " + s.Iterable.String() + " " + s.Body.String()
}

// WhileStmt is a while loop: while condition { body }.
type WhileStmt struct {
	Token     token.Token // 'while'
	Condition Expression
	Body      *BlockExpr
}

func (s *WhileStmt) statementNode()       {}
func (s *WhileStmt) TokenLiteral() string { return s.Token.Literal }
func (s *WhileStmt) String() string {
	return "while " + s.Condition.String() + " " + s.Body.String()
}

// BreakStmt exits the innermost loop.
type BreakStmt struct {
	Token token.Token // 'break'
}

func (s *BreakStmt) statementNode()       {}
func (s *BreakStmt) TokenLiteral() string { return s.Token.Literal }
func (s *BreakStmt) String() string       { return "break" }

// ContinueStmt skips to the next iteration of the innermost loop.
type ContinueStmt struct {
	Token token.Token // 'continue'
}

func (s *ContinueStmt) statementNode()       {}
func (s *ContinueStmt) TokenLiteral() string { return s.Token.Literal }
func (s *ContinueStmt) String() string       { return "continue" }

// DropStmt explicitly destroys a linear resource: drop x.
//
// For types that do not implement Copy, drop is the canonical way to release
// a resource before the end of its lexical scope. The compiler enforces that
// every linear value is either moved, returned, or dropped exactly once.
type DropStmt struct {
	Token token.Token // 'drop'
	Value *Ident      // the binding to drop
}

func (s *DropStmt) statementNode()       {}
func (s *DropStmt) TokenLiteral() string { return s.Token.Literal }
func (s *DropStmt) String() string       { return "drop " + s.Value.String() }

// EmitStmt emits a blockchain event/log: emit EventName { field: val, ... }.
type EmitStmt struct {
	Token  token.Token       // 'emit'
	Event  string            // name of the event type
	Fields map[string]Expression
}

func (s *EmitStmt) statementNode()       {}
func (s *EmitStmt) TokenLiteral() string { return s.Token.Literal }
func (s *EmitStmt) String() string {
	var out bytes.Buffer
	out.WriteString("emit ")
	out.WriteString(s.Event)
	out.WriteString(" { ")
	parts := make([]string, 0, len(s.Fields))
	for k, v := range s.Fields {
		parts = append(parts, k+": "+v.String())
	}
	out.WriteString(strings.Join(parts, ", "))
	out.WriteString(" }")
	return out.String()
}

// RequireStmt is a runtime assertion that reverts on failure:
// require(condition, "message").
type RequireStmt struct {
	Token     token.Token // 'require'
	Condition Expression
	Message   Expression // typically a StringLiteral; nil when omitted
}

func (s *RequireStmt) statementNode()       {}
func (s *RequireStmt) TokenLiteral() string { return s.Token.Literal }
func (s *RequireStmt) String() string {
	var out bytes.Buffer
	out.WriteString("require(")
	out.WriteString(s.Condition.String())
	if s.Message != nil {
		out.WriteString(", ")
		out.WriteString(s.Message.String())
	}
	out.WriteString(")")
	return out.String()
}

// ---------------------------------------------------------------------------
// Top-level declaration nodes
// ---------------------------------------------------------------------------

// FnDecl declares a named function: [pub] fn name(params) [-> RetType] { body }.
type FnDecl struct {
	Token      token.Token // 'fn'
	Public     bool
	Name       string
	Params     []Param
	ReturnType TypeExpr   // nil means unit
	Body       *BlockExpr
}

func (d *FnDecl) declarationNode()      {}
func (d *FnDecl) TokenLiteral() string  { return d.Token.Literal }
func (d *FnDecl) String() string {
	var out bytes.Buffer
	if d.Public {
		out.WriteString("pub ")
	}
	out.WriteString("fn ")
	out.WriteString(d.Name)
	out.WriteString("(")
	params := make([]string, len(d.Params))
	for i, p := range d.Params {
		params[i] = p.String()
	}
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(")")
	if d.ReturnType != nil {
		out.WriteString(" -> ")
		out.WriteString(d.ReturnType.String())
	}
	out.WriteString(" ")
	out.WriteString(d.Body.String())
	return out.String()
}

// StructDecl declares a product type: [pub] struct Name { fields }.
type StructDecl struct {
	Token  token.Token // 'struct'
	Public bool
	Name   string
	Fields []Field
}

func (d *StructDecl) declarationNode()      {}
func (d *StructDecl) TokenLiteral() string  { return d.Token.Literal }
func (d *StructDecl) String() string {
	var out bytes.Buffer
	if d.Public {
		out.WriteString("pub ")
	}
	out.WriteString("struct ")
	out.WriteString(d.Name)
	out.WriteString(" { ")
	parts := make([]string, len(d.Fields))
	for i, f := range d.Fields {
		parts[i] = f.String()
	}
	out.WriteString(strings.Join(parts, "; "))
	out.WriteString(" }")
	return out.String()
}

// EnumDecl declares a sum type: [pub] enum Name { Variant1, Variant2(Type), ... }.
type EnumDecl struct {
	Token    token.Token // 'enum'
	Public   bool
	Name     string
	Variants []EnumVariant
}

func (d *EnumDecl) declarationNode()      {}
func (d *EnumDecl) TokenLiteral() string  { return d.Token.Literal }
func (d *EnumDecl) String() string {
	var out bytes.Buffer
	if d.Public {
		out.WriteString("pub ")
	}
	out.WriteString("enum ")
	out.WriteString(d.Name)
	out.WriteString(" { ")
	parts := make([]string, len(d.Variants))
	for i, v := range d.Variants {
		parts[i] = v.String()
	}
	out.WriteString(strings.Join(parts, ", "))
	out.WriteString(" }")
	return out.String()
}

// TraitDecl declares an abstract interface: [pub] trait Name { method sigs }.
type TraitDecl struct {
	Token   token.Token // 'trait'
	Public  bool
	Name    string
	Methods []TraitMethod
}

func (d *TraitDecl) declarationNode()      {}
func (d *TraitDecl) TokenLiteral() string  { return d.Token.Literal }
func (d *TraitDecl) String() string {
	var out bytes.Buffer
	if d.Public {
		out.WriteString("pub ")
	}
	out.WriteString("trait ")
	out.WriteString(d.Name)
	out.WriteString(" { ")
	parts := make([]string, len(d.Methods))
	for i, m := range d.Methods {
		parts[i] = m.String()
	}
	out.WriteString(strings.Join(parts, "; "))
	out.WriteString(" }")
	return out.String()
}

// ImplDecl provides method implementations for a type, optionally satisfying
// a trait: impl [TraitName for] TypeName { methods }.
type ImplDecl struct {
	Token    token.Token // 'impl'
	Trait    string      // empty when not implementing a trait
	TypeName string
	Methods  []FnDecl
}

func (d *ImplDecl) declarationNode()      {}
func (d *ImplDecl) TokenLiteral() string  { return d.Token.Literal }
func (d *ImplDecl) String() string {
	var out bytes.Buffer
	out.WriteString("impl ")
	if d.Trait != "" {
		out.WriteString(d.Trait)
		out.WriteString(" for ")
	}
	out.WriteString(d.TypeName)
	out.WriteString(" { ")
	parts := make([]string, len(d.Methods))
	for i, m := range d.Methods {
		parts[i] = m.String()
	}
	out.WriteString(strings.Join(parts, " "))
	out.WriteString(" }")
	return out.String()
}

// AgentDecl declares a first-class agent type:
//
//	agent Name {
//	    state { fields }
//	    msg MsgName(params) { body }
//	    ...
//	}
//
// Agents are the unit of concurrency in PROBE's actor model. Each agent has
// isolated state and communicates exclusively via message passing.
type AgentDecl struct {
	Token    token.Token // 'agent'
	Public   bool
	Name     string
	State    *AgentStateBlock // may be nil if the agent has no state
	Handlers []MsgHandler
}

func (d *AgentDecl) declarationNode()      {}
func (d *AgentDecl) TokenLiteral() string  { return d.Token.Literal }
func (d *AgentDecl) String() string {
	var out bytes.Buffer
	if d.Public {
		out.WriteString("pub ")
	}
	out.WriteString("agent ")
	out.WriteString(d.Name)
	out.WriteString(" { ")
	if d.State != nil {
		out.WriteString(d.State.String())
		out.WriteString(" ")
	}
	for _, h := range d.Handlers {
		out.WriteString(h.String())
		out.WriteString(" ")
	}
	out.WriteString("}")
	return out.String()
}

// ResourceDecl declares a linear resource type (cannot be implicitly copied
// or dropped): [pub] resource Name { fields }.
//
// Every value of a resource type must be either moved, returned, or
// explicitly dropped exactly once. The compiler enforces this at compile time.
type ResourceDecl struct {
	Token  token.Token // 'resource'
	Public bool
	Name   string
	Fields []Field
}

func (d *ResourceDecl) declarationNode()      {}
func (d *ResourceDecl) TokenLiteral() string  { return d.Token.Literal }
func (d *ResourceDecl) String() string {
	var out bytes.Buffer
	if d.Public {
		out.WriteString("pub ")
	}
	out.WriteString("resource ")
	out.WriteString(d.Name)
	out.WriteString(" { ")
	parts := make([]string, len(d.Fields))
	for i, f := range d.Fields {
		parts[i] = f.String()
	}
	out.WriteString(strings.Join(parts, "; "))
	out.WriteString(" }")
	return out.String()
}

// TypeDecl introduces a type alias: [pub] type Name = OtherType.
type TypeDecl struct {
	Token  token.Token // 'type'
	Public bool
	Name   string
	Type   TypeExpr
}

func (d *TypeDecl) declarationNode()      {}
func (d *TypeDecl) TokenLiteral() string  { return d.Token.Literal }
func (d *TypeDecl) String() string {
	var out bytes.Buffer
	if d.Public {
		out.WriteString("pub ")
	}
	out.WriteString("type ")
	out.WriteString(d.Name)
	out.WriteString(" = ")
	out.WriteString(d.Type.String())
	return out.String()
}

// UseDecl brings an item from another module into scope: use module::item [as alias].
type UseDecl struct {
	Token token.Token // 'use'
	Path  []string    // e.g. ["std", "collections", "HashMap"]
	Alias string      // empty when there is no 'as' clause
}

func (d *UseDecl) declarationNode()      {}
func (d *UseDecl) TokenLiteral() string  { return d.Token.Literal }
func (d *UseDecl) String() string {
	var out bytes.Buffer
	out.WriteString("use ")
	out.WriteString(strings.Join(d.Path, "::"))
	if d.Alias != "" {
		out.WriteString(" as ")
		out.WriteString(d.Alias)
	}
	return out.String()
}

// ModDecl declares an inline or external module: [pub] mod name { decls }.
//
// When Declarations is nil the module body is external (in a separate file).
type ModDecl struct {
	Token        token.Token // 'mod'
	Public       bool
	Name         string
	Declarations []Declaration // nil for external modules
}

func (d *ModDecl) declarationNode()      {}
func (d *ModDecl) TokenLiteral() string  { return d.Token.Literal }
func (d *ModDecl) String() string {
	var out bytes.Buffer
	if d.Public {
		out.WriteString("pub ")
	}
	out.WriteString("mod ")
	out.WriteString(d.Name)
	if d.Declarations != nil {
		out.WriteString(" { ")
		parts := make([]string, len(d.Declarations))
		for i, decl := range d.Declarations {
			parts[i] = decl.String()
		}
		out.WriteString(strings.Join(parts, " "))
		out.WriteString(" }")
	}
	return out.String()
}

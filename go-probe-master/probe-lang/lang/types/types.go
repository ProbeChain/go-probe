// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.
//
// The ProbeChain is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Package types defines the PROBE language type system.
//
// Design principles:
//   - Linear types for resource safety (assets can't be duplicated or lost)
//   - Move-inspired: explicit move, copy, drop semantics
//   - Bytecode-level verification (safety holds even for buggy compiler output)
//   - Value types (bool, integers, floats) are freely copyable
//   - Resource types are linear: must be used exactly once
package types

import (
	"fmt"
	"strings"
)

// Kind categorizes the fundamental shape of a type.
type Kind int

const (
	KindVoid    Kind = iota
	KindBool
	KindU8
	KindU16
	KindU32
	KindU64
	KindU128
	KindU256
	KindI8
	KindI16
	KindI32
	KindI64
	KindF32
	KindF64
	KindString
	KindBytes
	KindAddress  // 20-byte blockchain address
	KindArray    // [T; N]
	KindSlice    // [T]
	KindRef      // &T
	KindMutRef   // &mut T
	KindStruct
	KindEnum
	KindFn
	KindAgent    // First-class agent type
	KindResource // Linear resource type (cannot copy/drop implicitly)
)

var kindNames = [...]string{
	KindVoid:     "void",
	KindBool:     "bool",
	KindU8:       "u8",
	KindU16:      "u16",
	KindU32:      "u32",
	KindU64:      "u64",
	KindU128:     "u128",
	KindU256:     "u256",
	KindI8:       "i8",
	KindI16:      "i16",
	KindI32:      "i32",
	KindI64:      "i64",
	KindF32:      "f32",
	KindF64:      "f64",
	KindString:   "string",
	KindBytes:    "bytes",
	KindAddress:  "address",
	KindArray:    "array",
	KindSlice:    "slice",
	KindRef:      "ref",
	KindMutRef:   "mut_ref",
	KindStruct:   "struct",
	KindEnum:     "enum",
	KindFn:       "fn",
	KindAgent:    "agent",
	KindResource: "resource",
}

func (k Kind) String() string {
	if int(k) < len(kindNames) {
		return kindNames[k]
	}
	return fmt.Sprintf("kind(%d)", k)
}

// Type is the interface that all PROBE types implement.
type Type interface {
	// Kind returns the fundamental category of this type.
	Kind() Kind

	// String returns the human-readable representation.
	String() string

	// Equals reports whether two types are structurally identical.
	Equals(other Type) bool

	// IsLinear reports whether this is a linear resource type.
	// Linear types must be used exactly once: they cannot be implicitly
	// copied or dropped. The verifier enforces this at bytecode level.
	IsLinear() bool

	// IsCopyable reports whether values of this type may be freely duplicated.
	// Value types (integers, bool, address) are copyable.
	// Linear resource types are not.
	IsCopyable() bool

	// Size returns the size of the type in bytes, used for VM register
	// allocation. Returns -1 for dynamically-sized types.
	Size() int
}

// ---- Primitive types -------------------------------------------------------

// primitiveType is the concrete implementation for all built-in scalar types.
type primitiveType struct {
	kind Kind
}

func (p *primitiveType) Kind() Kind        { return p.kind }
func (p *primitiveType) IsLinear() bool    { return false }
func (p *primitiveType) IsCopyable() bool  { return true }

func (p *primitiveType) String() string {
	return p.kind.String()
}

func (p *primitiveType) Equals(other Type) bool {
	if other == nil {
		return false
	}
	return p.kind == other.Kind()
}

func (p *primitiveType) Size() int {
	switch p.kind {
	case KindVoid:
		return 0
	case KindBool, KindU8, KindI8:
		return 1
	case KindU16, KindI16:
		return 2
	case KindU32, KindI32, KindF32:
		return 4
	case KindU64, KindI64, KindF64:
		return 8
	case KindU128:
		return 16
	case KindU256:
		return 32
	case KindAddress:
		return 20
	case KindString, KindBytes:
		return -1 // dynamically sized
	default:
		return -1
	}
}

// Pre-allocated singletons for all primitive types.
var (
	Void    Type = &primitiveType{kind: KindVoid}
	Bool    Type = &primitiveType{kind: KindBool}
	U8      Type = &primitiveType{kind: KindU8}
	U16     Type = &primitiveType{kind: KindU16}
	U32     Type = &primitiveType{kind: KindU32}
	U64     Type = &primitiveType{kind: KindU64}
	U128    Type = &primitiveType{kind: KindU128}
	U256    Type = &primitiveType{kind: KindU256}
	I8      Type = &primitiveType{kind: KindI8}
	I16     Type = &primitiveType{kind: KindI16}
	I32     Type = &primitiveType{kind: KindI32}
	I64     Type = &primitiveType{kind: KindI64}
	F32     Type = &primitiveType{kind: KindF32}
	F64     Type = &primitiveType{kind: KindF64}
	String  Type = &primitiveType{kind: KindString}
	Bytes   Type = &primitiveType{kind: KindBytes}
	Address Type = &primitiveType{kind: KindAddress}
)

// ---- Field -----------------------------------------------------------------

// Field represents a named field inside a struct or resource.
type Field struct {
	Name string
	Type Type
}

func (f Field) String() string {
	return fmt.Sprintf("%s: %s", f.Name, f.Type)
}

// ---- Composite types -------------------------------------------------------

// ArrayType is [Elem; Len].
type ArrayType struct {
	Elem Type
	Len  int
}

func (a *ArrayType) Kind() Kind        { return KindArray }
func (a *ArrayType) IsLinear() bool    { return a.Elem.IsLinear() }
func (a *ArrayType) IsCopyable() bool  { return a.Elem.IsCopyable() }
func (a *ArrayType) Size() int {
	elemSize := a.Elem.Size()
	if elemSize < 0 {
		return -1
	}
	return elemSize * a.Len
}
func (a *ArrayType) String() string {
	return fmt.Sprintf("[%s; %d]", a.Elem, a.Len)
}
func (a *ArrayType) Equals(other Type) bool {
	if other == nil || other.Kind() != KindArray {
		return false
	}
	o := other.(*ArrayType)
	return a.Len == o.Len && a.Elem.Equals(o.Elem)
}

// SliceType is [Elem] — a dynamically-sized sequence.
type SliceType struct {
	Elem Type
}

func (s *SliceType) Kind() Kind        { return KindSlice }
func (s *SliceType) IsLinear() bool    { return s.Elem.IsLinear() }
func (s *SliceType) IsCopyable() bool  { return s.Elem.IsCopyable() }
func (s *SliceType) Size() int         { return -1 }
func (s *SliceType) String() string    { return fmt.Sprintf("[%s]", s.Elem) }
func (s *SliceType) Equals(other Type) bool {
	if other == nil || other.Kind() != KindSlice {
		return false
	}
	return s.Elem.Equals(other.(*SliceType).Elem)
}

// RefType is &T (immutable reference) or &mut T (mutable reference).
// References are never linear: they do not own the underlying value.
type RefType struct {
	Inner   Type
	Mutable bool
}

func (r *RefType) Kind() Kind {
	if r.Mutable {
		return KindMutRef
	}
	return KindRef
}
func (r *RefType) IsLinear() bool   { return false }
func (r *RefType) IsCopyable() bool { return true } // references can be freely shared
func (r *RefType) Size() int        { return 8 }    // pointer width on 64-bit VM
func (r *RefType) String() string {
	if r.Mutable {
		return fmt.Sprintf("&mut %s", r.Inner)
	}
	return fmt.Sprintf("&%s", r.Inner)
}
func (r *RefType) Equals(other Type) bool {
	if other == nil {
		return false
	}
	o, ok := other.(*RefType)
	if !ok {
		return false
	}
	return r.Mutable == o.Mutable && r.Inner.Equals(o.Inner)
}

// StructType is a named product type.
// A struct is linear if any of its fields is linear.
type StructType struct {
	Name   string
	Fields []Field
}

func (s *StructType) Kind() Kind { return KindStruct }
func (s *StructType) IsLinear() bool {
	for _, f := range s.Fields {
		if f.Type.IsLinear() {
			return true
		}
	}
	return false
}
func (s *StructType) IsCopyable() bool {
	for _, f := range s.Fields {
		if !f.Type.IsCopyable() {
			return false
		}
	}
	return true
}
func (s *StructType) Size() int {
	total := 0
	for _, f := range s.Fields {
		sz := f.Type.Size()
		if sz < 0 {
			return -1
		}
		total += sz
	}
	return total
}
func (s *StructType) String() string {
	parts := make([]string, len(s.Fields))
	for i, f := range s.Fields {
		parts[i] = f.String()
	}
	return fmt.Sprintf("struct %s { %s }", s.Name, strings.Join(parts, ", "))
}
func (s *StructType) Equals(other Type) bool {
	if other == nil || other.Kind() != KindStruct {
		return false
	}
	o := other.(*StructType)
	if s.Name != o.Name || len(s.Fields) != len(o.Fields) {
		return false
	}
	for i := range s.Fields {
		if s.Fields[i].Name != o.Fields[i].Name {
			return false
		}
		if !s.Fields[i].Type.Equals(o.Fields[i].Type) {
			return false
		}
	}
	return true
}

// Variant represents one arm of an enum.
type Variant struct {
	Name   string
	Fields []Field // empty for unit variants
}

// EnumType is a named sum type.
type EnumType struct {
	Name     string
	Variants []Variant
}

func (e *EnumType) Kind() Kind { return KindEnum }
func (e *EnumType) IsLinear() bool {
	for _, v := range e.Variants {
		for _, f := range v.Fields {
			if f.Type.IsLinear() {
				return true
			}
		}
	}
	return false
}
func (e *EnumType) IsCopyable() bool {
	for _, v := range e.Variants {
		for _, f := range v.Fields {
			if !f.Type.IsCopyable() {
				return false
			}
		}
	}
	return true
}
func (e *EnumType) Size() int { return -1 } // discriminant + largest variant
func (e *EnumType) String() string {
	names := make([]string, len(e.Variants))
	for i, v := range e.Variants {
		names[i] = v.Name
	}
	return fmt.Sprintf("enum %s { %s }", e.Name, strings.Join(names, " | "))
}
func (e *EnumType) Equals(other Type) bool {
	if other == nil || other.Kind() != KindEnum {
		return false
	}
	o := other.(*EnumType)
	if e.Name != o.Name || len(e.Variants) != len(o.Variants) {
		return false
	}
	for i := range e.Variants {
		if e.Variants[i].Name != o.Variants[i].Name {
			return false
		}
	}
	return true
}

// FnType describes a function signature.
type FnType struct {
	Params []Type
	Return Type
}

func (f *FnType) Kind() Kind        { return KindFn }
func (f *FnType) IsLinear() bool    { return false }
func (f *FnType) IsCopyable() bool  { return true }
func (f *FnType) Size() int         { return 8 } // function pointer
func (f *FnType) String() string {
	params := make([]string, len(f.Params))
	for i, p := range f.Params {
		params[i] = p.String()
	}
	ret := "void"
	if f.Return != nil {
		ret = f.Return.String()
	}
	return fmt.Sprintf("fn(%s) -> %s", strings.Join(params, ", "), ret)
}
func (f *FnType) Equals(other Type) bool {
	if other == nil || other.Kind() != KindFn {
		return false
	}
	o := other.(*FnType)
	if len(f.Params) != len(o.Params) {
		return false
	}
	for i := range f.Params {
		if !f.Params[i].Equals(o.Params[i]) {
			return false
		}
	}
	retEq := (f.Return == nil && o.Return == nil) ||
		(f.Return != nil && o.Return != nil && f.Return.Equals(o.Return))
	return retEq
}

// AgentType is a first-class agent: a concurrent entity that communicates via
// typed messages.
type AgentType struct {
	Name     string
	MsgTypes []Type
}

func (a *AgentType) Kind() Kind        { return KindAgent }
func (a *AgentType) IsLinear() bool    { return false }
func (a *AgentType) IsCopyable() bool  { return true } // agent handles are copyable
func (a *AgentType) Size() int         { return 8 }    // agent ID / handle
func (a *AgentType) String() string {
	msgs := make([]string, len(a.MsgTypes))
	for i, m := range a.MsgTypes {
		msgs[i] = m.String()
	}
	return fmt.Sprintf("agent %s[%s]", a.Name, strings.Join(msgs, ", "))
}
func (a *AgentType) Equals(other Type) bool {
	if other == nil || other.Kind() != KindAgent {
		return false
	}
	o := other.(*AgentType)
	if a.Name != o.Name || len(a.MsgTypes) != len(o.MsgTypes) {
		return false
	}
	for i := range a.MsgTypes {
		if !a.MsgTypes[i].Equals(o.MsgTypes[i]) {
			return false
		}
	}
	return true
}

// ResourceType is the linear resource type.
//
// Resources model blockchain assets (tokens, NFTs, capabilities) that must
// be explicitly transferred or destroyed. The verifier guarantees:
//
//   - A resource value cannot be duplicated (no implicit copy).
//   - A resource value cannot be silently discarded (no implicit drop).
//   - Consumption is tracked at bytecode level, independent of the compiler.
type ResourceType struct {
	Name   string
	Fields []Field
}

func (r *ResourceType) Kind() Kind       { return KindResource }
func (r *ResourceType) IsLinear() bool   { return true }
func (r *ResourceType) IsCopyable() bool { return false }
func (r *ResourceType) Size() int {
	total := 0
	for _, f := range r.Fields {
		sz := f.Type.Size()
		if sz < 0 {
			return -1
		}
		total += sz
	}
	return total
}
func (r *ResourceType) String() string {
	parts := make([]string, len(r.Fields))
	for i, f := range r.Fields {
		parts[i] = f.String()
	}
	return fmt.Sprintf("resource %s { %s }", r.Name, strings.Join(parts, ", "))
}
func (r *ResourceType) Equals(other Type) bool {
	if other == nil || other.Kind() != KindResource {
		return false
	}
	o := other.(*ResourceType)
	if r.Name != o.Name || len(r.Fields) != len(o.Fields) {
		return false
	}
	for i := range r.Fields {
		if r.Fields[i].Name != o.Fields[i].Name {
			return false
		}
		if !r.Fields[i].Type.Equals(o.Fields[i].Type) {
			return false
		}
	}
	return true
}

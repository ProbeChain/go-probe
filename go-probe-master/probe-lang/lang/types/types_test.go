// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.
//
// The ProbeChain is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package types

import (
	"testing"
)

// ---- Primitive type tests --------------------------------------------------

func TestPrimitiveKinds(t *testing.T) {
	cases := []struct {
		typ      Type
		wantKind Kind
		wantStr  string
		wantSize int
	}{
		{Void, KindVoid, "void", 0},
		{Bool, KindBool, "bool", 1},
		{U8, KindU8, "u8", 1},
		{U16, KindU16, "u16", 2},
		{U32, KindU32, "u32", 4},
		{U64, KindU64, "u64", 8},
		{U128, KindU128, "u128", 16},
		{U256, KindU256, "u256", 32},
		{I8, KindI8, "i8", 1},
		{I16, KindI16, "i16", 2},
		{I32, KindI32, "i32", 4},
		{I64, KindI64, "i64", 8},
		{F32, KindF32, "f32", 4},
		{F64, KindF64, "f64", 8},
		{Address, KindAddress, "address", 20},
	}
	for _, tc := range cases {
		t.Run(tc.wantStr, func(t *testing.T) {
			if tc.typ.Kind() != tc.wantKind {
				t.Errorf("Kind() = %v, want %v", tc.typ.Kind(), tc.wantKind)
			}
			if tc.typ.String() != tc.wantStr {
				t.Errorf("String() = %q, want %q", tc.typ.String(), tc.wantStr)
			}
			if tc.typ.Size() != tc.wantSize {
				t.Errorf("Size() = %d, want %d", tc.typ.Size(), tc.wantSize)
			}
			if tc.typ.IsLinear() {
				t.Errorf("IsLinear() should be false for primitive %s", tc.wantStr)
			}
			if !tc.typ.IsCopyable() {
				t.Errorf("IsCopyable() should be true for primitive %s", tc.wantStr)
			}
		})
	}
}

func TestDynamicSizedPrimitives(t *testing.T) {
	for _, typ := range []Type{String, Bytes} {
		if typ.Size() != -1 {
			t.Errorf("%s: Size() = %d, want -1 (dynamic)", typ, typ.Size())
		}
	}
}

func TestPrimitiveEquals(t *testing.T) {
	if !U64.Equals(U64) {
		t.Error("U64.Equals(U64) should be true")
	}
	if U64.Equals(U32) {
		t.Error("U64.Equals(U32) should be false")
	}
	if U64.Equals(nil) {
		t.Error("U64.Equals(nil) should be false")
	}
}

// ---- Composite type tests --------------------------------------------------

func TestArrayType(t *testing.T) {
	a := &ArrayType{Elem: U64, Len: 4}
	if a.Kind() != KindArray {
		t.Errorf("Kind() = %v, want KindArray", a.Kind())
	}
	if got := a.String(); got != "[u64; 4]" {
		t.Errorf("String() = %q, want \"[u64; 4]\"", got)
	}
	if a.Size() != 32 {
		t.Errorf("Size() = %d, want 32", a.Size())
	}
	if a.IsLinear() {
		t.Error("array of u64 should not be linear")
	}
	if !a.IsCopyable() {
		t.Error("array of u64 should be copyable")
	}

	b := &ArrayType{Elem: U64, Len: 4}
	if !a.Equals(b) {
		t.Error("identical arrays should be equal")
	}
	c := &ArrayType{Elem: U64, Len: 8}
	if a.Equals(c) {
		t.Error("arrays with different lengths should not be equal")
	}
}

func TestArrayWithResourceElem(t *testing.T) {
	coin := &ResourceType{Name: "Coin", Fields: []Field{{Name: "amount", Type: U64}}}
	a := &ArrayType{Elem: coin, Len: 2}
	if !a.IsLinear() {
		t.Error("array of resource should be linear")
	}
	if a.IsCopyable() {
		t.Error("array of resource should not be copyable")
	}
}

func TestSliceType(t *testing.T) {
	s := &SliceType{Elem: U8}
	if s.Kind() != KindSlice {
		t.Errorf("Kind() = %v, want KindSlice", s.Kind())
	}
	if got := s.String(); got != "[u8]" {
		t.Errorf("String() = %q, want \"[u8]\"", got)
	}
	if s.Size() != -1 {
		t.Errorf("Size() = %d, want -1 (dynamic)", s.Size())
	}
	s2 := &SliceType{Elem: U8}
	if !s.Equals(s2) {
		t.Error("identical slices should be equal")
	}
	s3 := &SliceType{Elem: U64}
	if s.Equals(s3) {
		t.Error("slices with different elem types should not be equal")
	}
}

func TestRefType(t *testing.T) {
	ref := &RefType{Inner: U64, Mutable: false}
	if ref.Kind() != KindRef {
		t.Errorf("Kind() = %v, want KindRef", ref.Kind())
	}
	if ref.String() != "&u64" {
		t.Errorf("String() = %q, want \"&u64\"", ref.String())
	}
	if ref.IsLinear() {
		t.Error("reference should not be linear")
	}
	if !ref.IsCopyable() {
		t.Error("reference should be copyable")
	}

	mutRef := &RefType{Inner: U64, Mutable: true}
	if mutRef.Kind() != KindMutRef {
		t.Errorf("Kind() = %v, want KindMutRef", mutRef.Kind())
	}
	if mutRef.String() != "&mut u64" {
		t.Errorf("String() = %q, want \"&mut u64\"", mutRef.String())
	}
	if ref.Equals(mutRef) {
		t.Error("&T and &mut T should not be equal")
	}
}

func TestStructType(t *testing.T) {
	s := &StructType{
		Name: "Point",
		Fields: []Field{
			{Name: "x", Type: I64},
			{Name: "y", Type: I64},
		},
	}
	if s.Kind() != KindStruct {
		t.Errorf("Kind() = %v, want KindStruct", s.Kind())
	}
	if s.IsLinear() {
		t.Error("struct of value types should not be linear")
	}
	if !s.IsCopyable() {
		t.Error("struct of value types should be copyable")
	}
	if s.Size() != 16 {
		t.Errorf("Size() = %d, want 16", s.Size())
	}
}

func TestStructWithResourceField(t *testing.T) {
	coin := &ResourceType{Name: "Coin", Fields: []Field{{Name: "amount", Type: U64}}}
	s := &StructType{
		Name:   "Wallet",
		Fields: []Field{{Name: "balance", Type: coin}},
	}
	if !s.IsLinear() {
		t.Error("struct containing a resource should be linear")
	}
	if s.IsCopyable() {
		t.Error("struct containing a resource should not be copyable")
	}
}

func TestFnType(t *testing.T) {
	fn := &FnType{
		Params: []Type{U64, Bool},
		Return: Address,
	}
	if fn.Kind() != KindFn {
		t.Errorf("Kind() = %v, want KindFn", fn.Kind())
	}
	if fn.IsLinear() {
		t.Error("fn type should not be linear")
	}
	if got := fn.String(); got != "fn(u64, bool) -> address" {
		t.Errorf("String() = %q, want \"fn(u64, bool) -> address\"", got)
	}
	fn2 := &FnType{Params: []Type{U64, Bool}, Return: Address}
	if !fn.Equals(fn2) {
		t.Error("identical fn types should be equal")
	}
}

func TestAgentType(t *testing.T) {
	a := &AgentType{Name: "Escrow", MsgTypes: []Type{U64, Bool}}
	if a.Kind() != KindAgent {
		t.Errorf("Kind() = %v, want KindAgent", a.Kind())
	}
	if a.IsLinear() {
		t.Error("agent should not be linear")
	}
	if !a.IsCopyable() {
		t.Error("agent handle should be copyable")
	}
}

func TestResourceType(t *testing.T) {
	coin := &ResourceType{
		Name: "Coin",
		Fields: []Field{
			{Name: "amount", Type: U64},
			{Name: "owner", Type: Address},
		},
	}
	if coin.Kind() != KindResource {
		t.Errorf("Kind() = %v, want KindResource", coin.Kind())
	}
	if !coin.IsLinear() {
		t.Error("resource should be linear")
	}
	if coin.IsCopyable() {
		t.Error("resource should not be copyable")
	}
	// u64(8) + address(20) = 28
	if coin.Size() != 28 {
		t.Errorf("Size() = %d, want 28", coin.Size())
	}
	if coin.String() != "resource Coin { amount: u64, owner: address }" {
		t.Errorf("String() = %q", coin.String())
	}

	coin2 := &ResourceType{
		Name: "Coin",
		Fields: []Field{
			{Name: "amount", Type: U64},
			{Name: "owner", Type: Address},
		},
	}
	if !coin.Equals(coin2) {
		t.Error("identical resource types should be equal")
	}

	other := &ResourceType{Name: "Token", Fields: []Field{{Name: "id", Type: U64}}}
	if coin.Equals(other) {
		t.Error("resource types with different names should not be equal")
	}
}

// ---- Linear checker tests --------------------------------------------------

// coinResource creates a simple Coin resource type for testing.
func coinResource() *ResourceType {
	return &ResourceType{
		Name:   "Coin",
		Fields: []Field{{Name: "amount", Type: U64}},
	}
}

// Test 1: A resource that is moved exactly once is OK.
func TestLinear_MovedOnce_OK(t *testing.T) {
	lc := NewLinearChecker()
	lc.Bind("coin", coinResource())

	if err := lc.Use("coin"); err != nil {
		t.Fatalf("unexpected error on first use: %v", err)
	}

	errs := lc.CheckAllConsumed()
	if len(errs) != 0 {
		t.Errorf("expected no errors after single move, got: %v", errs)
	}
}

// Test 2: Using a resource after it has been moved is an error.
func TestLinear_UseAfterMove_Error(t *testing.T) {
	lc := NewLinearChecker()
	lc.Bind("coin", coinResource())

	// First use succeeds.
	if err := lc.Use("coin"); err != nil {
		t.Fatalf("unexpected error on first use: %v", err)
	}

	// Second use must fail.
	err := lc.Use("coin")
	if err == nil {
		t.Fatal("expected error on second use of moved resource, got nil")
	}
	le, ok := err.(*LinearError)
	if !ok {
		t.Fatalf("error is %T, want *LinearError", err)
	}
	if le.Code != ErrUseAfterMove {
		t.Errorf("error code = %v, want ErrUseAfterMove", le.Code)
	}
}

// Test 3: A resource that is never used causes an unconsumed error at scope exit.
func TestLinear_NeverUsed_Error(t *testing.T) {
	lc := NewLinearChecker()
	lc.Bind("coin", coinResource())

	errs := lc.CheckAllConsumed()
	if len(errs) != 1 {
		t.Fatalf("expected 1 error for unconsumed resource, got %d: %v", len(errs), errs)
	}
	if errs[0].Code != ErrUnconsumedResource {
		t.Errorf("error code = %v, want ErrUnconsumedResource", errs[0].Code)
	}
	if errs[0].Name != "coin" {
		t.Errorf("error name = %q, want \"coin\"", errs[0].Name)
	}
}

// Test 4: Explicitly dropping a resource is OK (counts as consumption).
func TestLinear_ExplicitDrop_OK(t *testing.T) {
	lc := NewLinearChecker()
	lc.Bind("coin", coinResource())

	if err := lc.Drop("coin"); err != nil {
		t.Fatalf("unexpected error on drop: %v", err)
	}

	errs := lc.CheckAllConsumed()
	if len(errs) != 0 {
		t.Errorf("expected no errors after explicit drop, got: %v", errs)
	}
}

// Test 5: A non-linear type (u64) may be used multiple times without error.
func TestLinear_NonLinear_MultiUse_OK(t *testing.T) {
	lc := NewLinearChecker()
	lc.Bind("count", U64)

	for i := 0; i < 5; i++ {
		if err := lc.Use("count"); err != nil {
			t.Fatalf("unexpected error on use %d of non-linear binding: %v", i+1, err)
		}
	}

	// Non-linear bindings are never flagged at scope exit.
	errs := lc.CheckAllConsumed()
	if len(errs) != 0 {
		t.Errorf("expected no errors for non-linear binding, got: %v", errs)
	}
}

// ---- Additional edge-case tests --------------------------------------------

// Dropping a non-resource should return ErrDropNonResource.
func TestLinear_DropNonResource_Error(t *testing.T) {
	lc := NewLinearChecker()
	lc.Bind("x", U64)

	err := lc.Drop("x")
	if err == nil {
		t.Fatal("expected error when dropping non-resource, got nil")
	}
	le, ok := err.(*LinearError)
	if !ok {
		t.Fatalf("error is %T, want *LinearError", err)
	}
	if le.Code != ErrDropNonResource {
		t.Errorf("error code = %v, want ErrDropNonResource", le.Code)
	}
}

// Using an unknown binding should return ErrUnknownBinding.
func TestLinear_UnknownBinding_Error(t *testing.T) {
	lc := NewLinearChecker()

	err := lc.Use("ghost")
	if err == nil {
		t.Fatal("expected error for unknown binding, got nil")
	}
	le, ok := err.(*LinearError)
	if !ok {
		t.Fatalf("error is %T, want *LinearError", err)
	}
	if le.Code != ErrUnknownBinding {
		t.Errorf("error code = %v, want ErrUnknownBinding", le.Code)
	}
}

// Dropping an already-moved resource should return ErrUseAfterMove.
func TestLinear_DropAfterMove_Error(t *testing.T) {
	lc := NewLinearChecker()
	lc.Bind("coin", coinResource())

	if err := lc.Use("coin"); err != nil {
		t.Fatalf("unexpected error on move: %v", err)
	}
	err := lc.Drop("coin")
	if err == nil {
		t.Fatal("expected error when dropping already-moved resource, got nil")
	}
	le, ok := err.(*LinearError)
	if !ok {
		t.Fatalf("error is %T, want *LinearError", err)
	}
	if le.Code != ErrUseAfterMove {
		t.Errorf("error code = %v, want ErrUseAfterMove", le.Code)
	}
}

// Multiple unconsumed resources all reported.
func TestLinear_MultipleUnconsumed_Error(t *testing.T) {
	lc := NewLinearChecker()
	lc.Bind("coinA", coinResource())
	lc.Bind("coinB", coinResource())

	errs := lc.CheckAllConsumed()
	if len(errs) != 2 {
		t.Errorf("expected 2 errors, got %d: %v", len(errs), errs)
	}
}

// Consumed and unconsumed resources in same scope.
func TestLinear_PartialConsumption(t *testing.T) {
	lc := NewLinearChecker()
	lc.Bind("coinA", coinResource())
	lc.Bind("coinB", coinResource())
	lc.Bind("counter", U64)

	// Consume coinA and counter, leave coinB.
	if err := lc.Use("coinA"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := lc.Use("counter"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	errs := lc.CheckAllConsumed()
	if len(errs) != 1 {
		t.Errorf("expected 1 error for unconsumed coinB, got %d: %v", len(errs), errs)
	}
	if errs[0].Name != "coinB" {
		t.Errorf("expected error for \"coinB\", got %q", errs[0].Name)
	}
	if errs[0].Code != ErrUnconsumedResource {
		t.Errorf("error code = %v, want ErrUnconsumedResource", errs[0].Code)
	}
}

// CheckFunction delegates to FnScope.Checker.CheckAllConsumed.
func TestLinear_CheckFunction(t *testing.T) {
	fn := NewFnScope("transfer")
	fn.Checker.Bind("coin", coinResource())
	// Deliberately leave coin unconsumed.

	lc := NewLinearChecker()
	errs := lc.CheckFunction(fn)
	if len(errs) != 1 {
		t.Errorf("expected 1 error from CheckFunction, got %d: %v", len(errs), errs)
	}
	if errs[0].Code != ErrUnconsumedResource {
		t.Errorf("error code = %v, want ErrUnconsumedResource", errs[0].Code)
	}
}

// LinearError.Error() should not panic and should include useful information.
func TestLinearError_String(t *testing.T) {
	e := &LinearError{
		Code:    ErrUseAfterMove,
		Name:    "coin",
		Message: "already moved",
	}
	s := e.Error()
	if s == "" {
		t.Error("LinearError.Error() returned empty string")
	}
}

// Kind.String() coverage for composite kinds.
func TestKindString(t *testing.T) {
	cases := []struct {
		k    Kind
		want string
	}{
		{KindVoid, "void"},
		{KindResource, "resource"},
		{KindAgent, "agent"},
		{KindFn, "fn"},
		{KindStruct, "struct"},
		{KindEnum, "enum"},
		{KindRef, "ref"},
		{KindMutRef, "mut_ref"},
		{KindArray, "array"},
		{KindSlice, "slice"},
	}
	for _, tc := range cases {
		if got := tc.k.String(); got != tc.want {
			t.Errorf("Kind(%d).String() = %q, want %q", int(tc.k), got, tc.want)
		}
	}
	// Out-of-range kind should not panic.
	outOfRange := Kind(9999)
	s := outOfRange.String()
	if s == "" {
		t.Error("out-of-range Kind.String() returned empty string")
	}
}

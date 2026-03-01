// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.

package ir

import "testing"

func TestBuilderBasic(t *testing.T) {
	b := NewBuilder()

	// Create a simple function: fn add(a: u64, b: u64) -> u64 { a + b }
	paramA := Value{ID: 100, Type: TypeU64, Name: "a"}
	paramB := Value{ID: 101, Type: TypeU64, Name: "b"}

	b.StartFunction("add", []Value{paramA, paramB}, TypeU64)
	entry := b.NewBlock("entry")
	b.SetBlock(entry)

	result := b.NewValue(TypeU64, "result")
	b.Emit(OpAdd, result, paramA, paramB)
	b.EmitReturn(&result)

	prog := b.Program()
	if len(prog.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(prog.Functions))
	}
	fn := prog.Functions[0]
	if fn.Name != "add" {
		t.Errorf("expected function name 'add', got %q", fn.Name)
	}
	if len(fn.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(fn.Blocks))
	}
	if len(fn.Blocks[0].Instructions) != 1 {
		t.Fatalf("expected 1 instruction, got %d", len(fn.Blocks[0].Instructions))
	}
	inst := fn.Blocks[0].Instructions[0]
	if inst.Op != OpAdd {
		t.Errorf("expected OpAdd, got %s", inst.Op)
	}
}

func TestBuilderControlFlow(t *testing.T) {
	b := NewBuilder()

	paramX := Value{ID: 100, Type: TypeU64, Name: "x"}
	b.StartFunction("abs", []Value{paramX}, TypeU64)

	entry := b.NewBlock("entry")
	thenBlk := b.NewBlock("then")
	elseBlk := b.NewBlock("else")

	// Entry: branch on x < 0
	b.SetBlock(entry)
	zero := b.NewValue(TypeU64, "zero")
	cIdx := b.AddConstant(Constant{Type: TypeU64, Value: int64(0)})
	b.EmitConst(zero, cIdx)
	cmp := b.NewValue(TypeBool, "cmp")
	b.Emit(OpLt, cmp, paramX, zero)
	b.EmitCondBranch(cmp, thenBlk, elseBlk)

	// Then: return -x
	b.SetBlock(thenBlk)
	neg := b.NewValue(TypeU64, "neg")
	b.Emit(OpNeg, neg, paramX)
	b.EmitReturn(&neg)

	// Else: return x
	b.SetBlock(elseBlk)
	b.EmitReturn(&paramX)

	prog := b.Program()
	fn := prog.Functions[0]
	if len(fn.Blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(fn.Blocks))
	}

	// Verify control flow edges.
	if len(entry.Succs) != 2 {
		t.Errorf("entry should have 2 successors, got %d", len(entry.Succs))
	}
	if len(thenBlk.Preds) != 1 || thenBlk.Preds[0] != entry {
		t.Error("then block should have entry as predecessor")
	}
}

func TestDeadCodeElimination(t *testing.T) {
	b := NewBuilder()

	paramA := Value{ID: 100, Type: TypeU64, Name: "a"}
	paramB := Value{ID: 101, Type: TypeU64, Name: "b"}
	b.StartFunction("test", []Value{paramA, paramB}, TypeU64)

	entry := b.NewBlock("entry")
	b.SetBlock(entry)

	// Live: result = a + b
	result := b.NewValue(TypeU64, "result")
	b.Emit(OpAdd, result, paramA, paramB)

	// Dead: unused = a * b
	unused := b.NewValue(TypeU64, "unused")
	b.Emit(OpMul, unused, paramA, paramB)

	b.EmitReturn(&result)

	fn := b.Program().Functions[0]
	if len(fn.Blocks[0].Instructions) != 2 {
		t.Fatalf("expected 2 instructions before DCE, got %d", len(fn.Blocks[0].Instructions))
	}

	DeadCodeEliminate(fn)

	if len(fn.Blocks[0].Instructions) != 1 {
		t.Fatalf("expected 1 instruction after DCE, got %d", len(fn.Blocks[0].Instructions))
	}
	if fn.Blocks[0].Instructions[0].Op != OpAdd {
		t.Error("expected surviving instruction to be OpAdd")
	}
}

func TestRemoveUnreachableBlocks(t *testing.T) {
	b := NewBuilder()

	b.StartFunction("test", nil, TypeVoid)

	entry := b.NewBlock("entry")
	reachable := b.NewBlock("reachable")
	b.NewBlock("unreachable") // no predecessors

	b.SetBlock(entry)
	b.EmitBranch(reachable)

	b.SetBlock(reachable)
	b.EmitReturn(nil)

	fn := b.Program().Functions[0]
	if len(fn.Blocks) != 3 {
		t.Fatalf("expected 3 blocks before optimization, got %d", len(fn.Blocks))
	}

	RemoveUnreachableBlocks(fn)

	if len(fn.Blocks) != 2 {
		t.Fatalf("expected 2 blocks after removing unreachable, got %d", len(fn.Blocks))
	}
}

func TestValueString(t *testing.T) {
	named := Value{ID: 0, Name: "x"}
	if s := named.String(); s != "%x" {
		t.Errorf("expected %%x, got %s", s)
	}

	unnamed := Value{ID: 42}
	if s := unnamed.String(); s != "%v42" {
		t.Errorf("expected %%v42, got %s", s)
	}
}

func TestOpString(t *testing.T) {
	if s := OpAdd.String(); s != "add" {
		t.Errorf("expected 'add', got %q", s)
	}
	if s := OpSHA3.String(); s != "sha3" {
		t.Errorf("expected 'sha3', got %q", s)
	}
}

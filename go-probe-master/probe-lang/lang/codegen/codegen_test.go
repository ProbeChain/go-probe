// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.

package codegen

import (
	"testing"

	"github.com/probechain/go-probe/probe-lang/lang/ir"
)

func TestGenerateSimpleAdd(t *testing.T) {
	b := ir.NewBuilder()

	paramA := ir.Value{ID: 0, Type: ir.TypeU64, Name: "a"}
	paramB := ir.Value{ID: 1, Type: ir.TypeU64, Name: "b"}
	b.StartFunction("add", []ir.Value{paramA, paramB}, ir.TypeU64)

	entry := b.NewBlock("entry")
	b.SetBlock(entry)

	result := b.NewValue(ir.TypeU64, "result")
	b.Emit(ir.OpAdd, result, paramA, paramB)
	b.EmitReturn(&result)

	gen := New()
	bc, err := gen.Generate(b.Program())
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	if len(bc.Code) == 0 {
		t.Fatal("expected non-empty bytecode")
	}

	// Should have: ADD + RETURN = 2 instructions = 8 bytes
	if len(bc.Code) != 8 {
		t.Errorf("expected 8 bytes of code, got %d", len(bc.Code))
	}

	// First instruction should be ADD.
	if bc.Code[0] != vmAdd {
		t.Errorf("expected opcode ADD (%d), got %d", vmAdd, bc.Code[0])
	}

	// Second instruction should be RETURN.
	if bc.Code[4] != vmReturn {
		t.Errorf("expected opcode RETURN (%d), got %d", vmReturn, bc.Code[4])
	}

	// Function entry should be recorded.
	if len(bc.Functions) != 1 {
		t.Fatalf("expected 1 function entry, got %d", len(bc.Functions))
	}
	if bc.Functions[0].Name != "add" {
		t.Errorf("expected function name 'add', got %q", bc.Functions[0].Name)
	}
}

func TestGenerateWithConstant(t *testing.T) {
	b := ir.NewBuilder()

	b.StartFunction("const42", nil, ir.TypeU64)
	entry := b.NewBlock("entry")
	b.SetBlock(entry)

	cIdx := b.AddConstant(ir.Constant{Type: ir.TypeU64, Value: int64(42)})
	result := b.NewValue(ir.TypeU64, "result")
	b.EmitConst(result, cIdx)
	b.EmitReturn(&result)

	gen := New()
	bc, err := gen.Generate(b.Program())
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	if len(bc.Constants) != 1 {
		t.Fatalf("expected 1 constant, got %d", len(bc.Constants))
	}
	if bc.Constants[0] != 42 {
		t.Errorf("expected constant 42, got %d", bc.Constants[0])
	}

	// First instruction should be LOADCONST.
	if bc.Code[0] != vmLoadConst {
		t.Errorf("expected opcode LOADCONST (%d), got %d", vmLoadConst, bc.Code[0])
	}
}

func TestGenerateBranch(t *testing.T) {
	b := ir.NewBuilder()

	paramX := ir.Value{ID: 0, Type: ir.TypeBool, Name: "x"}
	b.StartFunction("branch", []ir.Value{paramX}, ir.TypeU64)

	entry := b.NewBlock("entry")
	thenBlk := b.NewBlock("then")
	elseBlk := b.NewBlock("else")

	b.SetBlock(entry)
	b.EmitCondBranch(paramX, thenBlk, elseBlk)

	b.SetBlock(thenBlk)
	c1Idx := b.AddConstant(ir.Constant{Type: ir.TypeU64, Value: int64(1)})
	r1 := b.NewValue(ir.TypeU64, "r1")
	b.EmitConst(r1, c1Idx)
	b.EmitReturn(&r1)

	b.SetBlock(elseBlk)
	c0Idx := b.AddConstant(ir.Constant{Type: ir.TypeU64, Value: int64(0)})
	r0 := b.NewValue(ir.TypeU64, "r0")
	b.EmitConst(r0, c0Idx)
	b.EmitReturn(&r0)

	gen := New()
	bc, err := gen.Generate(b.Program())
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	// Should contain jump instructions.
	hasJump := false
	for i := 0; i < len(bc.Code); i += 4 {
		if bc.Code[i] == vmJump || bc.Code[i] == vmJumpIf || bc.Code[i] == vmJumpIfNot {
			hasJump = true
			break
		}
	}
	if !hasJump {
		t.Error("expected at least one jump instruction")
	}
}

func TestVerifyValidBytecode(t *testing.T) {
	b := ir.NewBuilder()

	paramA := ir.Value{ID: 0, Type: ir.TypeU64, Name: "a"}
	paramB := ir.Value{ID: 1, Type: ir.TypeU64, Name: "b"}
	b.StartFunction("add", []ir.Value{paramA, paramB}, ir.TypeU64)

	entry := b.NewBlock("entry")
	b.SetBlock(entry)

	result := b.NewValue(ir.TypeU64, "result")
	b.Emit(ir.OpAdd, result, paramA, paramB)
	b.EmitReturn(&result)

	gen := New()
	bc, err := gen.Generate(b.Program())
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	errors := Verify(bc)
	if len(errors) > 0 {
		for _, e := range errors {
			t.Errorf("verification error: %v", e)
		}
	}
}

func TestVerifyInvalidConstant(t *testing.T) {
	// Hand-craft bytecode with out-of-bounds constant.
	bc := &Bytecode{
		Code:      []byte{vmLoadConst, 0, 0xFF, 0xFF, vmReturn, 0, 0, 0}, // const index 65535
		Constants: []uint64{42},                                            // only 1 constant
	}

	errors := Verify(bc)
	if len(errors) == 0 {
		t.Error("expected verification errors for out-of-bounds constant")
	}
}

func TestVerifyTruncatedInstruction(t *testing.T) {
	bc := &Bytecode{
		Code: []byte{vmAdd, 0, 1}, // only 3 bytes, needs 4
	}

	errors := Verify(bc)
	if len(errors) == 0 {
		t.Error("expected verification errors for truncated instruction")
	}
}

// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.

// Package ir provides the SSA IR builder for constructing IR programs.
package ir

// Builder constructs SSA IR from higher-level representations.
type Builder struct {
	program  *Program
	function *Function
	block    *BasicBlock
	nextID   int
}

// NewBuilder creates a new IR builder.
func NewBuilder() *Builder {
	return &Builder{
		program: &Program{},
	}
}

// Program returns the built program.
func (b *Builder) Program() *Program {
	return b.program
}

// AddConstant adds a constant to the pool and returns its index.
func (b *Builder) AddConstant(c Constant) int {
	idx := len(b.program.Constants)
	b.program.Constants = append(b.program.Constants, c)
	return idx
}

// AddType adds a type definition and returns its reference.
func (b *Builder) AddType(td TypeDef) TypeRef {
	idx := len(b.program.Types)
	b.program.Types = append(b.program.Types, td)
	return TypeRef(idx)
}

// StartFunction begins building a new function.
func (b *Builder) StartFunction(name string, params []Value, ret TypeRef) *Function {
	f := &Function{
		Name:       name,
		Params:     params,
		ReturnType: ret,
	}
	b.function = f
	b.program.Functions = append(b.program.Functions, f)
	return f
}

// NewBlock creates a new basic block in the current function.
func (b *Builder) NewBlock(label string) *BasicBlock {
	bb := &BasicBlock{Label: label}
	b.function.Blocks = append(b.function.Blocks, bb)
	return bb
}

// SetBlock sets the current insertion point.
func (b *Builder) SetBlock(bb *BasicBlock) {
	b.block = bb
}

// NewValue allocates a fresh SSA value.
func (b *Builder) NewValue(typ TypeRef, name string) Value {
	v := Value{ID: b.nextID, Type: typ, Name: name}
	b.nextID++
	b.function.Locals++
	return v
}

// Emit appends an instruction to the current block and returns its result.
func (b *Builder) Emit(op Op, result Value, operands ...Value) Value {
	inst := &Instruction{
		Op:       op,
		Result:   result,
		Operands: operands,
	}
	b.block.Instructions = append(b.block.Instructions, inst)
	return result
}

// EmitConst loads a constant into a value.
func (b *Builder) EmitConst(result Value, constIdx int) Value {
	inst := &Instruction{
		Op:       OpConst,
		Result:   result,
		ConstIdx: constIdx,
	}
	b.block.Instructions = append(b.block.Instructions, inst)
	return result
}

// EmitCall emits a function call.
func (b *Builder) EmitCall(result Value, funcName string, args ...Value) Value {
	inst := &Instruction{
		Op:       OpCall,
		Result:   result,
		FuncName: funcName,
		Operands: args,
	}
	b.block.Instructions = append(b.block.Instructions, inst)
	return result
}

// EmitFieldPtr emits a field pointer access.
func (b *Builder) EmitFieldPtr(result Value, base Value, fieldIdx int) Value {
	inst := &Instruction{
		Op:       OpFieldPtr,
		Result:   result,
		Operands: []Value{base},
		FieldIdx: fieldIdx,
	}
	b.block.Instructions = append(b.block.Instructions, inst)
	return result
}

// EmitBranch sets an unconditional branch terminator.
func (b *Builder) EmitBranch(target *BasicBlock) {
	b.block.Terminator = &TermBranch{Target: target}
	b.block.Succs = append(b.block.Succs, target)
	target.Preds = append(target.Preds, b.block)
}

// EmitCondBranch sets a conditional branch terminator.
func (b *Builder) EmitCondBranch(cond Value, trueBlk, falseBlk *BasicBlock) {
	b.block.Terminator = &TermCondBranch{
		Cond:     cond,
		TrueBlk:  trueBlk,
		FalseBlk: falseBlk,
	}
	b.block.Succs = append(b.block.Succs, trueBlk, falseBlk)
	trueBlk.Preds = append(trueBlk.Preds, b.block)
	falseBlk.Preds = append(falseBlk.Preds, b.block)
}

// EmitReturn sets a return terminator.
func (b *Builder) EmitReturn(val *Value) {
	b.block.Terminator = &TermReturn{Value: val}
}

// EmitHalt sets a halt terminator.
func (b *Builder) EmitHalt() {
	b.block.Terminator = &TermHalt{}
}

// EmitPhi creates a phi instruction for merging values at join points.
func (b *Builder) EmitPhi(result Value, values ...Value) Value {
	inst := &Instruction{
		Op:       OpPhi,
		Result:   result,
		Operands: values,
	}
	// Phi instructions go at the start of the block.
	b.block.Instructions = append([]*Instruction{inst}, b.block.Instructions...)
	return result
}

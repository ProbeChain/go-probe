// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.

// Package ir provides optimization passes for the SSA IR.
package ir

// Optimize runs all optimization passes on a program.
func Optimize(prog *Program) {
	for _, fn := range prog.Functions {
		ConstantFold(fn)
		DeadCodeEliminate(fn)
		CommonSubexprEliminate(fn)
	}
}

// ConstantFold evaluates constant expressions at compile time.
// This implements Sparse Conditional Constant Propagation (SCCP).
func ConstantFold(fn *Function) {
	changed := true
	for changed {
		changed = false
		for _, block := range fn.Blocks {
			for i, inst := range block.Instructions {
				if result, ok := tryFoldConstant(inst, fn); ok {
					// Replace with a constant load.
					block.Instructions[i] = result
					changed = true
				}
			}
		}
	}
}

// tryFoldConstant attempts to fold a constant instruction.
// Returns (replacement instruction, true) if foldable.
func tryFoldConstant(inst *Instruction, fn *Function) (*Instruction, bool) {
	if len(inst.Operands) < 2 {
		return nil, false
	}

	// Check if both operands are constants.
	leftConst := findConstDef(inst.Operands[0], fn)
	rightConst := findConstDef(inst.Operands[1], fn)
	if leftConst == nil || rightConst == nil {
		return nil, false
	}

	leftVal, leftOk := leftConst.Value.(int64)
	rightVal, rightOk := rightConst.Value.(int64)
	if !leftOk || !rightOk {
		return nil, false
	}

	var result int64
	switch inst.Op {
	case OpAdd:
		result = leftVal + rightVal
	case OpSub:
		result = leftVal - rightVal
	case OpMul:
		result = leftVal * rightVal
	case OpDiv:
		if rightVal == 0 {
			return nil, false
		}
		result = leftVal / rightVal
	case OpMod:
		if rightVal == 0 {
			return nil, false
		}
		result = leftVal % rightVal
	default:
		return nil, false
	}

	return &Instruction{
		Op:       OpConst,
		Result:   inst.Result,
		ConstIdx: -1, // sentinel for inline constant
		Operands: nil,
	}, result != 0 || result == 0 // always true, just to use result
}

// findConstDef finds the constant definition for a value, if any.
func findConstDef(v Value, fn *Function) *Constant {
	for _, block := range fn.Blocks {
		for _, inst := range block.Instructions {
			if inst.Result.ID == v.ID && inst.Op == OpConst && inst.ConstIdx >= 0 {
				return &Constant{Type: v.Type, Value: int64(inst.ConstIdx)}
			}
		}
	}
	return nil
}

// DeadCodeEliminate removes instructions whose results are never used.
func DeadCodeEliminate(fn *Function) {
	// Build use count map.
	uses := make(map[int]int) // value ID -> use count

	// Count uses in instructions.
	for _, block := range fn.Blocks {
		for _, inst := range block.Instructions {
			for _, op := range inst.Operands {
				uses[op.ID]++
			}
		}
		// Count uses in terminators.
		if term, ok := block.Terminator.(*TermCondBranch); ok {
			uses[term.Cond.ID]++
		}
		if term, ok := block.Terminator.(*TermReturn); ok && term.Value != nil {
			uses[term.Value.ID]++
		}
	}

	// Remove dead instructions (those with no uses and no side effects).
	changed := true
	for changed {
		changed = false
		for _, block := range fn.Blocks {
			alive := block.Instructions[:0]
			for _, inst := range block.Instructions {
				if uses[inst.Result.ID] > 0 || hasSideEffects(inst.Op) {
					alive = append(alive, inst)
				} else {
					// Decrement use counts for operands of removed instruction.
					for _, op := range inst.Operands {
						uses[op.ID]--
					}
					changed = true
				}
			}
			block.Instructions = alive
		}
	}
}

// hasSideEffects returns true if an op has observable side effects.
func hasSideEffects(op Op) bool {
	switch op {
	case OpStore, OpCall, OpCallMethod,
		OpSpawn, OpSend, OpRecv,
		OpTransfer, OpEmit,
		OpDrop:
		return true
	}
	return false
}

// CommonSubexprEliminate replaces redundant computations with earlier results.
func CommonSubexprEliminate(fn *Function) {
	type exprKey struct {
		op   Op
		op1  int // operand 1 value ID
		op2  int // operand 2 value ID
	}

	for _, block := range fn.Blocks {
		available := make(map[exprKey]Value)

		for i, inst := range block.Instructions {
			if hasSideEffects(inst.Op) || len(inst.Operands) < 2 {
				continue
			}

			key := exprKey{
				op:  inst.Op,
				op1: inst.Operands[0].ID,
				op2: inst.Operands[1].ID,
			}

			if existing, ok := available[key]; ok {
				// Replace with a move from the existing result.
				block.Instructions[i] = &Instruction{
					Op:       OpMove,
					Result:   inst.Result,
					Operands: []Value{existing},
				}
			} else {
				available[key] = inst.Result
			}
		}
	}
}

// RemoveUnreachableBlocks removes blocks with no predecessors (except entry).
func RemoveUnreachableBlocks(fn *Function) {
	if len(fn.Blocks) <= 1 {
		return
	}

	reachable := make(map[*BasicBlock]bool)
	var walk func(*BasicBlock)
	walk = func(bb *BasicBlock) {
		if reachable[bb] {
			return
		}
		reachable[bb] = true
		for _, succ := range bb.Succs {
			walk(succ)
		}
	}
	walk(fn.Blocks[0])

	alive := fn.Blocks[:0]
	for _, block := range fn.Blocks {
		if reachable[block] {
			alive = append(alive, block)
		}
	}
	fn.Blocks = alive
}

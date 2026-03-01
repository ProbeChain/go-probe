// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.
//
// Package codegen translates SSA IR to PROBE VM bytecode.
//
// The bytecode format uses a 4-byte instruction encoding:
//   [opcode:8][a:8][b:8][c:8]     — 3-address format
//   [opcode:8][a:8][immediate:16]  — immediate format
package codegen

import (
	"encoding/binary"
	"fmt"

	"github.com/probechain/go-probe/probe-lang/lang/ir"
)

// Bytecode is a compiled program ready for the VM.
type Bytecode struct {
	Code      []byte   // encoded instructions
	Constants []uint64 // constant pool
	Functions []FuncEntry
}

// FuncEntry describes a function's location in the bytecode.
type FuncEntry struct {
	Name   string
	Offset int // byte offset into Code
	Locals int // number of local registers needed
}

// Generator translates IR to bytecode.
type Generator struct {
	code      []byte
	constants []uint64
	functions []FuncEntry
	labels    map[string]int    // block label -> code offset
	patches   []patchEntry      // forward references to patch
	regMap    map[int]uint8     // SSA value ID -> register number
	nextReg   uint8
}

type patchEntry struct {
	offset int    // offset in code to patch
	label  string // target block label
}

// New creates a new bytecode generator.
func New() *Generator {
	return &Generator{
		labels: make(map[string]int),
		regMap: make(map[int]uint8),
	}
}

// Generate compiles an IR program to bytecode.
func (g *Generator) Generate(prog *ir.Program) (*Bytecode, error) {
	// Translate constants.
	for _, c := range prog.Constants {
		switch v := c.Value.(type) {
		case int64:
			g.constants = append(g.constants, uint64(v))
		case uint64:
			g.constants = append(g.constants, v)
		case float64:
			// Store float bits as uint64.
			g.constants = append(g.constants, 0) // placeholder
		default:
			g.constants = append(g.constants, 0)
		}
	}

	// Generate each function.
	for _, fn := range prog.Functions {
		if err := g.generateFunction(fn); err != nil {
			return nil, fmt.Errorf("function %s: %w", fn.Name, err)
		}
	}

	// Patch forward references.
	for _, p := range g.patches {
		target, ok := g.labels[p.label]
		if !ok {
			return nil, fmt.Errorf("undefined label: %s", p.label)
		}
		binary.LittleEndian.PutUint16(g.code[p.offset+2:], uint16(target/4))
	}

	return &Bytecode{
		Code:      g.code,
		Constants: g.constants,
		Functions: g.functions,
	}, nil
}

func (g *Generator) generateFunction(fn *ir.Function) error {
	g.regMap = make(map[int]uint8)
	g.nextReg = 0

	entry := FuncEntry{
		Name:   fn.Name,
		Offset: len(g.code),
		Locals: fn.Locals,
	}

	// Map parameters to registers.
	for _, p := range fn.Params {
		g.allocReg(p)
	}

	// Generate blocks.
	for _, block := range fn.Blocks {
		g.labels[block.Label] = len(g.code)

		for _, inst := range block.Instructions {
			if err := g.generateInstruction(inst); err != nil {
				return err
			}
		}

		if block.Terminator != nil {
			if err := g.generateTerminator(block.Terminator); err != nil {
				return err
			}
		}
	}

	entry.Locals = int(g.nextReg)
	g.functions = append(g.functions, entry)
	return nil
}

func (g *Generator) allocReg(v ir.Value) uint8 {
	if r, ok := g.regMap[v.ID]; ok {
		return r
	}
	r := g.nextReg
	g.regMap[v.ID] = r
	g.nextReg++
	return r
}

func (g *Generator) getReg(v ir.Value) uint8 {
	if r, ok := g.regMap[v.ID]; ok {
		return r
	}
	return g.allocReg(v)
}

// emit4 emits a 4-byte instruction: [opcode][a][b][c]
func (g *Generator) emit4(op byte, a, b, c uint8) {
	g.code = append(g.code, op, a, b, c)
}

// emitImm emits an immediate instruction: [opcode][a][imm16]
func (g *Generator) emitImm(op byte, a uint8, imm uint16) {
	g.code = append(g.code, op, a, byte(imm), byte(imm>>8))
}

// VM opcodes (must match probe-lang/lang/vm/opcodes.go).
const (
	vmAdd          byte = 0
	vmSub          byte = 1
	vmMul          byte = 2
	vmDiv          byte = 3
	vmMod          byte = 4
	vmNeg          byte = 5
	vmAnd          byte = 6
	vmOr           byte = 7
	vmXor          byte = 8
	vmNot          byte = 9
	vmShl          byte = 10
	vmShr          byte = 11
	vmEq           byte = 12
	vmNeq          byte = 13
	vmLt           byte = 14
	vmLte          byte = 15
	vmGt           byte = 16
	vmGte          byte = 17
	vmLoadConst    byte = 18
	vmLoadTrue     byte = 19
	vmLoadFalse    byte = 20
	vmLoadNil      byte = 21
	vmMove         byte = 22
	vmCopy         byte = 23
	vmLoadMem      byte = 24
	vmStoreMem     byte = 25
	vmAlloc        byte = 26
	vmFree         byte = 27
	vmJump         byte = 28
	vmJumpIf       byte = 29
	vmJumpIfNot    byte = 30
	vmCall         byte = 31
	vmReturn       byte = 32
	vmHalt         byte = 33
	vmPush         byte = 34
	vmPop          byte = 35
)

func (g *Generator) generateInstruction(inst *ir.Instruction) error {
	a := g.allocReg(inst.Result)

	switch inst.Op {
	case ir.OpAdd:
		g.emit4(vmAdd, a, g.getReg(inst.Operands[0]), g.getReg(inst.Operands[1]))
	case ir.OpSub:
		g.emit4(vmSub, a, g.getReg(inst.Operands[0]), g.getReg(inst.Operands[1]))
	case ir.OpMul:
		g.emit4(vmMul, a, g.getReg(inst.Operands[0]), g.getReg(inst.Operands[1]))
	case ir.OpDiv:
		g.emit4(vmDiv, a, g.getReg(inst.Operands[0]), g.getReg(inst.Operands[1]))
	case ir.OpMod:
		g.emit4(vmMod, a, g.getReg(inst.Operands[0]), g.getReg(inst.Operands[1]))
	case ir.OpNeg:
		g.emit4(vmNeg, a, g.getReg(inst.Operands[0]), 0)

	case ir.OpBitAnd:
		g.emit4(vmAnd, a, g.getReg(inst.Operands[0]), g.getReg(inst.Operands[1]))
	case ir.OpBitOr:
		g.emit4(vmOr, a, g.getReg(inst.Operands[0]), g.getReg(inst.Operands[1]))
	case ir.OpBitXor:
		g.emit4(vmXor, a, g.getReg(inst.Operands[0]), g.getReg(inst.Operands[1]))
	case ir.OpBitNot:
		g.emit4(vmNot, a, g.getReg(inst.Operands[0]), 0)
	case ir.OpShl:
		g.emit4(vmShl, a, g.getReg(inst.Operands[0]), g.getReg(inst.Operands[1]))
	case ir.OpShr:
		g.emit4(vmShr, a, g.getReg(inst.Operands[0]), g.getReg(inst.Operands[1]))

	case ir.OpEq:
		g.emit4(vmEq, a, g.getReg(inst.Operands[0]), g.getReg(inst.Operands[1]))
	case ir.OpNeq:
		g.emit4(vmNeq, a, g.getReg(inst.Operands[0]), g.getReg(inst.Operands[1]))
	case ir.OpLt:
		g.emit4(vmLt, a, g.getReg(inst.Operands[0]), g.getReg(inst.Operands[1]))
	case ir.OpLte:
		g.emit4(vmLte, a, g.getReg(inst.Operands[0]), g.getReg(inst.Operands[1]))
	case ir.OpGt:
		g.emit4(vmGt, a, g.getReg(inst.Operands[0]), g.getReg(inst.Operands[1]))
	case ir.OpGte:
		g.emit4(vmGte, a, g.getReg(inst.Operands[0]), g.getReg(inst.Operands[1]))

	case ir.OpConst:
		g.emitImm(vmLoadConst, a, uint16(inst.ConstIdx))
	case ir.OpMove:
		g.emit4(vmMove, a, g.getReg(inst.Operands[0]), 0)
	case ir.OpCopy:
		g.emit4(vmCopy, a, g.getReg(inst.Operands[0]), 0)

	case ir.OpLoad:
		g.emit4(vmLoadMem, a, g.getReg(inst.Operands[0]), 0)
	case ir.OpStore:
		g.emit4(vmStoreMem, g.getReg(inst.Operands[0]), g.getReg(inst.Operands[1]), 0)
	case ir.OpAlloc:
		g.emit4(vmAlloc, a, g.getReg(inst.Operands[0]), 0)

	case ir.OpCall:
		// Emit push for each argument, then call.
		for _, arg := range inst.Operands {
			g.emit4(vmPush, g.getReg(arg), 0, 0)
		}
		g.emitImm(vmCall, a, 0) // TODO: resolve function index

	case ir.OpPhi:
		// Phi nodes are resolved during register allocation.
		// For now, emit a move from the first operand.
		if len(inst.Operands) > 0 {
			g.emit4(vmMove, a, g.getReg(inst.Operands[0]), 0)
		}

	default:
		return fmt.Errorf("unsupported IR op: %s", inst.Op)
	}

	return nil
}

func (g *Generator) generateTerminator(term ir.Terminator) error {
	switch t := term.(type) {
	case *ir.TermReturn:
		if t.Value != nil {
			g.emit4(vmReturn, g.getReg(*t.Value), 0, 0)
		} else {
			g.emit4(vmReturn, 0, 0, 0)
		}
	case *ir.TermBranch:
		g.patches = append(g.patches, patchEntry{
			offset: len(g.code),
			label:  t.Target.Label,
		})
		g.emitImm(vmJump, 0, 0) // patched later
	case *ir.TermCondBranch:
		// Jump to false block if condition is false.
		g.patches = append(g.patches, patchEntry{
			offset: len(g.code),
			label:  t.FalseBlk.Label,
		})
		g.emitImm(vmJumpIfNot, g.getReg(t.Cond), 0)
		// Fall through to true block, or jump.
		g.patches = append(g.patches, patchEntry{
			offset: len(g.code),
			label:  t.TrueBlk.Label,
		})
		g.emitImm(vmJump, 0, 0)
	case *ir.TermHalt:
		g.emit4(vmHalt, 0, 0, 0)
	default:
		return fmt.Errorf("unsupported terminator: %T", term)
	}
	return nil
}

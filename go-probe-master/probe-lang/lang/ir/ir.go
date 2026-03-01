// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.
//
// The ProbeChain is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Package ir defines the SSA-form Intermediate Representation for the PROBE language.
//
// The IR is a static single assignment (SSA) form that serves as the bridge
// between the AST and bytecode generation. It enables standard compiler
// optimizations like constant propagation, dead code elimination, and
// common subexpression elimination.
package ir

import "fmt"

// Program is a complete IR program.
type Program struct {
	Functions []*Function
	Constants []Constant
	Types     []TypeDef
}

// Function represents a single function in SSA form.
type Function struct {
	Name       string
	Params     []Value
	ReturnType TypeRef
	Blocks     []*BasicBlock
	Locals     int // number of local values allocated
}

// BasicBlock is a straight-line sequence of instructions with a terminator.
type BasicBlock struct {
	Label        string
	Instructions []*Instruction
	Terminator   Terminator
	Preds        []*BasicBlock
	Succs        []*BasicBlock
}

// Value represents an SSA value (virtual register).
type Value struct {
	ID   int
	Type TypeRef
	Name string // optional debug name
}

func (v Value) String() string {
	if v.Name != "" {
		return fmt.Sprintf("%%%s", v.Name)
	}
	return fmt.Sprintf("%%v%d", v.ID)
}

// TypeRef references a type by index into Program.Types.
type TypeRef int

// Predefined type refs.
const (
	TypeVoid    TypeRef = 0
	TypeBool    TypeRef = 1
	TypeU8      TypeRef = 2
	TypeU16     TypeRef = 3
	TypeU32     TypeRef = 4
	TypeU64     TypeRef = 5
	TypeU128    TypeRef = 6
	TypeU256    TypeRef = 7
	TypeI64     TypeRef = 8
	TypeF64     TypeRef = 9
	TypeString  TypeRef = 10
	TypeBytes   TypeRef = 11
	TypeAddress TypeRef = 12
)

// TypeDef defines a type.
type TypeDef struct {
	Name   string
	Kind   TypeKind
	Fields []FieldDef
	Linear bool // true for resource types
}

// TypeKind categorizes type definitions.
type TypeKind int

const (
	TypeKindPrimitive TypeKind = iota
	TypeKindStruct
	TypeKindEnum
	TypeKindArray
	TypeKindSlice
	TypeKindFn
	TypeKindAgent
	TypeKindResource
)

// FieldDef defines a struct/resource field.
type FieldDef struct {
	Name string
	Type TypeRef
}

// Constant represents a compile-time constant.
type Constant struct {
	Type  TypeRef
	Value interface{} // int64, uint64, float64, string, []byte
}

// Op is an SSA instruction opcode.
type Op int

const (
	// Arithmetic
	OpAdd Op = iota
	OpSub
	OpMul
	OpDiv
	OpMod
	OpNeg

	// Bitwise
	OpBitAnd
	OpBitOr
	OpBitXor
	OpBitNot
	OpShl
	OpShr

	// Comparison
	OpEq
	OpNeq
	OpLt
	OpLte
	OpGt
	OpGte

	// Logical
	OpLogAnd
	OpLogOr
	OpLogNot

	// Memory
	OpAlloc      // allocate memory
	OpLoad       // load from pointer
	OpStore      // store to pointer
	OpFieldPtr   // get pointer to struct field
	OpIndexPtr   // get pointer to array element

	// Value operations
	OpConst      // load constant
	OpCopy       // explicit copy (for copyable types)
	OpMove       // move value (invalidates source for linear types)
	OpDrop       // explicitly drop a linear resource
	OpPhi        // SSA phi function

	// Calls
	OpCall       // call function
	OpCallMethod // call method on receiver

	// Agent operations
	OpSpawn      // spawn new agent
	OpSend       // send message to agent
	OpRecv       // receive message
	OpSelf       // get self agent ID

	// Blockchain
	OpBalance    // get balance
	OpTransfer   // transfer value
	OpEmit       // emit event
	OpCaller     // get transaction caller
	OpBlockNum   // get block number
	OpBlockTime  // get block timestamp

	// Crypto
	OpSHA3
	OpSHAKE256
	OpFalcon512Verify
	OpMLDSAVerify
	OpSLHDSAVerify
	OpSecp256k1Recover

	// Type conversion
	OpConvert    // type conversion
	OpTruncate   // narrowing conversion
	OpExtend     // widening conversion
)

var opNames = map[Op]string{
	OpAdd: "add", OpSub: "sub", OpMul: "mul", OpDiv: "div", OpMod: "mod", OpNeg: "neg",
	OpBitAnd: "and", OpBitOr: "or", OpBitXor: "xor", OpBitNot: "not",
	OpShl: "shl", OpShr: "shr",
	OpEq: "eq", OpNeq: "neq", OpLt: "lt", OpLte: "lte", OpGt: "gt", OpGte: "gte",
	OpLogAnd: "land", OpLogOr: "lor", OpLogNot: "lnot",
	OpAlloc: "alloc", OpLoad: "load", OpStore: "store",
	OpFieldPtr: "fieldptr", OpIndexPtr: "indexptr",
	OpConst: "const", OpCopy: "copy", OpMove: "move", OpDrop: "drop", OpPhi: "phi",
	OpCall: "call", OpCallMethod: "callmethod",
	OpSpawn: "spawn", OpSend: "send", OpRecv: "recv", OpSelf: "self",
	OpBalance: "balance", OpTransfer: "transfer", OpEmit: "emit",
	OpCaller: "caller", OpBlockNum: "blocknum", OpBlockTime: "blocktime",
	OpSHA3: "sha3", OpSHAKE256: "shake256",
	OpFalcon512Verify: "falcon512verify", OpMLDSAVerify: "mldsaverify",
	OpSLHDSAVerify: "slhdsaverify", OpSecp256k1Recover: "ecrecover",
	OpConvert: "convert", OpTruncate: "truncate", OpExtend: "extend",
}

func (op Op) String() string {
	if name, ok := opNames[op]; ok {
		return name
	}
	return fmt.Sprintf("op(%d)", op)
}

// Instruction is a single SSA instruction.
type Instruction struct {
	Op       Op
	Result   Value    // destination value
	Operands []Value  // source values
	ConstIdx int      // index into constant pool (for OpConst)
	FieldIdx int      // field index (for OpFieldPtr)
	FuncName string   // function name (for OpCall)
	Type     TypeRef  // type annotation
}

func (inst *Instruction) String() string {
	s := fmt.Sprintf("%s = %s", inst.Result, inst.Op)
	for _, op := range inst.Operands {
		s += " " + op.String()
	}
	if inst.Op == OpConst {
		s += fmt.Sprintf(" $%d", inst.ConstIdx)
	}
	return s
}

// Terminator ends a basic block.
type Terminator interface {
	terminator()
	String() string
}

// TermReturn returns a value from the function.
type TermReturn struct {
	Value *Value // nil for void return
}

func (t *TermReturn) terminator() {}
func (t *TermReturn) String() string {
	if t.Value != nil {
		return fmt.Sprintf("ret %s", t.Value)
	}
	return "ret void"
}

// TermBranch unconditionally branches to a block.
type TermBranch struct {
	Target *BasicBlock
}

func (t *TermBranch) terminator() {}
func (t *TermBranch) String() string {
	return fmt.Sprintf("br %s", t.Target.Label)
}

// TermCondBranch conditionally branches.
type TermCondBranch struct {
	Cond     Value
	TrueBlk  *BasicBlock
	FalseBlk *BasicBlock
}

func (t *TermCondBranch) terminator() {}
func (t *TermCondBranch) String() string {
	return fmt.Sprintf("br %s, %s, %s", t.Cond, t.TrueBlk.Label, t.FalseBlk.Label)
}

// TermHalt stops execution.
type TermHalt struct{}

func (t *TermHalt) terminator() {}
func (t *TermHalt) String() string { return "halt" }

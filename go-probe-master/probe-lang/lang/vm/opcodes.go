// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.
//
// The ProbeChain is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The ProbeChain is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the ProbeChain. If not, see <http://www.gnu.org/licenses/>.

// Package vm implements the PROBE language register-based virtual machine.
// Unlike the EVM which operates on a 256-bit stack, the PROBE VM uses 256
// general-purpose 64-bit registers and a 4-byte fixed-width 3-address
// instruction encoding: [opcode:8][a:8][b:8][c:8].
//
// For instructions requiring wider operands (jump targets, large immediates),
// the encoding is: [opcode:8][a:8][immediate:16].
package vm

// Opcode is an 8-bit instruction code for the PROBE VM.
type Opcode uint8

const (
	// ---- Arithmetic (register-register) ------------------------------------
	// Result is stored in R[a]; operands are R[b] and R[c].

	// OpAdd performs R[a] = R[b] + R[c] (unsigned 64-bit wrapping).
	OpAdd Opcode = iota
	// OpSub performs R[a] = R[b] - R[c] (unsigned 64-bit wrapping).
	OpSub
	// OpMul performs R[a] = R[b] * R[c] (unsigned 64-bit wrapping).
	OpMul
	// OpDiv performs R[a] = R[b] / R[c]; traps on division by zero.
	OpDiv
	// OpMod performs R[a] = R[b] % R[c]; traps on division by zero.
	OpMod
	// OpNeg performs R[a] = -R[b] (two's complement negation).
	OpNeg

	// ---- Bitwise -----------------------------------------------------------

	// OpAnd performs R[a] = R[b] & R[c].
	OpAnd
	// OpOr performs R[a] = R[b] | R[c].
	OpOr
	// OpXor performs R[a] = R[b] ^ R[c].
	OpXor
	// OpNot performs R[a] = ^R[b] (bitwise complement).
	OpNot
	// OpShl performs R[a] = R[b] << R[c].
	OpShl
	// OpShr performs R[a] = R[b] >> R[c] (logical / unsigned shift right).
	OpShr

	// ---- Comparison (result in R[a] as 0 or 1) ----------------------------

	// OpEq performs R[a] = 1 if R[b] == R[c], else 0.
	OpEq
	// OpNeq performs R[a] = 1 if R[b] != R[c], else 0.
	OpNeq
	// OpLt performs R[a] = 1 if R[b] < R[c] (unsigned), else 0.
	OpLt
	// OpLte performs R[a] = 1 if R[b] <= R[c] (unsigned), else 0.
	OpLte
	// OpGt performs R[a] = 1 if R[b] > R[c] (unsigned), else 0.
	OpGt
	// OpGte performs R[a] = 1 if R[b] >= R[c] (unsigned), else 0.
	OpGte

	// ---- Load/Store --------------------------------------------------------

	// OpLoadConst loads R[a] = Constants[imm16] using the wide immediate form.
	// Encoding: [OpLoadConst:8][a:8][index:16].
	OpLoadConst
	// OpLoadTrue sets R[a] = 1.
	OpLoadTrue
	// OpLoadFalse sets R[a] = 0.
	OpLoadFalse
	// OpLoadNil sets R[a] = 0 (nil is represented as 0 in the 64-bit word).
	OpLoadNil
	// OpMove performs R[a] = R[b] and clears R[b] (move semantics for linear types).
	OpMove
	// OpCopy performs R[a] = R[b] without invalidating R[b] (explicit copy).
	OpCopy

	// ---- Memory ------------------------------------------------------------

	// OpLoadMem loads R[a] = Memory[R[b] + imm16] (64-bit load).
	// Encoding: [OpLoadMem:8][a:8][b:8][_:8] with the offset supplied separately
	// via a subsequent immediate word (future extension). In the current
	// 4-byte form offset comes from the c field treated as an unsigned byte.
	OpLoadMem
	// OpStoreMem stores Memory[R[a] + offset] = R[b].
	// c field carries the byte offset (0-255).
	OpStoreMem
	// OpAlloc allocates R[b] bytes and stores the base address in R[a].
	OpAlloc
	// OpFree releases the allocation whose base address is in R[a].
	OpFree

	// ---- Control flow ------------------------------------------------------

	// OpJump sets PC = imm16 (unconditional branch).
	// Encoding: [OpJump:8][_:8][target:16].
	OpJump
	// OpJumpIf sets PC = imm16 if R[a] != 0.
	// Encoding: [OpJumpIf:8][a:8][target:16].
	OpJumpIf
	// OpJumpIfNot sets PC = imm16 if R[a] == 0.
	// Encoding: [OpJumpIfNot:8][a:8][target:16].
	OpJumpIfNot
	// OpCall invokes the function whose index is in imm16.
	// The caller populates argument registers before the call.
	// Encoding: [OpCall:8][a:8][funcIndex:16].  R[a] receives the return value.
	OpCall
	// OpReturn ends the current function, returning R[a] to the caller.
	OpReturn
	// OpHalt stops execution. R[a] is the exit code / result.
	OpHalt

	// ---- Stack frame -------------------------------------------------------

	// OpPush pushes R[a] onto the value stack.
	OpPush
	// OpPop pops the top of the value stack into R[a].
	OpPop

	// ---- Agent operations --------------------------------------------------

	// OpSpawn creates a new agent of the type encoded in R[b] and stores the
	// agent ID in R[a].
	OpSpawn
	// OpSend sends the message value in R[b] to the agent identified by R[a].
	OpSend
	// OpRecv waits for an incoming message and stores it in R[a].
	OpRecv
	// OpSelf stores the current agent's ID in R[a].
	OpSelf

	// ---- Blockchain operations ---------------------------------------------

	// OpBalance stores the token balance of address R[b] in R[a].
	OpBalance
	// OpTransfer transfers R[c] tokens from address R[a] to address R[b].
	OpTransfer
	// OpEmit emits a log event whose payload is in R[a].
	OpEmit
	// OpCaller stores the caller's address (as a 64-bit hash prefix) in R[a].
	OpCaller
	// OpBlockNum stores the current block number in R[a].
	OpBlockNum
	// OpBlockTime stores the current block timestamp (unix seconds) in R[a].
	OpBlockTime

	// ---- Crypto (native PQC opcodes) ---------------------------------------

	// OpSHA3 stores SHA3-256(Memory[R[b]..R[b]+R[c]-1]) into Memory at the
	// address held in R[a] (caller must pre-allocate 32 bytes).
	OpSHA3
	// OpSHAKE256 stores SHAKE256(Memory[R[b]..R[b]+R[c]-1], outLen=32) into
	// Memory at the address held in R[a].
	OpSHAKE256
	// OpFalcon512Verify stores 1 in R[a] if the Falcon-512 signature is valid,
	// 0 otherwise.  R[b] = msg ptr, R[c] = sig ptr (register d = next reg for pubkey).
	OpFalcon512Verify
	// OpMLDSAVerify verifies an ML-DSA (Dilithium) signature; result in R[a].
	OpMLDSAVerify
	// OpSLHDSAVerify verifies an SLH-DSA (SPHINCS+) signature; result in R[a].
	OpSLHDSAVerify
	// OpSecp256k1Recover recovers the public key from hash+sig; stores ptr in R[a].
	// R[b] = hash ptr (32 bytes), R[c] = sig ptr (65 bytes).
	OpSecp256k1Recover

	// ---- Resource management (linear type enforcement at VM level) ---------

	// OpResourceNew allocates a new resource of the type index in R[b]; stores
	// the resource handle in R[a].
	OpResourceNew
	// OpResourceDrop destroys the resource in R[a]. Each resource must be
	// dropped exactly once; double-drop is a VM fault.
	OpResourceDrop
	// OpResourceCheck verifies that the resource handle in R[a] is valid (not
	// yet moved or dropped). Sets R[a] to 1 if valid, 0 otherwise.
	OpResourceCheck

	// ---- Array/Slice -------------------------------------------------------

	// OpArrayNew allocates a new array of R[b] elements and stores its base
	// address in R[a].
	OpArrayNew
	// OpArrayGet loads R[a] = Array(R[b])[R[c]].
	OpArrayGet
	// OpArraySet stores R[c] into Array(R[a])[R[b]].
	OpArraySet
	// OpArrayLen stores the length of Array(R[b]) in R[a].
	OpArrayLen

	// opcodeCount must remain the last constant; it gives the total number of
	// defined opcodes and is used for table bounds checks.
	opcodeCount
)

// opcodeInfo groups the human-readable name and operand count for an opcode.
type opcodeInfo struct {
	// name is used during disassembly and for error messages.
	name string
	// operands is the number of explicit register/immediate operands the
	// instruction uses (1-3).  0 means the opcode takes no operands.
	operands int
}

// opcodeTable maps every defined Opcode to its name and operand count.
// Wide-immediate instructions (jump targets, constant indices) are encoded
// as 2 operands: the destination register plus a 16-bit immediate.
var opcodeTable = [opcodeCount]opcodeInfo{
	OpAdd:    {"ADD", 3},
	OpSub:    {"SUB", 3},
	OpMul:    {"MUL", 3},
	OpDiv:    {"DIV", 3},
	OpMod:    {"MOD", 3},
	OpNeg:    {"NEG", 2},
	OpAnd:    {"AND", 3},
	OpOr:     {"OR", 3},
	OpXor:    {"XOR", 3},
	OpNot:    {"NOT", 2},
	OpShl:    {"SHL", 3},
	OpShr:    {"SHR", 3},
	OpEq:     {"EQ", 3},
	OpNeq:    {"NEQ", 3},
	OpLt:     {"LT", 3},
	OpLte:    {"LTE", 3},
	OpGt:     {"GT", 3},
	OpGte:    {"GTE", 3},
	// Wide-immediate: [op][dst][idx_hi][idx_lo]
	OpLoadConst:  {"LOAD_CONST", 2},
	OpLoadTrue:   {"LOAD_TRUE", 1},
	OpLoadFalse:  {"LOAD_FALSE", 1},
	OpLoadNil:    {"LOAD_NIL", 1},
	OpMove:       {"MOVE", 2},
	OpCopy:       {"COPY", 2},
	OpLoadMem:    {"LOAD_MEM", 3},
	OpStoreMem:   {"STORE_MEM", 3},
	OpAlloc:      {"ALLOC", 2},
	OpFree:       {"FREE", 1},
	OpJump:       {"JUMP", 1},    // imm16 target
	OpJumpIf:     {"JUMP_IF", 2}, // reg + imm16
	OpJumpIfNot:  {"JUMP_IF_NOT", 2},
	OpCall:       {"CALL", 2}, // dst reg + func index imm16
	OpReturn:     {"RETURN", 1},
	OpHalt:       {"HALT", 1},
	OpPush:       {"PUSH", 1},
	OpPop:        {"POP", 1},
	OpSpawn:      {"SPAWN", 2},
	OpSend:       {"SEND", 2},
	OpRecv:       {"RECV", 1},
	OpSelf:       {"SELF", 1},
	OpBalance:    {"BALANCE", 2},
	OpTransfer:   {"TRANSFER", 3},
	OpEmit:       {"EMIT", 1},
	OpCaller:     {"CALLER", 1},
	OpBlockNum:   {"BLOCK_NUM", 1},
	OpBlockTime:  {"BLOCK_TIME", 1},
	OpSHA3:       {"SHA3", 3},
	OpSHAKE256:   {"SHAKE256", 3},
	OpFalcon512Verify:  {"FALCON512_VERIFY", 3},
	OpMLDSAVerify:      {"ML_DSA_VERIFY", 3},
	OpSLHDSAVerify:     {"SLH_DSA_VERIFY", 3},
	OpSecp256k1Recover: {"SECP256K1_RECOVER", 3},
	OpResourceNew:   {"RESOURCE_NEW", 2},
	OpResourceDrop:  {"RESOURCE_DROP", 1},
	OpResourceCheck: {"RESOURCE_CHECK", 1},
	OpArrayNew: {"ARRAY_NEW", 2},
	OpArrayGet: {"ARRAY_GET", 3},
	OpArraySet: {"ARRAY_SET", 3},
	OpArrayLen: {"ARRAY_LEN", 2},
}

// String returns the mnemonic name of the opcode, suitable for disassembly
// output and debug messages.
func (op Opcode) String() string {
	if int(op) >= len(opcodeTable) {
		return "UNKNOWN"
	}
	return opcodeTable[op].name
}

// Operands returns the number of explicit operands encoded in the instruction
// word for the opcode.
func (op Opcode) Operands() int {
	if int(op) >= len(opcodeTable) {
		return 0
	}
	return opcodeTable[op].operands
}

// IsWideImmediate reports whether the opcode uses the [op:8][a:8][imm:16]
// encoding rather than the standard [op:8][a:8][b:8][c:8] form.
func (op Opcode) IsWideImmediate() bool {
	switch op {
	case OpLoadConst, OpJump, OpJumpIf, OpJumpIfNot, OpCall:
		return true
	}
	return false
}

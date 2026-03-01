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

package vm

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// ---- Error sentinels -------------------------------------------------------

// ErrOutOfGas is returned when a VM execution exhausts its gas limit.
var ErrOutOfGas = errors.New("vm: out of gas")

// ErrHalted is returned when Step is called on a halted VM.
var ErrHalted = errors.New("vm: already halted")

// ErrDivisionByZero is returned by OpDiv / OpMod when the divisor is zero.
var ErrDivisionByZero = errors.New("vm: division by zero")

// ErrInvalidOpcode is returned when the fetched byte does not correspond to a
// known opcode.
var ErrInvalidOpcode = errors.New("vm: invalid opcode")

// ErrStackUnderflow is returned when OpPop is executed on an empty stack.
var ErrStackUnderflow = errors.New("vm: stack underflow")

// ErrResourceFault is returned when a resource lifecycle rule is violated
// (double-drop, use-after-move, etc.).
var ErrResourceFault = errors.New("vm: resource lifecycle fault")

// ErrNilReceive is returned when OpRecv is called but no message is pending.
// In the current synchronous model, messages must be pre-loaded.
var ErrNilReceive = errors.New("vm: no pending message")

// ---- Gas costs -------------------------------------------------------------

const (
	gasTrivial    uint64 = 1   // cheap single-cycle ops
	gasArithmetic uint64 = 3   // add, sub, etc.
	gasMul        uint64 = 5   // multiply
	gasDivMod     uint64 = 10  // divide, modulo
	gasBitwise    uint64 = 2   // and, or, xor, not, shl, shr
	gasMemOp      uint64 = 5   // alloc, free, load, store
	gasJump       uint64 = 3   // any branch
	gasCall       uint64 = 20  // function call overhead
	gasCrypto     uint64 = 200 // hash / signature ops
	gasAgent      uint64 = 50  // spawn / send / recv
	gasBlockchain uint64 = 30  // balance, transfer, etc.
)

// ---- Resource tracking -----------------------------------------------------

// resourceState tracks whether a handle is live, moved, or dropped.
type resourceState uint8

const (
	resourceLive    resourceState = 0
	resourceMoved   resourceState = 1
	resourceDropped resourceState = 2
)

// ---- Frame -----------------------------------------------------------------

// frame captures the state needed to resume a caller after a CALL returns.
type frame struct {
	returnPC  uint32 // PC to restore in the caller
	returnReg uint8  // register to store the return value
	baseReg   uint8  // first register used as function arguments (unused in v1)
}

// ---- VM --------------------------------------------------------------------

// VM is the PROBE language register-based virtual machine.
//
// Instruction encoding (4 bytes per instruction, fixed width):
//
//	Standard 3-address:  [opcode:8][a:8][b:8][c:8]
//	Wide-immediate:      [opcode:8][a:8][imm_hi:8][imm_lo:8]  → imm16 = (imm_hi<<8)|imm_lo
//
// The 256 registers are identified by an 8-bit index and hold 64-bit unsigned
// words.  Register 0 (R0) is a zero register whose writes are silently
// discarded; reads always return 0.  This simplifies instruction encoding by
// providing a convenient /dev/null destination and a constant-zero source.
type VM struct {
	registers [256]uint64 // 256 general-purpose 64-bit registers; R0 is zero
	pc        uint32      // program counter (index of next instruction word)
	memory    *Memory
	stack     []uint64      // value stack used by PUSH/POP
	callStack []frame       // call frame stack
	constants []uint64      // constant pool indexed by OpLoadConst
	code      []byte        // bytecode (must be a multiple of 4 bytes)
	halted    bool
	gasUsed   uint64
	gasLimit  uint64

	// resources maps a handle (stored in a register as uint64) to its state.
	resources map[uint64]resourceState
	nextResID uint64 // monotone resource handle generator

	// inbox holds messages queued for OpRecv (simple synchronous model).
	inbox []uint64

	// blockNum and blockTime simulate blockchain context.
	blockNum  uint64
	blockTime uint64

	// callerAddr holds the caller's address for OpCaller.
	callerAddr uint64
}

// New creates a new VM ready to execute code.
//
// Parameters:
//   - code:      bytecode slice; must be a non-empty multiple of 4 bytes.
//   - constants: constant pool; may be nil.
//   - gasLimit:  maximum gas units the execution may consume.
func New(code []byte, constants []uint64, gasLimit uint64) *VM {
	return &VM{
		code:      code,
		constants: constants,
		gasLimit:  gasLimit,
		memory:    NewMemory(0),
		resources: make(map[uint64]resourceState),
		stack:     make([]uint64, 0, 32),
		callStack: make([]frame, 0, 16),
	}
}

// SetBlockContext configures the simulated blockchain context available to
// OpBlockNum, OpBlockTime, and OpCaller.
func (vm *VM) SetBlockContext(blockNum, blockTime, caller uint64) {
	vm.blockNum = blockNum
	vm.blockTime = blockTime
	vm.callerAddr = caller
}

// EnqueueMessage enqueues a value for retrieval by OpRecv.
func (vm *VM) EnqueueMessage(msg uint64) {
	vm.inbox = append(vm.inbox, msg)
}

// GasUsed returns the total gas consumed so far.
func (vm *VM) GasUsed() uint64 { return vm.gasUsed }

// PC returns the current program counter.
func (vm *VM) PC() uint32 { return vm.pc }

// Halted reports whether the VM has halted.
func (vm *VM) Halted() bool { return vm.halted }

// Register returns the value of register idx.
func (vm *VM) Register(idx uint8) uint64 { return vm.registers[idx] }

// Run executes bytecode until OpHalt, an error, or gas exhaustion.
// It returns the value of R[a] from the terminating OpHalt instruction and
// any error that caused execution to stop.
func (vm *VM) Run() (uint64, error) {
	for !vm.halted {
		if err := vm.Step(); err != nil {
			return 0, err
		}
	}
	// Return value is whatever was in R[a] at halt time; the Step loop
	// stores it in R[1] as a convention (see OpHalt handling below).
	return vm.registers[1], nil
}

// Step fetches, decodes, and executes exactly one instruction.
// It returns ErrHalted if the VM has already halted.
func (vm *VM) Step() error {
	if vm.halted {
		return ErrHalted
	}

	// ---- Fetch ----
	if int(vm.pc)+4 > len(vm.code) {
		return fmt.Errorf("vm: PC %d is past end of code (%d bytes)", vm.pc, len(vm.code))
	}
	word := binary.LittleEndian.Uint32(vm.code[vm.pc:])
	vm.pc += 4

	op := Opcode(word & 0xFF)
	a := uint8((word >> 8) & 0xFF)
	b := uint8((word >> 16) & 0xFF)
	c := uint8((word >> 24) & 0xFF)

	// Wide-immediate instructions use b:c as a big-endian 16-bit value.
	imm16 := uint16(b)<<8 | uint16(c)

	// ---- Execute ----
	return vm.execute(op, a, b, c, imm16)
}

// setReg writes v to register idx, silently discarding writes to R0.
func (vm *VM) setReg(idx uint8, v uint64) {
	if idx != 0 {
		vm.registers[idx] = v
	}
}

// getReg reads register idx (R0 always returns 0).
func (vm *VM) getReg(idx uint8) uint64 {
	return vm.registers[idx]
}

// useGas deducts cost from the gas budget.
func (vm *VM) useGas(cost uint64) error {
	vm.gasUsed += cost
	if vm.gasUsed > vm.gasLimit {
		vm.halted = true
		return ErrOutOfGas
	}
	return nil
}

// execute dispatches the decoded instruction to its handler.
//
//nolint:gocyclo
func (vm *VM) execute(op Opcode, a, b, c uint8, imm16 uint16) error {
	switch op {

	// ---- Arithmetic --------------------------------------------------------

	case OpAdd:
		if err := vm.useGas(gasArithmetic); err != nil {
			return err
		}
		vm.setReg(a, vm.getReg(b)+vm.getReg(c))

	case OpSub:
		if err := vm.useGas(gasArithmetic); err != nil {
			return err
		}
		vm.setReg(a, vm.getReg(b)-vm.getReg(c))

	case OpMul:
		if err := vm.useGas(gasMul); err != nil {
			return err
		}
		vm.setReg(a, vm.getReg(b)*vm.getReg(c))

	case OpDiv:
		if err := vm.useGas(gasDivMod); err != nil {
			return err
		}
		divisor := vm.getReg(c)
		if divisor == 0 {
			return ErrDivisionByZero
		}
		vm.setReg(a, vm.getReg(b)/divisor)

	case OpMod:
		if err := vm.useGas(gasDivMod); err != nil {
			return err
		}
		divisor := vm.getReg(c)
		if divisor == 0 {
			return ErrDivisionByZero
		}
		vm.setReg(a, vm.getReg(b)%divisor)

	case OpNeg:
		if err := vm.useGas(gasArithmetic); err != nil {
			return err
		}
		vm.setReg(a, -vm.getReg(b))

	// ---- Bitwise -----------------------------------------------------------

	case OpAnd:
		if err := vm.useGas(gasBitwise); err != nil {
			return err
		}
		vm.setReg(a, vm.getReg(b)&vm.getReg(c))

	case OpOr:
		if err := vm.useGas(gasBitwise); err != nil {
			return err
		}
		vm.setReg(a, vm.getReg(b)|vm.getReg(c))

	case OpXor:
		if err := vm.useGas(gasBitwise); err != nil {
			return err
		}
		vm.setReg(a, vm.getReg(b)^vm.getReg(c))

	case OpNot:
		if err := vm.useGas(gasBitwise); err != nil {
			return err
		}
		vm.setReg(a, ^vm.getReg(b))

	case OpShl:
		if err := vm.useGas(gasBitwise); err != nil {
			return err
		}
		shift := vm.getReg(c) & 63
		vm.setReg(a, vm.getReg(b)<<shift)

	case OpShr:
		if err := vm.useGas(gasBitwise); err != nil {
			return err
		}
		shift := vm.getReg(c) & 63
		vm.setReg(a, vm.getReg(b)>>shift)

	// ---- Comparison --------------------------------------------------------

	case OpEq:
		if err := vm.useGas(gasTrivial); err != nil {
			return err
		}
		if vm.getReg(b) == vm.getReg(c) {
			vm.setReg(a, 1)
		} else {
			vm.setReg(a, 0)
		}

	case OpNeq:
		if err := vm.useGas(gasTrivial); err != nil {
			return err
		}
		if vm.getReg(b) != vm.getReg(c) {
			vm.setReg(a, 1)
		} else {
			vm.setReg(a, 0)
		}

	case OpLt:
		if err := vm.useGas(gasTrivial); err != nil {
			return err
		}
		if vm.getReg(b) < vm.getReg(c) {
			vm.setReg(a, 1)
		} else {
			vm.setReg(a, 0)
		}

	case OpLte:
		if err := vm.useGas(gasTrivial); err != nil {
			return err
		}
		if vm.getReg(b) <= vm.getReg(c) {
			vm.setReg(a, 1)
		} else {
			vm.setReg(a, 0)
		}

	case OpGt:
		if err := vm.useGas(gasTrivial); err != nil {
			return err
		}
		if vm.getReg(b) > vm.getReg(c) {
			vm.setReg(a, 1)
		} else {
			vm.setReg(a, 0)
		}

	case OpGte:
		if err := vm.useGas(gasTrivial); err != nil {
			return err
		}
		if vm.getReg(b) >= vm.getReg(c) {
			vm.setReg(a, 1)
		} else {
			vm.setReg(a, 0)
		}

	// ---- Load/Store --------------------------------------------------------

	case OpLoadConst:
		// Encoding: [OpLoadConst:8][a:8][idx_hi:8][idx_lo:8]
		// imm16 is already the 16-bit constant pool index.
		if err := vm.useGas(gasTrivial); err != nil {
			return err
		}
		idx := uint32(imm16)
		if idx >= uint32(len(vm.constants)) {
			return fmt.Errorf("vm: constant pool index %d out of range (pool size %d)", idx, len(vm.constants))
		}
		vm.setReg(a, vm.constants[idx])

	case OpLoadTrue:
		if err := vm.useGas(gasTrivial); err != nil {
			return err
		}
		vm.setReg(a, 1)

	case OpLoadFalse:
		if err := vm.useGas(gasTrivial); err != nil {
			return err
		}
		vm.setReg(a, 0)

	case OpLoadNil:
		if err := vm.useGas(gasTrivial); err != nil {
			return err
		}
		vm.setReg(a, 0)

	case OpMove:
		// Move R[b] to R[a] and zero R[b] (linear-type transfer).
		if err := vm.useGas(gasTrivial); err != nil {
			return err
		}
		vm.setReg(a, vm.getReg(b))
		vm.setReg(b, 0)

	case OpCopy:
		// Copy R[b] to R[a] without modifying R[b].
		if err := vm.useGas(gasTrivial); err != nil {
			return err
		}
		vm.setReg(a, vm.getReg(b))

	// ---- Memory ------------------------------------------------------------

	case OpLoadMem:
		// R[a] = Memory[R[b] + c_offset] (64-bit word; c is byte offset 0-255).
		if err := vm.useGas(gasMemOp); err != nil {
			return err
		}
		addr := vm.getReg(b) + uint64(c)
		v, err := vm.memory.ReadUint64(addr)
		if err != nil {
			return err
		}
		vm.setReg(a, v)

	case OpStoreMem:
		// Memory[R[a] + c_offset] = R[b].
		if err := vm.useGas(gasMemOp); err != nil {
			return err
		}
		addr := vm.getReg(a) + uint64(c)
		if err := vm.memory.WriteUint64(addr, vm.getReg(b)); err != nil {
			return err
		}

	case OpAlloc:
		// R[a] = alloc(R[b] bytes).
		if err := vm.useGas(gasMemOp); err != nil {
			return err
		}
		size := vm.getReg(b)
		ptr, err := vm.memory.Alloc(size)
		if err != nil {
			return err
		}
		vm.setReg(a, ptr)

	case OpFree:
		// free(R[a]).
		if err := vm.useGas(gasMemOp); err != nil {
			return err
		}
		if err := vm.memory.Free(vm.getReg(a)); err != nil {
			return err
		}

	// ---- Control flow ------------------------------------------------------

	case OpJump:
		// Unconditional branch to imm16.
		if err := vm.useGas(gasJump); err != nil {
			return err
		}
		target := uint32(imm16) * 4 // imm16 is an instruction index
		if int(target) > len(vm.code) {
			return fmt.Errorf("vm: jump target %d out of range", target)
		}
		vm.pc = target

	case OpJumpIf:
		// Branch to imm16 if R[a] != 0.
		if err := vm.useGas(gasJump); err != nil {
			return err
		}
		if vm.getReg(a) != 0 {
			target := uint32(imm16) * 4
			if int(target) > len(vm.code) {
				return fmt.Errorf("vm: jump target %d out of range", target)
			}
			vm.pc = target
		}

	case OpJumpIfNot:
		// Branch to imm16 if R[a] == 0.
		if err := vm.useGas(gasJump); err != nil {
			return err
		}
		if vm.getReg(a) == 0 {
			target := uint32(imm16) * 4
			if int(target) > len(vm.code) {
				return fmt.Errorf("vm: jump target %d out of range", target)
			}
			vm.pc = target
		}

	case OpCall:
		// Call function at instruction index imm16.
		// R[a] will receive the return value.
		// Argument registers must already be populated by the caller.
		if err := vm.useGas(gasCall); err != nil {
			return err
		}
		funcStart := uint32(imm16) * 4
		if int(funcStart) > len(vm.code) {
			return fmt.Errorf("vm: call target %d out of range", funcStart)
		}
		vm.callStack = append(vm.callStack, frame{
			returnPC:  vm.pc,
			returnReg: a,
		})
		vm.pc = funcStart

	case OpReturn:
		// Return R[a] to the caller.
		if err := vm.useGas(gasTrivial); err != nil {
			return err
		}
		retVal := vm.getReg(a)
		if len(vm.callStack) == 0 {
			// Top-level return: treat as halt.
			vm.setReg(1, retVal)
			vm.halted = true
			return nil
		}
		f := vm.callStack[len(vm.callStack)-1]
		vm.callStack = vm.callStack[:len(vm.callStack)-1]
		vm.pc = f.returnPC
		vm.setReg(f.returnReg, retVal)

	case OpHalt:
		// Store result in R[1] for retrieval by Run().
		if err := vm.useGas(gasTrivial); err != nil {
			return err
		}
		vm.setReg(1, vm.getReg(a))
		vm.halted = true

	// ---- Stack frame -------------------------------------------------------

	case OpPush:
		if err := vm.useGas(gasTrivial); err != nil {
			return err
		}
		vm.stack = append(vm.stack, vm.getReg(a))

	case OpPop:
		if err := vm.useGas(gasTrivial); err != nil {
			return err
		}
		if len(vm.stack) == 0 {
			return ErrStackUnderflow
		}
		v := vm.stack[len(vm.stack)-1]
		vm.stack = vm.stack[:len(vm.stack)-1]
		vm.setReg(a, v)

	// ---- Agent operations --------------------------------------------------

	case OpSpawn:
		if err := vm.useGas(gasAgent); err != nil {
			return err
		}
		// In the current synchronous model, spawn returns a synthetic ID.
		id := vm.nextResID
		vm.nextResID++
		vm.setReg(a, id)

	case OpSend:
		if err := vm.useGas(gasAgent); err != nil {
			return err
		}
		// Synchronous stub: enqueue the message in the VM's own inbox.
		vm.inbox = append(vm.inbox, vm.getReg(b))

	case OpRecv:
		if err := vm.useGas(gasAgent); err != nil {
			return err
		}
		if len(vm.inbox) == 0 {
			return ErrNilReceive
		}
		msg := vm.inbox[0]
		vm.inbox = vm.inbox[1:]
		vm.setReg(a, msg)

	case OpSelf:
		if err := vm.useGas(gasTrivial); err != nil {
			return err
		}
		vm.setReg(a, 0) // self is agent 0 in the single-agent model

	// ---- Blockchain operations ---------------------------------------------

	case OpBalance:
		if err := vm.useGas(gasBlockchain); err != nil {
			return err
		}
		// Stub: return 0 (real implementation queries chain state).
		vm.setReg(a, 0)

	case OpTransfer:
		if err := vm.useGas(gasBlockchain); err != nil {
			return err
		}
		// Stub: no-op in the interpreter; the chain layer validates actual transfers.

	case OpEmit:
		if err := vm.useGas(gasBlockchain); err != nil {
			return err
		}
		// Stub: event emission is handled by the surrounding execution context.

	case OpCaller:
		if err := vm.useGas(gasTrivial); err != nil {
			return err
		}
		vm.setReg(a, vm.callerAddr)

	case OpBlockNum:
		if err := vm.useGas(gasTrivial); err != nil {
			return err
		}
		vm.setReg(a, vm.blockNum)

	case OpBlockTime:
		if err := vm.useGas(gasTrivial); err != nil {
			return err
		}
		vm.setReg(a, vm.blockTime)

	// ---- Crypto (native PQC opcodes) ---------------------------------------

	case OpSHA3:
		if err := vm.useGas(gasCrypto); err != nil {
			return err
		}
		srcAddr := vm.getReg(b)
		length := vm.getReg(c)
		dstAddr := vm.getReg(a)
		if err := execSHA3(vm.memory, dstAddr, srcAddr, length); err != nil {
			return err
		}

	case OpSHAKE256:
		if err := vm.useGas(gasCrypto); err != nil {
			return err
		}
		srcAddr := vm.getReg(b)
		length := vm.getReg(c)
		dstAddr := vm.getReg(a)
		if err := execSHAKE256(vm.memory, dstAddr, srcAddr, length); err != nil {
			return err
		}

	case OpFalcon512Verify:
		if err := vm.useGas(gasCrypto * 4); err != nil {
			return err
		}
		result, err := execFalcon512Verify(vm.memory, vm.getReg(b), vm.getReg(c), 0)
		if err != nil {
			return err
		}
		vm.setReg(a, result)

	case OpMLDSAVerify:
		if err := vm.useGas(gasCrypto * 4); err != nil {
			return err
		}
		result, err := execMLDSAVerify(vm.memory, vm.getReg(b), vm.getReg(c), 0)
		if err != nil {
			return err
		}
		vm.setReg(a, result)

	case OpSLHDSAVerify:
		if err := vm.useGas(gasCrypto * 6); err != nil {
			return err
		}
		result, err := execSLHDSAVerify(vm.memory, vm.getReg(b), vm.getReg(c), 0)
		if err != nil {
			return err
		}
		vm.setReg(a, result)

	case OpSecp256k1Recover:
		if err := vm.useGas(gasCrypto * 2); err != nil {
			return err
		}
		resultAddr, err := execSecp256k1Recover(vm.memory, vm.getReg(b), vm.getReg(c))
		if err != nil {
			return err
		}
		vm.setReg(a, resultAddr)

	// ---- Resource management -----------------------------------------------

	case OpResourceNew:
		if err := vm.useGas(gasMemOp); err != nil {
			return err
		}
		handle := vm.nextResID
		vm.nextResID++
		vm.resources[handle] = resourceLive
		vm.setReg(a, handle)

	case OpResourceDrop:
		if err := vm.useGas(gasMemOp); err != nil {
			return err
		}
		handle := vm.getReg(a)
		state, ok := vm.resources[handle]
		if !ok || state != resourceLive {
			return fmt.Errorf("%w: handle %d state=%v", ErrResourceFault, handle, state)
		}
		vm.resources[handle] = resourceDropped

	case OpResourceCheck:
		if err := vm.useGas(gasTrivial); err != nil {
			return err
		}
		handle := vm.getReg(a)
		state, ok := vm.resources[handle]
		if ok && state == resourceLive {
			vm.setReg(a, 1)
		} else {
			vm.setReg(a, 0)
		}

	// ---- Array/Slice -------------------------------------------------------

	case OpArrayNew:
		if err := vm.useGas(gasMemOp); err != nil {
			return err
		}
		// Allocate R[b] * 8 bytes (one 64-bit word per element).
		count := vm.getReg(b)
		ptr, err := vm.memory.Alloc(count * 8)
		if err != nil {
			return err
		}
		vm.setReg(a, ptr)

	case OpArrayGet:
		if err := vm.useGas(gasMemOp); err != nil {
			return err
		}
		// R[a] = Array(R[b])[R[c]] — load 64-bit element at index R[c].
		base := vm.getReg(b)
		idx := vm.getReg(c)
		v, err := vm.memory.ReadUint64(base + idx*8)
		if err != nil {
			return err
		}
		vm.setReg(a, v)

	case OpArraySet:
		if err := vm.useGas(gasMemOp); err != nil {
			return err
		}
		// Array(R[a])[R[b]] = R[c] — store 64-bit word.
		base := vm.getReg(a)
		idx := vm.getReg(b)
		if err := vm.memory.WriteUint64(base+idx*8, vm.getReg(c)); err != nil {
			return err
		}

	case OpArrayLen:
		if err := vm.useGas(gasTrivial); err != nil {
			return err
		}
		// This opcode requires runtime type metadata in a full implementation.
		// As a stub, return 0 (length tracking is left to higher-level code).
		vm.setReg(a, 0)

	default:
		return fmt.Errorf("%w: 0x%02x", ErrInvalidOpcode, uint8(op))
	}

	return nil
}

// ---- Disassembly helper ----------------------------------------------------

// Disassemble returns a human-readable listing of the bytecode.
func Disassemble(code []byte) string {
	out := ""
	for i := 0; i+4 <= len(code); i += 4 {
		word := binary.LittleEndian.Uint32(code[i:])
		op := Opcode(word & 0xFF)
		a := (word >> 8) & 0xFF
		b := (word >> 16) & 0xFF
		c := (word >> 24) & 0xFF
		imm16 := (b << 8) | c

		instrIdx := i / 4
		if op.IsWideImmediate() {
			out += fmt.Sprintf("[%04d] %-20s R%d, %d\n", instrIdx, op, a, imm16)
		} else {
			switch op.Operands() {
			case 1:
				out += fmt.Sprintf("[%04d] %-20s R%d\n", instrIdx, op, a)
			case 2:
				out += fmt.Sprintf("[%04d] %-20s R%d, R%d\n", instrIdx, op, a, b)
			case 3:
				out += fmt.Sprintf("[%04d] %-20s R%d, R%d, R%d\n", instrIdx, op, a, b, c)
			default:
				out += fmt.Sprintf("[%04d] %-20s\n", instrIdx, op)
			}
		}
	}
	return out
}

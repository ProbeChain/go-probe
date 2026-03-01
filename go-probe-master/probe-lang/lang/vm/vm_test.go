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
	"testing"

	"golang.org/x/crypto/sha3"
)

// ---- Bytecode builder helpers ----------------------------------------------

// instr encodes a standard 3-address instruction into a 4-byte little-endian
// word: [opcode:8][a:8][b:8][c:8].
func instr(op Opcode, a, b, c uint8) []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(op)|uint32(a)<<8|uint32(b)<<16|uint32(c)<<24)
	return buf
}

// instrWide encodes a wide-immediate instruction: [opcode:8][a:8][imm_hi:8][imm_lo:8].
// imm is split big-endian into the b and c byte slots.
func instrWide(op Opcode, a uint8, imm uint16) []byte {
	hi := uint8(imm >> 8)
	lo := uint8(imm & 0xFF)
	return instr(op, a, hi, lo)
}

// program concatenates instruction byte slices into a single bytecode block.
func program(instrs ...[]byte) []byte {
	var out []byte
	for _, i := range instrs {
		out = append(out, i...)
	}
	return out
}

// newTestVM creates a VM with a generous gas limit for tests that do not
// specifically test gas metering.
func newTestVM(code []byte, consts []uint64) *VM {
	return New(code, consts, 1_000_000)
}

// runVM is a test helper that runs the VM and fails the test on error.
func runVM(t *testing.T, v *VM) uint64 {
	t.Helper()
	result, err := v.Run()
	if err != nil {
		t.Fatalf("VM.Run returned unexpected error: %v", err)
	}
	return result
}

// ---- Opcode metadata tests -------------------------------------------------

func TestOpcodeString(t *testing.T) {
	cases := []struct {
		op   Opcode
		want string
	}{
		{OpAdd, "ADD"},
		{OpSub, "SUB"},
		{OpMul, "MUL"},
		{OpDiv, "DIV"},
		{OpMod, "MOD"},
		{OpNeg, "NEG"},
		{OpAnd, "AND"},
		{OpOr, "OR"},
		{OpXor, "XOR"},
		{OpNot, "NOT"},
		{OpShl, "SHL"},
		{OpShr, "SHR"},
		{OpEq, "EQ"},
		{OpNeq, "NEQ"},
		{OpLt, "LT"},
		{OpLte, "LTE"},
		{OpGt, "GT"},
		{OpGte, "GTE"},
		{OpLoadConst, "LOAD_CONST"},
		{OpLoadTrue, "LOAD_TRUE"},
		{OpLoadFalse, "LOAD_FALSE"},
		{OpLoadNil, "LOAD_NIL"},
		{OpJump, "JUMP"},
		{OpJumpIf, "JUMP_IF"},
		{OpJumpIfNot, "JUMP_IF_NOT"},
		{OpCall, "CALL"},
		{OpReturn, "RETURN"},
		{OpHalt, "HALT"},
	}
	for _, tc := range cases {
		if got := tc.op.String(); got != tc.want {
			t.Errorf("Opcode(%d).String() = %q; want %q", tc.op, got, tc.want)
		}
	}
}

func TestOpcodeUnknown(t *testing.T) {
	if got := Opcode(0xFF).String(); got != "UNKNOWN" {
		t.Errorf("unknown opcode String = %q; want UNKNOWN", got)
	}
}

// ---- Arithmetic tests ------------------------------------------------------

func TestAdd(t *testing.T) {
	// R2 = 10, R3 = 32
	// R4 = R2 + R3 = 42
	// HALT R4 → result
	code := program(
		instrWide(OpLoadConst, 2, 0), // R2 = constants[0] = 10
		instrWide(OpLoadConst, 3, 1), // R3 = constants[1] = 32
		instr(OpAdd, 4, 2, 3),         // R4 = R2 + R3
		instr(OpHalt, 4, 0, 0),        // halt with R4
	)
	v := newTestVM(code, []uint64{10, 32})
	if got := runVM(t, v); got != 42 {
		t.Errorf("Add: got %d; want 42", got)
	}
}

func TestSub(t *testing.T) {
	code := program(
		instrWide(OpLoadConst, 2, 0), // R2 = 100
		instrWide(OpLoadConst, 3, 1), // R3 = 58
		instr(OpSub, 4, 2, 3),         // R4 = 100 - 58 = 42
		instr(OpHalt, 4, 0, 0),
	)
	v := newTestVM(code, []uint64{100, 58})
	if got := runVM(t, v); got != 42 {
		t.Errorf("Sub: got %d; want 42", got)
	}
}

func TestMul(t *testing.T) {
	code := program(
		instrWide(OpLoadConst, 2, 0), // R2 = 6
		instrWide(OpLoadConst, 3, 1), // R3 = 7
		instr(OpMul, 4, 2, 3),         // R4 = 42
		instr(OpHalt, 4, 0, 0),
	)
	v := newTestVM(code, []uint64{6, 7})
	if got := runVM(t, v); got != 42 {
		t.Errorf("Mul: got %d; want 42", got)
	}
}

func TestDiv(t *testing.T) {
	code := program(
		instrWide(OpLoadConst, 2, 0), // R2 = 84
		instrWide(OpLoadConst, 3, 1), // R3 = 2
		instr(OpDiv, 4, 2, 3),         // R4 = 42
		instr(OpHalt, 4, 0, 0),
	)
	v := newTestVM(code, []uint64{84, 2})
	if got := runVM(t, v); got != 42 {
		t.Errorf("Div: got %d; want 42", got)
	}
}

func TestDivByZero(t *testing.T) {
	code := program(
		instrWide(OpLoadConst, 2, 0), // R2 = 10
		instr(OpDiv, 4, 2, 0),         // R4 = R2 / R0 (R0 is always 0)
		instr(OpHalt, 4, 0, 0),
	)
	v := newTestVM(code, []uint64{10})
	_, err := v.Run()
	if !errors.Is(err, ErrDivisionByZero) {
		t.Errorf("DivByZero: got %v; want ErrDivisionByZero", err)
	}
}

func TestMod(t *testing.T) {
	code := program(
		instrWide(OpLoadConst, 2, 0), // R2 = 127
		instrWide(OpLoadConst, 3, 1), // R3 = 5
		instr(OpMod, 4, 2, 3),         // R4 = 127 % 5 = 2
		instr(OpHalt, 4, 0, 0),
	)
	v := newTestVM(code, []uint64{127, 5})
	if got := runVM(t, v); got != 2 {
		t.Errorf("Mod: got %d; want 2", got)
	}
}

func TestNeg(t *testing.T) {
	// -1 in two's complement uint64 is 0xFFFFFFFFFFFFFFFF.
	code := program(
		instrWide(OpLoadConst, 2, 0), // R2 = 1
		instr(OpNeg, 3, 2, 0),         // R3 = -R2 = 0xFFFFFFFFFFFFFFFF
		instr(OpHalt, 3, 0, 0),
	)
	v := newTestVM(code, []uint64{1})
	got := runVM(t, v)
	if got != ^uint64(0) {
		t.Errorf("Neg: got %d; want %d (all-ones)", got, ^uint64(0))
	}
}

// ---- Bitwise tests ---------------------------------------------------------

func TestAnd(t *testing.T) {
	code := program(
		instrWide(OpLoadConst, 2, 0), // R2 = 0xFF
		instrWide(OpLoadConst, 3, 1), // R3 = 0x0F
		instr(OpAnd, 4, 2, 3),         // R4 = 0x0F
		instr(OpHalt, 4, 0, 0),
	)
	v := newTestVM(code, []uint64{0xFF, 0x0F})
	if got := runVM(t, v); got != 0x0F {
		t.Errorf("And: got 0x%x; want 0x0F", got)
	}
}

func TestOr(t *testing.T) {
	code := program(
		instrWide(OpLoadConst, 2, 0), // R2 = 0xF0
		instrWide(OpLoadConst, 3, 1), // R3 = 0x0F
		instr(OpOr, 4, 2, 3),          // R4 = 0xFF
		instr(OpHalt, 4, 0, 0),
	)
	v := newTestVM(code, []uint64{0xF0, 0x0F})
	if got := runVM(t, v); got != 0xFF {
		t.Errorf("Or: got 0x%x; want 0xFF", got)
	}
}

func TestXor(t *testing.T) {
	code := program(
		instrWide(OpLoadConst, 2, 0), // R2 = 0xFF
		instrWide(OpLoadConst, 3, 1), // R3 = 0x0F
		instr(OpXor, 4, 2, 3),         // R4 = 0xF0
		instr(OpHalt, 4, 0, 0),
	)
	v := newTestVM(code, []uint64{0xFF, 0x0F})
	if got := runVM(t, v); got != 0xF0 {
		t.Errorf("Xor: got 0x%x; want 0xF0", got)
	}
}

func TestShl(t *testing.T) {
	code := program(
		instrWide(OpLoadConst, 2, 0), // R2 = 1
		instrWide(OpLoadConst, 3, 1), // R3 = 3
		instr(OpShl, 4, 2, 3),         // R4 = 1 << 3 = 8
		instr(OpHalt, 4, 0, 0),
	)
	v := newTestVM(code, []uint64{1, 3})
	if got := runVM(t, v); got != 8 {
		t.Errorf("Shl: got %d; want 8", got)
	}
}

func TestShr(t *testing.T) {
	code := program(
		instrWide(OpLoadConst, 2, 0), // R2 = 16
		instrWide(OpLoadConst, 3, 1), // R3 = 2
		instr(OpShr, 4, 2, 3),         // R4 = 16 >> 2 = 4
		instr(OpHalt, 4, 0, 0),
	)
	v := newTestVM(code, []uint64{16, 2})
	if got := runVM(t, v); got != 4 {
		t.Errorf("Shr: got %d; want 4", got)
	}
}

// ---- Comparison tests ------------------------------------------------------

func TestEq(t *testing.T) {
	cases := []struct {
		a, b uint64
		want uint64
	}{
		{5, 5, 1},
		{5, 6, 0},
	}
	for _, tc := range cases {
		code := program(
			instrWide(OpLoadConst, 2, 0),
			instrWide(OpLoadConst, 3, 1),
			instr(OpEq, 4, 2, 3),
			instr(OpHalt, 4, 0, 0),
		)
		v := newTestVM(code, []uint64{tc.a, tc.b})
		if got := runVM(t, v); got != tc.want {
			t.Errorf("Eq(%d,%d): got %d; want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestNeq(t *testing.T) {
	code := program(
		instrWide(OpLoadConst, 2, 0),
		instrWide(OpLoadConst, 3, 1),
		instr(OpNeq, 4, 2, 3),
		instr(OpHalt, 4, 0, 0),
	)
	v := newTestVM(code, []uint64{3, 7})
	if got := runVM(t, v); got != 1 {
		t.Errorf("Neq: got %d; want 1", got)
	}
}

func TestLt(t *testing.T) {
	cases := []struct {
		a, b uint64
		want uint64
	}{
		{3, 7, 1},
		{7, 3, 0},
		{3, 3, 0},
	}
	for _, tc := range cases {
		code := program(
			instrWide(OpLoadConst, 2, 0),
			instrWide(OpLoadConst, 3, 1),
			instr(OpLt, 4, 2, 3),
			instr(OpHalt, 4, 0, 0),
		)
		v := newTestVM(code, []uint64{tc.a, tc.b})
		if got := runVM(t, v); got != tc.want {
			t.Errorf("Lt(%d,%d): got %d; want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestLte(t *testing.T) {
	cases := []struct {
		a, b uint64
		want uint64
	}{
		{3, 7, 1},
		{3, 3, 1},
		{7, 3, 0},
	}
	for _, tc := range cases {
		code := program(
			instrWide(OpLoadConst, 2, 0),
			instrWide(OpLoadConst, 3, 1),
			instr(OpLte, 4, 2, 3),
			instr(OpHalt, 4, 0, 0),
		)
		v := newTestVM(code, []uint64{tc.a, tc.b})
		if got := runVM(t, v); got != tc.want {
			t.Errorf("Lte(%d,%d): got %d; want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestGt(t *testing.T) {
	code := program(
		instrWide(OpLoadConst, 2, 0),
		instrWide(OpLoadConst, 3, 1),
		instr(OpGt, 4, 2, 3),
		instr(OpHalt, 4, 0, 0),
	)
	v := newTestVM(code, []uint64{10, 3})
	if got := runVM(t, v); got != 1 {
		t.Errorf("Gt: got %d; want 1", got)
	}
}

func TestGte(t *testing.T) {
	cases := []struct {
		a, b uint64
		want uint64
	}{
		{10, 3, 1},
		{3, 3, 1},
		{2, 3, 0},
	}
	for _, tc := range cases {
		code := program(
			instrWide(OpLoadConst, 2, 0),
			instrWide(OpLoadConst, 3, 1),
			instr(OpGte, 4, 2, 3),
			instr(OpHalt, 4, 0, 0),
		)
		v := newTestVM(code, []uint64{tc.a, tc.b})
		if got := runVM(t, v); got != tc.want {
			t.Errorf("Gte(%d,%d): got %d; want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

// ---- Load constant / booleans / nil ----------------------------------------

func TestLoadConst(t *testing.T) {
	code := program(
		instrWide(OpLoadConst, 5, 2), // R5 = constants[2] = 999
		instr(OpHalt, 5, 0, 0),
	)
	v := newTestVM(code, []uint64{0, 0, 999})
	if got := runVM(t, v); got != 999 {
		t.Errorf("LoadConst: got %d; want 999", got)
	}
}

func TestLoadTrue(t *testing.T) {
	code := program(
		instr(OpLoadTrue, 5, 0, 0),
		instr(OpHalt, 5, 0, 0),
	)
	v := newTestVM(code, nil)
	if got := runVM(t, v); got != 1 {
		t.Errorf("LoadTrue: got %d; want 1", got)
	}
}

func TestLoadFalse(t *testing.T) {
	code := program(
		instr(OpLoadTrue, 5, 0, 0),  // R5 = 1
		instr(OpLoadFalse, 5, 0, 0), // R5 = 0 (overwrite)
		instr(OpHalt, 5, 0, 0),
	)
	v := newTestVM(code, nil)
	if got := runVM(t, v); got != 0 {
		t.Errorf("LoadFalse: got %d; want 0", got)
	}
}

func TestLoadNil(t *testing.T) {
	code := program(
		instrWide(OpLoadConst, 5, 0), // R5 = 42
		instr(OpLoadNil, 5, 0, 0),    // R5 = 0 (nil)
		instr(OpHalt, 5, 0, 0),
	)
	v := newTestVM(code, []uint64{42})
	if got := runVM(t, v); got != 0 {
		t.Errorf("LoadNil: got %d; want 0", got)
	}
}

// ---- Move / Copy -----------------------------------------------------------

func TestMove(t *testing.T) {
	// MOVE should transfer the value and zero the source register.
	code := program(
		instrWide(OpLoadConst, 2, 0), // R2 = 77
		instr(OpMove, 3, 2, 0),        // R3 = R2, R2 = 0
		instr(OpAdd, 4, 2, 3),         // R4 = 0 + 77 = 77
		instr(OpHalt, 4, 0, 0),
	)
	v := newTestVM(code, []uint64{77})
	if got := runVM(t, v); got != 77 {
		t.Errorf("Move: got %d; want 77", got)
	}
	// Also verify R2 was zeroed.
	if v.registers[2] != 0 {
		t.Errorf("Move: source R2 was not zeroed; got %d", v.registers[2])
	}
}

func TestCopy(t *testing.T) {
	code := program(
		instrWide(OpLoadConst, 2, 0), // R2 = 55
		instr(OpCopy, 3, 2, 0),        // R3 = R2 (R2 unchanged)
		instr(OpAdd, 4, 2, 3),         // R4 = 55 + 55 = 110
		instr(OpHalt, 4, 0, 0),
	)
	v := newTestVM(code, []uint64{55})
	if got := runVM(t, v); got != 110 {
		t.Errorf("Copy: got %d; want 110", got)
	}
}

// ---- Control flow ----------------------------------------------------------

func TestUnconditionalJump(t *testing.T) {
	// Program layout (instruction indices 0-3):
	//   [0] LOAD_TRUE R5
	//   [1] JUMP to instr 3
	//   [2] LOAD_FALSE R5  ← should be skipped
	//   [3] HALT R5
	code := program(
		instr(OpLoadTrue, 5, 0, 0),  // [0]
		instrWide(OpJump, 0, 3),      // [1] jump to instruction index 3
		instr(OpLoadFalse, 5, 0, 0), // [2] skipped
		instr(OpHalt, 5, 0, 0),      // [3]
	)
	v := newTestVM(code, nil)
	if got := runVM(t, v); got != 1 {
		t.Errorf("UnconditionalJump: got %d; want 1", got)
	}
}

func TestJumpIfTaken(t *testing.T) {
	// [0] LOAD_TRUE R5
	// [1] JUMP_IF R5, target=3
	// [2] LOAD_FALSE R5  ← skipped because R5==1
	// [3] HALT R5
	code := program(
		instr(OpLoadTrue, 5, 0, 0),  // [0]
		instrWide(OpJumpIf, 5, 3),    // [1] jump to 3 if R5 != 0
		instr(OpLoadFalse, 5, 0, 0), // [2] skipped
		instr(OpHalt, 5, 0, 0),      // [3]
	)
	v := newTestVM(code, nil)
	if got := runVM(t, v); got != 1 {
		t.Errorf("JumpIf_taken: got %d; want 1", got)
	}
}

func TestJumpIfNotTaken(t *testing.T) {
	// R5 starts at 0; JUMP_IF should NOT branch.
	// [0] LOAD_FALSE R5      (R5=0)
	// [1] JUMP_IF R5, 3      (not taken, fall through)
	// [2] LOAD_TRUE R5       (R5=1)
	// [3] HALT R5
	code := program(
		instr(OpLoadFalse, 5, 0, 0), // [0]
		instrWide(OpJumpIf, 5, 3),    // [1] not taken
		instr(OpLoadTrue, 5, 0, 0),  // [2] executed
		instr(OpHalt, 5, 0, 0),      // [3]
	)
	v := newTestVM(code, nil)
	if got := runVM(t, v); got != 1 {
		t.Errorf("JumpIf_notTaken: got %d; want 1", got)
	}
}

func TestJumpIfNotBranchTaken(t *testing.T) {
	// JUMP_IF_NOT jumps when R5==0.
	// [0] LOAD_FALSE R5
	// [1] JUMP_IF_NOT R5, 3
	// [2] LOAD_TRUE R5       ← skipped
	// [3] HALT R5
	code := program(
		instr(OpLoadFalse, 5, 0, 0),  // [0]
		instrWide(OpJumpIfNot, 5, 3), // [1]
		instr(OpLoadTrue, 5, 0, 0),   // [2] skipped
		instr(OpHalt, 5, 0, 0),       // [3]
	)
	v := newTestVM(code, nil)
	if got := runVM(t, v); got != 0 {
		t.Errorf("JumpIfNot_taken: got %d; want 0", got)
	}
}

// ---- Call / Return ---------------------------------------------------------

// TestCallReturn tests a simple function call.
//
// Program layout:
//   [0] LOAD_CONST R2, 20
//   [1] LOAD_CONST R3, 22
//   [2] CALL R4, 4         → jump to instruction 4; return PC = instruction 3
//   [3] HALT R4
//   [4] ADD R10, R2, R3    function body: R10 = R2 + R3 = 42
//   [5] RETURN R10         → stored in R4, PC = instruction 3
func TestCallReturn(t *testing.T) {
	code := program(
		instrWide(OpLoadConst, 2, 0), // [0] R2 = 20
		instrWide(OpLoadConst, 3, 1), // [1] R3 = 22
		instrWide(OpCall, 4, 4),       // [2] R4 = call(fn at instr 4); return PC = instr 3
		instr(OpHalt, 4, 0, 0),        // [3] halt with R4
		instr(OpAdd, 10, 2, 3),        // [4] R10 = R2 + R3 = 42
		instr(OpReturn, 10, 0, 0),     // [5] return R10 → stored in R4, PC = 3
	)
	v := newTestVM(code, []uint64{20, 22})
	if got := runVM(t, v); got != 42 {
		t.Errorf("CallReturn: got %d; want 42", got)
	}
}

// ---- Memory operations -----------------------------------------------------

func TestMemoryAllocStoreLoad(t *testing.T) {
	// Allocate 8 bytes, store value 0xDEADBEEF, load it back.
	// [0] LOAD_CONST R2, 8     ; size = 8 bytes
	// [1] ALLOC R3, R2         ; R3 = ptr to 8-byte region
	// [2] LOAD_CONST R4, val   ; R4 = 0xDEADBEEF
	// [3] STORE_MEM R3+0, R4   ; Memory[R3+0] = R4
	// [4] LOAD_MEM R5, R3+0    ; R5 = Memory[R3+0]
	// [5] HALT R5
	const val = uint64(0xDEADBEEF)
	code := program(
		instrWide(OpLoadConst, 2, 0), // [0] R2 = 8
		instr(OpAlloc, 3, 2, 0),       // [1] R3 = alloc(R2)
		instrWide(OpLoadConst, 4, 1), // [2] R4 = 0xDEADBEEF
		instr(OpStoreMem, 3, 4, 0),    // [3] Memory[R3+0] = R4
		instr(OpLoadMem, 5, 3, 0),     // [4] R5 = Memory[R3+0]
		instr(OpHalt, 5, 0, 0),        // [5]
	)
	v := newTestVM(code, []uint64{8, val})
	if got := runVM(t, v); got != val {
		t.Errorf("MemAllocStoreLoad: got 0x%x; want 0x%x", got, val)
	}
}

func TestMemoryFree(t *testing.T) {
	// Alloc then free; verify no error.
	code := program(
		instrWide(OpLoadConst, 2, 0), // R2 = 16
		instr(OpAlloc, 3, 2, 0),       // R3 = alloc(16)
		instr(OpFree, 3, 0, 0),        // free(R3)
		instr(OpHalt, 0, 0, 0),        // halt R0 = 0
	)
	v := newTestVM(code, []uint64{16})
	if _, err := v.Run(); err != nil {
		t.Fatalf("MemFree: unexpected error: %v", err)
	}
}

func TestMemoryDoubleFree(t *testing.T) {
	code := program(
		instrWide(OpLoadConst, 2, 0),
		instr(OpAlloc, 3, 2, 0),
		instr(OpFree, 3, 0, 0),
		instr(OpFree, 3, 0, 0), // double free
		instr(OpHalt, 0, 0, 0),
	)
	v := newTestVM(code, []uint64{16})
	_, err := v.Run()
	if !errors.Is(err, ErrDoubleFree) {
		t.Errorf("DoubleFree: got %v; want ErrDoubleFree", err)
	}
}

func TestMemoryOutOfBounds(t *testing.T) {
	// Try to read from address 0 (not allocated).
	code := program(
		instr(OpLoadMem, 5, 0, 0), // Load from Memory[R0 + 0] = Memory[0] (not allocated)
		instr(OpHalt, 5, 0, 0),
	)
	v := newTestVM(code, nil)
	_, err := v.Run()
	if !errors.Is(err, ErrInvalidAddress) {
		t.Errorf("OutOfBounds: got %v; want ErrInvalidAddress", err)
	}
}

// ---- Array operations ------------------------------------------------------

func TestArrayNewGetSet(t *testing.T) {
	// Create array of 4 elements, set index 2 = 99, get index 2.
	code := program(
		instrWide(OpLoadConst, 2, 0),   // [0] R2 = 4 (element count)
		instr(OpArrayNew, 3, 2, 0),      // [1] R3 = new array(4)
		instrWide(OpLoadConst, 4, 1),   // [2] R4 = 2 (index)
		instrWide(OpLoadConst, 5, 2),   // [3] R5 = 99 (value)
		instr(OpArraySet, 3, 4, 5),      // [4] Array(R3)[R4] = R5
		instr(OpArrayGet, 6, 3, 4),      // [5] R6 = Array(R3)[R4]
		instr(OpHalt, 6, 0, 0),          // [6]
	)
	v := newTestVM(code, []uint64{4, 2, 99})
	if got := runVM(t, v); got != 99 {
		t.Errorf("ArrayGetSet: got %d; want 99", got)
	}
}

// ---- Push / Pop ------------------------------------------------------------

func TestPushPop(t *testing.T) {
	code := program(
		instrWide(OpLoadConst, 2, 0), // R2 = 42
		instr(OpPush, 2, 0, 0),        // push R2
		instrWide(OpLoadConst, 2, 1), // R2 = 0 (overwrite)
		instr(OpPop, 2, 0, 0),         // R2 = pop = 42
		instr(OpHalt, 2, 0, 0),
	)
	v := newTestVM(code, []uint64{42, 0})
	if got := runVM(t, v); got != 42 {
		t.Errorf("PushPop: got %d; want 42", got)
	}
}

func TestPopUnderflow(t *testing.T) {
	code := program(
		instr(OpPop, 2, 0, 0),
		instr(OpHalt, 0, 0, 0),
	)
	v := newTestVM(code, nil)
	_, err := v.Run()
	if !errors.Is(err, ErrStackUnderflow) {
		t.Errorf("PopUnderflow: got %v; want ErrStackUnderflow", err)
	}
}

// ---- Resource management ---------------------------------------------------

func TestResourceLifecycle(t *testing.T) {
	code := program(
		instrWide(OpLoadConst, 2, 0),    // R2 = resource type 0
		instr(OpResourceNew, 3, 2, 0),   // R3 = new resource handle
		instr(OpResourceCheck, 3, 0, 0), // R3 = 1 if live
		instr(OpResourceDrop, 3, 0, 0),  // drop (handle is now 1 from check, but handle was stored in R3 before check rewrote it)
		instr(OpHalt, 3, 0, 0),
	)
	// Note: ResourceCheck overwrites R3 with 1 (live), so the subsequent Drop
	// tries to drop handle=1. To properly test, save the handle in a different reg.
	code = program(
		instrWide(OpLoadConst, 2, 0),    // R2 = 0 (type index)
		instr(OpResourceNew, 3, 2, 0),   // R3 = new resource (handle)
		instr(OpCopy, 8, 3, 0),           // R8 = R3 (save handle)
		instr(OpResourceCheck, 3, 0, 0), // R3 = 1 (live)
		instr(OpResourceDrop, 8, 0, 0),  // drop handle stored in R8
		instr(OpHalt, 3, 0, 0),           // R3 = 1 (was live when checked)
	)
	v := newTestVM(code, []uint64{0})
	if got := runVM(t, v); got != 1 {
		t.Errorf("ResourceCheck: got %d; want 1", got)
	}
}

func TestResourceDoubleDrop(t *testing.T) {
	code := program(
		instrWide(OpLoadConst, 2, 0),
		instr(OpResourceNew, 3, 2, 0),
		instr(OpResourceDrop, 3, 0, 0),
		instr(OpResourceDrop, 3, 0, 0), // double drop — R3 still holds the handle value
		instr(OpHalt, 0, 0, 0),
	)
	v := newTestVM(code, []uint64{0})
	_, err := v.Run()
	if !errors.Is(err, ErrResourceFault) {
		t.Errorf("DoubleDrop: got %v; want ErrResourceFault", err)
	}
}

// ---- SHA3 hash opcode ------------------------------------------------------

func TestSHA3Opcode(t *testing.T) {
	const srcData = "hello"

	// Compute expected Keccak256("hello") using the sha3 package (same as
	// the execSHA3 implementation uses internally).
	h := sha3.NewLegacyKeccak256()
	h.Write([]byte(srcData))
	expected := h.Sum(nil) // 32 bytes

	// Build a VM with pre-allocated memory regions and run only the SHA3 opcode.
	v := New(nil, nil, 1_000_000)
	srcPtr, err := v.memory.Alloc(8)
	if err != nil {
		t.Fatalf("Alloc src: %v", err)
	}
	if err := v.memory.WriteSlice(srcPtr, []byte(srcData)); err != nil {
		t.Fatalf("WriteSlice: %v", err)
	}
	dstPtr, err := v.memory.Alloc(32)
	if err != nil {
		t.Fatalf("Alloc dst: %v", err)
	}

	consts := []uint64{dstPtr, srcPtr, uint64(len(srcData))}
	code := program(
		instrWide(OpLoadConst, 10, 0), // R10 = dstPtr
		instrWide(OpLoadConst, 11, 1), // R11 = srcPtr
		instrWide(OpLoadConst, 12, 2), // R12 = 5 (length)
		instr(OpSHA3, 10, 11, 12),      // SHA3(dst=R10, src=R11, len=R12)
		instr(OpHalt, 0, 0, 0),         // halt R0
	)
	v.code = code
	v.constants = consts

	if _, err := v.Run(); err != nil {
		t.Fatalf("SHA3 opcode: %v", err)
	}

	digest, err := v.memory.ReadSlice(dstPtr, 32)
	if err != nil {
		t.Fatalf("SHA3 read digest: %v", err)
	}
	for i, b := range expected {
		if digest[i] != b {
			t.Errorf("SHA3 digest[%d] = 0x%02x; want 0x%02x", i, digest[i], b)
		}
	}
}

// ---- SHAKE256 opcode -------------------------------------------------------

func TestSHAKE256Opcode(t *testing.T) {
	const srcData = "probe"

	v := New(nil, nil, 1_000_000)

	srcPtr, err := v.memory.Alloc(8)
	if err != nil {
		t.Fatalf("Alloc src: %v", err)
	}
	if err := v.memory.WriteSlice(srcPtr, []byte(srcData)); err != nil {
		t.Fatalf("WriteSlice: %v", err)
	}
	dstPtr, err := v.memory.Alloc(32)
	if err != nil {
		t.Fatalf("Alloc dst: %v", err)
	}

	consts := []uint64{dstPtr, srcPtr, uint64(len(srcData))}
	code := program(
		instrWide(OpLoadConst, 10, 0),
		instrWide(OpLoadConst, 11, 1),
		instrWide(OpLoadConst, 12, 2),
		instr(OpSHAKE256, 10, 11, 12),
		instr(OpHalt, 0, 0, 0),
	)
	v.code = code
	v.constants = consts

	if _, err := v.Run(); err != nil {
		t.Fatalf("SHAKE256 opcode: %v", err)
	}

	digest, err := v.memory.ReadSlice(dstPtr, 32)
	if err != nil {
		t.Fatalf("ReadSlice digest: %v", err)
	}
	// Verify the output is non-zero (a 32-byte all-zero output from SHAKE256 is incorrect).
	allZero := true
	for _, b := range digest {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("SHAKE256 produced all-zero output")
	}
}

// ---- Gas metering ----------------------------------------------------------

func TestGasExhaustion(t *testing.T) {
	// A loop that runs many ADD instructions should eventually exhaust gas.
	// gasLimit = 10 is far too small for 100 iterations of the loop.
	//
	// Program:
	//   [0] LOAD_CONST R2, 100   ; loop counter
	//   [1] LOAD_CONST R3, 1     ; constant 1
	//   [2] ADD R4, R4, R3       ; R4++ (loop body)
	//   [3] SUB R2, R2, R3       ; R2-- (decrement counter)
	//   [4] JUMP_IF_NOT R2, 6   ; exit when R2 == 0
	//   [5] JUMP 2               ; loop back
	//   [6] HALT R4
	code := program(
		instrWide(OpLoadConst, 2, 0), // [0] R2 = 100
		instrWide(OpLoadConst, 3, 1), // [1] R3 = 1
		instr(OpAdd, 4, 4, 3),         // [2] R4++
		instr(OpSub, 2, 2, 3),         // [3] R2--
		instrWide(OpJumpIfNot, 2, 6),  // [4] if R2==0 jump to [6]
		instrWide(OpJump, 0, 2),        // [5] jump to [2]
		instr(OpHalt, 4, 0, 0),         // [6]
	)
	v := New(code, []uint64{100, 1}, 10)
	_, err := v.Run()
	if !errors.Is(err, ErrOutOfGas) {
		t.Errorf("GasExhaustion: got %v; want ErrOutOfGas", err)
	}
}

func TestGasAccounting(t *testing.T) {
	// A simple program: LOAD_CONST (gasTrivial) + HALT (gasTrivial) = 2.
	code := program(
		instrWide(OpLoadConst, 2, 0),
		instr(OpHalt, 2, 0, 0),
	)
	v := newTestVM(code, []uint64{7})
	runVM(t, v)
	want := 2 * gasTrivial
	if v.GasUsed() != want {
		t.Errorf("GasAccounting: gasUsed=%d; want %d", v.GasUsed(), want)
	}
}

// ---- Fibonacci (complete program) ------------------------------------------

// TestFibonacci computes fib(10) = 55 using an iterative loop.
//
// Register assignments:
//
//	R2 = n (iteration counter, starts at 10)
//	R3 = a (fib[i-2], starts at 0)
//	R4 = b (fib[i-1], starts at 1)
//	R5 = tmp  (scratch)
//	R6 = 1    (constant one)
//
// Instruction layout:
//
//	[0]  LOAD_CONST R2, 10
//	[1]  LOAD_CONST R3, 0
//	[2]  LOAD_CONST R4, 1
//	[3]  LOAD_CONST R6, 1     ; constant 1
//	[4]  EQ R7, R2, R0        ; R7 = (n == 0)
//	[5]  JUMP_IF R7, 11       ; if n==0 jump to exit [11]
//	[6]  ADD R5, R3, R4       ; tmp = a + b
//	[7]  COPY R3, R4          ; a = b
//	[8]  COPY R4, R5          ; b = tmp
//	[9]  SUB R2, R2, R6       ; n--
//	[10] JUMP 4               ; loop back to [4]
//	[11] HALT R3              ; result = a = fib(10) = 55
func TestFibonacci(t *testing.T) {
	consts := []uint64{10, 0, 1}
	code := program(
		instrWide(OpLoadConst, 2, 0), // [0]  R2 = 10
		instrWide(OpLoadConst, 3, 1), // [1]  R3 = 0
		instrWide(OpLoadConst, 4, 2), // [2]  R4 = 1
		instrWide(OpLoadConst, 6, 2), // [3]  R6 = 1
		instr(OpEq, 7, 2, 0),          // [4]  R7 = (R2 == 0)
		instrWide(OpJumpIf, 7, 11),    // [5]  if R7 jump to [11]
		instr(OpAdd, 5, 3, 4),          // [6]  R5 = R3+R4
		instr(OpCopy, 3, 4, 0),         // [7]  R3 = R4
		instr(OpCopy, 4, 5, 0),         // [8]  R4 = R5
		instr(OpSub, 2, 2, 6),          // [9]  R2 = R2-1
		instrWide(OpJump, 0, 4),        // [10] jump to [4]
		instr(OpHalt, 3, 0, 0),         // [11] halt with R3 = fib(10)
	)
	v := newTestVM(code, consts)
	if got := runVM(t, v); got != 55 {
		t.Errorf("Fibonacci(10): got %d; want 55", got)
	}
}

// ---- Disassembly -----------------------------------------------------------

func TestDisassemble(t *testing.T) {
	code := program(
		instrWide(OpLoadConst, 2, 0),
		instr(OpAdd, 3, 2, 2),
		instr(OpHalt, 3, 0, 0),
	)
	out := Disassemble(code)
	if out == "" {
		t.Error("Disassemble returned empty string")
	}
	// Spot-check that opcode names appear.
	for _, want := range []string{"LOAD_CONST", "ADD", "HALT"} {
		if !containsStr(out, want) {
			t.Errorf("Disassemble output missing %q:\n%s", want, out)
		}
	}
}

// containsStr reports whether sub appears anywhere in s.
func containsStr(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// ---- Memory unit tests -----------------------------------------------------

func TestMemoryUnit(t *testing.T) {
	m := NewMemory(1024)

	// Basic alloc + read/write uint64.
	ptr, err := m.Alloc(16)
	if err != nil {
		t.Fatalf("Alloc: %v", err)
	}
	if err := m.WriteUint64(ptr, 0xCAFEBABE); err != nil {
		t.Fatalf("WriteUint64: %v", err)
	}
	v, err := m.ReadUint64(ptr)
	if err != nil {
		t.Fatalf("ReadUint64: %v", err)
	}
	if v != 0xCAFEBABE {
		t.Errorf("ReadUint64: got 0x%x; want 0xCAFEBABE", v)
	}

	// Out-of-bounds read.
	_, err = m.ReadUint64(ptr + 8192)
	if !errors.Is(err, ErrInvalidAddress) {
		t.Errorf("OOB read: got %v; want ErrInvalidAddress", err)
	}

	// Free.
	if err := m.Free(ptr); err != nil {
		t.Fatalf("Free: %v", err)
	}

	// Double free.
	if err := m.Free(ptr); !errors.Is(err, ErrDoubleFree) {
		t.Errorf("DoubleFree: got %v; want ErrDoubleFree", err)
	}

	// Out-of-memory: limit is 1024, try to alloc 2048.
	_, err = m.Alloc(2048)
	if !errors.Is(err, ErrOutOfMemory) {
		t.Errorf("OOM: got %v; want ErrOutOfMemory", err)
	}
}

func TestMemorySliceReadWrite(t *testing.T) {
	m := NewMemory(0)
	ptr, err := m.Alloc(16)
	if err != nil {
		t.Fatalf("Alloc: %v", err)
	}

	data := []byte{1, 2, 3, 4, 5}
	if err := m.WriteSlice(ptr, data); err != nil {
		t.Fatalf("WriteSlice: %v", err)
	}

	got, err := m.ReadSlice(ptr, 5)
	if err != nil {
		t.Fatalf("ReadSlice: %v", err)
	}
	for i, b := range data {
		if got[i] != b {
			t.Errorf("ReadSlice[%d] = %d; want %d", i, got[i], b)
		}
	}
}

// ---- Blockchain context opcodes --------------------------------------------

func TestBlockchainContext(t *testing.T) {
	code := program(
		instr(OpBlockNum, 2, 0, 0),  // R2 = blockNum
		instr(OpBlockTime, 3, 0, 0), // R3 = blockTime
		instr(OpCaller, 4, 0, 0),    // R4 = callerAddr
		instr(OpHalt, 2, 0, 0),      // halt with block num
	)
	v := newTestVM(code, nil)
	v.SetBlockContext(12345, 9999, 0xABCD)
	if got := runVM(t, v); got != 12345 {
		t.Errorf("BlockNum: got %d; want 12345", got)
	}
	if v.registers[3] != 9999 {
		t.Errorf("BlockTime: got %d; want 9999", v.registers[3])
	}
	if v.registers[4] != 0xABCD {
		t.Errorf("Caller: got 0x%x; want 0xABCD", v.registers[4])
	}
}

// ---- Agent opcodes (synchronous stub) --------------------------------------

func TestAgentRecv(t *testing.T) {
	code := program(
		instr(OpRecv, 5, 0, 0),   // R5 = dequeue message
		instr(OpHalt, 5, 0, 0),
	)
	v := newTestVM(code, nil)
	v.EnqueueMessage(77)
	if got := runVM(t, v); got != 77 {
		t.Errorf("AgentRecv: got %d; want 77", got)
	}
}

func TestAgentRecvEmpty(t *testing.T) {
	code := program(
		instr(OpRecv, 5, 0, 0),
		instr(OpHalt, 5, 0, 0),
	)
	v := newTestVM(code, nil)
	_, err := v.Run()
	if !errors.Is(err, ErrNilReceive) {
		t.Errorf("AgentRecvEmpty: got %v; want ErrNilReceive", err)
	}
}

// ---- R0 zero-register ------------------------------------------------------

func TestR0IsZero(t *testing.T) {
	// Writing to R0 should be silently discarded.
	code := program(
		instrWide(OpLoadConst, 0, 0), // attempt to write 42 to R0
		instr(OpHalt, 0, 0, 0),        // halt with R0 (should still be 0)
	)
	v := newTestVM(code, []uint64{42})
	if got := runVM(t, v); got != 0 {
		t.Errorf("R0IsZero: got %d; want 0 (R0 must always be 0)", got)
	}
}

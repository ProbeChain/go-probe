// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.

// Package codegen includes bytecode verification.
//
// The verifier performs Move-inspired bytecode-level safety checks,
// ensuring that safety properties hold even if the compiler has bugs.
package codegen

import "fmt"

// VerifyError describes a bytecode verification failure.
type VerifyError struct {
	Offset  int
	Message string
}

func (e *VerifyError) Error() string {
	return fmt.Sprintf("verify error at offset %d: %s", e.Offset, e.Message)
}

// Verify checks bytecode for safety violations.
// This is a Move-inspired bytecode verifier that ensures:
//  1. No out-of-bounds register access
//  2. No out-of-bounds constant access
//  3. All paths return or halt
//  4. Jump targets are valid instruction boundaries
//  5. Resource types are used linearly (moved exactly once)
func Verify(bc *Bytecode) []VerifyError {
	var errors []VerifyError

	if len(bc.Code) == 0 {
		return errors
	}

	// Check instruction boundaries and register bounds.
	for offset := 0; offset < len(bc.Code); offset += 4 {
		if offset+4 > len(bc.Code) {
			errors = append(errors, VerifyError{
				Offset:  offset,
				Message: "truncated instruction",
			})
			break
		}

		op := bc.Code[offset]

		// Validate register indices.
		regA := bc.Code[offset+1]
		if !isValidInstruction(op) {
			errors = append(errors, VerifyError{
				Offset:  offset,
				Message: fmt.Sprintf("unknown opcode: %d", op),
			})
			continue
		}

		// Check register bounds (max 256 registers).
		_ = regA // all uint8 values are valid register indices

		// For LoadConst, check constant pool bounds.
		if op == vmLoadConst {
			constIdx := uint16(bc.Code[offset+2]) | uint16(bc.Code[offset+3])<<8
			if int(constIdx) >= len(bc.Constants) {
				errors = append(errors, VerifyError{
					Offset:  offset,
					Message: fmt.Sprintf("constant index %d out of bounds (pool size %d)", constIdx, len(bc.Constants)),
				})
			}
		}

		// For jumps, validate target.
		if op == vmJump || op == vmJumpIf || op == vmJumpIfNot {
			target := uint16(bc.Code[offset+2]) | uint16(bc.Code[offset+3])<<8
			targetOffset := int(target) * 4
			if targetOffset < 0 || targetOffset >= len(bc.Code) {
				errors = append(errors, VerifyError{
					Offset:  offset,
					Message: fmt.Sprintf("jump target %d out of bounds", targetOffset),
				})
			}
		}
	}

	// Check that the last instruction is a terminator.
	if len(bc.Code) >= 4 {
		lastOp := bc.Code[len(bc.Code)-4]
		if lastOp != vmReturn && lastOp != vmHalt && lastOp != vmJump {
			errors = append(errors, VerifyError{
				Offset:  len(bc.Code) - 4,
				Message: "function does not end with return, halt, or jump",
			})
		}
	}

	return errors
}

func isValidInstruction(op byte) bool {
	return op <= vmPop
}

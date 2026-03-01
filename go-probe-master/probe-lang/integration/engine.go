// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.

// Package integration bridges the PROBE Language VM with the ProbeChain blockchain,
// enabling PROBE smart contracts to be deployed and executed on-chain.
package integration

import (
	"errors"
	"fmt"

	"github.com/probechain/go-probe/common"
	probevm "github.com/probechain/go-probe/probe-lang/lang/vm"
)

var (
	// ErrInvalidBytecode is returned when contract bytecode fails verification.
	ErrInvalidBytecode = errors.New("invalid PROBE bytecode")

	// ErrExecutionFailed is returned when contract execution fails.
	ErrExecutionFailed = errors.New("PROBE contract execution failed")

	// PROBEMagicPrefix identifies PROBE Language bytecode (vs EVM bytecode).
	// Contracts prefixed with this 4-byte magic are routed to the PROBE VM.
	PROBEMagicPrefix = []byte{0x50, 0x52, 0x42, 0x45} // "PRBE"
)

// Contract represents a deployed PROBE language contract.
type Contract struct {
	Address   common.Address
	Code      []byte   // raw bytecode (without magic prefix)
	Constants []uint64 // constant pool
}

// ExecutionContext provides blockchain state to the PROBE VM.
type ExecutionContext struct {
	Caller    common.Address
	Origin    common.Address
	Value     uint64 // attached value in pico
	GasLimit  uint64
	BlockNum  uint64
	BlockTime uint64
}

// ExecutionResult contains the output of a PROBE contract execution.
type ExecutionResult struct {
	ReturnValue uint64
	GasUsed     uint64
	Logs        []Log
	Success     bool
}

// Log is an event emitted during contract execution.
type Log struct {
	Address common.Address
	Topics  []common.Hash
	Data    []byte
}

// IsPROBEContract checks if bytecode is a PROBE Language contract.
func IsPROBEContract(code []byte) bool {
	if len(code) < len(PROBEMagicPrefix) {
		return false
	}
	for i, b := range PROBEMagicPrefix {
		if code[i] != b {
			return false
		}
	}
	return true
}

// DecodePROBEContract extracts the PROBE bytecode and constants from raw contract data.
// Format: [magic:4][numConstants:4][constants:numConstants*8][code:...]
func DecodePROBEContract(raw []byte) (*Contract, error) {
	if !IsPROBEContract(raw) {
		return nil, ErrInvalidBytecode
	}

	if len(raw) < 8 {
		return nil, fmt.Errorf("%w: too short", ErrInvalidBytecode)
	}

	numConst := uint32(raw[4]) | uint32(raw[5])<<8 | uint32(raw[6])<<16 | uint32(raw[7])<<24
	constEnd := 8 + int(numConst)*8

	if len(raw) < constEnd {
		return nil, fmt.Errorf("%w: truncated constant pool", ErrInvalidBytecode)
	}

	constants := make([]uint64, numConst)
	for i := 0; i < int(numConst); i++ {
		offset := 8 + i*8
		constants[i] = uint64(raw[offset]) | uint64(raw[offset+1])<<8 |
			uint64(raw[offset+2])<<16 | uint64(raw[offset+3])<<24 |
			uint64(raw[offset+4])<<32 | uint64(raw[offset+5])<<40 |
			uint64(raw[offset+6])<<48 | uint64(raw[offset+7])<<56
	}

	return &Contract{
		Code:      raw[constEnd:],
		Constants: constants,
	}, nil
}

// EncodePROBEContract encodes a PROBE contract for on-chain storage.
func EncodePROBEContract(code []byte, constants []uint64) []byte {
	numConst := uint32(len(constants))
	result := make([]byte, 0, 4+4+len(constants)*8+len(code))

	// Magic prefix
	result = append(result, PROBEMagicPrefix...)

	// Number of constants (little-endian u32)
	result = append(result, byte(numConst), byte(numConst>>8), byte(numConst>>16), byte(numConst>>24))

	// Constants (little-endian u64 each)
	for _, c := range constants {
		result = append(result,
			byte(c), byte(c>>8), byte(c>>16), byte(c>>24),
			byte(c>>32), byte(c>>40), byte(c>>48), byte(c>>56))
	}

	// Bytecode
	result = append(result, code...)
	return result
}

// Execute runs a PROBE contract in the VM with the given context.
func Execute(contract *Contract, ctx *ExecutionContext) (*ExecutionResult, error) {
	// Create and configure the VM.
	v := probevm.New(contract.Code, contract.Constants, ctx.GasLimit)

	// Set blockchain context.
	v.SetBlockContext(ctx.BlockNum, ctx.BlockTime, addressToUint64(ctx.Caller))

	// Run the contract.
	retVal, err := v.Run()

	result := &ExecutionResult{
		ReturnValue: retVal,
		GasUsed:     v.GasUsed(),
		Success:     err == nil,
	}

	if err != nil {
		return result, fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}

	return result, nil
}

// addressToUint64 converts the first 8 bytes of an address to a uint64 for VM registers.
func addressToUint64(addr common.Address) uint64 {
	var v uint64
	for i := 0; i < 8 && i < len(addr); i++ {
		v |= uint64(addr[i]) << (i * 8)
	}
	return v
}

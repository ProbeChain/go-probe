// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.

// Package integration provides RPC API methods for PROBE Language contracts.
package integration

import (
	"context"
	"fmt"

	"github.com/probechain/go-probe/common"
	"github.com/probechain/go-probe/common/hexutil"
)

// ProbeLanguageAPI provides RPC methods for PROBE Language operations.
type ProbeLanguageAPI struct{}

// NewProbeLanguageAPI creates a new PROBE Language RPC API.
func NewProbeLanguageAPI() *ProbeLanguageAPI {
	return &ProbeLanguageAPI{}
}

// CompileResult contains the output of compiling PROBE source code.
type CompileResult struct {
	Bytecode  hexutil.Bytes `json:"bytecode"`
	Constants []uint64      `json:"constants"`
	Success   bool          `json:"success"`
	Errors    []string      `json:"errors,omitempty"`
}

// CallResult contains the output of calling a PROBE contract.
type CallResult struct {
	ReturnValue hexutil.Uint64 `json:"returnValue"`
	GasUsed     hexutil.Uint64 `json:"gasUsed"`
	Success     bool           `json:"success"`
	Error       string         `json:"error,omitempty"`
}

// TokenInfo returns PROBE token metadata.
func (api *ProbeLanguageAPI) TokenInfo(_ context.Context) map[string]interface{} {
	return map[string]interface{}{
		"name":     "PROBE",
		"symbol":   "PROBE",
		"decimals": 18,
		"supply":   "10000000000",
	}
}

// IsPROBEContract checks if bytecode at an address is a PROBE Language contract.
func (api *ProbeLanguageAPI) IsPROBEContract(_ context.Context, code hexutil.Bytes) bool {
	return IsPROBEContract(code)
}

// SimulateCall simulates executing a PROBE contract without modifying state.
func (api *ProbeLanguageAPI) SimulateCall(_ context.Context, contractCode hexutil.Bytes, caller common.Address, gasLimit hexutil.Uint64) (*CallResult, error) {
	contract, err := DecodePROBEContract(contractCode)
	if err != nil {
		return &CallResult{
			Success: false,
			Error:   fmt.Sprintf("decode error: %v", err),
		}, nil
	}

	ctx := &ExecutionContext{
		Caller:   caller,
		GasLimit: uint64(gasLimit),
	}

	result, err := Execute(contract, ctx)
	if err != nil {
		return &CallResult{
			ReturnValue: hexutil.Uint64(result.ReturnValue),
			GasUsed:     hexutil.Uint64(result.GasUsed),
			Success:     false,
			Error:       err.Error(),
		}, nil
	}

	return &CallResult{
		ReturnValue: hexutil.Uint64(result.ReturnValue),
		GasUsed:     hexutil.Uint64(result.GasUsed),
		Success:     true,
	}, nil
}

// Version returns the PROBE Language version.
func (api *ProbeLanguageAPI) Version(_ context.Context) string {
	return "0.1.0"
}

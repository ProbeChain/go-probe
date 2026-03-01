// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.

// Package chain provides blockchain primitives for the PROBE standard library.
package chain

// Block represents a blockchain block header accessible from PROBE contracts.
type Block struct {
	Number    uint64
	Timestamp uint64
	Hash      [32]byte
	Parent    [32]byte
	Validator [20]byte
}

// Transaction represents a transaction context accessible from PROBE contracts.
type Transaction struct {
	Hash      [32]byte
	From      [20]byte
	To        [20]byte
	Value     uint64 // value in pico
	GasPrice  uint64
	GasLimit  uint64
	Nonce     uint64
	Data      []byte
}

// State provides access to on-chain state.
type State interface {
	GetBalance(addr [20]byte) uint64
	SetBalance(addr [20]byte, balance uint64)
	GetStorage(addr [20]byte, key [32]byte) [32]byte
	SetStorage(addr [20]byte, key [32]byte, value [32]byte)
	GetCode(addr [20]byte) []byte
	Exists(addr [20]byte) bool
}

// Log represents an emitted event log.
type Log struct {
	Address [20]byte
	Topics  [][32]byte
	Data    []byte
}

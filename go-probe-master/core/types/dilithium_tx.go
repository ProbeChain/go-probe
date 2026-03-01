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

package types

import (
	"math/big"

	"github.com/probechain/go-probe/common"
)

// DilithiumTx is a transaction type that uses CRYSTALS-Dilithium (ML-DSA-44)
// post-quantum digital signatures. Unlike ECDSA transactions, Dilithium does
// not support public key recovery from the signature, so the transaction must
// carry the public key explicitly.
type DilithiumTx struct {
	ChainID    *big.Int
	Nonce      uint64
	GasTipCap  *big.Int        // a.k.a. maxPriorityFeePerGas
	GasFeeCap  *big.Int        // a.k.a. maxFeePerGas
	Gas        uint64
	To         *common.Address `rlp:"nil"` // nil means contract creation
	Value      *big.Int
	Data       []byte
	AccessList AccessList

	// Dilithium signature fields (replaces V, R, S)
	PubKey    []byte // 1,312 bytes — Dilithium public key
	Signature []byte // 2,420 bytes — Dilithium signature
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *DilithiumTx) copy() TxData {
	cpy := &DilithiumTx{
		Nonce: tx.Nonce,
		To:    tx.To,
		Data:  common.CopyBytes(tx.Data),
		Gas:   tx.Gas,
		// These are copied below.
		AccessList: make(AccessList, len(tx.AccessList)),
		Value:      new(big.Int),
		ChainID:    new(big.Int),
		GasTipCap:  new(big.Int),
		GasFeeCap:  new(big.Int),
		PubKey:     common.CopyBytes(tx.PubKey),
		Signature:  common.CopyBytes(tx.Signature),
	}
	copy(cpy.AccessList, tx.AccessList)
	if tx.Value != nil {
		cpy.Value.Set(tx.Value)
	}
	if tx.ChainID != nil {
		cpy.ChainID.Set(tx.ChainID)
	}
	if tx.GasTipCap != nil {
		cpy.GasTipCap.Set(tx.GasTipCap)
	}
	if tx.GasFeeCap != nil {
		cpy.GasFeeCap.Set(tx.GasFeeCap)
	}
	return cpy
}

// accessors for innerTx.
func (tx *DilithiumTx) txType() byte           { return DilithiumTxType }
func (tx *DilithiumTx) chainID() *big.Int      { return tx.ChainID }
func (tx *DilithiumTx) protected() bool        { return true }
func (tx *DilithiumTx) accessList() AccessList { return tx.AccessList }
func (tx *DilithiumTx) data() []byte           { return tx.Data }
func (tx *DilithiumTx) gas() uint64            { return tx.Gas }
func (tx *DilithiumTx) gasFeeCap() *big.Int    { return tx.GasFeeCap }
func (tx *DilithiumTx) gasTipCap() *big.Int    { return tx.GasTipCap }
func (tx *DilithiumTx) gasPrice() *big.Int     { return tx.GasFeeCap }
func (tx *DilithiumTx) value() *big.Int        { return tx.Value }
func (tx *DilithiumTx) nonce() uint64          { return tx.Nonce }
func (tx *DilithiumTx) to() *common.Address    { return tx.To }

// rawSignatureValues returns zero values for V, R, S since Dilithium
// does not use ECDSA signature components.
func (tx *DilithiumTx) rawSignatureValues() (v, r, s *big.Int) {
	return new(big.Int), new(big.Int), new(big.Int)
}

// setSignatureValues is a no-op for Dilithium transactions since they
// use PubKey and Signature fields instead of V, R, S.
func (tx *DilithiumTx) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.ChainID = chainID
}

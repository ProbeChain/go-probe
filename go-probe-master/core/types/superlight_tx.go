// Copyright 2024 The go-probe Authors
// This file is part of the go-probe library.
//
// The go-probe library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-probe library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-probe library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"math/big"

	"github.com/probechain/go-probe/common"
)

// SuperlightTx is a transaction type for Superlight DEX operations.
// It carries order placement, cancellation, and settlement data on-chain.
type SuperlightTx struct {
	ChainID    *big.Int
	Nonce      uint64
	GasTipCap  *big.Int        // a.k.a. maxPriorityFeePerGas
	GasFeeCap  *big.Int        // a.k.a. maxFeePerGas
	Gas        uint64
	To         *common.Address `rlp:"nil"` // DEX settlement address
	Value      *big.Int
	Data       []byte          // RLP-encoded DEX operation (order/cancel/settle)
	AccessList AccessList

	// Standard signature fields
	V *big.Int
	R *big.Int
	S *big.Int
}

// copy creates a deep copy of the transaction data.
func (tx *SuperlightTx) copy() TxData {
	cpy := &SuperlightTx{
		Nonce: tx.Nonce,
		To:    tx.To,
		Data:  common.CopyBytes(tx.Data),
		Gas:   tx.Gas,
		// Deep copy below.
		AccessList: make(AccessList, len(tx.AccessList)),
		Value:      new(big.Int),
		ChainID:    new(big.Int),
		GasTipCap:  new(big.Int),
		GasFeeCap:  new(big.Int),
		V:          new(big.Int),
		R:          new(big.Int),
		S:          new(big.Int),
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
	if tx.V != nil {
		cpy.V.Set(tx.V)
	}
	if tx.R != nil {
		cpy.R.Set(tx.R)
	}
	if tx.S != nil {
		cpy.S.Set(tx.S)
	}
	return cpy
}

// accessors for innerTx.
func (tx *SuperlightTx) txType() byte           { return SuperlightTxType }
func (tx *SuperlightTx) chainID() *big.Int      { return tx.ChainID }
func (tx *SuperlightTx) protected() bool        { return true }
func (tx *SuperlightTx) accessList() AccessList { return tx.AccessList }
func (tx *SuperlightTx) data() []byte           { return tx.Data }
func (tx *SuperlightTx) gas() uint64            { return tx.Gas }
func (tx *SuperlightTx) gasFeeCap() *big.Int    { return tx.GasFeeCap }
func (tx *SuperlightTx) gasTipCap() *big.Int    { return tx.GasTipCap }
func (tx *SuperlightTx) gasPrice() *big.Int     { return tx.GasFeeCap }
func (tx *SuperlightTx) value() *big.Int        { return tx.Value }
func (tx *SuperlightTx) nonce() uint64          { return tx.Nonce }
func (tx *SuperlightTx) to() *common.Address    { return tx.To }

func (tx *SuperlightTx) rawSignatureValues() (v, r, s *big.Int) {
	return tx.V, tx.R, tx.S
}

func (tx *SuperlightTx) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.ChainID, tx.V, tx.R, tx.S = chainID, v, r, s
}

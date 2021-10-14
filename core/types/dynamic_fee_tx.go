// Copyright 2021 The go-probeum Authors
// This file is part of the go-probeum library.
//
// The go-probeum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-probeum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-probeum library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"github.com/probeum/go-probeum/common/hexutil"
	"math/big"

	"github.com/probeum/go-probeum/common"
)

type DynamicFeeTx struct {
	ChainID    *big.Int
	Nonce      uint64
	GasTipCap  *big.Int
	GasFeeCap  *big.Int
	Gas        uint64
	To         *common.Address `rlp:"nil"` // nil means contract creation
	BizType    uint8
	Value      *big.Int
	Data       []byte
	AccessList AccessList
	K          byte `json:"k" gencodec:"required"`
	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`

	From      *common.Address `rlp:"nil"`
	Owner     *common.Address `rlp:"nil"`
	Loss      *common.Address `rlp:"nil"`
	Asset     *common.Address `rlp:"nil"`
	Old       *common.Address `rlp:"nil"`
	New       *common.Address `rlp:"nil"`
	Initiator *common.Address `rlp:"nil"`
	Receiver  *common.Address `rlp:"nil"`
	Value2    *big.Int
	Mark      []byte
	Height    *big.Int
	AccType   *hexutil.Uint8
	LossType  *hexutil.Uint8
	PnsType   *hexutil.Uint8
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *DynamicFeeTx) copy() TxData {
	cpy := &DynamicFeeTx{
		Nonce: tx.Nonce,
		To:    tx.To,
		From:  tx.From,
		Data:  common.CopyBytes(tx.Data),
		Gas:   tx.Gas,
		// These are copied below.
		AccessList: make(AccessList, len(tx.AccessList)),
		Value:      new(big.Int),
		ChainID:    new(big.Int),
		GasTipCap:  new(big.Int),
		GasFeeCap:  new(big.Int),
		V:          new(big.Int),
		R:          new(big.Int),
		S:          new(big.Int),
		K:          tx.K,
		AccType:    tx.AccType,
		BizType:    tx.BizType,
		LossType:   tx.LossType,
		PnsType:    tx.PnsType,
		New:        tx.New,
		Old:        tx.Old,
		Loss:       tx.Loss,
		Receiver:   tx.Receiver,
		Mark:       common.CopyBytes(tx.Mark),
		Height:     tx.Height,
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
func (tx *DynamicFeeTx) txType() byte           { return DynamicFeeTxType }
func (tx *DynamicFeeTx) chainID() *big.Int      { return tx.ChainID }
func (tx *DynamicFeeTx) protected() bool        { return true }
func (tx *DynamicFeeTx) accessList() AccessList { return tx.AccessList }
func (tx *DynamicFeeTx) data() []byte           { return tx.Data }
func (tx *DynamicFeeTx) gas() uint64            { return tx.Gas }
func (tx *DynamicFeeTx) gasFeeCap() *big.Int    { return tx.GasFeeCap }
func (tx *DynamicFeeTx) gasTipCap() *big.Int    { return tx.GasTipCap }
func (tx *DynamicFeeTx) gasPrice() *big.Int     { return tx.GasFeeCap }
func (tx *DynamicFeeTx) value() *big.Int        { return tx.Value }
func (tx *DynamicFeeTx) nonce() uint64          { return tx.Nonce }
func (tx *DynamicFeeTx) to() *common.Address    { return tx.To }
func (tx *DynamicFeeTx) bizType() uint8         { return tx.BizType }

func (tx *DynamicFeeTx) from() *common.Address      { return tx.From }
func (tx *DynamicFeeTx) owner() *common.Address     { return tx.Owner }
func (tx *DynamicFeeTx) loss() *common.Address      { return tx.Loss }
func (tx *DynamicFeeTx) asset() *common.Address     { return tx.Asset }
func (tx *DynamicFeeTx) old() *common.Address       { return tx.Old }
func (tx *DynamicFeeTx) new() *common.Address       { return tx.New }
func (tx *DynamicFeeTx) initiator() *common.Address { return tx.Initiator }
func (tx *DynamicFeeTx) receiver() *common.Address  { return tx.Receiver }
func (tx *DynamicFeeTx) value2() *big.Int           { return tx.Value2 }
func (tx *DynamicFeeTx) height() *big.Int           { return tx.Height }
func (tx *DynamicFeeTx) mark() []byte               { return tx.Mark }
func (tx *DynamicFeeTx) accType() *hexutil.Uint8    { return tx.AccType }
func (tx *DynamicFeeTx) lossType() *hexutil.Uint8   { return tx.LossType }
func (tx *DynamicFeeTx) pnsType() *hexutil.Uint8    { return tx.PnsType }
func (tx *DynamicFeeTx) rawSignatureValues() (k byte, v, r, s *big.Int) {
	return tx.K, tx.V, tx.R, tx.S
}

func (tx *DynamicFeeTx) setSignatureValues(k byte, chainID, v, r, s *big.Int) {
	tx.ChainID, tx.V, tx.R, tx.S = chainID, v, r, s
	tx.K = k
}

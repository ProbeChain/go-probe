// Copyright 2020 The go-probeum Authors
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

// LegacyTx is the transaction data of regular Probeum transactions.
type LegacyTx struct {
	Nonce    uint64          // nonce of sender account
	GasPrice *big.Int        // wei per gas
	Gas      uint64          // gas limit
	To       *common.Address `rlp:"nil"` // nil means contract creation
	Value    *big.Int        // wei amount
	BizType  uint8
	Data     []byte   // contract invocation input data
	V, R, S  *big.Int // signature values

	From      *common.Address `rlp:"nil"`
	Owner     *common.Address `rlp:"nil"`
	Vote      *common.Address `rlp:"nil"`
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

// NewTransaction creates an unsigned legacy transaction.
// Deprecated: use NewTx instead.
func NewTransaction(nonce uint64, to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	return NewTx(&LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Value:    amount,
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})
}

// NewContractCreation creates an unsigned legacy transaction.
// Deprecated: use NewTx instead.
func NewContractCreation(nonce uint64, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	return NewTx(&LegacyTx{
		Nonce:    nonce,
		Value:    amount,
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *LegacyTx) copy() TxData {
	cpy := &LegacyTx{
		Nonce: tx.Nonce,
		From:  tx.From,
		To:    tx.To,
		Data:  common.CopyBytes(tx.Data),
		Gas:   tx.Gas,
		// These are initialized below.
		Value:    new(big.Int),
		GasPrice: new(big.Int),
		V:        new(big.Int),
		R:        new(big.Int),
		S:        new(big.Int),
		AccType:  tx.AccType,
		LossType: tx.LossType,
		PnsType:  tx.PnsType,
		BizType:  tx.BizType,
		New:      tx.New,
		Old:      tx.Old,
		Loss:     tx.Loss,
		Receiver: tx.Receiver,
		Mark:     common.CopyBytes(tx.Mark),
		Height:   tx.Height,
	}
	if tx.Value != nil {
		cpy.Value.Set(tx.Value)
	}
	if tx.GasPrice != nil {
		cpy.GasPrice.Set(tx.GasPrice)
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
func (tx *LegacyTx) txType() byte           { return LegacyTxType }
func (tx *LegacyTx) chainID() *big.Int      { return deriveChainId(tx.V) }
func (tx *LegacyTx) accessList() AccessList { return nil }
func (tx *LegacyTx) data() []byte           { return tx.Data }
func (tx *LegacyTx) gas() uint64            { return tx.Gas }
func (tx *LegacyTx) gasPrice() *big.Int     { return tx.GasPrice }
func (tx *LegacyTx) gasTipCap() *big.Int    { return tx.GasPrice }
func (tx *LegacyTx) gasFeeCap() *big.Int    { return tx.GasPrice }
func (tx *LegacyTx) value() *big.Int        { return tx.Value }
func (tx *LegacyTx) nonce() uint64          { return tx.Nonce }
func (tx *LegacyTx) to() *common.Address    { return tx.To }
func (tx *LegacyTx) bizType() uint8         { return tx.BizType }

func (tx *LegacyTx) from() *common.Address        { return tx.From }
func (tx *LegacyTx) setFrom(from *common.Address) { tx.From = from }
func (tx *LegacyTx) owner() *common.Address       { return tx.Owner }
func (tx *LegacyTx) vote() *common.Address        { return tx.Vote }
func (tx *LegacyTx) loss() *common.Address        { return tx.Loss }
func (tx *LegacyTx) asset() *common.Address       { return tx.Asset }
func (tx *LegacyTx) old() *common.Address         { return tx.Old }
func (tx *LegacyTx) new() *common.Address         { return tx.New }
func (tx *LegacyTx) initiator() *common.Address   { return tx.Initiator }
func (tx *LegacyTx) receiver() *common.Address    { return tx.Receiver }
func (tx *LegacyTx) value2() *big.Int             { return tx.Value2 }
func (tx *LegacyTx) height() *big.Int             { return tx.Height }
func (tx *LegacyTx) mark() []byte                 { return tx.Mark }
func (tx *LegacyTx) accType() *hexutil.Uint8      { return tx.AccType }
func (tx *LegacyTx) lossType() *hexutil.Uint8     { return tx.LossType }
func (tx *LegacyTx) pnsType() *hexutil.Uint8      { return tx.PnsType }

func (tx *LegacyTx) rawSignatureValues() (v, r, s *big.Int) {
	return tx.V, tx.R, tx.S
}

func (tx *LegacyTx) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.V, tx.R, tx.S = v, r, s
}

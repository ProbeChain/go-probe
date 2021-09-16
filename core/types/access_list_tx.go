// Copyright 2020 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

//go:generate gencodec -type AccessTuple -out gen_access_tuple.go

// AccessList is an EIP-2930 access list.
type AccessList []AccessTuple

// AccessTuple is the element type of an access list.
type AccessTuple struct {
	Address     common.Address `json:"address"        gencodec:"required"`
	StorageKeys []common.Hash  `json:"storageKeys"    gencodec:"required"`
}

// StorageKeys returns the total number of storage keys in the access list.
func (al AccessList) StorageKeys() int {
	sum := 0
	for _, tuple := range al {
		sum += len(tuple.StorageKeys)
	}
	return sum
}

// AccessListTx is the data of EIP-2930 access list transactions.
type AccessListTx struct {
	ChainID    *big.Int        // destination chain ID
	Nonce      uint64          // nonce of sender account
	GasPrice   *big.Int        // wei per gas
	Gas        uint64          // gas limit
	To         *common.Address `rlp:"nil"` // nil means contract creation
	BizType    uint8
	Value      *big.Int   // wei amount
	Data       []byte     // contract invocation input data
	AccessList AccessList // EIP-2930 access list
	K          byte
	V, R, S    *big.Int // signature values

	From        *common.Address `rlp:"nil"`
	Owner       *common.Address `rlp:"nil"`
	Beneficiary *common.Address `rlp:"nil"`
	Vote        *common.Address `rlp:"nil"`
	Loss        *common.Address `rlp:"nil"`
	Asset       *common.Address `rlp:"nil"`
	Old         *common.Address `rlp:"nil"`
	New         *common.Address `rlp:"nil"`
	Initiator   *common.Address `rlp:"nil"`
	Receiver    *common.Address `rlp:"nil"`
	Value2      *big.Int
	Mark        []byte
	InfoDigest  []byte
	Height      uint64
	AccType     *hexutil.Uint8
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *AccessListTx) copy() TxData {
	cpy := &AccessListTx{
		Nonce:   tx.Nonce,
		To:      tx.To,
		New:     tx.New,
		Data:    common.CopyBytes(tx.Data),
		BizType: tx.BizType,
		Gas:     tx.Gas,
		// These are copied below.
		AccessList: make(AccessList, len(tx.AccessList)),
		Value:      new(big.Int),
		ChainID:    new(big.Int),
		GasPrice:   new(big.Int),
		V:          new(big.Int),
		R:          new(big.Int),
		S:          new(big.Int),
		K:          tx.K,
		AccType:    tx.AccType,
	}
	copy(cpy.AccessList, tx.AccessList)
	if tx.Value != nil {
		cpy.Value.Set(tx.Value)
	}
	if tx.ChainID != nil {
		cpy.ChainID.Set(tx.ChainID)
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
func (tx *AccessListTx) txType() byte           { return AccessListTxType }
func (tx *AccessListTx) chainID() *big.Int      { return tx.ChainID }
func (tx *AccessListTx) protected() bool        { return true }
func (tx *AccessListTx) accessList() AccessList { return tx.AccessList }
func (tx *AccessListTx) data() []byte           { return tx.Data }
func (tx *AccessListTx) gas() uint64            { return tx.Gas }
func (tx *AccessListTx) gasPrice() *big.Int     { return tx.GasPrice }
func (tx *AccessListTx) gasTipCap() *big.Int    { return tx.GasPrice }
func (tx *AccessListTx) gasFeeCap() *big.Int    { return tx.GasPrice }
func (tx *AccessListTx) value() *big.Int        { return tx.Value }
func (tx *AccessListTx) nonce() uint64          { return tx.Nonce }
func (tx *AccessListTx) to() *common.Address    { return tx.To }
func (tx *AccessListTx) bizType() uint8         { return tx.BizType }

func (tx *AccessListTx) from() *common.Address        { return tx.From }
func (tx *AccessListTx) owner() *common.Address       { return tx.Owner }
func (tx *AccessListTx) beneficiary() *common.Address { return tx.Beneficiary }
func (tx *AccessListTx) vote() *common.Address        { return tx.Vote }
func (tx *AccessListTx) loss() *common.Address        { return tx.Loss }
func (tx *AccessListTx) asset() *common.Address       { return tx.Asset }
func (tx *AccessListTx) old() *common.Address         { return tx.Old }
func (tx *AccessListTx) new() *common.Address         { return tx.New }
func (tx *AccessListTx) initiator() *common.Address   { return tx.Initiator }
func (tx *AccessListTx) receiver() *common.Address    { return tx.Receiver }
func (tx *AccessListTx) value2() *big.Int             { return tx.Value2 }
func (tx *AccessListTx) height() uint64               { return tx.Height }
func (tx *AccessListTx) mark() []byte                 { return tx.Mark }
func (tx *AccessListTx) infoDigest() []byte           { return tx.InfoDigest }
func (tx *AccessListTx) accType() *hexutil.Uint8      { return tx.AccType }

func (tx *AccessListTx) rawSignatureValues() (k byte, v, r, s *big.Int) {
	return tx.K, tx.V, tx.R, tx.S
}

func (tx *AccessListTx) setSignatureValues(k byte, chainID, v, r, s *big.Int) {
	tx.ChainID, tx.V, tx.R, tx.S = chainID, v, r, s
	tx.K = k
}

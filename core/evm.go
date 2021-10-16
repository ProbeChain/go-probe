// Copyright 2016 The go-probeum Authors
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

package core

import (
	"github.com/probeum/go-probeum/core/types"
	"math/big"

	"github.com/probeum/go-probeum/common"
	"github.com/probeum/go-probeum/consensus"
	"github.com/probeum/go-probeum/core/vm"
)

// ChainContext supports retrieving headers and consensus parameters from the
// current blockchain to be used during transaction processing.
type ChainContext interface {
	// Engine retrieves the chain's consensus engine.
	Engine() consensus.Engine

	// GetHeader returns the hash corresponding to their hash.
	GetHeader(common.Hash, uint64) *types.Header
}

// NewEVMBlockContext creates a new context for use in the EVM.
func NewEVMBlockContext(header *types.Header, chain ChainContext, author *common.Address) vm.BlockContext {
	var (
		beneficiary common.Address
		baseFee     *big.Int
	)

	// If we don't have an explicit author (i.e. not mining), extract from the header
	if author == nil {
		beneficiary, _ = chain.Engine().Author(header) // Ignore error, we're past header validation
	} else {
		beneficiary = *author
	}
	if header.BaseFee != nil {
		baseFee = new(big.Int).Set(header.BaseFee)
	}
	return vm.BlockContext{
		CanTransfer:      CanTransfer,
		GetHash:          GetHashFn(header, chain),
		Coinbase:         beneficiary,
		BlockNumber:      new(big.Int).Set(header.Number),
		Time:             new(big.Int).SetUint64(header.Time),
		Difficulty:       new(big.Int).Set(header.Difficulty),
		BaseFee:          baseFee,
		GasLimit:         header.GasLimit,
		ContractTransfer: ContractTransfer,
		CallDB:           CallDB,
	}
}

// NewEVMTxContext creates a new transaction context for a single transaction.
func NewEVMTxContext(msg Message) vm.TxContext {
	return vm.TxContext{
		Origin:   msg.From(),
		GasPrice: new(big.Int).Set(msg.GasPrice()),

		From:      msg.From(),
		To:        msg.To(),
		Owner:     msg.Owner(),
		Loss:      msg.Loss(),
		Asset:     msg.Asset(),
		Old:       msg.Old(),
		New:       msg.New(),
		Initiator: msg.Initiator(),
		Receiver:  msg.Receiver(),

		BizType:  msg.BizType(),
		Value:    msg.Value(),
		Value2:   msg.Value2(),
		Height:   msg.Height(),
		Data:     msg.Data(),
		Mark:     msg.Mark(),
		AccType:  msg.AccType(),
		LossType: msg.LossType(),
		PnsType:  msg.PnsType(),
	}
}

// GetHashFn returns a GetHashFunc which retrieves header hashes by number
func GetHashFn(ref *types.Header, chain ChainContext) func(n uint64) common.Hash {
	// Cache will initially contain [refHash.parent],
	// Then fill up with [refHash.p, refHash.pp, refHash.ppp, ...]
	var cache []common.Hash

	return func(n uint64) common.Hash {
		// If there's no hash cache yet, make one
		if len(cache) == 0 {
			cache = append(cache, ref.ParentHash)
		}
		if idx := ref.Number.Uint64() - n - 1; idx < uint64(len(cache)) {
			return cache[idx]
		}
		// No luck in the cache, but we can start iterating from the last element we already know
		lastKnownHash := cache[len(cache)-1]
		lastKnownNumber := ref.Number.Uint64() - uint64(len(cache))

		for {
			header := chain.GetHeader(lastKnownHash, lastKnownNumber)
			if header == nil {
				break
			}
			cache = append(cache, header.ParentHash)
			lastKnownHash = header.ParentHash
			lastKnownNumber = header.Number.Uint64() - 1
			if n == lastKnownNumber {
				return lastKnownHash
			}
		}
		return common.Hash{}
	}
}

// CanTransfer checks whprobeer there are enough funds in the address' account to make a transfer.
// This does not take the necessary gas in to account to make the transfer valid.
func CanTransfer(db vm.StateDB, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

// ContractTransfer subtracts amount from sender and adds amount to recipient using the given Db
func ContractTransfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
	//fmt.Printf("ContractTransfer, sender:%s,to:%s,amount:%s\n", sender.String(), recipient.String(), amount.String())
	db.SubBalance(sender, amount)
	//db.AddBalance(recipient, amount)
}

//CallDB call database for update operation
func CallDB(db vm.StateDB, blockNumber *big.Int, txContext vm.TxContext) {
	switch txContext.BizType {
	case common.Register:
		db.Register(txContext)
	case common.Cancellation:
		db.Cancellation(txContext)
	case common.Transfer:
		db.Transfer(txContext)
	case common.ExchangeAsset:
		db.ExchangeAsset(txContext)
	case common.ContractCall:
		ContractTransfer(db, txContext.From, *txContext.To, txContext.Value)
	case common.SendLossReport:
		db.SendLossReport(blockNumber, txContext)
	case common.RevealLossReport:
		db.RevealLossReport(blockNumber, txContext)
	case common.TransferLostAccount:
		db.TransferLostAccount(txContext)
	case common.TransferLostAssetAccount:
		db.TransferLostAssetAccount(txContext)
	case common.RemoveLossReport:
		db.RemoveLossReport(txContext)
	case common.RejectLossReport:
		db.RejectLossReport(txContext)
	case common.Vote:
		db.Vote(txContext)
	case common.ApplyToBeDPoSNode:
		db.ApplyToBeDPoSNode(blockNumber, txContext)
	case common.UpdatingVotesOrData:
		db.UpdateDposAccount(txContext.From, *txContext.To, txContext.Data)
	case common.Redemption:
		db.Redemption(txContext)
	case common.ModifyLossType:
		db.ModifyLossType(txContext)
	case common.ModifyPnsOwner:
		db.ModifyPnsOwner(txContext)
	case common.ModifyPnsContent:
		db.ModifyPnsContent(txContext)
	}
}

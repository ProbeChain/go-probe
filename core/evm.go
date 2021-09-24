// Copyright 2016 The go-ethereum Authors
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

package core

import (
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/vm"
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
		CanTransfer:              CanTransfer,
		Transfer:                 Transfer,
		GetHash:                  GetHashFn(header, chain),
		Coinbase:                 beneficiary,
		BlockNumber:              new(big.Int).Set(header.Number),
		Time:                     new(big.Int).SetUint64(header.Time),
		Difficulty:               new(big.Int).Set(header.Difficulty),
		BaseFee:                  baseFee,
		GasLimit:                 header.GasLimit,
		Register:                 Register,
		Cancellation:             Cancellation,
		ContractTransfer:         ContractTransfer,
		SendLossReport:           SendLossReport,
		ApplyToBeDPoSNode:        ApplyToBeDPoSNode,
		Vote:                     Vote,
		Redemption:               Redemption,
		RevealLossReport:         RevealLossReport,
		TransferLostAccount:      TransferLostAccount,
		TransferLostAssetAccount: TransferLostAssetAccount,
		RemoveLossReport:         RemoveLossReport,
		RejectLossReport:         RejectLossReport,
		ExchangeAsset:            ExchangeAsset,
		ModifyLossType:           ModifyLossType,
		ModifyPnsOwner:           ModifyPnsOwner,
		ModifyPnsContent:         ModifyPnsContent,
	}
}

// NewEVMTxContext creates a new transaction context for a single transaction.
func NewEVMTxContext(msg Message) vm.TxContext {
	return vm.TxContext{
		Origin:   msg.From(),
		GasPrice: new(big.Int).Set(msg.GasPrice()),

		From:        msg.From(),
		To:          msg.To(),
		Owner:       msg.Owner(),
		Beneficiary: msg.Beneficiary(),
		Loss:        msg.Loss(),
		Asset:       msg.Asset(),
		Old:         msg.Old(),
		New:         msg.New(),
		Initiator:   msg.Initiator(),
		Receiver:    msg.Receiver(),

		BizType:    msg.BizType(),
		Value:      msg.Value(),
		Value2:     msg.Value2(),
		Height:     msg.Height(),
		Data:       msg.Data(),
		Mark:       msg.Mark(),
		InfoDigest: msg.InfoDigest(),
		AccType:    msg.AccType(),
		LossType:   msg.LossType(),
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

// CanTransfer checks whether there are enough funds in the address' account to make a transfer.
// This does not take the necessary gas in to account to make the transfer valid.
func CanTransfer(db vm.StateDB, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

// Transfer subtracts amount from sender and adds amount to recipient using the given Db
func Transfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
	fmt.Printf("Transfer, sender:%s,to:%s,amount:%s\n", sender.String(), recipient.String(), amount.String())
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}

// ExchangeAsset subtracts amount from sender and adds amount to recipient using the given Db
func ExchangeAsset(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
	fmt.Printf("Transfer, sender:%s,to:%s,amount:%s\n", sender.String(), recipient.String(), amount.String())
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}

// ContractTransfer subtracts amount from sender and adds amount to recipient using the given Db
func ContractTransfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
	fmt.Printf("ContractTransfer, sender:%s,to:%s,amount:%s\n", sender.String(), recipient.String(), amount.String())
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}

func Register(db vm.StateDB, sender common.Address, txContext vm.TxContext) {
	fmt.Printf("Register, sender:%s,new:%s,pledge:%s\n", sender.String(), txContext.New.String(), txContext.Value.String())
	db.SubBalance(sender, txContext.Value)
	db.GenerateAccount(txContext)
}

func Cancellation(db vm.StateDB, senderAccount, newAccount common.Address) {
	balance := db.GetBalance(senderAccount)
	db.SubBalance(senderAccount, balance)
	db.AddBalance(newAccount, balance)
}

func SendLossReport(db vm.StateDB, sender common.Address, amount *big.Int, txContext vm.TxContext) {
	fmt.Printf("SendLossReport, sender:%s,loss:%s,mark:%s,infoDigest:%s\n", sender, txContext.Loss, txContext.Mark, txContext.InfoDigest)
	db.SubBalance(sender, amount)
}

func ApplyToBeDPoSNode(db vm.StateDB, voteAddr common.Address, data []byte) {
	fmt.Printf("ApplyToBeDPoSNode, voteAddr:%s,data:%s\n", voteAddr, data)
	db.UpdateDposAccount(voteAddr, data)
}

// Vote subtracts amount from sender and adds amount to recipient using the given Db
func Vote(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
	fmt.Printf("Vote, sender:%s,to:%s,amount:%s\n", sender.String(), recipient.String(), amount.String())
	db.SubBalance(sender, amount)
	db.SetVoteRecordForRegular(sender, recipient, amount)
	db.AddVote(recipient, amount)
}

// Redemption redemption vote value
func Redemption(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.Redemption(sender, recipient, amount)
}

//RevealLossReport reveal loss report
func RevealLossReport(db vm.StateDB, sender common.Address, txContext vm.TxContext) {

}

//TransferLostAccount transfer lost account value
func TransferLostAccount(db vm.StateDB, sender common.Address, txContext vm.TxContext) {

}

//TransferLostAssetAccount transfer loss asset account value
func TransferLostAssetAccount(db vm.StateDB, sender common.Address, txContext vm.TxContext) {

}

//RemoveLossReport remove loss report
func RemoveLossReport(db vm.StateDB, sender common.Address, txContext vm.TxContext) {

}

//RejectLossReport reject loss report
func RejectLossReport(db vm.StateDB, sender common.Address, txContext vm.TxContext) {

}

//ModifyLossType modify loss type
func ModifyLossType(db vm.StateDB, sender common.Address, amount *big.Int, txContext vm.TxContext) {
	db.SubBalance(sender, amount)
	db.SetLossTypeForRegular(sender, uint8(*txContext.LossType))
}

//ModifyPnsOwner modify pns owner
func ModifyPnsOwner(db vm.StateDB, sender common.Address, txContext vm.TxContext) {

}

//ModifyPnsContent modify pns content
func ModifyPnsContent(db vm.StateDB, sender common.Address, txContext vm.TxContext) {

}

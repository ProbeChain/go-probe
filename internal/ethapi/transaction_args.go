// Copyright 2021 The go-ethereum Authors
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

package ethapi

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// TransactionArgs represents the arguments to construct a new transaction
// or a message call.
type TransactionArgs struct {
	From        *common.Address `json:"from"`
	To          *common.Address `json:"to"`
	Owner       *common.Address `json:"owner"`
	Beneficiary *common.Address `json:"beneficiary"`
	Loss        *common.Address `json:"loss"`
	Asset       *common.Address `json:"asset"`
	Old         *common.Address `json:"old"`
	New         *common.Address `json:"new"`
	Initiator   *common.Address `json:"initiator"`
	Receiver    *common.Address `json:"receiver"`
	BizType     *hexutil.Uint8  `json:"bizType"`

	Gas                  *hexutil.Uint64 `json:"gas"`
	GasPrice             *hexutil.Big    `json:"gasPrice"`
	MaxFeePerGas         *hexutil.Big    `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *hexutil.Big    `json:"maxPriorityFeePerGas"`
	Value                *hexutil.Big    `json:"value"`
	Value2               *hexutil.Big    `json:"value2"`
	Nonce                *hexutil.Uint64 `json:"nonce"`
	Height               *hexutil.Big    `json:"height"`

	Data       *hexutil.Bytes `json:"data"`
	Input      *hexutil.Bytes `json:"input"`
	Mark       *hexutil.Bytes `json:"mark"`
	InfoDigest *hexutil.Bytes `json:"infoDigest"`
	AccType    *hexutil.Uint8 `json:"accType"`
	// For non-legacy transactions
	AccessList *types.AccessList `json:"accessList,omitempty"`
	ChainID    *hexutil.Big      `json:"chainId,omitempty"`
}

// from retrieves the transaction sender address.
func (args *TransactionArgs) from() common.Address {
	if args.From == nil {
		return common.Address{}
	}
	return *args.From
}

// data retrieves the transaction calldata. Input field is preferred.
func (args *TransactionArgs) data() []byte {
	if args.Input != nil {
		return *args.Input
	}
	if args.Data != nil {
		return *args.Data
	}
	return nil
}

func (args *TransactionArgs) mark() []byte {
	if args.Mark != nil {
		return *args.Mark
	}
	return nil
}

func (args *TransactionArgs) infoDigest() []byte {
	if args.InfoDigest != nil {
		return *args.InfoDigest
	}
	return nil
}

func (args *TransactionArgs) value2() *big.Int {
	if args.Value2 != nil {
		return args.Value2.ToInt()
	}
	return nil
}

func (args *TransactionArgs) height() *big.Int {
	if args.Height != nil {
		return args.Height.ToInt()
	}
	return nil
}

// setDefaults fills in default values for unspecified tx fields.
func (args *TransactionArgs) setDefaults(ctx context.Context, b Backend) error {
	if args.GasPrice != nil && (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil) {
		return errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
	}
	// After london, default to 1559 unless gasPrice is set
	head := b.CurrentHeader()
	if b.ChainConfig().IsLondon(head.Number) && args.GasPrice == nil {
		if args.MaxPriorityFeePerGas == nil {
			tip, err := b.SuggestGasTipCap(ctx)
			if err != nil {
				return err
			}
			args.MaxPriorityFeePerGas = (*hexutil.Big)(tip)
		}
		if args.MaxFeePerGas == nil {
			gasFeeCap := new(big.Int).Add(
				(*big.Int)(args.MaxPriorityFeePerGas),
				new(big.Int).Mul(head.BaseFee, big.NewInt(2)),
			)
			args.MaxFeePerGas = (*hexutil.Big)(gasFeeCap)
		}
		if args.MaxFeePerGas.ToInt().Cmp(args.MaxPriorityFeePerGas.ToInt()) < 0 {
			return fmt.Errorf("maxFeePerGas (%v) < maxPriorityFeePerGas (%v)", args.MaxFeePerGas, args.MaxPriorityFeePerGas)
		}
	} else {
		if args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil {
			return errors.New("maxFeePerGas or maxPriorityFeePerGas specified but london is not active yet")
		}
		if args.GasPrice == nil {
			price, err := b.SuggestGasTipCap(ctx)
			if err != nil {
				return err
			}
			if b.ChainConfig().IsLondon(head.Number) {
				price.Add(price, head.BaseFee)
			}
			args.GasPrice = (*hexutil.Big)(price)
		}
	}
	var err error
	switch uint8(*args.BizType) {
	case common.Register:
		err = args.setDefaultsOfRegister(ctx, b)
	case common.Cancellation:
		err = args.setDefaultsOfCancellation(ctx, b)
	case common.RevokeCancellation:
		err = args.setDefaultsOfRevokeCancellation(ctx, b)
	case common.Transfer:
		err = args.setDefaultsOfTransfer(ctx, b)
	case common.ContractCall:
		err = args.setDefaultsOfContractCall(ctx, b)
	case common.SendLossReport:
		return args.setDefaultsOfSendLossReport(ctx, b)
	case common.Vote:
		return args.setDefaultsOfVote(ctx, b)
	//... todo 还有未实现的
	default:
		err = errors.New("unsupported business type")
	}
	if err != nil {
		log.Error("set defaults err ", err)
		return err
	}
	if args.ChainID == nil {
		id := (*hexutil.Big)(b.ChainConfig().ChainID)
		args.ChainID = id
	}
	return nil
}

// ToMessage converts th transaction arguments to the Message type used by the
// core evm. This method is used in calls and traces that do not require a real
// live transaction.
func (args *TransactionArgs) ToMessage(globalGasCap uint64, baseFee *big.Int) (types.Message, error) {
	// Reject invalid combinations of pre- and post-1559 fee styles
	if args.GasPrice != nil && (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil) {
		return types.Message{}, errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
	}
	// Set sender address or use zero address if none specified.
	addr := args.from()

	// Set default gas & gas price if none were set
	gas := globalGasCap
	if gas == 0 {
		gas = uint64(math.MaxUint64 / 2)
	}
	if args.Gas != nil {
		gas = uint64(*args.Gas)
	}
	if globalGasCap != 0 && globalGasCap < gas {
		log.Warn("Caller gas above allowance, capping", "requested", gas, "cap", globalGasCap)
		gas = globalGasCap
	}
	var (
		gasPrice  *big.Int
		gasFeeCap *big.Int
		gasTipCap *big.Int
	)
	if baseFee == nil {
		// If there's no basefee, then it must be a non-1559 execution
		gasPrice = new(big.Int)
		if args.GasPrice != nil {
			gasPrice = args.GasPrice.ToInt()
		}
		gasFeeCap, gasTipCap = gasPrice, gasPrice
	} else {
		// A basefee is provided, necessitating 1559-type execution
		if args.GasPrice != nil {
			// User specified the legacy gas field, convert to 1559 gas typing
			gasPrice = args.GasPrice.ToInt()
			gasFeeCap, gasTipCap = gasPrice, gasPrice
		} else {
			// User specified 1559 gas feilds (or none), use those
			gasFeeCap = new(big.Int)
			if args.MaxFeePerGas != nil {
				gasFeeCap = args.MaxFeePerGas.ToInt()
			}
			gasTipCap = new(big.Int)
			if args.MaxPriorityFeePerGas != nil {
				gasTipCap = args.MaxPriorityFeePerGas.ToInt()
			}
			// Backfill the legacy gasPrice for EVM execution, unless we're all zeroes
			gasPrice = new(big.Int)
			if gasFeeCap.BitLen() > 0 || gasTipCap.BitLen() > 0 {
				gasPrice = math.BigMin(new(big.Int).Add(gasTipCap, baseFee), gasFeeCap)
			}
		}
	}
	value := new(big.Int)
	if args.Value != nil {
		value = args.Value.ToInt()
	}
	data := args.data()
	var accessList types.AccessList
	if args.AccessList != nil {
		accessList = *args.AccessList
	}
	msg := types.NewMessage(
		addr, args.To, uint8(*args.BizType),
		0, value, gas,
		gasPrice, gasFeeCap, gasTipCap,
		data, accessList, false,
		args.Owner, args.Beneficiary,
		args.Loss, args.Asset,
		args.Old, args.New, args.Initiator,
		args.Receiver, args.mark(), args.infoDigest(),
		args.value2(), args.height(), args.AccType)
	return msg, nil
}

// toTransaction converts the arguments to a transaction.
// This assumes that setDefaults has been called.
func (args *TransactionArgs) toTransaction() *types.Transaction {
	switch uint8(*args.BizType) {
	case common.Register:
		return args.transactionOfRegister()
	case common.Cancellation:
		return args.transactionOfCancellation()
	case common.RevokeCancellation:
		return args.transactionOfRevokeCancellation()
	case common.Transfer:
		return args.transactionOfTransfer()
	case common.ContractCall:
		return args.transactionOfContractCall()
	case common.SendLossReport:
		return args.transactionOfSendLossReport()
	case common.Vote:
		return args.transactionOfVote()
	//... todo 还有未实现的
	default:
		return nil
	}
}

// ToTransaction converts the arguments to a transaction.
// This assumes that setDefaults has been called.
func (args *TransactionArgs) ToTransaction() *types.Transaction {
	return args.toTransaction()
}

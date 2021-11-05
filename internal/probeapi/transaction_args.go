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

package probeapi

import (
	"context"
	"errors"
	"fmt"
	"github.com/probeum/go-probeum/common"
	"github.com/probeum/go-probeum/common/hexutil"
	"github.com/probeum/go-probeum/common/math"
	"github.com/probeum/go-probeum/core/types"
	"github.com/probeum/go-probeum/log"
	"math/big"
)

// TransactionArgs represents the arguments to construct a new transaction
// or a message call.
type TransactionArgs struct {
	From                 *common.Address   `json:"from"`
	To                   *common.Address   `json:"to"`
	BizType              byte              `json:"bizType"`
	Gas                  *hexutil.Uint64   `json:"gas"`
	GasPrice             *hexutil.Big      `json:"gasPrice"`
	MaxFeePerGas         *hexutil.Big      `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *hexutil.Big      `json:"maxPriorityFeePerGas"`
	Value                *hexutil.Big      `json:"value"`
	Nonce                *hexutil.Uint64   `json:"nonce"`
	Data                 *hexutil.Bytes    `json:"data"`
	Input                *hexutil.Bytes    `json:"input"`
	AccessList           *types.AccessList `json:"accessList,omitempty"`
	ChainID              *hexutil.Big      `json:"chainId,omitempty"`
	ExtArgs              []byte
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
	if args.To == nil {
		args.BizType = common.CONTRACT_DEPLOY
		err = args.setDefaultsOfContractCall(ctx, b)
	} else {
		switch args.To.String() {
		case common.SPECIAL_ADDRESS_FOR_REGISTER_PNS:
			args.BizType = common.REGISTER_PNS
			err = args.setDefaultsOfRegisterPns(ctx, b)
		case common.SPECIAL_ADDRESS_FOR_REGISTER_AUTHORIZE:
			args.BizType = common.REGISTER_AUTHORIZE
			err = args.setDefaultsOfRegisterAuthorize(ctx, b)
		case common.SPECIAL_ADDRESS_FOR_REGISTER_LOSE:
			args.BizType = common.REGISTER_LOSE
			err = args.setDefaultsOfRegisterLose(ctx, b)
		case common.SPECIAL_ADDRESS_FOR_CANCELLATION:
			args.BizType = common.CANCELLATION
			err = args.setDefaultsOfCancellation(ctx, b)
		case common.SPECIAL_ADDRESS_FOR_SEND_LOSS_REPORT:
			args.BizType = common.SEND_LOSS_REPORT
			err = args.setDefaultsOfSendLossReport(ctx, b)
		case common.SPECIAL_ADDRESS_FOR_REVEAL_LOSS_REPORT:
			args.BizType = common.REVEAL_LOSS_REPORT
			err = args.setDefaultsOfRevealLossReport(ctx, b)
		case common.SPECIAL_ADDRESS_FOR_TRANSFER_LOST_ACCOUNT:
			args.BizType = common.TRANSFER_LOST_ACCOUNT
			err = args.setDefaultsOfTransferLostAccount(ctx, b)
		case common.SPECIAL_ADDRESS_FOR_REMOVE_LOSS_REPORT:
			args.BizType = common.REMOVE_LOSS_REPORT
			err = args.setDefaultsOfRemoveLossReport(ctx, b)
		case common.SPECIAL_ADDRESS_FOR_REJECT_LOSS_REPORT:
			args.BizType = common.REJECT_LOSS_REPORT
			err = args.setDefaultsOfRejectLossReport(ctx, b)
		case common.SPECIAL_ADDRESS_FOR_VOTE:
			args.BizType = common.VOTE
			err = args.setDefaultsOfVote(ctx, b)
		case common.SPECIAL_ADDRESS_FOR_APPLY_TO_BE_DPOS_NODE:
			args.BizType = common.APPLY_TO_BE_DPOS_NODE
			err = args.setDefaultsOfApplyToBeDPoSNode(ctx, b)
		case common.SPECIAL_ADDRESS_FOR_REDEMPTION:
			args.BizType = common.REDEMPTION
			err = args.setDefaultsOfRedemption(ctx, b)
		case common.SPECIAL_ADDRESS_FOR_MODIFY_PNS_OWNER:
			args.BizType = common.MODIFY_PNS_OWNER
			err = args.setDefaultsOfModifyPnsOwner(ctx, b)
		case common.SPECIAL_ADDRESS_FOR_MODIFY_PNS_CONTENT:
			args.BizType = common.MODIFY_PNS_CONTENT
			err = args.setDefaultsOfModifyPnsContent(ctx, b)
		default:
			args.BizType = common.TRANSFER
			err = args.setDefaultsOfTransfer(ctx, b)
		}
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
		addr, args.To, args.BizType,
		0, value, gas,
		gasPrice, gasFeeCap, gasTipCap,
		data, accessList, false, args.ExtArgs)
	return msg, nil
}

// toTransaction converts the arguments to a transaction.
// This assumes that setDefaults has been called.
func (args *TransactionArgs) toTransaction() *types.Transaction {
	var data types.TxData
	switch {
	case args.MaxFeePerGas != nil:
		al := types.AccessList{}
		if args.AccessList != nil {
			al = *args.AccessList
		}
		data = &types.DynamicFeeTx{
			From:       args.From,
			To:         args.To,
			ChainID:    (*big.Int)(args.ChainID),
			Nonce:      uint64(*args.Nonce),
			Gas:        uint64(*args.Gas),
			GasFeeCap:  (*big.Int)(args.MaxFeePerGas),
			GasTipCap:  (*big.Int)(args.MaxPriorityFeePerGas),
			Value:      (*big.Int)(args.Value),
			Data:       args.data(),
			AccessList: al,
			BizType:    args.BizType,
			ExtArgs:    args.ExtArgs,
		}
	case args.AccessList != nil:
		data = &types.AccessListTx{
			From:       args.From,
			To:         args.To,
			ChainID:    (*big.Int)(args.ChainID),
			Nonce:      uint64(*args.Nonce),
			Gas:        uint64(*args.Gas),
			GasPrice:   (*big.Int)(args.GasPrice),
			Value:      (*big.Int)(args.Value),
			Data:       args.data(),
			AccessList: *args.AccessList,
			BizType:    args.BizType,
			ExtArgs:    args.ExtArgs,
		}
	default:
		data = &types.LegacyTx{
			From:     args.From,
			To:       args.To,
			Nonce:    uint64(*args.Nonce),
			Gas:      uint64(*args.Gas),
			GasPrice: (*big.Int)(args.GasPrice),
			Value:    (*big.Int)(args.Value),
			Data:     args.data(),
			BizType:  args.BizType,
			ExtArgs:  args.ExtArgs,
		}
	}

	/*	switch args.BizType {
		case common.REGISTER_PNS:
		case common.REGISTER_AUTHORIZE:
		case common.REGISTER_LOSE:
		case common.CANCELLATION:
		case common.TRANSFER:
		case common.CONTRACT_CALL:
		case common.SEND_LOSS_REPORT:
		case common.REVEAL_LOSS_REPORT:
		case common.TRANSFER_LOST_ACCOUNT:
		case common.REMOVE_LOSS_REPORT:
		case common.REJECT_LOSS_REPORT:
		case common.VOTE:
		case common.APPLY_TO_BE_DPOS_NODE:
		case common.REDEMPTION:
		case common.MODIFY_PNS_OWNER:
		case common.MODIFY_PNS_CONTENT:
		default:
			return nil
		}*/

	return types.NewTx(data)
}

// ToTransaction converts the arguments to a transaction.
// This assumes that setDefaults has been called.
func (args *TransactionArgs) ToTransaction() *types.Transaction {
	return args.toTransaction()
}

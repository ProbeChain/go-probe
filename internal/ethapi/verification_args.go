package ethapi

import (
	"bytes"
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
)

//wxc todo 各种业务类型的默认值设置实现
// setDefaultsOfRegister set default parameters of register business type
func (args *TransactionArgs) setDefaultsOfRegister(ctx context.Context, b Backend) error{
	if args.Nonce == nil {
		nonce, err := b.GetPoolNonce(ctx, args.from())
		if err != nil {
			return err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}

	if args.Value == nil {
		//args.Value = new(hexutil.Big)

	}

/*	_,err := common.ValidAddress(*args.From)
	if err != nil {
		return err
	}*/
	newAccType,err := common.ValidAddress(*args.New)
	if err != nil {
		return err
	}
	args.Value = (*hexutil.Big)(new(big.Int).SetUint64(accounts.AmountOfPledgeForCreateAccount(newAccType)))

	//注册账号 todo
/*	if uint8(*args.BizType) == common.Register {
		_,err := common.ValidAddress(*args.From)
		if err != nil {
			return err
		}
		newAccType,err := common.ValidAddress(*args.New)
		if err != nil {
			return err
		}
		args.Value = (*hexutil.Big)(new(big.Int).SetUint64(accounts.AmountOfPledgeForCreateAccount(newAccType)))
	}*/


/*	if args.Data != nil && args.Input != nil && !bytes.Equal(*args.Data, *args.Input) {
		return errors.New(`both "data" and "input" are set and not equal. Please use "input" to pass transaction call data`)
	}*/
	if args.New == nil {
		return errors.New(`new account is not empty`)
	}
	// Estimate the gas usage if necessary.
	if args.Gas == nil {
		// These fields are immutable during the estimation, safe to
		// pass the pointer directly.
		callArgs := TransactionArgs{
			From:                 args.From,
			New:			  	  args.New,
			BizType:              args.BizType,
			GasPrice:             args.GasPrice,
			MaxFeePerGas:         args.MaxFeePerGas,
			MaxPriorityFeePerGas: args.MaxPriorityFeePerGas,
			Value:                args.Value,
			Data:                 args.Data,
			AccessList:           args.AccessList,
		}
		pendingBlockNr := rpc.BlockNumberOrHashWithNumber(rpc.PendingBlockNumber)
		estimated, err := DoEstimateGas(ctx, b, callArgs, pendingBlockNr, b.RPCGasCap())
		if err != nil {
			return err
		}
		args.Gas = &estimated
		log.Trace("Estimate gas usage automatically", "gas", args.Gas)
	}
	return nil
}

// setDefaultsOfCancellation set default parameters of cancellation business type
func (args *TransactionArgs) setDefaultsOfCancellation(ctx context.Context, b Backend) error{
	return nil
}

// setDefaultsOfRevokeCancellation set default parameters of revoke cancellation business type
func (args *TransactionArgs) setDefaultsOfRevokeCancellation(ctx context.Context, b Backend) error{
	return nil
}

// setDefaultsOfTransfer set default parameters of transfer business type
func (args *TransactionArgs) setDefaultsOfTransfer(ctx context.Context, b Backend) error{
	if args.Nonce == nil {
		nonce, err := b.GetPoolNonce(ctx, args.from())
		if err != nil {
			return err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	if args.Value == nil {
		args.Value = new(hexutil.Big)
	}
	if args.Data != nil && args.Input != nil && !bytes.Equal(*args.Data, *args.Input) {
		return errors.New(`both "data" and "input" are set and not equal. Please use "input" to pass transaction call data`)
	}
	if args.To == nil && len(args.data()) == 0 {
		return errors.New(`contract creation without any data provided`)
	}
	// Estimate the gas usage if necessary.
	if args.Gas == nil {
		// These fields are immutable during the estimation, safe to
		// pass the pointer directly.
		callArgs := TransactionArgs{
			From:                 args.From,
			To:                   args.To,
			BizType:              args.BizType,
			GasPrice:             args.GasPrice,
			MaxFeePerGas:         args.MaxFeePerGas,
			MaxPriorityFeePerGas: args.MaxPriorityFeePerGas,
			Value:                args.Value,
			Data:                 args.Data,
			AccessList:           args.AccessList,
		}
		pendingBlockNr := rpc.BlockNumberOrHashWithNumber(rpc.PendingBlockNumber)
		estimated, err := DoEstimateGas(ctx, b, callArgs, pendingBlockNr, b.RPCGasCap())
		if err != nil {
			return err
		}
		args.Gas = &estimated
		log.Trace("Estimate gas usage automatically", "gas", args.Gas)
	}
	return nil
}

// setDefaultsOfContractCall set default parameters of contract call business type
func (args *TransactionArgs) setDefaultsOfContractCall(ctx context.Context, b Backend) error{
	return nil
}

func (args *TransactionArgs) etDefaultsOfExchangeTransaction(ctx context.Context, b Backend) error{
	return nil
}

func (args *TransactionArgs) setDefaultsOfVotingForAnAccount(ctx context.Context, b Backend) error{
	return nil
}

func (args *TransactionArgs) setDefaultsOfApplyToBeDPoSNode(ctx context.Context, b Backend) error{
	return nil
}

func (args *TransactionArgs) setDefaultsOfUpdatingVotesOrData(ctx context.Context, b Backend) error{
	return nil
}

func (args *TransactionArgs) setDefaultsOfSendLossReport(ctx context.Context, b Backend) error{
	return nil
}

func (args *TransactionArgs) setDefaultsOfRevealLossMessage(ctx context.Context, b Backend) error{
	return nil
}

func (args *TransactionArgs) setDefaultsOfTransferLostAccountWhenTimeOut(ctx context.Context, b Backend) error{
	return nil
}

func (args *TransactionArgs) setDefaultsOfTransferLostAccountWhenConfirmed(ctx context.Context, b Backend) error{
	return nil
}

func (args *TransactionArgs) setDefaultsOfRejectLossReportWhenTimeOut(ctx context.Context, b Backend) error{
	return nil
}




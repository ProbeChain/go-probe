package ethapi

import (
	"bytes"
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

//wxc todo 各种业务类型的默认值设置实现
// setDefaultsOfRegister set default parameters of register business type
func setDefaultsOfRegister(ctx context.Context, b Backend,args *TransactionArgs) error{
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
			BizType:			  args.BizType,
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
func setDefaultsOfCancellation(ctx context.Context, b Backend,args *TransactionArgs) error{
	return nil
}

// setDefaultsOfRevokeCancellation set default parameters of revoke cancellation business type
func setDefaultsOfRevokeCancellation(ctx context.Context, b Backend,args *TransactionArgs) error{
	return nil
}

// setDefaultsOfTransfer set default parameters of transfer business type
func setDefaultsOfTransfer(ctx context.Context, b Backend,args *TransactionArgs) error{
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
			BizType:			  args.BizType,
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
func setDefaultsOfContractCall(ctx context.Context, b Backend,args *TransactionArgs) error{
	return nil
}

func setDefaultsOfExchangeTransaction(ctx context.Context, b Backend,args *TransactionArgs) error{
	return nil
}

func setDefaultsOfVotingForAnAccount(ctx context.Context, b Backend,args *TransactionArgs) error{
	return nil
}

func setDefaultsOfApplyToBeDPoSNode(ctx context.Context, b Backend,args *TransactionArgs) error{
	return nil
}

func setDefaultsOfUpdatingVotesOrData(ctx context.Context, b Backend,args *TransactionArgs) error{
	return nil
}

func setDefaultsOfSendLossReport(ctx context.Context, b Backend,args *TransactionArgs) error{
	return nil
}

func setDefaultsOfRevealLossMessage(ctx context.Context, b Backend,args *TransactionArgs) error{
	return nil
}

func setDefaultsOfTransferLostAccountWhenTimeOut(ctx context.Context, b Backend,args *TransactionArgs) error{
	return nil
}

func setDefaultsOfTransferLostAccountWhenConfirmed(ctx context.Context, b Backend,args *TransactionArgs) error{
	return nil
}

func setDefaultsOfRejectLossReportWhenTimeOut(ctx context.Context, b Backend,args *TransactionArgs) error{
	return nil
}




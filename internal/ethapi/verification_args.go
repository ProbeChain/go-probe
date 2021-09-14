package ethapi

import (
	"bytes"
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto/probe"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
)

//wxc todo 各种业务类型的默认值设置实现
// setDefaultsOfRegister set default parameters of register business type
func (args *TransactionArgs) setDefaultsOfRegister(ctx context.Context, b Backend) error {
	if args.AccType == nil {
		return errors.New(`account type must be specified`)
	}
	if !common.CheckAccType(uint8(*args.AccType)) {
		return accounts.ErrWrongAccountType
	}
	fromAccType, err := common.ValidAddress(*args.From)
	if err != nil {
		return err
	}
	if fromAccType != common.ACC_TYPE_OF_GENERAL {
		return accounts.ErrWrongAccountType
	}
	if args.Nonce == nil {
		nonce, err := b.GetPoolNonce(ctx, args.from())
		if err != nil {
			return err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	accountType := uint8(*args.AccType)
	var newAccount common.Address
	if accountType == common.ACC_TYPE_OF_PNS {
		newAccount, err = probe.CreatePNSAddress(args.from(), *args.Data, accountType)
	} else {
		newAccount, err = probe.CreateAddressForAccountType(args.from(), uint64(*args.Nonce), accountType, new(big.Int).SetUint64(uint64(*args.Nonce)))
	}
	if err != nil {
		return err
	}
	args.New = &newAccount
	if *args.From == *args.New {
		return errors.New("must not equals initiator")
	}
	args.Value = (*hexutil.Big)(new(big.Int).SetUint64(AmountOfPledgeForCreateAccount(accountType)))
	exist := b.Exist(*args.From)
	if !exist {
		return accounts.ErrUnknownAccount
	}
	exist = b.Exist(*args.New)
	if exist {
		return keystore.ErrAccountAlreadyExists
	}

	//todo 挂失账号的参数校验
	if accountType == common.ACC_TYPE_OF_LOSE {
		if args.Loss == nil {
			return errors.New("loss account must be specified")
		}
		if args.Receiver == nil {
			return errors.New("receiver account must be specified")
		}
		if !b.Exist(*args.Loss) {
			return accounts.ErrUnknownAccount
		}
		if !b.Exist(*args.Receiver) {
			return accounts.ErrUnknownAccount
		}
	}

	// Estimate the gas usage if necessary.
	if args.Gas == nil {
		// These fields are immutable during the estimation, safe to
		// pass the pointer directly.
		callArgs := TransactionArgs{
			From:                 args.From,
			BizType:              args.BizType,
			GasPrice:             args.GasPrice,
			MaxFeePerGas:         args.MaxFeePerGas,
			MaxPriorityFeePerGas: args.MaxPriorityFeePerGas,
			Value:                args.Value,
			Data:                 args.Data,
			AccessList:           args.AccessList,
			New:                  args.New,
			AccType:              args.AccType,
			Loss:                 args.Loss,
			Receiver:             args.Receiver,
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
func (args *TransactionArgs) setDefaultsOfCancellation(ctx context.Context, b Backend) error {
	return nil
}

// setDefaultsOfRevokeCancellation set default parameters of revoke cancellation business type
func (args *TransactionArgs) setDefaultsOfRevokeCancellation(ctx context.Context, b Backend) error {
	return nil
}

// setDefaultsOfTransfer set default parameters of transfer business type
func (args *TransactionArgs) setDefaultsOfTransfer(ctx context.Context, b Backend) error {
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
	if args.To == nil {
		return errors.New(`to address is not empty`)
	}

	fromAccType, err := common.ValidAddress(*args.From)
	if err != nil {
		return err
	}
	if fromAccType != common.ACC_TYPE_OF_GENERAL {
		return accounts.ErrNotSupported
	}
	toAccType, err := common.ValidAddress(*args.To)
	if err != nil {
		return err
	}
	if toAccType != common.ACC_TYPE_OF_GENERAL {
		return accounts.ErrNotSupported
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
func (args *TransactionArgs) setDefaultsOfContractCall(ctx context.Context, b Backend) error {
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

func (args *TransactionArgs) etDefaultsOfExchangeTransaction(ctx context.Context, b Backend) error {
	return nil
}

func (args *TransactionArgs) setDefaultsOfVotingForAnAccount(ctx context.Context, b Backend) error {
	return nil
}

func (args *TransactionArgs) setDefaultsOfApplyToBeDPoSNode(ctx context.Context, b Backend) error {
	return nil
}

func (args *TransactionArgs) setDefaultsOfUpdatingVotesOrData(ctx context.Context, b Backend) error {
	return nil
}

func (args *TransactionArgs) setDefaultsOfSendLossReport(ctx context.Context, b Backend) error {
	if args.Mark == nil {
		return errors.New(`mark is not empty`)
	}
	if args.InfoDigest == nil {
		return errors.New(`information digests is not empty`)
	}
	if args.Nonce == nil {
		nonce, err := b.GetPoolNonce(ctx, args.from())
		if err != nil {
			return err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	fromAccType, err := common.ValidAddress(*args.From)
	if err != nil {
		return err
	}
	if fromAccType != common.ACC_TYPE_OF_GENERAL {
		return accounts.ErrWrongAccountType
	}

	if args.New != nil {
		accType, err := common.ValidAddress(*args.New)
		if err != nil {
			return err
		}
		args.AccType = (*hexutil.Uint8)(&accType)
	}
	if args.New == nil && args.AccType != nil {
		if !common.CheckAccType(uint8(*args.AccType)) {
			return accounts.ErrWrongAccountType
		}
		var newAccount common.Address
		var err error
		if uint8(*args.AccType) == common.ACC_TYPE_OF_PNS {
			newAccount, err = probe.CreatePNSAddress(args.from(), *args.Data, uint8(*args.AccType))
		} else {
			newAccount, err = probe.CreateAddressForAccountType(args.from(), uint64(*args.Nonce), uint8(*args.AccType), new(big.Int).SetUint64(uint64(*args.Height)))
		}
		if err != nil {
			return err
		}
		args.New = &newAccount
	}

	if *args.From == *args.New {
		return errors.New("must not equals initiator")
	}
	args.Value = (*hexutil.Big)(new(big.Int).SetUint64(AmountOfPledgeForCreateAccount(uint8(*args.AccType))))
	/*	if args.Data != nil && args.Input != nil && !bytes.Equal(*args.Data, *args.Input) {
		return errors.New(`both "data" and "input" are set and not equal. Please use "input" to pass transaction call data`)
	}*/

	exist := b.Exist(*args.From)
	if !exist {
		return accounts.ErrUnknownAccount
	}

	exist = b.Exist(*args.New)
	if exist {
		return keystore.ErrAccountAlreadyExists
	}

	// Estimate the gas usage if necessary.
	if args.Gas == nil {
		// These fields are immutable during the estimation, safe to
		// pass the pointer directly.
		callArgs := TransactionArgs{
			From:                 args.From,
			New:                  args.New,
			AccType:              args.AccType,
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

func (args *TransactionArgs) setDefaultsOfRevealLossMessage(ctx context.Context, b Backend) error {
	return nil
}

func (args *TransactionArgs) setDefaultsOfTransferLostAccountWhenTimeOut(ctx context.Context, b Backend) error {
	return nil
}

func (args *TransactionArgs) setDefaultsOfTransferLostAccountWhenConfirmed(ctx context.Context, b Backend) error {
	return nil
}

func (args *TransactionArgs) setDefaultsOfRejectLossReportWhenTimeOut(ctx context.Context, b Backend) error {
	return nil
}

// AmountOfPledgeForCreateAccount amount of pledge for create a account
func AmountOfPledgeForCreateAccount(accType byte) uint64 {
	switch accType {
	case common.ACC_TYPE_OF_GENERAL:
		return params.AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_REGULAR
	case common.ACC_TYPE_OF_PNS:
		return params.AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_PNS
	case common.ACC_TYPE_OF_ASSET:
		return params.AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_DIGITAL_ASSET
	case common.ACC_TYPE_OF_CONTRACT:
		return params.AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_CONTRACT
	case common.ACC_TYPE_OF_AUTHORIZE:
		return params.AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_VOTING
	case common.ACC_TYPE_OF_LOSE:
		return params.AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_LOSS_REPORT
	default:
		return 0
	}
}

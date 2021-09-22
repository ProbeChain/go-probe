package ethapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto/probe"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
)

//wxc todo 各种业务类型的默认值设置实现
// setDefaultsOfRegister set default parameters of register business type
func (args *TransactionArgs) setDefaultsOfRegister(ctx context.Context, b Backend) error {
	currentBlockNumber := b.CurrentBlock().Number()
	if args.AccType == nil {
		return errors.New(`account type must be specified`)
	}
	accType := uint8(*args.AccType)
	if !common.CheckRegisterAccType(accType) {
		return accounts.ErrWrongAccountType
	}
	if args.New != nil {
		newAccType, err := common.ValidAddress(*args.New)
		if err != nil {
			return err
		}
		if accType != common.ACC_TYPE_OF_GENERAL {
			return accounts.ErrWrongAccountFormat
		}
		if accType != newAccType {
			return accounts.ErrWrongAccountFormat
		}
	} else {
		if accType == common.ACC_TYPE_OF_GENERAL {
			return errors.New(`regular account must be specified`)
		}
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
	if args.New == nil {
		var newAccount common.Address
		if accType == common.ACC_TYPE_OF_PNS {
			newAccount, err = probe.CreatePNSAddress(args.from(), *args.Data, accType)
		} else {
			newAccount, err = probe.CreateAddressForAccountType(args.from(), uint64(*args.Nonce), accType)
		}
		if err != nil {
			return err
		}
		args.New = &newAccount
	}
	if *args.From == *args.New {
		return errors.New("must not equals initiator")
	}
	pledgeAmount := common.AmountOfPledgeForCreateAccount(accType)
	if accType == common.ACC_TYPE_OF_AUTHORIZE {
		if args.Value == nil || args.Value.ToInt().Sign() < 1 {
			return errors.New(`pledge amount must be specified and greater than 0`)
		} else {
			args.Value = (*hexutil.Big)(new(big.Int).Add(args.Value.ToInt(), new(big.Int).SetUint64(pledgeAmount)))
		}
		if args.Height == nil || args.Height.ToInt().Cmp(currentBlockNumber) < 1 {
			return errors.New(`valid period block number must be specified and greater than current block number`)
		}
	} else {
		args.Value = (*hexutil.Big)(new(big.Int).SetUint64(pledgeAmount))
	}
	exist := b.Exist(*args.From)
	if !exist {
		return accounts.ErrUnknownAccount
	}
	exist = b.Exist(*args.New)
	if exist {
		return keystore.ErrAccountAlreadyExists
	}

	//todo 挂失账号的参数校验
	if accType == common.ACC_TYPE_OF_LOSE {
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
		lossAccType, err := common.ValidAddress(*args.Loss)
		if err != nil {
			return err
		}
		if lossAccType != common.ACC_TYPE_OF_GENERAL {
			return accounts.ErrWrongAccountType
		}
		receiverAccType, err := common.ValidAddress(*args.Receiver)
		if err != nil {
			return err
		}
		if receiverAccType != common.ACC_TYPE_OF_GENERAL {
			return accounts.ErrWrongAccountType
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
			Height:               args.Height,
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
		return errors.New("value is null")
	}
	if args.Value.ToInt().Sign() != 1 {
		return errors.New("value must be greater than 0")
	}
	//if args.Data != nil && args.Input != nil && !bytes.Equal(*args.Data, *args.Input) {
	//	return errors.New(`both "data" and "input" are set and not equal. Please use "input" to pass transaction call data`)
	//}
	if args.To == nil {
		return errors.New(`to address is null`)
	}

	fromAccType, err := common.ValidAddress(*args.From)
	if err != nil {
		return err
	}
	if !common.CheckTransferAccType(fromAccType) {
		return accounts.ErrUnsupportedAccountTransfer
	}
	toAccType, err := common.ValidAddress(*args.To)
	if err != nil {
		return err
	}
	if !common.CheckTransferAccType(toAccType) {
		return accounts.ErrUnsupportedAccountTransfer
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

//setDefaultsOfVote  set default parameters of vote business type
func (args *TransactionArgs) setDefaultsOfVote(ctx context.Context, b Backend) error {
	if args.Nonce == nil {
		nonce, err := b.GetPoolNonce(ctx, args.from())
		if err != nil {
			return err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	if args.Value == nil {
		return errors.New("value is null")
	}
	if args.Value.ToInt().Sign() != 1 {
		return errors.New("value must be greater than 0")
	}
	//if args.Data != nil && args.Input != nil && !bytes.Equal(*args.Data, *args.Input) {
	//	return errors.New(`both "data" and "input" are set and not equal. Please use "input" to pass transaction call data`)
	//}
	if args.To == nil {
		return errors.New(`vote account must be specified`)
	}
	fromAccType, err := common.ValidAddress(*args.From)
	if err != nil {
		return err
	}
	if fromAccType != common.ACC_TYPE_OF_GENERAL {
		return accounts.ErrUnsupportedAccountTransfer
	}
	toAccType, err := common.ValidAddress(*args.To)
	if err != nil {
		return err
	}
	if toAccType != common.ACC_TYPE_OF_AUTHORIZE {
		return accounts.ErrUnsupportedAccountTransfer
	}
	exist := b.Exist(*args.From)
	if !exist {
		return accounts.ErrUnknownAccount
	}
	exist = b.Exist(*args.To)
	if !exist {
		return accounts.ErrUnknownAccount
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

func (args *TransactionArgs) setDefaultsOfApplyToBeDPoSNode(ctx context.Context, b Backend) error {
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
	if *args.BizType != 0x22 {
		return errors.New("Illegal business request")
	}
	if args.To == nil {
		return errors.New("voteAccount Address is missing")
	}

	var dposMap map[string]interface{}
	voteData, err := hexutil.Decode(args.Data.String())
	if err != nil {
		return errors.New("The format of the address data parameter is incorrect,data begin with 0x, eg: 0x7b226970223a223139322e3136382e302e31222c22706f7274223a2231333037227d")
	}
	err = json.Unmarshal(voteData, &dposMap)
	if err != nil {
		return errors.New("The format of the address data parameter is incorrect,data begin with 0x")
	}
	if nil == dposMap["ip"] || nil == dposMap["port"] {
		return errors.New("voteAccount parameter data error ")
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

func (args *TransactionArgs) setDefaultsOfUpdatingVotesOrData(ctx context.Context, b Backend) error {
	if args.Nonce == nil {
		nonce, err := b.GetPoolNonce(ctx, args.from())
		if err != nil {
			return err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	if args.Value == nil {
		return errors.New(" ”Value“ parameter value is missing")
	}
	if args.Data != nil && args.Input != nil && !bytes.Equal(*args.Data, *args.Input) {
		return errors.New(`both "data" and "input" are set and not equal. Please use "input" to pass transaction call data`)
	}
	if *args.BizType != 0x23 {
		return errors.New("Illegal business request")
	}
	if args.To == nil {
		return errors.New("voteAccount Address is missing")
	}

	var dposMap map[string]interface{}
	voteData, err := hexutil.Decode(args.Data.String())
	if err != nil {
		return errors.New("The format of the address data parameter is incorrect,data begin with 0x, eg: 0x7b226970223a223139322e3136382e302e31222c22706f7274223a2231333037227d")
	}
	err = json.Unmarshal(voteData, &dposMap)
	if err != nil {
		return errors.New("The format of the address data parameter is incorrect,data begin with 0x")
	}
	if nil == dposMap["ip"] || nil == dposMap["port"] {
		return errors.New("voteAccount parameter data error ")
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

func (args *TransactionArgs) setDefaultsOfSendLossReport(ctx context.Context, b Backend) error {
	/*	if args.Loss == nil {
		return errors.New(`loss account must be specified`)
	}*/
	if args.Mark == nil {
		return errors.New(`mark must be specified`)
	}
	if args.InfoDigest == nil {
		return errors.New(`information digests mark must be specified`)
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
	/*	lossAccType, err := common.ValidAddress(*args.Loss)
		if err != nil {
			return err
		}
		if lossAccType != common.ACC_TYPE_OF_LOSE {
			return accounts.ErrWrongAccountType
		}*/
	if args.Value == nil {
		args.Value = new(hexutil.Big)
	}
	//args.Value = (*hexutil.Big)(new(big.Int).SetUint64(AmountOfPledgeForCreateAccount(uint8(*args.AccType))))
	/*	if args.Data != nil && args.Input != nil && !bytes.Equal(*args.Data, *args.Input) {
		return errors.New(`both "data" and "input" are set and not equal. Please use "input" to pass transaction call data`)
	}*/
	if !b.Exist(*args.From) {
		return accounts.ErrUnknownAccount
	}
	/*	if !b.Exist(*args.Loss) {
		return accounts.ErrUnknownAccount
	}*/

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
			Loss:                 args.Loss,
			Mark:                 args.Mark,
			InfoDigest:           args.InfoDigest,
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

package probeapi

import (
	"bytes"
	"context"
	"errors"
	"github.com/probeum/go-probeum/common"
	"github.com/probeum/go-probeum/common/hexutil"
	"github.com/probeum/go-probeum/rlp"

	"github.com/probeum/go-probeum/log"
	"github.com/probeum/go-probeum/rpc"
	"math/big"
)

// setDefaultsOfRegisterPns set default parameters of register business type
func (args *TransactionArgs) setDefaultsOfRegisterPns(ctx context.Context, b Backend) error {
	if err := args.checkNonce(ctx, b); err != nil {
		return err
	}
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	return args.DoEstimateGas(ctx, b)
}

// setDefaultsOfRegisterAuthorize set default parameters of register business type
func (args *TransactionArgs) setDefaultsOfRegisterAuthorize(ctx context.Context, b Backend) error {
	if err := args.checkNonce(ctx, b); err != nil {
		return err
	}
	currentBlockNumber := b.CurrentBlock().Number()
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	dataArgs := new(common.RegisterAuthorizeArgs)
	err := rlp.DecodeBytes(*args.Data, &dataArgs)
	if err != nil {
		return err
	}
	if args.Value == nil || args.Value.ToInt().Sign() < 1 {
		return errors.New(`pledge amount must be specified and greater than 0`)
	}
	if dataArgs.ValidPeriod.Cmp(currentBlockNumber) < 1 {
		return errors.New(`valid period block number must be specified and greater than current block number`)
	}
	return args.DoEstimateGas(ctx, b)
}

// setDefaultsOfRegisterLose set default parameters of register business type
func (args *TransactionArgs) setDefaultsOfRegisterLose(ctx context.Context, b Backend) error {
	if err := args.checkNonce(ctx, b); err != nil {
		return err
	}
	return args.DoEstimateGas(ctx, b)
}

// setDefaultsOfCancellation set default parameters of cancellation business type
func (args *TransactionArgs) setDefaultsOfCancellation(ctx context.Context, b Backend) error {
	if err := args.checkNonce(ctx, b); err != nil {
		return err
	}
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	cancellationArgs := new(common.CancellationArgs)
	err := rlp.DecodeBytes(*args.Data, &cancellationArgs)
	if err != nil {
		return err
	}
	return args.DoEstimateGas(ctx, b)
}

// setDefaultsOfTransfer set default parameters of transfer business type
func (args *TransactionArgs) setDefaultsOfTransfer(ctx context.Context, b Backend) error {
	if err := args.checkNonce(ctx, b); err != nil {
		return err
	}
	if err := common.ValidateNil(args.Value, "value"); err != nil {
		return err
	}
	if args.Value.ToInt().Sign() != 1 {
		return errors.New("value must be greater than 0")
	}
	if err := common.ValidateNil(args.To, "to"); err != nil {
		return err
	}
	return args.DoEstimateGas(ctx, b)
}

// setDefaultsOfContractCall set default parameters of contract call business type
func (args *TransactionArgs) setDefaultsOfContractCall(ctx context.Context, b Backend) error {
	if err := args.checkNonce(ctx, b); err != nil {
		return err
	}
	if args.Data != nil && args.Input != nil && !bytes.Equal(*args.Data, *args.Input) {
		return errors.New(`both "data" and "input" are set and not equal. Please use "input" to pass transaction call data`)
	}
	if args.To == nil && len(args.data()) == 0 {
		return errors.New(`contract creation without any data provided`)
	}
	return args.DoEstimateGas(ctx, b)
}

//setDefaultsOfVote  set default parameters of vote business type
func (args *TransactionArgs) setDefaultsOfVote(ctx context.Context, b Backend) error {
	if err := args.checkNonce(ctx, b); err != nil {
		return err
	}
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	voteArgs := new(common.VoteArgs)
	err := rlp.DecodeBytes(*args.Data, &voteArgs)
	if err != nil {
		return err
	}
	if err := common.ValidateNil(args.Value, "vote value"); err != nil {
		return err
	}
	if args.Value.ToInt().Sign() != 1 {
		return errors.New("value must be greater than 0")
	}
	return args.DoEstimateGas(ctx, b)
}

func (args *TransactionArgs) setDefaultsOfApplyToBeDPoSNode(ctx context.Context, b Backend) error {
	if err := args.checkNonce(ctx, b); err != nil {
		return err
	}
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	applyDPosArgs := new(common.ApplyDPosArgs)
	err := rlp.DecodeBytes(*args.Data, &applyDPosArgs)
	if err != nil {
		return err
	}
	return args.DoEstimateGas(ctx, b)
}

func (args *TransactionArgs) setDefaultsOfSendLossReport(ctx context.Context, b Backend) error {
	if err := args.checkNonce(ctx, b); err != nil {
		return err
	}
	/*	if err := common.ValidateNil(args.Mark, "mark"); err != nil {
			return err
		}
		if err := common.ValidateNil(args.Data, "information digests data"); err != nil {
			return err
		}
		if args.Nonce == nil {
			nonce, err := b.GetPoolNonce(ctx, args.from())
			if err != nil {
				return err
			}
			args.Nonce = (*hexutil.Uint64)(&nonce)
		}
		if err := common.ValidateAccType(args.From, common.ACC_TYPE_OF_REGULAR, "from"); err != nil {
			return err
		}
		if args.Value == nil {
			args.Value = new(hexutil.Big)
		}*/
	return args.DoEstimateGas(ctx, b)
}

func (args *TransactionArgs) setDefaultsOfRevealLossReport(ctx context.Context, b Backend) error {
	if err := args.checkNonce(ctx, b); err != nil {
		return err
	}
	/*	if args.Nonce == nil {
			nonce, err := b.GetPoolNonce(ctx, args.from())
			if err != nil {
				return err
			}
			args.Nonce = (*hexutil.Uint64)(&nonce)
		}
		if err := common.ValidateNil(args.Value, "value"); err != nil {
			return err
		}
		if args.Value.ToInt().Sign() != 1 {
			return errors.New("value must be greater than 0")
		}
		if err := common.ValidateNil(args.Data, "data"); err != nil {
			return err
		}
		if err := common.ValidateNil(args.From, "from account"); err != nil {
			return err
		}
		if err := common.ValidateNil(args.To, "loss account"); err != nil {
			return err
		}
		if err := common.ValidateNil(args.Old, "lost account"); err != nil {
			return err
		}
		if err := common.ValidateNil(args.New, "new account "); err != nil {
			return err
		}
		if err := common.ValidateAccType(args.From, common.ACC_TYPE_OF_REGULAR, "from"); err != nil {
			return err
		}
		if err := common.ValidateAccType(args.To, common.ACC_TYPE_OF_LOSS, "to"); err != nil {
			return err
		}*/

	return args.DoEstimateGas(ctx, b)
}
func (args *TransactionArgs) setDefaultsOfTransferLostAccount(ctx context.Context, b Backend) error {
	if err := args.checkNonce(ctx, b); err != nil {
		return err
	}
	if err := common.ValidateNil(args.From, "from account"); err != nil {
		return err
	}
	if err := common.ValidateNil(args.To, "loss account"); err != nil {
		return err
	}
	/*	if err := common.ValidateAccType(args.From, common.ACC_TYPE_OF_REGULAR, "from"); err != nil {
			return err
		}
		if err := common.ValidateAccType(args.To, common.ACC_TYPE_OF_LOSS, "to"); err != nil {
			return err
		}*/

	return args.DoEstimateGas(ctx, b)
}
func (args *TransactionArgs) setDefaultsOfRemoveLossReport(ctx context.Context, b Backend) error {
	if err := args.checkNonce(ctx, b); err != nil {
		return err
	}
	if err := common.ValidateNil(args.From, "from account"); err != nil {
		return err
	}
	if err := common.ValidateNil(args.To, "loss account"); err != nil {
		return err
	}
	/*	if err := common.ValidateAccType(args.From, common.ACC_TYPE_OF_REGULAR, "from"); err != nil {
			return err
		}
		if err := common.ValidateAccType(args.To, common.ACC_TYPE_OF_LOSS, "to"); err != nil {
			return err
		}*/
	args.Value = (*hexutil.Big)(new(big.Int).SetUint64(common.AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_LOSS))

	return args.DoEstimateGas(ctx, b)
}
func (args *TransactionArgs) setDefaultsOfRejectLossReport(ctx context.Context, b Backend) error {
	if err := args.checkNonce(ctx, b); err != nil {
		return err
	}
	if err := common.ValidateNil(args.From, "from account"); err != nil {
		return err
	}
	if err := common.ValidateNil(args.To, "loss account"); err != nil {
		return err
	}
	/*	if err := common.ValidateAccType(args.From, common.ACC_TYPE_OF_REGULAR, "from"); err != nil {
			return err
		}
		if err := common.ValidateAccType(args.To, common.ACC_TYPE_OF_LOSS, "to"); err != nil {
			return err
		}*/
	args.Value = (*hexutil.Big)(new(big.Int).SetUint64(common.AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_LOSS))

	return args.DoEstimateGas(ctx, b)
}

//setDefaultsOfRedemption  set default parameters of redemption business type
func (args *TransactionArgs) setDefaultsOfRedemption(ctx context.Context, b Backend) error {
	if err := args.checkNonce(ctx, b); err != nil {
		return err
	}
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	voteArgs := new(common.VoteArgs)
	err := rlp.DecodeBytes(*args.Data, &voteArgs)
	if err != nil {
		return err
	}
	return args.DoEstimateGas(ctx, b)
}

func (args *TransactionArgs) setDefaultsOfModifyPnsOwner(ctx context.Context, b Backend) error {
	if err := args.checkNonce(ctx, b); err != nil {
		return err
	}
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	pnsOwnerArgs := new(common.PnsOwnerArgs)
	err := rlp.DecodeBytes(*args.Data, &pnsOwnerArgs)
	if err != nil {
		return err
	}
	return args.DoEstimateGas(ctx, b)
}
func (args *TransactionArgs) setDefaultsOfModifyPnsContent(ctx context.Context, b Backend) error {
	if err := args.checkNonce(ctx, b); err != nil {
		return err
	}
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	pnsContentArgs := new(common.PnsContentArgs)
	err := rlp.DecodeBytes(*args.Data, &pnsContentArgs)
	if err != nil {
		return err
	}
	return args.DoEstimateGas(ctx, b)
}

func (args *TransactionArgs) DoEstimateGas(ctx context.Context, b Backend) error {
	if args.Gas == nil {
		// These fields are immutable during the estimation, safe to
		// pass the pointer directly.
		callArgs := TransactionArgs{
			From:                 args.From,
			To:                   args.To,
			Value:                args.Value,
			GasPrice:             args.GasPrice,
			MaxFeePerGas:         args.MaxFeePerGas,
			MaxPriorityFeePerGas: args.MaxPriorityFeePerGas,
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

func (args *TransactionArgs) checkNonce(ctx context.Context, b Backend) error {
	if args.Nonce == nil {
		nonce, err := b.GetPoolNonce(ctx, args.from())
		if err != nil {
			return err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	return nil
}

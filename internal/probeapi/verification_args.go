package probeapi

import (
	"bytes"
	"context"
	"errors"
	"github.com/probeum/go-probeum/common"
	"github.com/probeum/go-probeum/common/hexutil"
	"github.com/probeum/go-probeum/crypto"
	"github.com/probeum/go-probeum/rlp"

	"github.com/probeum/go-probeum/log"
	"github.com/probeum/go-probeum/rpc"
	"math/big"
)

// setDefaultsOfRegisterPns set default parameters of register business type
func (args *TransactionArgs) setDefaultsOfRegisterPns(ctx context.Context, b Backend) error {
	if args.Nonce == nil {
		nonce, err := b.GetPoolNonce(ctx, args.from())
		if err != nil {
			return err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	var pnsData string
	err := rlp.DecodeBytes(*args.Data, &pnsData)
	if err != nil {
		return err
	}
	var pnsAddress common.Address
	pnsAddress, err = crypto.CreatePNSAddress(args.from(), []byte(pnsData))
	if err != nil {
		return err
	}
	/*	argsBytes, err := rlp.EncodeToBytes(pnsAddress)
		if err != nil {
			return err
		}*/
	args.ExtArgs = pnsAddress.Bytes()
	return args.DoEstimateGas(ctx, b)
}

// setDefaultsOfRegisterAuthorize set default parameters of register business type
func (args *TransactionArgs) setDefaultsOfRegisterAuthorize(ctx context.Context, b Backend) error {
	if args.Nonce == nil {
		nonce, err := b.GetPoolNonce(ctx, args.from())
		if err != nil {
			return err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	currentBlockNumber := b.CurrentBlock().Number()
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	validPeriod := new(big.Int)
	err := rlp.DecodeBytes(*args.Data, &validPeriod)
	if err != nil {
		return err
	}
	if args.Value == nil || args.Value.ToInt().Sign() < 1 {
		return errors.New(`pledge amount must be specified and greater than 0`)
	}
	if validPeriod.Cmp(currentBlockNumber) < 1 {
		return errors.New(`valid period block number must be specified and greater than current block number`)
	}
	var newAccount common.Address
	newAccount, err = crypto.CreateAddressForAccountType(args.from(), uint64(*args.Nonce))
	if err != nil {
		return err
	}
	args.ExtArgs = newAccount.Bytes()
	return args.DoEstimateGas(ctx, b)
}

// setDefaultsOfRegisterLose set default parameters of register business type
func (args *TransactionArgs) setDefaultsOfRegisterLose(ctx context.Context, b Backend) error {
	if args.Nonce == nil {
		nonce, err := b.GetPoolNonce(ctx, args.from())
		if err != nil {
			return err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	var newAccount common.Address
	var err error
	newAccount, err = crypto.CreateAddressForAccountType(args.from(), uint64(*args.Nonce))
	if err != nil {
		return err
	}
	args.ExtArgs = newAccount.Bytes()
	return args.DoEstimateGas(ctx, b)
}

// setDefaultsOfCancellation set default parameters of cancellation business type
func (args *TransactionArgs) setDefaultsOfCancellation(ctx context.Context, b Backend) error {
	if args.Nonce == nil {
		nonce, err := b.GetPoolNonce(ctx, args.from())
		if err != nil {
			return err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	cancellationArgs := new(common.CancellationArgs)
	err := rlp.DecodeBytes(*args.Data, &cancellationArgs)
	if err != nil {
		return err
	}
	if err := common.ValidateNil(cancellationArgs.CancelAddress, "cancel address"); err != nil {
		return err
	}
	if err := common.ValidateNil(cancellationArgs.BeneficiaryAddress, "beneficiary address"); err != nil {
		return err
	}
	return args.DoEstimateGas(ctx, b)
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
	//set contract deploy fee
	if args.To == nil {
		args.Value = (*hexutil.Big)(new(big.Int).SetUint64(common.AmountOfPledgeForCreateAccount(common.ACC_TYPE_OF_CONTRACT)))
	}
	return args.DoEstimateGas(ctx, b)
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
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	var voteAddr common.Address
	err := rlp.DecodeBytes(*args.Data, &voteAddr)
	if err != nil {
		return err
	}
	if err := common.ValidateNil(voteAddr, "vote account"); err != nil {
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
	if args.Nonce == nil {
		nonce, err := b.GetPoolNonce(ctx, args.from())
		if err != nil {
			return err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	applyDPosArgs := new(common.ApplyDPosArgs)
	err := rlp.DecodeBytes(*args.Data, &applyDPosArgs)
	if err != nil {
		return err
	}
	args.Value = new(hexutil.Big)
	if err := common.ValidateNil(applyDPosArgs.VoteAddress, "vote address"); err != nil {
		return err
	}
	if err := common.ValidateNil(applyDPosArgs.NodeInfo, "node info"); err != nil {
		return err
	}

	return args.DoEstimateGas(ctx, b)
}

func (args *TransactionArgs) setDefaultsOfSendLossReport(ctx context.Context, b Backend) error {
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
		if err := common.ValidateAccType(args.From, common.ACC_TYPE_OF_GENERAL, "from"); err != nil {
			return err
		}
		if args.Value == nil {
			args.Value = new(hexutil.Big)
		}*/
	return args.DoEstimateGas(ctx, b)
}

func (args *TransactionArgs) setDefaultsOfRevealLossReport(ctx context.Context, b Backend) error {
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
		if err := common.ValidateAccType(args.From, common.ACC_TYPE_OF_GENERAL, "from"); err != nil {
			return err
		}
		if err := common.ValidateAccType(args.To, common.ACC_TYPE_OF_LOSE, "to"); err != nil {
			return err
		}*/

	return args.DoEstimateGas(ctx, b)
}
func (args *TransactionArgs) setDefaultsOfTransferLostAccount(ctx context.Context, b Backend) error {
	if args.Nonce == nil {
		nonce, err := b.GetPoolNonce(ctx, args.from())
		if err != nil {
			return err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	if err := common.ValidateNil(args.From, "from account"); err != nil {
		return err
	}
	if err := common.ValidateNil(args.To, "loss account"); err != nil {
		return err
	}
	/*	if err := common.ValidateAccType(args.From, common.ACC_TYPE_OF_GENERAL, "from"); err != nil {
			return err
		}
		if err := common.ValidateAccType(args.To, common.ACC_TYPE_OF_LOSE, "to"); err != nil {
			return err
		}*/

	return args.DoEstimateGas(ctx, b)
}
func (args *TransactionArgs) setDefaultsOfRemoveLossReport(ctx context.Context, b Backend) error {
	if args.Nonce == nil {
		nonce, err := b.GetPoolNonce(ctx, args.from())
		if err != nil {
			return err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	if err := common.ValidateNil(args.From, "from account"); err != nil {
		return err
	}
	if err := common.ValidateNil(args.To, "loss account"); err != nil {
		return err
	}
	/*	if err := common.ValidateAccType(args.From, common.ACC_TYPE_OF_GENERAL, "from"); err != nil {
			return err
		}
		if err := common.ValidateAccType(args.To, common.ACC_TYPE_OF_LOSE, "to"); err != nil {
			return err
		}*/
	pledgeAmount := common.AmountOfPledgeForCreateAccount(common.ACC_TYPE_OF_LOSE)
	args.Value = (*hexutil.Big)(new(big.Int).SetUint64(pledgeAmount))

	return args.DoEstimateGas(ctx, b)
}
func (args *TransactionArgs) setDefaultsOfRejectLossReport(ctx context.Context, b Backend) error {
	if args.Nonce == nil {
		nonce, err := b.GetPoolNonce(ctx, args.from())
		if err != nil {
			return err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	if err := common.ValidateNil(args.From, "from account"); err != nil {
		return err
	}
	if err := common.ValidateNil(args.To, "loss account"); err != nil {
		return err
	}
	/*	if err := common.ValidateAccType(args.From, common.ACC_TYPE_OF_GENERAL, "from"); err != nil {
			return err
		}
		if err := common.ValidateAccType(args.To, common.ACC_TYPE_OF_LOSE, "to"); err != nil {
			return err
		}*/
	pledgeAmount := common.AmountOfPledgeForCreateAccount(common.ACC_TYPE_OF_LOSE)
	args.Value = (*hexutil.Big)(new(big.Int).SetUint64(pledgeAmount))

	return args.DoEstimateGas(ctx, b)
}

//setDefaultsOfRedemption  set default parameters of redemption business type
func (args *TransactionArgs) setDefaultsOfRedemption(ctx context.Context, b Backend) error {
	if args.Nonce == nil {
		nonce, err := b.GetPoolNonce(ctx, args.from())
		if err != nil {
			return err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	voteAddr := new(common.Address)
	err := rlp.DecodeBytes(*args.Data, &voteAddr)
	if err != nil {
		return err
	}
	//todo ValidateNil是否可以去掉
	if err := common.ValidateNil(voteAddr, "vote account"); err != nil {
		return err
	}
	return args.DoEstimateGas(ctx, b)
}

func (args *TransactionArgs) setDefaultsOfModifyPnsOwner(ctx context.Context, b Backend) error {
	if args.Nonce == nil {
		nonce, err := b.GetPoolNonce(ctx, args.from())
		if err != nil {
			return err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	pnsOwnerArgs := new(common.PnsOwnerArgs)
	err := rlp.DecodeBytes(*args.Data, &pnsOwnerArgs)
	if err != nil {
		return err
	}
	if err := common.ValidateNil(args.From, "from address"); err != nil {
		return err
	}
	if err := common.ValidateNil(pnsOwnerArgs.PnsAddress, "pns address"); err != nil {
		return err
	}
	if err := common.ValidateNil(pnsOwnerArgs.OwnerAddress, "new owner address"); err != nil {
		return err
	}
	return args.DoEstimateGas(ctx, b)
}
func (args *TransactionArgs) setDefaultsOfModifyPnsContent(ctx context.Context, b Backend) error {
	if args.Nonce == nil {
		nonce, err := b.GetPoolNonce(ctx, args.from())
		if err != nil {
			return err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	pnsContentArgs := new(common.PnsContentArgs)
	err := rlp.DecodeBytes(*args.Data, &pnsContentArgs)
	if err != nil {
		return err
	}
	if err := common.ValidateNil(args.From, "from address"); err != nil {
		return err
	}
	if err := common.ValidateNil(pnsContentArgs.PnsAddress, "pns address"); err != nil {
		return err
	}
	if err := common.ValidateNil(pnsContentArgs.PnsData, "pns content data"); err != nil {
		return err
	}
	if err := common.ValidateNil(pnsContentArgs.PnsType, "pns type"); err != nil {
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
			BizType:              args.BizType,
			GasPrice:             args.GasPrice,
			MaxFeePerGas:         args.MaxFeePerGas,
			MaxPriorityFeePerGas: args.MaxPriorityFeePerGas,
			Data:                 args.Data,
			AccessList:           args.AccessList,
			ExtArgs:              args.ExtArgs,
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

package probeapi

import (
	"bytes"
	"errors"
	"github.com/probeum/go-probeum/common"
	"github.com/probeum/go-probeum/rlp"
)

// setDefaultsOfRegisterPns set default parameters of register business type
func (args *TransactionArgs) setDefaultsOfRegisterPns() error {
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	decode := new(common.StringDecodeType)
	if err := rlp.DecodeBytes(*args.Data, &decode); err != nil {
		return err
	}
	return nil
}

// setDefaultsOfRegisterAuthorize set default parameters of register business type
func (args *TransactionArgs) setDefaultsOfRegisterAuthorize(b Backend) error {
	currentBlockNumber := b.CurrentBlock().Number()
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	decode := new(common.IntDecodeType)
	if err := rlp.DecodeBytes(*args.Data, &decode); err != nil {
		return err
	}
	if args.Value.ToInt().Sign() < 1 {
		return errors.New(`pledge amount must be specified and greater than 0`)
	}
	if decode.Num.Cmp(currentBlockNumber) < 1 {
		return errors.New(`valid period block number must be specified and greater than current block number`)
	}
	return nil
}

// setDefaultsOfCancellation set default parameters of cancellation business type
func (args *TransactionArgs) setDefaultsOfCancellation() error {
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	decode := new(common.CancellationDecodeType)
	if err := rlp.DecodeBytes(*args.Data, &decode); err != nil {
		return err
	}
	return nil
}

// setDefaultsOfTransfer set default parameters of transfer business type
func (args *TransactionArgs) setDefaultsOfTransfer() error {
	if err := common.ValidateNil(args.To, "to"); err != nil {
		return err
	}
	if args.Data != nil && args.Input != nil && !bytes.Equal(*args.Data, *args.Input) {
		return errors.New(`both "data" and "input" are set and not equal. Please use "input" to pass transaction call data`)
	}
	return nil
}

// setDefaultsOfContractCall set default parameters of contract call business type
func (args *TransactionArgs) setDefaultsOfContractDeploy() error {

	if args.Data != nil && args.Input != nil && !bytes.Equal(*args.Data, *args.Input) {
		return errors.New(`both "data" and "input" are set and not equal. Please use "input" to pass transaction call data`)
	}
	if args.To == nil && len(args.data()) == 0 {
		return errors.New(`contract creation without any data provided`)
	}
	return nil
}

//setDefaultsOfVote  set default parameters of vote business type
func (args *TransactionArgs) setDefaultsOfVote() error {
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	decode := new(common.AddressDecodeType)
	if err := rlp.DecodeBytes(*args.Data, &decode); err != nil {
		return err
	}
	if args.Value.ToInt().Sign() != 1 {
		return errors.New("value must be greater than 0")
	}
	return nil
}

func (args *TransactionArgs) setDefaultsOfApplyToBeDPoSNode() error {
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	decode := new(common.ApplyDPosDecodeType)
	if err := rlp.DecodeBytes(*args.Data, &decode); err != nil {
		return err
	}
	return nil
}

func (args *TransactionArgs) setDefaultsOfSendLossReport() error {
	return errors.New("the current version does not support")
}
func (args *TransactionArgs) setDefaultsOfRevealLossReport() error {
	return errors.New("the current version does not support")
}
func (args *TransactionArgs) setDefaultsOfTransferLostAccount() error {
	return errors.New("the current version does not support")
}
func (args *TransactionArgs) setDefaultsOfRemoveLossReport() error {
	return errors.New("the current version does not support")
}
func (args *TransactionArgs) setDefaultsOfRejectLossReport() error {
	return errors.New("the current version does not support")
}

//setDefaultsOfRedemption  set default parameters of redemption business type
func (args *TransactionArgs) setDefaultsOfRedemption() error {
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	decode := new(common.AddressDecodeType)
	if err := rlp.DecodeBytes(*args.Data, &decode); err != nil {
		return err
	}
	return nil
}

func (args *TransactionArgs) setDefaultsOfModifyPnsOwner() error {
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	decode := new(common.PnsOwnerDecodeType)
	if err := rlp.DecodeBytes(*args.Data, &decode); err != nil {
		return err
	}
	return nil
}
func (args *TransactionArgs) setDefaultsOfModifyPnsContent() error {
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	decode := new(common.PnsContentDecodeType)
	if err := rlp.DecodeBytes(*args.Data, &decode); err != nil {
		return err
	}
	return nil
}

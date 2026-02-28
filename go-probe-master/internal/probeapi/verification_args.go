package probeapi

import (
	"bytes"
	"errors"
	"github.com/probeum/go-probeum/common"
	"github.com/probeum/go-probeum/rlp"
)

//setDefaultsOfRegisterPns set default parameters for register pns account
func (args *TransactionArgs) setDefaultsOfRegisterPns() error {
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	return nil
}

//setDefaultsOfRegisterAuthorize set default parameters for register authorize account
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

//setDefaultsOfRegisterLoss set default parameters for register loss report account
func (args *TransactionArgs) setDefaultsOfRegisterLoss() error {
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	decode := new(common.RegisterLossDecodeType)
	if err := rlp.DecodeBytes(*args.Data, &decode); err != nil {
		return err
	}
	return nil
}

//setDefaultsOfCancellation set default parameters for cancellation account
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

//setDefaultsOfTransfer set default parameters for transfer
func (args *TransactionArgs) setDefaultsOfTransfer() error {
	if err := common.ValidateNil(args.To, "to"); err != nil {
		return err
	}
	if args.Data != nil && args.Input != nil && !bytes.Equal(*args.Data, *args.Input) {
		return errors.New(`both "data" and "input" are set and not equal. Please use "input" to pass transaction call data`)
	}
	return nil
}

//setDefaultsOfContractDeploy set default parameters for contract deploy
func (args *TransactionArgs) setDefaultsOfContractDeploy() error {

	if args.Data != nil && args.Input != nil && !bytes.Equal(*args.Data, *args.Input) {
		return errors.New(`both "data" and "input" are set and not equal. Please use "input" to pass transaction call data`)
	}
	if args.To == nil && len(args.data()) == 0 {
		return errors.New(`contract creation without any data provided`)
	}
	return nil
}

//setDefaultsOfVote  set default parameters for vote
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

//setDefaultsOfApplyToBeDPoSNode set default parameters for apply dPoS node
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

//setDefaultsOfRevealLossReport set default parameters for reveal loss reporting
func (args *TransactionArgs) setDefaultsOfRevealLossReport() error {
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	decode := new(common.RevealLossReportDecodeType)
	if err := rlp.DecodeBytes(*args.Data, &decode); err != nil {
		return err
	}
	return nil
}

//setDefaultsOfTargetAddress set default parameters for transfer lost account balance、cancellation/reject loss reporting and redemption vote
func (args *TransactionArgs) setDefaultsOfTargetAddress() error {
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	decode := new(common.AddressDecodeType)
	if err := rlp.DecodeBytes(*args.Data, &decode); err != nil {
		return err
	}
	return nil
}

//setDefaultsOfModifyPnsOwner set default parameters for modify PNS owner
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

//setDefaultsOfModifyPnsContent set default parameters for modify PNS content
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

//setDefaultsOfModifyLossType set default parameters for modify regular account loss type
func (args *TransactionArgs) setDefaultsOfModifyLossType() error {
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	decode := new(common.ByteDecodeType)
	if err := rlp.DecodeBytes(*args.Data, &decode); err != nil {
		return err
	}
	return nil
}

//setDefaultsOfTransferLostAssociatedAccount set default parameters for transfer lost associated account, like PNS,authorize and votes had been cast
func (args *TransactionArgs) setDefaultsOfTransferLostAssociatedAccount() error {
	if err := common.ValidateNil(args.Data, "data"); err != nil {
		return err
	}
	decode := new(common.AssociatedAccountDecodeType)
	if err := rlp.DecodeBytes(*args.Data, &decode); err != nil {
		return err
	}
	return nil
}

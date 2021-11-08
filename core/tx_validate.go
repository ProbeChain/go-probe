package core

import (
	"errors"
	"fmt"
	"github.com/probeum/go-probeum/accounts"
	"github.com/probeum/go-probeum/common"
	"github.com/probeum/go-probeum/core/types"
	"github.com/probeum/go-probeum/crypto"
	"github.com/probeum/go-probeum/log"
	"github.com/probeum/go-probeum/rlp"
	"math/big"
)

// validateTx validate transaction of register business type
func (pool *TxPool) validateTxOfRegister(tx *types.Transaction, local bool) error {
	var sender *common.Address
	var err error
	if sender, err = pool.validateSender(tx, local); err != nil {
		return err
	}
	var newAccount common.Address
	switch tx.To().Hex() {
	case common.SPECIAL_ADDRESS_FOR_REGISTER_PNS:
		newAccount = crypto.CreatePNSAddress(*sender, tx.Data())
	case common.SPECIAL_ADDRESS_FOR_REGISTER_AUTHORIZE:
		newAccount = crypto.CreateAddress(*sender, tx.Nonce())
		dataArgs := new(common.RegisterAuthorizeArgs)
		if err := rlp.DecodeBytes(tx.Data(), &dataArgs); err != nil {
			return err
		}
		if dataArgs.ValidPeriod.Cmp(pool.chain.CurrentBlock().Number()) < 1 {
			return errors.New(`valid period block number must be specified and greater than current block number`)
		}
	case common.SPECIAL_ADDRESS_FOR_REGISTER_LOSE:
		newAccount = crypto.CreateAddress(*sender, tx.Nonce())
	}
	if pool.currentState.Exist(newAccount) {
		return ErrAccountAlreadyExists
	}
	return pool.validateGas(tx, local, sender)
}

func (pool *TxPool) validateTxOfCancellation(tx *types.Transaction, local bool) error {
	var sender *common.Address
	var err error
	if sender, err = pool.validateSender(tx, local); err != nil {
		return err
	}
	args := new(common.CancellationArgs)
	if err := rlp.DecodeBytes(tx.Data(), &args); err != nil {
		return err
	}
	cancelAccount := pool.currentState.GetStateObject(args.CancelAddress)
	if cancelAccount == nil {
		return ErrAccountNotExists
	}
	beneficiaryAccount := pool.currentState.GetStateObject(args.BeneficiaryAddress)
	if beneficiaryAccount == nil {
		return ErrAccountNotExists
	}
	if beneficiaryAccount.AccountType() != common.ACC_TYPE_OF_REGULAR {
		return errors.New("the beneficiary must be a regular account")
	}
	switch cancelAccount.AccountType() {
	case common.ACC_TYPE_OF_REGULAR:
		if cancelAccount.RegularAccount().VoteValue.Sign() > 0 {
			return errors.New("some tickets were not redeemed")
		}
		if args.CancelAddress != *sender {
			return errors.New("wrong owner")
		}
	case common.ACC_TYPE_OF_PNS:
		if cancelAccount.PnsAccount().Owner != *sender {
			return errors.New("wrong owner")
		}
	case common.ACC_TYPE_OF_AUTHORIZE:
		if pool.chain.CurrentBlock().Number().Cmp(cancelAccount.AuthorizeAccount().ValidPeriod) != 1 {
			return errors.New("voting is not over")
		}
		if cancelAccount.AuthorizeAccount().VoteValue.Sign() > 0 {
			return errors.New("some tickets were not redeemed,please inform the voters first")
		}
	case common.ACC_TYPE_OF_LOSS:
		if cancelAccount.LossAccount().State != common.LOSS_STATE_OF_SUCCESS {
			return errors.New("cancellation is not allowed in the current state")
		}
		if cancelAccount.LossAccount().NewAccount != *sender {
			return errors.New("wrong owner")
		}
	default:
		return accounts.ErrWrongAccountType
	}
	return pool.validateGas(tx, local, sender)
}

func (pool *TxPool) validateTxOfTransfer(tx *types.Transaction, local bool) error {
	var sender *common.Address
	var err error
	if sender, err = pool.validateSender(tx, local); err != nil {
		return err
	}
	toAccount := pool.currentState.GetStateObject(*tx.To())
	if toAccount == nil {
		log.Warn(fmt.Sprintf("receiver not exists, Will be created:%s", tx.To()))
		if tx.Value().Cmp(new(big.Int).SetUint64(common.AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_REGULAR)) != 1 {
			return errors.New("receiver not exists and will be created,but the deposit is not enough")
		}
	} else {
		if toAccount.AccountType() != common.ACC_TYPE_OF_REGULAR && toAccount.AccountType() != common.ACC_TYPE_OF_CONTRACT {
			return errors.New("unsupported receiver")
		}
	}
	return pool.validateGas(tx, local, sender)
}

func (pool *TxPool) validateTxOfContractDeploy(tx *types.Transaction, local bool) error {
	var sender *common.Address
	var err error
	if sender, err = pool.validateSender(tx, local); err != nil {
		return err
	}
	return pool.validateGas(tx, local, sender)
}

func (pool *TxPool) validateTxOfSendLossReport(tx *types.Transaction, local bool) error {
	return errors.New("the current version does not support")

	var sender *common.Address
	var err error
	if sender, err = pool.validateSender(tx, local); err != nil {
		return err
	}

	return pool.validateGas(tx, local, sender)
}
func (pool *TxPool) validateTxOfRevealLossReport(tx *types.Transaction, local bool) error {
	return errors.New("the current version does not support")

	var sender *common.Address
	var err error
	if sender, err = pool.validateSender(tx, local); err != nil {
		return err
	}
	/*	if err := pool.validateSender(tx, local); err != nil {
			return err
		}
		if err := common.ValidateNil(tx.Value, "value"); err != nil {
			return err
		}
		if tx.Value().Sign() != 1 {
			return errors.New("value must be greater than 0")
		}
		if err := common.ValidateNil(tx.Data, "data"); err != nil {
			return err
		}
		if err := common.ValidateNil(tx.To, "loss account"); err != nil {
			return err
		}
		if err := common.ValidateNil(tx.Old, "lost account"); err != nil {
			return err
		}
		if err := common.ValidateNil(tx.New, "new account "); err != nil {
			return err
		}
		if err := common.ValidateAccType(tx.To(), common.ACC_TYPE_OF_LOSS, "to"); err != nil {
			return err
		}
		if err := common.ValidateAccType(tx.Old(), common.ACC_TYPE_OF_REGULAR, "old"); err != nil {
			return err
		}
		if err := common.ValidateAccType(tx.New(), common.ACC_TYPE_OF_REGULAR, "new"); err != nil {
			return err
		}
		oldAccount := pool.currentState.GetRegular(*tx.Old())
		if oldAccount == nil {
			return errors.New("old account no exists")
		}
		newAccount := pool.currentState.GetRegular(*tx.New())
		if newAccount == nil {
			return errors.New("new account no exists")
		}
		minMultipleAmount := new(big.Int).Mul(tx.Value(), new(big.Int).SetUint64(common.MIN_PERCENTAGE_OF_PLEDGE_FOR_RETRIEVE_LOST_ACCOUNT))
		if minMultipleAmount.Cmp(oldAccount.Value) == -1 {
			return errors.New("insufficient pledge amount")
		}
		lossAccount := pool.currentState.GetLoss(*tx.To())
		if lossAccount == nil {
			return errors.New("loss account no exists")
		}
		if lossAccount.State != common.LOSS_STATE_OF_APPLY {
			return errors.New("loss account has been revealed")
		}
		if !common.ByteSliceEqual(lossAccount.InfoDigest[:], tx.Data()) {
			return errors.New("wrong information digest")
		}
		markLossAccounts := pool.currentState.GetMarkLossAccounts(tx.To().Last12BytesToHash())
		if markLossAccounts == nil || len(markLossAccounts) == 0 {
			return errors.New("mark information no exists")
		}*/
	return pool.validateGas(tx, local, sender)
}
func (pool *TxPool) validateTxOfTransferLostAccount(tx *types.Transaction, local bool) error {
	return errors.New("the current version does not support")

	var sender *common.Address
	var err error
	if sender, err = pool.validateSender(tx, local); err != nil {
		return err
	}
	if err := common.ValidateNil(tx.To, "loss account"); err != nil {
		return err
	}
	/*	if err := common.ValidateAccType(tx.To(), common.ACC_TYPE_OF_LOSS, "to"); err != nil {
		return err
	}*/
	if !pool.currentState.Exist(*tx.To()) {
		return ErrAccountNotExists
	}
	lossAccount := pool.currentState.GetLoss(*tx.To())
	if lossAccount == nil {
		return errors.New("loss account no exists")
	}
	if lossAccount.State != common.LOSS_STATE_OF_NOTICE {
		return errors.New("loss account no revealed")
	}
	currentBlockNumber := pool.chain.CurrentBlock().Number()
	lossReportAccount := pool.currentState.GetRegular(lossAccount.LossAccount)
	if lossReportAccount == nil {
		return errors.New("the loss report account no exists")
	}
	if lossReportAccount.LossType == 0 {
		return errors.New("account loss reporting is not allowed")
	}
	difference := new(big.Int).Sub(currentBlockNumber, lossAccount.Height)
	lossTypeConfigHeight := new(big.Int).Mul(new(big.Int).SetUint64(uint64(lossReportAccount.LossType)), new(big.Int).SetUint64(common.CYCLE_HEIGHT_OF_LOSS_TYPE))
	if difference.Cmp(lossTypeConfigHeight) == -1 {
		return errors.New("the loss reporting time is not over")
	}
	return pool.validateGas(tx, local, sender)
}
func (pool *TxPool) validateTxOfRemoveLossReport(tx *types.Transaction, local bool) error {
	return errors.New("the current version does not support")

	var sender *common.Address
	var err error
	if sender, err = pool.validateSender(tx, local); err != nil {
		return err
	}
	if err := common.ValidateNil(tx.To, "loss account"); err != nil {
		return err
	}
	/*	if err := common.ValidateAccType(tx.To(), common.ACC_TYPE_OF_LOSS, "to"); err != nil {
		return err
	}*/
	lossAccount := pool.currentState.GetLoss(*tx.To())
	if lossAccount == nil {
		return errors.New("loss account no exists")
	}
	if lossAccount.State != common.LOSS_STATE_OF_APPLY {
		return errors.New("loss reporting status is not allowed")
	}
	currentBlockNumber := pool.chain.CurrentBlock().Number()
	thresholdBlockNumber := new(big.Int).Add(lossAccount.Height, new(big.Int).SetUint64(common.THRESHOLD_HEIGHT_OF_REMOVE_LOSS_REPORT))
	if currentBlockNumber.Cmp(thresholdBlockNumber) < 1 {
		return errors.New("current block number less than threshold")
	}
	pledgeAmount := new(big.Int).SetUint64(common.AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_LOSS)
	if tx.Value().Cmp(pledgeAmount) != 0 {
		return errors.New("wrong value")
	}
	return pool.validateGas(tx, local, sender)
}
func (pool *TxPool) validateTxOfRejectLossReport(tx *types.Transaction, local bool) error {
	return errors.New("the current version does not support")

	var sender *common.Address
	var err error
	if sender, err = pool.validateSender(tx, local); err != nil {
		return err
	}
	if err := common.ValidateNil(tx.To, "loss account"); err != nil {
		return err
	}
	/*	if err := common.ValidateAccType(tx.To(), common.ACC_TYPE_OF_LOSS, "to"); err != nil {
		return err
	}*/
	lossAccount := pool.currentState.GetLoss(*tx.To())
	if lossAccount == nil {
		return errors.New("loss account no exists")
	}
	if *sender != lossAccount.LossAccount {
		return errors.New("illegal operation")
	}
	if lossAccount.State == common.LOSS_STATE_OF_SUCCESS {
		return errors.New("the account has been retrieved. operation is not allowed")
	}
	pledgeAmount := new(big.Int).SetUint64(common.AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_LOSS)
	if tx.Value().Cmp(pledgeAmount) != 0 {
		return errors.New("wrong value")
	}
	return pool.validateGas(tx, local, sender)
}
func (pool *TxPool) validateTxOfApplyToBeDPoSNode(tx *types.Transaction, local bool) error {
	var sender *common.Address
	var err error
	if sender, err = pool.validateSender(tx, local); err != nil {
		return err
	}
	args := new(common.ApplyDPosArgs)
	if err := rlp.DecodeBytes(tx.Data(), &args); err != nil {
		return err
	}
	voteAccount := pool.currentState.GetStateObject(args.VoteAddress)
	if voteAccount == nil {
		return ErrAccountNotExists
	}
	if voteAccount.AccountType() != common.ACC_TYPE_OF_AUTHORIZE {
		return ErrValidUnsupportedAccount
	}
	if voteAccount.AuthorizeAccount().ValidPeriod.Cmp(pool.chain.CurrentBlock().Number()) != 1 {
		return ErrValidPeriodTooLow
	}
	fromAccount := pool.currentState.GetStateObject(*sender)
	if fromAccount.RegularAccount().VoteAccount != (common.Address{}) && fromAccount.RegularAccount().VoteAccount != args.VoteAddress {
		return ErrInvalidCandidateDPOS
	}
	limitMaxValue := big.NewInt(1)
	limitMaxValue.Mul(voteAccount.AuthorizeAccount().PledgeValue, big.NewInt(10))
	if limitMaxValue.Cmp(voteAccount.AuthorizeAccount().VoteValue) < 0 {
		return ErrValidCandidateDPOSValue
	}
	return pool.validateGas(tx, local, sender)
}

func (pool *TxPool) validateTxOfVote(tx *types.Transaction, local bool) error {
	var sender *common.Address
	var err error
	if sender, err = pool.validateSender(tx, local); err != nil {
		return err
	}
	voteArgs := new(common.VoteArgs)
	if err := rlp.DecodeBytes(tx.Data(), &voteArgs); err != nil {
		return err
	}
	voteAccount := pool.currentState.GetStateObject(voteArgs.VoteAddress)
	if voteAccount == nil {
		return ErrAccountNotExists
	}
	if voteAccount.AccountType() != common.ACC_TYPE_OF_AUTHORIZE {
		return ErrValidUnsupportedAccount
	}
	if voteAccount.AuthorizeAccount().ValidPeriod.Cmp(pool.chain.CurrentBlock().Number()) != 1 {
		return ErrValidPeriodTooLow
	}
	fromAccount := pool.currentState.GetStateObject(*sender).RegularAccount()
	if fromAccount.VoteAccount != (common.Address{}) && fromAccount.VoteAccount != voteArgs.VoteAddress {
		return errors.New("other candidates have been supported")
	}
	return pool.validateGas(tx, local, sender)
}

func (pool *TxPool) validateTxOfRedemption(tx *types.Transaction, local bool) error {
	var sender *common.Address
	var err error
	if sender, err = pool.validateSender(tx, local); err != nil {
		return err
	}
	voteArgs := new(common.VoteArgs)
	if err := rlp.DecodeBytes(tx.Data(), &voteArgs); err != nil {
		return err
	}
	var voteAccount = pool.currentState.GetStateObject(voteArgs.VoteAddress)
	if voteAccount == nil {
		return ErrAccountNotExists
	}
	if voteAccount.AccountType() != common.ACC_TYPE_OF_AUTHORIZE {
		return ErrValidUnsupportedAccount
	}
	if voteAccount.AuthorizeAccount().Owner != *sender {
		return errors.New("wrong owner")
	}
	if pool.chain.CurrentBlock().Number().Cmp(voteAccount.AuthorizeAccount().ValidPeriod) != 1 {
		return errors.New("this election is not over")
	}
	return pool.validateGas(tx, local, sender)
}
func (pool *TxPool) validateTxOfModifyPnsOwner(tx *types.Transaction, local bool) error {
	var sender *common.Address
	var err error
	if sender, err = pool.validateSender(tx, local); err != nil {
		return err
	}
	args := new(common.PnsOwnerArgs)
	if err := rlp.DecodeBytes(tx.Data(), &args); err != nil {
		return err
	}
	var pnsAccount = pool.currentState.GetStateObject(args.PnsAddress)
	if pnsAccount == nil {
		return ErrAccountNotExists
	}
	if pnsAccount.AccountType() != common.ACC_TYPE_OF_PNS {
		return ErrValidUnsupportedAccount
	}
	var ownerAccount = pool.currentState.GetStateObject(args.OwnerAddress)
	if ownerAccount == nil {
		return ErrAccountNotExists
	}
	if ownerAccount.AccountType() != common.ACC_TYPE_OF_REGULAR {
		return ErrValidUnsupportedAccount
	}
	if pnsAccount.PnsAccount().Owner != *sender {
		return errors.New("wrong pns owner")
	}
	if pnsAccount.PnsAccount().Owner == args.OwnerAddress {
		return errors.New("consistent with current owner")
	}
	return pool.validateGas(tx, local, sender)
}
func (pool *TxPool) validateTxOfModifyPnsContent(tx *types.Transaction, local bool) error {
	var sender *common.Address
	var err error
	if sender, err = pool.validateSender(tx, local); err != nil {
		return err
	}
	args := new(common.PnsContentArgs)
	if err := rlp.DecodeBytes(tx.Data(), &args); err != nil {
		return err
	}
	var pnsAccount = pool.currentState.GetStateObject(args.PnsAddress)
	if pnsAccount == nil {
		return ErrAccountNotExists
	}
	if pnsAccount.AccountType() != common.ACC_TYPE_OF_PNS {
		return ErrValidUnsupportedAccount
	}
	if pnsAccount.PnsAccount().Owner != *sender {
		return errors.New("wrong pns owner")
	}
	return pool.validateGas(tx, local, sender)
}

func (pool *TxPool) validateGas(tx *types.Transaction, local bool, sender *common.Address) error {
	// Accept only legacy transactions until EIP-2718/2930 activates.
	if !pool.eip2718 && tx.Type() != types.LegacyTxType {
		return ErrTxTypeNotSupported
	}
	// Reject dynamic fee transactions until EIP-1559 activates.
	if !pool.eip1559 && tx.Type() == types.DynamicFeeTxType {
		return ErrTxTypeNotSupported
	}
	// Reject transactions over defined size to prevent DOS attacks
	if uint64(tx.Size()) > txMaxSize {
		return ErrOversizedData
	}
	// Transactions can't be negative. This may never happen using RLP decoded
	// transactions but may occur if you create a transaction using the RPC.
	if tx.Value().Sign() < 0 {
		return ErrNegativeValue
	}
	// Ensure the transaction doesn't exceed the current block limit gas.
	if pool.currentMaxGas < tx.Gas() {
		return ErrGasLimit
	}
	// Sanity check for extremely large numbers
	if tx.GasFeeCap().BitLen() > 256 {
		return ErrFeeCapVeryHigh
	}
	if tx.GasTipCap().BitLen() > 256 {
		return ErrTipVeryHigh
	}
	// Ensure gasFeeCap is greater than or equal to gasTipCap.
	if tx.GasFeeCapIntCmp(tx.GasTipCap()) < 0 {
		return ErrTipAboveFeeCap
	}
	// Drop non-local transactions under our own minimal accepted gas price or tip
	if !local && tx.GasTipCapIntCmp(pool.gasPrice) < 0 {
		return ErrUnderpriced
	}
	// Ensure the transaction adheres to nonce ordering
	if pool.currentState.GetNonce(*sender) > tx.Nonce() {
		return ErrNonceTooLow
	}
	// Transactor should have enough funds to cover the costs
	// cost == V + GP * GL
	balacne := pool.currentState.GetBalance(*sender)
	cost := tx.Cost()
	if balacne.Cmp(cost) < 0 {
		fmt.Printf("余额不足，无法支付GAS. from:%s, 余额：%s,cost: %d\n", sender.String(), balacne.String(), cost.Int64())
		return ErrInsufficientFunds
	}
	// Ensure the transaction has more gas than the basic tx fee.
	intrGas, err := IntrinsicGas(tx.Data(), tx.AccessList(), tx.To() == nil, true, pool.istanbul)
	if err != nil {
		return err
	}
	if tx.Gas() < intrGas {
		return ErrIntrinsicGas
	}
	return nil
}
func (pool *TxPool) validateSender(tx *types.Transaction, local bool) (*common.Address, error) {
	from, err := types.Sender(pool.signer, tx)
	if err != nil {
		return nil, ErrInvalidSender
	}
	fromAccount := pool.currentState.GetStateObject(from)
	if fromAccount == nil {
		return nil, errors.New("sender not exists")
	}
	if fromAccount.AccountType() != common.ACC_TYPE_OF_REGULAR {
		return nil, errors.New("unsupported sender account")
	}
	return &from, nil
}

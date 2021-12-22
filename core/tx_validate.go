package core

import (
	"bytes"
	"errors"
	"github.com/probeum/go-probeum/accounts"
	"github.com/probeum/go-probeum/common"
	"github.com/probeum/go-probeum/core/types"
	"github.com/probeum/go-probeum/crypto"
	"github.com/probeum/go-probeum/rlp"
	"math/big"
	"strings"
)

//validateTxOfRegister validate transaction for register PNS、authorize、lose account
func (pool *TxPool) validateTxOfRegister(tx *types.Transaction, sender *common.Address) error {
	var newAccount common.Address
	switch tx.To().Hex() {
	case common.SPECIAL_ADDRESS_FOR_REGISTER_PNS:
		if len(tx.Data()) == 0 {
			return errors.New("pns data cannot be empty")
		}
		newAccount = crypto.CreatePNSAddress(*sender, tx.Data())
	case common.SPECIAL_ADDRESS_FOR_REGISTER_AUTHORIZE:
		newAccount = crypto.CreateAddress(*sender, tx.Nonce())
		decode := new(common.IntDecodeType)
		if err := rlp.DecodeBytes(tx.Data(), &decode); err != nil {
			return err
		}
		if decode.Num.Cmp(pool.chain.CurrentBlock().Number()) < 1 {
			return errors.New(`valid period block number must be specified and greater than current block number`)
		}
	case common.SPECIAL_ADDRESS_FOR_REGISTER_LOSE:
		decode := new(common.RegisterLossDecodeType)
		if err := rlp.DecodeBytes(tx.Data(), &decode); err != nil {
			return err
		}
		if err := pool.currentState.CanLossMark(decode.LastBitsMark); err != nil {
			return err
		}
		newAccount = crypto.CreateAddress(*sender, tx.Nonce())
	}
	if pool.currentState.Exist(newAccount) {
		return ErrAccountAlreadyExists
	}
	return nil
}

//validateTxOfCancellation validate transaction for cancellation account
func (pool *TxPool) validateTxOfCancellation(tx *types.Transaction, sender *common.Address) error {
	decode := new(common.CancellationDecodeType)
	if err := rlp.DecodeBytes(tx.Data(), &decode); err != nil {
		return err
	}
	cancelAccount := pool.currentState.GetStateObject(decode.CancelAddress)
	if cancelAccount == nil {
		return ErrAccountNotExists
	}
	beneficiaryAccount := pool.currentState.GetStateObject(decode.BeneficiaryAddress)
	if beneficiaryAccount == nil {
		return ErrAccountNotExists
	}
	if beneficiaryAccount.AccountType() != common.ACC_TYPE_OF_REGULAR {
		return errors.New("new beneficiary must be a regular account")
	}
	switch cancelAccount.AccountType() {
	case common.ACC_TYPE_OF_REGULAR:
		if cancelAccount.RegularAccount().VoteValue.Sign() > 0 {
			return errors.New("some tickets were not redeemed")
		}
		if decode.CancelAddress != *sender {
			return errors.New("invalid owner")
		}
	case common.ACC_TYPE_OF_PNS:
		if cancelAccount.PnsAccount().Owner != *sender {
			return errors.New("invalid owner")
		}
	case common.ACC_TYPE_OF_AUTHORIZE:
		if pool.chain.CurrentBlock().Number().Cmp(cancelAccount.AuthorizeAccount().ValidPeriod) != 1 {
			return errors.New("voting is not over")
		}
		if cancelAccount.AuthorizeAccount().Owner != *sender {
			return errors.New("invalid owner")
		}
		if cancelAccount.AuthorizeAccount().VoteValue.Sign() > 0 {
			return errors.New("some tickets were not redeemed,please inform the voters first")
		}
	default:
		return accounts.ErrWrongAccountType
	}
	return nil
}

//validateTxOfTransfer validate transaction for transfer
func (pool *TxPool) validateTxOfTransfer(tx *types.Transaction) error {
	if common.IsReservedAddress(*tx.To()) {
		return errors.New("receiver address is the reserved address of the system")
	}
	toAccount := pool.currentState.GetStateObject(*tx.To())
	if toAccount == nil {
		if tx.Value().Cmp(new(big.Int).SetUint64(common.AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_REGULAR)) != 1 {
			return errors.New("receiver not exists and will be created,but the deposit is not enough")
		}
	} else {
		if toAccount.AccountType() != common.ACC_TYPE_OF_REGULAR && toAccount.AccountType() != common.ACC_TYPE_OF_CONTRACT {
			return errors.New("unsupported receiver")
		}
	}
	return nil
}

//validateTxOfRevealLossReport validate transaction for reveal loss reporting
func (pool *TxPool) validateTxOfRevealLossReport(tx *types.Transaction) error {
	decode := new(common.RevealLossReportDecodeType)
	if err := rlp.DecodeBytes(tx.Data(), &decode); err != nil {
		return err
	}
	lossStateObj := pool.currentState.GetStateObject(decode.LossAccount)
	if lossStateObj == nil {
		return errors.New("loss report account not exists")
	}
	if lossStateObj.AccountType() != common.ACC_TYPE_OF_LOSS {
		return errors.New("invalid loss report account")
	}
	if lossStateObj.LossAccount().State != common.LOSS_STATE_OF_APPLY {
		return ErrValidLossState
	}
	lossMarkStateObj := pool.currentState.GetStateObject(common.HexToAddress(common.SPECIAL_ADDRESS_FOR_REGISTER_LOSE))
	if lossMarkStateObj == nil {
		return errors.New("loss mark account not exists")
	}
	lastBitToInt := decode.OldAccount.Last10BitsToUint()
	if uint64(lossStateObj.LossAccount().LastBits) != lastBitToInt {
		return errors.New("inconsistent beneficiary account")
	}
	LossMark := lossMarkStateObj.LossMarkAccount().LossMark
	if !LossMark.GetMark(uint(lastBitToInt % common.LossMarkBitLength)) {
		return errors.New("revelation repeated")
	}
	oldStateObj := pool.currentState.GetStateObject(decode.OldAccount)
	if oldStateObj == nil {
		return errors.New("lost account not exists")
	}
	if oldStateObj.AccountType() != common.ACC_TYPE_OF_REGULAR {
		return errors.New("invalid lost account")
	}
	lossType := oldStateObj.RegularAccount().LossType
	if lossType.GetType() == common.UNSUPPORTED_OF_LOSS_TYPE {
		return errors.New("lost account not support")
	}
	if lossType.GetState() {
		return errors.New("lost account in the process of loss reporting")
	}
	newStateObj := pool.currentState.GetStateObject(decode.NewAccount)
	if newStateObj == nil {
		return errors.New("new beneficiary account not exists")
	}
	if newStateObj.AccountType() != common.ACC_TYPE_OF_REGULAR {
		return errors.New("invalid new beneficiary account")
	}
	var buffer bytes.Buffer
	buffer.Write(decode.OldAccount.Bytes())
	buffer.Write(decode.NewAccount.Bytes())
	buffer.Write(new(big.Int).SetUint64(uint64(decode.RandomNum)).Bytes())
	if lossStateObj.LossAccount().InfoDigest != crypto.Keccak256Hash(buffer.Bytes()) {
		return errors.New("digests is incorrect")
	}
	minMultipleAmount := new(big.Int).Mul(tx.Value(), new(big.Int).SetUint64(uint64(common.MIN_PERCENTAGE_OF_PLEDGE_FOR_RETRIEVE_LOST_ACCOUNT)))
	if minMultipleAmount.Cmp(oldStateObj.Balance()) == -1 {
		return errors.New("insufficient pledge amount")
	}
	return nil
}

//validateTxOfTransferLostAccount validate transaction for transfer lost account balance
func (pool *TxPool) validateTxOfTransferLostAccount(tx *types.Transaction) error {
	decode := new(common.AddressDecodeType)
	if err := rlp.DecodeBytes(tx.Data(), &decode); err != nil {
		return err
	}
	lossStateObj := pool.currentState.GetStateObject(decode.Addr)
	if lossStateObj == nil {
		return ErrAccountNotExists
	}
	if lossStateObj.AccountType() != common.ACC_TYPE_OF_LOSS {
		return ErrValidUnsupportedAccount
	}
	if lossStateObj.LossAccount().State != common.LOSS_STATE_OF_REVEAL {
		return ErrValidLossState
	}
	lostStateObj := pool.currentState.GetStateObject(lossStateObj.LossAccount().LostAccount)
	if lostStateObj == nil {
		return errors.New("lost account not exist")
	}
	lossType := lostStateObj.RegularAccount().LossType
	if lossType.GetType() == common.UNSUPPORTED_OF_LOSS_TYPE {
		return errors.New("lost account not support")
	}
	if !lossType.GetState() {
		return errors.New("lost account not in loss reporting")
	}
	currentBlockNumber := pool.chain.CurrentBlock().Number()
	intervalHeight := new(big.Int).Sub(currentBlockNumber, lossStateObj.LossAccount().Height)
	lossTypeHeight := new(big.Int).Mul(new(big.Int).SetUint64(uint64(lossType.GetType())), new(big.Int).SetUint64(common.CYCLE_HEIGHT_BLOCKS_OF_LOSS_TYPE))
	if intervalHeight.Cmp(lossTypeHeight) == -1 {
		return errors.New("loss reporting cycle is not over")
	}
	return nil
}

//validateTxOfRemoveLossReport validate transaction for remove loss reporting
func (pool *TxPool) validateTxOfRemoveLossReport(tx *types.Transaction) error {
	decode := new(common.AddressDecodeType)
	if err := rlp.DecodeBytes(tx.Data(), &decode); err != nil {
		return err
	}
	lossStateObj := pool.currentState.GetStateObject(decode.Addr)
	if lossStateObj == nil {
		return ErrAccountNotExists
	}
	if lossStateObj.AccountType() != common.ACC_TYPE_OF_LOSS {
		return ErrValidUnsupportedAccount
	}
	if lossStateObj.LossAccount().State != common.LOSS_STATE_OF_APPLY {
		return ErrValidLossState
	}
	currentBlockNumber := pool.chain.CurrentBlock().Number()
	thresholdBlockNumber := new(big.Int).Add(lossStateObj.LossAccount().Height, new(big.Int).SetUint64(common.THRESHOLD_HEIGHT_OF_REMOVE_LOSS_REPORT))
	if currentBlockNumber.Cmp(thresholdBlockNumber) < 1 {
		return errors.New("threshold height too low")
	}
	return nil
}

//validateTxOfRejectLossReport validate transaction for reject loss reporting
func (pool *TxPool) validateTxOfRejectLossReport(tx *types.Transaction, sender *common.Address) error {
	decode := new(common.AddressDecodeType)
	if err := rlp.DecodeBytes(tx.Data(), &decode); err != nil {
		return err
	}
	lossStateObj := pool.currentState.GetStateObject(decode.Addr)
	if lossStateObj == nil {
		return ErrAccountNotExists
	}
	if lossStateObj.LossAccount().State != common.LOSS_STATE_OF_REVEAL {
		return ErrValidLossState
	}
	if *sender != lossStateObj.LossAccount().LostAccount {
		return errors.New("owner is incorrect")
	}
	return nil
}

//validateTxOfApplyToBeDPoSNode validate transaction for apply dPoS node
func (pool *TxPool) validateTxOfApplyToBeDPoSNode(tx *types.Transaction, sender *common.Address) error {
	decode := new(common.ApplyDPosDecodeType)
	if err := rlp.DecodeBytes(tx.Data(), &decode); err != nil {
		return err
	}
	if len(decode.NodeInfo) == 0 || strings.Index(decode.NodeInfo, common.DPosNodePrefix) == -1 {
		return errors.New("illegal node info format")
	}
	voteAccount := pool.currentState.GetStateObject(decode.VoteAddress)
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
	if fromAccount.RegularAccount().VoteAccount != (common.Address{}) && fromAccount.RegularAccount().VoteAccount != decode.VoteAddress {
		return ErrInvalidCandidateDPOS
	}
	limitMaxValue := big.NewInt(1)
	limitMaxValue.Mul(voteAccount.AuthorizeAccount().PledgeValue, big.NewInt(10))
	if limitMaxValue.Cmp(voteAccount.AuthorizeAccount().VoteValue) < 0 {
		return ErrValidCandidateDPOSValue
	}
	return nil
}

//validateTxOfVote validate transaction for vote
func (pool *TxPool) validateTxOfVote(tx *types.Transaction, sender *common.Address) error {
	decode := new(common.AddressDecodeType)
	if err := rlp.DecodeBytes(tx.Data(), &decode); err != nil {
		return err
	}
	voteAccount := pool.currentState.GetStateObject(decode.Addr)
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
	if fromAccount.VoteAccount != (common.Address{}) && fromAccount.VoteAccount != decode.Addr {
		return errors.New("other candidates have been supported")
	}
	return nil
}

//validateTxOfRedemption validate transaction for redemption vote
func (pool *TxPool) validateTxOfRedemption(tx *types.Transaction, sender *common.Address) error {
	decode := new(common.AddressDecodeType)
	if err := rlp.DecodeBytes(tx.Data(), &decode); err != nil {
		return err
	}
	var voteAccount = pool.currentState.GetStateObject(decode.Addr)
	var senderAccount = pool.currentState.GetStateObject(*sender)
	if voteAccount == nil {
		return ErrAccountNotExists
	}
	if voteAccount.AccountType() != common.ACC_TYPE_OF_AUTHORIZE {
		return ErrValidUnsupportedAccount
	}
	if voteAccount.AuthorizeAccount().Owner != *sender && senderAccount.RegularAccount().VoteAccount != decode.Addr {
		return errors.New("no voting records found")
	}
	if pool.chain.CurrentBlock().Number().Cmp(voteAccount.AuthorizeAccount().ValidPeriod) != 1 {
		return errors.New("this election is not over")
	}
	return nil
}

//validateTxOfModifyPnsOwner validate transaction for modify PNS owner
func (pool *TxPool) validateTxOfModifyPnsOwner(tx *types.Transaction, sender *common.Address) error {
	decode := new(common.PnsOwnerDecodeType)
	if err := rlp.DecodeBytes(tx.Data(), &decode); err != nil {
		return err
	}
	var pnsAccount = pool.currentState.GetStateObject(decode.PnsAddress)
	if pnsAccount == nil {
		return ErrAccountNotExists
	}
	if pnsAccount.AccountType() != common.ACC_TYPE_OF_PNS {
		return ErrValidUnsupportedAccount
	}
	var ownerAccount = pool.currentState.GetStateObject(decode.OwnerAddress)
	if ownerAccount == nil {
		return ErrAccountNotExists
	}
	if ownerAccount.AccountType() != common.ACC_TYPE_OF_REGULAR {
		return ErrValidUnsupportedAccount
	}
	if pnsAccount.PnsAccount().Owner != *sender {
		return errors.New("invalid pns owner")
	}
	if pnsAccount.PnsAccount().Owner == decode.OwnerAddress {
		return errors.New("consistent with current owner")
	}
	return nil
}

//validateTxOfModifyPnsContent validate transaction for PNS content
func (pool *TxPool) validateTxOfModifyPnsContent(tx *types.Transaction, sender *common.Address) error {
	decode := new(common.PnsContentDecodeType)
	if err := rlp.DecodeBytes(tx.Data(), &decode); err != nil {
		return err
	}
	if len(decode.PnsData) == 0 {
		return errors.New("pns data cannot be empty")
	}
	var pnsAccount = pool.currentState.GetStateObject(decode.PnsAddress)
	if pnsAccount == nil {
		return ErrAccountNotExists
	}
	if pnsAccount.AccountType() != common.ACC_TYPE_OF_PNS {
		return ErrValidUnsupportedAccount
	}
	if pnsAccount.PnsAccount().Owner != *sender {
		return errors.New("invalid pns owner")
	}
	return nil
}

//validateTxOfModifyLossType validate transaction for modify regular account loss type
func (pool *TxPool) validateTxOfModifyLossType(tx *types.Transaction) error {
	decode := new(common.ByteDecodeType)
	if err := rlp.DecodeBytes(tx.Data(), &decode); err != nil {
		return err
	}
	if decode.Num > common.MAX_CYCLE_HEIGHT_OF_LOSS_TYPE {
		return errors.New("maximum value of the loss type is 127")
	}
	return nil
}

//validateTxOfTransferLostAssociatedAccount validate transaction for transfer lost associated account, like pns,authorize account
func (pool *TxPool) validateTxOfTransferLostAssociatedAccount(tx *types.Transaction) error {
	decode := new(common.AssociatedAccountDecodeType)
	if err := rlp.DecodeBytes(tx.Data(), &decode); err != nil {
		return err
	}
	var lossObj = pool.currentState.GetStateObject(decode.LossAccount)
	if lossObj == nil {
		return errors.New("loss report account not exists")
	}
	if lossObj.AccountType() != common.ACC_TYPE_OF_LOSS {
		return errors.New("invalid loss report account")
	}
	if lossObj.LossAccount().State != common.LOSS_STATE_OF_SUCCESS {
		return errors.New("loss report was not successful")
	}
	var lostObj = pool.currentState.GetStateObject(lossObj.LossAccount().LostAccount)
	if lostObj == nil {
		return errors.New("lost account not exists")
	}
	if lostObj.AccountType() != common.ACC_TYPE_OF_REGULAR {
		return errors.New("invalid lost account")
	}
	var newObj = pool.currentState.GetStateObject(lossObj.LossAccount().NewAccount)
	if newObj == nil {
		return errors.New("new beneficiary account not exists")
	}
	if newObj.AccountType() != common.ACC_TYPE_OF_REGULAR {
		return errors.New("invalid new beneficiary account")
	}
	switch tx.To().Hex() {
	case common.SPECIAL_ADDRESS_FOR_TRANSFER_LOST_ACCOUNT_PNS:
		var pnsObj = pool.currentState.GetStateObject(decode.AssociatedAccount)
		if pnsObj == nil {
			return errors.New("pns account not exists")
		}
		if pnsObj.AccountType() != common.ACC_TYPE_OF_PNS {
			return errors.New("invalid pns account")
		}
		if pnsObj.PnsAccount().Owner != lostObj.Address() {
			return errors.New("invalid pns owner")
		}
	case common.SPECIAL_ADDRESS_FOR_TRANSFER_LOST_ACCOUNT_AUTHORIZE:
		var authorizeObj = pool.currentState.GetStateObject(decode.AssociatedAccount)
		if authorizeObj == nil {
			return errors.New("authorize account not exists")
		}
		if authorizeObj.AccountType() != common.ACC_TYPE_OF_AUTHORIZE {
			return errors.New("invalid authorize account")
		}
		if authorizeObj.AuthorizeAccount().Owner != lostObj.Address() {
			return errors.New("invalid authorize owner")
		}
	}
	return nil
}

func (pool *TxPool) validateTxOfCancellationLossAccount(tx *types.Transaction) error {
	decode := new(common.AddressDecodeType)
	if err := rlp.DecodeBytes(tx.Data(), &decode); err != nil {
		return err
	}
	var lossObj = pool.currentState.GetStateObject(decode.Addr)
	if lossObj == nil {
		return errors.New("loss report account not exists")
	}
	if lossObj.AccountType() != common.ACC_TYPE_OF_LOSS {
		return errors.New("invalid loss report account")
	}
	if lossObj.LossAccount().State != common.LOSS_STATE_OF_SUCCESS {
		return errors.New("loss report was not successful")
	}
	return nil
}

//validateGas validate transaction for Gas
func (pool *TxPool) validateGas(tx *types.Transaction, local bool) error {
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

//validateSender validate transaction for sender
func (pool *TxPool) validateSender(tx *types.Transaction) (*common.Address, error) {
	sender, err := types.Sender(pool.signer, tx)
	if err != nil {
		return nil, ErrInvalidSender
	}
	fromAccount := pool.currentState.GetStateObject(sender)
	if fromAccount == nil {
		return nil, errors.New("sender not exists")
	}
	if fromAccount.AccountType() != common.ACC_TYPE_OF_REGULAR {
		return nil, ErrInvalidSender
	}
	// Ensure the transaction adheres to nonce ordering
	if pool.currentState.GetNonce(sender) > tx.Nonce() {
		return nil, ErrNonceTooLow
	}
	// Transactor should have enough funds to cover the costs
	// cost == V + GP * GL
	balance := pool.currentState.GetBalance(sender)
	cost := tx.Cost()
	if balance.Cmp(cost) < 0 {
		return nil, ErrInsufficientFunds
	}
	return &sender, nil
}

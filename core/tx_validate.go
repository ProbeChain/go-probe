package core

import (
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
)

// validateTx validate transaction of register business type
func (pool *TxPool) validateTxOfRegister(tx *types.Transaction, local bool) error {
	accType := uint8(*tx.AccType())
	if !common.CheckRegisterAccType(accType) {
		return accounts.ErrWrongAccountType
	}
	if err := common.ValidateAccType(tx.From(), common.ACC_TYPE_OF_GENERAL, "from"); err != nil {
		return err
	}
	if err := common.ValidateAccType(tx.New(), accType, "new"); err != nil {
		return err
	}
	if accType == common.ACC_TYPE_OF_AUTHORIZE {
		if tx.Height == nil || tx.Height().Cmp(pool.chain.CurrentBlock().Number()) < 1 {
			return errors.New(`valid period block number must be specified and greater than current block number`)
		}
	}
	if err := pool.validateSender(tx, local); err != nil {
		return err
	}
	if pool.currentState.Exist(*tx.New()) {
		return ErrAccountAlreadyExists
	}
	return pool.validateGas(tx, local)
}

func (pool *TxPool) validateTxOfCancellation(tx *types.Transaction, local bool) error {
	if err := pool.validateSender(tx, local); err != nil {
		return err
	}
	return pool.validateGas(tx, local)
}

func (pool *TxPool) validateTxOfRevokeCancellation(tx *types.Transaction, local bool) error {
	if err := pool.validateSender(tx, local); err != nil {
		return err
	}
	return pool.validateGas(tx, local)
}

func (pool *TxPool) validateTxOfTransfer(tx *types.Transaction, local bool) error {
	if err := pool.validateSender(tx, local); err != nil {
		return err
	}
	return pool.validateGas(tx, local)
}
func (pool *TxPool) validateTxOfExchangeAsset(tx *types.Transaction, local bool) error {
	if err := pool.validateSender(tx, local); err != nil {
		return err
	}
	return pool.validateGas(tx, local)
}

func (pool *TxPool) validateTxOfContractCall(tx *types.Transaction, local bool) error {
	if err := pool.validateSender(tx, local); err != nil {
		return err
	}
	return pool.validateGas(tx, local)
}

func (pool *TxPool) validateTxOfSendLossReport(tx *types.Transaction, local bool) error {
	if err := pool.validateSender(tx, local); err != nil {
		return err
	}
	return pool.validateGas(tx, local)
}
func (pool *TxPool) validateTxOfRevealLossReport(tx *types.Transaction, local bool) error {
	if err := pool.validateSender(tx, local); err != nil {
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
	if err := common.ValidateAccType(tx.To(), common.ACC_TYPE_OF_LOSE, "to"); err != nil {
		return err
	}
	if err := common.ValidateAccType(tx.Old(), common.ACC_TYPE_OF_GENERAL, "old"); err != nil {
		return err
	}
	if err := common.ValidateAccType(tx.New(), common.ACC_TYPE_OF_GENERAL, "new"); err != nil {
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
	minMultipleAmount := new(big.Int).Mul(tx.Value(), new(big.Int).SetUint64(common.MIN_MULTIPLE_OF_PLEDGE_FOR_RETRIEVE_LOST_ACCOUNT))
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
	if !common.ByteSliceEqual(lossAccount.InfoDigest, tx.Data()) {
		return errors.New("wrong information digest")
	}
	markLossAccounts := pool.currentState.GetMarkLossAccounts(tx.To().Last12BytesToHash())
	if markLossAccounts == nil || len(markLossAccounts) == 0 {
		return errors.New("mark information no exists")
	}
	return pool.validateGas(tx, local)
}
func (pool *TxPool) validateTxOfTransferLostAccount(tx *types.Transaction, local bool) error {
	if err := pool.validateSender(tx, local); err != nil {
		return err
	}
	if err := common.ValidateNil(tx.To, "loss account"); err != nil {
		return err
	}
	if err := common.ValidateAccType(tx.To(), common.ACC_TYPE_OF_LOSE, "to"); err != nil {
		return err
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
	return pool.validateGas(tx, local)
}
func (pool *TxPool) validateTxOfTransferLostAssetAccount(tx *types.Transaction, local bool) error {
	if err := pool.validateSender(tx, local); err != nil {
		return err
	}
	return pool.validateGas(tx, local)
}
func (pool *TxPool) validateTxOfRemoveLossReport(tx *types.Transaction, local bool) error {
	if err := pool.validateSender(tx, local); err != nil {
		return err
	}
	if err := common.ValidateNil(tx.To, "loss account"); err != nil {
		return err
	}
	if err := common.ValidateAccType(tx.To(), common.ACC_TYPE_OF_LOSE, "to"); err != nil {
		return err
	}
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
	return pool.validateGas(tx, local)
}
func (pool *TxPool) validateTxOfRejectLossReport(tx *types.Transaction, local bool) error {
	if err := pool.validateSender(tx, local); err != nil {
		return err
	}
	if err := common.ValidateNil(tx.To, "loss account"); err != nil {
		return err
	}
	if err := common.ValidateAccType(tx.To(), common.ACC_TYPE_OF_LOSE, "to"); err != nil {
		return err
	}
	lossAccount := pool.currentState.GetLoss(*tx.To())
	if lossAccount == nil {
		return errors.New("loss account no exists")
	}
	if *tx.From() != lossAccount.LossAccount {
		return errors.New("illegal operation")
	}
	if lossAccount.State == common.LOSS_STATE_OF_SUCCESS {
		return errors.New("the account has been retrieved. operation is not allowed")
	}
	return pool.validateGas(tx, local)
}
func (pool *TxPool) validateTxOfApplyToBeDPoSNode(tx *types.Transaction, local bool) error {
	if err := pool.validateSender(tx, local); err != nil {
		return err
	}
	voteAccount := pool.currentState.GetAuthorize(*tx.To())
	if voteAccount == nil {
		return ErrAccountNotExists
	}
	return pool.validateGas(tx, local)
}

func (pool *TxPool) validateTxOfUpdatingVotesOrData(tx *types.Transaction, local bool) error {
	if err := pool.validateSender(tx, local); err != nil {
		return err
	}
	//todo
	return pool.validateGas(tx, local)
}

func (pool *TxPool) validateTxOfVote(tx *types.Transaction, local bool) error {
	if err := pool.validateSender(tx, local); err != nil {
		return err
	}
	authorizeAccount := pool.currentState.GetAuthorize(*tx.To())
	if authorizeAccount == nil {
		return ErrAccountNotExists
	}
	if authorizeAccount.ValidPeriod.Cmp(pool.chain.CurrentBlock().Number()) != 1 {
		return ErrValidPeriodTooLow
	}
	regularAccount := pool.currentState.GetRegular(*tx.From())
	if regularAccount == nil {
		return ErrAccountNotExists
	}
	if regularAccount.VoteAccount != (common.Address{}) && regularAccount.VoteAccount != *tx.To() {
		return ErrInvalidCandidate
	}
	return pool.validateGas(tx, local)
}

func (pool *TxPool) validateTxOfRedemption(tx *types.Transaction, local bool) error {
	if err := pool.validateSender(tx, local); err != nil {
		return err
	}
	if err := common.ValidateNil(tx.To(), "authorize account"); err != nil {
		return err
	}
	if err := common.ValidateAccType(tx.To(), common.ACC_TYPE_OF_AUTHORIZE, "authorize"); err != nil {
		return err
	}
	var authorizeAccount = pool.currentState.GetAuthorize(*tx.To())
	var regularAccount = pool.currentState.GetRegular(*tx.From())
	if authorizeAccount == nil {
		return errors.New("authorize account not exists")
	}
	if authorizeAccount.Owner != *tx.From() && regularAccount.VoteAccount != *tx.To() {
		return errors.New("wrong vote account")
	}
	if pool.chain.CurrentBlock().Number().Cmp(authorizeAccount.ValidPeriod) != 1 {
		return errors.New("the voting cycle is not over")
	}
	return pool.validateGas(tx, local)
}
func (pool *TxPool) validateTxOfModifyLossType(tx *types.Transaction, local bool) error {
	if err := pool.validateSender(tx, local); err != nil {
		return err
	}
	return pool.validateGas(tx, local)
}
func (pool *TxPool) validateTxOfModifyPnsOwner(tx *types.Transaction, local bool) error {
	if err := pool.validateSender(tx, local); err != nil {
		return err
	}
	if err := common.ValidateNil(tx.To(), "pns account"); err != nil {
		return err
	}
	if err := common.ValidateNil(tx.New(), "new owner account"); err != nil {
		return err
	}
	if err := common.ValidateAccType(tx.To(), common.ACC_TYPE_OF_PNS, "to"); err != nil {
		return err
	}
	if err := common.ValidateAccType(tx.New(), common.ACC_TYPE_OF_PNS, "new owner"); err != nil {
		return err
	}
	if !pool.currentState.Exist(*tx.To()) {
		return errors.New("pns account not exist")
	}
	if !pool.currentState.Exist(*tx.New()) {
		return errors.New("new pns account not exist")
	}
	return pool.validateGas(tx, local)
}
func (pool *TxPool) validateTxOfModifyPnsContent(tx *types.Transaction, local bool) error {
	if err := pool.validateSender(tx, local); err != nil {
		return err
	}
	if err := common.ValidateNil(tx.To(), "pns account"); err != nil {
		return err
	}
	if err := common.ValidateAccType(tx.To(), common.ACC_TYPE_OF_PNS, "pns"); err != nil {
		return err
	}
	if !common.CheckPnsType(byte(*tx.PnsType())) {
		return errors.New("wrong pns type")
	}
	if len(tx.Data()) == 0 {
		return errors.New("pns content data must be specified")
	}
	return pool.validateGas(tx, local)
}

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
	// Ensure the transaction adheres to nonce ordering
	if pool.currentState.GetNonce(*tx.From()) > tx.Nonce() {
		return ErrNonceTooLow
	}
	// Transactor should have enough funds to cover the costs
	// cost == V + GP * GL
	balacne := pool.currentState.GetBalance(*tx.From())
	cost := tx.Cost()
	fmt.Printf("from:%s, 余额：%s,cost: %d\n", tx.From().String(), balacne.String(), cost.Int64())
	if balacne.Cmp(cost) < 0 {
		fmt.Println("余额不足，无法支付GAS")
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
func (pool *TxPool) validateSender(tx *types.Transaction, local bool) error {
	from, err := types.Sender(pool.signer, tx)
	if err != nil {
		return ErrInvalidSender
	}
	if err := common.ValidateAccType(&from, common.ACC_TYPE_OF_GENERAL, "from"); err != nil {
		return err
	}
	if from != *tx.From() {
		return errors.New("illegal sender")
	}
	if !pool.currentState.Exist(from) {
		return errors.New("sender not exists")
	}
	return nil
}

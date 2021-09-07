package core

import "github.com/ethereum/go-ethereum/core/types"

// wxc todo 交易校验
// validateTx validate transaction of register business type
func validateTxOfRegister(tx *types.Transaction, local bool, pool *TxPool) error {
	return nil
}

func validateTxOfCancellation(tx *types.Transaction, local bool, pool *TxPool) error {
	return nil
}

func validateTxOfRevokeCancellation(tx *types.Transaction, local bool, pool *TxPool) error {
	return nil
}

func validateTxOfTransfer(tx *types.Transaction, local bool, pool *TxPool) error {
	return nil
}

func validateTxOfContractCall(tx *types.Transaction, local bool, pool *TxPool) error {
	return nil
}
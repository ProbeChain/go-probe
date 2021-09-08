package ethapi

import (
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
)
// todo 各种交易类型的交易信息结构组装实现
func (args *TransactionArgs) transactionOfRegister() *types.Transaction {
	var data types.TxData
	switch {
	case args.MaxFeePerGas != nil:
		al := types.AccessList{}
		if args.AccessList != nil {
			al = *args.AccessList
		}
		data = &types.DynamicFeeTx{
			New:        args.New,
			BizType:    uint8(*args.BizType),
			ChainID:    (*big.Int)(args.ChainID),
			Nonce:      uint64(*args.Nonce),
			Gas:        uint64(*args.Gas),
			GasFeeCap:  (*big.Int)(args.MaxFeePerGas),
			GasTipCap:  (*big.Int)(args.MaxPriorityFeePerGas),
			Value:      (*big.Int)(args.Value),
			Data:       args.data(),
			AccessList: al,
		}
	case args.AccessList != nil:
		data = &types.AccessListTx{
			New:        args.New,
			BizType:    uint8(*args.BizType),
			ChainID:    (*big.Int)(args.ChainID),
			Nonce:      uint64(*args.Nonce),
			Gas:        uint64(*args.Gas),
			GasPrice:   (*big.Int)(args.GasPrice),
			Value:      (*big.Int)(args.Value),
			Data:       args.data(),
			AccessList: *args.AccessList,
		}
	default:
		data = &types.LegacyTx{
			New:        args.New,
			BizType:    uint8(*args.BizType),
			Nonce:      uint64(*args.Nonce),
			Gas:        uint64(*args.Gas),
			GasPrice:   (*big.Int)(args.GasPrice),
			Value:      (*big.Int)(args.Value),
			Data:       args.data(),
		}
	}
	return types.NewTx(data)
}

func (args *TransactionArgs) transactionOfCancellation() *types.Transaction{
	return nil
}

func (args *TransactionArgs) transactionOfRevokeCancellation()  *types.Transaction{
	return nil
}

func (args *TransactionArgs) transactionOfTransfer()  *types.Transaction{
	var data types.TxData
	switch {
	case args.MaxFeePerGas != nil:
		al := types.AccessList{}
		if args.AccessList != nil {
			al = *args.AccessList
		}
		data = &types.DynamicFeeTx{
			To:         args.To,
			BizType:    uint8(*args.BizType),
			ChainID:    (*big.Int)(args.ChainID),
			Nonce:      uint64(*args.Nonce),
			Gas:        uint64(*args.Gas),
			GasFeeCap:  (*big.Int)(args.MaxFeePerGas),
			GasTipCap:  (*big.Int)(args.MaxPriorityFeePerGas),
			Value:      (*big.Int)(args.Value),
			Data:       args.data(),
			AccessList: al,
		}
	case args.AccessList != nil:
		data = &types.AccessListTx{
			To:         args.To,
			BizType:    uint8(*args.BizType),
			ChainID:    (*big.Int)(args.ChainID),
			Nonce:      uint64(*args.Nonce),
			Gas:        uint64(*args.Gas),
			GasPrice:   (*big.Int)(args.GasPrice),
			Value:      (*big.Int)(args.Value),
			Data:       args.data(),
			AccessList: *args.AccessList,
		}
	default:
		data = &types.LegacyTx{
			To:         args.To,
			BizType:    uint8(*args.BizType),
			Nonce:      uint64(*args.Nonce),
			Gas:        uint64(*args.Gas),
			GasPrice:   (*big.Int)(args.GasPrice),
			Value:      (*big.Int)(args.Value),
			Data:       args.data(),
		}
	}
	return types.NewTx(data)
}

func (args *TransactionArgs) transactionOfContractCall()  *types.Transaction{
	return nil
}

func (args *TransactionArgs) transactionOfExchangeTransaction()  *types.Transaction{
	return nil
}

func (args *TransactionArgs) transactionOfVotingForAnAccount()  *types.Transaction{
	return nil
}

func (args *TransactionArgs) transactionOfApplyToBeDPoSNode()  *types.Transaction{
	return nil
}

func (args *TransactionArgs) transactionOfUpdatingVotesOrData()  *types.Transaction{
	return nil
}

func (args *TransactionArgs) transactionOfSendLossReport()  *types.Transaction{
	return nil
}

func (args *TransactionArgs) transactionOfRevealLossMessage()  *types.Transaction{
	return nil
}

func (args *TransactionArgs) transactionOfTransferLostAccountWhenTimeOut()  *types.Transaction{
	return nil
}

func (args *TransactionArgs) transactionOfTransferLostAccountWhenConfirmed()  *types.Transaction{
	return nil
}

func (args *TransactionArgs) transactionOfRejectLossReportWhenTimeOut()  *types.Transaction{
	return nil
}




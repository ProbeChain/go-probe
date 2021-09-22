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
			From:       args.From,
			ChainID:    (*big.Int)(args.ChainID),
			Nonce:      uint64(*args.Nonce),
			Gas:        uint64(*args.Gas),
			GasFeeCap:  (*big.Int)(args.MaxFeePerGas),
			GasTipCap:  (*big.Int)(args.MaxPriorityFeePerGas),
			Value:      (*big.Int)(args.Value),
			Data:       args.data(),
			AccessList: al,
			New:        args.New,
			AccType:    args.AccType,
			BizType:    uint8(*args.BizType),
			Loss:       args.Loss,
			Receiver:   args.Receiver,
			Height:     (*big.Int)(args.Height),
		}
	case args.AccessList != nil:
		data = &types.AccessListTx{
			From:       args.From,
			ChainID:    (*big.Int)(args.ChainID),
			Nonce:      uint64(*args.Nonce),
			Gas:        uint64(*args.Gas),
			GasPrice:   (*big.Int)(args.GasPrice),
			Value:      (*big.Int)(args.Value),
			Data:       args.data(),
			AccessList: *args.AccessList,
			New:        args.New,
			AccType:    args.AccType,
			BizType:    uint8(*args.BizType),
			Loss:       args.Loss,
			Receiver:   args.Receiver,
			Height:     (*big.Int)(args.Height),
		}
	default:
		data = &types.LegacyTx{
			From:     args.From,
			Nonce:    uint64(*args.Nonce),
			Gas:      uint64(*args.Gas),
			GasPrice: (*big.Int)(args.GasPrice),
			Value:    (*big.Int)(args.Value),
			Data:     args.data(),
			New:      args.New,
			AccType:  args.AccType,
			BizType:  uint8(*args.BizType),
			Loss:     args.Loss,
			Receiver: args.Receiver,
			Height:   (*big.Int)(args.Height),
		}
	}
	return types.NewTx(data)
}

func (args *TransactionArgs) transactionOfCancellation() *types.Transaction {
	return nil
}

func (args *TransactionArgs) transactionOfRevokeCancellation() *types.Transaction {
	return nil
}

func (args *TransactionArgs) transactionOfTransfer() *types.Transaction {
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
			To:       args.To,
			BizType:  uint8(*args.BizType),
			Nonce:    uint64(*args.Nonce),
			Gas:      uint64(*args.Gas),
			GasPrice: (*big.Int)(args.GasPrice),
			Value:    (*big.Int)(args.Value),
			Data:     args.data(),
		}
	}
	return types.NewTx(data)
}

func (args *TransactionArgs) transactionOfContractCall() *types.Transaction {
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
			To:       args.To,
			BizType:  uint8(*args.BizType),
			Nonce:    uint64(*args.Nonce),
			Gas:      uint64(*args.Gas),
			GasPrice: (*big.Int)(args.GasPrice),
			Value:    (*big.Int)(args.Value),
			Data:     args.data(),
		}
	}
	return types.NewTx(data)
}

func (args *TransactionArgs) transactionOfExchangeTransaction() *types.Transaction {
	return nil
}

func (args *TransactionArgs) transactionOfVote() *types.Transaction {
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
			To:       args.To,
			BizType:  uint8(*args.BizType),
			Nonce:    uint64(*args.Nonce),
			Gas:      uint64(*args.Gas),
			GasPrice: (*big.Int)(args.GasPrice),
			Value:    (*big.Int)(args.Value),
			Data:     args.data(),
		}
	}
	return types.NewTx(data)
}

func (args *TransactionArgs) transactionOfApplyToBeDPoSNode() *types.Transaction {
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
			Mark:       args.mark(),
			InfoDigest: args.infoDigest(),
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
			Mark:       args.mark(),
			InfoDigest: args.infoDigest(),
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
			Mark:       args.mark(),
			InfoDigest: args.infoDigest(),
		}
	}
	return types.NewTx(data)
}

func (args *TransactionArgs) transactionOfUpdatingVotesOrData() *types.Transaction {
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
			Mark:       args.mark(),
			InfoDigest: args.infoDigest(),
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
			Mark:       args.mark(),
			InfoDigest: args.infoDigest(),
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
			Mark:       args.mark(),
			InfoDigest: args.infoDigest(),
		}
	}
	return types.NewTx(data)
}

func (args *TransactionArgs) transactionOfSendLossReport() *types.Transaction {
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
			Mark:       args.mark(),
			InfoDigest: args.infoDigest(),
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
			Mark:       args.mark(),
			InfoDigest: args.infoDigest(),
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
			Mark:       args.mark(),
			InfoDigest: args.infoDigest(),
		}
	}
	return types.NewTx(data)
}

func (args *TransactionArgs) transactionOfRevealLossMessage() *types.Transaction {
	return nil
}

func (args *TransactionArgs) transactionOfTransferLostAccountWhenTimeOut() *types.Transaction {
	return nil
}

func (args *TransactionArgs) transactionOfTransferLostAccountWhenConfirmed() *types.Transaction {
	return nil
}

func (args *TransactionArgs) transactionOfRejectLossReportWhenTimeOut() *types.Transaction {
	return nil
}

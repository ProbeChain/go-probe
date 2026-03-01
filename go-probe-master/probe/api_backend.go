// Copyright 2015 The ProbeChain Authors
// This file is part of the ProbeChain.
//
// The ProbeChain is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The ProbeChain is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the ProbeChain. If not, see <http://www.gnu.org/licenses/>.

package probe

import (
	"context"
	"errors"
	"math/big"

	"github.com/probechain/go-probe/accounts"
	"github.com/probechain/go-probe/common"
	"github.com/probechain/go-probe/consensus"
	"github.com/probechain/go-probe/core"
	"github.com/probechain/go-probe/core/bloombits"
	"github.com/probechain/go-probe/core/rawdb"
	"github.com/probechain/go-probe/core/state"
	"github.com/probechain/go-probe/core/types"
	"github.com/probechain/go-probe/core/vm"
	"github.com/probechain/go-probe/event"
	"github.com/probechain/go-probe/miner"
	"github.com/probechain/go-probe/params"
	"github.com/probechain/go-probe/probe/downloader"
	"github.com/probechain/go-probe/probe/gasprice"
	"github.com/probechain/go-probe/probedb"
	"github.com/probechain/go-probe/rpc"
)

// ProbeAPIBackend implements probeapi.Backend for full nodes
type ProbeAPIBackend struct {
	extRPCEnabled       bool
	allowUnprotectedTxs bool
	probe               *Probeum
	gpo                 *gasprice.Oracle
}

// ChainConfig returns the active chain configuration.
func (b *ProbeAPIBackend) ChainConfig() *params.ChainConfig {
	return b.probe.blockchain.Config()
}

func (b *ProbeAPIBackend) CurrentBlock() *types.Block {
	return b.probe.blockchain.CurrentBlock()
}

func (b *ProbeAPIBackend) SetHead(number uint64) {
	b.probe.handler.downloader.Cancel()
	b.probe.blockchain.SetHead(number)
}

func (b *ProbeAPIBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		block := b.probe.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		return b.probe.blockchain.CurrentBlock().Header(), nil
	}
	return b.probe.blockchain.GetHeaderByNumber(uint64(number)), nil
}

func (b *ProbeAPIBackend) Validators(number rpc.BlockNumber) []*common.Validator {
	return b.probe.blockchain.GetValidators(uint64(number))
}

func (b *ProbeAPIBackend) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.HeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.probe.blockchain.GetHeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.probe.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		return header, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *ProbeAPIBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.probe.blockchain.GetHeaderByHash(hash), nil
}

func (b *ProbeAPIBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		//block := b.probe.miner.PendingBlock()
		//return block, nil
		return b.probe.blockchain.CurrentBlock(), nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		return b.probe.blockchain.CurrentBlock(), nil
	}
	return b.probe.blockchain.GetBlockByNumber(uint64(number)), nil
}

func (b *ProbeAPIBackend) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.probe.blockchain.GetBlockByHash(hash), nil
}

func (b *ProbeAPIBackend) BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.BlockByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.probe.blockchain.GetHeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.probe.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		block := b.probe.blockchain.GetBlock(hash, header.Number.Uint64())
		if block == nil {
			return nil, errors.New("header found, but block body is missing")
		}
		return block, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *ProbeAPIBackend) PendingBlockAndReceipts() (*types.Block, types.Receipts) {
	return b.probe.miner.PendingBlockAndReceipts()
}

func (b *ProbeAPIBackend) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if number == rpc.PendingBlockNumber {
		block, state := b.probe.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, nil, err
	}
	if header == nil {
		return nil, nil, errors.New("header not found")
	}
	stateDb, err := b.probe.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *ProbeAPIBackend) StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.StateAndHeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header, err := b.HeaderByHash(ctx, hash)
		if err != nil {
			return nil, nil, err
		}
		if header == nil {
			return nil, nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.probe.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, nil, errors.New("hash is not currently canonical")
		}
		stateDb, err := b.probe.BlockChain().StateAt(header.Root)
		return stateDb, header, err
	}
	return nil, nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *ProbeAPIBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return b.probe.blockchain.GetReceiptsByHash(hash), nil
}

func (b *ProbeAPIBackend) GetLogs(ctx context.Context, hash common.Hash) ([][]*types.Log, error) {
	receipts := b.probe.blockchain.GetReceiptsByHash(hash)
	if receipts == nil {
		return nil, nil
	}
	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}
	return logs, nil
}

func (b *ProbeAPIBackend) GetTd(ctx context.Context, hash common.Hash) *big.Int {
	return b.probe.blockchain.GetTdByHash(hash)
}

func (b *ProbeAPIBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmConfig *vm.Config) (*vm.EVM, func() error, error) {
	vmError := func() error { return nil }
	if vmConfig == nil {
		vmConfig = b.probe.blockchain.GetVMConfig()
	}
	txContext := core.NewEVMTxContext(msg)
	context := core.NewEVMBlockContext(header, b.probe.BlockChain(), nil)
	return vm.NewEVM(context, txContext, state, b.probe.blockchain.Config(), *vmConfig), vmError, nil
}

func (b *ProbeAPIBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.probe.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *ProbeAPIBackend) SubscribePendingLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.probe.miner.SubscribePendingLogs(ch)
}

func (b *ProbeAPIBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.probe.BlockChain().SubscribeChainEvent(ch)
}

func (b *ProbeAPIBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.probe.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *ProbeAPIBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.probe.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *ProbeAPIBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.probe.BlockChain().SubscribeLogsEvent(ch)
}

func (b *ProbeAPIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.probe.txPool.AddLocal(signedTx)
}

func (b *ProbeAPIBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.probe.txPool.Pending(false)
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *ProbeAPIBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.probe.txPool.Get(hash)
}

func (b *ProbeAPIBackend) GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, common.Hash, uint64, uint64, error) {
	tx, blockHash, blockNumber, index := rawdb.ReadTransaction(b.probe.ChainDb(), txHash)
	return tx, blockHash, blockNumber, index, nil
}

func (b *ProbeAPIBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.probe.txPool.Nonce(addr), nil
}

func (b *ProbeAPIBackend) Stats() (pending int, queued int) {
	return b.probe.txPool.Stats()
}

func (b *ProbeAPIBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.probe.TxPool().Content()
}

func (b *ProbeAPIBackend) TxPool() *core.TxPool {
	return b.probe.TxPool()
}

func (b *ProbeAPIBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.probe.TxPool().SubscribeNewTxsEvent(ch)
}

func (b *ProbeAPIBackend) Downloader() *downloader.Downloader {
	return b.probe.Downloader()
}

func (b *ProbeAPIBackend) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestTipCap(ctx)
}

func (b *ProbeAPIBackend) FeeHistory(ctx context.Context, blockCount int, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (firstBlock rpc.BlockNumber, reward [][]*big.Int, baseFee []*big.Int, gasUsedRatio []float64, err error) {
	return b.gpo.FeeHistory(ctx, blockCount, lastBlock, rewardPercentiles)
}

func (b *ProbeAPIBackend) ChainDb() probedb.Database {
	return b.probe.ChainDb()
}

func (b *ProbeAPIBackend) EventMux() *event.TypeMux {
	return b.probe.EventMux()
}

func (b *ProbeAPIBackend) AccountManager() *accounts.Manager {
	return b.probe.AccountManager()
}

func (b *ProbeAPIBackend) ExtRPCEnabled() bool {
	return b.extRPCEnabled
}

func (b *ProbeAPIBackend) UnprotectedAllowed() bool {
	return b.allowUnprotectedTxs
}

func (b *ProbeAPIBackend) RPCGasCap() uint64 {
	return b.probe.config.RPCGasCap
}

func (b *ProbeAPIBackend) RPCTxFeeCap() float64 {
	return b.probe.config.RPCTxFeeCap
}

func (b *ProbeAPIBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.probe.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *ProbeAPIBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.probe.bloomRequests)
	}
}

func (b *ProbeAPIBackend) Engine() consensus.Engine {
	return b.probe.engine
}

func (b *ProbeAPIBackend) CurrentHeader() *types.Header {
	return b.probe.blockchain.CurrentHeader()
}

func (b *ProbeAPIBackend) Miner() *miner.Miner {
	return b.probe.Miner()
}

func (b *ProbeAPIBackend) StartMining(threads int) error {
	return b.probe.StartMining(threads)
}

func (b *ProbeAPIBackend) StateAtBlock(ctx context.Context, block *types.Block, reexec uint64, base *state.StateDB, checkLive bool) (*state.StateDB, error) {
	return b.probe.stateAtBlock(block, reexec, base, checkLive)
}

func (b *ProbeAPIBackend) StateAtTransaction(ctx context.Context, block *types.Block, txIndex int, reexec uint64) (core.Message, vm.BlockContext, *state.StateDB, error) {
	return b.probe.stateAtTransaction(block, txIndex, reexec)
}

func (b *ProbeAPIBackend) Exist(addr common.Address) bool {
	return b.TxPool().Exist(addr)
}

// Copyright 2016 The go-probeum Authors
// This file is part of the go-probeum library.
//
// The go-probeum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-probeum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-probeum library. If not, see <http://www.gnu.org/licenses/>.

package les

import (
	"context"
	"errors"
	"math/big"

	"github.com/probeum/go-probeum/accounts"
	"github.com/probeum/go-probeum/common"
	"github.com/probeum/go-probeum/consensus"
	"github.com/probeum/go-probeum/core"
	"github.com/probeum/go-probeum/core/bloombits"
	"github.com/probeum/go-probeum/core/rawdb"
	"github.com/probeum/go-probeum/core/state"
	"github.com/probeum/go-probeum/core/types"
	"github.com/probeum/go-probeum/core/vm"
	"github.com/probeum/go-probeum/probe/downloader"
	"github.com/probeum/go-probeum/probe/gasprice"
	"github.com/probeum/go-probeum/probedb"
	"github.com/probeum/go-probeum/event"
	"github.com/probeum/go-probeum/light"
	"github.com/probeum/go-probeum/params"
	"github.com/probeum/go-probeum/rpc"
)

type LesApiBackend struct {
	extRPCEnabled       bool
	allowUnprotectedTxs bool
	probe                 *LightProbeum
	gpo                 *gasprice.Oracle
}

func (b *LesApiBackend) ChainConfig() *params.ChainConfig {
	return b.probe.chainConfig
}

func (b *LesApiBackend) CurrentBlock() *types.Block {
	return types.NewBlockWithHeader(b.probe.BlockChain().CurrentHeader())
}

func (b *LesApiBackend) SetHead(number uint64) {
	b.probe.handler.downloader.Cancel()
	b.probe.blockchain.SetHead(number)
}

func (b *LesApiBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	if number == rpc.PendingBlockNumber {
		return nil, nil
	}
	if number == rpc.LatestBlockNumber {
		return b.probe.blockchain.CurrentHeader(), nil
	}
	return b.probe.blockchain.GetHeaderByNumberOdr(ctx, uint64(number))
}

func (b *LesApiBackend) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.HeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header, err := b.HeaderByHash(ctx, hash)
		if err != nil {
			return nil, err
		}
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

func (b *LesApiBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.probe.blockchain.GetHeaderByHash(hash), nil
}

func (b *LesApiBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	header, err := b.HeaderByNumber(ctx, number)
	if header == nil || err != nil {
		return nil, err
	}
	return b.BlockByHash(ctx, header.Hash())
}

func (b *LesApiBackend) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.probe.blockchain.GetBlockByHash(ctx, hash)
}

func (b *LesApiBackend) BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.BlockByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		block, err := b.BlockByHash(ctx, hash)
		if err != nil {
			return nil, err
		}
		if block == nil {
			return nil, errors.New("header found, but block body is missing")
		}
		if blockNrOrHash.RequireCanonical && b.probe.blockchain.GetCanonicalHash(block.NumberU64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		return block, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *LesApiBackend) PendingBlockAndReceipts() (*types.Block, types.Receipts) {
	return nil, nil
}

func (b *LesApiBackend) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	header, err := b.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, nil, err
	}
	if header == nil {
		return nil, nil, errors.New("header not found")
	}
	return light.NewState(ctx, header, b.probe.odr), header, nil
}

func (b *LesApiBackend) StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.StateAndHeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.probe.blockchain.GetHeaderByHash(hash)
		if header == nil {
			return nil, nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.probe.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, nil, errors.New("hash is not currently canonical")
		}
		return light.NewState(ctx, header, b.probe.odr), header, nil
	}
	return nil, nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *LesApiBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	if number := rawdb.ReadHeaderNumber(b.probe.chainDb, hash); number != nil {
		return light.GetBlockReceipts(ctx, b.probe.odr, hash, *number)
	}
	return nil, nil
}

func (b *LesApiBackend) GetLogs(ctx context.Context, hash common.Hash) ([][]*types.Log, error) {
	if number := rawdb.ReadHeaderNumber(b.probe.chainDb, hash); number != nil {
		return light.GetBlockLogs(ctx, b.probe.odr, hash, *number)
	}
	return nil, nil
}

func (b *LesApiBackend) GetTd(ctx context.Context, hash common.Hash) *big.Int {
	if number := rawdb.ReadHeaderNumber(b.probe.chainDb, hash); number != nil {
		return b.probe.blockchain.GetTdOdr(ctx, hash, *number)
	}
	return nil
}

func (b *LesApiBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmConfig *vm.Config) (*vm.EVM, func() error, error) {
	if vmConfig == nil {
		vmConfig = new(vm.Config)
	}
	txContext := core.NewEVMTxContext(msg)
	context := core.NewEVMBlockContext(header, b.probe.blockchain, nil)
	return vm.NewEVM(context, txContext, state, b.probe.chainConfig, *vmConfig), state.Error, nil
}

func (b *LesApiBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.probe.txPool.Add(ctx, signedTx)
}

func (b *LesApiBackend) RemoveTx(txHash common.Hash) {
	b.probe.txPool.RemoveTx(txHash)
}

func (b *LesApiBackend) GetPoolTransactions() (types.Transactions, error) {
	return b.probe.txPool.GetTransactions()
}

func (b *LesApiBackend) GetPoolTransaction(txHash common.Hash) *types.Transaction {
	return b.probe.txPool.GetTransaction(txHash)
}

func (b *LesApiBackend) GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, common.Hash, uint64, uint64, error) {
	return light.GetTransaction(ctx, b.probe.odr, txHash)
}

func (b *LesApiBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.probe.txPool.GetNonce(ctx, addr)
}

func (b *LesApiBackend) Stats() (pending int, queued int) {
	return b.probe.txPool.Stats(), 0
}

func (b *LesApiBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.probe.txPool.Content()
}

func (b *LesApiBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.probe.txPool.SubscribeNewTxsEvent(ch)
}

func (b *LesApiBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.probe.blockchain.SubscribeChainEvent(ch)
}

func (b *LesApiBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.probe.blockchain.SubscribeChainHeadEvent(ch)
}

func (b *LesApiBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.probe.blockchain.SubscribeChainSideEvent(ch)
}

func (b *LesApiBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.probe.blockchain.SubscribeLogsEvent(ch)
}

func (b *LesApiBackend) SubscribePendingLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return event.NewSubscription(func(quit <-chan struct{}) error {
		<-quit
		return nil
	})
}

func (b *LesApiBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.probe.blockchain.SubscribeRemovedLogsEvent(ch)
}

func (b *LesApiBackend) Downloader() *downloader.Downloader {
	return b.probe.Downloader()
}

func (b *LesApiBackend) ProtocolVersion() int {
	return b.probe.LesVersion() + 10000
}

func (b *LesApiBackend) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestTipCap(ctx)
}

func (b *LesApiBackend) FeeHistory(ctx context.Context, blockCount int, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (firstBlock rpc.BlockNumber, reward [][]*big.Int, baseFee []*big.Int, gasUsedRatio []float64, err error) {
	return b.gpo.FeeHistory(ctx, blockCount, lastBlock, rewardPercentiles)
}

func (b *LesApiBackend) ChainDb() probedb.Database {
	return b.probe.chainDb
}

func (b *LesApiBackend) AccountManager() *accounts.Manager {
	return b.probe.accountManager
}

func (b *LesApiBackend) ExtRPCEnabled() bool {
	return b.extRPCEnabled
}

func (b *LesApiBackend) UnprotectedAllowed() bool {
	return b.allowUnprotectedTxs
}

func (b *LesApiBackend) RPCGasCap() uint64 {
	return b.probe.config.RPCGasCap
}

func (b *LesApiBackend) RPCTxFeeCap() float64 {
	return b.probe.config.RPCTxFeeCap
}

func (b *LesApiBackend) BloomStatus() (uint64, uint64) {
	if b.probe.bloomIndexer == nil {
		return 0, 0
	}
	sections, _, _ := b.probe.bloomIndexer.Sections()
	return params.BloomBitsBlocksClient, sections
}

func (b *LesApiBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.probe.bloomRequests)
	}
}

func (b *LesApiBackend) Engine() consensus.Engine {
	return b.probe.engine
}

func (b *LesApiBackend) CurrentHeader() *types.Header {
	return b.probe.blockchain.CurrentHeader()
}

func (b *LesApiBackend) StateAtBlock(ctx context.Context, block *types.Block, reexec uint64, base *state.StateDB, checkLive bool) (*state.StateDB, error) {
	return b.probe.stateAtBlock(ctx, block, reexec)
}

func (b *LesApiBackend) StateAtTransaction(ctx context.Context, block *types.Block, txIndex int, reexec uint64) (core.Message, vm.BlockContext, *state.StateDB, error) {
	return b.probe.stateAtTransaction(ctx, block, txIndex, reexec)
}

func (b *LesApiBackend) Exist(addr common.Address) bool {
	return false
}
func (b *LesApiBackend) DposAccounts(number rpc.BlockNumber) []*common.DPoSAccount {
	return nil
}

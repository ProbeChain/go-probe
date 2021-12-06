// Copyright 2015 The go-probeum Authors
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

package miner

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	greatri2 "github.com/probeum/go-probeum/consensus/greatri"
	probehash2 "github.com/probeum/go-probeum/consensus/probeash"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/probeum/go-probeum/common"
	"github.com/probeum/go-probeum/consensus"
	"github.com/probeum/go-probeum/core"
	"github.com/probeum/go-probeum/core/state"
	"github.com/probeum/go-probeum/core/types"
	"github.com/probeum/go-probeum/event"
	"github.com/probeum/go-probeum/log"
	"github.com/probeum/go-probeum/params"
	"github.com/probeum/go-probeum/trie"
)

const (

	// resultQueueSize is the size of channel listening to sealing result.
	resultQueueSize = 10

	// txChanSize is the size of channel listening to NewTxsEvent.
	// The number is referenced from the size of tx pool.
	txChanSize = 4096

	// chainHeadChanSize is the size of channel listening to ChainHeadEvent.
	chainHeadChanSize = 10

	// chainSideChanSize is the size of channel listening to ChainSideEvent.
	chainSideChanSize = 10

	// powAnswerChanSize is the size of channel listening to PowAnswerEvent.
	powAnswerChanSize = 10

	// resubmitAdjustChanSize is the size of resubmitting interval adjustment channel.
	resubmitAdjustChanSize = 10

	// powAnswerChanSize is the size of resubmitting interval adjustment channel.
	powMinerResultChanSize = 10

	// miningLogAtDepth is the number of confirmations before logging successful mining.
	miningLogAtDepth = 7

	// minRecommitInterval is the minimal time interval to recreate the mining block with
	// any newly arrived transactions.
	minRecommitInterval = 1 * time.Second

	// maxRecommitInterval is the maximum time interval to recreate the mining block with
	// any newly arrived transactions.
	maxRecommitInterval = 15 * time.Second

	// intervalAdjustRatio is the impact a single interval adjustment has on sealing work
	// resubmitting interval.
	intervalAdjustRatio = 0.1

	// intervalAdjustBias is applied during the new resubmit interval calculation in favor of
	// increasing upper limit or decreasing lower limit so that the limit can be reachable.
	intervalAdjustBias = 200 * 1000.0 * 1000.0

	// staleThreshold is the maximum depth of the acceptable stale block.
	staleThreshold = 7
)

var (
	// DposWitnessNumber is the total number of dpos witness nodes.
	//@todo just for oneNode test
	DposWitnessNumber uint = 5

	// MostDposWitness number of witness to product stabilizing block
	MostDposWitness = DposWitnessNumber*2/3 + 1

	// LeastDposWitness the least number of witness to product block
	//LeastDposWitness = DposWitnessNumber*1/3 + 1
	LeastDposWitness = DposWitnessNumber*1/2 + 1

	// dposAckChanSize is the size of channel listening to DposAckEvent.
	dposAckChanSize = DposWitnessNumber * 10

	// min diffcult
	minDifficulty int64 = 5000000
)

// environment is the worker's current environment and holds all of the current state information.
type environment struct {
	signer types.Signer

	state     *state.StateDB // apply state changes here
	ancestors mapset.Set     // ancestor set (used for checking uncle parent validity)
	family    mapset.Set     // family set (used for checking uncle invalidity)
	uncles    mapset.Set     // uncle set
	tcount    int            // tx count in cycle
	gasPool   *core.GasPool  // available gas used to pack transactions

	header          *types.Header
	txs             []*types.Transaction
	powAnswerUncles []*types.PowAnswer
	dposAcks        []*types.DposAck

	receipts []*types.Receipt
}

// task contains all information for consensus engine sealing and result submitting.
type task struct {
	receipts  []*types.Receipt
	state     *state.StateDB
	block     *types.Block
	createdAt time.Time
}

const (
	commitInterruptNone int32 = iota
	commitInterruptNewHead
	commitInterruptResubmit
)

// newWorkReq represents a request for new sealing work submitting with relative interrupt notifier.
type newWorkReq struct {
	interrupt                *int32
	noempty                  bool
	currentEffectBlockNumber *big.Int
	newBlockNumber           *big.Int
	newBlockType             types.BlockType
}

// intervalAdjust represents a resubmitting interval adjustment.
type intervalAdjust struct {
	ratio float64
	inc   bool
}

// worker is the main object which takes care of submitting new work to consensus engine
// and gathering the sealing result.
type worker struct {
	config      *Config
	chainConfig *params.ChainConfig
	engine      consensus.Engine
	powEngine   consensus.Engine
	probe       Backend
	chain       *core.BlockChain

	// Feeds
	pendingLogsFeed event.Feed

	// Subscriptions
	mux          *event.TypeMux
	txsCh        chan core.NewTxsEvent
	txsSub       event.Subscription
	chainHeadCh  chan core.ChainHeadEvent
	chainHeadSub event.Subscription
	chainSideCh  chan core.ChainSideEvent
	chainSideSub event.Subscription

	powAnswerCh  chan core.PowAnswerEvent
	powAnswerSub event.Subscription

	dposAckCh  chan core.DposAckEvent
	dposAckSub event.Subscription

	// Channels
	newWorkCh          chan *newWorkReq
	taskCh             chan *task
	startCh            chan struct{}
	exitCh             chan struct{}
	resubmitIntervalCh chan time.Duration
	resubmitAdjustCh   chan *intervalAdjust

	//pow miner
	powMinerResultCh chan *types.PowAnswer

	current      *environment                 // An environment for current running cycle.
	localUncles  map[common.Hash]*types.Block // A set of side blocks generated locally as the possible uncle blocks.
	remoteUncles map[common.Hash]*types.Block // A set of side blocks as the possible uncle blocks.
	unconfirmed  *unconfirmedBlocks           // A set of locally mined blocks pending canonicalness confirmations.

	mu        sync.RWMutex // The lock used to protect the coinbase and extra fields
	muProduce sync.RWMutex
	coinbase  common.Address
	extra     []byte

	pendingMu    sync.RWMutex
	pendingTasks map[common.Hash]*task

	snapshotMu       sync.RWMutex // The lock used to protect the block snapshot and state snapshot
	snapshotBlock    *types.Block
	snapshotReceipts types.Receipts
	snapshotState    *state.StateDB

	// atomic status counters
	running int32 // The indicator whprobeer the consensus engine is running or not.
	//newTxs  int32 // New arrival transaction count since last sealing work submitting.

	// noempty is the flag used to control whprobeer the feature of pre-seal empty
	// block is enabled. The default value is false(pre-seal is enabled by default).
	// But in some special scenario the consensus engine will seal blocks instantaneously,
	// in this case this feature will add all empty blocks into canonical chain
	// non-stop and no real transaction will be included.
	noempty uint32

	// External functions
	isLocalBlock func(block *types.Block) bool // Function used to determine whprobeer the specified block is mined by local miner.

	// Test hooks
	newTaskHook          func(*task)                        // Method to call upon receiving a new sealing task.
	skipSealHook         func(*task) bool                   // Method to decide whether skipping the sealing.
	fullTaskHook         func()                             // Method to call before pushing the full sealing task.
	resubmitHook         func(time.Duration, time.Duration) // Method to call upon updating resubmitting interval.
	visualBlockNumber    *big.Int
	effectBlockNumber    *big.Int
	delaySealBlockNumber *big.Int
}

func newWorker(config *Config, chainConfig *params.ChainConfig, engine consensus.Engine, powEngine consensus.Engine, probe Backend, mux *event.TypeMux,
	isLocalBlock func(*types.Block) bool, init bool) *worker {
	chain := probe.BlockChain()
	number := chain.CurrentHeader().Number.Uint64()
	size := chain.GetDposAccountSize(number)
	updateDposParams(size)
	log.Info("UpdateDposParams newWorker", "blockNumber", number, "size", size, "DposWitnessNumber", DposWitnessNumber,
		"MostDposWitness", MostDposWitness, "LeastDposWitness", LeastDposWitness)

	worker := &worker{
		config:               config,
		chainConfig:          chainConfig,
		coinbase:             config.Probeerbase,
		engine:               engine,
		powEngine:            powEngine,
		probe:                probe,
		mux:                  mux,
		chain:                chain,
		isLocalBlock:         isLocalBlock,
		localUncles:          make(map[common.Hash]*types.Block),
		remoteUncles:         make(map[common.Hash]*types.Block),
		unconfirmed:          newUnconfirmedBlocks(probe.BlockChain(), miningLogAtDepth),
		pendingTasks:         make(map[common.Hash]*task),
		txsCh:                make(chan core.NewTxsEvent, txChanSize),
		chainHeadCh:          make(chan core.ChainHeadEvent, chainHeadChanSize),
		chainSideCh:          make(chan core.ChainSideEvent, chainSideChanSize),
		powAnswerCh:          make(chan core.PowAnswerEvent, powAnswerChanSize),
		dposAckCh:            make(chan core.DposAckEvent, dposAckChanSize),
		newWorkCh:            make(chan *newWorkReq),
		taskCh:               make(chan *task),
		exitCh:               make(chan struct{}),
		startCh:              make(chan struct{}, 1),
		resubmitIntervalCh:   make(chan time.Duration),
		resubmitAdjustCh:     make(chan *intervalAdjust, resubmitAdjustChanSize),
		powMinerResultCh:     make(chan *types.PowAnswer, powMinerResultChanSize),
		visualBlockNumber:    new(big.Int).SetUint64(0),
		effectBlockNumber:    new(big.Int).SetUint64(0),
		delaySealBlockNumber: new(big.Int).SetUint64(0),
	}
	// Subscribe NewTxsEvent for tx pool
	worker.txsSub = probe.TxPool().SubscribeNewTxsEvent(worker.txsCh)
	// Subscribe events for blockchain
	worker.chainHeadSub = probe.BlockChain().SubscribeChainHeadEvent(worker.chainHeadCh)
	worker.chainSideSub = probe.BlockChain().SubscribeChainSideEvent(worker.chainSideCh)

	worker.powAnswerSub = probe.BlockChain().SubscribePowAnswerEvent(worker.powAnswerCh)
	worker.dposAckSub = probe.BlockChain().SubscribeDposAckEvent(worker.dposAckCh)

	// Sanitize recommit interval if the user-specified one is too short.
	recommit := worker.config.Recommit
	if recommit < minRecommitInterval {
		log.Warn("Sanitizing miner recommit interval", "provided", recommit, "updated", minRecommitInterval)
		recommit = minRecommitInterval
	}

	go worker.mainLoop()
	go worker.newWorkLoop(recommit)
	go worker.taskLoop()
	go worker.powMinerNewWorkLoop()
	go worker.powMinerResultLoop()

	// Submit first work to initialize pending state.
	if init {
		//worker.startCh <- struct{}{}
	}
	return worker
}

// setProbeerbase sets the probeerbase used to initialize the block coinbase field.
func (w *worker) setProbeerbase(addr common.Address) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.coinbase = addr
}

// setExtra sets the content used to initialize the block extra field.
func (w *worker) setExtra(extra []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.extra = extra
}

// setRecommitInterval updates the interval for miner sealing work recommitting.
func (w *worker) setRecommitInterval(interval time.Duration) {
	w.resubmitIntervalCh <- interval
}

// disablePreseal disables pre-sealing mining feature
func (w *worker) disablePreseal() {
	atomic.StoreUint32(&w.noempty, 1)
}

// enablePreseal enables pre-sealing mining feature
func (w *worker) enablePreseal() {
	atomic.StoreUint32(&w.noempty, 0)
}

// pending returns the pending state and corresponding block.
func (w *worker) pending() (*types.Block, *state.StateDB) {
	s, _ := w.chain.StateAt(w.chain.CurrentBlock().Root())
	return w.chain.CurrentBlock(), s
	//// return a snapshot to avoid contention on currentMu mutex
	//w.snapshotMu.RLock()
	//defer w.snapshotMu.RUnlock()
	//if w.snapshotState == nil {
	//	return nil, nil
	//}
	//return w.snapshotBlock, w.snapshotState.Copy()
}

// pendingBlock returns pending block.
func (w *worker) pendingBlock() *types.Block {
	// return a snapshot to avoid contention on currentMu mutex
	w.snapshotMu.RLock()
	defer w.snapshotMu.RUnlock()
	return w.snapshotBlock
}

// pendingBlockAndReceipts returns pending block and corresponding receipts.
func (w *worker) pendingBlockAndReceipts() (*types.Block, types.Receipts) {
	// return a snapshot to avoid contention on currentMu mutex
	w.snapshotMu.RLock()
	defer w.snapshotMu.RUnlock()
	return w.snapshotBlock, w.snapshotReceipts
}

// start sets the running status as 1 and triggers new work submitting.
func (w *worker) start() {
	atomic.StoreInt32(&w.running, 1)
	w.startCh <- struct{}{}
}

// stop sets the running status as 0.
func (w *worker) stop() {
	atomic.StoreInt32(&w.running, 0)
}

// isRunning returns an indicator whprobeer worker is running or not.
func (w *worker) isRunning() bool {
	return atomic.LoadInt32(&w.running) == 1
}

func (w *worker) imProducerOnSpecBlock(blockNumber uint64) bool {
	account := w.chain.GetSealDposAccount(blockNumber)
	if account == nil {
		log.Error("somprobeing wrong in get dpos account, neeQd to check", "blockNumber", blockNumber)
		return false
	}
	log.Debug("producer ", "blockNumber:", blockNumber, "mine:", w.coinbase, " curOwner:", account.Owner, " curNode:", account.Enode.String(), " eq:", account.Owner == w.coinbase)
	return account.Owner == w.coinbase
}

func (w *worker) imProducer(blockNumber uint64) bool {
	return w.imProducerOnSpecBlock(blockNumber)
}

func (w *worker) imDposWorkNode(blockNumber *big.Int) bool {
	ret := w.chain.CheckIsDposAccount(blockNumber.Uint64(), w.coinbase)
	log.Trace("check if i'm dpos node", "ret", ret, "block Number", blockNumber.Uint64(), "addr", w.coinbase)
	return ret
}

// close terminates all background threads maintained by the worker.
// Note the worker does not support being closed multiple times.
func (w *worker) close() {
	if w.current != nil && w.current.state != nil {
		w.current.state.StopPrefetcher()
	}
	atomic.StoreInt32(&w.running, 0)
	close(w.exitCh)
}

// recalcRecommit recalculates the resubmitting interval upon feedback.
func recalcRecommit(minRecommit, prev time.Duration, target float64, inc bool) time.Duration {
	var (
		prevF = float64(prev.Nanoseconds())
		next  float64
	)
	if inc {
		next = prevF*(1-intervalAdjustRatio) + intervalAdjustRatio*(target+intervalAdjustBias)
		max := float64(maxRecommitInterval.Nanoseconds())
		if next > max {
			next = max
		}
	} else {
		next = prevF*(1-intervalAdjustRatio) + intervalAdjustRatio*(target-intervalAdjustBias)
		min := float64(minRecommit.Nanoseconds())
		if next < min {
			next = min
		}
	}
	return time.Duration(int64(next))
}

func (w *worker) sendAck(blockNumber uint64, ackType types.DposAckType) error {
	position, err := w.chain.GetDposAccountIndex(blockNumber, w.coinbase)
	if err != nil {
		log.Error("somprobeing wrong in get dpos account postion", "blockNumber", blockNumber)
		return err
	}
	ack := &types.DposAck{
		EpochPosition: uint8(position),
		Number:        new(big.Int).SetUint64(blockNumber),
		AckType:       ackType,
	}
	if blockNumber <= w.chain.CurrentBlock().NumberU64() && ackType == types.AckTypeAgree {
		ack.BlockHash = w.probe.BlockChain().GetBlockByNumber(blockNumber).Hash()
	} else {
		ack.BlockHash = types.EmptyUncleHash
	}

	greatri, ok := w.engine.(*greatri2.Greatri)
	if !ok {
		log.Error("somprobeing wrong in produce dposAck", "blockNumber", blockNumber)
		return err
	}
	ackSig, err := greatri.DposAckSig(ack)
	if err != nil {
		log.Error("somprobeing wrong in DposAckSig", "blockNumber", blockNumber)
		return err
	}
	ack.WitnessSig = append(ack.WitnessSig, ackSig...)

	w.mux.Post(core.DposAckEvent{DposAck: ack})
	return nil
}

//func (w *worker) getCurrentBlock() (*big.Int, *big.Int) {
//	effectBlockNumber := w.chain.CurrentBlock().Number()
//	if effectBlockNumber.Cmp(w.visualBlockNumber) < 0 {
//		return w.visualBlockNumber, effectBlockNumber
//	} else {
//		return effectBlockNumber, effectBlockNumber
//	}
//}

func (w *worker) checkPowAnswerNumber(blockNumber *big.Int, effectBlockNumber *big.Int) (bool, int) {
	visualBlockCount := new(big.Int)
	visualBlockCount.Sub(blockNumber, effectBlockNumber)
	answerNumber := len(w.chain.GetPowAnswers(effectBlockNumber))
	log.Debug("check answer", "answerNumber", answerNumber, "int(visualBlockCount.Uint64() + 1)", int(visualBlockCount.Uint64()+1))
	return answerNumber >= int(visualBlockCount.Uint64()+1), answerNumber
}

func (w *worker) printBlock(block *types.Block) {
	bs, err1 := json.Marshal(block.Header())
	if err1 != nil {
		log.Info("json encode failed")
	}
	var out bytes.Buffer
	json.Indent(&out, bs, "", "\t")
	log.Trace("new block header in step 3:", out.String(), nil)

	bs2, err2 := json.Marshal(block.Body())
	if err2 != nil {
		log.Info("json encode failed")
	}
	var out2 bytes.Buffer
	json.Indent(&out2, bs2, "", "\t")
	log.Trace("new block body in step 3:", out2.String(), nil)
}

// newWorkLoop is a standalone goroutine to submit new mining work upon received events.
func (w *worker) newWorkLoop(recommit time.Duration) {
	time.Sleep(2 * time.Second)

	var (
		interrupt               *int32
		minRecommit             = recommit // minimal resubmit interval specified by user.
		stopCh                  chan struct{}
		timerSealDealineSetFlag = false
		timerDelaySealSetFlag   = false
		rejectBlockNumber       = new(big.Int).SetUint64(0)
		sealedBlockNumber       uint64
	)

	//used when not received enough dposAck
	timerDelaySeal := time.NewTimer(0)
	defer timerDelaySeal.Stop()
	<-timerDelaySeal.C // discard the initial tick

	//deadline of seal after received a pow answer
	timerSealDealine := time.NewTimer(0)
	defer timerSealDealine.Stop()
	<-timerSealDealine.C // discard the initial tick

	// commit aborts in-flight transaction execution with given signal and resubmits a new one.
	commit := func(noempty bool, s int32, newBlockNumber uint64) bool {
		if sealedBlockNumber >= newBlockNumber {
			log.Info("can't produce the same block number", "blockNumber", newBlockNumber)
			return false
		}
		sealedBlockNumber = newBlockNumber
		log.Debug("sealedBlockNumber changed", "sealedBlockNumber", sealedBlockNumber)

		newBlockType := types.BlockTypeVisual
		if newBlockNumber == w.visualBlockNumber.Uint64()+1 {
			newBlockType = types.BlockTypeEffect
		}
		//interrupt the flying pack work;
		if interrupt != nil {
			atomic.StoreInt32(interrupt, s)
		}
		interrupt = new(int32)
		select {
		case w.newWorkCh <- &newWorkReq{interrupt: interrupt, noempty: noempty,
			currentEffectBlockNumber: w.effectBlockNumber, newBlockNumber: new(big.Int).SetUint64(newBlockNumber), newBlockType: newBlockType}:
			log.Debug("commit new work", "newBlockNumber", newBlockNumber, "blockType", newBlockType)
		case <-w.exitCh:
			return false
		}
		return true
	}
	// clearPending cleans the stale pending tasks.
	clearPending := func(number uint64) {
		w.pendingMu.Lock()
		for h, t := range w.pendingTasks {
			if t.block.NumberU64()+staleThreshold <= number {
				delete(w.pendingTasks, h)
			}
		}
		w.pendingMu.Unlock()
	}
	// interrupt aborts the in-flight sealing task.
	interruptSeal := func() {
		if stopCh != nil {
			close(stopCh)
			stopCh = nil
		}
	}

	for {
		select {
		case <-w.startCh:
			clearPending(w.chain.CurrentBlock().NumberU64())
			//commit(false, commitInterruptNewHead)
			time.Sleep(2 * time.Second)
			w.chainHeadCh <- core.ChainHeadEvent{}

		case block := <-w.chainHeadCh:
			var blockNumber *big.Int
			if nil == block.Block {
				blockNumber = w.chain.CurrentBlock().Number()
			} else {
				blockNumber = block.Block.Number()
			}
			newBlock := w.chain.GetBlockByNumber(blockNumber.Uint64())

			log.Trace("worker received new block", "blockNumber", blockNumber)
			if blockNumber.Uint64() != 0 && blockNumber.Uint64() <= w.visualBlockNumber.Uint64() {
				log.Debug("received a block that we have reject",
					"current block number", w.visualBlockNumber, "effectBlockNumber", w.effectBlockNumber, "visualBlockNumber", w.visualBlockNumber)
				if newBlock.Header().DposSigAddr == w.coinbase && blockNumber.Uint64() > w.effectBlockNumber.Uint64() {
					if commit(false, commitInterruptNewHead, blockNumber.Uint64()+1) {
						log.Info("my job to continue producing block", "addr", w.coinbase,
							"current block number", w.visualBlockNumber, "effectBlockNumber", w.effectBlockNumber, "visualBlockNumber", w.visualBlockNumber)
					}
				}
				continue
			}
			w.visualBlockNumber.Set(blockNumber)
			w.effectBlockNumber.Set(blockNumber)
			log.Trace("block number update", "effectBlockNumber", w.effectBlockNumber, "currentBlock",
				w.chain.CurrentBlock().Number(), "visualBlockNumber", w.visualBlockNumber)
			if blockNumber.Uint64() >= rejectBlockNumber.Uint64() {
				timerSealDealine.Stop()
				timerSealDealineSetFlag = false
			}
			//if w.imProducer(blockNumber.Uint64() + 1) {
			//	dposAgreeAckNum := w.chain.GetDposAckSize(blockNumber, types.AckTypeAgree)
			//	if nil != w.chain.GetLatestPowAnswer(blockNumber) &&
			//		dposAgreeAckNum >= int(MostDposWitness) {
			//		//received powAnswer and received 2/3 witness node ack
			//		if commit(false, commitInterruptNewHead, blockNumber) {
			//			log.Info("received new base block, is my turn to produce new block", "addr", w.coinbase,
			//				"current block number", blockNumber)
			//		}
			//	}
			//}

			if w.imDposWorkNode(blockNumber) {
				//todo: send oppose ack if i received a disorganized block
				if nil == w.sendAck(blockNumber.Uint64(), types.AckTypeAgree) {
					log.Info("received new block, send dposAck to all success", "addr", w.coinbase,
						"current block number", blockNumber)
				} else {
					log.Info("received new block, send dposAck to all failed", "addr", w.coinbase,
						"current block number", blockNumber)
				}
			}

			if w.chainConfig.Probeash != nil {
				interruptSeal()
				stopCh = make(chan struct{})
				probeash, ok := w.powEngine.(*probehash2.Probeash)
				if !ok {
					log.Error("somprobeing wrong in produce dposAck", "blockNumber", blockNumber)
					continue
				}

				if err := probeash.PowSeal(w.chain, newBlock, w.powMinerResultCh, stopCh, w.coinbase); err != nil {
					log.Warn("pow Miner failed", "err", err)
				}
			}

		case ack := <-w.dposAckCh:
			log.Trace("receive ack", "blockNumber", ack.DposAck.Number, "type", ack.DposAck.AckType, "sig", hex.EncodeToString(ack.DposAck.WitnessSig))
			if w.visualBlockNumber.Uint64() != ack.DposAck.Number.Uint64() {
				continue
			}

			dposAgreeAckNum := w.chain.GetDposAckSize(w.visualBlockNumber, types.AckTypeAgree)
			dposOpposeAckNum := w.chain.GetDposAckSize(w.visualBlockNumber, types.AckTypeOppose)
			if w.imProducer(w.visualBlockNumber.Uint64() + 1) {
				//is the producer
				ret, answerNumber := w.checkPowAnswerNumber(w.visualBlockNumber, w.effectBlockNumber)
				if ret && (dposAgreeAckNum >= int(MostDposWitness) || dposOpposeAckNum >= int(MostDposWitness)) {
					//received powAnswer and received 2/3 witness node ack
					timerDelaySeal.Stop()
					if commit(false, commitInterruptNewHead, w.effectBlockNumber.Uint64()+1) {
						log.Info("received enough dposAcks, is my turn to produce new block", "addr", w.coinbase,
							"current block number", w.visualBlockNumber)
					}
				} else if !timerDelaySealSetFlag && ret && (dposAgreeAckNum >= int(LeastDposWitness) || dposOpposeAckNum >= int(LeastDposWitness)) {
					timerDelaySealSetFlag = true
					timerDelaySeal.Reset(consensus.Time2delaySeal * time.Second)
					w.delaySealBlockNumber.Add(w.effectBlockNumber, common.Big1)
					log.Debug("received dposAck, but not enough dposAck, wait...", "addr", w.coinbase, "waitTime", consensus.Time2delaySeal,
						"effectBlockNumber", w.effectBlockNumber, "visualBlockNumber", w.visualBlockNumber,
						"dposAgreeAckNum", dposAgreeAckNum, "dposOpposeAckNum", dposOpposeAckNum, "answerNumber", answerNumber)
				} else {
					log.Trace("i'm producer, but nothing to do", "timerDelaySealSetFlag", timerDelaySealSetFlag, "answer check", ret,
						"dposAgreeAckNum", dposAgreeAckNum, "dposOpposeAckNum", dposOpposeAckNum)
				}
			} else if w.imDposWorkNode(w.visualBlockNumber) {
				ret, answerNumber := w.checkPowAnswerNumber(w.visualBlockNumber, w.effectBlockNumber)
				if !timerSealDealineSetFlag && ret &&
					(dposAgreeAckNum >= int(LeastDposWitness) || dposOpposeAckNum >= int(LeastDposWitness)) {
					timerSealDealineSetFlag = true
					rejectBlockNumber.Add(w.visualBlockNumber, common.Big1)
					timerSealDealine.Reset(consensus.Time2SealDeadline * time.Second)
					log.Debug("received enough acks, not my turn to produce, setup a timer to wait new block", "addr", w.coinbase,
						"effectBlockNumber", w.effectBlockNumber, "visualBlockNumber", w.visualBlockNumber,
						"dposAgreeAckNum", dposAgreeAckNum, "dposOpposeAckNum", dposOpposeAckNum, "answerNumber", answerNumber)
				} else {
					log.Trace("i'm dpos node, but nothing to do", "timerSealDealineSetFlag", timerSealDealineSetFlag, "answer check", ret,
						"dposAgreeAckNum", dposAgreeAckNum, "dposOpposeAckNum", dposOpposeAckNum)
				}
			}

		case answer := <-w.powAnswerCh:
			log.Trace("receive answer", "blockNumber", answer.PowAnswer.Number, "miner", answer.PowAnswer.Miner)
			if w.effectBlockNumber.Uint64() != answer.PowAnswer.Number.Uint64() {
				log.Debug("the block number in powAnser is mismatch", "effectBlockNumber", w.effectBlockNumber.Uint64(),
					"powAnswer block number", answer.PowAnswer.Number.Uint64())
				continue
			}

			dposAgreeAckNum := w.chain.GetDposAckSize(w.visualBlockNumber, types.AckTypeAgree)
			dposOpposeAckNum := w.chain.GetDposAckSize(w.visualBlockNumber, types.AckTypeOppose)
			if w.imProducer(w.visualBlockNumber.Uint64() + 1) {
				//is the producer
				ret, answerNumber := w.checkPowAnswerNumber(w.visualBlockNumber, w.effectBlockNumber)
				if ret && (dposAgreeAckNum >= int(MostDposWitness) || dposOpposeAckNum >= int(MostDposWitness)) {
					//received 2/3 witness node ack
					timerDelaySeal.Stop()
					if commit(false, commitInterruptNewHead, w.effectBlockNumber.Uint64()+1) {
						log.Info("received powAnswer, is my turn to produce new block", "addr", w.coinbase,
							"effectBlockNumber", w.effectBlockNumber, "visualBlockNumber", w.visualBlockNumber,
							"dposAgreeAckNum", dposAgreeAckNum, "dposOpposeAckNum", dposOpposeAckNum, "answerNumber", answerNumber)
					}
				} else if !timerDelaySealSetFlag && ret && (dposAgreeAckNum >= int(LeastDposWitness) || dposOpposeAckNum >= int(LeastDposWitness)) {
					timerDelaySealSetFlag = true
					timerDelaySeal.Reset(consensus.Time2delaySeal * time.Second)
					w.delaySealBlockNumber.Add(w.effectBlockNumber, common.Big1)
					log.Debug("received powAnswer, but not enough dposAck, wait...", "addr", w.coinbase, "waitTime", consensus.Time2delaySeal,
						"effectBlockNumber", w.effectBlockNumber, "visualBlockNumber", w.visualBlockNumber,
						"dposAgreeAckNum", dposAgreeAckNum, "dposOpposeAckNum", dposOpposeAckNum, "answerNumber", answerNumber, "rejectBlockNumber:", rejectBlockNumber)
				} else {
					log.Trace("i'm producer, but nothing to do", "timerDelaySealSetFlag", timerDelaySealSetFlag, "answer check", ret,
						"dposAgreeAckNum", dposAgreeAckNum, "dposOpposeAckNum", dposOpposeAckNum)
				}
			} else if w.imDposWorkNode(w.visualBlockNumber) {
				//not the producer
				ret, answerNumber := w.checkPowAnswerNumber(w.visualBlockNumber, w.effectBlockNumber)
				if !timerSealDealineSetFlag && ret &&
					(dposAgreeAckNum >= int(LeastDposWitness) || dposOpposeAckNum >= int(LeastDposWitness)) {
					//todo: consider if i should wait MostDposWitness to start a timer?
					timerSealDealineSetFlag = true
					rejectBlockNumber.Add(w.visualBlockNumber, common.Big1)
					timerSealDealine.Reset(consensus.Time2SealDeadline * time.Second)
					log.Debug("received powAnswer, not my turn to produce, setup a timer to wait new block", "addr", w.coinbase,
						"effectBlockNumber", w.effectBlockNumber, "visualBlockNumber", w.visualBlockNumber,
						"dposAgreeAckNum", dposAgreeAckNum, "dposOpposeAckNum", dposOpposeAckNum, "answerNumber", answerNumber, "rejectBlockNumber:", rejectBlockNumber)
				} else {
					log.Trace("i'm dpos node, but nothing to do", "timerSealDealineSetFlag", timerSealDealineSetFlag, "answer check", ret,
						"dposAgreeAckNum", dposAgreeAckNum, "dposOpposeAckNum", dposOpposeAckNum)
				}
			}

		case <-timerDelaySeal.C:
			baseBlockNumber := new(big.Int).Sub(w.delaySealBlockNumber, common.Big1)
			dposAgreeAckNum := w.chain.GetDposAckSize(baseBlockNumber, types.AckTypeAgree)
			dposOpposeAckNum := w.chain.GetDposAckSize(baseBlockNumber, types.AckTypeOppose)
			if dposAgreeAckNum >= int(LeastDposWitness) || dposOpposeAckNum >= int(LeastDposWitness) {
				//received 1/3 witness node ack
				if commit(false, commitInterruptNewHead, w.delaySealBlockNumber.Uint64()) {
					log.Info("timerDelaySeal is expired, we have LeastDposWitness to produce new block",
						"addr", w.coinbase, "dposAgreeAckNum", dposAgreeAckNum, "dposOpposeAckNum", dposOpposeAckNum,
						"current block number", baseBlockNumber)
				}
			} else {
				//todo: warning, consider another 3 seconds timer to seal
				log.Info("timerDelaySeal is expired, we DON'T have LeastDposWitness to produce new block",
					"addr", w.coinbase, "dposAgreeAckNum", dposAgreeAckNum, "dposOpposeAckNum", dposOpposeAckNum,
					"current block number", baseBlockNumber)
			}
			timerDelaySealSetFlag = false

		case <-timerSealDealine.C:
			if rejectBlockNumber.Uint64() == (w.visualBlockNumber.Uint64() + 1) {
				if nil == w.sendAck(rejectBlockNumber.Uint64(), types.AckTypeOppose) {
					log.Info("timerSealDealine expired, send reject ack to all dpos node success", "addr", w.coinbase,
						"rejectBlockNumber", rejectBlockNumber)
				} else {
					log.Info("timerSealDealine expired, send reject ack to all dpos node failed", "addr", w.coinbase,
						"rejectBlockNumber", rejectBlockNumber)
				}
				w.visualBlockNumber.Set(rejectBlockNumber)
				log.Trace("block number update", "effectBlockNumber", w.effectBlockNumber, "currentBlock",
					w.chain.CurrentBlock().Number(), "visualBlockNumber", w.visualBlockNumber)
			} else {
				log.Info("timerSealDealine expired, but visualBlockNumber is change", "addr", w.coinbase,
					"rejectBlockNumber", rejectBlockNumber, "(w.visualBlockNumber.Uint64() + 1)", w.visualBlockNumber.Uint64()+1)
			}
			timerSealDealineSetFlag = false

		case interval := <-w.resubmitIntervalCh:
			// Adjust resubmit interval explicitly by user.
			if interval < minRecommitInterval {
				log.Warn("Sanitizing miner recommit interval", "provided", interval, "updated", minRecommitInterval)
				interval = minRecommitInterval
			}
			log.Info("Miner recommit interval update", "from", minRecommit, "to", interval)
			minRecommit, recommit = interval, interval

			if w.resubmitHook != nil {
				w.resubmitHook(minRecommit, recommit)
			}

		case adjust := <-w.resubmitAdjustCh:
			// Adjust resubmit interval by feedback.
			if adjust.inc {
				before := recommit
				target := float64(recommit.Nanoseconds()) / adjust.ratio
				recommit = recalcRecommit(minRecommit, recommit, target, true)
				log.Trace("Increase miner recommit interval", "from", before, "to", recommit)
			} else {
				before := recommit
				recommit = recalcRecommit(minRecommit, recommit, float64(minRecommit.Nanoseconds()), false)
				log.Trace("Decrease miner recommit interval", "from", before, "to", recommit)
			}

			if w.resubmitHook != nil {
				w.resubmitHook(minRecommit, recommit)
			}

		case <-w.exitCh:
			return
		}
	}
}

// mainLoop is a standalone goroutine to regenerate the sealing task based on the received event.
func (w *worker) mainLoop() {
	defer w.txsSub.Unsubscribe()
	defer w.chainHeadSub.Unsubscribe()
	defer w.chainSideSub.Unsubscribe()

	for {
		select {
		case req := <-w.newWorkCh:
			w.dposCommitNewWork(req.interrupt, req.noempty, req.currentEffectBlockNumber, req.newBlockNumber, req.newBlockType)
		case <-w.txsCh:
			continue
		case <-w.exitCh:
			return
		case <-w.txsSub.Err():
			return
		case <-w.chainHeadSub.Err():
			return
		case <-w.chainSideSub.Err():
			return
		case <-w.powAnswerSub.Err():
			return
		case <-w.dposAckSub.Err():
			return
		}
	}
}

// taskLoop is a standalone goroutine to write new sealed block to db
func (w *worker) taskLoop() {
	for {
		select {
		case task := <-w.taskCh:
			block := task.block
			hash := block.Hash()

			// Different block could share same sealhash, deep copy here to prevent write-write conflict.
			var (
				receipts = make([]*types.Receipt, len(task.receipts))
				logs     []*types.Log
			)
			for i, receipt := range task.receipts {
				// add block location fields
				receipt.BlockHash = hash
				receipt.BlockNumber = block.Number()
				receipt.TransactionIndex = uint(i)

				receipts[i] = new(types.Receipt)
				*receipts[i] = *receipt
				// Update the block hash in all logs since it is now available and not when the
				// receipt/log of individual transactions were created.
				for _, log := range receipt.Logs {
					log.BlockHash = hash
				}
				logs = append(logs, receipt.Logs...)
			}

			// Commit block and state to database.
			_, err := w.chain.WriteBlockWithState(block, receipts, logs, task.state, true)
			if err != nil {
				log.Error("Failed writing block to chain", "err", err)
				continue
			}
			log.Info("Successfully sealed new block", "number", block.Number(), "hash", hash)

			// Broadcast the block and announce chain insertion event
			w.mux.Post(core.NewMinedBlockEvent{Block: block})

			// Insert the block into the set of pending ones to resultLoop for confirmations
			w.unconfirmed.Insert(block.NumberU64(), block.Hash())

		case <-w.exitCh:
			return
		}
	}
}

func (w *worker) powMinerNewWorkLoop() {
	for {
		select {
		//case <-w.chainHeadCh:
		case <-w.exitCh:
			return
		}
	}
}

func (w *worker) powMinerResultLoop() {
	for {
		select {
		case powMinerResult := <-w.powMinerResultCh:
			log.Info("new powMinerResult", ":", powMinerResult)
			w.mux.Post(core.PowAnswerEvent{PowAnswer: powMinerResult})

		case <-w.exitCh:
			return
		}
	}
}

// makeCurrent creates a new environment for the current cycle.
func (w *worker) makeCurrent(parent *types.Block, header *types.Header) error {
	// Retrieve the parent state to execute on top and start a prefetcher for
	// the miner to speed block sealing up a bit
	state, err := w.chain.StateAt(parent.Root())
	if err != nil {
		return err
	}
	state.StartPrefetcher("miner")

	env := &environment{
		signer:    types.MakeSigner(w.chainConfig, header.Number),
		state:     state,
		ancestors: mapset.NewSet(),
		family:    mapset.NewSet(),
		uncles:    mapset.NewSet(),
		header:    header,
	}
	// when 08 is processed ancestors contain 07 (quick block)
	for _, ancestor := range w.chain.GetBlocksFromHash(parent.Hash(), 7) {
		for _, uncle := range ancestor.Uncles() {
			env.family.Add(uncle.Hash())
		}
		env.family.Add(ancestor.Hash())
		env.ancestors.Add(ancestor.Hash())
	}
	// Keep track of transactions which return errors so they can be removed
	env.tcount = 0

	// Swap out the old work with the new one, terminating any leftover prefetcher
	// processes in the mean time and starting a new one.
	if w.current != nil && w.current.state != nil {
		w.current.state.StopPrefetcher()
	}
	w.current = env
	return nil
}

// commitUncle adds the given block to uncle block set, returns error if failed to add.
func (w *worker) commitUncle(env *environment, uncle *types.Header) error {
	hash := uncle.Hash()
	if env.uncles.Contains(hash) {
		return errors.New("uncle not unique")
	}
	if env.header.ParentHash == uncle.ParentHash {
		return errors.New("uncle is sibling")
	}
	if !env.ancestors.Contains(uncle.ParentHash) {
		return errors.New("uncle's parent unknown")
	}
	if env.family.Contains(hash) {
		return errors.New("uncle already included")
	}
	env.uncles.Add(uncle.Hash())
	return nil
}

// updateSnapshot updates pending snapshot block and state.
// Note this function assumes the current variable is thread safe.
func (w *worker) updateSnapshot() {
	w.snapshotMu.Lock()
	defer w.snapshotMu.Unlock()

	var uncles []*types.Header
	w.current.uncles.Each(func(item interface{}) bool {
		hash, ok := item.(common.Hash)
		if !ok {
			return false
		}
		uncle, exist := w.localUncles[hash]
		if !exist {
			uncle, exist = w.remoteUncles[hash]
		}
		if !exist {
			return false
		}
		uncles = append(uncles, uncle.Header())
		return false
	})

	w.snapshotBlock = types.NewBlock(
		w.current.header,
		w.current.txs,
		uncles,
		w.current.receipts,
		trie.NewStackTrie(nil),
	)
	w.snapshotReceipts = w.current.receipts
	w.snapshotState = w.current.state.Copy()
}

func (w *worker) commitTransaction(tx *types.Transaction, coinbase common.Address) ([]*types.Log, error) {
	snap := w.current.state.Snapshot()

	receipt, err := core.ApplyTransaction(w.chainConfig, w.chain, &coinbase, w.current.gasPool, w.current.state, w.current.header, tx, &w.current.header.GasUsed, *w.chain.GetVMConfig())
	if err != nil {
		w.current.state.RevertToSnapshot(snap)
		return nil, err
	}
	w.current.txs = append(w.current.txs, tx)
	w.current.receipts = append(w.current.receipts, receipt)

	return receipt.Logs, nil
}

func (w *worker) commitTransactions(txs *types.TransactionsByPriceAndNonce, coinbase common.Address, interrupt *int32) bool {
	// Short circuit if current is nil
	if w.current == nil {
		return true
	}

	gasLimit := w.current.header.GasLimit
	if w.current.gasPool == nil {
		w.current.gasPool = new(core.GasPool).AddGas(gasLimit)
	}

	var coalescedLogs []*types.Log

	for {
		// In the following three cases, we will interrupt the execution of the transaction.
		// (1) new head block event arrival, the interrupt signal is 1
		// (2) worker start or restart, the interrupt signal is 1
		// (3) worker recreate the mining block with any newly arrived transactions, the interrupt signal is 2.
		// For the first two cases, the semi-finished work will be discarded.
		// For the third case, the semi-finished work will be submitted to the consensus engine.
		if interrupt != nil && atomic.LoadInt32(interrupt) != commitInterruptNone {
			// Notify resubmit loop to increase resubmitting interval due to too frequent commits.
			if atomic.LoadInt32(interrupt) == commitInterruptResubmit {
				ratio := float64(gasLimit-w.current.gasPool.Gas()) / float64(gasLimit)
				if ratio < 0.1 {
					ratio = 0.1
				}
				w.resubmitAdjustCh <- &intervalAdjust{
					ratio: ratio,
					inc:   true,
				}
			}
			return atomic.LoadInt32(interrupt) == commitInterruptNewHead
		}
		// If we don't have enough gas for any further transactions then we're done
		if w.current.gasPool.Gas() < params.TxGas {
			log.Trace("Not enough gas for further transactions", "have", w.current.gasPool, "want", params.TxGas)
			break
		}
		// Retrieve the next transaction and abort if all done
		tx := txs.Peek()
		if tx == nil {
			break
		}
		// Error may be ignored here. The error has already been checked
		// during transaction acceptance is the transaction pool.
		//
		// We use the eip155 signer regardless of the current hf.
		from, _ := types.Sender(w.current.signer, tx)
		// Check whprobeer the tx is replay protected. If we're not in the EIP155 hf
		// phase, start ignoring the sender until we do.
		if tx.Protected() && !w.chainConfig.IsEIP155(w.current.header.Number) {
			log.Trace("Ignoring reply protected transaction", "hash", tx.Hash(), "eip155", w.chainConfig.EIP155Block)

			txs.Pop()
			continue
		}
		// Start executing the transaction
		w.current.state.Prepare(tx.Hash(), w.current.tcount)

		logs, err := w.commitTransaction(tx, coinbase)
		switch {
		case errors.Is(err, core.ErrGasLimitReached):
			// Pop the current out-of-gas transaction without shifting in the next from the account
			log.Trace("Gas limit exceeded for current block", "sender", from)
			txs.Pop()

		case errors.Is(err, core.ErrNonceTooLow):
			// New head notification data race between the transaction pool and miner, shift
			log.Trace("Skipping transaction with low nonce", "sender", from, "nonce", tx.Nonce())
			txs.Shift()

		case errors.Is(err, core.ErrNonceTooHigh):
			// Reorg notification data race between the transaction pool and miner, skip account =
			log.Trace("Skipping account with hight nonce", "sender", from, "nonce", tx.Nonce())
			txs.Pop()

		case errors.Is(err, nil):
			// Everything ok, collect the logs and shift in the next transaction from the same account
			coalescedLogs = append(coalescedLogs, logs...)
			w.current.tcount++
			txs.Shift()

		case errors.Is(err, core.ErrTxTypeNotSupported):
			// Pop the unsupported transaction without shifting in the next from the account
			log.Trace("Skipping unsupported transaction type", "sender", from, "type", tx.Type())
			txs.Pop()

		default:
			// Strange error, discard the transaction and get the next in line (note, the
			// nonce-too-high clause will prevent us from executing in vain).
			log.Debug("Transaction failed, account skipped", "hash", tx.Hash(), "err", err)
			txs.Shift()
		}
	}

	if !w.isRunning() && len(coalescedLogs) > 0 {
		// We don't push the pendingLogsEvent while we are mining. The reason is that
		// when we are mining, the worker will regenerate a mining block every 3 seconds.
		// In order to avoid pushing the repeated pendingLog, we disable the pending log pushing.

		// make a copy, the state caches the logs and these logs get "upgraded" from pending to mined
		// logs by filling in the block hash when the block was mined by the local miner. This can
		// cause a race condition if a log was "upgraded" before the PendingLogsEvent is processed.
		cpy := make([]*types.Log, len(coalescedLogs))
		for i, l := range coalescedLogs {
			cpy[i] = new(types.Log)
			*cpy[i] = *l
		}
		w.pendingLogsFeed.Send(cpy)
	}
	// Notify resubmit loop to decrease resubmitting interval if current interval is larger
	// than the user-specified one.
	if interrupt != nil {
		w.resubmitAdjustCh <- &intervalAdjust{inc: false}
	}
	return false
}

// commitNewWork generates several new sealing tasks based on the parent block.
func (w *worker) commitNewWork(interrupt *int32, noempty bool, timestamp int64) {
}

func (w *worker) dposCommitNewWork(interrupt *int32, noempty bool, currentEffectBlockNumber *big.Int, newBlockNumber *big.Int,
	newBlockType types.BlockType) *types.Block {
	w.mu.RLock()
	defer w.mu.RUnlock()
	w.muProduce.Lock()
	defer w.muProduce.Unlock()
	if w.coinbase == (common.Address{}) {
		log.Error("Refusing to mine without coinbase")
		return nil
	}

	parentBlockNum := new(big.Int).SetUint64(newBlockNumber.Uint64() - 1)
	parent := w.chain.GetBlockByNumber(parentBlockNum.Uint64())

	realParent := w.chain.GetRealBlockByNumber(parentBlockNum.Uint64())

	timestamp := time.Now().Unix()
	if parent.Time() >= uint64(timestamp) {
		timestamp = int64(parent.Time() + 1)
	}

	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     newBlockNumber,
		GasLimit:   core.CalcGasLimit(realParent.GasUsed(), realParent.GasLimit(), w.config.GasFloor, w.config.GasCeil),
		Extra:      w.extra,
		Time:       uint64(timestamp),
	}
	if newBlockType == types.BlockTypeVisual {
		header.Extra = params.VisualBlockExtra.Bytes()
	}
	header.Nonce = types.BlockNonce{}
	header.MixDigest = common.Hash{}
	header.Difficulty = probehash2.CalcDifficulty(w.chainConfig, uint64(timestamp), realParent.Header())

	log.Info("", "calc Difficulty :  ", header.Difficulty)
	header.Coinbase = common.Address{}
	header.DposSigAddr = w.coinbase

	// Could potentially happen if starting to mine in an odd state.
	err := w.makeCurrent(parent, header)
	if err != nil {
		log.Error("Failed to create mining context", "err", err)
		return nil
	}

	if newBlockType == types.BlockTypeEffect {
		//Process txs
		// Fill the block with all available pending transactions.
		pending, err := w.probe.TxPool().Pending(true)
		if err != nil {
			log.Error("Failed to fetch pending transactions", "err", err)
			return nil
		}
		// Split the pending transactions into locals and remotes
		localTxs, remoteTxs := make(map[common.Address]types.Transactions), pending
		for _, account := range w.probe.TxPool().Locals() {
			if txs := remoteTxs[account]; len(txs) > 0 {
				delete(remoteTxs, account)
				localTxs[account] = txs
			}
		}
		if len(localTxs) > 0 {
			txs := types.NewTransactionsByPriceAndNonce(w.current.signer, localTxs, header.BaseFee)
			if w.commitTransactions(txs, w.coinbase, interrupt) {
				return nil
			}
		}
		if len(remoteTxs) > 0 {
			txs := types.NewTransactionsByPriceAndNonce(w.current.signer, remoteTxs, header.BaseFee)
			if w.commitTransactions(txs, w.coinbase, interrupt) {
				return nil
			}
		}

		w.current.powAnswerUncles = w.probe.BlockChain().GetUnclePowAnswers(realParent.Number())
	}

	//process powAnswers and dposAcks
	if newBlockNumber.Uint64()-currentEffectBlockNumber.Uint64() == 1 {
		w.current.dposAcks = w.probe.BlockChain().GetDposAck(parentBlockNum, types.AckTypeAgree)
	} else {
		w.current.dposAcks = w.probe.BlockChain().GetDposAck(parentBlockNum, types.AckTypeOppose)
	}
	if len(w.current.dposAcks) < int(LeastDposWitness) {
		log.Error("not enough dposAck in blockchain! something error", "parentBlockNum", parentBlockNum)
		return nil
	}
	ackCount := types.DposAckCount{
		parentBlockNum,
		uint(len(w.current.dposAcks)),
	}
	w.current.header.DposAckCountList = append(w.current.header.DposAckCountList, &ackCount)

	answers := w.probe.BlockChain().GetLatestPowAnswer(realParent.Number())
	if answers == nil {
		log.Error("Refusing to mine without PowAnswers, something error, need to check")
		return nil
	}
	w.current.header.PowAnswers = append(w.current.header.PowAnswers, answers)

	// Deep copy receipts here to avoid interaction between different tasks.
	receipts := copyReceipts(w.current.receipts)
	s := w.current.state.Copy()

	//finalize seal
	greatri, _ := w.engine.(*greatri2.Greatri)
	greatri.DposFinalize(w.chain, header, s, w.current.txs, w.current.powAnswerUncles)
	block := types.DposNewBlock(w.current.header, w.current.txs, w.current.powAnswerUncles, w.current.dposAcks, receipts,
		trie.NewStackTrie(nil), newBlockType)
	if err := w.engine.Seal(w.chain, block, nil, nil); err != nil {
		log.Warn("Block sealing failed", "err", err)
	}

	select {
	case w.taskCh <- &task{receipts: receipts, state: s, block: block, createdAt: time.Now()}:
		log.Info("", "extra： ", common.Bytes2Hex(block.Extra()))
		log.Debug("Commit new block", "number", block.Number(), "blockHash", block.Hash(), "blockType", newBlockType,
			"txs", w.current.tcount, "gas", block.GasUsed(), "fees", totalFees(block, receipts))
		return block

	case <-w.exitCh:
		log.Info("Worker has exited")
		return nil
	}
}

// copyReceipts makes a deep copy of the given receipts.
func copyReceipts(receipts []*types.Receipt) []*types.Receipt {
	result := make([]*types.Receipt, len(receipts))
	for i, l := range receipts {
		cpy := *l
		result[i] = &cpy
	}
	return result
}

// postSideBlock fires a side chain event, only use it for testing.
func (w *worker) postSideBlock(event core.ChainSideEvent) {
	select {
	case w.chainSideCh <- event:
	case <-w.exitCh:
	}
}

// totalFees computes total consumed miner fees in ETH. Block transactions and receipts have to have the same order.
func totalFees(block *types.Block, receipts []*types.Receipt) *big.Float {
	feesWei := new(big.Int)
	for i, tx := range block.Transactions() {
		minerFee, _ := tx.EffectiveGasTip(block.BaseFee())
		feesWei.Add(feesWei, new(big.Int).Mul(new(big.Int).SetUint64(receipts[i].GasUsed), minerFee))
	}
	return new(big.Float).Quo(new(big.Float).SetInt(feesWei), new(big.Float).SetInt(big.NewInt(params.Probeer)))
}

// SetMinDifficulty
func (w *worker) SetMinDifficulty(difficulty int64) {
	minDifficulty = difficulty
}

// GetMinDifficulty
func (w *worker) GetMinDifficulty() int64 {
	return minDifficulty
}

// calcDifficulty
func calcDifficulty(time uint64, parent *types.Header) *big.Int {
	bigTime := new(big.Int).SetUint64(time)
	bigParentTime := new(big.Int).SetUint64(parent.Time)

	// holds intermediate values to make the algo easier to read & audit
	x := new(big.Int)
	y := new(big.Int)

	// (2 if len(parent_uncles) else 1) - (block_timestamp - parent_timestamp) // 10
	x.Sub(bigTime, bigParentTime)
	x.Div(x, big.NewInt(10))
	if parent.UncleHash == types.EmptyUncleHash {
		x.Sub(big.NewInt(1), x)
	} else {
		x.Sub(big.NewInt(2), x)
	}
	// max((2 if len(parent_uncles) else 1) - (block_timestamp - parent_timestamp) // 10, -10)
	if x.Cmp(big.NewInt(-10)) < 0 {
		x.Set(big.NewInt(-10))
	}
	// parent_diff + (parent_diff / 1024 * max((2 if len(parent.uncles) else 1) - ((timestamp - parent.timestamp) ÷ 10), -10))
	y.Div(parent.Difficulty, big.NewInt(1024))
	x.Mul(y, x)
	x.Add(parent.Difficulty, x)

	// minimum difficulty can ever be (before exponential factor)
	if x.Cmp(big.NewInt(minDifficulty)) < 0 {
		x.Set(big.NewInt(minDifficulty))
	}
	return x
}

// updateDposParams
func updateDposParams(dposSize int) {
	DposWitnessNumber = uint(dposSize)
	MostDposWitness = DposWitnessNumber*2/3 + 1
	//LeastDposWitness = DposWitnessNumber*1/3 + 1
	LeastDposWitness = DposWitnessNumber*1/2 + 1
	dposAckChanSize = DposWitnessNumber * 10
}

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

package probe

import (
	"errors"
	"math"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/status-im/keycard-go/hexutils"

	"github.com/probeum/go-probeum/common"
	"github.com/probeum/go-probeum/core"
	"github.com/probeum/go-probeum/core/forkid"
	"github.com/probeum/go-probeum/core/types"
	"github.com/probeum/go-probeum/event"
	"github.com/probeum/go-probeum/log"
	"github.com/probeum/go-probeum/p2p"
	"github.com/probeum/go-probeum/params"
	"github.com/probeum/go-probeum/probe/downloader"
	"github.com/probeum/go-probeum/probe/fetcher"
	"github.com/probeum/go-probeum/probe/protocols/probe"
	"github.com/probeum/go-probeum/probe/protocols/snap"
	"github.com/probeum/go-probeum/probedb"
	"github.com/probeum/go-probeum/trie"
)

const (
	// txChanSize is the size of channel listening to NewTxsEvent.
	// The number is referenced from the size of tx pool.
	txChanSize = 4096
)

var (
	syncChallengeTimeout = 15 * time.Second // Time allowance for a node to reply to the sync progress challenge
)

// txPool defines the methods needed from a transaction pool implementation to
// support all the operations needed by the Probeum chain protocols.
type txPool interface {
	// Has returns an indicator whprobeer txpool has a transaction
	// cached with the given hash.
	Has(hash common.Hash) bool

	// Get retrieves the transaction from local txpool with given
	// tx hash.
	Get(hash common.Hash) *types.Transaction

	// AddRemotes should add the given transactions to the pool.
	AddRemotes([]*types.Transaction) []error

	// Pending should return pending transactions.
	// The slice should be modifiable by the caller.
	Pending(enforceTips bool) (map[common.Address]types.Transactions, error)

	// SubscribeNewTxsEvent should return an event subscription of
	// NewTxsEvent and send events to the given channel.
	SubscribeNewTxsEvent(chan<- core.NewTxsEvent) event.Subscription
}

// handlerConfig is the collection of initialization parameters to create a full
// node network handler.
type handlerConfig struct {
	Database   probedb.Database          // Database for direct sync insertions
	Chain      *core.BlockChain          // Blockchain to serve data from
	TxPool     txPool                    // Transaction pool to propagate from
	Network    uint64                    // Network identifier to adfvertise
	Sync       downloader.SyncMode       // Whprobeer to fast or full sync
	BloomCache uint64                    // Megabytes to alloc for fast sync bloom
	EventMux   *event.TypeMux            // Legacy event mux, deprecate for `feed`
	Checkpoint *params.TrustedCheckpoint // Hard coded checkpoint for sync challenges
	Whitelist  map[uint64]common.Hash    // Hard coded whitelist for sync challenged
}

type handler struct {
	networkID  uint64
	forkFilter forkid.Filter // Fork ID filter, constant across the lifetime of the node

	fastSync  uint32 // Flag whprobeer fast sync is enabled (gets disabled if we already have blocks)
	snapSync  uint32 // Flag whprobeer fast sync should operate on top of the snap protocol
	acceptTxs uint32 // Flag whprobeer we're considered synchronised (enables transaction processing)

	checkpointNumber uint64      // Block number for the sync progress validator to cross reference
	checkpointHash   common.Hash // Block hash for the sync progress validator to cross reference

	database probedb.Database
	txpool   txPool
	chain    *core.BlockChain
	maxPeers int

	downloader   *downloader.Downloader
	stateBloom   *trie.SyncBloom
	blockFetcher *fetcher.BlockFetcher
	txFetcher    *fetcher.TxFetcher
	peers        *peerSet

	eventMux      *event.TypeMux
	txsCh         chan core.NewTxsEvent
	txsSub        event.Subscription
	minedBlockSub *event.TypeMuxSubscription
	powAnswerSub  *event.TypeMuxSubscription
	dposAckSub    *event.TypeMuxSubscription

	whitelist map[uint64]common.Hash

	// channels for fetcher, syncer, txsyncLoop
	txsyncCh chan *txsync
	quitSync chan struct{}

	chainSync *chainSyncer
	wg        sync.WaitGroup
	peerWG    sync.WaitGroup
}

// newHandler returns a handler for all Probeum chain management protocol.
func newHandler(config *handlerConfig) (*handler, error) {
	// Create the protocol manager with the base fields
	if config.EventMux == nil {
		config.EventMux = new(event.TypeMux) // Nicety initialization for tests
	}
	h := &handler{
		networkID:  config.Network,
		forkFilter: forkid.NewFilter(config.Chain),
		eventMux:   config.EventMux,
		database:   config.Database,
		txpool:     config.TxPool,
		chain:      config.Chain,
		peers:      newPeerSet(),
		whitelist:  config.Whitelist,
		txsyncCh:   make(chan *txsync),
		quitSync:   make(chan struct{}),
	}
	if config.Sync == downloader.FullSync {
		// The database seems empty as the current block is the genesis. Yet the fast
		// block is ahead, so fast sync was enabled for this node at a certain point.
		// The scenarios where this can happen is
		// * if the user manually (or via a bad block) rolled back a fast sync node
		//   below the sync point.
		// * the last fast sync is not finished while user specifies a full sync this
		//   time. But we don't have any recent state for full sync.
		// In these cases however it's safe to reenable fast sync.
		fullBlock, fastBlock := h.chain.CurrentBlock(), h.chain.CurrentFastBlock()
		if fullBlock.NumberU64() == 0 && fastBlock.NumberU64() > 0 {
			h.fastSync = uint32(1)
			log.Warn("Switch sync mode from full sync to fast sync")
		}
	} else {
		if h.chain.CurrentBlock().NumberU64() > 0 {
			// Print warning log if database is not empty to run fast sync.
			log.Warn("Switch sync mode from fast sync to full sync")
		} else {
			// If fast sync was requested and our database is empty, grant it
			h.fastSync = uint32(1)
			if config.Sync == downloader.SnapSync {
				h.snapSync = uint32(1)
			}
		}
	}
	// If we have trusted checkpoints, enforce them on the chain
	if config.Checkpoint != nil {
		h.checkpointNumber = (config.Checkpoint.SectionIndex+1)*params.CHTFrequency - 1
		h.checkpointHash = config.Checkpoint.SectionHead
	}
	// Construct the downloader (long sync) and its backing state bloom if fast
	// sync is requested. The downloader is responsible for deallocating the state
	// bloom when it's done.
	// Note: we don't enable it if snap-sync is performed, since it's very heavy
	// and the heal-portion of the snap sync is much lighter than fast. What we particularly
	// want to avoid, is a 90%-finished (but restarted) snap-sync to begin
	// indexing the entire trie
	if atomic.LoadUint32(&h.fastSync) == 1 && atomic.LoadUint32(&h.snapSync) == 0 {
		h.stateBloom = trie.NewSyncBloom(config.BloomCache, config.Database)
	}
	h.downloader = downloader.New(h.checkpointNumber, config.Database, h.stateBloom, h.eventMux, h.chain, nil, h.removePeer)

	// Construct the fetcher (short sync)
	validator := func(header *types.Header) error {
		return h.chain.Engine().VerifyHeader(h.chain, header, true)
	}
	heighter := func() uint64 {
		return h.chain.CurrentBlock().NumberU64()
	}
	inserter := func(blocks types.Blocks) (int, error) {
		// If sync hasn't reached the checkpoint yet, deny importing weird blocks.
		//
		// Ideally we would also compare the head block's timestamp and similarly reject
		// the propagated block if the head is too old. Unfortunately there is a corner
		// case when starting new networks, where the genesis might be ancient (0 unix)
		// which would prevent full nodes from accepting it.
		if h.chain.CurrentBlock().NumberU64() < h.checkpointNumber {
			log.Warn("Unsynced yet, discarded propagated block", "number", blocks[0].Number(), "hash", blocks[0].Hash())
			return 0, nil
		}
		// If fast sync is running, deny importing weird blocks. This is a problematic
		// clause when starting up a new network, because fast-syncing miners might not
		// accept each others' blocks until a restart. Unfortunately we haven't figured
		// out a way yet where nodes can decide unilaterally whprobeer the network is new
		// or not. This should be fixed if we figure out a solution.
		if atomic.LoadUint32(&h.fastSync) == 1 {
			log.Warn("Fast syncing, discarded propagated block", "number", blocks[0].Number(), "hash", blocks[0].Hash())
			return 0, nil
		}
		n, err := h.chain.InsertChain(blocks)
		if err == nil {
			atomic.StoreUint32(&h.acceptTxs, 1) // Mark initial sync done on any fetcher import
		}
		return n, err
	}
	h.blockFetcher = fetcher.NewBlockFetcher(false, nil, h.chain.GetBlockByHash, validator, h.BroadcastBlock, heighter, nil, inserter, h.removePeer)

	fetchTx := func(peer string, hashes []common.Hash) error {
		p := h.peers.peer(peer)
		if p == nil {
			return errors.New("unknown peer")
		}
		return p.RequestTxs(hashes)
	}
	h.txFetcher = fetcher.NewTxFetcher(h.txpool.Has, h.txpool.AddRemotes, fetchTx)
	h.chainSync = newChainSyncer(h)
	return h, nil
}

// runProbePeer registers an probe peer into the joint probe/snap peerset, adds it to
// various subsistems and starts handling messages.
func (h *handler) runProbePeer(peer *probe.Peer, handler probe.Handler) error {
	// If the peer has a `snap` extension, wait for it to connect so we can have
	// a uniform initialization/teardown mechanism
	snap, err := h.peers.waitSnapExtension(peer)
	if err != nil {
		peer.Log().Error("Snapshot extension barrier failed", "err", err)
		return err
	}
	// TODO(karalabe): Not sure why this is needed
	if !h.chainSync.handlePeerEvent(peer) {
		return p2p.DiscQuitting
	}
	h.peerWG.Add(1)
	defer h.peerWG.Done()

	// Execute the Probeum handshake
	var (
		genesis = h.chain.Genesis()
		head    = h.chain.CurrentHeader()
		hash    = head.Hash()
		number  = head.Number.Uint64()
		td      = h.chain.GetTd(hash, number)
	)
	forkID := forkid.NewID(h.chain.Config(), h.chain.Genesis().Hash(), h.chain.CurrentHeader().Number.Uint64())
	if err := peer.Handshake(h.networkID, td, hash, genesis.Hash(), forkID, h.forkFilter); err != nil {
		peer.Log().Debug("Probeum handshake failed", "err", err)
		return err
	}
	reject := false // reserved peer slots
	if atomic.LoadUint32(&h.snapSync) == 1 {
		if snap == nil {
			// If we are running snap-sync, we want to reserve roughly half the peer
			// slots for peers supporting the snap protocol.
			// The logic here is; we only allow up to 5 more non-snap peers than snap-peers.
			if all, snp := h.peers.len(), h.peers.snapLen(); all-snp > snp+5 {
				reject = true
			}
		}
	}
	// Ignore maxPeers if this is a trusted peer
	if !peer.Peer.Info().Network.Trusted {
		if reject || h.peers.len() >= h.maxPeers {
			return p2p.DiscTooManyPeers
		}
	}
	peer.Log().Debug("Probeum peer connected", "name", peer.Name())

	// Register the peer locally
	if err := h.peers.registerPeer(peer, snap); err != nil {
		peer.Log().Error("Probeum peer registration failed", "err", err)
		return err
	}
	defer h.unregisterPeer(peer.ID())

	p := h.peers.peer(peer.ID())
	if p == nil {
		return errors.New("peer dropped during handling")
	}
	// Register the peer in the downloader. If the downloader considers it banned, we disconnect
	if err := h.downloader.RegisterPeer(peer.ID(), peer.Version(), peer); err != nil {
		peer.Log().Error("Failed to register peer in probe syncer", "err", err)
		return err
	}
	if snap != nil {
		if err := h.downloader.SnapSyncer.Register(snap); err != nil {
			peer.Log().Error("Failed to register peer in snap syncer", "err", err)
			return err
		}
	}
	h.chainSync.handlePeerEvent(peer)

	// Propagate existing transactions. new transactions appearing
	// after this will be sent via broadcasts.
	h.syncTransactions(peer)

	// If we have a trusted CHT, reject all peers below that (avoid fast sync eclipse)
	if h.checkpointHash != (common.Hash{}) {
		// Request the peer's checkpoint header for chain height/weight validation
		if err := peer.RequestHeadersByNumber(h.checkpointNumber, 1, 0, false); err != nil {
			return err
		}
		// Start a timer to disconnect if the peer doesn't reply in time
		p.syncDrop = time.AfterFunc(syncChallengeTimeout, func() {
			peer.Log().Warn("Checkpoint challenge timed out, dropping", "addr", peer.RemoteAddr(), "type", peer.Name())
			h.removePeer(peer.ID())
		})
		// Make sure it's cleaned up if the peer dies off
		defer func() {
			if p.syncDrop != nil {
				p.syncDrop.Stop()
				p.syncDrop = nil
			}
		}()
	}
	// If we have any explicit whitelist block hashes, request them
	for number := range h.whitelist {
		if err := peer.RequestHeadersByNumber(number, 1, 0, false); err != nil {
			return err
		}
	}
	// Handle incoming messages until the connection is torn down
	return handler(peer)
}

// runSnapExtension registers a `snap` peer into the joint probe/snap peerset and
// starts handling inbound messages. As `snap` is only a satellite protocol to
// `probe`, all subsystem registrations and lifecycle management will be done by
// the main `probe` handler to prevent strange races.
func (h *handler) runSnapExtension(peer *snap.Peer, handler snap.Handler) error {
	h.peerWG.Add(1)
	defer h.peerWG.Done()

	if err := h.peers.registerSnapExtension(peer); err != nil {
		peer.Log().Error("Snapshot extension registration failed", "err", err)
		return err
	}
	return handler(peer)
}

// removePeer requests disconnection of a peer.
func (h *handler) removePeer(id string) {
	peer := h.peers.peer(id)
	if peer != nil {
		peer.Peer.Disconnect(p2p.DiscUselessPeer)
	}
}

// unregisterPeer removes a peer from the downloader, fetchers and main peer set.
func (h *handler) unregisterPeer(id string) {
	// Create a custom logger to avoid printing the entire id
	var logger log.Logger
	if len(id) < 16 {
		// Tests use short IDs, don't choke on them
		logger = log.New("peer", id)
	} else {
		logger = log.New("peer", id[:8])
	}
	// Abort if the peer does not exist
	peer := h.peers.peer(id)
	if peer == nil {
		logger.Error("Probeum peer removal failed", "err", errPeerNotRegistered)
		return
	}
	// Remove the `probe` peer if it exists
	logger.Debug("Removing Probeum peer", "snap", peer.snapExt != nil)

	// Remove the `snap` extension if it exists
	if peer.snapExt != nil {
		h.downloader.SnapSyncer.Unregister(id)
	}
	h.downloader.UnregisterPeer(id)
	h.txFetcher.Drop(id)

	if err := h.peers.unregisterPeer(id); err != nil {
		logger.Error("Probeum peer removal failed", "err", err)
	}
}

func (h *handler) Start(maxPeers int) {
	h.maxPeers = maxPeers

	// broadcast transactions
	h.wg.Add(1)
	h.txsCh = make(chan core.NewTxsEvent, txChanSize)
	h.txsSub = h.txpool.SubscribeNewTxsEvent(h.txsCh)
	go h.txBroadcastLoop()

	// broadcast mined blocks
	h.wg.Add(1)
	h.minedBlockSub = h.eventMux.Subscribe(core.NewMinedBlockEvent{})
	go h.minedBroadcastLoop()

	// broadcast powAnswer
	h.wg.Add(1)
	h.powAnswerSub = h.eventMux.Subscribe(core.PowAnswerEvent{})
	go h.powAnswerBroadcastLoop()

	// broadcast dposAck
	h.wg.Add(1)
	h.dposAckSub = h.eventMux.Subscribe(core.DposAckEvent{})
	go h.dposAckBroadcastLoop()

	// start sync handlers
	h.wg.Add(2)
	go h.chainSync.loop()
	go h.txsyncLoop64() // TODO(karalabe): Legacy initial tx echange, drop with probe/64.
}

func (h *handler) Stop() {
	h.txsSub.Unsubscribe()        // quits txBroadcastLoop
	h.minedBlockSub.Unsubscribe() // quits blockBroadcastLoop
	h.powAnswerSub.Unsubscribe()  // quits powAnswerBroadcastLoop
	h.dposAckSub.Unsubscribe()    // quits dposAckBroadcastLoop

	// Quit chainSync and txsync64.
	// After this is done, no new peers will be accepted.
	close(h.quitSync)
	h.wg.Wait()

	// Disconnect existing sessions.
	// This also closes the gate for any new registrations on the peer set.
	// sessions which are already established but not added to h.peers yet
	// will exit when they try to register.
	h.peers.close()
	h.peerWG.Wait()

	log.Info("Probeum protocol stopped")
}

// BroadcastBlock will either propagate a block to a subset of its peers, or
// will only announce its availability (depending what's requested).
func (h *handler) BroadcastBlock(block *types.Block, propagate bool) {
	hash := block.Hash()
	peers := h.peers.peersWithoutBlock(hash)

	// If propagation is requested, send to a subset of the peer
	if propagate {
		// Calculate the TD of the block (it's not imported yet, so block.Td is not valid)
		var td *big.Int
		if parent := h.chain.GetBlock(block.ParentHash(), block.NumberU64()-1); parent != nil {
			td = new(big.Int).Add(block.Difficulty(), h.chain.GetTd(block.ParentHash(), block.NumberU64()-1))
		} else {
			log.Error("Propagating dangling block", "number", block.Number(), "hash", hash)
			return
		}
		// Send the block to a subset of our peers
		transfer := peers[:int(math.Sqrt(float64(len(peers))))]
		//transfer := peers
		for _, peer := range transfer {
			peer.AsyncSendNewBlock(block, td)
		}
		log.Trace("Propagated block", "hash", hash, "recipients", len(transfer), "duration", common.PrettyDuration(time.Since(block.ReceivedAt)))
		return
	}
	// Otherwise if the block is indeed in out own chain, announce it
	if h.chain.HasBlock(hash, block.NumberU64()) {
		for _, peer := range peers {
			peer.AsyncSendNewBlockHash(block)
		}
		log.Trace("Announced block", "hash", hash, "recipients", len(peers), "duration", common.PrettyDuration(time.Since(block.ReceivedAt)))
	}
}

// BroadcastTransactions will propagate a batch of transactions
// - To a square root of all peers
// - And, separately, as announcements to all peers which are not known to
// already have the given transaction.
func (h *handler) BroadcastTransactions(txs types.Transactions) {
	var (
		annoCount   int // Count of announcements made
		annoPeers   int
		directCount int // Count of the txs sent directly to peers
		directPeers int // Count of the peers that were sent transactions directly

		txset = make(map[*probePeer][]common.Hash) // Set peer->hash to transfer directly
		annos = make(map[*probePeer][]common.Hash) // Set peer->hash to announce

	)
	// Broadcast transactions to a batch of peers not knowing about it
	for _, tx := range txs {
		peers := h.peers.peersWithoutTransaction(tx.Hash())
		// Send the tx unconditionally to a subset of our peers
		numDirect := int(math.Sqrt(float64(len(peers))))
		for _, peer := range peers[:numDirect] {
			txset[peer] = append(txset[peer], tx.Hash())
		}
		// For the remaining peers, send announcement only
		for _, peer := range peers[numDirect:] {
			annos[peer] = append(annos[peer], tx.Hash())
		}
	}
	for peer, hashes := range txset {
		directPeers++
		directCount += len(hashes)
		peer.AsyncSendTransactions(hashes)
	}
	for peer, hashes := range annos {
		annoPeers++
		annoCount += len(hashes)
		peer.AsyncSendPooledTransactionHashes(hashes)
	}
	log.Debug("Transaction broadcast", "txs", len(txs),
		"announce packs", annoPeers, "announced hashes", annoCount,
		"tx packs", directPeers, "broadcast txs", directCount)
}

// BroadcastPowAnswer broadcast PowAnswer to all peers
func (h *handler) BroadcastPowAnswer(powAnswer *types.PowAnswer) {
	if h.chain.CheckPowAnswerSketchy(powAnswer) {
		h.chain.HandlePowAnswer(powAnswer)
		peers := h.peers.peersWithoutPowAnswers(powAnswer)
		filter := peers[:int(math.Sqrt(float64(len(peers))))]
		for _, peer := range filter {
			peer.MarkPowAnswer(powAnswer.Id())
			peer.AsyncSendPowAnswer(powAnswer)
		}
		log.Debug("PowAnswer broadcast", "number", powAnswer.Number, "nonce", powAnswer.Nonce.Uint64(), "miner", powAnswer.Miner)
	} else {
		log.Debug("PowAnswer broadcast fail, because the pow answer is too old", "number", powAnswer.Number, "nonce", powAnswer.Nonce.Uint64(), "miner", powAnswer.Miner, "Chain Number", h.chain.CurrentBlock().NumberU64())
	}
}

// BroadcastDposAck broadcast dpos ack to all peers
func (h *handler) BroadcastDposAck(dposAck *types.DposAck) {
	check := h.chain.CheckDposAckSketchy(dposAck)
	if check {
		h.chain.HandleDposAck(dposAck)
		peers := h.peers.peersWithoutDposAcks(dposAck)
		filter := peers[:int(math.Sqrt(float64(len(peers))))]
		for _, peer := range filter {
			//log.Debug("BroadcastDposAck", "ack", common.BytesToHash(dposAck.WitnessSig))
			peer.MarkDposAck(dposAck.Id())
			peer.AsyncSendDposAck(dposAck)
		}
		log.Debug("DposAck broadcast", "number", dposAck.Number, "witnessSig", hexutils.BytesToHex(dposAck.WitnessSig), "BlockHash", dposAck.BlockHash, "Type", dposAck.AckType)
	} else {
		log.Debug("DposAck broadcast fail, because the dpos ack is illegality", "check", check, "number", dposAck.Number, "witnessSig", hexutils.BytesToHex(dposAck.WitnessSig), "BlockHash", dposAck.BlockHash, "Type", dposAck.AckType)
	}
}

// minedBroadcastLoop sends mined blocks to connected peers.
func (h *handler) minedBroadcastLoop() {
	defer h.wg.Done()

	for obj := range h.minedBlockSub.Chan() {
		if ev, ok := obj.Data.(core.NewMinedBlockEvent); ok {
			h.BroadcastBlock(ev.Block, true) // First propagate block to peers
			//h.BroadcastBlock(ev.Block, false) // Only then announce to the rest
		}
	}
}

// txBroadcastLoop announces new transactions to connected peers.
func (h *handler) txBroadcastLoop() {
	defer h.wg.Done()
	for {
		select {
		case event := <-h.txsCh:
			h.BroadcastTransactions(event.Txs)
		case <-h.txsSub.Err():
			return
		}
	}
}

// powAnswerBroadcastLoop sends pow answer to connected peers.
func (h *handler) powAnswerBroadcastLoop() {
	defer h.wg.Done()

	for obj := range h.powAnswerSub.Chan() {
		if ev, ok := obj.Data.(core.PowAnswerEvent); ok {
			h.BroadcastPowAnswer(ev.PowAnswer)
		}
	}
}

// dposAckBroadcastLoop sends dpos ack to connected peers.
func (h *handler) dposAckBroadcastLoop() {
	defer h.wg.Done()

	for obj := range h.dposAckSub.Chan() {
		if ev, ok := obj.Data.(core.DposAckEvent); ok {
			//log.Debug("dposAckBroadcastLoop", "ack", common.BytesToHash(ev.DposAck.WitnessSig))
			h.BroadcastDposAck(ev.DposAck)
		}
	}
}

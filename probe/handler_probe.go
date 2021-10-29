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
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/status-im/keycard-go/hexutils"

	"github.com/probeum/go-probeum/common"
	"github.com/probeum/go-probeum/core"
	"github.com/probeum/go-probeum/core/types"
	"github.com/probeum/go-probeum/log"
	"github.com/probeum/go-probeum/p2p/enode"
	"github.com/probeum/go-probeum/probe/protocols/probe"
	"github.com/probeum/go-probeum/trie"
)

// probeHandler implements the probe.Backend interface to handle the various network
// packets that are sent as replies or broadcasts.
type probeHandler handler

func (h *probeHandler) Chain() *core.BlockChain     { return h.chain }
func (h *probeHandler) StateBloom() *trie.SyncBloom { return h.stateBloom }
func (h *probeHandler) TxPool() probe.TxPool        { return h.txpool }

// RunPeer is invoked when a peer joins on the `probe` protocol.
func (h *probeHandler) RunPeer(peer *probe.Peer, hand probe.Handler) error {
	return (*handler)(h).runProbePeer(peer, hand)
}

// PeerInfo retrieves all known `probe` information about a peer.
func (h *probeHandler) PeerInfo(id enode.ID) interface{} {
	if p := h.peers.peer(id.String()); p != nil {
		return p.info()
	}
	return nil
}

// AcceptTxs retrieves whprobeer transaction processing is enabled on the node
// or if inbound transactions should simply be dropped.
func (h *probeHandler) AcceptTxs() bool {
	return atomic.LoadUint32(&h.acceptTxs) == 1
}

// Handle is invoked from a peer's message handler when it receives a new remote
// message that the handler couldn't consume and serve itself.
func (h *probeHandler) Handle(peer *probe.Peer, packet probe.Packet) error {
	// Consume any broadcasts and announces, forwarding the rest to the downloader
	switch packet := packet.(type) {
	case *probe.BlockHeadersPacket:
		return h.handleHeaders(peer, *packet)

	case *probe.BlockBodiesPacket:
		txset, uncleset, powAnswerUncles, dposAcks := packet.Unpack()
		return h.handleBodies(peer, txset, uncleset, powAnswerUncles, dposAcks)

	case *probe.NodeDataPacket:
		if err := h.downloader.DeliverNodeData(peer.ID(), *packet); err != nil {
			log.Debug("Failed to deliver node state data", "err", err)
		}
		return nil

	case *probe.ReceiptsPacket:
		if err := h.downloader.DeliverReceipts(peer.ID(), *packet); err != nil {
			log.Debug("Failed to deliver receipts", "err", err)
		}
		return nil

	case *probe.NewBlockHashesPacket:
		hashes, numbers := packet.Unpack()
		return h.handleBlockAnnounces(peer, hashes, numbers)

	case *probe.NewBlockPacket:
		return h.handleBlockBroadcast(peer, packet.Block, packet.TD)

	case *probe.NewPowAnswerPacket:
		return h.handlePowAnswerBroadcast(peer, packet.PowAnswer)

	case *probe.NewDposAckPacket:
		return h.handleDposAckBroadcast(peer, packet.DposAck)

	case *probe.NewPooledTransactionHashesPacket:
		return h.txFetcher.Notify(peer.ID(), *packet)

	case *probe.TransactionsPacket:
		return h.txFetcher.Enqueue(peer.ID(), *packet, false)

	case *probe.PooledTransactionsPacket:
		return h.txFetcher.Enqueue(peer.ID(), *packet, true)
	default:
		return fmt.Errorf("unexpected probe packet type: %T", packet)
	}
}

// handleHeaders is invoked from a peer's message handler when it transmits a batch
// of headers for the local node to process.
func (h *probeHandler) handleHeaders(peer *probe.Peer, headers []*types.Header) error {
	p := h.peers.peer(peer.ID())
	if p == nil {
		return errors.New("unregistered during callback")
	}
	// If no headers were received, but we're expencting a checkpoint header, consider it that
	if len(headers) == 0 && p.syncDrop != nil {
		// Stop the timer either way, decide later to drop or not
		p.syncDrop.Stop()
		p.syncDrop = nil

		// If we're doing a fast (or snap) sync, we must enforce the checkpoint block to avoid
		// eclipse attacks. Unsynced nodes are welcome to connect after we're done
		// joining the network
		if atomic.LoadUint32(&h.fastSync) == 1 {
			peer.Log().Warn("Dropping unsynced node during sync", "addr", peer.RemoteAddr(), "type", peer.Name())
			return errors.New("unsynced node cannot serve sync")
		}
	}
	// Filter out any explicitly requested headers, deliver the rest to the downloader
	filter := len(headers) == 1
	if filter {
		// If it's a potential sync progress check, validate the content and advertised chain weight
		if p.syncDrop != nil && headers[0].Number.Uint64() == h.checkpointNumber {
			// Disable the sync drop timer
			p.syncDrop.Stop()
			p.syncDrop = nil

			// Validate the header and either drop the peer or continue
			if headers[0].Hash() != h.checkpointHash {
				return errors.New("checkpoint hash mismatch")
			}
			return nil
		}
		// Otherwise if it's a whitelisted block, validate against the set
		if want, ok := h.whitelist[headers[0].Number.Uint64()]; ok {
			if hash := headers[0].Hash(); want != hash {
				peer.Log().Info("Whitelist mismatch, dropping peer", "number", headers[0].Number.Uint64(), "hash", hash, "want", want)
				return errors.New("whitelist block mismatch")
			}
			peer.Log().Debug("Whitelist block verified", "number", headers[0].Number.Uint64(), "hash", want)
		}
		// Irrelevant of the fork checks, send the header to the fetcher just in case
		headers = h.blockFetcher.FilterHeaders(peer.ID(), headers, time.Now())
	}
	if len(headers) > 0 || !filter {
		err := h.downloader.DeliverHeaders(peer.ID(), headers)
		if err != nil {
			log.Debug("Failed to deliver headers", "err", err)
		}
	}
	return nil
}

// handleBodies is invoked from a peer's message handler when it transmits a batch
// of block bodies for the local node to process.
func (h *probeHandler) handleBodies(peer *probe.Peer, txs [][]*types.Transaction, uncles [][]*types.Header,
	powAnswerUncles [][]*types.PowAnswer, dposAcks [][]*types.DposAck) error {
	// Filter out any explicitly requested bodies, deliver the rest to the downloader
	filter := len(txs) > 0 || len(uncles) > 0
	if filter {
		txs, uncles, powAnswerUncles, dposAcks = h.blockFetcher.FilterBodies(peer.ID(), txs, uncles, powAnswerUncles, dposAcks, time.Now())
	}
	if len(txs) > 0 || len(uncles) > 0 || len(powAnswerUncles) > 0 || len(dposAcks) > 0 || !filter {
		err := h.downloader.DeliverBodies(peer.ID(), txs, uncles, powAnswerUncles, dposAcks)
		if err != nil {
			log.Debug("Failed to deliver bodies", "err", err)
		}
	}
	return nil
}

// handleBlockAnnounces is invoked from a peer's message handler when it transmits a
// batch of block announcements for the local node to process.
func (h *probeHandler) handleBlockAnnounces(peer *probe.Peer, hashes []common.Hash, numbers []uint64) error {
	// Schedule all the unknown hashes for retrieval
	var (
		unknownHashes  = make([]common.Hash, 0, len(hashes))
		unknownNumbers = make([]uint64, 0, len(numbers))
	)
	for i := 0; i < len(hashes); i++ {
		if !h.chain.HasBlock(hashes[i], numbers[i]) {
			unknownHashes = append(unknownHashes, hashes[i])
			unknownNumbers = append(unknownNumbers, numbers[i])
		}
	}
	for i := 0; i < len(unknownHashes); i++ {
		h.blockFetcher.Notify(peer.ID(), unknownHashes[i], unknownNumbers[i], time.Now(), peer.RequestOneHeader, peer.RequestBodies)
	}
	return nil
}

// handleBlockBroadcast is invoked from a peer's message handler when it transmits a
// block broadcast for the local node to process.
func (h *probeHandler) handleBlockBroadcast(peer *probe.Peer, block *types.Block, td *big.Int) error {
	// Schedule the block for import
	h.blockFetcher.Enqueue(peer.ID(), block)

	// Assuming the block is importable by the peer, but possibly not yet done so,
	// calculate the head hash and TD that the peer truly must have.
	var (
		trueHead = block.ParentHash()
		trueTD   = new(big.Int).Sub(td, block.Difficulty())
	)
	// Update the peer's total difficulty if better than the previous
	if _, td := peer.Head(); trueTD.Cmp(td) > 0 {
		peer.SetHead(trueHead, trueTD)
		h.chainSync.handlePeerEvent(peer)
	}
	return nil
}

// handlePowAnswerBroadcast is invoked from a peer's message handler when it transmits a
// pow answer broadcast for the local node to process.
func (h *probeHandler) handlePowAnswerBroadcast(peer *probe.Peer, powAnswer *types.PowAnswer) error {
	// boardcast pow answer again
	if h.chain.CheckPowAnswerSketchy(powAnswer) {
		peer.KnownPowAnswer(powAnswer.Id())
		for _, peer := range h.peers.peersWithoutPowAnswers(powAnswer) {
			if err := peer.SendNewPowAnswer(powAnswer); err != nil {
				log.Debug("SendNewPowAnswer", "err", err)
			}
		}
		h.chain.HandlePowAnswer(powAnswer)
	} else {
		log.Debug("CheckPowAnswer Fail", "powAnswer.Number", powAnswer.Number.Uint64(), "Chain Number", h.chain.CurrentBlock().NumberU64())
	}
	return nil
}

// handleDposAckBroadcast is invoked from a peer's message handler when it transmits a
// dpos ack for the local node to process.
func (h *probeHandler) handleDposAckBroadcast(peer *probe.Peer, dposAck *types.DposAck) error {
	check := h.chain.CheckDposAckSketchy(dposAck)
	if check {
		peer.KnownDposAck(dposAck.Id())
		for _, peer := range h.peers.peersWithoutDposAcks(dposAck) {
			if err := peer.SendNewDposAck(dposAck); err != nil {
				log.Debug("SendNewDposAck", "err", err)
			}
		}
		h.chain.HandleDposAck(dposAck)
	} else {
		log.Debug("DposAck broadcast fail, because the dpos ack is illegality", "check", check, "number", dposAck.Number, "witnessSig", hexutils.BytesToHex(dposAck.WitnessSig), "BlockHash", dposAck.BlockHash, "Type", dposAck.AckType)
	}
	return nil
}

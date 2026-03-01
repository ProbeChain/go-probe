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

package downloader

import (
	"fmt"
	"sort"

	"github.com/probechain/go-probe/core/types"
)

// peerDropFn is a callback type for dropping a peer detected as malicious.
type peerDropFn func(id string)

// dataPack is a data message returned by a peer for some query.
type dataPack interface {
	PeerId() string
	Items() int
	Stats() string
}

// headerPack is a batch of block headers returned by a peer.
type headerPack struct {
	peerID  string
	headers []*types.Header
}

func (p *headerPack) PeerId() string { return p.peerID }
func (p *headerPack) Items() int     { return len(p.headers) }
func (p *headerPack) Stats() string  { return fmt.Sprintf("%d", len(p.headers)) }

// bodyPack is a batch of block bodies returned by a peer.
type bodyPack struct {
	peerID          string
	transactions    [][]*types.Transaction
	uncles          [][]*types.Header
	powAnswerUncles [][]*types.BehaviorProof
	acks        [][]*types.Ack
}

func (p *bodyPack) PeerId() string { return p.peerID }
func (p *bodyPack) Items() int {
	allLens := []int{len(p.transactions), len(p.uncles), len(p.powAnswerUncles), len(p.acks)}
	sort.Ints(allLens)
	return allLens[len(allLens)-1]
}
func (p *bodyPack) Stats() string {
	return fmt.Sprintf("%d:%d:%d:%d", len(p.transactions), len(p.uncles), len(p.powAnswerUncles), len(p.acks))
}

// receiptPack is a batch of receipts returned by a peer.
type receiptPack struct {
	peerID   string
	receipts [][]*types.Receipt
}

func (p *receiptPack) PeerId() string { return p.peerID }
func (p *receiptPack) Items() int     { return len(p.receipts) }
func (p *receiptPack) Stats() string  { return fmt.Sprintf("%d", len(p.receipts)) }

// statePack is a batch of states returned by a peer.
type statePack struct {
	peerID string
	states [][]byte
}

func (p *statePack) PeerId() string { return p.peerID }
func (p *statePack) Items() int     { return len(p.states) }
func (p *statePack) Stats() string  { return fmt.Sprintf("%d", len(p.states)) }

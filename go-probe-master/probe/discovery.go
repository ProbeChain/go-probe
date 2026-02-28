// Copyright 2019 The go-probeum Authors
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
	"github.com/probechain/go-probe/core"
	"github.com/probechain/go-probe/core/forkid"
	"github.com/probechain/go-probe/p2p/enode"
	"github.com/probechain/go-probe/rlp"
)

// probeEntry is the "probe" ENR entry which advertises probe protocol
// on the discovery network.
type probeEntry struct {
	ForkID forkid.ID // Fork identifier per EIP-2124

	// Ignore additional fields (for forward compatibility).
	Rest []rlp.RawValue `rlp:"tail"`
}

// ENRKey implements enr.Entry.
func (e probeEntry) ENRKey() string {
	return "probe"
}

// startProbeEntryUpdate starts the ENR updater loop.
func (probe *Probeum) startProbeEntryUpdate(ln *enode.LocalNode) {
	var newHead = make(chan core.ChainHeadEvent, 10)
	sub := probe.blockchain.SubscribeChainHeadEvent(newHead)

	go func() {
		defer sub.Unsubscribe()
		for {
			select {
			case <-newHead:
				ln.Set(probe.currentProbeEntry())
			case <-sub.Err():
				// Would be nice to sync with probe.Stop, but there is no
				// good way to do that.
				return
			}
		}
	}()
}

func (probe *Probeum) currentProbeEntry() *probeEntry {
	return &probeEntry{ForkID: forkid.NewID(probe.blockchain.Config(), probe.blockchain.Genesis().Hash(),
		probe.blockchain.CurrentHeader().Number.Uint64())}
}

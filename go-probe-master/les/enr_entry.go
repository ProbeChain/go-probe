// Copyright 2019 The ProbeChain Authors
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

package les

import (
	"github.com/probechain/go-probe/core/forkid"
	"github.com/probechain/go-probe/p2p/dnsdisc"
	"github.com/probechain/go-probe/p2p/enode"
	"github.com/probechain/go-probe/rlp"
)

// lesEntry is the "les" ENR entry. This is set for LES servers only.
type lesEntry struct {
	// Ignore additional fields (for forward compatibility).
	VfxVersion uint
	Rest       []rlp.RawValue `rlp:"tail"`
}

func (lesEntry) ENRKey() string { return "les" }

// probeEntry is the "probe" ENR entry. This is redeclared here to avoid depending on package probe.
type probeEntry struct {
	ForkID forkid.ID
	Tail   []rlp.RawValue `rlp:"tail"`
}

func (probeEntry) ENRKey() string { return "probe" }

// setupDiscovery creates the node discovery source for the probe protocol.
func (probe *LightProbeum) setupDiscovery() (enode.Iterator, error) {
	it := enode.NewFairMix(0)

	// Enable DNS discovery.
	if len(probe.config.ProbeDiscoveryURLs) != 0 {
		client := dnsdisc.NewClient(dnsdisc.Config{})
		dns, err := client.NewIterator(probe.config.ProbeDiscoveryURLs...)
		if err != nil {
			return nil, err
		}
		it.AddSource(dns)
	}

	// Enable DHT.
	if probe.udpEnabled {
		it.AddSource(probe.p2pServer.DiscV5.RandomNodes())
	}

	forkFilter := forkid.NewFilter(probe.blockchain)
	iterator := enode.Filter(it, func(n *enode.Node) bool { return nodeIsServer(forkFilter, n) })
	return iterator, nil
}

// nodeIsServer checks whprobeer n is an LES server node.
func nodeIsServer(forkFilter forkid.Filter, n *enode.Node) bool {
	var les lesEntry
	var probe probeEntry
	return n.Load(&les) == nil && n.Load(&probe) == nil && forkFilter(probe.ForkID) == nil
}

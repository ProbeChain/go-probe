// Copyright 2020 The go-probeum Authors
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

package probetest

import (
	"os"
	"testing"
	"time"

	"github.com/probeum/go-probeum/probe"
	"github.com/probeum/go-probeum/probe/probeconfig"
	"github.com/probeum/go-probeum/internal/utesting"
	"github.com/probeum/go-probeum/node"
	"github.com/probeum/go-probeum/p2p"
)

var (
	genesisFile   = "./testdata/genesis.json"
	halfchainFile = "./testdata/halfchain.rlp"
	fullchainFile = "./testdata/chain.rlp"
)

func TestProbeSuite(t *testing.T) {
	gprobe, err := runGprobe()
	if err != nil {
		t.Fatalf("could not run gprobe: %v", err)
	}
	defer gprobe.Close()

	suite, err := NewSuite(gprobe.Server().Self(), fullchainFile, genesisFile)
	if err != nil {
		t.Fatalf("could not create new test suite: %v", err)
	}
	for _, test := range suite.AllProbeTests() {
		t.Run(test.Name, func(t *testing.T) {
			result := utesting.RunTAP([]utesting.Test{{Name: test.Name, Fn: test.Fn}}, os.Stdout)
			if result[0].Failed {
				t.Fatal()
			}
		})
	}
}

// runGprobe creates and starts a gprobe node
func runGprobe() (*node.Node, error) {
	stack, err := node.New(&node.Config{
		P2P: p2p.Config{
			ListenAddr:  "127.0.0.1:0",
			NoDiscovery: true,
			MaxPeers:    10, // in case a test requires multiple connections, can be changed in the future
			NoDial:      true,
		},
	})
	if err != nil {
		return nil, err
	}

	err = setupGprobe(stack)
	if err != nil {
		stack.Close()
		return nil, err
	}
	if err = stack.Start(); err != nil {
		stack.Close()
		return nil, err
	}
	return stack, nil
}

func setupGprobe(stack *node.Node) error {
	chain, err := loadChain(halfchainFile, genesisFile)
	if err != nil {
		return err
	}

	backend, err := probe.New(stack, &probeconfig.Config{
		Genesis:                 &chain.genesis,
		NetworkId:               chain.genesis.Config.ChainID.Uint64(), // 19763
		DatabaseCache:           10,
		TrieCleanCache:          10,
		TrieCleanCacheJournal:   "",
		TrieCleanCacheRejournal: 60 * time.Minute,
		TrieDirtyCache:          16,
		TrieTimeout:             60 * time.Minute,
		SnapshotCache:           10,
	})
	if err != nil {
		return err
	}

	_, err = backend.BlockChain().InsertChain(chain.blocks[1:])
	return err
}

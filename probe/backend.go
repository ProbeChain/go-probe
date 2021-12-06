// Copyright 2014 The go-probeum Authors
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

// Package probe implements the Probeum protocol.
package probe

import (
	"errors"
	"fmt"
	"github.com/probeum/go-probeum/consensus/greatri"
	probehash2 "github.com/probeum/go-probeum/consensus/probeash"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/probeum/go-probeum/accounts"
	"github.com/probeum/go-probeum/common"
	"github.com/probeum/go-probeum/common/hexutil"
	"github.com/probeum/go-probeum/consensus"
	"github.com/probeum/go-probeum/consensus/clique"
	"github.com/probeum/go-probeum/core"
	"github.com/probeum/go-probeum/core/bloombits"
	"github.com/probeum/go-probeum/core/rawdb"
	"github.com/probeum/go-probeum/core/state/pruner"
	"github.com/probeum/go-probeum/core/types"
	"github.com/probeum/go-probeum/core/vm"
	"github.com/probeum/go-probeum/event"
	"github.com/probeum/go-probeum/internal/probeapi"
	"github.com/probeum/go-probeum/log"
	"github.com/probeum/go-probeum/miner"
	"github.com/probeum/go-probeum/node"
	"github.com/probeum/go-probeum/p2p"
	"github.com/probeum/go-probeum/p2p/dnsdisc"
	"github.com/probeum/go-probeum/p2p/enode"
	"github.com/probeum/go-probeum/params"
	"github.com/probeum/go-probeum/probe/downloader"
	"github.com/probeum/go-probeum/probe/filters"
	"github.com/probeum/go-probeum/probe/gasprice"
	"github.com/probeum/go-probeum/probe/probeconfig"
	"github.com/probeum/go-probeum/probe/protocols/probe"
	"github.com/probeum/go-probeum/probe/protocols/snap"
	"github.com/probeum/go-probeum/probedb"
	"github.com/probeum/go-probeum/rlp"
	"github.com/probeum/go-probeum/rpc"
)

// Config contains the configuration options of the ETH protocol.
// Deprecated: use probeconfig.Config instead.
type Config = probeconfig.Config

// Probeum implements the Probeum full node service.
type Probeum struct {
	config *probeconfig.Config

	// Handlers
	txPool              *core.TxPool
	blockchain          *core.BlockChain
	handler             *handler
	probeDialCandidates enode.Iterator
	snapDialCandidates  enode.Iterator

	// DB interfaces
	chainDb probedb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	powEngine      consensus.Engine
	accountManager *accounts.Manager

	bloomRequests     chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer      *core.ChainIndexer             // Bloom indexer operating during block imports
	closeBloomHandler chan struct{}

	APIBackend *ProbeAPIBackend

	miner       *miner.Miner
	gasPrice    *big.Int
	probeerbase common.Address

	networkID     uint64
	netRPCService *probeapi.PublicNetAPI

	p2pServer *p2p.Server

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and probeerbase)
}

// New creates a new Probeum object (including the
// initialisation of the common Probeum object)
func New(stack *node.Node, config *probeconfig.Config) (*Probeum, error) {
	// Ensure configuration values are compatible and sane
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run probe.Probeum in light sync mode, use les.LightProbeum")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	if config.Miner.GasPrice == nil || config.Miner.GasPrice.Cmp(common.Big0) <= 0 {
		log.Warn("Sanitizing invalid miner gas price", "provided", config.Miner.GasPrice, "updated", probeconfig.Defaults.Miner.GasPrice)
		config.Miner.GasPrice = new(big.Int).Set(probeconfig.Defaults.Miner.GasPrice)
	}
	if config.NoPruning && config.TrieDirtyCache > 0 {
		if config.SnapshotCache > 0 {
			config.TrieCleanCache += config.TrieDirtyCache * 3 / 5
			config.SnapshotCache += config.TrieDirtyCache * 2 / 5
		} else {
			config.TrieCleanCache += config.TrieDirtyCache
		}
		config.TrieDirtyCache = 0
	}
	log.Info("Allocated trie memory caches", "clean", common.StorageSize(config.TrieCleanCache)*1024*1024, "dirty", common.StorageSize(config.TrieDirtyCache)*1024*1024)

	// Transfer mining-related config to the probeash config.
	probeashConfig := config.Probeash
	probeashConfig.NotifyFull = config.Miner.NotifyFull

	// Assemble the Probeum object
	chainDb, err := stack.OpenDatabaseWithFreezer("chaindata", config.DatabaseCache, config.DatabaseHandles, config.DatabaseFreezer, "probe/db/chaindata/", false)
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlockWithOverride(chainDb, config.Genesis, config.OverrideLondon, stack.InstanceDir())
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	if err := pruner.RecoverPruning(stack.ResolvePath(""), chainDb, stack.ResolvePath(config.TrieCleanCacheJournal)); err != nil {
		log.Error("Failed to recover state", "error", err)
	}
	probe := &Probeum{
		config:            config,
		chainDb:           chainDb,
		eventMux:          stack.EventMux(),
		accountManager:    stack.AccountManager(),
		closeBloomHandler: make(chan struct{}),
		networkID:         config.NetworkId,
		gasPrice:          config.Miner.GasPrice,
		probeerbase:       config.Miner.Probeerbase,
		bloomRequests:     make(chan chan *bloombits.Retrieval),
		bloomIndexer:      core.NewBloomIndexer(chainDb, params.BloomBitsBlocks, params.BloomConfirms),
		p2pServer:         stack.Server(),
	}

	if chainConfig.Probeash != nil {
		probe.powEngine = probeconfig.CreatePowConsensusEngine(stack, chainConfig, &probeashConfig, config.Miner.Notify, config.Miner.Noverify, chainDb)
	}
	probe.engine = probeconfig.CreateConsensusEngine(stack, chainConfig, &probeashConfig, config.Miner.Notify, config.Miner.Noverify, chainDb, probe.powEngine)

	bcVersion := rawdb.ReadDatabaseVersion(chainDb)
	var dbVer = "<nil>"
	if bcVersion != nil {
		dbVer = fmt.Sprintf("%d", *bcVersion)
	}
	log.Info("Initialising Probeum protocol", "network", config.NetworkId, "dbversion", dbVer)

	if !config.SkipBcVersionCheck {
		if bcVersion != nil && *bcVersion > core.BlockChainVersion {
			return nil, fmt.Errorf("database version is v%d, Gprobe %s only supports v%d", *bcVersion, params.VersionWithMeta, core.BlockChainVersion)
		} else if bcVersion == nil || *bcVersion < core.BlockChainVersion {
			if bcVersion != nil { // only print warning on upgrade, not on init
				log.Warn("Upgrade blockchain database version", "from", dbVer, "to", core.BlockChainVersion)
			}
			rawdb.WriteDatabaseVersion(chainDb, core.BlockChainVersion)
		}
	}
	var (
		vmConfig = vm.Config{
			EnablePreimageRecording: config.EnablePreimageRecording,
			EWASMInterpreter:        config.EWASMInterpreter,
			EVMInterpreter:          config.EVMInterpreter,
		}
		cacheConfig = &core.CacheConfig{
			TrieCleanLimit:      config.TrieCleanCache,
			TrieCleanJournal:    stack.ResolvePath(config.TrieCleanCacheJournal),
			TrieCleanRejournal:  config.TrieCleanCacheRejournal,
			TrieCleanNoPrefetch: config.NoPrefetch,
			TrieDirtyLimit:      config.TrieDirtyCache,
			TrieDirtyDisabled:   config.NoPruning,
			TrieTimeLimit:       config.TrieTimeout,
			SnapshotLimit:       config.SnapshotCache,
			Preimages:           config.Preimages,
			DataDir:             stack.DataDir(),
		}
	)
	probe.blockchain, err = core.NewBlockChain(chainDb, cacheConfig, chainConfig, probe.engine, vmConfig, probe.shouldPreserve, &config.TxLookupLimit, probe.p2pServer, probe.powEngine)
	originDifficulty := probe.blockchain.GetBlockByNumber(0).Difficulty().Int64()
	if probeash, ok := probe.powEngine.(*probehash2.Probeash); ok {
		probeash.SetMinDifficulty(originDifficulty)
	}
	if err != nil {
		return nil, err
	}

	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		probe.blockchain.SetHead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	probe.bloomIndexer.Start(probe.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = stack.ResolvePath(config.TxPool.Journal)
	}
	probe.txPool = core.NewTxPool(config.TxPool, chainConfig, probe.blockchain)

	// Permit the downloader to use the trie cache allowance during fast sync
	cacheLimit := cacheConfig.TrieCleanLimit + cacheConfig.TrieDirtyLimit + cacheConfig.SnapshotLimit
	checkpoint := config.Checkpoint
	if checkpoint == nil {
		checkpoint = params.TrustedCheckpoints[genesisHash]
	}
	if probe.handler, err = newHandler(&handlerConfig{
		Database:   chainDb,
		Chain:      probe.blockchain,
		TxPool:     probe.txPool,
		Network:    config.NetworkId,
		Sync:       config.SyncMode,
		BloomCache: uint64(cacheLimit),
		EventMux:   probe.eventMux,
		Checkpoint: checkpoint,
		Whitelist:  config.Whitelist,
	}); err != nil {
		return nil, err
	}

	probe.miner = miner.New(probe, &config.Miner, chainConfig, probe.EventMux(), probe.engine, probe.powEngine, probe.isLocalBlock)
	probe.miner.SetMinDifficulty(originDifficulty)
	probe.miner.SetExtra(makeExtraData(config.Miner.ExtraData))

	coinbase, err := probe.Probeerbase()
	if err == nil {
		probe.SetProbeerbase(coinbase)
	}

	probe.APIBackend = &ProbeAPIBackend{stack.Config().ExtRPCEnabled(), stack.Config().AllowUnprotectedTxs, probe, nil}
	if probe.APIBackend.allowUnprotectedTxs {
		log.Info("Unprotected transactions allowed")
	}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.Miner.GasPrice
	}
	probe.APIBackend.gpo = gasprice.NewOracle(probe.APIBackend, gpoParams)

	// Setup DNS discovery iterators.
	dnsclient := dnsdisc.NewClient(dnsdisc.Config{})
	probe.probeDialCandidates, err = dnsclient.NewIterator(probe.config.ProbeDiscoveryURLs...)
	if err != nil {
		return nil, err
	}
	probe.snapDialCandidates, err = dnsclient.NewIterator(probe.config.SnapDiscoveryURLs...)
	if err != nil {
		return nil, err
	}

	// Start the RPC service
	probe.netRPCService = probeapi.NewPublicNetAPI(probe.p2pServer, config.NetworkId)

	// Register the backend on the node
	stack.RegisterAPIs(probe.APIs())
	stack.RegisterProtocols(probe.Protocols())
	stack.RegisterLifecycle(probe)
	// Check for unclean shutdown
	if uncleanShutdowns, discards, err := rawdb.PushUncleanShutdownMarker(chainDb); err != nil {
		log.Error("Could not update unclean-shutdown-marker list", "error", err)
	} else {
		if discards > 0 {
			log.Warn("Old unclean shutdowns found", "count", discards)
		}
		for _, tstamp := range uncleanShutdowns {
			t := time.Unix(int64(tstamp), 0)
			log.Warn("Unclean shutdown detected", "booted", t,
				"age", common.PrettyAge(t))
		}
	}

	dposAccounts := probe.blockchain.GetDposAccounts(probe.blockchain.CurrentHeader().Number.Uint64())
	nodes := make([]*enode.Node, 0, len(dposAccounts))
	for _, account := range dposAccounts {
		dposEnode, err := enode.Parse(enode.ValidSchemes, string(account.Enode[:]))
		if err != nil {
			log.Error(fmt.Sprintf("Node URL %s: %v\n", string(account.Enode[:]), err))
			continue
		}
		nodes = append(nodes, dposEnode)
	}
	probe.p2pServer.Config.StaticNodes = append(probe.p2pServer.Config.StaticNodes, nodes...)
	probe.p2pServer.Config.TrustedNodes = append(probe.p2pServer.Config.TrustedNodes, nodes...)

	return probe, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"gprobe",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// APIs return the collection of RPC services the probeum package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *Probeum) APIs() []rpc.API {
	apis := probeapi.GetAPIs(s.APIBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "probe",
			Version:   "1.0",
			Service:   NewPublicProbeumAPI(s),
			Public:    true,
		}, {
			Namespace: "probe",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "probe",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.handler.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "probe",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.APIBackend, false, 5*time.Minute),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *Probeum) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *Probeum) Probeerbase() (eb common.Address, err error) {
	s.lock.RLock()
	probeerbase := s.probeerbase
	s.lock.RUnlock()

	if probeerbase != (common.Address{}) {
		return probeerbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			probeerbase := accounts[0].Address

			s.lock.Lock()
			s.probeerbase = probeerbase
			s.lock.Unlock()

			log.Info("Probeerbase automatically configured", "address", probeerbase)
			return probeerbase, nil
		}
	}
	log.Info("probeerbase must be explicitly specified")
	return common.Address{}, fmt.Errorf("probeerbase must be explicitly specified")
}

// isLocalBlock checks whprobeer the specified block is mined
// by local miner accounts.
//
// We regard two types of accounts as local miner account: probeerbase
// and accounts specified via `txpool.locals` flag.
func (s *Probeum) isLocalBlock(block *types.Block) bool {
	author, err := s.engine.Author(block.Header())
	if err != nil {
		log.Warn("Failed to retrieve block author", "number", block.NumberU64(), "hash", block.Hash(), "err", err)
		return false
	}
	// Check whprobeer the given address is probeerbase.
	s.lock.RLock()
	probeerbase := s.probeerbase
	s.lock.RUnlock()
	if author == probeerbase {
		return true
	}
	// Check whprobeer the given address is specified by `txpool.local`
	// CLI flag.
	for _, account := range s.config.TxPool.Locals {
		if account == author {
			return true
		}
	}
	return false
}

// shouldPreserve checks whprobeer we should preserve the given block
// during the chain reorg depending on whprobeer the author of block
// is a local account.
func (s *Probeum) shouldPreserve(block *types.Block) bool {
	// The reason we need to disable the self-reorg preserving for clique
	// is it can be probable to introduce a deadlock.
	//
	// e.g. If there are 7 available signers
	//
	// r1   A
	// r2     B
	// r3       C
	// r4         D
	// r5   A      [X] F G
	// r6    [X]
	//
	// In the round5, the inturn signer E is offline, so the worst case
	// is A, F and G sign the block of round5 and reject the block of opponents
	// and in the round6, the last available signer B is offline, the whole
	// network is stuck.
	if _, ok := s.engine.(*clique.Clique); ok {
		return false
	}
	return s.isLocalBlock(block)
}

// SetProbeerbase sets the mining reward address.
func (s *Probeum) SetProbeerbase(probeerbase common.Address) {
	s.lock.Lock()
	s.probeerbase = probeerbase
	s.lock.Unlock()

	s.miner.SetProbeerbase(probeerbase)
}

// StartMining starts the miner with the given number of CPU threads. If mining
// is already running, this method adjust the number of threads allowed to use
// and updates the minimum price required by the transaction pool.
func (s *Probeum) StartMining(threads int) error {
	// Update the thread count within the consensus engine
	type threaded interface {
		SetThreads(threads int)
	}
	if th, ok := s.powEngine.(threaded); ok {
		log.Info("Updated mining threads", "threads", threads)
		if threads == 0 {
			threads = -1 // Disable the miner from within
		}
		th.SetThreads(threads)
	}
	// If the miner was not running, initialize it
	if !s.IsMining() {
		// Propagate the initial price point to the transaction pool
		s.lock.RLock()
		price := s.gasPrice
		s.lock.RUnlock()
		s.txPool.SetGasPrice(price)

		// Configure the local mining address
		eb, err := s.Probeerbase()
		if err != nil {
			log.Error("Cannot start mining without probeerbase", "err", err)
			return fmt.Errorf("probeerbase missing: %v", err)
		}
		//if clique, ok := s.engine.(*clique.Clique); ok {
		//	wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		//	if wallet == nil || err != nil {
		//		log.Error("Probeerbase account unavailable locally", "err", err)
		//		return fmt.Errorf("signer missing: %v", err)
		//	}
		//	clique.Authorize(eb, wallet.SignData)
		//}

		if greatri, ok := s.engine.(*greatri.Greatri); ok {
			wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
			if wallet == nil || err != nil {
				log.Error("Coinbase account unavailable locally", "err", err)
				return fmt.Errorf("signer missing: %v", err)
			}
			greatri.Authorize(eb, wallet.SignData)
		}

		// If mining is started, we can disable the transaction rejection mechanism
		// introduced to speed sync times.
		atomic.StoreUint32(&s.handler.acceptTxs, 1)

		go s.miner.Start(eb)
	}
	return nil
}

// StopMining terminates the miner, both at the consensus engine level as well as
// at the block creation level.
func (s *Probeum) StopMining() {
	// Update the thread count within the consensus engine
	type threaded interface {
		SetThreads(threads int)
	}
	if th, ok := s.engine.(threaded); ok {
		th.SetThreads(-1)
	}
	// Stop the block creating itself
	s.miner.Stop()
}

func (s *Probeum) IsMining() bool      { return s.miner.Mining() }
func (s *Probeum) Miner() *miner.Miner { return s.miner }

func (s *Probeum) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *Probeum) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *Probeum) TxPool() *core.TxPool               { return s.txPool }
func (s *Probeum) EventMux() *event.TypeMux           { return s.eventMux }
func (s *Probeum) Engine() consensus.Engine           { return s.engine }
func (s *Probeum) ChainDb() probedb.Database          { return s.chainDb }
func (s *Probeum) IsListening() bool                  { return true } // Always listening
func (s *Probeum) Downloader() *downloader.Downloader { return s.handler.downloader }
func (s *Probeum) Synced() bool                       { return atomic.LoadUint32(&s.handler.acceptTxs) == 1 }
func (s *Probeum) ArchiveMode() bool                  { return s.config.NoPruning }
func (s *Probeum) BloomIndexer() *core.ChainIndexer   { return s.bloomIndexer }

// Protocols returns all the currently configured
// network protocols to start.
func (s *Probeum) Protocols() []p2p.Protocol {
	protos := probe.MakeProtocols((*probeHandler)(s.handler), s.networkID, s.probeDialCandidates)
	if s.config.SnapshotCache > 0 {
		protos = append(protos, snap.MakeProtocols((*snapHandler)(s.handler), s.snapDialCandidates)...)
	}
	return protos
}

// Start implements node.Lifecycle, starting all internal goroutines needed by the
// Probeum protocol implementation.
func (s *Probeum) Start() error {
	probe.StartENRUpdater(s.blockchain, s.p2pServer.LocalNode())

	// Start the bloom bits servicing goroutines
	s.startBloomHandlers(params.BloomBitsBlocks)

	// Figure out a max peers count based on the server limits
	maxPeers := s.p2pServer.MaxPeers
	if s.config.LightServ > 0 {
		if s.config.LightPeers >= s.p2pServer.MaxPeers {
			return fmt.Errorf("invalid peer config: light peer count (%d) >= total peer count (%d)", s.config.LightPeers, s.p2pServer.MaxPeers)
		}
		maxPeers -= s.config.LightPeers
	}
	// Start the networking layer and the light server if requested
	s.handler.Start(maxPeers)

	return nil
}

// Stop implements node.Lifecycle, terminating all internal goroutines used by the
// Probeum protocol.
func (s *Probeum) Stop() error {
	// Stop all the peer-related stuff first.
	s.probeDialCandidates.Close()
	s.snapDialCandidates.Close()
	s.handler.Stop()

	// Then stop everything else.
	s.bloomIndexer.Close()
	close(s.closeBloomHandler)
	s.txPool.Stop()
	s.miner.Stop()
	s.blockchain.Stop()
	s.engine.Close()
	rawdb.PopUncleanShutdownMarker(s.chainDb)
	s.chainDb.Close()
	s.eventMux.Stop()

	return nil
}

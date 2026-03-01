// Copyright 2024 The ProbeChain Authors
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

// Package pob implements the Proof-of-Behavior consensus engine.
// PoB links consensus influence to verifiably trustworthy conduct via three pillars:
// 1. Layered Utility Scoring - AI Agent scores each validator's actions
// 2. Dynamic Weight Adaptation - Validator block-production weight adjusts based on scores
// 3. Decentralized Verification with Proportional Slashing
package pob

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/big"
	"runtime"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/probechain/go-probe/accounts"
	"github.com/probechain/go-probe/common"
	"github.com/probechain/go-probe/consensus"
	atomicClock "github.com/probechain/go-probe/core/atomic"
	"github.com/probechain/go-probe/consensus/misc"
	"github.com/probechain/go-probe/core/state"
	"github.com/probechain/go-probe/core/types"
	"github.com/probechain/go-probe/crypto"
	"github.com/probechain/go-probe/crypto/dilithium"
	"github.com/probechain/go-probe/crypto/secp256k1"
	"github.com/probechain/go-probe/log"
	"github.com/probechain/go-probe/params"
	"github.com/probechain/go-probe/probedb"
	"github.com/probechain/go-probe/rlp"
	"github.com/probechain/go-probe/rpc"
	"github.com/probechain/go-probe/trie"
	"golang.org/x/crypto/sha3"
)

const (
	checkpointInterval = 1024 // Number of blocks after which to save the vote snapshot to the database
	inmemorySnapshots  = 128  // Number of recent vote snapshots to keep in memory
	inmemorySignatures = 4096 // Number of recent block signatures to keep in memory
	maxUncleBehaviorProof  = 5
	wiggleTime         = 500 * time.Millisecond // Random delay (per validator) to allow concurrent signers
)

// ProofOfBehavior protocol constants.
var (
	BlockRewardPowMiner      = big.NewInt(3e+18)
	BlockRewardPobValidator  = big.NewInt(1e+18)
	BlockRewardPowMinerUncle = big.NewInt(5e+17)

	epochLength = uint64(30000) // Default number of blocks after which to checkpoint

	extraVanity = 32                     // Fixed number of extra-data prefix bytes reserved for vanity
	extraSeal   = crypto.SignatureLength // Fixed number of extra-data suffix bytes reserved for seal

	uncleHash = types.CalcBehaviorProofUncleHash(nil) // Always Keccak256(RLP([])) as uncles are meaningless

	diffInTurn = big.NewInt(2) // Block difficulty for in-turn validators
	diffNoTurn = big.NewInt(1) // Block difficulty for out-of-turn validators

	allowedFutureBlockTimeSeconds          = int64(15) // Max seconds from current time allowed for blocks
	allowedFutureBlockTimeSecondsStellar   = int64(1)  // Max seconds from current time allowed in StellarSpeed mode
)

// Various error messages to mark blocks invalid.
var (
	errUnknownBlock              = errors.New("unknown block")
	errInvalidCheckpointBeneficiary = errors.New("beneficiary in checkpoint block non-zero")
	errInvalidVote               = errors.New("vote nonce not 0x00..0 or 0xff..f")
	errMissingVanity             = errors.New("extra-data 32 byte vanity prefix missing")
	errMissingSignature          = errors.New("extra-data 65 byte signature suffix missing")
	errExtraValidators           = errors.New("non-checkpoint block contains extra validator list")
	errInvalidCheckpointValidators = errors.New("invalid validator list on checkpoint block")
	errMismatchingCheckpointValidators = errors.New("mismatching validator list on checkpoint block")
	errInvalidMixDigest          = errors.New("non-zero mix digest")
	errInvalidUncleHash          = errors.New("non empty uncle hash")
	errInvalidDifficulty         = errors.New("invalid difficulty")
	errWrongDifficulty           = errors.New("wrong difficulty")
	errInvalidTimestamp           = errors.New("invalid timestamp")
	errInvalidVotingChain        = errors.New("invalid voting chain")
	errUnauthorizedValidator     = errors.New("unauthorized validator")
	errRecentlySigned            = errors.New("recently signed")
)

var (
	errOlderBlockTime  = errors.New("timestamp older than parent")
	errTooManyUncles   = errors.New("too many uncles")
	errDuplicateUncle  = errors.New("duplicate uncle")
	errUncleIsAncestor = errors.New("uncle is ancestor")
	errDanglingUncle   = errors.New("uncle's parent is not ancestor")
	errInvalidPoW      = errors.New("invalid proof-of-work")
)

type Mode uint

const (
	ModeNormal Mode = iota
	ModeShared
	ModeTest
	ModeFake
	ModeFullFake
)

// Config are the configuration parameters of the pob engine.
type Config struct {
	CacheDir         string
	CachesInMem      int
	CachesOnDisk     int
	CachesLockMmap   bool
	DatasetDir       string
	DatasetsInMem    int
	DatasetsOnDisk   int
	DatasetsLockMmap bool
	PowMode          Mode
	NotifyFull       bool
	Log              log.Logger `toml:"-"`
}

// SignerFn hashes and signs the data to be signed by a backing account.
type SignerFn func(signer accounts.Account, mimeType string, message []byte) ([]byte, error)

// ecrecover extracts the ProbeChain account address from a signed header.
// Supports both ECDSA (65-byte) and Dilithium signatures in header.Extra.
func ecrecover(header *types.Header, sigcache *lru.ARCCache) (common.Address, error) {
	hash := header.Hash()
	if address, known := sigcache.Get(hash); known {
		return address.(common.Address), nil
	}
	if len(header.Extra) < extraSeal {
		return common.Address{}, errMissingSignature
	}
	signature := header.Extra[len(header.Extra)-extraSeal:]
	pubkey, err := crypto.Ecrecover(SealHash(header).Bytes(), signature)
	if err != nil {
		return common.Address{}, err
	}
	var signer common.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])
	sigcache.Add(hash, signer)
	return signer, nil
}

// ProofOfBehavior is the Proof-of-Behavior consensus engine.
type ProofOfBehavior struct {
	pobConfig   *params.PobConfig   // PoB consensus engine configuration parameters
	chainConfig *params.ChainConfig // Chain configuration
	config      Config              // Engine config (for PoW mode settings)
	db          probedb.Database    // Database to store and retrieve snapshot checkpoints
	agent       *BehaviorAgent      // AI behavior scoring agent

	recents    *lru.ARCCache // Snapshots for recent block to speed up reorgs
	signatures *lru.ARCCache // Signatures of recent blocks to speed up mining

	proposals map[common.Address]bool // Current list of proposals we are pushing

	signer common.Address // ProbeChain address of the signing key
	signFn SignerFn       // Signer function to authorize hashes with
	lock   sync.RWMutex   // Protects the signer fields

	// The fields below are for testing only
	fakeDiff bool // Skip difficulty verifications
}

// NewFaker creates a PoB consensus engine with a fake scheme that accepts
// all blocks as valid. Used for testing.
func NewFaker() *ProofOfBehavior {
	return &ProofOfBehavior{
		config: Config{
			PowMode: ModeFake,
			Log:     log.Root(),
		},
		pobConfig: &params.PobConfig{Period: 15, Epoch: 30000},
		fakeDiff:  true,
	}
}

// NewFakeFailer creates a PoB consensus engine with a fake scheme that
// accepts all blocks as valid apart from the single one specified.
func NewFakeFailer(fail uint64) *ProofOfBehavior {
	return &ProofOfBehavior{
		config: Config{
			PowMode: ModeFake,
			Log:     log.Root(),
		},
		pobConfig: &params.PobConfig{Period: 15, Epoch: 30000},
		fakeDiff:  true,
	}
}

// NewFakeDelayer creates a PoB consensus engine with a fake scheme that
// delays sealing by the given duration.
func NewFakeDelayer(delay time.Duration) *ProofOfBehavior {
	return &ProofOfBehavior{
		config: Config{
			PowMode: ModeFake,
			Log:     log.Root(),
		},
		pobConfig: &params.PobConfig{Period: 15, Epoch: 30000},
		fakeDiff:  true,
	}
}

// NewFullFaker creates a PoB consensus engine with a full fake scheme that
// accepts all blocks as valid, without checking any consensus rules.
func NewFullFaker() *ProofOfBehavior {
	return &ProofOfBehavior{
		config: Config{
			PowMode: ModeFullFake,
			Log:     log.Root(),
		},
		pobConfig: &params.PobConfig{Period: 15, Epoch: 30000},
		fakeDiff:  true,
	}
}

// New creates a ProofOfBehavior consensus engine with the initial validators
// set to the ones provided by the user.
func New(config *params.PobConfig, db probedb.Database, chainConfig *params.ChainConfig) *ProofOfBehavior {
	conf := *config
	if conf.Epoch == 0 {
		conf.Epoch = epochLength
	}
	if conf.InitialScore == 0 {
		conf.InitialScore = defaultInitialScore
	}
	if conf.SlashFraction == 0 {
		conf.SlashFraction = 1000 // Default: 10% (1000 basis points)
	}
	if conf.DemotionThreshold == 0 {
		conf.DemotionThreshold = 1000 // Default: score below 1000 demotes
	}

	recents, _ := lru.NewARC(inmemorySnapshots)
	signatures, _ := lru.NewARC(inmemorySignatures)

	return &ProofOfBehavior{
		pobConfig:   &conf,
		chainConfig: chainConfig,
		db:          db,
		agent:       NewBehaviorAgent(),
		recents:     recents,
		signatures:  signatures,
		proposals:   make(map[common.Address]bool),
	}
}

// Author implements consensus.Engine, returning the ProbeChain address recovered
// from the signature in the header's extra-data section.
func (c *ProofOfBehavior) Author(header *types.Header) (common.Address, error) {
	return header.ValidatorAddr, nil
}

// VerifyHeader checks whether a header conforms to the consensus rules.
func (c *ProofOfBehavior) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header, seal bool) error {
	if c.config.PowMode == ModeFullFake {
		return nil
	}
	number := header.Number.Uint64()
	if chain.GetHeader(header.Hash(), number) != nil {
		return nil
	}
	parent, err, diff := c.FindRealParentHeader(chain, header, nil, -1)
	if err != nil {
		return err
	}
	return c.verifyHeader(chain, header, parent, false, seal, time.Now().Unix(), diff)
}

// FindRealParentHeader walks backwards through visual blocks to find the real parent.
func (c *ProofOfBehavior) FindRealParentHeader(chain consensus.ChainHeaderReader, header *types.Header, headers []*types.Header, index int) (*types.Header, error, int64) {
	var parent = header
	var diff int64 = 1
	for {
		if index > 0 && headers != nil {
			parent = headers[index-1]
			if parent.Hash() != headers[index].ParentHash || new(big.Int).Sub(headers[index].Number, parent.Number).Cmp(common.Big1) != 0 {
				return nil, consensus.ErrUnknownAncestor, diff
			}
			index--
		} else if index == 0 {
			parent = chain.GetHeader(headers[0].ParentHash, headers[0].Number.Uint64()-1)
			index--
		} else {
			parent = chain.GetHeader(parent.ParentHash, parent.Number.Uint64()-1)
		}
		if parent == nil {
			return nil, consensus.ErrUnknownAncestor, diff
		}
		if !parent.IsVisual() {
			return parent, nil, diff
		}
		diff++
	}
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers concurrently.
func (c *ProofOfBehavior) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	if c.config.PowMode == ModeFullFake || len(headers) == 0 {
		abort, results := make(chan struct{}), make(chan error, len(headers))
		for i := 0; i < len(headers); i++ {
			results <- nil
		}
		return abort, results
	}

	workers := runtime.GOMAXPROCS(0)
	if len(headers) < workers {
		workers = len(headers)
	}

	var (
		inputs  = make(chan int)
		done    = make(chan int, workers)
		errs    = make([]error, len(headers))
		abort   = make(chan struct{})
		unixNow = time.Now().Unix()
	)
	for i := 0; i < workers; i++ {
		go func() {
			for index := range inputs {
				errs[index] = c.verifyHeaderWorker(chain, headers, seals, index, unixNow)
				done <- index
			}
		}()
	}

	errorsOut := make(chan error, len(headers))
	go func() {
		defer close(inputs)
		var (
			in, out = 0, 0
			checked = make([]bool, len(headers))
			inputs  = inputs
		)
		for {
			select {
			case inputs <- in:
				if in++; in == len(headers) {
					inputs = nil
				}
			case index := <-done:
				for checked[index] = true; checked[out]; out++ {
					errorsOut <- errs[out]
					if out == len(headers)-1 {
						return
					}
				}
			case <-abort:
				return
			}
		}
	}()
	return abort, errorsOut
}

func (c *ProofOfBehavior) verifyHeaderWorker(chain consensus.ChainHeaderReader, headers []*types.Header, seals []bool, index int, unixNow int64) error {
	parent, err, diff := c.FindRealParentHeader(chain, headers[index], headers, index)
	if err != nil {
		return err
	}
	return c.verifyHeader(chain, headers[index], parent, false, seals[index], unixNow, diff)
}

// verifyHeader checks whether a header conforms to the consensus rules.
func (c *ProofOfBehavior) verifyHeader(chain consensus.ChainHeaderReader, header, parent *types.Header, uncle bool, seal bool, unixNow int64, diff int64) error {
	log.Trace("pob verifyHeader", "block number", header.Number, "seal", seal)

	// Verify the ValidatorSig matches ValidatorAddr
	addr, err := c.RecoverOwner(header)
	if err != nil || addr != header.ValidatorAddr {
		return fmt.Errorf("ValidatorAddr err : %s > %s", addr.String(), header.ValidatorAddr.String())
	}

	// Ensure that the extra-data contains behavior data on checkpoint, but none otherwise
	number := header.Number.Uint64()
	checkpoint := number%c.pobConfig.Epoch == 0
	behaviorDataLen := len(header.Extra) - extraVanity - extraSeal
	if behaviorDataLen < 0 {
		behaviorDataLen = 0
	}
	if !checkpoint && behaviorDataLen != 0 {
		return errExtraValidators
	}

	// Verify the header's timestamp
	if !uncle {
		futureLimit := allowedFutureBlockTimeSeconds
		if chain.Config().IsStellarSpeed(header.Number) {
			futureLimit = allowedFutureBlockTimeSecondsStellar
		}
		if header.Time > uint64(unixNow+futureLimit) {
			return consensus.ErrFutureBlock
		}
	}
	// In StellarSpeed mode, allow same-second blocks ordered by AtomicTime
	if header.Time < parent.Time {
		return errOlderBlockTime
	}
	if header.Time == parent.Time && !chain.Config().IsStellarSpeed(header.Number) {
		return errOlderBlockTime
	}

	// Verify the block's difficulty (PoB uses in-turn / out-of-turn difficulty)
	if !c.fakeDiff {
		if header.Difficulty == nil || header.Difficulty.Sign() <= 0 {
			return fmt.Errorf("invalid difficulty: have %v, want positive", header.Difficulty)
		}
	}

	// Verify gas limits
	cap := uint64(0x7fffffffffffffff)
	if header.GasLimit > cap {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, cap)
	}
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("invalid gasUsed: have %d, gasLimit %d", header.GasUsed, header.GasLimit)
	}

	// Verify the block's gas usage and (if applicable) verify the base fee.
	if !chain.Config().IsLondon(header.Number) {
		if header.BaseFee != nil {
			return fmt.Errorf("invalid baseFee before fork: have %d, expected 'nil'", header.BaseFee)
		}
		if err := misc.VerifyGaslimit(parent.GasLimit, header.GasLimit); err != nil {
			return err
		}
	} else if err := misc.VerifyEip1559Header(chain.Config(), parent, header); err != nil {
		return err
	}

	// Verify that the block number is parent's +diff
	if new(big.Int).Sub(header.Number, parent.Number).Cmp(big.NewInt(diff)) != 0 {
		return consensus.ErrInvalidNumber
	}

	// In PoB, behavior proofs are verified separately; no PoW seal check needed.

	// Optional AtomicTime validation: if present, verify it is well-formed
	// and not unreasonably far from the header timestamp. Don't reject blocks
	// without AtomicTime for backward compatibility.
	if len(header.AtomicTime) > 0 {
		at, err := atomicClock.DecodeAtomicTimestamp(header.AtomicTime)
		if err != nil {
			return fmt.Errorf("invalid AtomicTime encoding: %v", err)
		}
		// AtomicTime seconds should be within 60s of header.Time
		diff := int64(at.Seconds) - int64(header.Time)
		if diff < 0 {
			diff = -diff
		}
		if diff > 60 {
			return fmt.Errorf("AtomicTime diverges from header timestamp by %d seconds", diff)
		}
	}

	return nil
}

// VerifyUncles implements consensus.Engine.
func (c *ProofOfBehavior) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) > 0 {
		return errors.New("uncles not allowed in PoB")
	}
	return nil
}

// VerifyUncleBehaviorProofs implements consensus.Engine.
func (c *ProofOfBehavior) VerifyUncleBehaviorProofs(chain consensus.ChainReader, block *types.Block) error {
	header := block.Header()
	for _, answer := range block.BehaviorProofUncles() {
		if err := c.verifyBehaviorProof(chain, answer, !chain.Config().IsShenzhen(header.Number)); err != nil {
			return err
		}
	}
	return nil
}

func (c *ProofOfBehavior) verifyBehaviorProof(chain consensus.ChainHeaderReader, answer *types.BehaviorProof, isBeforeUnclePowFix bool) error {
	var header *types.Header
	if isBeforeUnclePowFix {
		header = chain.GetHeaderByNumber(answer.Number.Uint64())
	} else {
		header = chain.GetHeader(answer.BlockHash, answer.Number.Uint64())
	}
	if header == nil {
		return fmt.Errorf("verifyBehaviorProof header is nil")
	}
	// In pure PoB, behavior proofs are validated by score-based rules, not PoW.
	return nil
}

// VerifyValidatorInfo implements consensus.Engine, verifying that the producer was
// selected by weighted behavior score.
func (c *ProofOfBehavior) VerifyValidatorInfo(chain consensus.ChainReader, block *types.Block) error {
	miner := block.Header().ValidatorAddr
	isVisual := block.Header().IsVisual()
	num := block.NumberU64()
	isProducer := chain.CheckIsProducerAccount(num, miner)

	if (isProducer && isVisual) || (!isProducer && !isVisual) {
		return fmt.Errorf("not visual not allow visual extra")
	}

	if !chain.CheckAcks(block) {
		return fmt.Errorf("acks not legal")
	}

	return nil
}

// snapshot retrieves the authorization snapshot at a given point in time.
func (c *ProofOfBehavior) snapshot(chain consensus.ChainHeaderReader, number uint64, hash common.Hash, parents []*types.Header) (*Snapshot, error) {
	var (
		headers []*types.Header
		snap    *Snapshot
	)
	for snap == nil {
		// If an in-memory snapshot was found, use that
		if s, ok := c.recents.Get(hash); ok {
			snap = s.(*Snapshot)
			break
		}
		// If an on-disk checkpoint snapshot can be found, use that
		if number%checkpointInterval == 0 {
			if s, err := loadSnapshot(c.pobConfig, c.signatures, c.db, hash); err == nil {
				log.Trace("Loaded voting snapshot from disk", "number", number, "hash", hash)
				snap = s
				break
			}
		}
		// If we're at the genesis, snapshot the initial state.
		if number == 0 {
			genesis := chain.GetHeaderByNumber(0)
			if genesis == nil {
				return nil, errUnknownBlock
			}
			// Extract initial validators from PobConfig
			validators := make([]common.Address, 0)
			for _, v := range c.pobConfig.ValidatorList {
				validators = append(validators, v.Owner)
			}
			snap = newSnapshot(c.pobConfig, c.signatures, 0, genesis.Hash(), validators)
			if err := snap.store(c.db); err != nil {
				return nil, err
			}
			log.Info("Stored genesis voting snapshot to disk")
			break
		}
		// No snapshot for this header, gather the header and move backward
		var header *types.Header
		if len(parents) > 0 {
			header = parents[len(parents)-1]
			if header.Hash() != hash || header.Number.Uint64() != number {
				return nil, consensus.ErrUnknownAncestor
			}
			parents = parents[:len(parents)-1]
		} else {
			header = chain.GetHeader(hash, number)
			if header == nil {
				return nil, consensus.ErrUnknownAncestor
			}
		}
		headers = append(headers, header)
		number, hash = number-1, header.ParentHash
	}
	// Previous snapshot found, apply any pending headers on top of it
	for i := 0; i < len(headers)/2; i++ {
		headers[i], headers[len(headers)-1-i] = headers[len(headers)-1-i], headers[i]
	}
	snap, err := snap.apply(headers)
	if err != nil {
		return nil, err
	}
	c.recents.Add(snap.Hash, snap)

	// If we've generated a new checkpoint snapshot, save to disk
	if snap.Number%checkpointInterval == 0 && len(headers) > 0 {
		if err = snap.store(c.db); err != nil {
			return nil, err
		}
		log.Trace("Stored voting snapshot to disk", "number", snap.Number, "hash", snap.Hash)
	}
	return snap, err
}

// Prepare implements consensus.Engine, preparing all the consensus fields of the
// header for running the transactions on top.
func (c *ProofOfBehavior) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	return nil
}

// accumulateRewards distributes rewards proportional to behavior scores.
func accumulateRewards(config *params.ChainConfig, statedb *state.StateDB, header *types.Header, powUncles []*types.BehaviorProof) {
	// Base reward to the PoB validator
	statedb.AddBalance(header.ValidatorAddr, new(big.Int).Set(BlockRewardPobValidator))
	// Rewards for PoW miners
	for _, answer := range header.BehaviorProofs {
		statedb.AddBalance(answer.Miner, new(big.Int).Set(BlockRewardPowMiner))
	}
	// Rewards for PoW uncle miners
	for _, answer := range powUncles {
		statedb.AddBalance(answer.Miner, new(big.Int).Set(BlockRewardPowMinerUncle))
	}
}

// PobFinalize runs post-transaction state modifications including behavior-score-weighted rewards.
func (c *ProofOfBehavior) PobFinalize(chain consensus.ChainHeaderReader, header *types.Header, statedb *state.StateDB, txs []*types.Transaction, powUncles []*types.BehaviorProof) {
	accumulateRewards(chain.Config(), statedb, header, powUncles)
	header.Root = statedb.IntermediateRoot(chain.Config().IsEIP158(header.Number))
}

// Finalize implements consensus.Engine.
func (c *ProofOfBehavior) Finalize(chain consensus.ChainHeaderReader, header *types.Header, statedb *state.StateDB, txs []*types.Transaction, uncles []*types.Header) {
	header.Root = statedb.IntermediateRoot(chain.Config().IsEIP158(header.Number))
}

// FinalizeAndAssemble implements consensus.Engine.
func (c *ProofOfBehavior) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, statedb *state.StateDB, txs []*types.Transaction, uncles []*types.BehaviorProof, receipts []*types.Receipt) (*types.Block, error) {
	c.Finalize(chain, header, statedb, txs, nil)
	return types.NewBlock(header, txs, nil, receipts, trie.NewStackTrie(nil)), nil
}

// Authorize injects a private key into the consensus engine to mint new blocks with.
func (c *ProofOfBehavior) Authorize(signer common.Address, signFn SignerFn) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.signer = signer
	c.signFn = signFn
}

// Seal implements consensus.Engine, signing the block with the validator's key.
func (c *ProofOfBehavior) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	header := block.Header()
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}

	// Sign the block
	sighash, err := c.signFn(accounts.Account{Address: c.signer}, accounts.MimetypeDataWithValidator, PobRLP(header))
	if err != nil {
		return err
	}
	block.SetValidatorSig(sighash)
	return nil
}

// AckSig signs a PoB acknowledgment.
func (c *ProofOfBehavior) AckSig(ack *types.Ack) ([]byte, error) {
	sighash, err := c.signFn(accounts.Account{Address: c.signer}, accounts.MimetypeDataWithValidator, PobAckRLP(ack))
	if err != nil {
		return nil, err
	}
	return sighash, nil
}

// CalcDifficulty is the difficulty adjustment algorithm.
// In PoB, difficulty reflects behavior score: higher score -> lower difficulty (in-turn).
func (c *ProofOfBehavior) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	return new(big.Int).Set(diffNoTurn)
}

func calcDifficulty(snap *Snapshot, validator common.Address) *big.Int {
	if snap.inturn(snap.Number+1, snap.Hash, validator) {
		return new(big.Int).Set(diffInTurn)
	}
	return new(big.Int).Set(diffNoTurn)
}

// SealHash returns the hash of a block prior to it being sealed.
func (c *ProofOfBehavior) SealHash(header *types.Header) common.Hash {
	return SealHash(header)
}

// Close implements consensus.Engine. It's a noop for pob.
func (c *ProofOfBehavior) Close() error {
	return nil
}

// APIs implements consensus.Engine, returning the user facing RPC API.
func (c *ProofOfBehavior) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return []rpc.API{{
		Namespace: "pob",
		Version:   "1.0",
		Service:   &API{chain: chain, pob: c},
		Public:    false,
	}}
}

// RecoverOwner recovers the signer address from the ValidatorSig.
// Supports both ECDSA (65-byte sig) and Dilithium (pubkey+sig) signatures.
func (c *ProofOfBehavior) RecoverOwner(header *types.Header) (common.Address, error) {
	sigLen := len(header.ValidatorSig)
	dilithiumSigLen := dilithium.PublicKeySize + dilithium.SignatureSize // 1312 + 2420 = 3732

	if sigLen == dilithiumSigLen {
		// Dilithium path: ValidatorSig = pubkey(1312) || signature(2420)
		pubBytes := header.ValidatorSig[:dilithium.PublicKeySize]
		sigBytes := header.ValidatorSig[dilithium.PublicKeySize:]
		pub, err := dilithium.UnmarshalPublicKey(pubBytes)
		if err != nil {
			return common.Address{}, fmt.Errorf("invalid Dilithium pubkey in ValidatorSig: %v", err)
		}
		msg := crypto.Keccak256(PobRLP(header))
		if !dilithium.Verify(pub, msg, sigBytes) {
			return common.Address{}, fmt.Errorf("invalid Dilithium signature in ValidatorSig")
		}
		return dilithium.PubkeyToAddress(pub), nil
	}

	// ECDSA path (default)
	pubkey, err := secp256k1.RecoverPubkey(crypto.Keccak256(PobRLP(header)), header.ValidatorSig)
	if err == nil {
		publicKey, err := crypto.UnmarshalPubkey(pubkey)
		if err == nil {
			return crypto.PubkeyToAddress(*publicKey), nil
		}
	}
	return common.Address{}, nil
}

// SealHash returns the hash of a block prior to it being sealed.
func SealHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewLegacyKeccak256()
	encodeSigHeader(hasher, header)
	hasher.(crypto.KeccakState).Read(hash[:])
	return hash
}

// PobRLP returns the rlp bytes which needs to be signed for the proof-of-behavior
// sealing. The RLP to sign consists of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
func PobRLP(header *types.Header) []byte {
	b := new(bytes.Buffer)
	encodeSigHeader(b, header)
	return b.Bytes()
}

func encodeSigHeader(w io.Writer, header *types.Header) {
	enc := []interface{}{
		header.ValidatorAddr,
		header.AckCountList,
		header.AcksHash,
		header.BehaviorProofs,
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.Extra[:len(header.Extra)-1], // Yes, this will panic if extra is too short
	}
	if header.BaseFee != nil {
		enc = append(enc, header.BaseFee)
	}
	if err := rlp.Encode(w, enc); err != nil {
		panic("can't encode: " + err.Error())
	}
}

// PobAckRLP returns the RLP bytes for signing a Ack.
func PobAckRLP(ack *types.Ack) []byte {
	b := new(bytes.Buffer)
	encodeSigAck(b, ack)
	return b.Bytes()
}

func encodeSigAck(w io.Writer, ack *types.Ack) {
	enc := []interface{}{
		ack.EpochPosition,
		ack.Number,
		ack.BlockHash,
		ack.AckType,
	}
	if err := rlp.Encode(w, enc); err != nil {
		panic("can't encode: " + err.Error())
	}
}

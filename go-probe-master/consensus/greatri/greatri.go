// Copyright 2017 The go-probeum Authors
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

// Package greatri implements the proof-of-authority consensus engine.
package greatri

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/probeum/go-probeum/consensus/probeash"
	"github.com/probeum/go-probeum/crypto/secp256k1"

	"github.com/probeum/go-probeum/log"
	"io"
	"math/big"
	"runtime"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/probeum/go-probeum/accounts"
	"github.com/probeum/go-probeum/common"
	"github.com/probeum/go-probeum/common/hexutil"
	"github.com/probeum/go-probeum/consensus"
	"github.com/probeum/go-probeum/consensus/misc"
	"github.com/probeum/go-probeum/core/state"
	"github.com/probeum/go-probeum/core/types"
	"github.com/probeum/go-probeum/crypto"
	"github.com/probeum/go-probeum/crypto/dilithium"
	"github.com/probeum/go-probeum/params"
	"github.com/probeum/go-probeum/probedb"
	"github.com/probeum/go-probeum/rlp"
	"github.com/probeum/go-probeum/rpc"
	"github.com/probeum/go-probeum/trie"
	"golang.org/x/crypto/sha3"
)

const (
	checkpointInterval = 1024 // Number of blocks after which to save the vote snapshot to the database
	inmemorySnapshots  = 128  // Number of recent vote snapshots to keep in memory
	inmemorySignatures = 4096 // Number of recent block signatures to keep in memory
	maxUnclePowAnswer  = 5
	wiggleTime         = 500 * time.Millisecond // Random delay (per signer) to allow concurrent signers
)

// Greatri proof-of-authority protocol constants.
var (
	BlockRewardPowMiner      = big.NewInt(3e+18)
	BlockRewardDposSigner    = big.NewInt(1e+18)
	BlockRewardPowMinerUncle = big.NewInt(5e+17)

	epochLength = uint64(30000) // Default number of blocks after which to checkpoint and reset the pending votes

	extraVanity = 32                     // Fixed number of extra-data prefix bytes reserved for signer vanity
	extraSeal   = crypto.SignatureLength // Fixed number of extra-data suffix bytes reserved for signer seal

	nonceAuthVote = hexutil.MustDecode("0xffffffffffffffff") // Magic nonce number to vote on adding a new signer
	nonceDropVote = hexutil.MustDecode("0x0000000000000000") // Magic nonce number to vote on removing a signer.

	uncleHash = types.CalcPowAnswerUncleHash(nil) // Always Keccak256(RLP([])) as uncles are meaningless outside of PoW.

	diffInTurn                    = big.NewInt(2) // Block difficulty for in-turn signatures
	diffNoTurn                    = big.NewInt(1) // Block difficulty for out-of-turn signatures
	allowedFutureBlockTimeSeconds = int64(15)     // Max seconds from current time allowed for blocks, before they're considered future blocks
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	// errUnknownBlock is returned when the list of signers is requested for a block
	// that is not part of the local blockchain.
	errUnknownBlock = errors.New("unknown block")

	// errInvalidCheckpointBeneficiary is returned if a checkpoint/epoch transition
	// block has a beneficiary set to non-zeroes.
	errInvalidCheckpointBeneficiary = errors.New("beneficiary in checkpoint block non-zero")

	// errInvalidVote is returned if a nonce value is somprobeing else that the two
	// allowed constants of 0x00..0 or 0xff..f.
	errInvalidVote = errors.New("vote nonce not 0x00..0 or 0xff..f")

	// errInvalidCheckpointVote is returned if a checkpoint/epoch transition block
	// has a vote nonce set to non-zeroes.
	errInvalidCheckpointVote = errors.New("vote nonce in checkpoint block non-zero")

	// errMissingVanity is returned if a block's extra-data section is shorter than
	// 32 bytes, which is required to store the signer vanity.
	errMissingVanity = errors.New("extra-data 32 byte vanity prefix missing")

	// errMissingSignature is returned if a block's extra-data section doesn't seem
	// to contain a 65 byte secp256k1 signature.
	errMissingSignature = errors.New("extra-data 65 byte signature suffix missing")

	// errExtraSigners is returned if non-checkpoint block contain signer data in
	// their extra-data fields.
	errExtraSigners = errors.New("non-checkpoint block contains extra signer list")

	// errInvalidCheckpointSigners is returned if a checkpoint block contains an
	// invalid list of signers (i.e. non divisible by 20 bytes).
	errInvalidCheckpointSigners = errors.New("invalid signer list on checkpoint block")

	// errMismatchingCheckpointSigners is returned if a checkpoint block contains a
	// list of signers different than the one the local node calculated.
	errMismatchingCheckpointSigners = errors.New("mismatching signer list on checkpoint block")

	// errInvalidMixDigest is returned if a block's mix digest is non-zero.
	errInvalidMixDigest = errors.New("non-zero mix digest")

	// errInvalidUncleHash is returned if a block contains an non-empty uncle list.
	errInvalidUncleHash = errors.New("non empty uncle hash")

	// errInvalidDifficulty is returned if the difficulty of a block neither 1 or 2.
	errInvalidDifficulty = errors.New("invalid difficulty")

	// errWrongDifficulty is returned if the difficulty of a block doesn't match the
	// turn of the signer.
	errWrongDifficulty = errors.New("wrong difficulty")

	// errInvalidTimestamp is returned if the timestamp of a block is lower than
	// the previous block's timestamp + the minimum block period.
	errInvalidTimestamp = errors.New("invalid timestamp")

	// errInvalidVotingChain is returned if an authorization list is attempted to
	// be modified via out-of-range or non-contiguous headers.
	errInvalidVotingChain = errors.New("invalid voting chain")

	// errUnauthorizedSigner is returned if a header is signed by a non-authorized entity.
	errUnauthorizedSigner = errors.New("unauthorized signer")

	// errRecentlySigned is returned if a header is signed by an authorized entity
	// that already signed a header recently, thus is temporarily not allowed to.
	errRecentlySigned = errors.New("recently signed")
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

// Config are the configuration parameters of the probeash.
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

	// When set, notifications sent by the remote sealer will
	// be block header JSON objects instead of work package arrays.
	NotifyFull bool

	Log log.Logger `toml:"-"`
}

// SignerFn hashes and signs the data to be signed by a backing account.
type SignerFn func(signer accounts.Account, mimeType string, message []byte) ([]byte, error)

// ecrecover extracts the Probeum account address from a signed header.
func ecrecover(header *types.Header, sigcache *lru.ARCCache) (common.Address, error) {
	// If the signature's already cached, return that
	hash := header.Hash()
	if address, known := sigcache.Get(hash); known {
		return address.(common.Address), nil
	}
	// Retrieve the signature from the header extra-data
	if len(header.Extra) < extraSeal {
		return common.Address{}, errMissingSignature
	}
	signature := header.Extra[len(header.Extra)-extraSeal:]

	// Recover the public key and the Probeum address
	pubkey, err := crypto.Ecrecover(SealHash(header).Bytes(), signature)
	if err != nil {
		return common.Address{}, err
	}
	var signer common.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])

	sigcache.Add(hash, signer)
	return signer, nil
}

// Greatri is the proof-of-authority consensus engine proposed to support the
// Probeum testnet following the Ropsten attacks.
type Greatri struct {
	dposConfig  *params.DposConfig // Consensus engine configuration parameters
	chainConfig *params.ChainConfig
	config      Config           //
	db          probedb.Database // Database to store and retrieve snapshot checkpoints
	powEngine   consensus.Engine

	recents    *lru.ARCCache // Snapshots for recent block to speed up reorgs
	signatures *lru.ARCCache // Signatures of recent blocks to speed up mining

	proposals map[common.Address]bool // Current list of proposals we are pushing

	signer common.Address // Probeum address of the signing key
	signFn SignerFn       // Signer function to authorize hashes with
	lock   sync.RWMutex   // Protects the signer fields

	// The fields below are for testing only
	fakeDiff bool // Skip difficulty verifications
}

// New creates a Greatri proof-of-authority consensus engine with the initial
// signers set to the ones provided by the user.
func New(config *params.DposConfig, db probedb.Database, powEngine consensus.Engine, chainConfig *params.ChainConfig) *Greatri {
	// Set any missing consensus parameters to their defaults
	conf := *config
	if conf.Epoch == 0 {
		conf.Epoch = epochLength
	}
	// Allocate the snapshot caches and create the engine
	recents, _ := lru.NewARC(inmemorySnapshots)
	signatures, _ := lru.NewARC(inmemorySignatures)

	return &Greatri{
		dposConfig:  &conf,
		chainConfig: chainConfig,
		db:          db,
		powEngine:   powEngine,
		recents:     recents,
		signatures:  signatures,
		proposals:   make(map[common.Address]bool),
	}
}

// Author implements consensus.Engine, returning the Probeum address recovered
// from the signature in the header's extra-data section.
func (c *Greatri) Author(header *types.Header) (common.Address, error) {
	//return ecrecover(header, c.signatures)
	return header.DposSigAddr, nil
}

// VerifyHeader checks whprobeer a header conforms to the consensus rules.
func (c *Greatri) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header, seal bool) error {
	// currently not support fake mode
	if c.config.PowMode == ModeFullFake {
		return nil
	}
	// Short circuit if the header is known, or its parent not
	number := header.Number.Uint64()
	if chain.GetHeader(header.Hash(), number) != nil {
		return nil
	}
	parent, err, diff := c.FindRealParentHeader(chain, header, nil, -1)
	if err != nil {
		return err
	}
	// Sanity checks passed, do a proper verification
	return c.verifyHeader(chain, header, parent, false, seal, time.Now().Unix(), diff)
}

func (c *Greatri) FindRealParentHeader(chain consensus.ChainHeaderReader, header *types.Header, headers []*types.Header, index int) (*types.Header, error, int64) {
	var parent = header
	var diff int64 = 1
	for {
		if index > 0 && headers != nil {
			parent = headers[index-1]
			if parent.Hash() != headers[index].ParentHash || new(big.Int).Sub(headers[index].Number, parent.Number).Cmp(common.Big1) != 0 {
				log.Debug("parent hash not equal : ", "num:", parent.Number.Uint64(), "diff:", new(big.Int).Sub(headers[index].Number, parent.Number), "parent:", parent.Hash().String(), "next:", headers[index].ParentHash.String())
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
			log.Error("", "consensus.ErrUnknownAncestor: ", header.Difficulty.String())
			return nil, consensus.ErrUnknownAncestor, diff
		}
		if !parent.IsVisual() {
			return parent, nil, diff
		}
		log.Debug("this is a visual block ", "num:", parent.Number.String())
		diff++

	}
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers. The
// method returns a quit channel to abort the operations and a results channel to
// retrieve the async verifications (the order is that of the input slice).
func (c *Greatri) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	// If we're running a full engine faking, accept any input as valid
	if c.config.PowMode == ModeFullFake || len(headers) == 0 {
		abort, results := make(chan struct{}), make(chan error, len(headers))
		for i := 0; i < len(headers); i++ {
			results <- nil
		}
		return abort, results
	}

	// Spawn as many workers as allowed threads
	workers := runtime.GOMAXPROCS(0)
	if len(headers) < workers {
		workers = len(headers)
	}

	// Create a task channel and spawn the verifiers
	var (
		inputs  = make(chan int)
		done    = make(chan int, workers)
		errors  = make([]error, len(headers))
		abort   = make(chan struct{})
		unixNow = time.Now().Unix()
	)
	for i := 0; i < workers; i++ {
		go func() {
			for index := range inputs {
				errors[index] = c.verifyHeaderWorker(chain, headers, seals, index, unixNow)
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
					// Reached end of headers. Stop sending to workers.
					inputs = nil
				}
			case index := <-done:
				for checked[index] = true; checked[out]; out++ {
					errorsOut <- errors[out]
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

func (c *Greatri) verifyHeaderWorker(chain consensus.ChainHeaderReader, headers []*types.Header, seals []bool, index int, unixNow int64) error {
	parent, err, diff := c.FindRealParentHeader(chain, headers[index], headers, index)
	if err != nil {
		return err
	}
	return c.verifyHeader(chain, headers[index], parent, false, seals[index], unixNow, diff)
}

// verifyHeader checks whprobeer a header conforms to the consensus rules.The
// caller may optionally pass in a batch of parents (ascending order) to avoid
// looking those up from the database. This is useful for concurrently verifying
// a batch of new headers.
func (c *Greatri) verifyHeader(chain consensus.ChainHeaderReader, header, parent *types.Header, uncle bool, seal bool, unixNow int64, diff int64) error {
	log.Trace("enter verifyHeader", "block number", header.Number, "seal", seal, "dposign", common.Bytes2Hex(header.DposSig), "ackHash", header.DposAcksHash.String())
	//return nil
	// Ensure that the header's extra-data section is of a reasonable size

	addr, err := c.RecoverOwner(header)
	if err != nil || addr != header.DposSigAddr {
		return fmt.Errorf("DposSigAddr err : %s > %s", addr.String(), header.DposSigAddr.String())
	}

	log.Trace("enter verifyHeader", "block number", header.Number, "addr", addr.String(), "dposign", header.DposSigAddr.String(), "dposign", common.Bytes2Hex(header.DposSig), "ackHash", header.DposAcksHash.String())

	if uint64(len(header.Extra)) > params.MaximumExtraDataSize {
		return fmt.Errorf("extra-data too long: %d > %d", len(header.Extra), params.MaximumExtraDataSize)
	}

	// Verify the header's timestamp
	if !uncle {
		if header.Time > uint64(unixNow+allowedFutureBlockTimeSeconds) {
			return consensus.ErrFutureBlock
		}
	}
	if header.Time <= parent.Time {
		return errOlderBlockTime
	}
	// Verify the block's difficulty based on its timestamp and parent's difficulty
	expected := probeash.CalcDifficulty(chain.Config(), header.Time, parent)

	if expected.Cmp(header.Difficulty) != 0 {
		return fmt.Errorf("invalid difficulty: have %v, want %v", header.Difficulty, expected)
	}
	// Verify that the gas limit is <= 2^63-1
	cap := uint64(0x7fffffffffffffff)
	if header.GasLimit > cap {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, cap)
	}
	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("invalid gasUsed: have %d, gasLimit %d", header.GasUsed, header.GasLimit)
	}
	// Verify the block's gas usage and (if applicable) verify the base fee.
	if !chain.Config().IsLondon(header.Number) {
		// Verify BaseFee not present before EIP-1559 fork.
		if header.BaseFee != nil {
			return fmt.Errorf("invalid baseFee before fork: have %d, expected 'nil'", header.BaseFee)
		}
		err := misc.VerifyGaslimit(parent.GasLimit, header.GasLimit)
		if err != nil {
			log.Info("", err)
			return err
		}
	} else if err := misc.VerifyEip1559Header(chain.Config(), parent, header); err != nil {
		// Verify the header's EIP-1559 attributes.
		return err
	}
	// Verify that the block number is parent's +1
	if new(big.Int).Sub(header.Number, parent.Number).Cmp(big.NewInt(diff)) != 0 {
		return consensus.ErrInvalidNumber
	}
	// Verify the engine specific seal securing the block
	if seal {
		pow, ok := c.powEngine.(*probeash.Probeash)
		if !ok {
			log.Warn("DispatchPowAnswer err! pow is not a pow engine")
		}
		for _, answer := range header.PowAnswers {
			//baseBlock := c.
			err := pow.PowVerifySeal(chain, parent, false, answer)
			if err != nil {
				log.Debug("PowVerifySeal failed", "block number", header.Number, "answer", answer)
				return err
			}
		}
		log.Debug("PowVerifySeal pass", "block number", header.Number)
	}
	// If all checks passed, validate any special fields for hard forks
	if err := misc.VerifyDAOHeaderExtraData(chain.Config(), header); err != nil {
		return err
	}
	if err := misc.VerifyForkHashes(chain.Config(), header, uncle); err != nil {
		return err
	}
	return nil
}

// verifyCascadingFields verifies all the header fields that are not standalone,
// rather depend on a batch of previous headers. The caller may optionally pass
// in a batch of parents (ascending order) to avoid looking those up from the
// database. This is useful for concurrently verifying a batch of new headers.
func (c *Greatri) verifyCascadingFields(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header) error {
	// The genesis block is the always valid dead-end
	number := header.Number.Uint64()
	if number == 0 {
		return nil
	}
	// Ensure that the block's timestamp isn't too close to its parent
	var parent *types.Header
	if len(parents) > 0 {
		parent = parents[len(parents)-1]
	} else {
		parent = chain.GetHeader(header.ParentHash, number-1)
	}
	if parent == nil || parent.Number.Uint64() != number-1 || parent.Hash() != header.ParentHash {
		return consensus.ErrUnknownAncestor
	}
	if parent.Time+c.dposConfig.Period > header.Time {
		return errInvalidTimestamp
	}
	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("invalid gasUsed: have %d, gasLimit %d", header.GasUsed, header.GasLimit)
	}
	if !chain.Config().IsLondon(header.Number) {
		// Verify BaseFee not present before EIP-1559 fork.
		if header.BaseFee != nil {
			return fmt.Errorf("invalid baseFee before fork: have %d, want <nil>", header.BaseFee)
		}
		if err := misc.VerifyGaslimit(parent.GasLimit, header.GasLimit); err != nil {
			return err
		}
	} else if err := misc.VerifyEip1559Header(chain.Config(), parent, header); err != nil {
		// Verify the header's EIP-1559 attributes.
		return err
	}
	//// Retrieve the snapshot needed to verify this header and cache it
	//snap, err := c.snapshot(chain, number-1, header.ParentHash, parents)
	//if err != nil {
	//	return err
	//}
	//// If the block is a checkpoint block, verify the signer list
	//if number%c.config.Epoch == 0 {
	//	signers := make([]byte, len(snap.Signers)*common.AddressLength)
	//	for i, signer := range snap.signers() {
	//		copy(signers[i*common.AddressLength:], signer[:])
	//	}
	//	extraSuffix := len(header.Extra) - extraSeal
	//	if !bytes.Equal(header.Extra[extraVanity:extraSuffix], signers) {
	//		return errMismatchingCheckpointSigners
	//	}
	//}
	// All basic checks passed, verify the seal and return
	return c.verifySeal(chain, header, parents)
}

//// snapshot retrieves the authorization snapshot at a given point in time.
func (c *Greatri) snapshot(chain consensus.ChainHeaderReader, number uint64, hash common.Hash, parents []*types.Header) (*Snapshot, error) {
	//	// Search for a snapshot in memory or on disk for checkpoints
	//	var (
	//		headers []*types.Header
	//		snap    *Snapshot
	//	)
	//	for snap == nil {
	//		// If an in-memory snapshot was found, use that
	//		if s, ok := c.recents.Get(hash); ok {
	//			snap = s.(*Snapshot)
	//			break
	//		}
	//		// If an on-disk checkpoint snapshot can be found, use that
	//		if number%checkpointInterval == 0 {
	//			if s, err := loadSnapshot(c.config, c.signatures, c.db, hash); err == nil {
	//				log.Trace("Loaded voting snapshot from disk", "number", number, "hash", hash)
	//				snap = s
	//				break
	//			}
	//		}
	//		// If we're at the genesis, snapshot the initial state. Alternatively if we're
	//		// at a checkpoint block without a parent (light client CHT), or we have piled
	//		// up more headers than allowed to be reorged (chain reinit from a freezer),
	//		// consider the checkpoint trusted and snapshot it.
	//		if number == 0 || (number%c.config.Epoch == 0 && (len(headers) > params.FullImmutabilityThreshold || chain.GetHeaderByNumber(number-1) == nil)) {
	//			checkpoint := chain.GetHeaderByNumber(number)
	//			if checkpoint != nil {
	//				hash := checkpoint.Hash()
	//
	//				signers := make([]common.Address, (len(checkpoint.Extra)-extraVanity-extraSeal)/common.AddressLength)
	//				for i := 0; i < len(signers); i++ {
	//					copy(signers[i][:], checkpoint.Extra[extraVanity+i*common.AddressLength:])
	//				}
	//				snap = newSnapshot(c.config, c.signatures, number, hash, signers)
	//				if err := snap.store(c.db); err != nil {
	//					return nil, err
	//				}
	//				log.Info("Stored checkpoint snapshot to disk", "number", number, "hash", hash)
	//				break
	//			}
	//		}
	//		// No snapshot for this header, gather the header and move backward
	//		var header *types.Header
	//		if len(parents) > 0 {
	//			// If we have explicit parents, pick from there (enforced)
	//			header = parents[len(parents)-1]
	//			if header.Hash() != hash || header.Number.Uint64() != number {
	//				return nil, consensus.ErrUnknownAncestor
	//			}
	//			parents = parents[:len(parents)-1]
	//		} else {
	//			// No explicit parents (or no more left), reach out to the database
	//			header = chain.GetHeader(hash, number)
	//			if header == nil {
	//				return nil, consensus.ErrUnknownAncestor
	//			}
	//		}
	//		headers = append(headers, header)
	//		number, hash = number-1, header.ParentHash
	//	}
	//	// Previous snapshot found, apply any pending headers on top of it
	//	for i := 0; i < len(headers)/2; i++ {
	//		headers[i], headers[len(headers)-1-i] = headers[len(headers)-1-i], headers[i]
	//	}
	//	snap, err := snap.apply(headers)
	//	if err != nil {
	//		return nil, err
	//	}
	//	c.recents.Add(snap.Hash, snap)
	//
	//	// If we've generated a new checkpoint snapshot, save to disk
	//	if snap.Number%checkpointInterval == 0 && len(headers) > 0 {
	//		if err = snap.store(c.db); err != nil {
	//			return nil, err
	//		}
	//		log.Trace("Stored voting snapshot to disk", "number", snap.Number, "hash", snap.Hash)
	//	}
	return new(Snapshot), nil
}

// VerifyUncles implements consensus.Engine, always returning an error for any
// uncles as this consensus mechanism doesn't permit uncles.
func (c *Greatri) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) > 0 {
		return errors.New("uncles not allowed")
	}
	return nil
}

func (greatri *Greatri) IsBeforeUnclePowFix(block *types.Block) bool {
	if block == nil {
		return false
	}
	myChainId := greatri.chainConfig.ChainID.Uint64()
	chain := myChainId == params.FrcMainChainConfig.ChainID.Uint64() || myChainId == params.FrcTestChainConfig.ChainID.Uint64() || myChainId == params.ProbeTestChainConfig.ChainID.Uint64()
	isShenZhen := !greatri.chainConfig.IsShenzhen(block.Number())
	return chain && isShenZhen
}

// VerifyUnclePowAnswers verifies that the given block's UnclePowAnswers  conform to the consensus
func (greatri *Greatri) VerifyUnclePowAnswers(chain consensus.ChainReader, block *types.Block) error {
	powAnswers := block.PowAnswerUncles()
	isVisual := block.Header().IsVisual()
	if (isVisual && len(powAnswers) > 0) || (isVisual && len(powAnswers) > 64) {
		log.Debug("VerifyUnclePowAnswers fail", "isVisual", isVisual, "len(powAnswers)", len(powAnswers))
		return fmt.Errorf("len(powAnswers) error")
	}

	realParentHeader, err, _ := greatri.FindRealParentHeader(chain, block.Header(), nil, -1)
	if err != nil {
		return err
	}

	used := make(map[common.Hash]*types.PowAnswer)

	blockNUm := block.NumberU64()
	if blockNUm > 1 {
		uncleHeader := block.Header()
		for {
			uncleBlock := chain.GetBlock(uncleHeader.ParentHash, uncleHeader.Number.Uint64()-1)
			if uncleBlock == nil {
				return fmt.Errorf("uncleblock not found")
			}
			for _, answer := range uncleBlock.PowAnswers() {
				if answer != nil {
					used[answer.Id()] = answer
				}
			}
			for _, answer := range uncleBlock.PowAnswerUncles() {
				if answer != nil {
					used[answer.Id()] = answer
				}
			}
			uncleHeader = uncleBlock.Header()

			if uncleBlock.NumberU64() == 0 || (realParentHeader.Number.Uint64() > uncleHeader.Number.Uint64() && realParentHeader.Number.Uint64()-uncleHeader.Number.Uint64() >= maxUnclePowAnswer) {
				break
			}

		}

	}

	isBeforeUnclePowFix := greatri.IsBeforeUnclePowFix(block)
	for _, answer := range block.PowAnswers() {
		if !isBeforeUnclePowFix {
			if used[answer.Id()] != nil {
				return fmt.Errorf("powAnswer used")
			}
			used[answer.Id()] = answer
		}

	}

	for _, answer := range powAnswers {
		differ := int(realParentHeader.Number.Uint64() - answer.Number.Uint64())
		minDiffer := 1
		if isBeforeUnclePowFix {
			minDiffer = 0
		}
		if differ < minDiffer || differ > 5 {
			log.Debug("VerifyUnclePowAnswers answer is too far", "realParentHeader.Number  : ", realParentHeader.Number, "answer.Number", answer.Number)
			return fmt.Errorf("answer is too far ")
		}

		if !isBeforeUnclePowFix && used[answer.Id()] != nil {
			return fmt.Errorf("uncle powAnswer used")
		}
		verify := greatri.verifyPowAnswer(chain, answer, isBeforeUnclePowFix)
		if verify != nil {
			log.Error("VerifyUnclePowAnswers", "fail  : ", block.NumberU64())
			return verify
		}
		used[answer.Id()] = answer
	}

	return nil
}

// VerifyDposInfo verifies that the given block's dposInfo  conform to the consensus
func (greatri *Greatri) VerifyDposInfo(chain consensus.ChainReader, block *types.Block) error {

	miner := block.Header().DposSigAddr
	isVisual := block.Header().IsVisual()
	num := block.NumberU64()
	isProducer := chain.CheckIsProducerAccount(num, miner)

	if (isProducer && isVisual) || (!isProducer && !isVisual) {
		log.Debug("not visual  not allow  visual extra ", "isProducer:", isProducer, " visual:", isVisual, "num", num)
		//return nil
		return fmt.Errorf(" not visual  not allow  visual extra")
	}

	if !chain.CheckAcks(block) {
		log.Debug("acks not legal  ", "isProducer:", isProducer, " visual:", isVisual, "num", num)
		//return nil
		return fmt.Errorf(" acks not legal")
	}

	return nil
}

func (c *Greatri) verifyPowAnswer(chain consensus.ChainHeaderReader, answer *types.PowAnswer, isBeforeUnclePowFix bool) error {
	var header *types.Header
	if isBeforeUnclePowFix {
		header = chain.GetHeaderByNumber(answer.Number.Uint64())
	} else {
		header = chain.GetHeader(answer.BlockHash, answer.Number.Uint64())
	}
	if header == nil {
		return fmt.Errorf("verifyPowAnswer header is nil ")
	}
	pow, ok := c.powEngine.(*probeash.Probeash)
	if !ok {
		return fmt.Errorf("DispatchPowAnswer err! pow is not a pow engine")
	}
	err := pow.PowVerifySeal(chain, header, false, answer)
	if err != nil {
		log.Debug("PowVerifySeal failed", "block number", header.Number, "answer", answer)
		return err
	}
	return nil
}

// verifySeal checks whprobeer the signature contained in the header satisfies the
// consensus protocol requirements. The method accepts an optional list of parent
// headers that aren't yet part of the local blockchain to generate the snapshots
// from.
func (c *Greatri) verifySeal(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header) error {
	// Verifying the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}
	//// Retrieve the snapshot needed to verify this header and cache it
	//snap, err := c.snapshot(chain, number-1, header.ParentHash, parents)
	//if err != nil {
	//	return err
	//}
	//
	//// Resolve the authorization key and check against signers
	//signer, err := ecrecover(header, c.signatures)
	//if err != nil {
	//	return err
	//}
	//if _, ok := snap.Signers[signer]; !ok {
	//	return errUnauthorizedSigner
	//}
	//for seen, recent := range snap.Recents {
	//	if recent == signer {
	//		// Signer is among recents, only fail if the current block doesn't shift it out
	//		if limit := uint64(len(snap.Signers)/2 + 1); seen > number-limit {
	//			return errRecentlySigned
	//		}
	//	}
	//}
	//// Ensure that the difficulty corresponds to the turn-ness of the signer
	//if !c.fakeDiff {
	//	inturn := snap.inturn(header.Number.Uint64(), signer)
	//	if inturn && header.Difficulty.Cmp(diffInTurn) != 0 {
	//		return errWrongDifficulty
	//	}
	//	if !inturn && header.Difficulty.Cmp(diffNoTurn) != 0 {
	//		return errWrongDifficulty
	//	}
	//}
	return nil
}
// RecoverOwner recovers the signer address from the DposSig.
// Supports both ECDSA (65-byte sig) and Dilithium (pubkey+sig) signatures.
func (c *Greatri) RecoverOwner(header *types.Header) (common.Address, error) {
	sigLen := len(header.DposSig)
	dilithiumSigLen := dilithium.PublicKeySize + dilithium.SignatureSize // 1312 + 2420 = 3732

	if sigLen == dilithiumSigLen {
		// Dilithium path: DposSig = pubkey(1312) || signature(2420)
		pubBytes := header.DposSig[:dilithium.PublicKeySize]
		sigBytes := header.DposSig[dilithium.PublicKeySize:]
		pub, err := dilithium.UnmarshalPublicKey(pubBytes)
		if err != nil {
			return common.Address{}, fmt.Errorf("invalid Dilithium pubkey in DposSig: %v", err)
		}
		msg := crypto.Keccak256(GreatriRLP(header))
		if !dilithium.Verify(pub, msg, sigBytes) {
			return common.Address{}, fmt.Errorf("invalid Dilithium signature in DposSig")
		}
		return dilithium.PubkeyToAddress(pub), nil
	}

	// ECDSA path (default)
	pubkey, err := secp256k1.RecoverPubkey(crypto.Keccak256(GreatriRLP(header)), header.DposSig)
	if err == nil {
		publicKey, err := crypto.UnmarshalPubkey(pubkey)
		if err == nil {
			return crypto.PubkeyToAddress(*publicKey), nil
		}
	}
	return common.Address{}, nil
}

func (c *Greatri) verifyHeaderSeal(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header) error {
	//account := chain.GetSealDposAccount(blockNumber)
	//
	_, err := c.RecoverOwner(header)
	//if err == nil {
	//	for _, account := range accounts {
	//		if bytes.Compare(account.Owner.Bytes(), owner.Bytes()) == 0 {
	//			if dposAck.AckType == types.AckTypeOppose {
	//				return true
	//			} else {
	//				curHash := bc.GetHeaderByNumber(dposAck.Number.Uint64()).Hash()
	//				if curHash == dposAck.BlockHash {
	//					return true
	//				} else {
	//					log.Debug("CheckDposAck Fail, hash not match", "signer", owner, "err", err)
	//				}
	//			}
	//		}
	//	}
	//	log.Debug("CheckDposAck Fail, singer is not the dpos node", "signer", owner, "err", err)
	//}
	return err
}

// Prepare implements consensus.Engine, preparing all the consensus fields of the
// header for running the transactions on top.
func (c *Greatri) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	return nil
}

func accumulateRewards(config *params.ChainConfig, state *state.StateDB, header *types.Header, powUncles []*types.PowAnswer) {
	//log.Debug("enter accumulateRewards")
	state.AddBalance(header.DposSigAddr, new(big.Int).Set(BlockRewardDposSigner))
	for _, answer := range header.PowAnswers {
		state.AddBalance(answer.Miner, new(big.Int).Set(BlockRewardPowMiner))
	}
	for _, answer := range powUncles {
		state.AddBalance(answer.Miner, new(big.Int).Set(BlockRewardPowMinerUncle))
	}
}

// Finalize implements consensus.Engine, ensuring no uncles are set, nor block
// rewards given.
func (c *Greatri) DposFinalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, powUncles []*types.PowAnswer) {
	accumulateRewards(chain.Config(), state, header, powUncles)
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
}

func (c *Greatri) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header) {
	// No block rewards in PoA, so the state remains as is and uncles are dropped
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	//header.UncleHash = types.CalcUncleHash(nil)
}

// FinalizeAndAssemble implements consensus.Engine, ensuring no uncles are set,
// nor block rewards given, and returns the final block.
func (c *Greatri) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.PowAnswer, receipts []*types.Receipt) (*types.Block, error) {
	// Finalize block
	c.Finalize(chain, header, state, txs, nil)

	// Assemble and return the final block for sealing
	return types.NewBlock(header, txs, nil, receipts, trie.NewStackTrie(nil)), nil
}

// Authorize injects a private key into the consensus engine to mint new blocks
// with.
func (c *Greatri) Authorize(signer common.Address, signFn SignerFn) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.signer = signer
	c.signFn = signFn
}

func (c *Greatri) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	header := block.Header()
	number := header.Number.Uint64()
	// Sealing the genesis block is not supported
	if number == 0 {
		return errUnknownBlock
	}

	// Sign all the things!
	sighash, err := c.signFn(accounts.Account{Address: c.signer}, accounts.MimetypeDataWithValidator, GreatriRLP(header))
	if err != nil {
		return err
	}
	block.SetDposSig(sighash)
	return nil
}

func (c *Greatri) DposAckSig(ack *types.DposAck) ([]byte, error) {
	sighash, err := c.signFn(accounts.Account{Address: c.signer}, accounts.MimetypeDataWithValidator, GreatriDposAckRLP(ack))
	if err != nil {
		return nil, err
	}
	return sighash, nil
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficulty
// that a new block should have:
// * DIFF_NOTURN(2) if BLOCK_NUMBER % SIGNER_COUNT != SIGNER_INDEX
// * DIFF_INTURN(1) if BLOCK_NUMBER % SIGNER_COUNT == SIGNER_INDEX
func (c *Greatri) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	//snap, err := c.snapshot(chain, parent.Number.Uint64(), parent.Hash(), nil)
	//if err != nil {
	//	return nil
	//}
	//return calcDifficulty(snap, c.signer)
	return new(big.Int).Set(diffNoTurn)
}

func calcDifficulty(snap *Snapshot, signer common.Address) *big.Int {
	if snap.inturn(snap.Number+1, signer) {
		return new(big.Int).Set(diffInTurn)
	}
	return new(big.Int).Set(diffNoTurn)
}

// SealHash returns the hash of a block prior to it being sealed.
func (c *Greatri) SealHash(header *types.Header) common.Hash {
	return SealHash(header)
}

// Close implements consensus.Engine. It's a noop for greatri as there are no background threads.
func (c *Greatri) Close() error {
	return nil
}

// APIs implements consensus.Engine, returning the user facing RPC API to allow
// controlling the signer voting.
func (c *Greatri) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return []rpc.API{{
		Namespace: "greatri",
		Version:   "1.0",
		Service:   &API{chain: chain, greatri: c},
		Public:    false,
	}}
}

// SealHash returns the hash of a block prior to it being sealed.
func SealHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewLegacyKeccak256()
	encodeSigHeader(hasher, header)
	hasher.(crypto.KeccakState).Read(hash[:])
	return hash
}

// GreatriRLP returns the rlp bytes which needs to be signed for the proof-of-authority
// sealing. The RLP to sign consists of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
//
// Note, the method requires the extra data to be at least 65 bytes, otherwise it
// panics. This is done to avoid accidentally using both forms (signature present
// or not), which could be abused to produce different hashes for the same header.
func GreatriRLP(header *types.Header) []byte {
	b := new(bytes.Buffer)
	encodeSigHeader(b, header)
	return b.Bytes()
}

func encodeSigHeader(w io.Writer, header *types.Header) {
	enc := []interface{}{
		header.DposSigAddr,
		header.DposAckCountList,
		header.DposAcksHash,
		header.PowAnswers,
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
		//header.MixDigest,
		//header.Nonce,
	}
	if header.BaseFee != nil {
		enc = append(enc, header.BaseFee)
	}
	if err := rlp.Encode(w, enc); err != nil {
		panic("can't encode: " + err.Error())
	}
}

func GreatriDposAckRLP(DposAck *types.DposAck) []byte {
	b := new(bytes.Buffer)
	encodeSigDposAck(b, DposAck)
	return b.Bytes()
}

func encodeSigDposAck(w io.Writer, DposAck *types.DposAck) {
	enc := []interface{}{
		DposAck.EpochPosition,
		DposAck.Number,
		DposAck.BlockHash,
		DposAck.AckType,
	}
	if err := rlp.Encode(w, enc); err != nil {
		panic("can't encode: " + err.Error())
	}
}

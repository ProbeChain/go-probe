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

package pob

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math/big"
	"sort"
	"time"

	"github.com/probechain/go-probe/common"
	"github.com/probechain/go-probe/core/types"
	"github.com/probechain/go-probe/log"
	"github.com/probechain/go-probe/params"
	"github.com/probechain/go-probe/probedb"
	lru "github.com/hashicorp/golang-lru"
	"golang.org/x/crypto/sha3"
)

// Vote represents a single vote that an authorized validator made to modify the
// list of authorizations.
type Vote struct {
	Signer    common.Address `json:"signer"`    // Authorized validator that cast this vote
	Block     uint64         `json:"block"`      // Block number the vote was cast in
	Address   common.Address `json:"address"`    // Account being voted on
	Authorize bool           `json:"authorize"`  // Whether to authorize or deauthorize
}

// Tally is a simple vote tally to keep the current score of votes.
type Tally struct {
	Authorize bool `json:"authorize"` // Whether the vote is about authorizing or kicking
	Votes     int  `json:"votes"`     // Number of votes until now wanting to pass the proposal
}

// Snapshot is the state of the authorization and behavior scoring at a given point in time.
type Snapshot struct {
	config   *params.PobConfig // Consensus engine parameters
	sigcache *lru.ARCCache     // Cache of recent block signatures

	Number     uint64                              `json:"number"`     // Block number where the snapshot was created
	Hash       common.Hash                         `json:"hash"`       // Block hash where the snapshot was created
	Validators map[common.Address]*BehaviorScore   `json:"validators"` // Active validators + behavior scores
	Histories  map[common.Address]*ValidatorHistory `json:"histories"`  // Validator action histories
	Recents    map[uint64]common.Address            `json:"recents"`    // Set of recent block producers
	Votes      []*Vote                              `json:"votes"`      // List of votes cast in chronological order
	Tally      map[common.Address]Tally             `json:"tally"`      // Current vote tally
	PubKeys    map[common.Address][]byte            `json:"pubkeys"`    // Dilithium public keys for validators (optional)
}

// validatorsAscending implements the sort interface to allow sorting a list of addresses.
type validatorsAscending []common.Address

func (s validatorsAscending) Len() int           { return len(s) }
func (s validatorsAscending) Less(i, j int) bool { return bytes.Compare(s[i][:], s[j][:]) < 0 }
func (s validatorsAscending) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// newSnapshot creates a new snapshot with the specified startup parameters.
func newSnapshot(config *params.PobConfig, sigcache *lru.ARCCache, number uint64, hash common.Hash, validators []common.Address) *Snapshot {
	initialScore := config.InitialScore
	if initialScore == 0 {
		initialScore = defaultInitialScore
	}
	snap := &Snapshot{
		config:     config,
		sigcache:   sigcache,
		Number:     number,
		Hash:       hash,
		Validators: make(map[common.Address]*BehaviorScore),
		Histories:  make(map[common.Address]*ValidatorHistory),
		Recents:    make(map[uint64]common.Address),
		Tally:      make(map[common.Address]Tally),
		PubKeys:    make(map[common.Address][]byte),
	}
	for _, v := range validators {
		snap.Validators[v] = DefaultBehaviorScore(initialScore, number)
		snap.Histories[v] = &ValidatorHistory{}
	}
	return snap
}

// loadSnapshot loads an existing snapshot from the database.
func loadSnapshot(config *params.PobConfig, sigcache *lru.ARCCache, db probedb.Database, hash common.Hash) (*Snapshot, error) {
	blob, err := db.Get(append([]byte("pob-"), hash[:]...))
	if err != nil {
		return nil, err
	}
	snap := new(Snapshot)
	if err := json.Unmarshal(blob, snap); err != nil {
		return nil, err
	}
	snap.config = config
	snap.sigcache = sigcache
	return snap, nil
}

// store inserts the snapshot into the database.
func (s *Snapshot) store(db probedb.Database) error {
	blob, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return db.Put(append([]byte("pob-"), s.Hash[:]...), blob)
}

// copy creates a deep copy of the snapshot.
func (s *Snapshot) copy() *Snapshot {
	cpy := &Snapshot{
		config:     s.config,
		sigcache:   s.sigcache,
		Number:     s.Number,
		Hash:       s.Hash,
		Validators: make(map[common.Address]*BehaviorScore),
		Histories:  make(map[common.Address]*ValidatorHistory),
		Recents:    make(map[uint64]common.Address),
		Votes:      make([]*Vote, len(s.Votes)),
		Tally:      make(map[common.Address]Tally),
	}
	for addr, score := range s.Validators {
		scoreCopy := *score
		cpy.Validators[addr] = &scoreCopy
	}
	for addr, hist := range s.Histories {
		histCopy := *hist
		cpy.Histories[addr] = &histCopy
	}
	for block, validator := range s.Recents {
		cpy.Recents[block] = validator
	}
	for address, tally := range s.Tally {
		cpy.Tally[address] = tally
	}
	copy(cpy.Votes, s.Votes)
	return cpy
}

// validVote returns whether it makes sense to cast the specified vote in the
// given snapshot context.
func (s *Snapshot) validVote(address common.Address, authorize bool) bool {
	_, validator := s.Validators[address]
	return (validator && !authorize) || (!validator && authorize)
}

// cast adds a new vote into the tally.
func (s *Snapshot) cast(address common.Address, authorize bool) bool {
	if !s.validVote(address, authorize) {
		return false
	}
	if old, ok := s.Tally[address]; ok {
		old.Votes++
		s.Tally[address] = old
	} else {
		s.Tally[address] = Tally{Authorize: authorize, Votes: 1}
	}
	return true
}

// uncast removes a previously cast vote from the tally.
func (s *Snapshot) uncast(address common.Address, authorize bool) bool {
	tally, ok := s.Tally[address]
	if !ok {
		return false
	}
	if tally.Authorize != authorize {
		return false
	}
	if tally.Votes > 1 {
		tally.Votes--
		s.Tally[address] = tally
	} else {
		delete(s.Tally, address)
	}
	return true
}

// apply creates a new authorization snapshot by applying the given headers to the original one.
func (s *Snapshot) apply(headers []*types.Header) (*Snapshot, error) {
	if len(headers) == 0 {
		return s, nil
	}
	for i := 0; i < len(headers)-1; i++ {
		if headers[i+1].Number.Uint64() != headers[i].Number.Uint64()+1 {
			return nil, errInvalidVotingChain
		}
	}
	if headers[0].Number.Uint64() != s.Number+1 {
		return nil, errInvalidVotingChain
	}

	snap := s.copy()
	var (
		start  = time.Now()
		logged = time.Now()
	)
	for i, header := range headers {
		number := header.Number.Uint64()

		// Remove any votes on checkpoint blocks
		if number%s.config.Epoch == 0 {
			snap.Votes = nil
			snap.Tally = make(map[common.Address]Tally)
		}

		// Delete the oldest validator from the recent list
		if limit := uint64(len(snap.Validators)/2 + 1); number >= limit {
			delete(snap.Recents, number-limit)
		}

		// Track the block producer
		producer := header.ValidatorAddr
		if _, ok := snap.Validators[producer]; ok {
			snap.Recents[number] = producer
			// Update history: block proposed
			if hist, ok := snap.Histories[producer]; ok {
				hist.BlocksProposed++
			}
		}

		// Header authorized, discard any previous votes from the signer
		for j, vote := range snap.Votes {
			if vote.Signer == producer && vote.Address == header.Coinbase {
				snap.uncast(vote.Address, vote.Authorize)
				snap.Votes = append(snap.Votes[:j], snap.Votes[j+1:]...)
				break
			}
		}

		// Tally up the new vote from the producer
		var authorize bool
		if snap.cast(header.Coinbase, authorize) {
			snap.Votes = append(snap.Votes, &Vote{
				Signer:    producer,
				Block:     number,
				Address:   header.Coinbase,
				Authorize: authorize,
			})
		}

		// If the vote passed, update the list of validators
		if tally := snap.Tally[header.Coinbase]; tally.Votes > len(snap.Validators)/2 {
			initialScore := snap.config.InitialScore
			if initialScore == 0 {
				initialScore = defaultInitialScore
			}
			if tally.Authorize {
				snap.Validators[header.Coinbase] = DefaultBehaviorScore(initialScore, number)
				snap.Histories[header.Coinbase] = &ValidatorHistory{}
			} else {
				delete(snap.Validators, header.Coinbase)
				delete(snap.Histories, header.Coinbase)

				if limit := uint64(len(snap.Validators)/2 + 1); number >= limit {
					delete(snap.Recents, number-limit)
				}
				for j := 0; j < len(snap.Votes); j++ {
					if snap.Votes[j].Signer == header.Coinbase {
						snap.uncast(snap.Votes[j].Address, snap.Votes[j].Authorize)
						snap.Votes = append(snap.Votes[:j], snap.Votes[j+1:]...)
						j--
					}
				}
			}
			for j := 0; j < len(snap.Votes); j++ {
				if snap.Votes[j].Address == header.Coinbase {
					snap.Votes = append(snap.Votes[:j], snap.Votes[j+1:]...)
					j--
				}
			}
			delete(snap.Tally, header.Coinbase)
		}

		if time.Since(logged) > 8*time.Second {
			log.Info("Reconstructing voting history", "processed", i, "total", len(headers), "elapsed", common.PrettyDuration(time.Since(start)))
			logged = time.Now()
		}
	}
	if time.Since(start) > 8*time.Second {
		log.Info("Reconstructed voting history", "processed", len(headers), "elapsed", common.PrettyDuration(time.Since(start)))
	}
	snap.Number += uint64(len(headers))
	snap.Hash = headers[len(headers)-1].Hash()

	return snap, nil
}

// validators retrieves the list of active validators in ascending order.
func (s *Snapshot) validators() []common.Address {
	vals := make([]common.Address, 0, len(s.Validators))
	for v := range s.Validators {
		vals = append(vals, v)
	}
	sort.Sort(validatorsAscending(vals))
	return vals
}

// totalScore sums up all active validator scores.
func (s *Snapshot) totalScore() uint64 {
	var total uint64
	for _, score := range s.Validators {
		total += score.Total
	}
	return total
}

// selectProducer selects the block producer using weighted random selection by behavior score.
// The selection is deterministic: seed = keccak256(parentHash ++ number).
func (s *Snapshot) selectProducer(number uint64, parentHash common.Hash) common.Address {
	vals := s.validators()
	if len(vals) == 0 {
		return common.Address{}
	}
	if len(vals) == 1 {
		return vals[0]
	}

	total := s.totalScore()
	if total == 0 {
		// Fallback to round-robin if all scores are zero
		return vals[number%uint64(len(vals))]
	}

	// Deterministic seed from parent hash and block number
	seed := makeSeed(parentHash, number)

	// Weighted random selection
	target := seed % total
	var cumulative uint64
	for _, v := range vals {
		cumulative += s.Validators[v].Total
		if target < cumulative {
			return v
		}
	}
	// Fallback: return last validator
	return vals[len(vals)-1]
}

// inturn returns if a validator at a given block height is the selected producer.
func (s *Snapshot) inturn(number uint64, parentHash common.Hash, validator common.Address) bool {
	return s.selectProducer(number, parentHash) == validator
}

// makeSeed generates a deterministic seed from parentHash and block number using keccak256.
func makeSeed(parentHash common.Hash, number uint64) uint64 {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(parentHash[:])
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], number)
	hasher.Write(buf[:])
	var hash common.Hash
	hasher.(interface{ Sum([]byte) []byte }).Sum(hash[:0])
	return new(big.Int).SetBytes(hash[:8]).Uint64()
}

// encodeBehaviorData encodes the validator set and scores for checkpoint blocks.
// Layout: [1B: count N][N × (20B address + 8B score)]
func encodeBehaviorData(snap *Snapshot) []byte {
	vals := snap.validators()
	n := len(vals)
	data := make([]byte, 1+n*28)
	data[0] = byte(n)
	for i, v := range vals {
		offset := 1 + i*28
		copy(data[offset:offset+20], v[:])
		binary.BigEndian.PutUint64(data[offset+20:offset+28], snap.Validators[v].Total)
	}
	return data
}

// decodeBehaviorData decodes the validator set and scores from checkpoint extra-data.
func decodeBehaviorData(data []byte) (map[common.Address]uint64, error) {
	if len(data) < 1 {
		return nil, errInvalidCheckpointValidators
	}
	n := int(data[0])
	if len(data) != 1+n*28 {
		return nil, errInvalidCheckpointValidators
	}
	result := make(map[common.Address]uint64, n)
	for i := 0; i < n; i++ {
		offset := 1 + i*28
		var addr common.Address
		copy(addr[:], data[offset:offset+20])
		score := binary.BigEndian.Uint64(data[offset+20 : offset+28])
		result[addr] = score
	}
	return result, nil
}

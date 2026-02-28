// Copyright 2024 The go-probeum Authors
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

package pob

import (
	"github.com/probeum/go-probeum/common"
)

const (
	// maxScore is the maximum behavior score (basis points).
	maxScore = uint64(10000)
	// defaultInitialScore is the default starting score for new validators.
	defaultInitialScore = uint64(5000)
)

// BehaviorScore holds the composite and per-dimension scores for a validator.
type BehaviorScore struct {
	Total              uint64 `json:"total"`              // Composite score (0-10000 basis points)
	Liveness           uint64 `json:"liveness"`           // Liveness dimension score
	Correctness        uint64 `json:"correctness"`        // Correctness dimension score
	Cooperation        uint64 `json:"cooperation"`        // Cooperation dimension score
	Consistency        uint64 `json:"consistency"`        // Consistency dimension score
	SignalSovereignty  uint64 `json:"signalSovereignty"`  // Signal sovereignty dimension score
	LastUpdate         uint64 `json:"lastUpdate"`         // Block number of last score update
}

// ValidatorHistory tracks the on-chain actions of a validator for scoring.
type ValidatorHistory struct {
	BlocksProposed   uint64 `json:"blocksProposed"`
	BlocksMissed     uint64 `json:"blocksMissed"`
	InvalidProposals uint64 `json:"invalidProposals"`
	AcksGiven        uint64 `json:"acksGiven"`
	AcksMissed       uint64 `json:"acksMissed"`
	SlashCount       uint64 `json:"slashCount"`
	RydbergVerified  uint64 `json:"rydbergVerified"` // Blocks with verified Rydberg time source
	RadioSyncs       uint64 `json:"radioSyncs"`      // Successful radio-based time syncs
	StellarBlocks    uint64 `json:"stellarBlocks"`    // Blocks produced with AtomicTime present
}

// BehaviorAgent is the AI scoring agent that evaluates validator behavior
// across five dimensions: liveness, correctness, cooperation, consistency,
// and signal sovereignty.
type BehaviorAgent struct {
	// Weights for each dimension: [liveness, correctness, cooperation, consistency, signalSovereignty].
	// Each is expressed as a percentage out of 100 (must sum to 100).
	weights [5]uint64
}

// NewBehaviorAgent creates a new BehaviorAgent with the default dimension weights.
func NewBehaviorAgent() *BehaviorAgent {
	return &BehaviorAgent{
		weights: [5]uint64{25, 25, 18, 17, 15}, // liveness, correctness, cooperation, consistency, signalSovereignty
	}
}

// EvaluateValidator scores a validator based on its history.
// Returns a BehaviorScore with per-dimension and total scores in basis points (0-10000).
func (ba *BehaviorAgent) EvaluateValidator(addr common.Address, history *ValidatorHistory, blockNumber uint64) *BehaviorScore {
	liveness := ba.calcLiveness(history)
	correctness := ba.calcCorrectness(history)
	cooperation := ba.calcCooperation(history)
	consistency := ba.calcConsistency(history)
	signalSovereignty := ba.calcSignalSovereignty(history)

	total := (liveness*ba.weights[0] + correctness*ba.weights[1] +
		cooperation*ba.weights[2] + consistency*ba.weights[3] +
		signalSovereignty*ba.weights[4]) / 100

	if total > maxScore {
		total = maxScore
	}

	return &BehaviorScore{
		Total:             total,
		Liveness:          liveness,
		Correctness:       correctness,
		Cooperation:       cooperation,
		Consistency:       consistency,
		SignalSovereignty: signalSovereignty,
		LastUpdate:        blockNumber,
	}
}

// calcLiveness scores based on block production rate.
// Perfect production = maxScore, deducted for misses.
func (ba *BehaviorAgent) calcLiveness(h *ValidatorHistory) uint64 {
	totalOpportunities := h.BlocksProposed + h.BlocksMissed
	if totalOpportunities == 0 {
		return maxScore // No opportunities yet, assume perfect
	}
	return (h.BlocksProposed * maxScore) / totalOpportunities
}

// calcCorrectness scores based on valid vs invalid proposals.
func (ba *BehaviorAgent) calcCorrectness(h *ValidatorHistory) uint64 {
	totalProposals := h.BlocksProposed + h.InvalidProposals
	if totalProposals == 0 {
		return maxScore
	}
	return (h.BlocksProposed * maxScore) / totalProposals
}

// calcCooperation scores based on acknowledgment participation.
func (ba *BehaviorAgent) calcCooperation(h *ValidatorHistory) uint64 {
	totalAcks := h.AcksGiven + h.AcksMissed
	if totalAcks == 0 {
		return maxScore
	}
	return (h.AcksGiven * maxScore) / totalAcks
}

// calcConsistency scores inversely proportional to slash count.
// No slashes = maxScore. Each slash reduces by 1000 bp.
func (ba *BehaviorAgent) calcConsistency(h *ValidatorHistory) uint64 {
	penalty := h.SlashCount * 1000
	if penalty >= maxScore {
		return 0
	}
	return maxScore - penalty
}

// calcSignalSovereignty scores based on a validator's Stellar-Class capabilities:
// Rydberg-verified blocks, radio-based time syncs, and AtomicTime block production.
// Validators without stellar capabilities receive a neutral baseline score (5000)
// so they are not penalized below the default starting point.
func (ba *BehaviorAgent) calcSignalSovereignty(h *ValidatorHistory) uint64 {
	totalStellarOps := h.RydbergVerified + h.RadioSyncs + h.StellarBlocks
	if totalStellarOps == 0 {
		return defaultInitialScore // Neutral baseline — no penalty for non-stellar nodes
	}

	// Score components:
	// - Rydberg verification: 40% weight (demonstrates atomic receiver)
	// - Radio syncs: 30% weight (demonstrates RF time sync capability)
	// - Stellar blocks: 30% weight (demonstrates AtomicTime block production)
	var score uint64

	// Rydberg component: proportion of stellar blocks that have Rydberg verification
	if h.StellarBlocks > 0 {
		rydbergRatio := (h.RydbergVerified * maxScore) / h.StellarBlocks
		if rydbergRatio > maxScore {
			rydbergRatio = maxScore
		}
		score += rydbergRatio * 40 / 100
	}

	// Radio sync component: capped at maxScore for 100+ syncs
	radioScore := h.RadioSyncs * 100 // Each sync adds 100 bp
	if radioScore > maxScore {
		radioScore = maxScore
	}
	score += radioScore * 30 / 100

	// Stellar block production component
	stellarScore := h.StellarBlocks * 50 // Each stellar block adds 50 bp
	if stellarScore > maxScore {
		stellarScore = maxScore
	}
	score += stellarScore * 30 / 100

	if score > maxScore {
		score = maxScore
	}
	return score
}

// UpdateScores re-evaluates all validators in the snapshot and returns updated scores.
func (ba *BehaviorAgent) UpdateScores(validators map[common.Address]*BehaviorScore,
	histories map[common.Address]*ValidatorHistory, blockNumber uint64) map[common.Address]*BehaviorScore {

	updated := make(map[common.Address]*BehaviorScore, len(validators))
	for addr := range validators {
		history, ok := histories[addr]
		if !ok {
			history = &ValidatorHistory{}
		}
		updated[addr] = ba.EvaluateValidator(addr, history, blockNumber)
	}
	return updated
}

// ProportionalSlash reduces a validator's score proportionally to the severity.
// severity is in basis points (0-10000); the actual deduction is:
//
//	deduction = currentScore * severity * slashFraction / (10000 * 10000)
//
// Returns the new total score.
func (ba *BehaviorAgent) ProportionalSlash(score *BehaviorScore, severity uint64, slashFraction uint64) uint64 {
	if severity > maxScore {
		severity = maxScore
	}
	deduction := (score.Total * severity * slashFraction) / (maxScore * maxScore)
	if deduction >= score.Total {
		score.Total = 0
	} else {
		score.Total -= deduction
	}
	return score.Total
}

// DefaultBehaviorScore returns a behavior score initialized to the given initial score.
func DefaultBehaviorScore(initialScore uint64, blockNumber uint64) *BehaviorScore {
	return &BehaviorScore{
		Total:             initialScore,
		Liveness:          maxScore,
		Correctness:       maxScore,
		Cooperation:       maxScore,
		Consistency:       maxScore,
		SignalSovereignty: defaultInitialScore, // Neutral baseline for non-stellar nodes
		LastUpdate:        blockNumber,
	}
}

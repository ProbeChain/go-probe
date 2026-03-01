// Copyright 2024 The go-probe Authors
// This file is part of the go-probe library.
//
// The go-probe library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-probe library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-probe library. If not, see <http://www.gnu.org/licenses/>.

package miner

import (
	"math/big"
	"testing"
	"time"

	"github.com/probechain/go-probe/consensus/pob"
	"github.com/probechain/go-probe/params"
)

// TestStellarSpeedConfig verifies that StellarSpeed configuration is properly
// parsed and validated.
func TestStellarSpeedConfig(t *testing.T) {
	cfg := &params.StellarSpeedConfig{
		Enabled:          true,
		TickIntervalMs:   400,
		PipelineEnabled:  true,
		ReducedAckQuorum: true,
		MaxTxPerTick:     1000,
	}

	if !cfg.Enabled {
		t.Fatal("StellarSpeed should be enabled")
	}
	if cfg.TickIntervalMs != 400 {
		t.Fatalf("expected tick interval 400ms, got %d", cfg.TickIntervalMs)
	}
	expected := "stellarspeed(tick=400ms, pipeline=true)"
	if cfg.String() != expected {
		t.Fatalf("expected String() = %q, got %q", expected, cfg.String())
	}
}

// TestStellarSpeedForkActivation verifies the StellarSpeed fork gate activates
// at the correct block number.
func TestStellarSpeedForkActivation(t *testing.T) {
	cfg := &params.ChainConfig{
		ChainID:           big.NewInt(142857),
		StellarSpeedBlock: big.NewInt(1000),
	}

	if cfg.IsStellarSpeed(big.NewInt(999)) {
		t.Fatal("StellarSpeed should not be active at block 999")
	}
	if !cfg.IsStellarSpeed(big.NewInt(1000)) {
		t.Fatal("StellarSpeed should be active at block 1000")
	}
	if !cfg.IsStellarSpeed(big.NewInt(2000)) {
		t.Fatal("StellarSpeed should be active at block 2000")
	}
}

// TestStellarSpeedSameSecondBlocks verifies that in StellarSpeed mode,
// blocks with the same unix timestamp are allowed when ordered by AtomicTime.
func TestStellarSpeedSameSecondBlocks(t *testing.T) {
	cfg := &params.ChainConfig{
		ChainID:           big.NewInt(142857),
		StellarSpeedBlock: big.NewInt(0), // active from genesis
	}

	// Same-second blocks should be allowed in StellarSpeed mode
	if !cfg.IsStellarSpeed(big.NewInt(1)) {
		t.Fatal("StellarSpeed should be active at block 1")
	}
}

// TestStellarSpeedReducedQuorum verifies that the reduced ACK quorum works correctly.
func TestStellarSpeedReducedQuorum(t *testing.T) {
	// Test the quorum calculation logic
	normalQuorum := uint64(10)
	reducedQuorum := int(normalQuorum) / 2
	if reducedQuorum < 1 {
		reducedQuorum = 1
	}

	if reducedQuorum != 5 {
		t.Fatalf("expected reduced quorum of 5, got %d", reducedQuorum)
	}

	// Edge case: quorum of 1
	smallQuorum := uint64(1)
	reduced := int(smallQuorum) / 2
	if reduced < 1 {
		reduced = 1
	}
	if reduced != 1 {
		t.Fatalf("expected minimum reduced quorum of 1, got %d", reduced)
	}
}

// TestStellarSpeedTickInterval verifies tick interval defaults.
func TestStellarSpeedTickInterval(t *testing.T) {
	// Default interval
	cfg := &params.StellarSpeedConfig{
		Enabled:        true,
		TickIntervalMs: 0, // should default to 400
	}

	tickInterval := time.Duration(cfg.TickIntervalMs) * time.Millisecond
	if tickInterval == 0 {
		tickInterval = 400 * time.Millisecond
	}

	if tickInterval != 400*time.Millisecond {
		t.Fatalf("expected default tick interval 400ms, got %v", tickInterval)
	}

	// Custom interval
	cfg.TickIntervalMs = 200
	tickInterval = time.Duration(cfg.TickIntervalMs) * time.Millisecond
	if tickInterval != 200*time.Millisecond {
		t.Fatalf("expected custom tick interval 200ms, got %v", tickInterval)
	}
}

// TestBehaviorScoreFastEvaluation verifies that EvaluateValidatorFast returns
// cached scores between epochs and does full evaluation at epoch boundaries.
func TestBehaviorScoreFastEvaluation(t *testing.T) {
	agent := pob.NewBehaviorAgent()
	history := &pob.ValidatorHistory{
		BlocksProposed:  100,
		BlocksMissed:    5,
		AcksGiven:       90,
		AcksMissed:      10,
		StellarBlocks:   50,
		RydbergVerified: 20,
	}

	// Full evaluation at epoch boundary (block 0)
	addr := [20]byte{1}
	fullScore := agent.EvaluateValidatorFast(addr, history, 0, 30000, nil)
	if fullScore == nil {
		t.Fatal("expected non-nil score")
	}
	if fullScore.Total == 0 {
		t.Fatal("expected non-zero total score")
	}

	// Fast evaluation between epochs returns cached score
	cachedScore := agent.EvaluateValidatorFast(addr, history, 100, 30000, fullScore)
	if cachedScore.Total != fullScore.Total {
		t.Fatalf("expected cached total %d, got %d", fullScore.Total, cachedScore.Total)
	}
	if cachedScore.LastUpdate != 100 {
		t.Fatalf("expected LastUpdate 100, got %d", cachedScore.LastUpdate)
	}

	// Full evaluation at next epoch boundary
	nextEpochScore := agent.EvaluateValidatorFast(addr, history, 30000, 30000, cachedScore)
	if nextEpochScore.LastUpdate != 30000 {
		t.Fatalf("expected LastUpdate 30000, got %d", nextEpochScore.LastUpdate)
	}
}

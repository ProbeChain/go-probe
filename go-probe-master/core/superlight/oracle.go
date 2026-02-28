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

package superlight

import (
	"math/big"
)

// PriceOracle provides external price data for the DEX.
// Implementations may fetch from on-chain oracles, external APIs, or TWAP calculations.
type PriceOracle interface {
	// GetPrice returns the current price for a trading pair.
	// Returns nil if price is unavailable.
	GetPrice(pair TradingPair) *big.Int

	// GetTWAP returns the time-weighted average price over the given window (in seconds).
	GetTWAP(pair TradingPair, windowSeconds uint64) *big.Int
}

// NoOpOracle is a placeholder oracle that returns nil for all queries.
// Used when no external price feed is configured.
type NoOpOracle struct{}

// GetPrice returns nil (no price available).
func (o *NoOpOracle) GetPrice(pair TradingPair) *big.Int {
	return nil
}

// GetTWAP returns nil (no TWAP available).
func (o *NoOpOracle) GetTWAP(pair TradingPair, windowSeconds uint64) *big.Int {
	return nil
}

// Copyright 2017 The ProbeChain Authors
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

package params

// These are the multipliers for PROBE token denominations.
// Example: To get the pico value of an amount in 'gpico', use
//
//    new(big.Int).Mul(value, big.NewInt(params.GPico))
//
const (
	Pico    = 1
	GPico   = 1_000_000_000
	Probeer = 1_000_000_000_000_000_000 // 1e18 = 1 PROBE
)

// PROBE token metadata.
const (
	TokenName     = "PROBE"
	TokenSymbol   = "PROBE"
	TokenDecimals = 18
	TotalSupply   = 10_000_000_000 // 10 billion PROBE (in whole units)
)

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

	"github.com/probechain/go-probe/common"
)

// OrderSide represents the side of an order (buy or sell).
type OrderSide uint8

const (
	OrderSideBuy  OrderSide = 0
	OrderSideSell OrderSide = 1
)

// OrderStatus represents the current status of an order.
type OrderStatus uint8

const (
	OrderStatusOpen      OrderStatus = 0
	OrderStatusFilled    OrderStatus = 1
	OrderStatusPartial   OrderStatus = 2
	OrderStatusCancelled OrderStatus = 3
)

// TradingPair identifies a pair of assets that can be traded.
type TradingPair struct {
	BaseAsset  common.Address `json:"baseAsset"`  // Token contract address (zero for native PROBE)
	QuoteAsset common.Address `json:"quoteAsset"` // Token contract address (zero for native PROBE)
}

// String returns a human-readable representation of the trading pair.
func (tp TradingPair) String() string {
	return tp.BaseAsset.Hex() + "/" + tp.QuoteAsset.Hex()
}

// Order represents a limit order in the Superlight DEX.
type Order struct {
	ID        common.Hash    `json:"id"`        // Unique order ID (hash of order fields)
	Owner     common.Address `json:"owner"`     // Account that placed the order
	Pair      TradingPair    `json:"pair"`      // Trading pair
	Side      OrderSide      `json:"side"`      // Buy or sell
	Price     *big.Int       `json:"price"`     // Price in quote asset (scaled by 1e18)
	Amount    *big.Int       `json:"amount"`    // Total amount in base asset
	Filled    *big.Int       `json:"filled"`    // Amount already filled
	Status    OrderStatus    `json:"status"`    // Current order status
	Timestamp uint64         `json:"timestamp"` // Block timestamp when order was placed
	BlockNum  uint64         `json:"blockNum"`  // Block number when order was placed
}

// Remaining returns the unfilled amount of the order.
func (o *Order) Remaining() *big.Int {
	return new(big.Int).Sub(o.Amount, o.Filled)
}

// IsFilled returns true if the order is completely filled.
func (o *Order) IsFilled() bool {
	return o.Filled.Cmp(o.Amount) >= 0
}

// Trade represents a completed trade between two orders.
type Trade struct {
	ID         common.Hash    `json:"id"`         // Unique trade ID
	MakerOrder common.Hash    `json:"makerOrder"` // Maker order ID
	TakerOrder common.Hash    `json:"takerOrder"` // Taker order ID
	Maker      common.Address `json:"maker"`      // Maker address
	Taker      common.Address `json:"taker"`      // Taker address
	Pair       TradingPair    `json:"pair"`        // Trading pair
	Price      *big.Int       `json:"price"`       // Execution price
	Amount     *big.Int       `json:"amount"`      // Trade amount in base asset
	MakerFee   *big.Int       `json:"makerFee"`    // Fee charged to maker
	TakerFee   *big.Int       `json:"takerFee"`    // Fee charged to taker
	Timestamp  uint64         `json:"timestamp"`   // Block timestamp
	BlockNum   uint64         `json:"blockNum"`    // Block number
}

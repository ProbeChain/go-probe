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
	"sort"
	"sync"

	"github.com/probechain/go-probe/common"
)

// PriceLevel represents all orders at a single price point.
type PriceLevel struct {
	Price  *big.Int
	Orders []*Order // FIFO queue of orders at this price
}

// TotalAmount returns the total remaining amount at this price level.
func (pl *PriceLevel) TotalAmount() *big.Int {
	total := new(big.Int)
	for _, order := range pl.Orders {
		total.Add(total, order.Remaining())
	}
	return total
}

// OrderBook is an in-memory order book for a single trading pair.
// Buy orders (bids) are sorted descending by price.
// Sell orders (asks) are sorted ascending by price.
type OrderBook struct {
	mu   sync.RWMutex
	Pair TradingPair

	Bids []*PriceLevel // Sorted descending by price (highest first)
	Asks []*PriceLevel // Sorted ascending by price (lowest first)

	// Index for fast order lookup by ID
	orderIndex map[common.Hash]*Order
}

// NewOrderBook creates a new empty order book for the given trading pair.
func NewOrderBook(pair TradingPair) *OrderBook {
	return &OrderBook{
		Pair:       pair,
		Bids:       make([]*PriceLevel, 0),
		Asks:       make([]*PriceLevel, 0),
		orderIndex: make(map[common.Hash]*Order),
	}
}

// AddOrder inserts an order into the appropriate side of the book.
func (ob *OrderBook) AddOrder(order *Order) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	ob.orderIndex[order.ID] = order

	if order.Side == OrderSideBuy {
		ob.addToBids(order)
	} else {
		ob.addToAsks(order)
	}
}

// RemoveOrder removes an order from the book by ID.
func (ob *OrderBook) RemoveOrder(orderID common.Hash) *Order {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	order, ok := ob.orderIndex[orderID]
	if !ok {
		return nil
	}
	delete(ob.orderIndex, orderID)

	if order.Side == OrderSideBuy {
		ob.removeFromLevels(&ob.Bids, order)
	} else {
		ob.removeFromLevels(&ob.Asks, order)
	}
	return order
}

// GetOrder returns an order by ID, or nil if not found.
func (ob *OrderBook) GetOrder(orderID common.Hash) *Order {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return ob.orderIndex[orderID]
}

// BestBid returns the highest bid price, or nil if no bids.
func (ob *OrderBook) BestBid() *big.Int {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	if len(ob.Bids) == 0 {
		return nil
	}
	return new(big.Int).Set(ob.Bids[0].Price)
}

// BestAsk returns the lowest ask price, or nil if no asks.
func (ob *OrderBook) BestAsk() *big.Int {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	if len(ob.Asks) == 0 {
		return nil
	}
	return new(big.Int).Set(ob.Asks[0].Price)
}

// Depth returns the number of price levels on each side.
func (ob *OrderBook) Depth() (bids, asks int) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return len(ob.Bids), len(ob.Asks)
}

// addToBids inserts an order into bids (sorted descending by price, FIFO within level).
func (ob *OrderBook) addToBids(order *Order) {
	for _, level := range ob.Bids {
		if level.Price.Cmp(order.Price) == 0 {
			level.Orders = append(level.Orders, order)
			return
		}
	}
	// New price level
	newLevel := &PriceLevel{
		Price:  new(big.Int).Set(order.Price),
		Orders: []*Order{order},
	}
	ob.Bids = append(ob.Bids, newLevel)
	sort.Slice(ob.Bids, func(i, j int) bool {
		return ob.Bids[i].Price.Cmp(ob.Bids[j].Price) > 0 // Descending
	})
}

// addToAsks inserts an order into asks (sorted ascending by price, FIFO within level).
func (ob *OrderBook) addToAsks(order *Order) {
	for _, level := range ob.Asks {
		if level.Price.Cmp(order.Price) == 0 {
			level.Orders = append(level.Orders, order)
			return
		}
	}
	// New price level
	newLevel := &PriceLevel{
		Price:  new(big.Int).Set(order.Price),
		Orders: []*Order{order},
	}
	ob.Asks = append(ob.Asks, newLevel)
	sort.Slice(ob.Asks, func(i, j int) bool {
		return ob.Asks[i].Price.Cmp(ob.Asks[j].Price) < 0 // Ascending
	})
}

// removeFromLevels removes an order from the given price levels.
func (ob *OrderBook) removeFromLevels(levels *[]*PriceLevel, order *Order) {
	for i, level := range *levels {
		if level.Price.Cmp(order.Price) == 0 {
			for j, o := range level.Orders {
				if o.ID == order.ID {
					level.Orders = append(level.Orders[:j], level.Orders[j+1:]...)
					break
				}
			}
			// Remove empty price level
			if len(level.Orders) == 0 {
				*levels = append((*levels)[:i], (*levels)[i+1:]...)
			}
			return
		}
	}
}

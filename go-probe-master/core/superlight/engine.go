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
	"crypto/sha256"
	"errors"
	"math/big"
	"sync"

	"github.com/probechain/go-probe/common"
)

var (
	ErrOrderNotFound     = errors.New("order not found")
	ErrInvalidPrice      = errors.New("price must be positive")
	ErrInvalidAmount     = errors.New("amount must be positive")
	ErrNotOrderOwner     = errors.New("not order owner")
	ErrOrderAlreadyFilled = errors.New("order already filled or cancelled")
)

// MatchingEngine is the core DEX matching engine. It manages order books
// for multiple trading pairs and executes price-time priority matching.
type MatchingEngine struct {
	mu         sync.RWMutex
	books      map[TradingPair]*OrderBook
	trades     []*Trade
	tradeIndex map[common.Hash]*Trade
}

// NewMatchingEngine creates a new matching engine.
func NewMatchingEngine() *MatchingEngine {
	return &MatchingEngine{
		books:      make(map[TradingPair]*OrderBook),
		trades:     make([]*Trade, 0),
		tradeIndex: make(map[common.Hash]*Trade),
	}
}

// getOrCreateBook returns the order book for a pair, creating it if needed.
func (me *MatchingEngine) getOrCreateBook(pair TradingPair) *OrderBook {
	book, ok := me.books[pair]
	if !ok {
		book = NewOrderBook(pair)
		me.books[pair] = book
	}
	return book
}

// PlaceOrder places a new order and attempts to match it against existing orders.
// Returns the placed order and any resulting trades.
func (me *MatchingEngine) PlaceOrder(owner common.Address, pair TradingPair, side OrderSide,
	price, amount *big.Int, timestamp uint64, blockNum uint64) (*Order, []*Trade, error) {

	if price == nil || price.Sign() <= 0 {
		return nil, nil, ErrInvalidPrice
	}
	if amount == nil || amount.Sign() <= 0 {
		return nil, nil, ErrInvalidAmount
	}

	me.mu.Lock()
	defer me.mu.Unlock()

	// Generate order ID from fields
	orderID := generateOrderID(owner, pair, side, price, amount, blockNum)

	order := &Order{
		ID:        orderID,
		Owner:     owner,
		Pair:      pair,
		Side:      side,
		Price:     new(big.Int).Set(price),
		Amount:    new(big.Int).Set(amount),
		Filled:    new(big.Int),
		Status:    OrderStatusOpen,
		Timestamp: timestamp,
		BlockNum:  blockNum,
	}

	book := me.getOrCreateBook(pair)

	// Try to match against opposite side
	trades := me.matchOrder(order, book, timestamp, blockNum)

	// If order has remaining amount, add to the book
	if order.Remaining().Sign() > 0 && order.Status != OrderStatusCancelled {
		if order.Filled.Sign() > 0 {
			order.Status = OrderStatusPartial
		}
		book.AddOrder(order)
	} else if order.Remaining().Sign() == 0 {
		order.Status = OrderStatusFilled
	}

	return order, trades, nil
}

// CancelOrder cancels an open order. Only the owner can cancel.
func (me *MatchingEngine) CancelOrder(orderID common.Hash, owner common.Address) (*Order, error) {
	me.mu.Lock()
	defer me.mu.Unlock()

	for _, book := range me.books {
		order := book.GetOrder(orderID)
		if order == nil {
			continue
		}
		if order.Owner != owner {
			return nil, ErrNotOrderOwner
		}
		if order.Status == OrderStatusFilled || order.Status == OrderStatusCancelled {
			return nil, ErrOrderAlreadyFilled
		}
		order.Status = OrderStatusCancelled
		book.RemoveOrder(orderID)
		return order, nil
	}
	return nil, ErrOrderNotFound
}

// GetOrderbook returns a snapshot of the order book for a pair.
func (me *MatchingEngine) GetOrderbook(pair TradingPair, depth int) (bids, asks []PriceLevelSnapshot) {
	me.mu.RLock()
	defer me.mu.RUnlock()

	book, ok := me.books[pair]
	if !ok {
		return nil, nil
	}

	book.mu.RLock()
	defer book.mu.RUnlock()

	bidCount := len(book.Bids)
	if depth > 0 && depth < bidCount {
		bidCount = depth
	}
	for i := 0; i < bidCount; i++ {
		bids = append(bids, PriceLevelSnapshot{
			Price:  new(big.Int).Set(book.Bids[i].Price),
			Amount: book.Bids[i].TotalAmount(),
			Count:  len(book.Bids[i].Orders),
		})
	}

	askCount := len(book.Asks)
	if depth > 0 && depth < askCount {
		askCount = depth
	}
	for i := 0; i < askCount; i++ {
		asks = append(asks, PriceLevelSnapshot{
			Price:  new(big.Int).Set(book.Asks[i].Price),
			Amount: book.Asks[i].TotalAmount(),
			Count:  len(book.Asks[i].Orders),
		})
	}

	return bids, asks
}

// GetTrades returns recent trades, up to limit.
func (me *MatchingEngine) GetTrades(pair TradingPair, limit int) []*Trade {
	me.mu.RLock()
	defer me.mu.RUnlock()

	var result []*Trade
	// Walk backwards for most recent first
	for i := len(me.trades) - 1; i >= 0 && len(result) < limit; i-- {
		if me.trades[i].Pair == pair {
			result = append(result, me.trades[i])
		}
	}
	return result
}

// PriceLevelSnapshot is a read-only snapshot of a price level.
type PriceLevelSnapshot struct {
	Price  *big.Int `json:"price"`
	Amount *big.Int `json:"amount"`
	Count  int      `json:"count"`
}

// matchOrder attempts to match the incoming order against the opposite side.
func (me *MatchingEngine) matchOrder(order *Order, book *OrderBook, timestamp, blockNum uint64) []*Trade {
	var trades []*Trade
	var oppositeSide *[]*PriceLevel

	if order.Side == OrderSideBuy {
		oppositeSide = &book.Asks
	} else {
		oppositeSide = &book.Bids
	}

	for len(*oppositeSide) > 0 && order.Remaining().Sign() > 0 {
		bestLevel := (*oppositeSide)[0]

		// Check price compatibility
		if order.Side == OrderSideBuy && order.Price.Cmp(bestLevel.Price) < 0 {
			break // Buy price too low
		}
		if order.Side == OrderSideSell && order.Price.Cmp(bestLevel.Price) > 0 {
			break // Sell price too high
		}

		// Match against orders at this level (FIFO)
		for len(bestLevel.Orders) > 0 && order.Remaining().Sign() > 0 {
			makerOrder := bestLevel.Orders[0]

			// Calculate fill amount
			fillAmount := new(big.Int).Set(order.Remaining())
			if makerOrder.Remaining().Cmp(fillAmount) < 0 {
				fillAmount.Set(makerOrder.Remaining())
			}

			// Create trade
			tradeID := generateTradeID(makerOrder.ID, order.ID, fillAmount, blockNum)
			trade := &Trade{
				ID:         tradeID,
				MakerOrder: makerOrder.ID,
				TakerOrder: order.ID,
				Maker:      makerOrder.Owner,
				Taker:      order.Owner,
				Pair:       order.Pair,
				Price:      new(big.Int).Set(bestLevel.Price), // Execute at maker's price
				Amount:     fillAmount,
				MakerFee:   new(big.Int), // Fees calculated by Manager
				TakerFee:   new(big.Int),
				Timestamp:  timestamp,
				BlockNum:   blockNum,
			}
			trades = append(trades, trade)
			me.trades = append(me.trades, trade)
			me.tradeIndex[tradeID] = trade

			// Update fill amounts
			makerOrder.Filled.Add(makerOrder.Filled, fillAmount)
			order.Filled.Add(order.Filled, fillAmount)

			// Remove fully filled maker order
			if makerOrder.IsFilled() {
				makerOrder.Status = OrderStatusFilled
				delete(book.orderIndex, makerOrder.ID)
				bestLevel.Orders = bestLevel.Orders[1:]
			} else {
				makerOrder.Status = OrderStatusPartial
			}
		}

		// Remove empty level
		if len(bestLevel.Orders) == 0 {
			*oppositeSide = (*oppositeSide)[1:]
		}
	}

	return trades
}

// generateOrderID creates a deterministic order ID from order fields.
func generateOrderID(owner common.Address, pair TradingPair, side OrderSide, price, amount *big.Int, blockNum uint64) common.Hash {
	h := sha256.New()
	h.Write(owner.Bytes())
	h.Write(pair.BaseAsset.Bytes())
	h.Write(pair.QuoteAsset.Bytes())
	h.Write([]byte{byte(side)})
	h.Write(price.Bytes())
	h.Write(amount.Bytes())
	h.Write(new(big.Int).SetUint64(blockNum).Bytes())
	return common.BytesToHash(h.Sum(nil))
}

// generateTradeID creates a deterministic trade ID.
func generateTradeID(makerID, takerID common.Hash, amount *big.Int, blockNum uint64) common.Hash {
	h := sha256.New()
	h.Write(makerID.Bytes())
	h.Write(takerID.Bytes())
	h.Write(amount.Bytes())
	h.Write(new(big.Int).SetUint64(blockNum).Bytes())
	return common.BytesToHash(h.Sum(nil))
}

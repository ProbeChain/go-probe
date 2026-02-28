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
	"context"
	"math/big"

	"github.com/probechain/go-probe/common"
)

// PublicSuperlightAPI provides the public RPC API for the Superlight DEX.
type PublicSuperlightAPI struct {
	manager *Manager
}

// NewPublicSuperlightAPI creates a new Superlight DEX API.
func NewPublicSuperlightAPI(manager *Manager) *PublicSuperlightAPI {
	return &PublicSuperlightAPI{manager: manager}
}

// OrderbookResult is the JSON-RPC response for GetOrderbook.
type OrderbookResult struct {
	Pair string               `json:"pair"`
	Bids []PriceLevelSnapshot `json:"bids"`
	Asks []PriceLevelSnapshot `json:"asks"`
}

// GetOrderbook returns the current state of the order book for a trading pair.
func (api *PublicSuperlightAPI) GetOrderbook(_ context.Context, baseAsset, quoteAsset common.Address, depth int) (*OrderbookResult, error) {
	pair := TradingPair{BaseAsset: baseAsset, QuoteAsset: quoteAsset}
	bids, asks := api.manager.engine.GetOrderbook(pair, depth)

	return &OrderbookResult{
		Pair: pair.String(),
		Bids: bids,
		Asks: asks,
	}, nil
}

// TradeResult is the JSON-RPC response for GetTrades.
type TradeResult struct {
	ID        common.Hash    `json:"id"`
	Maker     common.Address `json:"maker"`
	Taker     common.Address `json:"taker"`
	Price     *big.Int       `json:"price"`
	Amount    *big.Int       `json:"amount"`
	MakerFee  *big.Int       `json:"makerFee"`
	TakerFee  *big.Int       `json:"takerFee"`
	Timestamp uint64         `json:"timestamp"`
	BlockNum  uint64         `json:"blockNum"`
}

// GetTrades returns recent trades for a trading pair.
func (api *PublicSuperlightAPI) GetTrades(_ context.Context, baseAsset, quoteAsset common.Address, limit int) ([]TradeResult, error) {
	pair := TradingPair{BaseAsset: baseAsset, QuoteAsset: quoteAsset}
	if limit <= 0 {
		limit = 50
	}
	trades := api.manager.engine.GetTrades(pair, limit)

	results := make([]TradeResult, len(trades))
	for i, trade := range trades {
		results[i] = TradeResult{
			ID:        trade.ID,
			Maker:     trade.Maker,
			Taker:     trade.Taker,
			Price:     trade.Price,
			Amount:    trade.Amount,
			MakerFee:  trade.MakerFee,
			TakerFee:  trade.TakerFee,
			Timestamp: trade.Timestamp,
			BlockNum:  trade.BlockNum,
		}
	}
	return results, nil
}

// GetPrice returns the current best price for a trading pair from the oracle.
func (api *PublicSuperlightAPI) GetPrice(_ context.Context, baseAsset, quoteAsset common.Address) (*big.Int, error) {
	pair := TradingPair{BaseAsset: baseAsset, QuoteAsset: quoteAsset}
	price := api.manager.oracle.GetPrice(pair)
	if price == nil {
		// Fall back to best bid/ask midpoint
		bids, asks := api.manager.engine.GetOrderbook(pair, 1)
		if len(bids) > 0 && len(asks) > 0 {
			mid := new(big.Int).Add(bids[0].Price, asks[0].Price)
			mid.Div(mid, big.NewInt(2))
			return mid, nil
		}
		return nil, nil
	}
	return price, nil
}

// OrderResult is the JSON-RPC response for GetOrder.
type OrderResult struct {
	ID        common.Hash    `json:"id"`
	Owner     common.Address `json:"owner"`
	Side      OrderSide      `json:"side"`
	Price     *big.Int       `json:"price"`
	Amount    *big.Int       `json:"amount"`
	Filled    *big.Int       `json:"filled"`
	Remaining *big.Int       `json:"remaining"`
	Status    OrderStatus    `json:"status"`
	Timestamp uint64         `json:"timestamp"`
}

// GetOrder returns the status of an order by ID.
func (api *PublicSuperlightAPI) GetOrder(_ context.Context, baseAsset, quoteAsset common.Address, orderID common.Hash) (*OrderResult, error) {
	pair := TradingPair{BaseAsset: baseAsset, QuoteAsset: quoteAsset}

	api.manager.engine.mu.RLock()
	defer api.manager.engine.mu.RUnlock()

	book, ok := api.manager.engine.books[pair]
	if !ok {
		return nil, ErrOrderNotFound
	}
	order := book.GetOrder(orderID)
	if order == nil {
		return nil, ErrOrderNotFound
	}

	return &OrderResult{
		ID:        order.ID,
		Owner:     order.Owner,
		Side:      order.Side,
		Price:     order.Price,
		Amount:    order.Amount,
		Filled:    order.Filled,
		Remaining: order.Remaining(),
		Status:    order.Status,
		Timestamp: order.Timestamp,
	}, nil
}

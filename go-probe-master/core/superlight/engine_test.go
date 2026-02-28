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
	"testing"

	"github.com/probechain/go-probe/common"
)

var (
	testPair = TradingPair{
		BaseAsset:  common.HexToAddress("0x1111111111111111111111111111111111111111"),
		QuoteAsset: common.HexToAddress("0x0000000000000000000000000000000000000000"),
	}
	alice = common.HexToAddress("0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	bob   = common.HexToAddress("0xBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB")
	carol = common.HexToAddress("0xCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC")
)

func TestPlaceAndMatchOrders(t *testing.T) {
	engine := NewMatchingEngine()

	// Alice places a sell order: 10 tokens at price 100
	sellOrder, trades, err := engine.PlaceOrder(alice, testPair, OrderSideSell,
		big.NewInt(100), big.NewInt(10), 1000, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(trades) != 0 {
		t.Fatal("expected no trades for first order")
	}
	if sellOrder.Status != OrderStatusOpen {
		t.Fatalf("expected open status, got %d", sellOrder.Status)
	}

	// Bob places a buy order: 10 tokens at price 100 (should match)
	buyOrder, trades, err := engine.PlaceOrder(bob, testPair, OrderSideBuy,
		big.NewInt(100), big.NewInt(10), 1001, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if buyOrder.Status != OrderStatusFilled {
		t.Fatalf("expected filled status, got %d", buyOrder.Status)
	}

	trade := trades[0]
	if trade.Amount.Cmp(big.NewInt(10)) != 0 {
		t.Fatalf("expected trade amount 10, got %s", trade.Amount)
	}
	if trade.Price.Cmp(big.NewInt(100)) != 0 {
		t.Fatalf("expected trade price 100, got %s", trade.Price)
	}
	if trade.Maker != alice {
		t.Fatal("expected alice as maker")
	}
	if trade.Taker != bob {
		t.Fatal("expected bob as taker")
	}
}

func TestPartialFill(t *testing.T) {
	engine := NewMatchingEngine()

	// Alice sells 20 tokens at 100
	_, _, err := engine.PlaceOrder(alice, testPair, OrderSideSell,
		big.NewInt(100), big.NewInt(20), 1000, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Bob buys only 5 tokens at 100 (partial fill)
	_, trades, err := engine.PlaceOrder(bob, testPair, OrderSideBuy,
		big.NewInt(100), big.NewInt(5), 1001, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if trades[0].Amount.Cmp(big.NewInt(5)) != 0 {
		t.Fatalf("expected fill of 5, got %s", trades[0].Amount)
	}

	// Order book should still have 15 tokens on the sell side
	bids, asks := engine.GetOrderbook(testPair, 10)
	if len(bids) != 0 {
		t.Fatal("expected no bids")
	}
	if len(asks) != 1 {
		t.Fatalf("expected 1 ask level, got %d", len(asks))
	}
	if asks[0].Amount.Cmp(big.NewInt(15)) != 0 {
		t.Fatalf("expected 15 remaining, got %s", asks[0].Amount)
	}
}

func TestCancelOrder(t *testing.T) {
	engine := NewMatchingEngine()

	// Alice places an order
	order, _, err := engine.PlaceOrder(alice, testPair, OrderSideSell,
		big.NewInt(100), big.NewInt(10), 1000, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Bob tries to cancel Alice's order (should fail)
	_, err = engine.CancelOrder(order.ID, bob)
	if err != ErrNotOrderOwner {
		t.Fatalf("expected ErrNotOrderOwner, got %v", err)
	}

	// Alice cancels her own order
	cancelled, err := engine.CancelOrder(order.ID, alice)
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != OrderStatusCancelled {
		t.Fatalf("expected cancelled status, got %d", cancelled.Status)
	}

	// Order book should be empty
	bids, asks := engine.GetOrderbook(testPair, 10)
	if len(bids) != 0 || len(asks) != 0 {
		t.Fatal("expected empty order book after cancel")
	}
}

func TestPriceTimePriority(t *testing.T) {
	engine := NewMatchingEngine()

	// Multiple sell orders at different prices
	_, _, _ = engine.PlaceOrder(alice, testPair, OrderSideSell,
		big.NewInt(110), big.NewInt(5), 1000, 1) // Higher price
	_, _, _ = engine.PlaceOrder(bob, testPair, OrderSideSell,
		big.NewInt(100), big.NewInt(5), 1001, 2) // Lower price (should match first)
	_, _, _ = engine.PlaceOrder(carol, testPair, OrderSideSell,
		big.NewInt(100), big.NewInt(5), 1002, 3) // Same price as bob, but later (FIFO)

	// Large buy order should match bob first (best price), then carol (same price, FIFO)
	_, trades, err := engine.PlaceOrder(alice, testPair, OrderSideBuy,
		big.NewInt(110), big.NewInt(12), 1003, 4)
	if err != nil {
		t.Fatal(err)
	}

	if len(trades) != 3 {
		t.Fatalf("expected 3 trades, got %d", len(trades))
	}

	// First trade: 5 from bob at 100
	if trades[0].Maker != bob {
		t.Fatal("first trade should be with bob")
	}
	if trades[0].Price.Cmp(big.NewInt(100)) != 0 {
		t.Fatalf("first trade price should be 100, got %s", trades[0].Price)
	}

	// Second trade: 5 from carol at 100
	if trades[1].Maker != carol {
		t.Fatal("second trade should be with carol")
	}

	// Third trade: 2 from alice's sell at 110
	if trades[2].Price.Cmp(big.NewInt(110)) != 0 {
		t.Fatalf("third trade price should be 110, got %s", trades[2].Price)
	}
	if trades[2].Amount.Cmp(big.NewInt(2)) != 0 {
		t.Fatalf("third trade amount should be 2, got %s", trades[2].Amount)
	}
}

func TestNoMatchWhenPricesDoNotCross(t *testing.T) {
	engine := NewMatchingEngine()

	// Sell at 110
	_, _, _ = engine.PlaceOrder(alice, testPair, OrderSideSell,
		big.NewInt(110), big.NewInt(10), 1000, 1)

	// Buy at 100 (below sell) - no match
	_, trades, err := engine.PlaceOrder(bob, testPair, OrderSideBuy,
		big.NewInt(100), big.NewInt(10), 1001, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(trades) != 0 {
		t.Fatal("expected no trades when prices don't cross")
	}

	// Both orders should be in the book
	bids, asks := engine.GetOrderbook(testPair, 10)
	if len(bids) != 1 || len(asks) != 1 {
		t.Fatal("expected one bid and one ask")
	}
}

func TestInvalidOrderParameters(t *testing.T) {
	engine := NewMatchingEngine()

	// Zero price
	_, _, err := engine.PlaceOrder(alice, testPair, OrderSideBuy,
		big.NewInt(0), big.NewInt(10), 1000, 1)
	if err != ErrInvalidPrice {
		t.Fatalf("expected ErrInvalidPrice, got %v", err)
	}

	// Negative amount
	_, _, err = engine.PlaceOrder(alice, testPair, OrderSideBuy,
		big.NewInt(100), big.NewInt(-1), 1000, 1)
	if err != ErrInvalidAmount {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}

	// Nil price
	_, _, err = engine.PlaceOrder(alice, testPair, OrderSideBuy,
		nil, big.NewInt(10), 1000, 1)
	if err != ErrInvalidPrice {
		t.Fatalf("expected ErrInvalidPrice, got %v", err)
	}
}

func TestGetTrades(t *testing.T) {
	engine := NewMatchingEngine()

	// Create matching orders to generate trades
	_, _, _ = engine.PlaceOrder(alice, testPair, OrderSideSell,
		big.NewInt(100), big.NewInt(10), 1000, 1)
	_, _, _ = engine.PlaceOrder(bob, testPair, OrderSideBuy,
		big.NewInt(100), big.NewInt(10), 1001, 2)

	trades := engine.GetTrades(testPair, 10)
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
}

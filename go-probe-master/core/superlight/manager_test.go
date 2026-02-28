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
	"github.com/probechain/go-probe/params"
)

func TestManagerProcessPlaceOrder(t *testing.T) {
	config := &params.SuperlightConfig{
		Enabled:             true,
		MakerFeeBps:         10,
		TakerFeeBps:         30,
		MaxOrdersPerAccount: 100,
	}
	mgr := NewManager(config)

	// Encode a place order: sell 10 tokens at price 1000
	data := make([]byte, 106) // OpPlaceOrder(1) + side(1) + base(20) + quote(20) + price(32) + amount(32)
	data[0] = OpPlaceOrder
	data[1] = byte(OrderSideSell)
	copy(data[2:22], common.HexToAddress("0x1111111111111111111111111111111111111111").Bytes())
	// quote is zero (native PROBE) - already zero in slice
	price := big.NewInt(1000)
	priceBytes := price.Bytes()
	copy(data[42+(32-len(priceBytes)):74], priceBytes)
	amount := big.NewInt(10)
	amountBytes := amount.Bytes()
	copy(data[74+(32-len(amountBytes)):106], amountBytes)

	trades, err := mgr.ProcessDEXTransaction(alice, data, 1000, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(trades) != 0 {
		t.Fatal("expected no trades for first order")
	}

	// Now place a matching buy order
	data[0] = OpPlaceOrder
	data[1] = byte(OrderSideBuy)
	trades, err = mgr.ProcessDEXTransaction(bob, data, 1001, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
}

func TestManagerFees(t *testing.T) {
	config := &params.SuperlightConfig{
		Enabled:     true,
		MakerFeeBps: 10, // 0.1%
		TakerFeeBps: 30, // 0.3%
	}
	mgr := NewManager(config)

	// Place and match orders
	pair := TradingPair{
		BaseAsset:  common.HexToAddress("0x1111111111111111111111111111111111111111"),
		QuoteAsset: common.Address{},
	}
	_, _, _ = mgr.engine.PlaceOrder(alice, pair, OrderSideSell,
		big.NewInt(1e18), big.NewInt(100), 1000, 1)
	_, trades, _ := mgr.engine.PlaceOrder(bob, pair, OrderSideBuy,
		big.NewInt(1e18), big.NewInt(100), 1001, 2)

	// Calculate fees
	for _, trade := range trades {
		mgr.calculateFees(trade)
	}

	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}

	// Verify fees are non-negative
	if trades[0].MakerFee.Sign() < 0 {
		t.Fatal("maker fee should not be negative")
	}
	if trades[0].TakerFee.Sign() < 0 {
		t.Fatal("taker fee should not be negative")
	}

	// Taker fee should be larger than maker fee (30 bps > 10 bps)
	if trades[0].TakerFee.Cmp(trades[0].MakerFee) < 0 {
		t.Fatal("taker fee should be >= maker fee")
	}
}

func TestManagerDisabled(t *testing.T) {
	config := &params.SuperlightConfig{
		Enabled: false,
	}
	mgr := NewManager(config)

	data := []byte{OpPlaceOrder, byte(OrderSideBuy)}
	_, err := mgr.ProcessDEXTransaction(alice, data, 1000, 1)
	if err != ErrDEXNotEnabled {
		t.Fatalf("expected ErrDEXNotEnabled, got %v", err)
	}
}

func TestManagerInvalidOp(t *testing.T) {
	config := &params.SuperlightConfig{Enabled: true}
	mgr := NewManager(config)

	// Empty data
	_, err := mgr.ProcessDEXTransaction(alice, []byte{}, 1000, 1)
	if err != ErrInvalidDEXOp {
		t.Fatalf("expected ErrInvalidDEXOp, got %v", err)
	}

	// Unknown op code
	_, err = mgr.ProcessDEXTransaction(alice, []byte{0xFF}, 1000, 1)
	if err != ErrInvalidDEXOp {
		t.Fatalf("expected ErrInvalidDEXOp, got %v", err)
	}
}

func TestSettleTrade(t *testing.T) {
	config := &params.SuperlightConfig{Enabled: true}
	mgr := NewManager(config)

	trade := &Trade{
		ID:     common.HexToHash("0x1234"),
		Maker:  alice,
		Taker:  bob,
		Amount: big.NewInt(100),
		Price:  big.NewInt(1000),
	}

	// Settlement should succeed (placeholder implementation)
	err := mgr.SettleTrade(trade)
	if err != nil {
		t.Fatal(err)
	}
}

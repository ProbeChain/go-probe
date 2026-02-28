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
	"errors"
	"math/big"
	"sync"

	"github.com/probechain/go-probe/common"
	"github.com/probechain/go-probe/log"
	"github.com/probechain/go-probe/params"
)

var (
	ErrDEXNotEnabled = errors.New("superlight DEX not enabled")
	ErrInvalidDEXOp  = errors.New("invalid DEX operation")
)

// DEX operation types encoded in transaction data.
const (
	OpPlaceOrder  byte = 0x01
	OpCancelOrder byte = 0x02
)

// Manager coordinates the Superlight DEX matching engine with on-chain state.
// It processes DEX transactions and settles trades via the special address.
type Manager struct {
	mu     sync.RWMutex
	config *params.SuperlightConfig
	engine *MatchingEngine
	oracle PriceOracle
}

// NewManager creates a new Superlight DEX manager.
func NewManager(config *params.SuperlightConfig) *Manager {
	if config == nil {
		config = &params.SuperlightConfig{
			MakerFeeBps:         10,  // 0.1%
			TakerFeeBps:         30,  // 0.3%
			MaxOrdersPerAccount: 100,
		}
	}
	return &Manager{
		config: config,
		engine: NewMatchingEngine(),
		oracle: &NoOpOracle{},
	}
}

// SetOracle sets the price oracle for the DEX.
func (m *Manager) SetOracle(oracle PriceOracle) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.oracle = oracle
}

// Engine returns the matching engine for direct access.
func (m *Manager) Engine() *MatchingEngine {
	return m.engine
}

// ProcessDEXTransaction processes a Superlight DEX transaction.
// It parses the operation from tx data and executes it.
func (m *Manager) ProcessDEXTransaction(from common.Address, data []byte,
	timestamp uint64, blockNum uint64) ([]*Trade, error) {

	if !m.config.Enabled {
		return nil, ErrDEXNotEnabled
	}
	if len(data) < 1 {
		return nil, ErrInvalidDEXOp
	}

	switch data[0] {
	case OpPlaceOrder:
		return m.processPlaceOrder(from, data[1:], timestamp, blockNum)
	case OpCancelOrder:
		return nil, m.processCancelOrder(from, data[1:])
	default:
		return nil, ErrInvalidDEXOp
	}
}

// processPlaceOrder decodes and executes a place order operation.
// Data format: [side(1)] [baseAsset(20)] [quoteAsset(20)] [price(32)] [amount(32)]
func (m *Manager) processPlaceOrder(from common.Address, data []byte, timestamp, blockNum uint64) ([]*Trade, error) {
	if len(data) < 105 { // 1 + 20 + 20 + 32 + 32
		return nil, ErrInvalidDEXOp
	}

	side := OrderSide(data[0])
	baseAsset := common.BytesToAddress(data[1:21])
	quoteAsset := common.BytesToAddress(data[21:41])
	price := new(big.Int).SetBytes(data[41:73])
	amount := new(big.Int).SetBytes(data[73:105])

	pair := TradingPair{BaseAsset: baseAsset, QuoteAsset: quoteAsset}

	order, trades, err := m.engine.PlaceOrder(from, pair, side, price, amount, timestamp, blockNum)
	if err != nil {
		return nil, err
	}

	// Calculate fees for each trade
	for _, trade := range trades {
		m.calculateFees(trade)
	}

	log.Debug("Superlight order placed", "orderID", order.ID.Hex(), "side", side,
		"price", price, "amount", amount, "trades", len(trades))

	return trades, nil
}

// processCancelOrder decodes and executes a cancel order operation.
// Data format: [orderID(32)]
func (m *Manager) processCancelOrder(from common.Address, data []byte) error {
	if len(data) < 32 {
		return ErrInvalidDEXOp
	}

	orderID := common.BytesToHash(data[:32])
	order, err := m.engine.CancelOrder(orderID, from)
	if err != nil {
		return err
	}

	log.Debug("Superlight order cancelled", "orderID", order.ID.Hex())
	return nil
}

// calculateFees computes maker and taker fees for a trade.
func (m *Manager) calculateFees(trade *Trade) {
	// Fee = amount * price * feeBps / 10000
	quoteValue := new(big.Int).Mul(trade.Amount, trade.Price)
	quoteValue.Div(quoteValue, big.NewInt(1e18)) // Normalize from price scaling

	trade.MakerFee = new(big.Int).Mul(quoteValue, new(big.Int).SetUint64(m.config.MakerFeeBps))
	trade.MakerFee.Div(trade.MakerFee, big.NewInt(10000))

	trade.TakerFee = new(big.Int).Mul(quoteValue, new(big.Int).SetUint64(m.config.TakerFeeBps))
	trade.TakerFee.Div(trade.TakerFee, big.NewInt(10000))
}

// SettleTrade processes on-chain settlement for a trade.
// In a full implementation, this would transfer tokens between maker and taker
// through the DEX settlement special address.
func (m *Manager) SettleTrade(trade *Trade) error {
	log.Info("Settling trade", "tradeID", trade.ID.Hex(),
		"maker", trade.Maker.Hex(), "taker", trade.Taker.Hex(),
		"amount", trade.Amount, "price", trade.Price)

	// Settlement is performed by the state transition function when it
	// encounters a SuperlightTx targeting SPECIAL_ADDRESS_FOR_DEX_SETTLEMENT.
	// The state changes (balance transfers) happen in core/state_transition.go.
	return nil
}

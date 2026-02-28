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

// AtomicSwapBridge defines the interface for cross-chain atomic swaps.
// This enables the Superlight DEX to support trading assets across chains
// using Hash Time-Locked Contracts (HTLC).
type AtomicSwapBridge interface {
	// InitiateSwap creates a new HTLC on the source chain.
	// Returns the swap ID and the hash lock.
	InitiateSwap(
		sender common.Address,
		receiver common.Address,
		amount *big.Int,
		hashLock common.Hash,
		timeLockSeconds uint64,
	) (swapID common.Hash, err error)

	// RedeemSwap claims the locked funds using the secret preimage.
	RedeemSwap(swapID common.Hash, secret []byte) error

	// RefundSwap refunds the locked funds after the time lock expires.
	RefundSwap(swapID common.Hash) error

	// GetSwapStatus returns the current status of a swap.
	GetSwapStatus(swapID common.Hash) (SwapStatus, error)
}

// SwapStatus represents the state of an atomic swap.
type SwapStatus uint8

const (
	SwapStatusPending  SwapStatus = 0
	SwapStatusRedeemed SwapStatus = 1
	SwapStatusRefunded SwapStatus = 2
	SwapStatusExpired  SwapStatus = 3
)

// NoOpBridge is a placeholder atomic swap bridge that returns errors for all operations.
type NoOpBridge struct{}

func (b *NoOpBridge) InitiateSwap(sender, receiver common.Address, amount *big.Int,
	hashLock common.Hash, timeLockSeconds uint64) (common.Hash, error) {
	return common.Hash{}, ErrDEXNotEnabled
}

func (b *NoOpBridge) RedeemSwap(swapID common.Hash, secret []byte) error {
	return ErrDEXNotEnabled
}

func (b *NoOpBridge) RefundSwap(swapID common.Hash) error {
	return ErrDEXNotEnabled
}

func (b *NoOpBridge) GetSwapStatus(swapID common.Hash) (SwapStatus, error) {
	return SwapStatusPending, ErrDEXNotEnabled
}

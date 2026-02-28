// Copyright 2024 The go-probeum Authors
// This file is part of the go-probeum library.
//
// The go-probeum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-probeum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-probeum library. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"github.com/probeum/go-probeum/common"
	"github.com/probeum/go-probeum/crypto/dilithium"
)

// dilithiumVerify implements a precompiled contract for Dilithium signature verification.
// Input format: hash(32) || pubkey(1312) || sig(2420) = 3764 bytes total.
// Output: 32 bytes — left-padded 20-byte address if valid, zero bytes if invalid.
type dilithiumVerify struct{}

const (
	dilithiumVerifyInputLen = 32 + dilithium.PublicKeySize + dilithium.SignatureSize // 3764
	dilithiumVerifyGas      = 10000
)

func (c *dilithiumVerify) RequiredGas(input []byte) uint64 {
	return dilithiumVerifyGas
}

func (c *dilithiumVerify) Run(input []byte) ([]byte, error) {
	// Pad input if too short
	if len(input) < dilithiumVerifyInputLen {
		padded := make([]byte, dilithiumVerifyInputLen)
		copy(padded, input)
		input = padded
	}

	hash := input[:32]
	pubkeyBytes := input[32 : 32+dilithium.PublicKeySize]
	sigBytes := input[32+dilithium.PublicKeySize : dilithiumVerifyInputLen]

	pub, err := dilithium.UnmarshalPublicKey(pubkeyBytes)
	if err != nil {
		return make([]byte, 32), nil
	}

	if !dilithium.Verify(pub, hash, sigBytes) {
		return make([]byte, 32), nil
	}

	// Return the address derived from the public key, left-padded to 32 bytes
	addr := dilithium.PubkeyToAddress(pub)
	result := make([]byte, 32)
	copy(result[12:], addr[:])
	return result, nil
}

// DilithiumVerifyPrecompileAddress is the address of the Dilithium verify precompile.
var DilithiumVerifyPrecompileAddress = common.BytesToAddress([]byte{20})

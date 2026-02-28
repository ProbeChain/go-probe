// Copyright 2021 The go-probeum Authors
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

// Package probe re-exports cryptographic functions from the parent crypto package
// under the probe namespace.
package probe

import (
	"crypto/ecdsa"

	"github.com/probeum/go-probeum/crypto"
)

// Type aliases for ECDSA key types.
type PrivateKey = ecdsa.PrivateKey
type PublicKey = ecdsa.PublicKey

// Re-exported functions from the crypto package.
var (
	GenerateKey    = crypto.GenerateKey
	HexToECDSA     = crypto.HexToECDSA
	LoadECDSA      = crypto.LoadECDSA
	SaveECDSA      = crypto.SaveECDSA
	FromECDSAPub   = crypto.FromECDSAPub
	FromECDSA      = crypto.FromECDSA
	ToECDSA        = crypto.ToECDSA
	PubkeyToAddress = crypto.PubkeyToAddress
)

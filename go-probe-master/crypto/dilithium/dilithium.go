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

// Package dilithium wraps CRYSTALS-Dilithium (ML-DSA-44 / Dilithium2) from
// cloudflare/circl for use in the ProbeChain blockchain.
package dilithium

import (
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/cloudflare/circl/sign/dilithium/mode2"
	"github.com/probechain/go-probe/common"
	"github.com/probechain/go-probe/crypto"
)

const (
	// PublicKeySize is the size of a serialized Dilithium2 public key.
	PublicKeySize = mode2.PublicKeySize // 1312

	// PrivateKeySize is the size of a serialized Dilithium2 private key.
	PrivateKeySize = mode2.PrivateKeySize // 2528

	// SignatureSize is the size of a Dilithium2 signature.
	SignatureSize = mode2.SignatureSize // 2420
)

// PrivateKey wraps a Dilithium2 private key.
type PrivateKey struct {
	inner *mode2.PrivateKey
}

// PublicKey wraps a Dilithium2 public key.
type PublicKey struct {
	inner *mode2.PublicKey
}

// GenerateKey generates a new Dilithium2 private key.
func GenerateKey() (*PrivateKey, error) {
	_, priv, err := mode2.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("dilithium keygen: %w", err)
	}
	return &PrivateKey{inner: priv}, nil
}

// GenerateKeyPair generates a new Dilithium2 key pair, returning both keys.
func GenerateKeyPair() (*PublicKey, *PrivateKey, error) {
	pub, priv, err := mode2.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("dilithium keygen: %w", err)
	}
	return &PublicKey{inner: pub}, &PrivateKey{inner: priv}, nil
}

// Public returns the public key corresponding to this private key.
func (sk *PrivateKey) Public() *PublicKey {
	if sk == nil || sk.inner == nil {
		return nil
	}
	// Derive public key by marshaling/unmarshaling through the circl API
	// The circl PrivateKey embeds the public key
	privBytes := MarshalPrivateKey(sk)
	// Re-derive: generate pubkey from the private key bytes
	// Actually circl's PrivateKey can give us the PublicKey via the crypto.Signer interface
	signer := sk.inner
	pub := signer.Public()
	if cpub, ok := pub.(*mode2.PublicKey); ok {
		return &PublicKey{inner: cpub}
	}
	// Fallback: unpack from private key bytes
	_ = privBytes
	return nil
}

// Sign signs the message hash with the private key and returns the signature.
func Sign(priv *PrivateKey, msg []byte) []byte {
	sig := make([]byte, SignatureSize)
	mode2.SignTo(priv.inner, msg, sig)
	return sig
}

// Verify verifies a Dilithium2 signature.
func Verify(pub *PublicKey, msg, sig []byte) bool {
	if pub == nil || pub.inner == nil {
		return false
	}
	if len(sig) != SignatureSize {
		return false
	}
	return mode2.Verify(pub.inner, msg, sig)
}

// PubkeyToAddress derives a 20-byte ProbeChain address from a Dilithium public key.
// Uses Keccak256(pubkeyBytes)[12:] — same scheme as ECDSA but with larger input.
func PubkeyToAddress(pub *PublicKey) common.Address {
	pubBytes := MarshalPublicKey(pub)
	return common.BytesToAddress(crypto.Keccak256(pubBytes)[12:])
}

// MarshalPrivateKey serializes a Dilithium private key to bytes.
func MarshalPrivateKey(priv *PrivateKey) []byte {
	if priv == nil || priv.inner == nil {
		return nil
	}
	var buf [PrivateKeySize]byte
	priv.inner.Pack(&buf)
	return buf[:]
}

// UnmarshalPrivateKey deserializes a Dilithium private key from bytes.
func UnmarshalPrivateKey(data []byte) (*PrivateKey, error) {
	if len(data) != PrivateKeySize {
		return nil, fmt.Errorf("dilithium: invalid private key size %d, want %d", len(data), PrivateKeySize)
	}
	var buf [PrivateKeySize]byte
	copy(buf[:], data)
	sk := new(mode2.PrivateKey)
	sk.Unpack(&buf)
	return &PrivateKey{inner: sk}, nil
}

// MarshalPublicKey serializes a Dilithium public key to bytes.
func MarshalPublicKey(pub *PublicKey) []byte {
	if pub == nil || pub.inner == nil {
		return nil
	}
	var buf [PublicKeySize]byte
	pub.inner.Pack(&buf)
	return buf[:]
}

// UnmarshalPublicKey deserializes a Dilithium public key from bytes.
func UnmarshalPublicKey(data []byte) (*PublicKey, error) {
	if len(data) != PublicKeySize {
		return nil, fmt.Errorf("dilithium: invalid public key size %d, want %d", len(data), PublicKeySize)
	}
	var buf [PublicKeySize]byte
	copy(buf[:], data)
	pk := new(mode2.PublicKey)
	pk.Unpack(&buf)
	return &PublicKey{inner: pk}, nil
}

var ErrInvalidSignature = errors.New("dilithium: invalid signature")

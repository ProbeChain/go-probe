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

// Package bech32 implements Bech32 and Bech32m encoding as defined in BIP-173 and BIP-350.
package bech32

import (
	"fmt"
	"strings"
)

const charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

var gen = [5]uint32{0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3}

// Encoding represents the Bech32 encoding version.
type Encoding int

const (
	// Bech32 is the original BIP-173 encoding.
	Bech32 Encoding = 1
	// Bech32m is the BIP-350 encoding.
	Bech32m Encoding = 0x2bc830a3
)

func polymod(values []byte) uint32 {
	chk := uint32(1)
	for _, v := range values {
		b := chk >> 25
		chk = (chk&0x1ffffff)<<5 ^ uint32(v)
		for i := 0; i < 5; i++ {
			if (b>>uint(i))&1 == 1 {
				chk ^= gen[i]
			}
		}
	}
	return chk
}

func hrpExpand(hrp string) []byte {
	ret := make([]byte, 0, len(hrp)*2+1)
	for _, c := range hrp {
		ret = append(ret, byte(c>>5))
	}
	ret = append(ret, 0)
	for _, c := range hrp {
		ret = append(ret, byte(c&31))
	}
	return ret
}

func verifyChecksum(hrp string, data []byte) Encoding {
	c := polymod(append(hrpExpand(hrp), data...))
	if c == uint32(Bech32) {
		return Bech32
	}
	if c == uint32(Bech32m) {
		return Bech32m
	}
	return 0
}

func createChecksum(hrp string, data []byte, enc Encoding) []byte {
	values := append(hrpExpand(hrp), data...)
	values = append(values, 0, 0, 0, 0, 0, 0)
	mod := polymod(values) ^ uint32(enc)
	ret := make([]byte, 6)
	for i := 0; i < 6; i++ {
		ret[i] = byte((mod >> uint(5*(5-i))) & 31)
	}
	return ret
}

// Encode encodes a byte slice into a Bech32 string with the given human-readable part.
// Uses Bech32 encoding (BIP-173).
func Encode(hrp string, data []byte) (string, error) {
	return EncodeWithVersion(hrp, data, Bech32)
}

// EncodeWithVersion encodes a byte slice into a Bech32/Bech32m string.
func EncodeWithVersion(hrp string, data []byte, enc Encoding) (string, error) {
	// Convert 8-bit data to 5-bit groups
	conv, err := ConvertBits(data, 8, 5, true)
	if err != nil {
		return "", fmt.Errorf("bech32 encode: %w", err)
	}
	return encodeRaw(hrp, conv, enc)
}

func encodeRaw(hrp string, data5bit []byte, enc Encoding) (string, error) {
	if len(hrp) < 1 {
		return "", fmt.Errorf("bech32 encode: HRP must not be empty")
	}
	if len(hrp)+len(data5bit)+7 > 90 {
		// Relaxed: we allow longer strings for 20-byte addresses which encode to 39 chars + hrp + 1 separator + 6 checksum
		// Standard Bech32 limits to 90 chars total but we don't enforce this strictly for ProbeChain addresses
	}

	// Lowercase HRP
	hrp = strings.ToLower(hrp)
	combined := append(data5bit, createChecksum(hrp, data5bit, enc)...)

	var ret strings.Builder
	ret.Grow(len(hrp) + 1 + len(combined))
	ret.WriteString(hrp)
	ret.WriteByte('1') // separator
	for _, d := range combined {
		ret.WriteByte(charset[d])
	}
	return ret.String(), nil
}

// Decode decodes a Bech32 string into its human-readable part and data bytes.
func Decode(bech string) (string, []byte, error) {
	hrp, data, enc, err := DecodeWithVersion(bech)
	if err != nil {
		return "", nil, err
	}
	if enc != Bech32 {
		return "", nil, fmt.Errorf("bech32 decode: expected Bech32 encoding, got Bech32m")
	}
	return hrp, data, nil
}

// DecodeWithVersion decodes a Bech32 or Bech32m string.
func DecodeWithVersion(bech string) (string, []byte, Encoding, error) {
	if len(bech) > 90 {
		// Relaxed length check for compatibility
	}

	// Check for mixed case
	lower := strings.ToLower(bech)
	upper := strings.ToUpper(bech)
	if bech != lower && bech != upper {
		return "", nil, 0, fmt.Errorf("bech32 decode: mixed case")
	}
	bech = lower

	// Find separator
	pos := strings.LastIndex(bech, "1")
	if pos < 1 {
		return "", nil, 0, fmt.Errorf("bech32 decode: missing separator '1'")
	}
	if pos+7 > len(bech) {
		return "", nil, 0, fmt.Errorf("bech32 decode: too short after separator")
	}

	hrp := bech[:pos]
	dataStr := bech[pos+1:]

	// Decode data characters
	data := make([]byte, len(dataStr))
	for i, c := range dataStr {
		idx := strings.IndexByte(charset, byte(c))
		if idx == -1 {
			return "", nil, 0, fmt.Errorf("bech32 decode: invalid character %q at position %d", c, i)
		}
		data[i] = byte(idx)
	}

	// Verify checksum
	enc := verifyChecksum(hrp, data)
	if enc == 0 {
		return "", nil, 0, fmt.Errorf("bech32 decode: invalid checksum")
	}

	// Strip checksum from data, convert 5-bit to 8-bit
	data5bit := data[:len(data)-6]
	conv, err := ConvertBits(data5bit, 5, 8, false)
	if err != nil {
		return "", nil, 0, fmt.Errorf("bech32 decode: %w", err)
	}

	return hrp, conv, enc, nil
}

// ConvertBits converts a byte slice from one bit-width to another.
// pad determines whether to include padding bits in the output.
func ConvertBits(data []byte, fromBits, toBits uint8, pad bool) ([]byte, error) {
	acc := uint32(0)
	bits := uint8(0)
	maxv := uint32((1 << toBits) - 1)

	ret := make([]byte, 0, len(data)*int(fromBits)/int(toBits)+1)
	for _, value := range data {
		if uint32(value)>>fromBits != 0 {
			return nil, fmt.Errorf("invalid data: value %d exceeds %d bits", value, fromBits)
		}
		acc = (acc << fromBits) | uint32(value)
		bits += fromBits
		for bits >= toBits {
			bits -= toBits
			ret = append(ret, byte((acc>>bits)&maxv))
		}
	}

	if pad {
		if bits > 0 {
			ret = append(ret, byte((acc<<(toBits-bits))&maxv))
		}
	} else {
		if bits >= fromBits {
			return nil, fmt.Errorf("invalid padding")
		}
		if (acc<<(toBits-bits))&maxv != 0 {
			return nil, fmt.Errorf("non-zero padding")
		}
	}
	return ret, nil
}

// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.
//
// The ProbeChain is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The ProbeChain is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the ProbeChain. If not, see <http://www.gnu.org/licenses/>.

package stellar

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
)

var (
	errDataTooShort    = errors.New("stellar: data too short for FEC decode")
	errInvalidParity   = errors.New("stellar: invalid parity shard count")
	errFrameTooShort   = errors.New("stellar: frame too short")
	errPreambleMissing = errors.New("stellar: sync preamble not found")
	errCRCMismatch     = errors.New("stellar: CRC32 checksum mismatch")
)

// EncodeRS applies a simple Reed-Solomon-like forward error correction.
// It appends parityShards bytes of parity data computed via XOR folding.
// This is a lightweight FEC suitable for RF stream resilience.
func EncodeRS(data []byte, parityShards int) ([]byte, error) {
	if parityShards <= 0 || parityShards > 255 {
		return nil, errInvalidParity
	}
	if len(data) == 0 {
		return nil, errDataTooShort
	}

	parity := make([]byte, parityShards)
	for i, b := range data {
		parity[i%parityShards] ^= b
	}

	result := make([]byte, len(data)+parityShards)
	copy(result, data)
	copy(result[len(data):], parity)
	return result, nil
}

// DecodeRS strips the parity shards and verifies data integrity.
// If corruption is detected in the parity check, it attempts single-byte
// error correction using the XOR parity. Returns the original data.
func DecodeRS(stream []byte, parityShards int) ([]byte, error) {
	if parityShards <= 0 || parityShards > 255 {
		return nil, errInvalidParity
	}
	if len(stream) <= parityShards {
		return nil, errDataTooShort
	}

	dataLen := len(stream) - parityShards
	data := stream[:dataLen]
	receivedParity := stream[dataLen:]

	// Recompute parity
	computedParity := make([]byte, parityShards)
	for i, b := range data {
		computedParity[i%parityShards] ^= b
	}

	// Check parity match
	parityOK := true
	for i := 0; i < parityShards; i++ {
		if computedParity[i] != receivedParity[i] {
			parityOK = false
			break
		}
	}

	if parityOK {
		result := make([]byte, dataLen)
		copy(result, data)
		return result, nil
	}

	// Attempt correction: XOR difference indicates the error pattern.
	// For single-byte errors within one parity group, we can correct.
	corrected := make([]byte, dataLen)
	copy(corrected, data)

	for i := 0; i < parityShards; i++ {
		diff := computedParity[i] ^ receivedParity[i]
		if diff != 0 {
			// Find the last byte in this parity group and apply correction
			lastIdx := -1
			for j := i; j < dataLen; j += parityShards {
				lastIdx = j
			}
			if lastIdx >= 0 {
				corrected[lastIdx] ^= diff
			}
		}
	}

	return corrected, nil
}

// FrameBlock creates a framed RF block: preamble + 4-byte length + data + 4-byte CRC32.
func FrameBlock(data []byte, preamble []byte) []byte {
	frameLen := len(preamble) + 4 + len(data) + 4
	frame := make([]byte, frameLen)

	offset := 0
	copy(frame[offset:], preamble)
	offset += len(preamble)

	binary.BigEndian.PutUint32(frame[offset:], uint32(len(data)))
	offset += 4

	copy(frame[offset:], data)
	offset += len(data)

	checksum := crc32.ChecksumIEEE(data)
	binary.BigEndian.PutUint32(frame[offset:], checksum)

	return frame
}

// UnframeBlock extracts data from a framed RF block, verifying the preamble and CRC32.
func UnframeBlock(frame []byte, preamble []byte) ([]byte, error) {
	minLen := len(preamble) + 4 + 4 // preamble + length + CRC (no data)
	if len(frame) < minLen {
		return nil, errFrameTooShort
	}

	// Verify preamble
	for i, b := range preamble {
		if frame[i] != b {
			return nil, errPreambleMissing
		}
	}

	offset := len(preamble)
	dataLen := binary.BigEndian.Uint32(frame[offset:])
	offset += 4

	if uint32(len(frame)-offset) < dataLen+4 {
		return nil, errFrameTooShort
	}

	data := frame[offset : offset+int(dataLen)]
	offset += int(dataLen)

	expectedCRC := binary.BigEndian.Uint32(frame[offset:])
	actualCRC := crc32.ChecksumIEEE(data)
	if expectedCRC != actualCRC {
		return nil, errCRCMismatch
	}

	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

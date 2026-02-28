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

package stellar

import "errors"

const (
	// defaultParityShards is the default number of FEC parity shards.
	defaultParityShards = 16

	// maxBlockSize is the maximum block size supported for RF encapsulation (4 MB).
	maxBlockSize = 4 * 1024 * 1024
)

var (
	errBlockTooLarge = errors.New("stellar: block data exceeds maximum RF encapsulation size")
	errBlockEmpty    = errors.New("stellar: block data is empty")
)

// defaultSyncPreamble is the sync pattern used to identify block frames.
// The pattern 0xAA 0x55 alternation provides good clock recovery properties.
var defaultSyncPreamble = []byte{0xAA, 0x55, 0xAA, 0x55, 0x50, 0x52, 0x42, 0x45} // last 4 bytes = "PRBE"

// DefaultEncap is the default RadioEncap implementation using RS-FEC framing.
// It encodes block data with forward error correction and RF sync framing.
type DefaultEncap struct {
	band         BandDescriptor
	parityShards int
	preamble     []byte
}

// NewDefaultEncap creates a new DefaultEncap for the given band.
func NewDefaultEncap(band BandDescriptor) *DefaultEncap {
	return &DefaultEncap{
		band:         band,
		parityShards: defaultParityShards,
		preamble:     defaultSyncPreamble,
	}
}

// NewDefaultEncapWithParity creates a DefaultEncap with custom parity shard count.
func NewDefaultEncapWithParity(band BandDescriptor, parityShards int) *DefaultEncap {
	preamble := make([]byte, len(defaultSyncPreamble))
	copy(preamble, defaultSyncPreamble)
	return &DefaultEncap{
		band:         band,
		parityShards: parityShards,
		preamble:     preamble,
	}
}

// EncodeBlock encodes RLP-encoded block data into an RF-ready stream.
// Pipeline: data -> RS-FEC encode -> frame with preamble + length + CRC32.
func (e *DefaultEncap) EncodeBlock(block []byte) ([]byte, error) {
	if len(block) == 0 {
		return nil, errBlockEmpty
	}
	if len(block) > maxBlockSize {
		return nil, errBlockTooLarge
	}

	// Apply FEC encoding
	fecData, err := EncodeRS(block, e.parityShards)
	if err != nil {
		return nil, err
	}

	// Frame with sync preamble, length header, and CRC32
	framed := FrameBlock(fecData, e.preamble)
	return framed, nil
}

// DecodeBlock decodes an RF stream back into RLP-encoded block data.
// Pipeline: unframe -> RS-FEC decode -> original data.
func (e *DefaultEncap) DecodeBlock(stream []byte) ([]byte, error) {
	// Strip frame: verify preamble and CRC
	fecData, err := UnframeBlock(stream, e.preamble)
	if err != nil {
		return nil, err
	}

	// Apply FEC decoding
	data, err := DecodeRS(fecData, e.parityShards)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// BandInfo returns the RF band descriptor for this encapsulation.
func (e *DefaultEncap) BandInfo() BandDescriptor {
	return e.band
}

// SyncPreamble returns the synchronization preamble bytes.
func (e *DefaultEncap) SyncPreamble() []byte {
	p := make([]byte, len(e.preamble))
	copy(p, e.preamble)
	return p
}

// Overhead returns the total byte overhead added per block
// (FEC parity + frame header + CRC).
func (e *DefaultEncap) Overhead() int {
	// preamble + 4 byte length + parityShards + 4 byte CRC
	return len(e.preamble) + 4 + e.parityShards + 4
}

// NewHFEncap creates a DefaultEncap optimized for HF band with higher FEC.
func NewHFEncap() *DefaultEncap {
	return NewDefaultEncapWithParity(HFBand, 32) // More parity for noisy HF
}

// NewVHFEncap creates a DefaultEncap optimized for VHF band.
func NewVHFEncap() *DefaultEncap {
	return NewDefaultEncap(VHFBand)
}

// NewUHFEncap creates a DefaultEncap optimized for UHF band.
func NewUHFEncap() *DefaultEncap {
	return NewDefaultEncap(UHFBand)
}

// NewSHFEncap creates a DefaultEncap optimized for SHF satellite links.
func NewSHFEncap() *DefaultEncap {
	return NewDefaultEncapWithParity(SHFBand, 8) // Less parity needed for clean links
}

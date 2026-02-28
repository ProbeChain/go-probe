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

// Package stellar provides the RadioEncap interface for encoding block data as
// raw RF streams, supporting Rydberg atomic receivers across MHz-THz bands.
// This enables Stellar-Class Resilience for off-grid and interstellar scenarios.
package stellar

// RadioEncap defines the interface for encoding and decoding block data
// as raw RF-compatible byte streams.
type RadioEncap interface {
	// EncodeBlock encodes RLP-encoded block data into an RF-ready stream
	// with FEC, framing, and modulation-appropriate encoding.
	EncodeBlock(block []byte) ([]byte, error)

	// DecodeBlock decodes an RF stream back into RLP-encoded block data,
	// applying FEC error correction.
	DecodeBlock(stream []byte) ([]byte, error)

	// BandInfo returns the RF band descriptor for this encapsulation.
	BandInfo() BandDescriptor

	// SyncPreamble returns the synchronization preamble bytes used to
	// identify the start of a block frame in the RF stream.
	SyncPreamble() []byte
}

// BandDescriptor describes an RF frequency band and its parameters.
type BandDescriptor struct {
	Name      string // Band name: "HF", "VHF", "UHF", "SHF", "EHF", "THz"
	MinFreqHz uint64 // Minimum frequency in Hz
	MaxFreqHz uint64 // Maximum frequency in Hz
	ChannelBW uint64 // Channel bandwidth in Hz
	ModScheme string // Modulation scheme: "BPSK", "QPSK", "8PSK", "16QAM"
}

// RydbergCapability describes the capabilities of a Rydberg atomic receiver node.
type RydbergCapability struct {
	Bands          []BandDescriptor // Supported RF bands
	SensitivityDBm float64          // Receiver sensitivity in dBm
	AtomicRef      bool             // Whether the receiver has atomic clock reference
	MaxDataRate    uint64           // Maximum data rate in bits per second
}

// Standard band presets for common RF allocations.
var (
	// HFBand covers 3-30 MHz (High Frequency), used for long-range skywave propagation.
	HFBand = BandDescriptor{
		Name:      "HF",
		MinFreqHz: 3_000_000,
		MaxFreqHz: 30_000_000,
		ChannelBW: 3_000,
		ModScheme: "BPSK",
	}

	// VHFBand covers 30-300 MHz (Very High Frequency), used for line-of-sight comms.
	VHFBand = BandDescriptor{
		Name:      "VHF",
		MinFreqHz: 30_000_000,
		MaxFreqHz: 300_000_000,
		ChannelBW: 25_000,
		ModScheme: "QPSK",
	}

	// UHFBand covers 300 MHz - 3 GHz (Ultra High Frequency).
	UHFBand = BandDescriptor{
		Name:      "UHF",
		MinFreqHz: 300_000_000,
		MaxFreqHz: 3_000_000_000,
		ChannelBW: 200_000,
		ModScheme: "QPSK",
	}

	// SHFBand covers 3-30 GHz (Super High Frequency), used for satellite links.
	SHFBand = BandDescriptor{
		Name:      "SHF",
		MinFreqHz: 3_000_000_000,
		MaxFreqHz: 30_000_000_000,
		ChannelBW: 1_000_000,
		ModScheme: "8PSK",
	}

	// EHFBand covers 30-300 GHz (Extremely High Frequency), millimeter wave.
	EHFBand = BandDescriptor{
		Name:      "EHF",
		MinFreqHz: 30_000_000_000,
		MaxFreqHz: 300_000_000_000,
		ChannelBW: 10_000_000,
		ModScheme: "16QAM",
	}

	// THzBand covers 300 GHz - 3 THz (Terahertz), experimental Rydberg-only band.
	THzBand = BandDescriptor{
		Name:      "THz",
		MinFreqHz: 300_000_000_000,
		MaxFreqHz: 3_000_000_000_000,
		ChannelBW: 100_000_000,
		ModScheme: "BPSK",
	}
)

// AllBands returns all standard band presets.
func AllBands() []BandDescriptor {
	return []BandDescriptor{HFBand, VHFBand, UHFBand, SHFBand, EHFBand, THzBand}
}

// DefaultRydbergCapability returns a baseline Rydberg receiver capability
// covering HF through EHF bands with atomic clock reference.
func DefaultRydbergCapability() *RydbergCapability {
	return &RydbergCapability{
		Bands:          []BandDescriptor{HFBand, VHFBand, UHFBand, SHFBand, EHFBand},
		SensitivityDBm: -140.0, // Rydberg receivers achieve ~-140 dBm sensitivity
		AtomicRef:      true,
		MaxDataRate:    1_000_000, // 1 Mbps baseline
	}
}

// FullSpectrumRydbergCapability returns a Rydberg receiver that includes THz band.
func FullSpectrumRydbergCapability() *RydbergCapability {
	return &RydbergCapability{
		Bands:          AllBands(),
		SensitivityDBm: -150.0,
		AtomicRef:      true,
		MaxDataRate:    10_000_000, // 10 Mbps with THz
	}
}

// BandByName looks up a standard band preset by name. Returns zero value if not found.
func BandByName(name string) (BandDescriptor, bool) {
	for _, b := range AllBands() {
		if b.Name == name {
			return b, true
		}
	}
	return BandDescriptor{}, false
}

// DataRateForBand estimates the achievable data rate for a given band descriptor
// based on channel bandwidth and modulation scheme. Returns bits per second.
func DataRateForBand(band BandDescriptor) uint64 {
	var bitsPerSymbol uint64
	switch band.ModScheme {
	case "BPSK":
		bitsPerSymbol = 1
	case "QPSK":
		bitsPerSymbol = 2
	case "8PSK":
		bitsPerSymbol = 3
	case "16QAM":
		bitsPerSymbol = 4
	default:
		bitsPerSymbol = 1
	}
	// Nyquist rate: symbols/sec ≈ bandwidth, data rate = symbols/sec * bits/symbol
	// Apply 0.5 efficiency factor for FEC and framing overhead
	return band.ChannelBW * bitsPerSymbol / 2
}

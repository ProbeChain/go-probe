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

// Package atomic provides AtomicTimestamp, a high-precision time anchor for
// Stellar-Class Resilience. It embeds clock source metadata and uncertainty
// bounds for interstellar-scale or off-grid block time ordering.
package atomic

import (
	"encoding/binary"
	"errors"
	"math"
	"time"
)

// ClockSource identifies the time synchronization source.
type ClockSource uint8

const (
	ClockSourceSystem  ClockSource = 0 // OS system clock
	ClockSourceNTP     ClockSource = 1 // Network Time Protocol
	ClockSourcePTP     ClockSource = 2 // Precision Time Protocol (IEEE 1588)
	ClockSourceGNSS    ClockSource = 3 // GNSS (GPS/Galileo/GLONASS) receiver
	ClockSourceRydberg ClockSource = 4 // Rydberg atomic clock reference
)

// AtomicTimestampSize is the fixed byte size of an encoded AtomicTimestamp.
const AtomicTimestampSize = 17

// speedOfLight in meters per second (vacuum).
const speedOfLight = 299_792_458

var (
	errTimestampTooShort = errors.New("atomic: encoded timestamp too short")
	errTimestampTooLong  = errors.New("atomic: encoded timestamp too long")
)

// AtomicTimestamp represents a high-precision timestamp with clock source
// metadata and uncertainty bounds. It is designed to be embedded in block
// headers as a 17-byte optional field.
type AtomicTimestamp struct {
	Seconds     uint64      // Unix seconds (TAI-based)
	Nanoseconds uint32      // Sub-second precision (0-999999999)
	ClockSource ClockSource // Source of time synchronization
	Uncertainty uint32      // Estimated uncertainty in nanoseconds
}

// Encode serializes the AtomicTimestamp into a fixed 17-byte representation.
// Layout: [8 bytes seconds][4 bytes nanoseconds][1 byte source][4 bytes uncertainty]
func (t *AtomicTimestamp) Encode() []byte {
	buf := make([]byte, AtomicTimestampSize)
	binary.BigEndian.PutUint64(buf[0:8], t.Seconds)
	binary.BigEndian.PutUint32(buf[8:12], t.Nanoseconds)
	buf[12] = byte(t.ClockSource)
	binary.BigEndian.PutUint32(buf[13:17], t.Uncertainty)
	return buf
}

// DecodeAtomicTimestamp decodes a 17-byte representation into an AtomicTimestamp.
func DecodeAtomicTimestamp(data []byte) (*AtomicTimestamp, error) {
	if len(data) < AtomicTimestampSize {
		return nil, errTimestampTooShort
	}
	if len(data) > AtomicTimestampSize {
		return nil, errTimestampTooLong
	}
	return &AtomicTimestamp{
		Seconds:     binary.BigEndian.Uint64(data[0:8]),
		Nanoseconds: binary.BigEndian.Uint32(data[8:12]),
		ClockSource: ClockSource(data[12]),
		Uncertainty: binary.BigEndian.Uint32(data[13:17]),
	}, nil
}

// Now creates an AtomicTimestamp from the current system time with the
// specified clock source. The uncertainty is set based on the source type.
func Now(source ClockSource) *AtomicTimestamp {
	now := time.Now()
	return &AtomicTimestamp{
		Seconds:     uint64(now.Unix()),
		Nanoseconds: uint32(now.Nanosecond()),
		ClockSource: source,
		Uncertainty: defaultUncertainty(source),
	}
}

// defaultUncertainty returns the typical uncertainty in nanoseconds
// for a given clock source.
func defaultUncertainty(source ClockSource) uint32 {
	switch source {
	case ClockSourceRydberg:
		return 1 // ~1 ns, atomic precision
	case ClockSourceGNSS:
		return 100 // ~100 ns from GPS
	case ClockSourcePTP:
		return 1_000 // ~1 μs
	case ClockSourceNTP:
		return 10_000_000 // ~10 ms
	default:
		return 100_000_000 // ~100 ms for system clock
	}
}

// Compare compares two AtomicTimestamps. Returns:
//
//	-1 if t < other
//	 0 if t == other
//	+1 if t > other
func (t *AtomicTimestamp) Compare(other *AtomicTimestamp) int {
	if t.Seconds < other.Seconds {
		return -1
	}
	if t.Seconds > other.Seconds {
		return 1
	}
	if t.Nanoseconds < other.Nanoseconds {
		return -1
	}
	if t.Nanoseconds > other.Nanoseconds {
		return 1
	}
	return 0
}

// LightDelay calculates the one-way light propagation delay in nanoseconds
// for a given distance in meters. Useful for interstellar time corrections.
func LightDelay(distanceMeters float64) uint64 {
	delaySeconds := distanceMeters / float64(speedOfLight)
	return uint64(delaySeconds * 1e9)
}

// IsHighPrecision returns true if the timestamp comes from a high-precision
// source (PTP, GNSS, or Rydberg) with uncertainty under 10 μs.
func (t *AtomicTimestamp) IsHighPrecision() bool {
	return t.Uncertainty <= 10_000 // 10 μs threshold
}

// IsRydbergSynced returns true if the timestamp was derived from a Rydberg
// atomic clock reference.
func (t *AtomicTimestamp) IsRydbergSynced() bool {
	return t.ClockSource == ClockSourceRydberg
}

// ToTime converts the AtomicTimestamp to a standard Go time.Time.
func (t *AtomicTimestamp) ToTime() time.Time {
	return time.Unix(int64(t.Seconds), int64(t.Nanoseconds))
}

// FromTime creates an AtomicTimestamp from a Go time.Time with the given source.
func FromTime(tt time.Time, source ClockSource) *AtomicTimestamp {
	return &AtomicTimestamp{
		Seconds:     uint64(tt.Unix()),
		Nanoseconds: uint32(tt.Nanosecond()),
		ClockSource: source,
		Uncertainty: defaultUncertainty(source),
	}
}

// DurationBetween returns the time difference in nanoseconds between two timestamps.
// Returns a signed value (negative if other is later).
func (t *AtomicTimestamp) DurationBetween(other *AtomicTimestamp) int64 {
	secDiff := int64(t.Seconds) - int64(other.Seconds)
	nsDiff := int64(t.Nanoseconds) - int64(other.Nanoseconds)
	return secDiff*1e9 + nsDiff
}

// WithinUncertainty returns true if two timestamps overlap within their
// combined uncertainty bounds.
func (t *AtomicTimestamp) WithinUncertainty(other *AtomicTimestamp) bool {
	diff := t.DurationBetween(other)
	if diff < 0 {
		diff = -diff
	}
	combinedUncertainty := int64(t.Uncertainty) + int64(other.Uncertainty)
	return diff <= combinedUncertainty
}

// SourceName returns a human-readable name for the clock source.
func (s ClockSource) String() string {
	switch s {
	case ClockSourceSystem:
		return "System"
	case ClockSourceNTP:
		return "NTP"
	case ClockSourcePTP:
		return "PTP"
	case ClockSourceGNSS:
		return "GNSS"
	case ClockSourceRydberg:
		return "Rydberg"
	default:
		return "Unknown"
	}
}

// LightDelayAU returns the light delay in nanoseconds for a distance
// specified in Astronomical Units (1 AU ≈ 149,597,870,700 meters).
func LightDelayAU(au float64) uint64 {
	meters := au * 149_597_870_700.0
	return LightDelay(meters)
}

// LightDelayLY returns the light delay in nanoseconds for a distance
// specified in light-years (1 ly ≈ 9.461×10^15 meters).
// Returns math.MaxUint64 if the result would overflow.
func LightDelayLY(ly float64) uint64 {
	meters := ly * 9.461e15
	delayNs := (meters / float64(speedOfLight)) * 1e9
	if delayNs > float64(math.MaxUint64) {
		return math.MaxUint64
	}
	return uint64(delayNs)
}

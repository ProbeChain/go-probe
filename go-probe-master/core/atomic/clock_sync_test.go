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

package atomic

import (
	"bytes"
	"testing"
	"time"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	ts := &AtomicTimestamp{
		Seconds:     1700000000,
		Nanoseconds: 123456789,
		ClockSource: ClockSourceGNSS,
		Uncertainty: 100,
	}

	encoded := ts.Encode()
	if len(encoded) != AtomicTimestampSize {
		t.Fatalf("encoded size = %d, want %d", len(encoded), AtomicTimestampSize)
	}

	decoded, err := DecodeAtomicTimestamp(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if decoded.Seconds != ts.Seconds {
		t.Errorf("Seconds = %d, want %d", decoded.Seconds, ts.Seconds)
	}
	if decoded.Nanoseconds != ts.Nanoseconds {
		t.Errorf("Nanoseconds = %d, want %d", decoded.Nanoseconds, ts.Nanoseconds)
	}
	if decoded.ClockSource != ts.ClockSource {
		t.Errorf("ClockSource = %d, want %d", decoded.ClockSource, ts.ClockSource)
	}
	if decoded.Uncertainty != ts.Uncertainty {
		t.Errorf("Uncertainty = %d, want %d", decoded.Uncertainty, ts.Uncertainty)
	}
}

func TestDecodeTooShort(t *testing.T) {
	_, err := DecodeAtomicTimestamp([]byte{0x01, 0x02})
	if err != errTimestampTooShort {
		t.Fatalf("expected errTimestampTooShort, got %v", err)
	}
}

func TestDecodeTooLong(t *testing.T) {
	_, err := DecodeAtomicTimestamp(make([]byte, 18))
	if err != errTimestampTooLong {
		t.Fatalf("expected errTimestampTooLong, got %v", err)
	}
}

func TestNow(t *testing.T) {
	before := time.Now().Unix()
	ts := Now(ClockSourceSystem)
	after := time.Now().Unix()

	if ts.Seconds < uint64(before) || ts.Seconds > uint64(after) {
		t.Errorf("Now() seconds %d not in range [%d, %d]", ts.Seconds, before, after)
	}
	if ts.ClockSource != ClockSourceSystem {
		t.Errorf("ClockSource = %d, want System(0)", ts.ClockSource)
	}
}

func TestCompare(t *testing.T) {
	a := &AtomicTimestamp{Seconds: 100, Nanoseconds: 500}
	b := &AtomicTimestamp{Seconds: 100, Nanoseconds: 600}
	c := &AtomicTimestamp{Seconds: 101, Nanoseconds: 0}

	if a.Compare(b) != -1 {
		t.Error("a should be less than b")
	}
	if b.Compare(a) != 1 {
		t.Error("b should be greater than a")
	}
	if a.Compare(a) != 0 {
		t.Error("a should equal a")
	}
	if a.Compare(c) != -1 {
		t.Error("a should be less than c")
	}
}

func TestLightDelay(t *testing.T) {
	// Light travels ~1 meter in ~3.33 ns
	delay := LightDelay(299_792_458) // 1 second of light travel
	if delay < 999_000_000 || delay > 1_001_000_000 {
		t.Errorf("1 light-second delay = %d ns, expected ~1e9", delay)
	}
}

func TestLightDelayAU(t *testing.T) {
	// 1 AU ≈ 499 seconds of light travel
	delay := LightDelayAU(1.0)
	expectedNs := uint64(499 * 1e9) // approximately
	if delay < expectedNs-2e9 || delay > expectedNs+2e9 {
		t.Errorf("1 AU delay = %d ns, expected ~%d", delay, expectedNs)
	}
}

func TestIsHighPrecision(t *testing.T) {
	highPrec := &AtomicTimestamp{ClockSource: ClockSourceRydberg, Uncertainty: 1}
	if !highPrec.IsHighPrecision() {
		t.Error("Rydberg source should be high precision")
	}

	lowPrec := &AtomicTimestamp{ClockSource: ClockSourceSystem, Uncertainty: 100_000_000}
	if lowPrec.IsHighPrecision() {
		t.Error("System source should not be high precision")
	}
}

func TestIsRydbergSynced(t *testing.T) {
	ts := &AtomicTimestamp{ClockSource: ClockSourceRydberg}
	if !ts.IsRydbergSynced() {
		t.Error("should be Rydberg synced")
	}
	ts2 := &AtomicTimestamp{ClockSource: ClockSourceNTP}
	if ts2.IsRydbergSynced() {
		t.Error("should not be Rydberg synced")
	}
}

func TestWithinUncertainty(t *testing.T) {
	a := &AtomicTimestamp{Seconds: 100, Nanoseconds: 0, Uncertainty: 1000}
	b := &AtomicTimestamp{Seconds: 100, Nanoseconds: 500, Uncertainty: 1000}
	if !a.WithinUncertainty(b) {
		t.Error("timestamps 500ns apart with 2000ns combined uncertainty should overlap")
	}

	c := &AtomicTimestamp{Seconds: 100, Nanoseconds: 0, Uncertainty: 10}
	d := &AtomicTimestamp{Seconds: 100, Nanoseconds: 100, Uncertainty: 10}
	if c.WithinUncertainty(d) {
		t.Error("timestamps 100ns apart with 20ns combined uncertainty should not overlap")
	}
}

func TestClockSourceString(t *testing.T) {
	tests := []struct {
		source ClockSource
		want   string
	}{
		{ClockSourceSystem, "System"},
		{ClockSourceNTP, "NTP"},
		{ClockSourcePTP, "PTP"},
		{ClockSourceGNSS, "GNSS"},
		{ClockSourceRydberg, "Rydberg"},
		{ClockSource(99), "Unknown"},
	}
	for _, tt := range tests {
		if got := tt.source.String(); got != tt.want {
			t.Errorf("ClockSource(%d).String() = %s, want %s", tt.source, got, tt.want)
		}
	}
}

func TestEncodeProducesConsistentBytes(t *testing.T) {
	ts := &AtomicTimestamp{
		Seconds:     1700000000,
		Nanoseconds: 0,
		ClockSource: ClockSourceRydberg,
		Uncertainty: 1,
	}
	enc1 := ts.Encode()
	enc2 := ts.Encode()
	if !bytes.Equal(enc1, enc2) {
		t.Error("Encode should produce identical bytes for same timestamp")
	}
}

func TestToTimeAndFromTime(t *testing.T) {
	original := time.Date(2024, 6, 15, 12, 0, 0, 500_000_000, time.UTC)
	ts := FromTime(original, ClockSourcePTP)
	recovered := ts.ToTime()

	if recovered.Unix() != original.Unix() {
		t.Errorf("seconds mismatch: got %d, want %d", recovered.Unix(), original.Unix())
	}
	if recovered.Nanosecond() != original.Nanosecond() {
		t.Errorf("nanoseconds mismatch: got %d, want %d", recovered.Nanosecond(), original.Nanosecond())
	}
}

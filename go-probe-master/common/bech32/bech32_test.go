package bech32

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
)

func TestEncodeDecodeRoundtrip(t *testing.T) {
	tests := []struct {
		name string
		hrp  string
		data []byte
	}{
		{
			name: "20-byte address",
			hrp:  "pro",
			data: hexDecode("0000000000000000000000000000000000000101"),
		},
		{
			name: "zero address",
			hrp:  "pro",
			data: hexDecode("0000000000000000000000000000000000000000"),
		},
		{
			name: "all ones",
			hrp:  "pro",
			data: hexDecode("ffffffffffffffffffffffffffffffffffffffff"),
		},
		{
			name: "typical address",
			hrp:  "pro",
			data: hexDecode("5aAeb6053ba3EEdb3A6467688c0F67dB869d0D20"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := Encode(tt.hrp, tt.data)
			if err != nil {
				t.Fatalf("Encode() error: %v", err)
			}

			// Must start with hrp + "1"
			if encoded[:len(tt.hrp)+1] != tt.hrp+"1" {
				t.Errorf("encoded doesn't start with %s1: got %s", tt.hrp, encoded)
			}

			hrp, decoded, err := Decode(encoded)
			if err != nil {
				t.Fatalf("Decode() error: %v", err)
			}

			if hrp != tt.hrp {
				t.Errorf("HRP mismatch: got %q, want %q", hrp, tt.hrp)
			}

			if !bytes.Equal(decoded, tt.data) {
				t.Errorf("data mismatch:\n  got:  %x\n  want: %x", decoded, tt.data)
			}
		})
	}
}

func TestDecodeInvalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"no separator", "proqpzry9x8gf2tvdw0s3jn54khce6mua7l"},
		{"empty hrp", "1qpzry9x8gf2tvdw0s3jn54khce6mua7l"},
		{"invalid char", "pro1qpzry9x8gf2tvdw0s3jn54khce6mua7!"},
		{"mixed case", "Pro1qpzry9x8gf2tvdw0s3jn54khce6mua7l"},
		{"too short after sep", "pro1abcde"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := Decode(tt.input)
			if err == nil {
				t.Errorf("expected error for input %q", tt.input)
			}
		})
	}
}

func TestConvertBits(t *testing.T) {
	// 8-bit to 5-bit and back
	data := []byte{0xff, 0x00, 0xab}
	conv5, err := ConvertBits(data, 8, 5, true)
	if err != nil {
		t.Fatalf("ConvertBits 8→5: %v", err)
	}
	conv8, err := ConvertBits(conv5, 5, 8, false)
	if err != nil {
		t.Fatalf("ConvertBits 5→8: %v", err)
	}
	if !bytes.Equal(conv8, data) {
		t.Errorf("roundtrip failed: got %x, want %x", conv8, data)
	}
}

func TestKnownVector(t *testing.T) {
	// Encode a known address and verify it decodes back
	addr := hexDecode("0000000000000000000000000000000000000101")
	encoded, err := Encode("pro", addr)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	t.Logf("pro1 encoding of 0x...0101: %s", encoded)

	hrp, decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if hrp != "pro" {
		t.Errorf("hrp = %q, want pro", hrp)
	}
	if !bytes.Equal(decoded, addr) {
		t.Errorf("decoded = %x, want %x", decoded, addr)
	}
}

func TestCaseInsensitiveDecode(t *testing.T) {
	addr := hexDecode("5aAeb6053ba3EEdb3A6467688c0F67dB869d0D20")
	encoded, err := Encode("pro", addr)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	// Should decode the fully uppercase version too
	upper := strings.ToUpper(encoded)
	hrp, decoded, err := Decode(upper)
	if err != nil {
		t.Fatalf("Decode uppercase: %v", err)
	}
	if hrp != "pro" {
		t.Errorf("hrp = %q, want pro", hrp)
	}
	if !bytes.Equal(decoded, addr) {
		t.Errorf("decoded = %x, want %x", decoded, addr)
	}
}

func hexDecode(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

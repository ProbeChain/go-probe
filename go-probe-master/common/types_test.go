// Copyright 2015 The go-probeum Authors
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

package common

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/imroc/biu"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

func TestBytesConversion(t *testing.T) {
	bytes := []byte{5}
	hash := BytesToHash(bytes)

	var exp Hash
	exp[31] = 5

	if hash != exp {
		t.Errorf("expected %x got %x", exp, hash)
	}
}

func TestIsHexAddress(t *testing.T) {
	tests := []struct {
		str string
		exp bool
	}{
		{"0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed", true},
		{"5aaeb6053f3e94c9b9a09f33669435e7ef1beaed", true},
		{"0X5aaeb6053f3e94c9b9a09f33669435e7ef1beaed", true},
		{"0XAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", true},
		{"0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", true},
		{"0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed1", false},
		{"0x5aaeb6053f3e94c9b9a09f33669435e7ef1beae", false},
		{"5aaeb6053f3e94c9b9a09f33669435e7ef1beaed11", false},
		{"0xxaaeb6053f3e94c9b9a09f33669435e7ef1beaed", false},
	}

	for _, test := range tests {
		if result := IsHexAddress(test.str); result != test.exp {
			t.Errorf("IsHexAddress(%s) == %v; expected %v",
				test.str, result, test.exp)
		}
	}
}

func TestHashJsonValidation(t *testing.T) {
	var tests = []struct {
		Prefix string
		Size   int
		Error  string
	}{
		{"", 62, "json: cannot unmarshal hex string without 0x prefix into Go value of type common.Hash"},
		{"0x", 66, "hex string has length 66, want 64 for common.Hash"},
		{"0x", 63, "json: cannot unmarshal hex string of odd length into Go value of type common.Hash"},
		{"0x", 0, "hex string has length 0, want 64 for common.Hash"},
		{"0x", 64, ""},
		{"0X", 64, ""},
	}
	for _, test := range tests {
		input := `"` + test.Prefix + strings.Repeat("0", test.Size) + `"`
		var v Hash
		err := json.Unmarshal([]byte(input), &v)
		if err == nil {
			if test.Error != "" {
				t.Errorf("%s: error mismatch: have nil, want %q", input, test.Error)
			}
		} else {
			if err.Error() != test.Error {
				t.Errorf("%s: error mismatch: have %q, want %q", input, err, test.Error)
			}
		}
	}
}

func TestAddressUnmarshalJSON(t *testing.T) {
	var tests = []struct {
		Input     string
		ShouldErr bool
		Output    *big.Int
	}{
		{"", true, nil},
		{`""`, true, nil},
		{`"0x"`, true, nil},
		{`"0x00"`, true, nil},
		{`"0xG000000000000000000000000000000000000000"`, true, nil},
		{`"0x0000000000000000000000000000000000000000"`, false, big.NewInt(0)},
		{`"0x0000000000000000000000000000000000000010"`, false, big.NewInt(16)},
	}
	for i, test := range tests {
		var v Address
		err := json.Unmarshal([]byte(test.Input), &v)
		if err != nil && !test.ShouldErr {
			t.Errorf("test #%d: unexpected error: %v", i, err)
		}
		if err == nil {
			if test.ShouldErr {
				t.Errorf("test #%d: expected error, got none", i)
			}
			if got := new(big.Int).SetBytes(v.Bytes()); got.Cmp(test.Output) != 0 {
				t.Errorf("test #%d: address mismatch: have %v, want %v", i, got, test.Output)
			}
		}
	}
}

func TestAddressHexChecksum(t *testing.T) {
	// Hex() now returns Bech32. Verify addresses encode/decode correctly.
	var tests = []struct {
		Input string
	}{
		{"0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"},
		{"0xfb6916095ca1df60bb79ce92ce3ea74c37c5d359"},
		{"0xdbf03b407c01e7cd3cbea99509d93f8dddc8c6fb"},
		{"0xd1220a0cf47c7b9be7a2e6ba89f429762e7b9adb"},
		{"0xa"},
		{"0x0a"},
		{"0x00a"},
		{"0x000000000000000000000000000000000000000a"},
	}
	for i, test := range tests {
		addr := HexToAddress(test.Input)
		bech32Str := addr.Hex()
		// Output must start with "pro1"
		if !strings.HasPrefix(bech32Str, "pro1") {
			t.Errorf("test #%d: expected pro1 prefix, got %s", i, bech32Str)
		}
		// Roundtrip: parse Bech32 back and compare bytes
		decoded := HexToAddress(bech32Str)
		if decoded != addr {
			t.Errorf("test #%d: roundtrip failed: %x != %x", i, decoded, addr)
		}
	}
	// Verify AddressToHex still returns EIP-55 checksummed hex
	addr := HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed")
	hexStr := AddressToHex(addr)
	if hexStr != "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed" {
		t.Errorf("AddressToHex mismatch: got %s", hexStr)
	}
}

func BenchmarkAddressHex(b *testing.B) {
	testAddr := HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed")
	for n := 0; n < b.N; n++ {
		testAddr.Hex()
	}
}

func TestMixedcaseAccount_Address(t *testing.T) {
	// With Bech32, ValidChecksum compares original string against Bech32 output.
	// A hex-input address will not match the Bech32 Hex() output, so ValidChecksum
	// returns false for hex originals. Bech32 originals should match.
	addr := HexToAddress("0xAe967917c465db8578ca9024c205720b1a3651A9")
	bech32Addr := addr.Hex()

	// Create from Bech32 — should be valid
	ma1 := NewMixedcaseAddress(addr)
	if !ma1.ValidChecksum() {
		t.Errorf("Expected valid checksum for Bech32 address, got invalid: %s", ma1.String())
	}

	// Create from Bech32 string — should be valid
	ma2, err := NewMixedcaseAddressFromString(bech32Addr)
	if err != nil {
		t.Fatal(err)
	}
	if !ma2.ValidChecksum() {
		t.Errorf("Expected valid checksum for Bech32 string, got invalid")
	}

	// These should throw exceptions:
	var r2 []MixedcaseAddress
	for _, r := range []string{
		`["0x11111111111111111111122222222222233333"]`,     // Too short
		`["0x111111111111111111111222222222222333332"]`,    // Too short
		`["0x11111111111111111111122222222222233333234"]`,  // Too long
		`["0x111111111111111111111222222222222333332344"]`, // Too long
		`["1111111111111111111112222222222223333323"]`,     // Missing 0x
		`["x1111111111111111111112222222222223333323"]`,    // Missing 0
		`["0xG111111111111111111112222222222223333323"]`,   // Non-hex
	} {
		if err := json.Unmarshal([]byte(r), &r2); err == nil {
			t.Errorf("Expected failure, input %v", r)
		}
	}
}

func TestHash_Scan(t *testing.T) {
	type args struct {
		src interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "working scan",
			args: args{src: []byte{
				0xb2, 0x6f, 0x2b, 0x34, 0x2a, 0xab, 0x24, 0xbc, 0xf6, 0x3e,
				0xa2, 0x18, 0xc6, 0xa9, 0x27, 0x4d, 0x30, 0xab, 0x9a, 0x15,
				0xa2, 0x18, 0xc6, 0xa9, 0x27, 0x4d, 0x30, 0xab, 0x9a, 0x15,
				0x10, 0x00,
			}},
			wantErr: false,
		},
		{
			name:    "non working scan",
			args:    args{src: int64(1234567890)},
			wantErr: true,
		},
		{
			name: "invalid length scan",
			args: args{src: []byte{
				0xb2, 0x6f, 0x2b, 0x34, 0x2a, 0xab, 0x24, 0xbc, 0xf6, 0x3e,
				0xa2, 0x18, 0xc6, 0xa9, 0x27, 0x4d, 0x30, 0xab, 0x9a, 0x15,
				0xa2, 0x18, 0xc6, 0xa9, 0x27, 0x4d, 0x30, 0xab, 0x9a, 0x15,
			}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Hash{}
			if err := h.Scan(tt.args.src); (err != nil) != tt.wantErr {
				t.Errorf("Hash.Scan() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				for i := range h {
					if h[i] != tt.args.src.([]byte)[i] {
						t.Errorf(
							"Hash.Scan() didn't scan the %d src correctly (have %X, want %X)",
							i, h[i], tt.args.src.([]byte)[i],
						)
					}
				}
			}
		})
	}
}

func TestHash_Value(t *testing.T) {
	b := []byte{
		0xb2, 0x6f, 0x2b, 0x34, 0x2a, 0xab, 0x24, 0xbc, 0xf6, 0x3e,
		0xa2, 0x18, 0xc6, 0xa9, 0x27, 0x4d, 0x30, 0xab, 0x9a, 0x15,
		0xa2, 0x18, 0xc6, 0xa9, 0x27, 0x4d, 0x30, 0xab, 0x9a, 0x15,
		0x10, 0x00,
	}
	var usedH Hash
	usedH.SetBytes(b)
	tests := []struct {
		name    string
		h       Hash
		want    driver.Value
		wantErr bool
	}{
		{
			name:    "Working value",
			h:       usedH,
			want:    b,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.h.Value()
			if (err != nil) != tt.wantErr {
				t.Errorf("Hash.Value() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Hash.Value() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddress_Scan(t *testing.T) {
	type args struct {
		src interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "working scan",
			args: args{src: []byte{
				0xb2, 0x6f, 0x2b, 0x34, 0x2a, 0xab, 0x24, 0xbc, 0xf6, 0x3e,
				0xa2, 0x18, 0xc6, 0xa9, 0x27, 0x4d, 0x30, 0xab, 0x9a, 0x15,
			}},
			wantErr: false,
		},
		{
			name:    "non working scan",
			args:    args{src: int64(1234567890)},
			wantErr: true,
		},
		{
			name: "invalid length scan",
			args: args{src: []byte{
				0xb2, 0x6f, 0x2b, 0x34, 0x2a, 0xab, 0x24, 0xbc, 0xf6, 0x3e,
				0xa2, 0x18, 0xc6, 0xa9, 0x27, 0x4d, 0x30, 0xab, 0x9a,
			}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Address{}
			if err := a.Scan(tt.args.src); (err != nil) != tt.wantErr {
				t.Errorf("Address.Scan() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				for i := range a {
					if a[i] != tt.args.src.([]byte)[i] {
						t.Errorf(
							"Address.Scan() didn't scan the %d src correctly (have %X, want %X)",
							i, a[i], tt.args.src.([]byte)[i],
						)
					}
				}
			}
		})
	}
}

func TestAddress_Value(t *testing.T) {
	b := []byte{
		0xb2, 0x6f, 0x2b, 0x34, 0x2a, 0xab, 0x24, 0xbc, 0xf6, 0x3e,
		0xa2, 0x18, 0xc6, 0xa9, 0x27, 0x4d, 0x30, 0xab, 0x9a, 0x15,
	}
	var usedA Address
	usedA.SetBytes(b)
	tests := []struct {
		name    string
		a       Address
		want    driver.Value
		wantErr bool
	}{
		{
			name:    "Working value",
			a:       usedA,
			want:    b,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.a.Value()
			if (err != nil) != tt.wantErr {
				t.Errorf("Address.Value() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Address.Value() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddress_Format(t *testing.T) {
	b := []byte{
		0xb2, 0x6f, 0x2b, 0x34, 0x2a, 0xab, 0x24, 0xbc, 0xf6, 0x3e,
		0xa2, 0x18, 0xc6, 0xa9, 0x27, 0x4d, 0x30, 0xab, 0x9a, 0x15,
	}
	var addr Address
	addr.SetBytes(b)

	// Expected Bech32 encoding of this address
	bech32Str := addr.Hex()

	tests := []struct {
		name string
		out  string
		want string
	}{
		{
			name: "println",
			out:  fmt.Sprintln(addr),
			want: bech32Str + "\n",
		},
		{
			name: "print",
			out:  fmt.Sprint(addr),
			want: bech32Str,
		},
		{
			name: "printf-s",
			out: func() string {
				buf := new(bytes.Buffer)
				fmt.Fprintf(buf, "%s", addr)
				return buf.String()
			}(),
			want: bech32Str,
		},
		{
			name: "printf-q",
			out:  fmt.Sprintf("%q", addr),
			want: `"` + bech32Str + `"`,
		},
		{
			name: "printf-x",
			out:  fmt.Sprintf("%x", addr),
			want: "b26f2b342aab24bcf63ea218c6a9274d30ab9a15",
		},
		{
			name: "printf-X",
			out:  fmt.Sprintf("%X", addr),
			want: "B26F2B342AAB24BCF63EA218C6A9274D30AB9A15",
		},
		{
			name: "printf-#x",
			out:  fmt.Sprintf("%#x", addr),
			want: "0xb26f2b342aab24bcf63ea218c6a9274d30ab9a15",
		},
		{
			name: "printf-v",
			out:  fmt.Sprintf("%v", addr),
			want: bech32Str,
		},
		// The original default formatter for byte slice
		{
			name: "printf-d",
			out:  fmt.Sprintf("%d", addr),
			want: "[178 111 43 52 42 171 36 188 246 62 162 24 198 169 39 77 48 171 154 21]",
		},
		// Invalid format char.
		{
			name: "printf-t",
			out:  fmt.Sprintf("%t", addr),
			want: "%!t(address=b26f2b342aab24bcf63ea218c6a9274d30ab9a15)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.out != tt.want {
				t.Errorf("%s does not render as expected:\n got %s\nwant %s", tt.name, tt.out, tt.want)
			}
		})
	}
}

func TestHash_Format(t *testing.T) {
	var hash Hash
	hash.SetBytes([]byte{
		0xb2, 0x6f, 0x2b, 0x34, 0x2a, 0xab, 0x24, 0xbc, 0xf6, 0x3e,
		0xa2, 0x18, 0xc6, 0xa9, 0x27, 0x4d, 0x30, 0xab, 0x9a, 0x15,
		0xa2, 0x18, 0xc6, 0xa9, 0x27, 0x4d, 0x30, 0xab, 0x9a, 0x15,
		0x10, 0x00,
	})

	tests := []struct {
		name string
		out  string
		want string
	}{
		{
			name: "println",
			out:  fmt.Sprintln(hash),
			want: "0xb26f2b342aab24bcf63ea218c6a9274d30ab9a15a218c6a9274d30ab9a151000\n",
		},
		{
			name: "print",
			out:  fmt.Sprint(hash),
			want: "0xb26f2b342aab24bcf63ea218c6a9274d30ab9a15a218c6a9274d30ab9a151000",
		},
		{
			name: "printf-s",
			out: func() string {
				buf := new(bytes.Buffer)
				fmt.Fprintf(buf, "%s", hash)
				return buf.String()
			}(),
			want: "0xb26f2b342aab24bcf63ea218c6a9274d30ab9a15a218c6a9274d30ab9a151000",
		},
		{
			name: "printf-q",
			out:  fmt.Sprintf("%q", hash),
			want: `"0xb26f2b342aab24bcf63ea218c6a9274d30ab9a15a218c6a9274d30ab9a151000"`,
		},
		{
			name: "printf-x",
			out:  fmt.Sprintf("%x", hash),
			want: "b26f2b342aab24bcf63ea218c6a9274d30ab9a15a218c6a9274d30ab9a151000",
		},
		{
			name: "printf-X",
			out:  fmt.Sprintf("%X", hash),
			want: "B26F2B342AAB24BCF63EA218C6A9274D30AB9A15A218C6A9274D30AB9A151000",
		},
		{
			name: "printf-#x",
			out:  fmt.Sprintf("%#x", hash),
			want: "0xb26f2b342aab24bcf63ea218c6a9274d30ab9a15a218c6a9274d30ab9a151000",
		},
		{
			name: "printf-#X",
			out:  fmt.Sprintf("%#X", hash),
			want: "0XB26F2B342AAB24BCF63EA218C6A9274D30AB9A15A218C6A9274D30AB9A151000",
		},
		{
			name: "printf-v",
			out:  fmt.Sprintf("%v", hash),
			want: "0xb26f2b342aab24bcf63ea218c6a9274d30ab9a15a218c6a9274d30ab9a151000",
		},
		// The original default formatter for byte slice
		{
			name: "printf-d",
			out:  fmt.Sprintf("%d", hash),
			want: "[178 111 43 52 42 171 36 188 246 62 162 24 198 169 39 77 48 171 154 21 162 24 198 169 39 77 48 171 154 21 16 0]",
		},
		// Invalid format char.
		{
			name: "printf-t",
			out:  fmt.Sprintf("%t", hash),
			want: "%!t(hash=b26f2b342aab24bcf63ea218c6a9274d30ab9a15a218c6a9274d30ab9a151000)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.out != tt.want {
				t.Errorf("%s does not render as expected:\n got %s\nwant %s", tt.name, tt.out, tt.want)
			}
		})
	}
}

func TestBigToAddress(t *testing.T) {
	addr := HexToAddress("0x73a852B3A0f63397f9f0DA7b8A0f7FF72d790b08")
	fmt.Println(addr.Bytes()) //[115 168 82 179 160 246 51 151 249 240 218 123 138 15 127 247 45 121 11 8]
	last10Bytes := addr[10:]
	fmt.Println(last10Bytes) //[218 123 138 15 127 247 45 121 11 8]
	addr1 := BytesToAddress(last10Bytes)
	fmt.Println("last10BytesToAddress:", addr1)
	a := new(big.Int).SetBytes(last10Bytes)
	fmt.Println(a.Uint64())
	fmt.Println(a.Uint64() % 1024)
	addr2 := BigToAddress(a)
	fmt.Println("BigToAddress:", addr2)
	//first10Bytes := addr2[:10]
	first10Bytes := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	fmt.Println(first10Bytes)
	fmt.Println(new(big.Int).SetBytes(first10Bytes))
	fmt.Println(addr2[10:])
}

func TestBigToAddress2(t *testing.T) {
	addr := HexToAddress("0x897638B555Fa1584965A1E1c4d4302264ac9432b")
	fmt.Println(addr.Bytes()) //[115 168 82 179 160 246 51 151 249 240 218 123 138 15 127 247 45 121 11 8]
	last10Bytes := addr[19:]
	fmt.Println(last10Bytes) //[218 123 138 15 127 247 45 121 11 8]
	addr1 := BytesToAddress(last10Bytes)
	fmt.Println("last10BytesToAddress:", addr1)
	a := new(big.Int).SetBytes(last10Bytes)
	addr2 := BigToAddress(a)
	fmt.Println("BigToAddress:", addr2)

}

func TestBigToAddress3(t *testing.T) {
	lost := HexToAddress("0x897638B555Fa1584965A1E1c4d4302264ac9432b")

	//beni := HexToAddress("0x28fd633B72cA9828542A7dA8E3426E11C831D4Bd")
	/*	var buffer bytes.Buffer
		buffer.WriteString(lost.String())
		buffer.WriteString(beni.String())
		buffer.WriteString(strconv.Itoa(int(uint32(123456))))
		h := md5.New()
		h.Write([]byte(buffer.String()))
		fmt.Println(buffer.String())
		fmt.Println(hex.EncodeToString(h.Sum(nil)))*/
	//flag := new(big.Int).SetUint64(1)
	bytes := lost.Bytes()[18:]
	fmt.Println("start：", biu.ToBinaryString(bytes)) //[01000011 00101011]
	a := new(big.Int).SetBytes(bytes)
	b := new(big.Int).Lsh(a, 6)
	c := new(big.Int).SetBytes(b.Bytes()[1:])
	fmt.Println(biu.ToBinaryString(c.Bytes())) //[11001010 11000000]
	d := new(big.Int).Rsh(c, 6)
	fmt.Println(biu.ToBinaryString(d.Bytes()))
	fmt.Println(d.String())
}

func TestBigToAddress4(t *testing.T) {
	var a [128]byte
	a[0] = 1
	fmt.Println("start：", biu.ToBinaryString(a[:]))
	b := new(big.Int).SetBytes(a[:])
	//c := b|(1<<20)
	flag := new(big.Int).SetUint64(1)
	index := uint(0)
	c := new(big.Int).Or(b, new(big.Int).Lsh(flag, index))
	fmt.Printf("set loc[%d]:%s\n", index, biu.ToBinaryString(c.Bytes()))
	fmt.Printf("get loc[%d]:%d\n", index, c.Bit(0))
	d := new(big.Int).SetBytes(c.Bytes())
	e := new(big.Int).AndNot(d, new(big.Int).Lsh(flag, index)) //z = x &^ y
	fmt.Printf("set loc[%d]:%s\n", index, biu.ToBinaryString(e.Bytes()))
	fmt.Printf("get loc[%d]:%d\n", index, e.Bit(int(index)))
	f := new(big.Int).SetBytes(e.Bytes())
	//d:=(a<<4)>>7
	fmt.Println(f.Bit(3))
	//new(big.Int).Lsh(f, uint(1023)).
}

func TestBigToAddress5(t *testing.T) {
	a := new(LossMark)
	a[0] = 1
	fmt.Println("start：", biu.ToBinaryString(a[:]))
	a.SetMark(0, false)
	fmt.Println(a.GetMarkedIndex())
	fmt.Println("update 0：", biu.ToBinaryString(a[:]))
	a.SetMark(0, true)
	fmt.Println(a.GetMarkedIndex())
	fmt.Println("update 1：", biu.ToBinaryString(a[:]))
	a.SetMark(0, false)
	fmt.Println(a.GetMarkedIndex())
	fmt.Println("update 0：", biu.ToBinaryString(a[:]))
}
func TestBigToAddress6(t *testing.T) {
	c := LossType(0)
	fmt.Println("start ", c)
	fmt.Println("state", c.GetState())
	d := c.SetState(false)
	fmt.Println("state", d.GetState())
	fmt.Println("start ", d)
	fmt.Println("type d", d.GetType(), "num", d)
	e := d.SetType(0)
	fmt.Println("type e", e.GetType(), "num", e)
	f := d.SetType(127)
	fmt.Println("type f", f.GetType(), "num", f)
	g := f.SetState(false)
	fmt.Println("type g", g.GetType(), "num", g)
}

func TestDpos(t *testing.T) {
	epoch := uint64(60)
	number := uint64(61)

	lastConfirmNumber := GetLastConfirmPoint(number, epoch)
	lastRoundId := CalcDPosNodeRoundId(lastConfirmNumber, epoch)
	fmt.Printf("current block beblog dPos：confirmNumber：%d，roundId：%d\n", lastConfirmNumber, lastRoundId)

	confirmNumber := GetCurrentConfirmPoint(number, epoch)
	confirmRoundId := CalcDPosNodeRoundId(confirmNumber, epoch)
	//confirmRoundId2 := CalcDPosNodeRoundId(number, epoch)
	fmt.Printf("confirmNumber：%d，roundId：%d\n", confirmNumber, confirmRoundId)
	//fmt.Printf("confirmNumber：%d，roundId2：%d\n", confirmNumber, confirmRoundId2)
	if number >= confirmNumber {
		fmt.Printf("next block beblog dPos：confirmNumber：%d，roundId：%d\n", confirmNumber, confirmRoundId)
	} else {
		fmt.Printf("next block beblog dPos：confirmNumber：%s，roundId：%d\n", "nil", confirmRoundId)
	}
}

func TestSpecialAddress(t *testing.T) {
	arr := []string{
		"0x0000000000000000000000000000000000000101",
		"0x0000000000000000000000000000000000000102",
		"0x0000000000000000000000000000000000000103",
		"0x0000000000000000000000000000000000000104",
		"0x0000000000000000000000000000000000000105",
		"0x0000000000000000000000000000000000000106",
		"0x0000000000000000000000000000000000000107",
		"0x0000000000000000000000000000000000000108",
		"0x0000000000000000000000000000000000000109",
		"0x0000000000000000000000000000000000000110",
		"0x0000000000000000000000000000000000000111",
		"0x0000000000000000000000000000000000000112",
		"0x0000000000000000000000000000000000000113",
		"0x0000000000000000000000000000000000000114",
		"0x0000000000000000000000000000000000000115",
		"0x0000000000000000000000000000000000000116",
		"0x0000000000000000000000000000000000000117",
		"0x0000000000000000000000000000000000000118",
		"0x0000000000000000000000000000000000000119",
		"0x0000000000000000000000000000000000000120",
		"0x0000000000000000000000000000000000000200",
		"0xFb6Ba8741A1F36132E7A4a8DA55e167d1baC98cC",
	}
	for _, v := range arr {
		addr := HexToAddress(v)
		_ = addr.Hash().Big().Uint64() <= 512
	}
}

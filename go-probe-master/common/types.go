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
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
	"strings"

	"github.com/probeum/go-probeum/common/bech32"
	"github.com/probeum/go-probeum/common/hexutil"
	"golang.org/x/crypto/sha3"
)

// Lengths of hashes and addresses in bytes.
const (
	//HashLength is the expected length of the hash
	HashLength = 32
	//AddressLength is the expected length of the address
	AddressLength = 20
	//DPosEnodeLength is the expected length of dPos enode
	DPosEnodeLength = 256
	//DPosNodeLength is the expected length of dPos node
	DPosNodeLength = 64
	//DPosNodeIntervalConfirmPoint is the dPos node confirm point
	DPosNodeIntervalConfirmPoint = 64
	//DPosNodePrefix the prefix of dPos node connection information
	DPosNodePrefix = "enode://"
	//LossMarkLength is the expected length of loss mark
	LossMarkLength = 128
	//LossMarkBitLength is the expected length of loss mark bits
	LossMarkBitLength = LossMarkLength * 8
)

// AddressHRP is the human-readable part for Bech32-encoded ProbeChain addresses.
const AddressHRP = "pro"

var (
	hashT    = reflect.TypeOf(Hash{})
	addressT = reflect.TypeOf(Address{})
)

// Hash represents the 32 byte Keccak256 hash of arbitrary data.
type Hash [HashLength]byte

// BytesToHash sets b to hash.
// If b is larger than len(h), b will be cropped from the left.
func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}

// BigToHash sets byte representation of b to hash.
// If b is larger than len(h), b will be cropped from the left.
func BigToHash(b *big.Int) Hash { return BytesToHash(b.Bytes()) }

// HexToHash sets byte representation of s to hash.
// If b is larger than len(h), b will be cropped from the left.
func HexToHash(s string) Hash { return BytesToHash(FromHex(s)) }

// Bytes gets the byte representation of the underlying hash.
func (h Hash) Bytes() []byte { return h[:] }

// Big converts a hash to a big integer.
func (h Hash) Big() *big.Int { return new(big.Int).SetBytes(h[:]) }

// Hex converts a hash to a hex string.
func (h Hash) Hex() string { return hexutil.Encode(h[:]) }

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (h Hash) TerminalString() string {
	return fmt.Sprintf("%x..%x", h[:3], h[29:])
}

// String implements the stringer interface and is used also by the logger when
// doing full logging into a file.
func (h Hash) String() string {
	return h.Hex()
}

// Format implements fmt.Formatter.
// Hash supports the %v, %s, %v, %x, %X and %d format verbs.
func (h Hash) Format(s fmt.State, c rune) {
	hexb := make([]byte, 2+len(h)*2)
	copy(hexb, "0x")
	hex.Encode(hexb[2:], h[:])

	switch c {
	case 'x', 'X':
		if !s.Flag('#') {
			hexb = hexb[2:]
		}
		if c == 'X' {
			hexb = bytes.ToUpper(hexb)
		}
		fallthrough
	case 'v', 's':
		s.Write(hexb)
	case 'q':
		q := []byte{'"'}
		s.Write(q)
		s.Write(hexb)
		s.Write(q)
	case 'd':
		fmt.Fprint(s, ([len(h)]byte)(h))
	default:
		fmt.Fprintf(s, "%%!%c(hash=%x)", c, h)
	}
}

// UnmarshalText parses a hash in hex syntax.
func (h *Hash) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("Hash", input, h[:])
}

// UnmarshalJSON parses a hash in hex syntax.
func (h *Hash) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(hashT, input, h[:])
}

// MarshalText returns the hex representation of h.
func (h Hash) MarshalText() ([]byte, error) {
	return hexutil.Bytes(h[:]).MarshalText()
}

// SetBytes sets the hash to the value of b.
// If b is larger than len(h), b will be cropped from the left.
func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}

	copy(h[HashLength-len(b):], b)
}

// Generate implements testing/quick.Generator.
func (h Hash) Generate(rand *rand.Rand, size int) reflect.Value {
	m := rand.Intn(len(h))
	for i := len(h) - 1; i > m; i-- {
		h[i] = byte(rand.Uint32())
	}
	return reflect.ValueOf(h)
}

// Scan implements Scanner for database/sql.
func (h *Hash) Scan(src interface{}) error {
	srcB, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("can't scan %T into Hash", src)
	}
	if len(srcB) != HashLength {
		return fmt.Errorf("can't scan []byte of len %d into Hash, want %d", len(srcB), HashLength)
	}
	copy(h[:], srcB)
	return nil
}

// Value implements valuer for database/sql.
func (h Hash) Value() (driver.Value, error) {
	return h[:], nil
}

// ImplementsGraphQLType returns true if Hash implements the specified GraphQL type.
func (Hash) ImplementsGraphQLType(name string) bool { return name == "Bytes32" }

// UnmarshalGraphQL unmarshals the provided GraphQL query data.
func (h *Hash) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		err = h.UnmarshalText([]byte(input))
	default:
		err = fmt.Errorf("unexpected type %T for Hash", input)
	}
	return err
}

// UnprefixedHash allows marshaling a Hash without 0x prefix.
type UnprefixedHash Hash

// UnmarshalText decodes the hash from hex. The 0x prefix is optional.
func (h *UnprefixedHash) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedUnprefixedText("UnprefixedHash", input, h[:])
}

// MarshalText encodes the hash as hex.
func (h UnprefixedHash) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(h[:])), nil
}

/////////// Address

// Address represents the 25 byte address of an Probeum account.
type Address [AddressLength]byte

type DposEnode [DPosEnodeLength]byte

type LossMark [LossMarkLength]byte

//LossType loss reporting type bits[0]: loss reporting status,bits[1~7]:loss reporting period
type LossType byte

type DPoSAccount struct {
	Enode DposEnode
	Owner Address
}

type DPoSCandidateAccount struct {
	Enode       DposEnode
	Owner       Address
	VoteAccount Address
	VoteValue   *big.Int
}

func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}

func BytesToDposEnode(b []byte) DposEnode {
	var n DposEnode
	n.SetBytes(b)
	return n
}

// BigToAddress returns Address with byte values of b.
// If b is larger than len(h), b will be cropped from the left.
func BigToAddress(b *big.Int) Address { return BytesToAddress(b.Bytes()) }

// HexToAddress returns Address with byte values of s.
// Accepts both 0x-prefixed hex and pro1-prefixed Bech32 format.
// If s is larger than len(h), s will be cropped from the left.
func HexToAddress(s string) Address {
	if hasProPrefix(s) {
		addr, err := Bech32ToAddress(s)
		if err == nil {
			return addr
		}
	}
	return BytesToAddress(FromHex(s))
}

// IsHexAddress verifies whether a string can represent a valid hex-encoded
// or Bech32-encoded Probeum address.
func IsHexAddress(s string) bool {
	if hasProPrefix(s) {
		return IsProbeAddress(s)
	}
	if has0xPrefix(s) {
		s = s[2:]
	}
	return len(s) == 2*AddressLength && isHex(s)
}

// Bech32ToAddress parses a pro1... Bech32 string to an Address.
func Bech32ToAddress(s string) (Address, error) {
	hrp, data, err := bech32.Decode(s)
	if err != nil {
		return Address{}, err
	}
	if hrp != AddressHRP {
		return Address{}, fmt.Errorf("invalid HRP: got %q, want %q", hrp, AddressHRP)
	}
	if len(data) != AddressLength {
		return Address{}, fmt.Errorf("invalid address length: got %d, want %d", len(data), AddressLength)
	}
	var addr Address
	copy(addr[:], data)
	return addr, nil
}

// IsProbeAddress validates whether s is a valid pro1... Bech32 address.
func IsProbeAddress(s string) bool {
	_, err := Bech32ToAddress(s)
	return err == nil
}

// AddressToHex returns the 0x-prefixed EIP-55 checksummed hex representation
// of the address. Useful for debugging and backward compatibility.
func AddressToHex(a Address) string {
	return string(a.checksumHex())
}

// Bytes gets the string representation of the underlying address.
func (a Address) Bytes() []byte { return a[:] }

func (a Address) Equal(address Address) bool { return bytes.Compare(a.Bytes(), address.Bytes()) == 0 }

// Last10BitsToUint intercepts last 10 bits to convert uint64
func (a Address) Last10BitsToUint() uint64 {
	last2Bytes := a.Bytes()[18:]
	b := new(big.Int).SetBytes(last2Bytes)
	c := new(big.Int).Lsh(b, 6)
	d := new(big.Int).SetBytes(c.Bytes()[1:])
	e := new(big.Int).Rsh(d, 6)
	return e.Uint64()
}

// Hash converts an address to a hash by left-padding it with zeros.
func (a Address) Hash() Hash { return BytesToHash(a[:]) }

// Hex returns the Bech32-encoded pro1... representation of the address.
func (a Address) Hex() string {
	s, err := bech32.Encode(AddressHRP, a[:])
	if err != nil {
		// Fallback to hex on encoding error (should never happen for valid addresses)
		return string(a.checksumHex())
	}
	return s
}

// String implements fmt.Stringer.
func (a Address) String() string {
	return a.Hex()
}

func (a *Address) checksumHex() []byte {
	buf := a.hex()

	// compute checksum
	sha := sha3.NewLegacyKeccak256()
	sha.Write(buf[2:])
	hash := sha.Sum(nil)
	for i := 2; i < len(buf); i++ {
		hashByte := hash[(i-2)/2]
		if i%2 == 0 {
			hashByte = hashByte >> 4
		} else {
			hashByte &= 0xf
		}
		if buf[i] > '9' && hashByte > 7 {
			buf[i] -= 32
		}
	}
	return buf[:]
}

func (a Address) hex() []byte {
	var buf [len(a)*2 + 2]byte
	copy(buf[:2], "0x")
	hex.Encode(buf[2:], a[:])
	return buf[:]
}

// Format implements fmt.Formatter.
// Address supports the %v, %s, %v, %x, %X and %d format verbs.
// %v and %s output Bech32 (pro1...), %x and %X output raw hex.
func (a Address) Format(s fmt.State, c rune) {
	switch c {
	case 'v', 's':
		s.Write([]byte(a.Hex()))
	case 'q':
		q := []byte{'"'}
		s.Write(q)
		s.Write([]byte(a.Hex()))
		s.Write(q)
	case 'x', 'X':
		// %x outputs raw hex (for debugging).
		hexBytes := a.hex()
		if !s.Flag('#') {
			hexBytes = hexBytes[2:]
		}
		if c == 'X' {
			hexBytes = bytes.ToUpper(hexBytes)
		}
		s.Write(hexBytes)
	case 'd':
		fmt.Fprint(s, ([len(a)]byte)(a))
	default:
		fmt.Fprintf(s, "%%!%c(address=%x)", c, a)
	}
}

// SetBytes sets the address to the value of b.
// If b is larger than len(a), b will be cropped from the left.
func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-AddressLength:]
	}
	copy(a[AddressLength-len(b):], b)
}

func (n *DposEnode) SetBytes(b []byte) {
	if len(b) > len(n) {
		b = b[len(b)-DPosEnodeLength:]
	}
	copy(n[DPosEnodeLength-len(b):], b)
}

// MarshalText returns the Bech32 representation of a.
func (a Address) MarshalText() ([]byte, error) {
	return []byte(a.Hex()), nil
}

// UnmarshalText parses an address in Bech32 (pro1...) or hex (0x...) syntax.
func (a *Address) UnmarshalText(input []byte) error {
	s := string(input)
	if hasProPrefix(s) {
		addr, err := Bech32ToAddress(s)
		if err != nil {
			return err
		}
		*a = addr
		return nil
	}
	return hexutil.UnmarshalFixedText("Address", input, a[:])
}

// UnmarshalJSON parses an address in Bech32 (pro1...) or hex (0x...) syntax.
func (a *Address) UnmarshalJSON(input []byte) error {
	// Try to unquote JSON string
	if len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"' {
		s := string(input[1 : len(input)-1])
		if hasProPrefix(s) {
			addr, err := Bech32ToAddress(s)
			if err != nil {
				return err
			}
			*a = addr
			return nil
		}
	}
	return hexutil.UnmarshalFixedJSON(addressT, input, a[:])
}

// Scan implements Scanner for database/sql.
func (a *Address) Scan(src interface{}) error {
	srcB, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("can't scan %T into Address", src)
	}
	if len(srcB) != AddressLength {
		return fmt.Errorf("can't scan []byte of len %d into Address, want %d", len(srcB), AddressLength)
	}
	copy(a[:], srcB)
	return nil
}

// Value implements valuer for database/sql.
func (a Address) Value() (driver.Value, error) {
	return a[:], nil
}

// ImplementsGraphQLType returns true if Hash implements the specified GraphQL type.
func (a Address) ImplementsGraphQLType(name string) bool { return name == "Address" }

// UnmarshalGraphQL unmarshals the provided GraphQL query data.
// Accepts both pro1... Bech32 and 0x... hex formats.
func (a *Address) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		err = a.UnmarshalText([]byte(input))
	default:
		err = fmt.Errorf("unexpected type %T for Address", input)
	}
	return err
}

// UnprefixedAddress allows marshaling an Address without 0x prefix.
type UnprefixedAddress Address

// UnmarshalText decodes the address from hex. The 0x prefix is optional.
func (a *UnprefixedAddress) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedUnprefixedText("UnprefixedAddress", input, a[:])
}

// MarshalText encodes the address as hex.
func (a UnprefixedAddress) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(a[:])), nil
}

// MixedcaseAddress retains the original string, which may or may not be
// correctly checksummed
type MixedcaseAddress struct {
	addr     Address
	original string
}

// NewMixedcaseAddress constructor (mainly for testing)
func NewMixedcaseAddress(addr Address) MixedcaseAddress {
	return MixedcaseAddress{addr: addr, original: addr.Hex()}
}

// NewMixedcaseAddressFromString is mainly meant for unit-testing.
// Accepts both 0x... hex and pro1... Bech32 formats.
func NewMixedcaseAddressFromString(addrStr string) (*MixedcaseAddress, error) {
	if !IsHexAddress(addrStr) {
		return nil, errors.New("invalid address")
	}
	addr := HexToAddress(addrStr)
	return &MixedcaseAddress{addr: addr, original: addrStr}, nil
}

// UnmarshalJSON parses MixedcaseAddress from Bech32 or hex JSON string.
func (ma *MixedcaseAddress) UnmarshalJSON(input []byte) error {
	var s string
	if err := json.Unmarshal(input, &s); err != nil {
		return err
	}
	if hasProPrefix(s) {
		addr, err := Bech32ToAddress(s)
		if err != nil {
			return err
		}
		ma.addr = addr
		ma.original = s
		return nil
	}
	if err := hexutil.UnmarshalFixedJSON(addressT, input, ma.addr[:]); err != nil {
		return err
	}
	ma.original = s
	return nil
}

// MarshalJSON marshals the address in Bech32 format.
func (ma *MixedcaseAddress) MarshalJSON() ([]byte, error) {
	return json.Marshal(ma.addr.Hex())
}

// Address returns the address
func (ma *MixedcaseAddress) Address() Address {
	return ma.addr
}

// String implements fmt.Stringer
func (ma *MixedcaseAddress) String() string {
	if ma.ValidChecksum() {
		return fmt.Sprintf("%s [chksum ok]", ma.original)
	}
	return fmt.Sprintf("%s [chksum INVALID]", ma.original)
}

// ValidChecksum returns true if the address has valid checksum
func (ma *MixedcaseAddress) ValidChecksum() bool {
	return ma.original == ma.addr.Hex()
}

// Original returns the mixed-case input string
func (ma *MixedcaseAddress) Original() string {
	return ma.original
}

// ValidateAddress return the accountType byte value for the input address
func ValidCheckAddress(v string) (c byte, err error) {
	return 0, errors.New("unsupported account type")
}

// UnmarshalJSON enode
func (enode *DposEnode) UnmarshalJSON(input []byte) error {
	var textEnode string
	err := json.Unmarshal(input, &textEnode)
	if err != nil {
		return err
	}
	*enode = BytesToDposEnode([]byte(textEnode))
	return nil
}

// MarshalJSON marshals the original value
func (enode *DposEnode) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(enode[:]))
}

func (enode *DposEnode) String() string {
	s := string(enode[:])
	i := strings.Index(s, DPosNodePrefix)
	if i == -1 {
		return ""
	}
	return string([]byte(s)[i:])
}

//SetMark set the value of the specified index
func (a *LossMark) SetMark(index uint, flag bool) error {
	if index > (LossMarkBitLength - 1) {
		return ErrIndexOutOfBounds
	}
	var b *big.Int
	mark := new(big.Int).SetUint64(1)
	num := new(big.Int).SetBytes(a[:])
	if flag {
		b = new(big.Int).Or(num, new(big.Int).Lsh(mark, index))
	} else {
		b = new(big.Int).AndNot(num, new(big.Int).Lsh(mark, index))
	}
	dst := make([]byte, LossMarkLength)
	src := b.Bytes()
	copy(dst[LossMarkLength-len(src):], src)
	copy(a[:], dst)
	return nil
}

// GetMark return the value of the specified index
func (a *LossMark) GetMark(index uint) bool {
	if index > (LossMarkBitLength - 1) {
		return false
	}
	return new(big.Int).SetBytes(a[:]).Bit(int(index)) > 0
}

// GetMarkedIndex return the value of the marked index
func (a *LossMark) GetMarkedIndex() []uint16 {
	markInt := new(big.Int).SetBytes(a[:])
	var ret []uint16
	for i := 0; i < LossMarkBitLength; i++ {
		if markInt.Bit(i) > 0 {
			ret = append(ret, uint16(i))
		}
	}
	return ret
}

//GetState return loss reporting status
func (a *LossType) GetState() bool {
	b := new(big.Int).SetUint64(uint64(*a))
	return b.Bit(0) > 0
}

//SetState set loss reporting status
func (a *LossType) SetState(flag bool) LossType {
	var c *big.Int
	mark := new(big.Int).SetUint64(1)
	b := new(big.Int).SetUint64(uint64(*a))
	if flag {
		c = new(big.Int).Or(b, new(big.Int).Lsh(mark, 0))
	} else {
		c = new(big.Int).AndNot(b, new(big.Int).Lsh(mark, 0))
	}
	if c.Sign() > 0 {
		return LossType(c.Bytes()[0])
	}
	return LossType(0)
}

//GetType return loss reporting cycle time
func (a *LossType) GetType() byte {
	bytes := new(big.Int).Rsh(new(big.Int).SetUint64(uint64(*a)), 1).Bytes()
	if len(bytes) == 0 {
		return 0
	}
	return bytes[0]
}

//SetType set loss reporting cycle period time
func (a *LossType) SetType(period byte) LossType {
	flag := a.GetState()
	b := new(big.Int).Lsh(new(big.Int).SetUint64(uint64(period)), 1)
	var c LossType
	if b.Sign() > 0 {
		c = LossType(b.Bytes()[0])
	} else {
		c = LossType(0)
	}
	return c.SetState(flag)
}

//CalcDPosNodeRoundId calculation DPos node round id
func CalcDPosNodeRoundId(blockNumber, dPosEpoch uint64) uint64 {
	if blockNumber == 0 {
		return blockNumber
	}
	confirmBlockNum := dPosEpoch / 2
	if dPosEpoch > DPosNodeIntervalConfirmPoint {
		confirmBlockNum = DPosNodeIntervalConfirmPoint
	}
	factor := blockNumber + confirmBlockNum + dPosEpoch - 1
	return (factor - factor%dPosEpoch) / dPosEpoch
}

//IsConfirmPoint calculation DPos node confirm point
func IsConfirmPoint(blockNumber, dPosEpoch uint64) bool {
	confirmBlockNum := dPosEpoch / 2
	if dPosEpoch > DPosNodeIntervalConfirmPoint {
		confirmBlockNum = DPosNodeIntervalConfirmPoint
	}
	return blockNumber%((blockNumber/dPosEpoch+1)*dPosEpoch-confirmBlockNum) == 0
}

//GetLastConfirmPoint returns the dPos last confirmation point at the specified height
func GetLastConfirmPoint(blockNumber, dPosEpoch uint64) uint64 {
	if blockNumber <= dPosEpoch {
		return 0
	}
	confirmBlockNum := dPosEpoch / 2
	if dPosEpoch > DPosNodeIntervalConfirmPoint {
		confirmBlockNum = DPosNodeIntervalConfirmPoint
	}
	return (blockNumber-1)/dPosEpoch*dPosEpoch - confirmBlockNum
}

//GetCurrentConfirmPoint returns the dPos current confirmation point at the specified height
func GetCurrentConfirmPoint(blockNumber, dPosEpoch uint64) uint64 {
	lastConfirmPoint := GetLastConfirmPoint(blockNumber, dPosEpoch)
	var currConfirmPoint uint64
	if lastConfirmPoint == 0 {
		confirmBlockNum := dPosEpoch / 2
		if dPosEpoch > DPosNodeIntervalConfirmPoint {
			confirmBlockNum = DPosNodeIntervalConfirmPoint
		}
		currConfirmPoint = dPosEpoch - confirmBlockNum
	} else {
		currConfirmPoint = lastConfirmPoint + dPosEpoch
	}
	return currConfirmPoint
}

type IntDecodeType struct {
	Num big.Int
}

type ByteDecodeType struct {
	Num byte
}

type AddressDecodeType struct {
	Addr Address
}

type StringDecodeType struct {
	Text string
}

type CancellationDecodeType struct {
	CancelAddress      Address
	BeneficiaryAddress Address
}

type ApplyDPosDecodeType struct {
	VoteAddress Address
	NodeInfo    string
}

type PnsOwnerDecodeType struct {
	PnsAddress   Address
	OwnerAddress Address
}

type PnsContentDecodeType struct {
	PnsAddress Address
	PnsType    byte
	PnsData    string
}

type RegisterLossDecodeType struct {
	LastBitsMark uint32
	InfoDigest   Hash
}

type RevealLossReportDecodeType struct {
	LossAccount Address //loss reporting address
	OldAccount  Address //lost address
	NewAccount  Address //beneficiary address
	RandomNum   uint32  //random number
}

type AssociatedAccountDecodeType struct {
	LossAccount       Address //loss reporting address
	AssociatedAccount Address //associated address
}

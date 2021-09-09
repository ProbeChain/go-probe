package probe

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"hash"
	"io"
	"io/ioutil"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"golang.org/x/crypto/sha3"
)

type PublicKey struct {
	X *big.Int
	Y *big.Int
	elliptic.Curve
	K byte
}

func (pub *PublicKey) ExportECDSAPubKey() *ecdsa.PublicKey {
	return &ecdsa.PublicKey{Curve: pub.Curve, X: pub.X, Y: pub.Y}
}

func ImportECDSAPublic(pub *ecdsa.PublicKey, k byte) *PublicKey {
	return &PublicKey{
		X:     pub.X,
		Y:     pub.Y,
		Curve: pub.Curve,
		K:     k,
	}
}

type PrivateKey struct {
	PublicKey
	D *big.Int
	K byte
}

// Import an ECDSA private key as an probe private key.
func ImportECDSA(prv *ecdsa.PrivateKey, k byte) *PrivateKey {
	pub := ImportECDSAPublic(&prv.PublicKey, k)
	return &PrivateKey{*pub, prv.D, k}
}

func (prv *PrivateKey) ExportECDSA() *ecdsa.PrivateKey {
	pub := &prv.PublicKey
	pubECDSA := pub.ExportECDSAPubKey()
	return &ecdsa.PrivateKey{PublicKey: *pubECDSA, D: prv.D}
}

var (
	one              = new(big.Int).SetInt64(1)
	secp256k1N, _    = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	secp256k1halfN   = new(big.Int).Div(secp256k1N, big.NewInt(2))
	errInvalidPubkey = errors.New("invalid secp256k1 public key")
)

// randFieldElement returns a random element of the field underlying the given
// curve using the procedure given in [NSA] A.2.1.
func randFieldElement(c elliptic.Curve, rand io.Reader) (k *big.Int, err error) {
	params := c.Params()
	b := make([]byte, params.BitSize/8+8)
	_, err = io.ReadFull(rand, b)
	if err != nil {
		return
	}

	k = new(big.Int).SetBytes(b)
	n := new(big.Int).Sub(params.N, one)
	k.Mod(k, n)
	k.Add(k, one)
	return
}

/*func GenerateKey(c elliptic.Curve, rand io.Reader) (*PrivateKey, error) {
	k, err := randFieldElement(c, rand)
	if err != nil {
		return nil, err
	}

	priv := new(PrivateKey)
	priv.PublicKey.Curve = c
	priv.D = k
	priv.PublicKey.X, priv.PublicKey.Y = c.ScalarBaseMult(k.Bytes())
	return priv, nil
}*/

// GenerateKey generates a new private key.
func GenerateKey() (*PrivateKey, error) {
	return GenerateKeyByType(0x00)
}

func GenerateKeyByTypeForRand(k byte, rand io.Reader) (*PrivateKey, error) {
	key, err := ecdsa.GenerateKey(S256ByType(k), rand)
	if err != nil {
		fmt.Println("Error: ", err.Error())
	}
	return ImportECDSA(key, k), nil
}

func GenerateKeyForRand(rand io.Reader) (*PrivateKey, error) {
	k := byte(0x00)
	key, err := ecdsa.GenerateKey(S256ByType(k), rand)
	if err != nil {
		fmt.Println("Error: ", err.Error())
	}
	return ImportECDSA(key, k), nil
}

func GenerateKeyByType(k byte) (*PrivateKey, error) {
	key, err := ecdsa.GenerateKey(S256ByType(k), rand.Reader)
	if err != nil {
		fmt.Println("Error: ", err.Error())
	}
	return ImportECDSA(key, k), nil
}

func PubkeyToAddress(p PublicKey) common.Address {
	/*
		pubBytes := FromECDSAPub(&p)
		return common.BytesToAddress(Keccak256(pubBytes[1:])[12:])
	*/
	pubBytes := FromECDSAPub(&p)
	return PubkeyBytesToAddress(pubBytes, p.K)
}

// FromECDSA exports a private key into a binary dump.
func FromECDSA(priv *PrivateKey) []byte {
	/*
		if priv == nil {
			return nil
		}
		return math.PaddedBigBytes(priv.D, priv.Params().BitSize/8)
	*/
	if priv == nil {
		return nil
	}
	b := math.PaddedBigBytes(priv.D, priv.Params().BitSize/8)
	c := make([]byte, len(b)+1)
	c[0] = priv.K
	copy(c[1:], b)
	return c
}

func HexToECDSA(hexkey string) (*PrivateKey, error) {
	b, err := hex.DecodeString(hexkey)
	if byteErr, ok := err.(hex.InvalidByteError); ok {
		return nil, fmt.Errorf("invalid hex character %q in private key", byte(byteErr))
	} else if err != nil {
		return nil, errors.New("invalid hex data for private key")
	}
	//k := b[0]
	//ecdsaKey, err := ToECDSA(b)
	return ToECDSA(b)
	//return ImportECDSA(ecdsaKey, k), nil
}

func FromECDSAPub(pub *PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return elliptic.Marshal(S256ByType(pub.K), pub.X, pub.Y)
}

func SaveECDSA(file string, key *PrivateKey) error {
	k := hex.EncodeToString(FromECDSA(key))
	return ioutil.WriteFile(file, []byte(k), 0600)
}

// ToECDSA creates a private key with the given D value.
func ToECDSA(d []byte) (*PrivateKey, error) {
	return toECDSA(d, true)
}

// LoadECDSA loads a secp256k1 private key from the given file.
func LoadECDSA(file string) (*PrivateKey, error) {
	fd, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	r := bufio.NewReader(fd)
	buf := make([]byte, 66)
	n, err := readASCII(buf, r)
	if err != nil {
		return nil, err
	} else if n != len(buf) {
		return nil, fmt.Errorf("key file too short, want 64 hex characters")
	}
	if err := checkKeyFileEnd(r); err != nil {
		return nil, err
	}

	return HexToECDSA(string(buf))
}

// readASCII reads into 'buf', stopping when the buffer is full or
// when a non-printable control character is encountered.
func readASCII(buf []byte, r *bufio.Reader) (n int, err error) {
	for ; n < len(buf); n++ {
		buf[n], err = r.ReadByte()
		switch {
		case err == io.EOF || buf[n] < '!':
			return n, nil
		case err != nil:
			return n, err
		}
	}
	return n, nil
}

// checkKeyFileEnd skips over additional newlines at the end of a key file.
func checkKeyFileEnd(r *bufio.Reader) error {
	for i := 0; ; i++ {
		b, err := r.ReadByte()
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case b != '\n' && b != '\r':
			return fmt.Errorf("invalid character %q at end of key file", b)
		case i >= 2:
			return errors.New("key file too long, want 64 hex characters")
		}
	}
}

// toECDSA creates a private key with the given D value. The strict parameter
// controls whether the key's length should be enforced at the curve size or
// it can also accept legacy encodings (0 prefixes).
func toECDSA(d []byte, strict bool) (*PrivateKey, error) {
	priv := new(PrivateKey)
	//priv.C = d[0]
	//priv.PublicKey.C = priv.C
	k := d[0]
	d = d[1:]
	priv.PublicKey.Curve = S256ByType(k)
	if strict && 8*len(d) != priv.Params().BitSize {
		return nil, fmt.Errorf("invalid length, need %d bits", priv.Params().BitSize)
	}
	priv.D = new(big.Int).SetBytes(d)

	// The priv.D must < N
	if priv.D.Cmp(secp256k1N) >= 0 {
		return nil, fmt.Errorf("invalid private key, >=N")
	}
	// The priv.D must not be zero or negative.
	if priv.D.Sign() <= 0 {
		return nil, fmt.Errorf("invalid private key, zero or negative")
	}

	priv.PublicKey.X, priv.PublicKey.Y = priv.PublicKey.Curve.ScalarBaseMult(d)
	if priv.PublicKey.X == nil {
		return nil, errors.New("invalid private key")
	}
	priv.K = k
	return priv, nil
}

// S256 returns an instance of the secp256k1 curve.
func S256() elliptic.Curve {
	return secp256k1.S256()
}

func S256ByType(c byte) elliptic.Curve {
	return secp256k1.S256ByType(c)
}

type KeccakState interface {
	hash.Hash
	Read([]byte) (int, error)
}

// NewKeccakState creates a new KeccakState
func NewKeccakState() KeccakState {
	return sha3.NewLegacyKeccak256().(KeccakState)
}

// Keccak256 calculates and returns the Keccak256 hash of the input data.
func Keccak256(data ...[]byte) []byte {
	b := make([]byte, 32)
	d := NewKeccakState()
	for _, b := range data {
		d.Write(b)
	}
	d.Read(b)
	return b
}
func PubkeyBytesToAddress(pubKey []byte, fromAcType byte) common.Address {
	b := Keccak256(pubKey[1:])[12:]
	c := make([]byte, len(b)+1)
	c[0] = fromAcType
	copy(c[1:], b)
	checkSumBytes := common.CheckSum(c)
	return common.BytesToAddress(append(c, checkSumBytes...))
}

func UnmarshalPubkey(pub []byte) (*PublicKey, error) {
	//TODO node start need set default k
	k := byte(0x00)
	if len(pub) == 66 {
		k = pub[65]
	}
	x, y := elliptic.Unmarshal(S256ByType(k), pub[:65])
	if x == nil {
		return nil, errInvalidPubkey
	}
	return &PublicKey{Curve: S256ByType(k), X: x, Y: y, K: k}, nil
}

func ToECDSAUnsafe(d []byte) *PrivateKey {
	priv, _ := toECDSA(d, false)
	return priv
}

func CreateAddressForAccountType(address common.Address, nonce uint64, K byte) (add common.Address, err error) {
	k1, err := common.ValidAddress(address)
	if k1 != 0x00 || err != nil {
		return address, err
	}
	data, _ := rlp.EncodeToBytes([]interface{}{address, nonce})
	return PubkeyBytesToAddress(Keccak256(data)[12:], K), nil
}

func CreatePNSAddressStr(address string, pns []byte, K byte) (add common.Address, err error) {

	k1, err := common.ValidCheckAddress(address)
	if k1 != 0x00 || err != nil {
		log.Crit("Failed to Create PNSAddress from address", "err", err)
		return common.HexToAddress(address), err
	}
	if len(pns) <= 0 {
		return common.HexToAddress(address), errors.New("Creat PNSAddress error,PNS parameter is invalid")
	}

	b, err := hexutil.Decode(address)
	return PubkeyBytesToAddress(Keccak256([]byte{K}, b, pns)[12:], K), nil
}

func CreatePNSAddress(address common.Address, pns []byte, K byte) (add common.Address, err error) {
	if len(pns) <= 0 {
		return address, errors.New("Creat PNSAddress error,PNS parameter is invalid")
	}
	k1, err := common.ValidAddress(address)
	if k1 != 0x00 || err != nil {
		return address, err
	}
	return PubkeyBytesToAddress(Keccak256([]byte{K}, address.Bytes(), pns)[12:], K), nil
}

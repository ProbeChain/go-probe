package probe

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"io"
	"math/big"
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

var one = new(big.Int).SetInt64(1)

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

func GenerateKeyByType(k byte) (*PrivateKey, error) {
	key, err := crypto.GenerateKeyByType(k)
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
	pubBytes := crypto.FromECDSAPub(p.ExportECDSAPubKey())
	return crypto.PubkeyBytesToAddress(pubBytes, p.K)
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
	k := b[0]
	ecdsaKey, err := crypto.ToECDSA(b)
	return ImportECDSA(ecdsaKey, k), nil
}

func FromECDSAPub(pub *PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return crypto.FromECDSAPub(pub.ExportECDSAPubKey())
}

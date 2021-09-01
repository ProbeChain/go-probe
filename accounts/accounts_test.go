// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package accounts

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestTextHash(t *testing.T) {
	hash := TextHash([]byte("Hello Joe"))
	want := hexutil.MustDecode("0xa080337ae51c4e064c189e113edd0ba391df9206e2f49db658bb32cf2911730b")
	if !bytes.Equal(hash, want) {
		t.Fatalf("wrong hash: %x", hash)
	}
}

func TestGenerate(t *testing.T) {
	//Create an account
	key, err := crypto.GenerateKeyByType(0x01)
	if err != nil {
		fmt.Println("Error: ", err.Error())
	}
	hexPriKey := hex.EncodeToString(crypto.FromECDSA(key))
	//不含0x的私钥65
	fmt.Printf("private key [%d] [%v]\n", len(hexPriKey), hexPriKey)
	//Get the address
	address := crypto.PubkeyToAddress(key.PublicKey).Hex()
	fmt.Printf("address[%d][%v]\n", len(address), address)
	//	address2 := PubkeyToNormalAddress(key.PublicKey).Hex()
	//fmt.Printf("address[%d][%v]\n", len(address2), address2)

	acc1Key, _ := crypto.HexToECDSA(hexPriKey)
	fmt.Println("public key from prikey  \n", hexutil.Encode(crypto.FromECDSAPub(&acc1Key.PublicKey)))
	address1 := crypto.PubkeyToAddress(acc1Key.PublicKey).Hex()
	fmt.Println("address1 ", address1)

	var c, err2 = common.ValidCheckAddress(address)
	if err2 != nil {
		fmt.Printf("failed GenerateKey with %s.", err2)
	}
	fmt.Printf("flag[%T][%X]\n", c, c)

}

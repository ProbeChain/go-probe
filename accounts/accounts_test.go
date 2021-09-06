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
	"github.com/ethereum/go-ethereum/crypto/probe"
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

func TestCreateAddressForAccountType(t *testing.T) {
	//Create an account
	k := byte(0x02)
	key, err := crypto.GenerateKeyByType(k)
	if err != nil {
		fmt.Println("Error: ", err.Error())
	}
	hexPriKey := hex.EncodeToString(crypto.FromECDSAByType(key, k))
	//不含0x的私钥65
	fmt.Printf("private key [%d] [%v]\n", len(hexPriKey), hexPriKey)
	//Get the address
	address := crypto.PubkeyToAddressForType(key.PublicKey, k)
	fmt.Printf("address[%d][%v]\n", len(address), address)

	var c, err2 = common.ValidCheckAddress(address.Hex())
	if err2 != nil {
		fmt.Printf("failed GenerateKey with %s.", err2)
	}
	fmt.Printf("flag[%T][%X]\n", c, c)
	//address_02 := crypto.CreateAddressForAccountType(address, uint64(123456), 0x02)
	//fmt.Printf("address2[%d][%v]\n", len(address_02), address_02)

	/*acc1Key, _ := crypto.HexToECDSA(hexPriKey)
	fmt.Println("public key from prikey  \n", hexutil.Encode(crypto.FromECDSAPub(&acc1Key.PublicKey)))
	address1 := crypto.PubkeyToAddress(acc1Key.PublicKey).Hex()
	fmt.Println("address1 ", address1)*/

}

func TestCreateAddressForProbeAccountType(t *testing.T) {

	k := byte(0x02)
	key, err := probe.GenerateKeyByType(k)
	if err != nil {
		fmt.Println("Error: ", err.Error())
	}
	hexPriKey := hex.EncodeToString(probe.FromECDSA(key))
	//不含0x的私钥65
	fmt.Printf("private key [%d] [%v]\n", len(hexPriKey), hexPriKey)
	//Get the address
	address := probe.PubkeyToAddress(key.PublicKey)
	fmt.Printf("address[%d][%v]\n", len(address), address)

	var c, err2 = common.ValidCheckAddress(address.Hex())
	if err2 != nil {
		fmt.Printf("failed GenerateKey with %s.", err2)
	}
	fmt.Printf("flag[%T][%X]\n", c, c)

	acc1Key, _ := probe.HexToECDSA(hexPriKey)
	fmt.Println("public key from prikey  \n", hexutil.Encode(probe.FromECDSAPub(&acc1Key.PublicKey)))
	address1 := probe.PubkeyToAddress(acc1Key.PublicKey).Hex()
	fmt.Println("address1 ", address1)
}

func TestSign(*testing.T) {

	//生成公私钥
	k := byte(0x02)
	key, err := probe.GenerateKeyByType(k)
	if err != nil {
		fmt.Println("Error: ", err.Error())
	}
	hexPriKey := hex.EncodeToString(probe.FromECDSA(key))
	//不含0x的私钥65
	fmt.Printf("private key [%d] [%v]\n", len(hexPriKey), hexPriKey)
	//Get the address
	address := probe.PubkeyToAddress(key.PublicKey)
	fmt.Printf("address[%d][%v]\n", len(address), address)

	key1, _ := crypto.HexToECDSA(hexPriKey)
	addrtest := crypto.PubkeyToAddressForType(key1.PublicKey, k).Hex()
	//测试数据签名
	msg := crypto.Keccak256([]byte("foo"))
	sig, err := crypto.Sign(msg, key1)
	if err != nil {
		fmt.Println("Error: ", err.Error())
	}
	k = sig[len(sig)-1]
	sig = sig[:len(sig)-1]
	recoveredPub, err := crypto.Ecrecover(msg, sig)
	pubKey, _ := crypto.UnmarshalPubkeyForType(recoveredPub, k)
	recoveredAddr := crypto.PubkeyToAddressForType(*pubKey, k)

	// should be equal to SigToPub
	recoveredPub2, _ := crypto.SigToPubForType(msg, sig, k)
	recoveredAddr2 := crypto.PubkeyToAddressForType(*recoveredPub2, k)
	//验证签名
	fmt.Println("addrtest ", addrtest)
	fmt.Println("recoveredAddr ", recoveredAddr.String())
	fmt.Println("recoveredAddr2 ", recoveredAddr2.String())

}

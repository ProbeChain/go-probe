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
	key, err := probe.GenerateKeyByType(0x00)
	if err != nil {
		fmt.Println("Error: ", err.Error())
	}
	hexPriKey := hex.EncodeToString(probe.FromECDSA(key))
	//不含0x的私钥65
	fmt.Printf("private key [%d] [%v]\n", len(hexPriKey), hexPriKey)
	//Get the address
	address := probe.PubkeyToAddress(key.PublicKey).Hex()
	fmt.Printf("address[%d][%v]\n", len(address), address)
	//	address2 := PubkeyToNormalAddress(key.PublicKey).Hex()
	//fmt.Printf("address[%d][%v]\n", len(address2), address2)

	acc1Key, _ := probe.HexToECDSA(hexPriKey)
	fmt.Println("public key from prikey  \n", hexutil.Encode(probe.FromECDSAPub(&acc1Key.PublicKey)))
	address1 := probe.PubkeyToAddress(acc1Key.PublicKey).Hex()
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
	address_02, _ := probe.CreateAddressForAccountType(address, uint64(123456), 0x02)
	fmt.Printf("address2[%d][%v]\n", len(address_02), address_02)

}

func TestCreateAddressForProbeGenerateKeyByType(t *testing.T) {

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

func TestSign01(*testing.T) {
	k := byte(0x03)
	key, err := probe.GenerateKeyByType(k)
	if err != nil {
		fmt.Println("Error: ", err.Error())
	}
	hexPriKey := hex.EncodeToString(probe.FromECDSA(key))
	//带有0x的私钥
	fmt.Println("private key have 0x   n", hexutil.Encode(probe.FromECDSA(key)))
	//不含0x的私钥65
	fmt.Printf("private key [%d] [%v]\n", len(hexPriKey), hexPriKey)
	//Get the address
	address := probe.PubkeyToAddress(key.PublicKey)
	fmt.Printf("address[%d][%v]\n", len(address), address.Hex())

	hexStr := "5eabdc3deb6c6caaa80e063d6f11784ff06a57c6a06d43b7ede9a805e5fc29b2"
	//hexPriKey := "033b2dd38d41445e25d626808d39c3359117c5ba9145740cd38a3b430f13153c97"
	digestHash, _ := hex.DecodeString(hexStr)
	key1, _ := probe.HexToECDSA(hexPriKey)
	sig, _ := crypto.Sign(digestHash, key1)
	fmt.Println("sig ", hex.EncodeToString(sig))
	recoveredPub, _ := crypto.Ecrecover(digestHash, sig)
	pubKey, _ := probe.UnmarshalPubkey(recoveredPub)
	recoveredAddr := probe.PubkeyToAddress(*pubKey)

	recoveredPub2, _ := crypto.SigToPub(digestHash, sig)
	recoveredAddr2 := probe.PubkeyToAddress(*recoveredPub2)
	tt := probe.FromECDSAPub(&key1.PublicKey)
	//addrtest:=hexutil.Encode(probe.FromECDSAPub(&key1.PublicKey))
	addrtest := hexutil.Encode(tt)
	fmt.Println("addrtest ", addrtest)
	fmt.Println("recoveredAddr ", recoveredAddr.String())
	fmt.Println("recoveredAddr2 ", recoveredAddr2.String())
}
func TestSign(*testing.T) {

	/*//生成公私钥
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

	key1, _ := probe.HexToECDSA(hexPriKey)
	//addrtest := probe.PubkeyToAddressForType(key1.PublicKey, k).Hex()
	//测试数据签名
	msg := probe.Keccak256([]byte("foo"))
	sig, err := probe.Sign(msg, key1)
	if err != nil {
		fmt.Println("Error: ", err.Error())
	}
	k = sig[len(sig)-1]
	sig = sig[:len(sig)-1]
	recoveredPub, err := probe.Ecrecover(msg, sig)
	pubKey, _ := probe.UnmarshalPubkeyForType(recoveredPub, k)
	recoveredAddr := probe.PubkeyToAddressForType(*pubKey, k)

	// should be equal to SigToPub
	recoveredPub2, _ := probe.SigToPubForType(msg, sig, k)
	recoveredAddr2 := probe.PubkeyToAddressForType(*recoveredPub2, k)
	//验证签名
	//fmt.Println("addrtest ", addrtest)
	fmt.Println("recoveredAddr ", recoveredAddr.String())
	fmt.Println("recoveredAddr2 ", recoveredAddr2.String())*/

	key, err := probe.GenerateKeyByType(0x03)
	if err != nil {
		fmt.Println("failed GenerateKey with: ", err.Error())
	}

	fmt.Println("private key have 0x   \n", hexutil.Encode(probe.FromECDSA(key)))
	fmt.Println("private key no 0x \n", hex.EncodeToString(probe.FromECDSA(key)))

	/*if err := probe.SaveECDSA("privatekey", key); err != nil {
		log.Error(fmt.Sprintf("Failed to persist node key: %v", err))
	}*/

	fmt.Println("public key have 0x   \n", hexutil.Encode(probe.FromECDSAPub(&key.PublicKey)))
	fmt.Println("public key no 0x \n", hex.EncodeToString(probe.FromECDSAPub(&key.PublicKey)))

	//由私钥字符串转换私钥
	acc1Key, _ := probe.HexToECDSA("0342875f5170623c91e184b9e91d7c1dd381d4e3a9af9ccba2e656626005baf21a")
	address1 := probe.PubkeyToAddress(acc1Key.PublicKey)
	fmt.Println("address ", address1.String())
	fmt.Println("************************** ")
	dummyAddr := common.HexToAddress("031C98b32Cf0990eCAeB2706E3Fb70F6ad04663c199dC96463")
	fmt.Println("dummyAddr", dummyAddr.String())
	fmt.Println("address ", common.BytesToHash(common.FromHex("031C98b32Cf0990eCAeB2706E3Fb70F6ad04663c199dC96463")))
	fmt.Println("priveaddress ", common.BytesToHash(common.FromHex("03faeb343468fdb38cd39114af7b6b9a3452768116fed047623c138100d9bd4e4e")))
	/*//字节转地址
	addr3      := common.BytesToAddress([]byte("ethereum"))
	fmt.Println("address ",addr3.String())

	//字节转hash
	hash1 := common.BytesToHash([]byte("topic1"))
	fmt.Println("hash ",hash1.String())*/

	var testAddrHex = "031C98b32Cf0990eCAeB2706E3Fb70F6ad04663c199dC96463"
	var testPrivHex = "03faeb343468fdb38cd39114af7b6b9a3452768116fed047623c138100d9bd4e4e"
	key1, _ := probe.HexToECDSA(testPrivHex)
	addrtest := common.HexToAddress(testAddrHex)

	msg := probe.Keccak256([]byte("foo"))
	sig, err := crypto.Sign(msg, key1)
	recoveredPub, err := crypto.Ecrecover(msg, sig)
	pubKey, _ := probe.UnmarshalPubkey(recoveredPub)
	recoveredAddr := probe.PubkeyToAddress(*pubKey)

	// should be equal to SigToPub
	recoveredPub2, _ := crypto.SigToPub(msg, sig)
	recoveredAddr2 := probe.PubkeyToAddress(*recoveredPub2)

	fmt.Println("addrtest ", addrtest.String())
	fmt.Println("recoveredAddr ", recoveredAddr.String())
	fmt.Println("recoveredAddr2 ", recoveredAddr2.String())
}

func TestHexToAddress(*testing.T) {
	key, err := probe.GenerateKeyByType(0x04)
	if err != nil {
		fmt.Println("failed GenerateKey with: ", err.Error())
	}

	fmt.Println("private key have 0x   \n", hexutil.Encode(probe.FromECDSA(key)))
	fmt.Println("private key no 0x \n", hex.EncodeToString(probe.FromECDSA(key)))

	/*	if err := probe.SaveECDSA("privatekey", key); err != nil {
		log.Error(fmt.Sprintf("Failed to persist node key: %v", err))
	}*/

	fmt.Println("public key have 0x   \n", hexutil.Encode(probe.FromECDSAPub(&key.PublicKey)))
	fmt.Println("public key no 0x \n", hex.EncodeToString(probe.FromECDSAPub(&key.PublicKey)))
	oldPkAddr := probe.PubkeyToAddress(key.PublicKey)

	fmt.Println("oldPkAddr", oldPkAddr.String())
	fmt.Println("address ", common.BytesToHash(common.FromHex(oldPkAddr.String())))
	pkHex := oldPkAddr.String()
	b, _ := hexutil.Decode(pkHex)
	pkAddr := common.BytesToAddress(b)
	fmt.Printf("oldPkAddr have 0X %s,pkAddr have 0x  %s \n", hexutil.Encode(oldPkAddr[:]), hexutil.Encode(pkAddr[:]))
}

func TestStrToHex(*testing.T) {
	str := "{\"ip\":\"192.168.0.1\",\"port\":\"1307\"}"
	hexutil.Encode([]byte(str))

	fmt.Println("public key have 0x   \n", hexutil.Encode([]byte(str)))

	str2 := "{\"ip\":\"192.169.1.3\",\"port\":\"1307\"}"
	fmt.Println("public key have 0x   \n", hexutil.Encode([]byte(str2)))

	var emptyCodeHash = crypto.Keccak256(nil)
	fmt.Println("address ", hexutil.Encode(emptyCodeHash))

	var emptyCodeHash2 = probe.Keccak256(nil)
	fmt.Println("address ", hexutil.Encode(emptyCodeHash2))
}

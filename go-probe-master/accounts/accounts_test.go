// Copyright 2015 The ProbeChain Authors
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

package accounts

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/probechain/go-probe/common"
	"github.com/probechain/go-probe/common/hexutil"
	"github.com/probechain/go-probe/crypto"
	"github.com/probechain/go-probe/p2p/enode"
	"math/big"
	"net"
	"testing"
)

func TestTextHash(t *testing.T) {
	hash := TextHash([]byte("Hello Joe"))
	want := hexutil.MustDecode("0x6f1788b6cef82c22e7150b27ecf47b602a2a1e770a2bb6a37fda322ff0e64111")
	if !bytes.Equal(hash, want) {
		t.Fatalf("wrong hash: %x", hash)
	}
}

func TestGenerate(t *testing.T) {
	//Create an account
	key, err := crypto.GenerateKey()
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
	//crypto.FromECDSAPub(&key)[1:]
	fmt.Println("public key from prikey  \n", hexutil.Encode(crypto.FromECDSAPub(&acc1Key.PublicKey)))
	address1 := crypto.PubkeyToAddress(acc1Key.PublicKey).Hex()
	fmt.Println("address1 ", address1)

	var c, err2 = common.ValidCheckAddress(address)
	if err2 != nil {
		fmt.Printf("failed GenerateKey with %s.", err2)
	}
	fmt.Printf("flag[%T][%X]\n", c, c)

}

func TestCreateAddress(t *testing.T) {
	//Create an account
	key, err := crypto.GenerateKey()
	if err != nil {
		fmt.Println("Error: ", err.Error())
	}
	hexPriKey := hex.EncodeToString(crypto.FromECDSA(key))
	//不含0x的私钥65
	fmt.Printf("private key [%d] [%v]\n", len(hexPriKey), hexPriKey)
	//Get the address
	address := crypto.PubkeyToAddress(key.PublicKey)
	fmt.Printf("address[%d][%v]\n", len(address), address)

	var c, err2 = common.ValidCheckAddress(address.Hex())
	if err2 != nil {
		fmt.Printf("failed GenerateKey with %s.", err2)
	}

	address2 := "0X02219BC9DA0E58CF135C032533BDE56F0C40699E16A411EE71"
	var _, err3 = common.ValidCheckAddress(address2)
	if err3 != nil {
		fmt.Printf("failed GenerateKey with %s.", err2)
	}
	fmt.Printf("flag[%T][%X]\n", c, c)
	address_02 := crypto.CreateAddress(address, uint64(123456))
	fmt.Printf("address2[%d][%v]\n", len(address_02), address_02)

}

func TestCreateAddressForProbeGenerateKeyByType(t *testing.T) {

	key, err := crypto.GenerateKey()
	if err != nil {
		fmt.Println("Error: ", err.Error())
	}
	hexPriKey := hex.EncodeToString(crypto.FromECDSA(key))
	//不含0x的私钥65
	fmt.Printf("private key [%d] [%v]\n", len(hexPriKey), hexPriKey)
	//Get the address
	address := crypto.PubkeyToAddress(key.PublicKey)
	fmt.Printf("address[%d][%v]\n", len(address), address)

	var c, err2 = common.ValidCheckAddress(address.Hex())
	if err2 != nil {
		fmt.Printf("failed GenerateKey with %s.", err2)
	}
	fmt.Printf("flag[%T][%X]\n", c, c)

	acc1Key, _ := crypto.HexToECDSA(hexPriKey)
	fmt.Println("public key from prikey  \n", hexutil.Encode(crypto.FromECDSAPub(&acc1Key.PublicKey)))
	address1 := crypto.PubkeyToAddress(acc1Key.PublicKey).Hex()
	fmt.Println("address1 ", address1)
}

func TestSign01(*testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		fmt.Println("Error: ", err.Error())
	}
	hexPriKey := hex.EncodeToString(crypto.FromECDSA(key))
	//带有0x的私钥
	fmt.Println("private key have 0x   n", hexutil.Encode(crypto.FromECDSA(key)))
	//不含0x的私钥65
	fmt.Printf("private key [%d] [%v]\n", len(hexPriKey), hexPriKey)
	//Get the address
	address := crypto.PubkeyToAddress(key.PublicKey)
	fmt.Printf("address[%d][%v]\n", len(address), address.Hex())

	hexStr := "5eabdc3deb6c6caaa80e063d6f11784ff06a57c6a06d43b7ede9a805e5fc29b2"
	//hexPriKey := "033b2dd38d41445e25d626808d39c3359117c5ba9145740cd38a3b430f13153c97"
	digestHash, _ := hex.DecodeString(hexStr)
	key1, _ := crypto.HexToECDSA(hexPriKey)
	sig, _ := crypto.Sign(digestHash, key1)
	fmt.Println("sig ", hex.EncodeToString(sig))
	recoveredPub, _ := crypto.Ecrecover(digestHash, sig)
	pubKey, _ := crypto.UnmarshalPubkey(recoveredPub)
	recoveredAddr := crypto.PubkeyToAddress(*pubKey)

	recoveredPub2, _ := crypto.SigToPub(digestHash, sig)
	recoveredAddr2 := crypto.PubkeyToAddress(*recoveredPub2)
	tt := crypto.FromECDSAPub(&key1.PublicKey)
	//addrtest:=hexutil.Encode(crypto.FromECDSAPub(&key1.PublicKey))
	addrtest := hexutil.Encode(tt)
	fmt.Println("addrtest ", addrtest)
	fmt.Println("recoveredAddr ", recoveredAddr.String())
	fmt.Println("recoveredAddr2 ", recoveredAddr2.String())
}
func TestHexToAddress(*testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		fmt.Println("failed GenerateKey with: ", err.Error())
	}

	fmt.Println("private key have 0x   \n", hexutil.Encode(crypto.FromECDSA(key)))
	fmt.Println("private key no 0x \n", hex.EncodeToString(crypto.FromECDSA(key)))

	/*	if err := crypto.SaveECDSA("privatekey", key); err != nil {
		log.Error(fmt.Sprintf("Failed to persist node key: %v", err))
	}*/

	fmt.Println("public key have 0x   \n", hexutil.Encode(crypto.FromECDSAPub(&key.PublicKey)))
	fmt.Println("public key no 0x \n", hex.EncodeToString(crypto.FromECDSAPub(&key.PublicKey)))
	oldPkAddr := crypto.PubkeyToAddress(key.PublicKey)

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

	str2 := "{\"enode\":\"0076d15321c35e84d5e31c5bc344b93106af04cc97f21e1840664dd9561cbc6a6c362e01654a8bc41145155c58d2ca613ec6f165c3fc4fe7626be0734467730652\",\"ip\":\"172.16.0.103\",\"port\":\"40000\"}"
	fmt.Println("public key have 0x   \n", hexutil.Encode([]byte(str2)))

	/*var emptyCodeHash = crypto.Keccak256(nil)
	fmt.Println("address ", hexutil.Encode(emptyCodeHash))

	var emptyCodeHash2 = crypto.Keccak256(nil)
	fmt.Println("address ", hexutil.Encode(emptyCodeHash2))*/
}


func TestCheckValidteTest(*testing.T) {
	var c, err2 = common.ValidCheckAddress("0x03b16d6687a30d93bef1a4d80952e8323f8758c1cd8c0c148a")
	if err2 != nil {
		fmt.Printf("failed GenerateKey with %s.", err2)
	}
	fmt.Printf("flag[%T][%X]\n", c, c)
}

type ResolveUDPAddrTest struct {
	network       string
	litAddrOrName string
	addr          *net.UDPAddr
	err           error
}

func TestPrintValidatorNode(*testing.T) {
	//privateKey, _ := crypto.GenerateKey()
	privateKey, _ := crypto.HexToECDSA("7308dacbb9ba9b3c97a14ef0faac7ccfb7851ccb003936a14d36d2ced0bf7087")
	fmt.Println("private key have 0x   \n", hexutil.Encode(crypto.FromECDSA(privateKey)))
	address := crypto.PubkeyToAddress(privateKey.PublicKey)
	fmt.Println("address ", address.String())
	netPort := 30302
	tt := ResolveUDPAddrTest{"udp4", "[::]:30302", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: netPort}, nil}
	addr, _ := net.ResolveUDPAddr(tt.network, tt.litAddrOrName)

	fmt.Println("public key have 0x   \n", hexutil.Encode(crypto.FromECDSAPub(&privateKey.PublicKey)))
	n := enode.NewV4(&privateKey.PublicKey, addr.IP, netPort, addr.Port)
	fmt.Println("address-nodeKey:", n.URLv4())

}

func TestBigIntAdd(*testing.T) {
	a := big.NewInt(1)
	b := big.NewInt(2)
	a.Add(a, b)
	fmt.Printf("a = %v    b = %v   a = %v\n", a, b, a)
}

func TestAccounTypeFoGenrateSign(*testing.T) {
	var testAddrHex = "007245ec242315371bA8E44BAA39e5c0AaC14De2620B6b8Cb4"
	toAddress := common.HexToAddress(testAddrHex)
	nonce := uint64(2)
	contractAddr := crypto.CreateAddress(toAddress, nonce)
	//contractAddr = common.HexToAddress("03f112c97935863463bc34871f506A9A8c3741a1CE0f8F60c9")
	fmt.Println("contractAddr", contractAddr.String())
}

func TestDigest(*testing.T) {
	adr1 := common.HexToAddress("0x28fd633B72cA9828542A7dA8E3426E11C831D4Bd")
	adr2 := common.HexToAddress("0x897638B555Fa1584965A1E1c4d4302264ac9432b")
	randomNum := uint32(123456)
	var buffer bytes.Buffer
	buffer.Write(adr1.Bytes())
	buffer.Write(adr2.Bytes())
	buffer.Write(new(big.Int).SetUint64(uint64(randomNum)).Bytes())
	h := crypto.Keccak256Hash(buffer.Bytes())
	fmt.Println(h)
}

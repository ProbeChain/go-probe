// Copyright 2019 The go-probeum Authors
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

package state

import (
	"bytes"
	"fmt"
	"github.com/probechain/go-probe/common"
	"github.com/probechain/go-probe/rlp"
	"math/big"
	"testing"
)

func BenchmarkCutOriginal(b *testing.B) {
	value := common.HexToHash("0x01")
	for i := 0; i < b.N; i++ {
		bytes.TrimLeft(value[:], "\x00")
	}
}

func BenchmarkCutsetterFn(b *testing.B) {
	value := common.HexToHash("0x01")
	cutSetFn := func(r rune) bool { return r == 0 }
	for i := 0; i < b.N; i++ {
		bytes.TrimLeftFunc(value[:], cutSetFn)
	}
}

func BenchmarkCutCustomTrim(b *testing.B) {
	value := common.HexToHash("0x01")
	for i := 0; i < b.N; i++ {
		common.TrimLeftZeroes(value[:])
	}
}

func TestStateObjectDPosListAccount(t *testing.T) {
	candidateDPOSAccounts := []common.DPoSCandidateAccount{
		{common.BytesToDposEnode([]byte("enode://04d8169d48d39be0b120321c00cdec41723eb622bfc7d9f86cddde848e3477f9d062d6b8c9dac7f300b1f30a571f83004730fc96b94cf93bae93b3b64ec42d4e45@127.0.0.1:30001")), common.HexToAddress("0x00Dbe1397F55F3b0A29CB3E075D84a21c6ebc7F709cd4Aa6e8"), common.HexToAddress("0x04dEF296C8BC7fd08c54D4455208dFFFf920387401A8AEE022"), big.NewInt(1000)},
		{common.BytesToDposEnode([]byte("enode://04fe3919ab384a77c7b046b89948b0f4a54d3edb3fef40423d9a485d3e4493610ede73d9e602799d0fec519956b9712296420445256a64aeb7c515b03b327d87d3@127.0.0.1:30002")), common.HexToAddress("0x003392909cAfB2bad305ce6287dEe4cB8e151bB0D5E5075596"), common.HexToAddress("0x043ee8e34b94be2f0539335E74e5155c541a6dEd4324Af50E0"), big.NewInt(2000)},
		{common.BytesToDposEnode([]byte("enode://04d8169d48d39be0b120321c00cdec41723eb622bfc7d9f86cddde848e3477f9d062d6b8c9dac7f300b1f30a571f83004730fc96b94cf93bae93b3b64ec42d4e45@127.0.0.1:30001")), common.HexToAddress("0x00Dbe1397F55F3b0A29CB3E075D84a21c6ebc7F709cd4Aa6e8"), common.HexToAddress("0x04306AE9D686bAcf68A91434683Af58F8E6e73BC638E65F86A"), big.NewInt(1000)},
		{common.BytesToDposEnode([]byte("enode://04fe3919ab384a77c7b046b89948b0f4a54d3edb3fef40423d9a485d3e4493610ede73d9e602799d0fec519956b9712296420445256a64aeb7c515b03b327d87d3@127.0.0.1:30002")), common.HexToAddress("0x003392909cAfB2bad305ce6287dEe4cB8e151bB0D5E5075596"), common.HexToAddress("0x04C8f1c2cBe39E9576226Bc08bAd22F4CDC8ad0a6c5A079d1F"), big.NewInt(2000)},
		{common.BytesToDposEnode([]byte("enode://04d8169d48d39be0b120321c00cdec41723eb622bfc7d9f86cddde848e3477f9d062d6b8c9dac7f300b1f30a571f83004730fc96b94cf93bae93b3b64ec42d4e45@127.0.0.1:30001")), common.HexToAddress("0x00Dbe1397F55F3b0A29CB3E075D84a21c6ebc7F709cd4Aa6e8"), common.HexToAddress("0x046b46ab27cBbC3eC0c01Acb21C81bB0Ad285690e77B37968d"), big.NewInt(1000)},
		{common.BytesToDposEnode([]byte("enode://04fe3919ab384a77c7b046b89948b0f4a54d3edb3fef40423d9a485d3e4493610ede73d9e602799d0fec519956b9712296420445256a64aeb7c515b03b327d87d3@127.0.0.1:30002")), common.HexToAddress("0x003392909cAfB2bad305ce6287dEe4cB8e151bB0D5E5075596"), common.HexToAddress("0x04DA3572dcAc2354E633754153a7B02719dCc59B6b0700574c"), big.NewInt(2000)},
	}
	/*	a, _ := rlp.EncodeToBytes(candidateDPOSAccounts)
		c := new(dPosCandidateAccounts)
		rlp.DecodeBytes(a, &c)
		fmt.Println(12233)*/
	a := DPosListAccount{
		DPosCandidateAccounts: candidateDPOSAccounts,
		RoundId:               0,
		AccType:               7,
	}
	b, _ := rlp.EncodeToBytes(a)

	c := new(DPosListAccount)
	rlp.DecodeBytes(b, &c)
	fmt.Println(12233)

}

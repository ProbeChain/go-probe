package state

import (
	"fmt"
	"github.com/probechain/go-probe/common"
	"math/big"
	"strings"
	"testing"
)

func TestValidatorCandidate(t *testing.T) {
	candidateValidators := []common.DPoSCandidateAccount{
		{common.BytesToValidatorEnode([]byte("enode://04d8169d48d39be0b120321c00cdec41723eb622bfc7d9f86cddde848e3477f9d062d6b8c9dac7f300b1f30a571f83004730fc96b94cf93bae93b3b64ec42d4e45@127.0.0.1:30001")), common.HexToAddress("0x00Dbe1397F55F3b0A29CB3E075D84a21c6ebc7F709cd4Aa6e8"), common.HexToAddress("0x04dEF296C8BC7fd08c54D4455208dFFFf920387401A8AEE022"), big.NewInt(1000)},
		{common.BytesToValidatorEnode([]byte("enode://04fe3919ab384a77c7b046b89948b0f4a54d3edb3fef40423d9a485d3e4493610ede73d9e602799d0fec519956b9712296420445256a64aeb7c515b03b327d87d3@127.0.0.1:30002")), common.HexToAddress("0x003392909cAfB2bad305ce6287dEe4cB8e151bB0D5E5075596"), common.HexToAddress("0x043ee8e34b94be2f0539335E74e5155c541a6dEd4324Af50E0"), big.NewInt(2000)},
		{common.BytesToValidatorEnode([]byte("enode://04d8169d48d39be0b120321c00cdec41723eb622bfc7d9f86cddde848e3477f9d062d6b8c9dac7f300b1f30a571f83004730fc96b94cf93bae93b3b64ec42d4e45@127.0.0.1:30003")), common.HexToAddress("0x00Dbe1397F55F3b0A29CB3E075D84a21c6ebc7F709cd4Aa6e8"), common.HexToAddress("0x04306AE9D686bAcf68A91434683Af58F8E6e73BC638E65F86A"), big.NewInt(1000)},
		{common.BytesToValidatorEnode([]byte("enode://04fe3919ab384a77c7b046b89948b0f4a54d3edb3fef40423d9a485d3e4493610ede73d9e602799d0fec519956b9712296420445256a64aeb7c515b03b327d87d3@127.0.0.1:30004")), common.HexToAddress("0x003392909cAfB2bad305ce6287dEe4cB8e151bB0D5E5075596"), common.HexToAddress("0x04C8f1c2cBe39E9576226Bc08bAd22F4CDC8ad0a6c5A079d1F"), big.NewInt(2000)},
		{common.BytesToValidatorEnode([]byte("enode://04d8169d48d39be0b120321c00cdec41723eb622bfc7d9f86cddde848e3477f9d062d6b8c9dac7f300b1f30a571f83004730fc96b94cf93bae93b3b64ec42d4e45@127.0.0.1:30005")), common.HexToAddress("0x00Dbe1397F55F3b0A29CB3E075D84a21c6ebc7F709cd4Aa6e8"), common.HexToAddress("0x046b46ab27cBbC3eC0c01Acb21C81bB0Ad285690e77B37968d"), big.NewInt(1000)},
		{common.BytesToValidatorEnode([]byte("enode://04fe3919ab384a77c7b046b89948b0f4a54d3edb3fef40423d9a485d3e4493610ede73d9e602799d0fec519956b9712296420445256a64aeb7c515b03b327d87d3@127.0.0.1:30004")), common.HexToAddress("0x003392909cAfB2bad305ce6287dEe4cB8e151bB0D5E5075596"), common.HexToAddress("0x04DA3572dcAc2354E633754153a7B02719dCc59B6b0700574c"), big.NewInt(2000)},
	}
	a := ValidatorListAccount{
		ValidatorCandidates: candidateValidators,
		RoundId:               0,
		AccType:               7,
	}

	c := a.ValidatorCandidates.GetPresetDPosAccounts()
	for _, v := range c {
		fmt.Println(v.Enode.String())
	}
	fmt.Println(len(c))
}

func TestValidatorCandidate2(t *testing.T) {
	/*	a1 := common.DPoSCandidateAccount{common.BytesToValidatorEnode([]byte("enode://@192.168.0.4:80")), [20]byte{1}, [20]byte{1}, big.NewInt(100)}
		a2 := common.DPoSCandidateAccount{common.BytesToValidatorEnode([]byte("enode://@192.168.0.4:80")), [20]byte{1}, [20]byte{2}, big.NewInt(200)}
		a3 := common.DPoSCandidateAccount{common.BytesToValidatorEnode([]byte("enode://@192.168.0.4:80")), [20]byte{1}, [20]byte{3}, big.NewInt(300)}
		for i, aa := range presetDPosAccounts3 {
			fmt.Printf("%d, Owner:%s,Vote:%s,VoteValue:%d,Enode:%s\n", i+1, aa.Owner, aa.VoteAccount, aa.VoteValue, parseIp(aa.Enode))
		}*/

	point := common.GetLastConfirmPoint(3000, 3000)
	fmt.Println("ret:", point)
}

func parseIp(enode common.ValidatorEnode) string {
	s := string(enode[:])
	i := strings.Index(s, "@")
	return string([]byte(s)[i:])
}

func TestValidatorCandidate3(t *testing.T) {
	adr1 := common.HexToAddress("0x00Dbe1397F55F3b0A29CB3E075D84a21c6ebc7F709cd4Aa6e8")
	adr2 := common.HexToAddress("0x003392909cAfB2bad305ce6287dEe4cB8e151bB0D5E5075596")
	fmt.Println(adr1.Hash().Big())
	fmt.Println(adr2.Hash().Big())
	if adr1.Hash().Big().Cmp(adr2.Hash().Big()) == 1 {
		fmt.Println("dfsfs")
	}

}

func TestValidatorCandidate4(t *testing.T) {
	adr := common.HexToAddress("0x28fd633B72cA9828542A7dA8E3426E11C831D4Bd")
	bytes := adr.Bytes()
	fmt.Println(bytes) //[40 253 99 59 114 202 152 40 84 42 125 168 227 66 110 17 200 49 212 189]
	last10Bytes := bytes[10:]
	fmt.Println(last10Bytes) //[125 168 227 66 110 17 200 49 212 189]
	a := new(big.Int).SetBytes(last10Bytes)
	fmt.Println(a.Uint64())
	fmt.Println(a.Uint64() % 1024)

	bytes2 := []byte{255, 255, 255, 255, 255, 255, 255, 255}
	str2 := new(big.Int).SetBytes(bytes2)
	fmt.Println(str2.String())
	fmt.Println(str2.Uint64()) //18446744073709551615
	//18446744073709551615
	fmt.Println(str2.Uint64() % 1024)

}

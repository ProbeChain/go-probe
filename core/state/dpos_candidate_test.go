package state

import (
	"fmt"
	"github.com/probeum/go-probeum/common"
	"math/big"
	"strings"
	"testing"
)

func TestDPosCandidate(t *testing.T) {
	/*	candidateDPOSAccounts := []common.DPoSCandidateAccount{
		{common.BytesToDposEnode([]byte("enode://@192.168.0.4:80")), [25]byte{1}, [25]byte{1}, big.NewInt(1), big.NewInt(100)},
		{common.BytesToDposEnode([]byte("enode://@192.168.0.4:80")), [25]byte{1}, [25]byte{1}, big.NewInt(2), big.NewInt(200)},
		{common.BytesToDposEnode([]byte("enode://@192.168.0.4:80")), [25]byte{1}, [25]byte{1}, big.NewInt(3), big.NewInt(1000)}, // 2

		{common.BytesToDposEnode([]byte("enode://@192.168.0.4:81")), [25]byte{1}, [25]byte{2}, big.NewInt(1), big.NewInt(200)}, //
		{common.BytesToDposEnode([]byte("enode://@192.168.0.4:80")), [25]byte{1}, [25]byte{2}, big.NewInt(1), big.NewInt(100)},
		{common.BytesToDposEnode([]byte("enode://@192.168.0.4:80")), [25]byte{1}, [25]byte{2}, big.NewInt(1), big.NewInt(100)}, // 1

		{common.BytesToDposEnode([]byte("enode://@192.168.0.4:80")), [25]byte{0}, [25]byte{3}, big.NewInt(1), big.NewInt(100)},
		{common.BytesToDposEnode([]byte("enode://@192.168.0.4:80")), [25]byte{0}, [25]byte{3}, big.NewInt(2), big.NewInt(200)},
		{common.BytesToDposEnode([]byte("enode://@192.168.0.4:82")), [25]byte{0}, [25]byte{3}, big.NewInt(3), big.NewInt(300)}, // 3

		{common.BytesToDposEnode([]byte("enode://@192.168.0.4:83")), [25]byte{0}, [25]byte{4}, big.NewInt(3), big.NewInt(300)},
		{common.BytesToDposEnode([]byte("enode://@192.168.0.4:84")), [25]byte{0}, [25]byte{5}, big.NewInt(3), big.NewInt(300)},
		{common.BytesToDposEnode([]byte("enode://@192.168.0.4:85")), [25]byte{0}, [25]byte{6}, big.NewInt(3), big.NewInt(300)},
		{common.BytesToDposEnode([]byte("enode://@192.168.0.4:86")), [25]byte{0}, [25]byte{7}, big.NewInt(3), big.NewInt(300)},
		{common.BytesToDposEnode([]byte("enode://@192.168.0.4:87")), [25]byte{0}, [25]byte{8}, big.NewInt(3), big.NewInt(300)},
		{common.BytesToDposEnode([]byte("enode://@192.168.0.4:88")), [25]byte{0}, [25]byte{9}, big.NewInt(3), big.NewInt(300)},
		{common.BytesToDposEnode([]byte("enode://@192.168.0.4:89")), [25]byte{0}, [25]byte{10}, big.NewInt(3), big.NewInt(300)},
	}*/

	candidateDPOSAccounts := []common.DPoSCandidateAccount{
		{common.BytesToDposEnode([]byte("enode://04d8169d48d39be0b120321c00cdec41723eb622bfc7d9f86cddde848e3477f9d062d6b8c9dac7f300b1f30a571f83004730fc96b94cf93bae93b3b64ec42d4e45@127.0.0.1:30001")), common.HexToAddress("0x00Dbe1397F55F3b0A29CB3E075D84a21c6ebc7F709cd4Aa6e8"), common.HexToAddress("0x04dEF296C8BC7fd08c54D4455208dFFFf920387401A8AEE022"), big.NewInt(1000)},
		{common.BytesToDposEnode([]byte("enode://04fe3919ab384a77c7b046b89948b0f4a54d3edb3fef40423d9a485d3e4493610ede73d9e602799d0fec519956b9712296420445256a64aeb7c515b03b327d87d3@127.0.0.1:30002")), common.HexToAddress("0x003392909cAfB2bad305ce6287dEe4cB8e151bB0D5E5075596"), common.HexToAddress("0x043ee8e34b94be2f0539335E74e5155c541a6dEd4324Af50E0"), big.NewInt(2000)},
		{common.BytesToDposEnode([]byte("enode://04d8169d48d39be0b120321c00cdec41723eb622bfc7d9f86cddde848e3477f9d062d6b8c9dac7f300b1f30a571f83004730fc96b94cf93bae93b3b64ec42d4e45@127.0.0.1:30001")), common.HexToAddress("0x00Dbe1397F55F3b0A29CB3E075D84a21c6ebc7F709cd4Aa6e8"), common.HexToAddress("0x04306AE9D686bAcf68A91434683Af58F8E6e73BC638E65F86A"), big.NewInt(1000)},
		{common.BytesToDposEnode([]byte("enode://04fe3919ab384a77c7b046b89948b0f4a54d3edb3fef40423d9a485d3e4493610ede73d9e602799d0fec519956b9712296420445256a64aeb7c515b03b327d87d3@127.0.0.1:30002")), common.HexToAddress("0x003392909cAfB2bad305ce6287dEe4cB8e151bB0D5E5075596"), common.HexToAddress("0x04C8f1c2cBe39E9576226Bc08bAd22F4CDC8ad0a6c5A079d1F"), big.NewInt(2000)},
		{common.BytesToDposEnode([]byte("enode://04d8169d48d39be0b120321c00cdec41723eb622bfc7d9f86cddde848e3477f9d062d6b8c9dac7f300b1f30a571f83004730fc96b94cf93bae93b3b64ec42d4e45@127.0.0.1:30001")), common.HexToAddress("0x00Dbe1397F55F3b0A29CB3E075D84a21c6ebc7F709cd4Aa6e8"), common.HexToAddress("0x046b46ab27cBbC3eC0c01Acb21C81bB0Ad285690e77B37968d"), big.NewInt(1000)},
		{common.BytesToDposEnode([]byte("enode://04fe3919ab384a77c7b046b89948b0f4a54d3edb3fef40423d9a485d3e4493610ede73d9e602799d0fec519956b9712296420445256a64aeb7c515b03b327d87d3@127.0.0.1:30002")), common.HexToAddress("0x003392909cAfB2bad305ce6287dEe4cB8e151bB0D5E5075596"), common.HexToAddress("0x04DA3572dcAc2354E633754153a7B02719dCc59B6b0700574c"), big.NewInt(2000)},
	}
	for _, dd := range candidateDPOSAccounts {
		GetDPosCandidates().AddDPosCandidate(dd)
	}
	dPosCandidateAccounts := GetDPosCandidates().GetDPosCandidateAccounts()
	for i, aa := range dPosCandidateAccounts {
		fmt.Printf("%d, Owner:%s,Vote:%s,VoteValue:%d,Enode:%s\n", i+1, aa.Owner, aa.Vote, aa.VoteValue, parseIp(aa.Enode))
	}

	fmt.Println("---------presetDPosAccounts-----------")
	presetDPosAccounts := GetDPosCandidates().GetPresetDPosAccounts()
	for i, bb := range presetDPosAccounts {
		fmt.Printf("%d, Owner:%s,Enode:%s\n", i+1, bb.Owner, parseIp(bb.Enode))
	}
	/*
			fmt.Println("---------presetDPosAccounts2-----------")
		presetDPosAccounts2 := GetDPosCandidates().GetPresetDPosAccounts(false)
		for _, bb := range presetDPosAccounts2 {
			fmt.Printf("Owner:%s,Enode:%s\n", bb.Owner, bb.Enode)
		}
	*/
	/*	fmt.Println("---------last-----------")
		presetDPosAccounts3 := GetDPosCandidates().GetDPosCandidateAccounts()
		for i, aa := range presetDPosAccounts3 {
			fmt.Printf("%d, Owner:%s,Vote:%s,VoteValue:%d,Enode:%s\n", i+1, aa.Owner, aa.Vote, aa.VoteValue, parseIp(aa.Enode))
		}*/

}

func TestDPosCandidate2(t *testing.T) {
	a1 := common.DPoSCandidateAccount{common.BytesToDposEnode([]byte("enode://@192.168.0.4:80")), [25]byte{1}, [25]byte{1}, big.NewInt(100)}
	a2 := common.DPoSCandidateAccount{common.BytesToDposEnode([]byte("enode://@192.168.0.4:80")), [25]byte{1}, [25]byte{2}, big.NewInt(200)}
	a3 := common.DPoSCandidateAccount{common.BytesToDposEnode([]byte("enode://@192.168.0.4:80")), [25]byte{1}, [25]byte{3}, big.NewInt(300)}
	GetDPosCandidates().AddDPosCandidate(a1)
	GetDPosCandidates().AddDPosCandidate(a2)
	GetDPosCandidates().AddDPosCandidate(a3)

	GetDPosCandidates().DeleteDPosCandidate(a2)

	presetDPosAccounts3 := GetDPosCandidates().GetDPosCandidateAccounts()
	for i, aa := range presetDPosAccounts3 {
		fmt.Printf("%d, Owner:%s,Vote:%s,VoteValue:%d,Enode:%s\n", i+1, aa.Owner, aa.Vote, aa.VoteValue, parseIp(aa.Enode))
	}
}

func parseIp(enode common.DposEnode) string {
	s := string(enode[:])
	i := strings.Index(s, "@")
	return string([]byte(s)[i:])
}

func TestDPosCandidate3(t *testing.T) {
	adr1 := common.HexToAddress("0x00Dbe1397F55F3b0A29CB3E075D84a21c6ebc7F709cd4Aa6e8")
	adr2 := common.HexToAddress("0x003392909cAfB2bad305ce6287dEe4cB8e151bB0D5E5075596")
	fmt.Println(adr1.Hash().Big())
	fmt.Println(adr2.Hash().Big())
	if adr1.Hash().Big().Cmp(adr2.Hash().Big()) == 1 {
		fmt.Println("dfsfs")
	}

}

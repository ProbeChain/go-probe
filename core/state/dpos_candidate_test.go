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
		{common.BytesToDposEnode([]byte("enode://@192.168.0.4:80")), [25]byte{0}, common.HexToAddress("0x003392909cAfB2bad305ce6287dEe4cB8e151bB0D5E5075596"), big.NewInt(101)},
		{common.BytesToDposEnode([]byte("enode://@192.168.0.4:81")), [25]byte{1}, common.HexToAddress("0x00Dbe1397F55F3b0A29CB3E075D84a21c6ebc7F709cd4Aa6e8"), big.NewInt(100)},
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

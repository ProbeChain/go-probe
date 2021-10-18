package state

import (
	"fmt"
	"github.com/probeum/go-probeum/common"
	"math/big"
	"strings"
	"testing"
)

func TestDPosCandidate(t *testing.T) {
	candidateDPOSAccounts := []DPoSCandidateAccount{
		{common.BytesToDposEnode([]byte("enode://000000000000000000000000@192.168.0.3:80")), [25]byte{0}, byte(0), big.NewInt(1), big.NewInt(100)},
		{common.BytesToDposEnode([]byte("enode://000000000000000000000001@192.168.0.4:80")), [25]byte{1}, byte(0), big.NewInt(2), big.NewInt(500)},
		{common.BytesToDposEnode([]byte("enode://000000000000000000000002@192.168.1.3:80")), [25]byte{2}, byte(0), big.NewInt(3), big.NewInt(400)},
		{common.BytesToDposEnode([]byte("enode://000000000000000000000003@192.168.1.4:80")), [25]byte{3}, byte(0), big.NewInt(4), big.NewInt(400)},
	}

	for _, dd := range candidateDPOSAccounts {
		GetDPosCandidates().AddDPosCandidate(dd)
	}
	dPosCandidateAccounts := GetDPosCandidates().GetDPosCandidateAccounts()
	for _, aa := range *dPosCandidateAccounts {
		fmt.Printf("Owner:%s,Mark:%d,Weight:%d,DelegateValue:%d,Enode:%s\n", aa.Owner, aa.Mark, aa.Weight, aa.DelegateValue, aa.Enode)
	}

	fmt.Println("---------presetDPosAccounts-----------")
	presetDPosAccounts, hasNew := GetDPosCandidates().GetPresetDPosAccounts(false)
	for _, bb := range presetDPosAccounts {
		fmt.Printf("hasNew:%t,Owner:%s,Enode:%s\n", hasNew, bb.Owner, bb.Enode)
	}

	fmt.Println("---------presetDPosAccounts2-----------")
	presetDPosAccounts2, hasNew2 := GetDPosCandidates().GetPresetDPosAccounts(false)
	for _, bb := range presetDPosAccounts2 {
		fmt.Printf("hasNew:%t,Owner:%s,Enode:%s\n", hasNew2, bb.Owner, bb.Enode)
	}

	fmt.Println("---------presetDPosAccounts3-----------")
	presetDPosAccounts3 := GetDPosCandidates().GetDPosCandidateAccounts()
	for _, aa := range *presetDPosAccounts3 {
		fmt.Printf("Owner:%s,Mark:%d,Weight:%d,DelegateValue:%d,Enode:%s\n", aa.Owner, aa.Mark, aa.Weight, aa.DelegateValue, aa.Enode)
	}

}

func TestDPosCandidate2(t *testing.T) {
	candidate := DPoSCandidateAccount{common.BytesToDposEnode([]byte("enode://000000000000000000000000@192.168.0.3:80")), [25]byte{0}, byte(0), big.NewInt(1), big.NewInt(100)}
	s := string(candidate.Enode[:])
	i := strings.Index(s, "enode://")
	a := []byte(s)[i:]
	fmt.Printf("%s\n", string(a))
}

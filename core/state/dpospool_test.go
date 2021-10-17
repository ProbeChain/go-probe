package state

import (
	"fmt"
	"github.com/probeum/go-probeum/common"
	"math/big"
	"testing"
)

func TestDpospool(t *testing.T) {
	candidateDPOSAccounts := []DPoSCandidateAccount{
		{common.BytesToDposEnode([]byte("enode://000000000000000000000000@192.168.0.3:80")), [25]byte{0}, big.NewInt(3), big.NewInt(1), big.NewInt(100)},
		//{common.BytesToDposEnode([]byte("enode://000000000000000000000001@192.168.0.4:80")), [25]byte{1}, big.NewInt(3), big.NewInt(2), big.NewInt(500)},
		//{common.BytesToDposEnode([]byte("enode://000000000000000000000002@192.168.1.3:80")), [25]byte{2}, big.NewInt(3), big.NewInt(3), big.NewInt(400)},
		//{common.BytesToDposEnode([]byte("enode://000000000000000000000003@192.168.1.4:80")), [25]byte{3}, big.NewInt(3), big.NewInt(4), big.NewInt(400)},
	}

	for _, dd := range candidateDPOSAccounts {
		GetDPosList().AddDPosCandidate(dd)
	}
	dPosCandidateAccounts := GetDPosList().GetDPosCandidateAccounts()
	for _, aa := range *dPosCandidateAccounts {
		fmt.Printf("Owner:%s,Height:%d,Weight:%d,DelegateValue:%d,Enode:%s\n", aa.Owner, aa.Height, aa.Weight, aa.DelegateValue, aa.Enode)
	}

	fmt.Println("---------presetDPosAccounts-----------")
	presetDPosAccounts := GetDPosList().GetPresetDPosAccounts(false)
	for _, bb := range presetDPosAccounts {
		fmt.Printf("Owner:%s,Enode:%s\n", bb.Owner, bb.Enode)
	}

	fmt.Println("---------presetDPosAccounts2-----------")
	presetDPosAccounts2 := GetDPosList().GetDPosCandidateAccounts()
	for _, aa := range *presetDPosAccounts2 {
		fmt.Printf("Owner:%s,Height:%d,Weight:%d,DelegateValue:%d,Enode:%s\n", aa.Owner, aa.Height, aa.Weight, aa.DelegateValue, aa.Enode)
	}

}

func TestDpospool2(t *testing.T) {
	epoch := 100
	number := 101 + epoch - 1

	dposNo := number - (number)%epoch
	fmt.Println(dposNo)
}

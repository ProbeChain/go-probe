package state

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
)

func TestBuildHashForDPos(t *testing.T) {
	accounts := []common.DPoSAccount{common.DPoSAccount{
		Enode: []byte{0x01},
		Owner: common.Address{1},
	}, common.DPoSAccount{
		Enode: []byte{0x02},
		Owner: common.Address{2},
	}}
	pos := BuildHashForDPos(accounts)
	fmt.Printf(" after BuildHashForDPos：%v \n", len(pos.Bytes()))
	fmt.Printf(" after BuildHashForDPos hash：%v \n", pos)
}

func TestBuildHashForDPosCandidate(t *testing.T) {
	var accounts = []DPoSCandidateAccount{DPoSCandidateAccount{
		Enode:         []byte{0x01},
		Owner:         common.Address{1},
		Weight:        big.NewInt(1),
		DelegateValue: big.NewInt(1),
	}, DPoSCandidateAccount{
		Enode:         []byte{0x02},
		Owner:         common.Address{2},
		Weight:        big.NewInt(2),
		DelegateValue: big.NewInt(2),
	}}
	pos := BuildHashForDPosCandidate(accounts)
	fmt.Printf(" after BuildHashForDPosCandidate：%v \n", len(pos.Bytes()))
	fmt.Printf(" after BuildHashForDPosCandidate HASH：%v \n", pos)
}

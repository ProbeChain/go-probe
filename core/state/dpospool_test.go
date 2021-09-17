package state

import (
	"fmt"
	"math/big"
	"net"
	"testing"
)

func TestDpospool(t *testing.T) {
	candidateDPOSAccounts := []DPoSCandidateAccount{
		{net.ParseIP("192.168.0.3"), 80, [25]byte{0}, big.NewInt(3), big.NewInt(300)},
		{net.ParseIP("192.168.0.4"), 80, [25]byte{0}, big.NewInt(3), big.NewInt(200)},
		{net.ParseIP("192.168.1.3"), 80, [25]byte{0}, big.NewInt(3), big.NewInt(400)},
		{net.ParseIP("192.168.1.4"), 80, [25]byte{0}, big.NewInt(3), big.NewInt(500)},
	}
	var aSortedLinkedList = NewSortedLinkedList(4, compareValue)
	for _, candidateDPOS := range candidateDPOSAccounts {
		aSortedLinkedList.PutOnTop(candidateDPOS)
	}
	for element := aSortedLinkedList.List.Front(); element != nil; element = element.Next() {
		fmt.Println(element.Value.(DPoSCandidateAccount))
	}
	dposCandidateAccount := DPoSCandidateAccount{net.ParseIP("192.168.2.3"), 80, [25]byte{0}, big.NewInt(5), big.NewInt(400)}
	aSortedLinkedList.PutOnTop(dposCandidateAccount)
	fmt.Println("****************************")
	for element := aSortedLinkedList.List.Front(); element != nil; element = element.Next() {
		fmt.Println(element.Value.(DPoSCandidateAccount))
	}
}

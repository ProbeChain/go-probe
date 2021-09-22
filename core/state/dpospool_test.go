package state

import (
	"fmt"
	"math/big"
	"testing"
)

func TestDpospool(t *testing.T) {
	candidateDPOSAccounts := []DPoSCandidateAccount{
		{[]byte("enode://000000000000000000000000@192.168.0.3:80"), [25]byte{0}, big.NewInt(3), big.NewInt(300)},
		{[]byte("enode://000000000000000000000001@192.168.0.4:80"), [25]byte{1}, big.NewInt(3), big.NewInt(200)},
		{[]byte("enode://000000000000000000000002@192.168.1.3:80"), [25]byte{2}, big.NewInt(3), big.NewInt(400)},
		{[]byte("enode://000000000000000000000003@192.168.1.4:80"), [25]byte{3}, big.NewInt(3), big.NewInt(500)},
	}
	var aSortedLinkedList = NewSortedLinkedList(4, compareValue)
	for _, candidateDPOS := range candidateDPOSAccounts {
		aSortedLinkedList.PutOnTop(candidateDPOS)
	}
	for element := aSortedLinkedList.List.Front(); element != nil; element = element.Next() {
		fmt.Println(element.Value.(DPoSCandidateAccount))
	}
	dposCandidateAccount := DPoSCandidateAccount{[]byte("0@192.168.2.3:80"), [25]byte{0}, big.NewInt(5), big.NewInt(400)}
	aSortedLinkedList.PutOnTop(dposCandidateAccount)
	fmt.Println("****************************")
	for element := aSortedLinkedList.List.Front(); element != nil; element = element.Next() {
		fmt.Println(element.Value.(DPoSCandidateAccount))
	}

}

func TestRemoveDpospool(t *testing.T) {
	candidateDPOSAccounts := []DPoSCandidateAccount{
		{[]byte("enode://000000000000000000000000@192.168.0.3:80"), [25]byte{0}, big.NewInt(3), big.NewInt(300)},
		{[]byte("enode://000000000000000000000001@192.168.0.4:80"), [25]byte{1}, big.NewInt(3), big.NewInt(200)},
		{[]byte("enode://000000000000000000000002@192.168.1.3:80"), [25]byte{2}, big.NewInt(3), big.NewInt(400)},
		{[]byte("enode://000000000000000000000003@192.168.1.4:80"), [25]byte{3}, big.NewInt(3), big.NewInt(500)},
	}
	var aSortedLinkedList = NewSortedLinkedList(4, compareValue)
	for _, candidateDPOS := range candidateDPOSAccounts {
		aSortedLinkedList.PutOnTop(candidateDPOS)
	}
	for element := aSortedLinkedList.List.Front(); element != nil; element = element.Next() {
		fmt.Println(element.Value.(DPoSCandidateAccount))
	}
	fmt.Println("****************************")
	dposCandidateAccount := DPoSCandidateAccount{[]byte("enode://000000000000000000000000@192.168.1.3:80"), [25]byte{0}, big.NewInt(5), big.NewInt(400)}
	aSortedLinkedList.remove(dposCandidateAccount)
	for element := aSortedLinkedList.List.Front(); element != nil; element = element.Next() {
		fmt.Println(element.Value.(DPoSCandidateAccount))
	}

	fmt.Println("****************************")
}

func TestMul(t *testing.T) {
	limitMaxValue := big.NewInt(1)
	limitMaxValue.Mul(big.NewInt(20), big.NewInt(10))
	fmt.Printf("Big Int: %v\n", limitMaxValue)
}
